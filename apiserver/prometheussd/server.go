/**
 * Tencent is pleased to support the open source community by making Polaris available.
 *
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 *
 * Licensed under the BSD 3-Clause License (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://opensource.org/licenses/BSD-3-Clause
 *
 * Unless required by applicable law or agreed to in writing, software distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 */

package prometheussd

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/emicklei/go-restful"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris-server/apiserver"
	"github.com/polarismesh/polaris-server/common/connlimit"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/service"
)

// PrometheusServer HTTP API服务器
type PrometheusServer struct {
	listenIP        string
	listenPort      uint32
	option          map[string]interface{}
	openAPI         map[string]apiserver.APIConfig
	connLimitConfig *connlimit.Config
	start           bool
	restart         bool
	exitCh          chan struct{}

	server       *http.Server
	namingServer service.DiscoverServer
}

// GetPort 获取端口
func (h *PrometheusServer) GetPort() uint32 {
	return h.listenPort
}

// GetProtocol 获取Server的协议
func (h *PrometheusServer) GetProtocol() string {
	return "prometheus"
}

// Initialize 初始化HTTP API服务器
func (h *PrometheusServer) Initialize(_ context.Context, option map[string]interface{},
	api map[string]apiserver.APIConfig) error {
	h.option = option
	h.openAPI = api
	h.listenIP, _ = option["listenIP"].(string)
	h.listenPort = uint32(option["listenPort"].(int))
	// 连接数限制的配置
	if raw, _ := option["connLimit"].(map[interface{}]interface{}); raw != nil {
		connLimitConfig, err := connlimit.ParseConnLimitConfig(raw)
		if err != nil {
			return err
		}
		h.connLimitConfig = connLimitConfig
	}
	return nil
}

// Run 启动HTTP API服务器
func (h *PrometheusServer) Run(errCh chan error) {
	log.Infof("[API-Server][Prometheus] start server")
	h.exitCh = make(chan struct{}, 1)
	h.start = true
	defer func() {
		close(h.exitCh)
		h.start = false
	}()

	var err error

	// 引入功能模块和插件
	h.namingServer, err = service.GetServer()
	if err != nil {
		log.Errorf("%v", err)
		errCh <- err
		return
	}

	// 初始化http server
	address := fmt.Sprintf("%v:%v", h.listenIP, h.listenPort)

	wsContainer, err := h.createRestfulContainer()
	if err != nil {
		errCh <- err
		return
	}

	server := http.Server{Addr: address, Handler: wsContainer, WriteTimeout: 1 * time.Minute}
	var ln net.Listener
	ln, err = net.Listen("tcp", address)
	if err != nil {
		log.Errorf("[API-Server][Prometheus] net listen(%s) err: %s", address, err.Error())
		errCh <- err
		return
	}

	ln = &tcpKeepAliveListener{ln.(*net.TCPListener)}
	// 开启最大连接数限制
	if h.connLimitConfig != nil && h.connLimitConfig.OpenConnLimit {
		log.Infof("[API-Server][Prometheus] server use max connection limit per ip: %d, http max limit: %d",
			h.connLimitConfig.MaxConnPerHost, h.connLimitConfig.MaxConnLimit)
		ln, err = connlimit.NewListener(ln, h.GetProtocol(), h.connLimitConfig)
		if err != nil {
			log.Errorf("conn limit init err: %s", err.Error())
			errCh <- err
			return
		}
	}
	h.server = &server

	// 开始对外服务
	err = server.Serve(ln)
	if err != nil {
		log.Errorf("%+v", err)
		if !h.restart {
			log.Info("[API-Server][Prometheus] not in restart progress", zap.Error(err))
			errCh <- err
		}

		return
	}

	log.Infof("[API-Server][Prometheus] server stop")
}

// Stop shutdown server
func (h *PrometheusServer) Stop() {
	// 释放connLimit的数据，如果没有开启，也需要执行一下
	// 目的：防止restart的时候，connLimit冲突
	connlimit.RemoveLimitListener(h.GetProtocol())
	if h.server != nil {
		_ = h.server.Close()
	}
}

// Restart restart server
func (h *PrometheusServer) Restart(option map[string]interface{}, api map[string]apiserver.APIConfig,
	errCh chan error) error {
	log.Infof("[API-Server][Prometheus] restart server new config: %+v", option)
	// 备份一下option
	backupOption := h.option
	// 备份一下api
	backupAPI := h.openAPI

	// 设置restart标记，防止stop的时候把错误抛出
	h.restart = true
	// 关闭PrometheusServer
	h.Stop()
	// 等待PrometheusServer退出
	if h.start {
		<-h.exitCh
	}

	log.Info("[API-Server][Prometheus] old server has stopped, begin restart")

	ctx := context.Background()
	if err := h.Initialize(ctx, option, api); err != nil {
		h.restart = false
		if initErr := h.Initialize(ctx, backupOption, backupAPI); initErr != nil {
			log.Errorf("[API-Server][Prometheus] tart with backup cfg err: %s", initErr.Error())
			return initErr
		}
		go h.Run(errCh)

		log.Errorf("[API-Server][Prometheus] restart initialize err: %s", err.Error())
		return err
	}

	log.Infof("[API-Server][Prometheus] init successfully, restart it")
	h.restart = false
	go h.Run(errCh)
	return nil
}

// createRestfulContainer create handler
func (h *PrometheusServer) createRestfulContainer() (*restful.Container, error) {
	wsContainer := restful.NewContainer()

	// 增加CORS TODO
	cors := restful.CrossOriginResourceSharing{
		// ExposeHeaders:  []string{"X-My-Header"},
		AllowedHeaders: []string{"Content-Type", "Accept", "Request-Id"},
		AllowedMethods: []string{"GET", "POST", "PUT"},
		CookiesAllowed: false,
		Container:      wsContainer}
	wsContainer.Filter(cors.Filter)

	// Add container filter to respond to OPTIONS
	wsContainer.Filter(wsContainer.OPTIONSFilter)

	wsContainer.Filter(h.process)

	service, err := h.GetPrometheusDiscoveryServer([]string{})
	if err != nil {
		return nil, err
	}
	wsContainer.Add(service)

	return wsContainer, nil
}

// process 在接收和回复时统一处理请求
func (h *PrometheusServer) process(req *restful.Request, rsp *restful.Response, chain *restful.FilterChain) {
	func() {
		if err := h.preprocess(req, rsp); err != nil {
			return
		}

		chain.ProcessFilter(req, rsp)
	}()

	h.postProcess(req, rsp)
}

// preprocess 请求预处理
func (h *PrometheusServer) preprocess(req *restful.Request, _ *restful.Response) error {
	// 设置开始时间
	req.SetAttribute("start-time", time.Now())
	return nil
}

// postProcess 请求后处理：统计
func (h *PrometheusServer) postProcess(req *restful.Request, _ *restful.Response) {
	now := time.Now()

	// 接口调用统计
	path := req.Request.URL.Path
	if path != "/" {
		// 去掉最后一个"/"
		path = strings.TrimSuffix(path, "/")
	}
	startTime, _ := req.Attribute("start-time").(time.Time)
	// 打印耗时超过1s的请求
	if diff := now.Sub(startTime); diff > time.Second {
		log.Info("[API-Server][Prometheus] handling time > 1s",
			zap.String("client-address", req.Request.RemoteAddr),
			zap.String("user-agent", req.HeaderParameter("User-Agent")),
			zap.String("request-id", req.HeaderParameter("Request-Id")),
			zap.String("method", req.Request.Method),
			zap.String("url", path),
			zap.Duration("handling-time", diff),
		)
	}
}

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
// 来自net/http
type tcpKeepAliveListener struct {
	*net.TCPListener
}

var defaultAlivePeriodTime = 3 * time.Minute

// Accept 来自于net/http
func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	err = tc.SetKeepAlive(true)
	if err != nil {
		return nil, err
	}

	err = tc.SetKeepAlivePeriod(defaultAlivePeriodTime)
	if err != nil {
		return nil, err
	}

	return tc, nil
}
