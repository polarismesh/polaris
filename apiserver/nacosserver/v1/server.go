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

package v1

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/emicklei/go-restful/v3"
	"github.com/pkg/errors"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/apiserver"
	"github.com/polarismesh/polaris/apiserver/nacosserver/core"
	"github.com/polarismesh/polaris/apiserver/nacosserver/model"
	"github.com/polarismesh/polaris/apiserver/nacosserver/v1/config"
	"github.com/polarismesh/polaris/apiserver/nacosserver/v1/discover"
	nacoshttp "github.com/polarismesh/polaris/apiserver/nacosserver/v1/http"
	"github.com/polarismesh/polaris/auth"
	keepalive "github.com/polarismesh/polaris/common/conn/keepalive"
	connlimit "github.com/polarismesh/polaris/common/conn/limit"
	"github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/metrics"
	"github.com/polarismesh/polaris/common/secure"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
)

func NewNacosV1Server(store *core.NacosDataStorage, options ...option) (*NacosV1Server, error) {
	svr := &NacosV1Server{
		store: store,
		discoverOpt: &discover.ServerOption{
			Store: store,
		},
		discoverSvr: &discover.DiscoverServer{},
		configOpt: &config.ServerOption{
			Store: store,
		},
		configSvr: &config.ConfigServer{},
	}

	for i := range options {
		options[i](svr)
	}
	if err := svr.discoverSvr.Initialize(svr.discoverOpt); err != nil {
		return nil, err
	}
	if err := svr.configSvr.Initialize(svr.configOpt); err != nil {
		return nil, err
	}
	return svr, nil
}

// NacosV1Server HTTP API服务器
type NacosV1Server struct {
	listenIP   string
	listenPort uint32

	connLimitConfig *connlimit.Config
	tlsInfo         *secure.TLSInfo
	option          map[string]interface{}
	openAPI         map[string]apiserver.APIConfig
	start           bool

	server    *http.Server
	rateLimit plugin.Ratelimit
	whitelist plugin.Whitelist

	pushCenter core.PushCenter
	store      *core.NacosDataStorage

	checker auth.UserServer

	discoverOpt *discover.ServerOption
	discoverSvr *discover.DiscoverServer
	configOpt   *config.ServerOption
	configSvr   *config.ConfigServer
}

// GetProtocol 获取Server的协议
func (h *NacosV1Server) GetProtocol() string {
	return "nacos-http"
}

// Initialize 初始化HTTP API服务器
func (h *NacosV1Server) Initialize(_ context.Context, option map[string]interface{}, port uint32,
	apiConf map[string]apiserver.APIConfig) error {
	h.option = option
	h.openAPI = apiConf
	h.listenIP = option["listenIP"].(string)
	h.listenPort = port
	if rateLimit := plugin.GetRatelimit(); rateLimit != nil {
		log.Infof("nacos http server open the ratelimit")
		h.rateLimit = rateLimit
	}

	if whitelist := plugin.GetWhitelist(); whitelist != nil {
		log.Infof("nacos http server open the whitelist")
		h.whitelist = whitelist
	}
	return nil
}

// Run 启动HTTP API服务器
func (h *NacosV1Server) Run(errCh chan error) {
	log.Infof("start nacos http server")
	h.start = true
	defer func() {
		h.start = false
	}()

	// 初始化http server
	address := fmt.Sprintf("%v:%v", h.listenIP, h.listenPort)

	var wsContainer *restful.Container
	wsContainer, err := h.createRestfulContainer()
	if err != nil {
		errCh <- err
		return
	}

	server := http.Server{Addr: address, Handler: wsContainer, WriteTimeout: 1 * time.Minute}
	ln, err := net.Listen("tcp", address)
	if err != nil {
		log.Errorf("nacos http server net listen(%s) err: %s", address, err.Error())
		errCh <- err
		return
	}

	ln = keepalive.NewTcpKeepAliveListener(keepalive.DefaultAlivePeriodTime, ln.(*net.TCPListener))
	// 开启最大连接数限制
	if h.connLimitConfig != nil && h.connLimitConfig.OpenConnLimit {
		log.Infof("nacos http server use max connection limit per ip: %d, http max limit: %d",
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
	if h.tlsInfo.IsEmpty() {
		err = server.Serve(ln)
	} else {
		err = server.ServeTLS(ln, h.tlsInfo.CertFile, h.tlsInfo.KeyFile)
	}
	if err != nil {
		log.Errorf("%+v", err)
		errCh <- err
		return
	}

	log.Infof("nacos http server stop")
}

// Stop shutdown server
func (h *NacosV1Server) Stop() {
	// 释放connLimit的数据，如果没有开启，也需要执行一下
	// 目的：防止restart的时候，connLimit冲突
	connlimit.RemoveLimitListener(h.GetProtocol())
	if h.server != nil {
		_ = h.server.Close()
	}
}

// createRestfulContainer create handler
func (h *NacosV1Server) createRestfulContainer() (*restful.Container, error) {
	wsContainer := restful.NewContainer()

	// 增加CORS TODO
	cors := restful.CrossOriginResourceSharing{
		// ExposeHeaders:  []string{"X-My-Header"},
		AllowedHeaders: []string{"Content-Type", "Accept", "Request-Id"},
		AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch},
		CookiesAllowed: false,
		Container:      wsContainer}
	wsContainer.Filter(cors.Filter)

	// Incr container filter to respond to OPTIONS
	wsContainer.Filter(wsContainer.OPTIONSFilter)
	wsContainer.Filter(h.process)

	clientSvc, err := h.discoverSvr.GetClientServer()
	if err != nil {
		return nil, err
	}
	wsContainer.Add(clientSvc)
	clientSvc, err = h.configSvr.GetClientServer()
	if err != nil {
		return nil, err
	}
	wsContainer.Add(clientSvc)
	debugSvc, err := h.GetDebugServer()
	if err != nil {
		return nil, err
	}
	wsContainer.Add(debugSvc)

	authSvc, err := h.GetAuthServer()
	if err != nil {
		return nil, err
	}
	wsContainer.Add(authSvc)

	return wsContainer, nil
}

// process 在接收和回复时统一处理请求
func (h *NacosV1Server) process(req *restful.Request, rsp *restful.Response, chain *restful.FilterChain) {
	func() {
		if err := h.preprocess(req, rsp); err != nil {
			return
		}

		chain.ProcessFilter(req, rsp)
	}()

	h.postProcess(req, rsp)
}

// preprocess 请求预处理
func (h *NacosV1Server) preprocess(req *restful.Request, rsp *restful.Response) error {
	// 设置开始时间
	req.SetAttribute("start-time", time.Now())
	requestURL := req.Request.URL.String()
	// 打印请求
	nacoslog.Info("receive request",
		zap.String("client-address", req.Request.RemoteAddr),
		zap.String("user-agent", req.HeaderParameter("User-Agent")),
		zap.String("method", req.Request.Method),
		zap.String("url", requestURL),
	)

	// 限流
	if err := h.enterRateLimit(req, rsp); err != nil {
		return err
	}

	// 处理 jwt
	accessToken := req.QueryParameter("accessToken")
	if accessToken != "" {
		req.Request.Header.Set(utils.HeaderAuthTokenKey, accessToken)
	}

	return nil
}

// postProcess 请求后处理：统计
func (h *NacosV1Server) postProcess(req *restful.Request, rsp *restful.Response) {
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

	recordApiCall := true
	if !ok {
		code = uint32(rsp.StatusCode())
		recordApiCall = code != http.StatusNotFound
	}

	diff := now.Sub(startTime)
	// 打印耗时超过1s的请求
	if diff > time.Second {
		nacoslog.Info("nacos http server handling time > 1s",
			zap.String("client-address", req.Request.RemoteAddr),
			zap.String("user-agent", req.HeaderParameter("User-Agent")),
			utils.ZapRequestID(req.HeaderParameter("Request-Id")),
			zap.String("method", req.Request.Method),
			zap.String("url", req.Request.URL.String()),
			zap.Duration("handling-time", diff),
		)
	}

	if recordApiCall {
		plugin.GetStatis().ReportCallMetrics(metrics.CallMetric{
			Type:     metrics.ServerCallMetric,
			API:      method,
			Protocol: "HTTP",
			Code:     int(code),
			Duration: diff,
		})
	}
}

// enterAuth 访问鉴权
func (h *NacosV1Server) enterAuth(req *restful.Request, rsp *restful.Response) error {
	// 判断白名单插件是否开启
	if h.whitelist == nil {
		return nil
	}

	rid := req.HeaderParameter("Request-Id")

	address := req.Request.RemoteAddr
	segments := strings.Split(address, ":")
	if len(segments) != 2 {
		return nil
	}
	if !h.whitelist.Contain(segments[0]) {
		log.Error("nacos http server http access is not allowed",
			zap.String("client", address),
			utils.ZapRequestID(rid))
		nacoshttp.WrirteNacosErrorResponse(&model.NacosApiError{
			DetailErrCode: int32(apimodel.Code_NotAllowedAccess),
		}, rsp)
		return errors.New("nacos http server http access is not allowed")
	}
	return nil
}

// enterRateLimit 访问限制
func (h *NacosV1Server) enterRateLimit(req *restful.Request, rsp *restful.Response) error {
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
		log.Error("nacos http server ip ratelimit is not allow", zap.String("client", address),
			utils.ZapRequestID(rid))
		nacoshttp.WrirteNacosErrorResponse(&model.NacosApiError{
			DetailErrCode: int32(apimodel.Code_IPRateLimit),
		}, rsp)
		return errors.New("ip ratelimit is not allow")
	}

	// 接口级限流
	apiName := fmt.Sprintf("%s:%s", req.Request.Method,
		strings.TrimSuffix(req.Request.URL.Path, "/"))
	if ok := h.rateLimit.Allow(plugin.APIRatelimit, apiName); !ok {
		log.Error("nacos http server api ratelimit is not allow", zap.String("client", address),
			utils.ZapRequestID(rid), zap.String("api", apiName))
		nacoshttp.WrirteNacosErrorResponse(&model.NacosApiError{
			DetailErrCode: int32(apimodel.Code_APIRateLimit),
		}, rsp)
		return errors.New("api ratelimit is not allow")
	}

	return nil
}
