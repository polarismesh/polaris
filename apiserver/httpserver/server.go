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

package httpserver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"strings"
	"sync"
	"time"

	"github.com/polarismesh/polaris-server/auth"
	"github.com/polarismesh/polaris-server/healthcheck"
	"github.com/polarismesh/polaris-server/plugin/statis/local"

	"github.com/emicklei/go-restful"
	"github.com/pkg/errors"
	"github.com/polarismesh/polaris-server/apiserver"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/connlimit"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/naming"
	"github.com/polarismesh/polaris-server/plugin"
	"go.uber.org/zap"
)

/**
 * HTTPServer HTTP API服务器
 */
type HTTPServer struct {
	listenIP        string
	listenPort      uint32
	connLimitConfig *connlimit.Config
	option          map[string]interface{}
	openAPI         map[string]apiserver.APIConfig
	start           bool
	restart         bool
	exitCh          chan struct{}

	enablePprof bool

	freeMemMu *sync.Mutex

	server *http.Server

	namingServer      naming.DiscoverServer
	healthCheckServer *healthcheck.Server
	rateLimit         plugin.Ratelimit
	statis            plugin.Statis
	auth              plugin.Auth

	userServer     auth.UserServer
	strategyServer auth.AuthStrategyServer
	authMgn        auth.AuthManager
}

const (
	// Discover discover string
	Discover string = "Discover"
)

/**
 * GetPort 获取端口
 */
func (h *HTTPServer) GetPort() uint32 {
	return h.listenPort
}

/**
 * GetProtocol 获取Server的协议
 */
func (h *HTTPServer) GetProtocol() string {
	return "http"
}

/**
 * Initialize 初始化HTTP API服务器
 */
func (h *HTTPServer) Initialize(_ context.Context, option map[string]interface{},
	api map[string]apiserver.APIConfig) error {
	h.option = option
	h.openAPI = api
	h.listenIP = option["listenIP"].(string)
	h.listenPort = uint32(option["listenPort"].(int))
	h.enablePprof, _ = option["enablePprof"].(bool)
	// 连接数限制的配置
	if raw, _ := option["connLimit"].(map[interface{}]interface{}); raw != nil {
		connLimitConfig, err := connlimit.ParseConnLimitConfig(raw)
		if err != nil {
			return err
		}
		h.connLimitConfig = connLimitConfig
	}
	if rateLimit := plugin.GetRatelimit(); rateLimit != nil {
		log.Infof("http server open the ratelimit")
		h.rateLimit = rateLimit
	}

	if auth := plugin.GetAuth(); auth != nil {
		log.Infof("http server open the auth")
		h.auth = auth
	}

	h.freeMemMu = new(sync.Mutex)

	return nil
}

/**
 * Run 启动HTTP API服务器
 */
func (h *HTTPServer) Run(errCh chan error) {
	log.Infof("start httpserver")
	h.exitCh = make(chan struct{}, 1)
	h.start = true
	defer func() {
		close(h.exitCh)
		h.start = false
	}()

	var err error
	// 引入功能模块和插件
	h.namingServer, err = naming.GetServer()
	if err != nil {
		log.Errorf("%v", err)
		errCh <- err
		return
	}

	authMgn, err := auth.GetAuthManager()
	if err != nil {
		log.Errorf("%v", err)
		errCh <- err
		return
	}

	h.authMgn = authMgn
	h.userServer = authMgn.GetUserServer()
	h.strategyServer = authMgn.GetAuthStrategyServer()

	h.healthCheckServer, err = healthcheck.GetServer()
	if err != nil {
		log.Errorf("%v", err)
		errCh <- err
		return
	}
	h.statis = plugin.GetStatis()

	// 初始化http server
	address := fmt.Sprintf("%v:%v", h.listenIP, h.listenPort)

	var wsContainer *restful.Container
	wsContainer, err = h.createRestfulContainer()
	if err != nil {
		errCh <- err
		return
	}

	server := http.Server{Addr: address, Handler: wsContainer, WriteTimeout: 1 * time.Minute}
	var ln net.Listener
	ln, err = net.Listen("tcp", address)
	if err != nil {
		log.Errorf("net listen(%s) err: %s", address, err.Error())
		errCh <- err
		return
	}

	ln = &tcpKeepAliveListener{ln.(*net.TCPListener)}
	// 开启最大连接数限制
	if h.connLimitConfig != nil && h.connLimitConfig.OpenConnLimit {
		log.Infof("http server use max connection limit per ip: %d, http max limit: %d",
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
			log.Infof("not in restart progress, broadcast error")
			errCh <- err
		}

		return
	}

	log.Infof("httpserver stop")
}

// Stop shutdown server
func (h *HTTPServer) Stop() {
	// 释放connLimit的数据，如果没有开启，也需要执行一下
	// 目的：防止restart的时候，connLimit冲突
	connlimit.RemoveLimitListener(h.GetProtocol())
	if h.server != nil {
		_ = h.server.Close()
	}
}

// Restart restart server
func (h *HTTPServer) Restart(option map[string]interface{}, api map[string]apiserver.APIConfig,
	errCh chan error) error {
	log.Infof("restart httpserver new config: %+v", option)
	// 备份一下option
	backupOption := h.option
	// 备份一下api
	backupAPI := h.openAPI

	// 设置restart标记，防止stop的时候把错误抛出
	h.restart = true
	// 关闭httpserver
	h.Stop()
	// 等待httpserver退出
	if h.start {
		<-h.exitCh
	}

	log.Infof("old httpserver has stopped, begin restart httpserver")

	ctx := context.Background()
	if err := h.Initialize(ctx, option, api); err != nil {
		h.restart = false
		if initErr := h.Initialize(ctx, backupOption, backupAPI); initErr != nil {
			log.Errorf("start httpserver with backup cfg err: %s", initErr.Error())
			return initErr
		}
		go h.Run(errCh)

		log.Errorf("restart httpserver initialize err: %s", err.Error())
		return err
	}

	log.Infof("init httpserver successfully, restart it")
	h.restart = false
	go h.Run(errCh)
	return nil
}

// 创建handler
func (h *HTTPServer) createRestfulContainer() (*restful.Container, error) {
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

	for name, config := range h.openAPI {
		switch name {
		case "admin":
			if config.Enable {
				wsContainer.Add(h.GetAdminServer())
				wsContainer.Add(h.GetMaintainAccessServer())
			}
		case "console":
			if config.Enable {
				service, err := h.GetConsoleAccessServer(config.Include)
				if err != nil {
					return nil, err
				}

				wsContainer.Add(service)
				wsContainer.Add(h.GetAuthServer())
			}
		case "client":
			if config.Enable {
				service, err := h.GetClientAccessServer(config.Include)
				if err != nil {
					return nil, err
				}
				wsContainer.Add(service)
			}
		default:
			log.Errorf("api %s does not exist in httpserver", name)
			return nil, fmt.Errorf("api %s does not exist in httpserver", name)
		}
	}

	if h.enablePprof {
		h.enablePprofAccess(wsContainer)
	}

	statis := plugin.GetStatis()
	if _, ok := statis.(*local.StatisWorker); ok {
		h.enablePrometheusAccess(wsContainer)
	}

	return wsContainer, nil
}

// 开启pprof接口
func (h *HTTPServer) enablePprofAccess(wsContainer *restful.Container) {
	log.Infof("open http access for pprof")
	wsContainer.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	wsContainer.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	wsContainer.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	wsContainer.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
}

// 开启 Prometheus 接口
func (h *HTTPServer) enablePrometheusAccess(wsContainer *restful.Container) {
	log.Infof("open http access for prometheus")

	statis := plugin.GetStatis().(*local.StatisWorker)

	wsContainer.Handle("/metrics", statis.GetPrometheusHandler())
}

/**
 * @brief 在接收和回复时统一处理请求
 */
func (h *HTTPServer) process(req *restful.Request, rsp *restful.Response, chain *restful.FilterChain) {
	func() {
		if err := h.preprocess(req, rsp); err != nil {
			return
		}

		chain.ProcessFilter(req, rsp)
	}()

	h.postProcess(req, rsp)
}

/**
 * @brief 请求预处理
 */
func (h *HTTPServer) preprocess(req *restful.Request, rsp *restful.Response) error {
	// 设置开始时间
	req.SetAttribute("start-time", time.Now())

	// 处理请求ID
	requestID := req.HeaderParameter("Request-Id")
	if requestID == "" {
		// TODO: 设置请求ID
	}

	platformID := req.HeaderParameter("Platform-Id")
	requestURL := req.Request.URL.String()
	if !strings.Contains(requestURL, Discover) {
		// 打印请求
		log.Info("receive request",
			zap.String("client-address", req.Request.RemoteAddr),
			zap.String("user-agent", req.HeaderParameter("User-Agent")),
			zap.String("request-id", requestID),
			zap.String("platform-id", platformID),
			zap.String("method", req.Request.Method),
			zap.String("url", requestURL),
		)
	}

	// 管理端接口访问鉴权
	if strings.Contains(requestURL, "naming") {
		if err := h.enterAuth(req, rsp); err != nil {
			return err
		}
	}

	// 限流
	if err := h.enterRateLimit(req, rsp); err != nil {
		return err
	}

	return nil
}

/**
 * postProcess 请求后处理：统计
 */
func (h *HTTPServer) postProcess(req *restful.Request, rsp *restful.Response) {
	now := time.Now()

	// 接口调用统计
	path := req.Request.URL.Path
	if path != "/" {
		// 去掉最后一个"/"
		path = strings.TrimSuffix(path, "/")
	}
	method := req.Request.Method + ":" + path
	startTime := req.Attribute("start-time").(time.Time)
	code, ok := req.Attribute(utils.PolarisCode).(uint32)
	if !ok {
		code = uint32(rsp.StatusCode())
	}

	diff := now.Sub(startTime)
	// 打印耗时超过1s的请求
	if diff > time.Second {
		log.Info("handling time > 1s",
			zap.String("client-address", req.Request.RemoteAddr),
			zap.String("user-agent", req.HeaderParameter("User-Agent")),
			zap.String("request-id", req.HeaderParameter("Request-Id")),
			zap.String("method", req.Request.Method),
			zap.String("url", req.Request.URL.String()),
			zap.Duration("handling-time", diff),
		)
	}

	_ = h.statis.AddAPICall(method, "HTTP", int(code), diff.Nanoseconds())
}

/**
 * @brief 访问鉴权
 */
func (h *HTTPServer) enterAuth(req *restful.Request, rsp *restful.Response) error {
	// 判断鉴权插件是否开启
	if h.auth == nil {
		return nil
	}

	rid := req.HeaderParameter("Request-Id")
	pid := req.HeaderParameter("Platform-Id")
	pToken := req.HeaderParameter("Platform-Token")

	address := req.Request.RemoteAddr
	segments := strings.Split(address, ":")
	if len(segments) != 2 {
		return nil
	}

	if !h.auth.IsWhiteList(segments[0]) && !h.auth.Allow(pid, pToken) {
		log.Error("http access is not allowed",
			zap.String("client", address),
			zap.String("request-id", rid),
			zap.String("platform-id", pid),
			zap.String("platform-token", pToken))
		HTTPResponse(req, rsp, api.NotAllowedAccess)
		return errors.New("http access is not allowed")
	}
	return nil
}

// 访问限制
func (h *HTTPServer) enterRateLimit(req *restful.Request, rsp *restful.Response) error {
	// 检查限流插件是否开启
	if h.rateLimit == nil {
		return nil
	}

	rid := req.HeaderParameter("Request-Id")
	// IP级限流
	// 先获取当前请求的address
	address := req.Request.RemoteAddr
	segments := strings.Split(address, ":")
	if len(segments) != 2 {
		return nil
	}
	if ok := h.rateLimit.Allow(plugin.IPRatelimit, segments[0]); !ok {
		log.Error("ip ratelimit is not allow", zap.String("client", address),
			zap.String("request-id", rid))
		HTTPResponse(req, rsp, api.IPRateLimit)
		return errors.New("ip ratelimit is not allow")
	}

	// 接口级限流
	apiName := fmt.Sprintf("%s:%s", req.Request.Method,
		strings.TrimSuffix(req.Request.URL.Path, "/"))
	if ok := h.rateLimit.Allow(plugin.APIRatelimit, apiName); !ok {
		log.Error("api ratelimit is not allow", zap.String("client", address),
			zap.String("request-id", rid), zap.String("api", apiName))
		HTTPResponse(req, rsp, api.APIRateLimit)
		return errors.New("api ratelimit is not allow")
	}

	return nil
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
