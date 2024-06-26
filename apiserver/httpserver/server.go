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
	"strings"
	"sync/atomic"
	"time"

	restful "github.com/emicklei/go-restful/v3"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/pkg/errors"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/admin"
	"github.com/polarismesh/polaris/apiserver"
	confighttp "github.com/polarismesh/polaris/apiserver/httpserver/config"
	v1 "github.com/polarismesh/polaris/apiserver/httpserver/discover/v1"
	v2 "github.com/polarismesh/polaris/apiserver/httpserver/discover/v2"
	httpcommon "github.com/polarismesh/polaris/apiserver/httpserver/utils"
	"github.com/polarismesh/polaris/auth"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/conn/keepalive"
	connlimit "github.com/polarismesh/polaris/common/conn/limit"
	commonlog "github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/metrics"
	"github.com/polarismesh/polaris/common/secure"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/config"
	"github.com/polarismesh/polaris/namespace"
	"github.com/polarismesh/polaris/plugin"
	"github.com/polarismesh/polaris/service"
	"github.com/polarismesh/polaris/service/healthcheck"
)

var (
	cacheTypes = map[string]struct{}{
		apiservice.DiscoverResponse_INSTANCE.String():        {},
		apiservice.DiscoverResponse_ROUTING.String():         {},
		apiservice.DiscoverResponse_RATE_LIMIT.String():      {},
		apiservice.DiscoverResponse_CIRCUIT_BREAKER.String(): {},
		apiservice.DiscoverResponse_FAULT_DETECTOR.String():  {},
		apiservice.DiscoverResponse_SERVICES.String():        {},
	}
)

// HTTPServer HTTP API服务器
type HTTPServer struct {
	listenIP        string
	listenPort      uint32
	connLimitConfig *connlimit.Config
	tlsInfo         *secure.TLSInfo
	option          map[string]interface{}
	openAPI         map[string]apiserver.APIConfig
	start           bool
	restart         bool
	exitCh          chan struct{}

	enablePprof   *atomic.Bool
	enableSwagger bool

	server            *http.Server
	maintainServer    admin.AdminOperateServer
	namespaceServer   namespace.NamespaceOperateServer
	namingServer      service.DiscoverServer
	configServer      config.ConfigCenterServer
	healthCheckServer *healthcheck.Server
	rateLimit         plugin.Ratelimit
	statis            plugin.Statis
	whitelist         plugin.Whitelist

	discoverV1 v1.HTTPServerV1
	discoverV2 v2.HTTPServerV2
	configSvr  confighttp.HTTPServer

	userMgn     auth.UserServer
	strategyMgn auth.StrategyServer

	// apiserverSlots
	apiserverSlots map[string]apiserver.Apiserver
}

const (
	// Discover discover string
	Discover               string = "Discover"
	defaultAlivePeriodTime        = 3 * time.Minute
)

// GetPort 获取端口
func (h *HTTPServer) GetPort() uint32 {
	return h.listenPort
}

// GetProtocol 获取Server的协议
func (h *HTTPServer) GetProtocol() string {
	return "http"
}

// Initialize 初始化HTTP API服务器
func (h *HTTPServer) Initialize(ctx context.Context, option map[string]interface{},
	apiConf map[string]apiserver.APIConfig) error {
	h.option = option
	h.openAPI = apiConf
	h.listenIP = option["listenIP"].(string)
	h.listenPort = uint32(option["listenPort"].(int))
	h.enablePprof = &atomic.Bool{}
	if val, _ := option["enablePprof"].(bool); val {
		h.enablePprof.Store(true)
	}
	h.enableSwagger, _ = option["enableSwagger"].(bool)
	h.apiserverSlots, _ = ctx.Value(utils.ContextAPIServerSlot{}).(map[string]apiserver.Apiserver)
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

	if whitelist := plugin.GetWhitelist(); whitelist != nil {
		log.Infof("http server open the whitelist")
		h.whitelist = whitelist
	}

	if raw, _ := option["enableCacheProto"].(bool); raw {
		types := make([]string, 0, len(cacheTypes))
		for k := range cacheTypes {
			types = append(types, k)
		}
		httpcommon.InitProtoCache(option, types, discoverCacheConvert)
	}

	// tls 配置信息
	if raw, _ := option["tls"].(map[interface{}]interface{}); raw != nil {
		tlsConfig, err := secure.ParseTLSConfig(raw)
		if err != nil {
			return err
		}
		h.tlsInfo = &secure.TLSInfo{
			CertFile:      tlsConfig.CertFile,
			KeyFile:       tlsConfig.KeyFile,
			TrustedCAFile: tlsConfig.TrustedCAFile,
		}
	}

	metrics.SetMetricsPort(int32(h.listenPort))
	return nil
}

// Run 启动HTTP API服务器
func (h *HTTPServer) Run(errCh chan error) {
	log.Infof("start httpserver")
	h.exitCh = make(chan struct{}, 1)
	h.start = true
	defer func() {
		close(h.exitCh)
		h.start = false
	}()

	var err error

	h.maintainServer, err = admin.GetServer()
	if err != nil {
		log.Errorf("%v", err)
		errCh <- err
		return
	}

	// 引入命名空间模块
	h.namespaceServer, err = namespace.GetServer()
	if err != nil {
		log.Errorf("%v", err)
		errCh <- err
		return
	}

	// 引入功能模块和插件
	h.namingServer, err = service.GetServer()
	if err != nil {
		log.Errorf("%v", err)
		errCh <- err
		return
	}

	userMgn, err := auth.GetUserServer()
	if err != nil {
		log.Errorf("%v", err)
		errCh <- err
		return
	}

	strategyMgn, err := auth.GetStrategyServer()
	if err != nil {
		log.Errorf("%v", err)
		errCh <- err
		return
	}

	h.userMgn = userMgn
	h.strategyMgn = strategyMgn

	h.healthCheckServer, err = healthcheck.GetServer()
	if err != nil {
		log.Errorf("%v", err)
		errCh <- err
		return
	}
	h.statis = plugin.GetStatis()

	// 初始化配置中心模块
	h.configServer, err = config.GetServer()
	if err != nil {
		log.Errorf("set config server to http server error. %v", err)
		errCh <- err
		return
	}

	h.discoverV1 = *v1.NewV1Server(h.namespaceServer, h.namingServer, h.healthCheckServer)
	h.discoverV2 = *v2.NewV2Server(h.namespaceServer, h.namingServer, h.healthCheckServer)
	h.configSvr = *confighttp.NewServer(h.maintainServer, h.namespaceServer, h.configServer)

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

	ln = keepalive.NewTcpKeepAliveListener(defaultAlivePeriodTime, ln.(*net.TCPListener))
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
	if h.tlsInfo.IsEmpty() {
		err = server.Serve(ln)
	} else {
		err = server.ServeTLS(ln, h.tlsInfo.CertFile, h.tlsInfo.KeyFile)
	}
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
	// stop http server
	if h.server != nil {
		// 延迟三秒，等待http server关闭，做到流量无损。
		// 在此之前已经建立的链接，会正常执行业务，若执行时长超过3秒，则会抛出异常。
		// 若是在3秒内提前处理完所有请求，h.server会提前关闭。
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := h.server.Shutdown(ctx); nil != err {
			log.Errorf("httpserver shutdown failed, err: %v\n", err)
		}
	}
}

// Restart restart server
func (h *HTTPServer) Restart(option map[string]interface{}, apiConf map[string]apiserver.APIConfig,
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
	if err := h.Initialize(ctx, option, apiConf); err != nil {
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

// createRestfulContainer create handler
func (h *HTTPServer) createRestfulContainer() (*restful.Container, error) {
	wsContainer := restful.NewContainer()
	wsContainer.RecoverHandler(h.recoverFunc)

	// 增加CORS TODO
	cors := restful.CrossOriginResourceSharing{
		// ExposeHeaders:  []string{"X-My-Header"},
		AllowedHeaders: []string{"Content-Type", "Accept", "Request-Id"},
		AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut},
		CookiesAllowed: false,
		Container:      wsContainer}
	wsContainer.Filter(cors.Filter)

	// Incr container filter to respond to OPTIONS
	wsContainer.Filter(wsContainer.OPTIONSFilter)

	wsContainer.Filter(h.process)

	for name, apiConfig := range h.openAPI {
		switch name {
		case "admin":
			if apiConfig.Enable {
				wsContainer.Add(h.GetIndexServer())
				wsContainer.Add(h.GetAdminAccessServer())
			}
		case "console":
			if apiConfig.Enable {
				namingServiceV1, err := h.discoverV1.GetConsoleAccessServer(apiConfig.Include)
				if err != nil {
					return nil, err
				}
				wsContainer.Add(namingServiceV1)

				namingServiceV2, err := h.discoverV2.GetConsoleAccessServer(apiConfig.Include)
				if err != nil {
					return nil, err
				}
				wsContainer.Add(namingServiceV2)

				configService, err := h.configSvr.GetConsoleAccessServer(apiConfig.Include)
				if err != nil {
					return nil, err
				}
				wsContainer.Add(configService)

				ws := new(restful.WebService)
				ws.Path("/core/v1").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)
				if err := h.GetClientServer(ws); err != nil {
					return nil, err
				}
				if err := h.GetCoreV1ConsoleAccessServer(ws, apiConfig.Include); err != nil {
					return nil, err
				}
				if err := h.GetAuthServer(ws); err != nil {
					return nil, err
				}

				prometheusSvc, err := h.GetPrometheusDiscoveryServer(apiConfig.Include)
				if err != nil {
					return nil, err
				}
				wsContainer.Add(prometheusSvc)

				wsContainer.Add(ws)
			}
		case "client":
			if apiConfig.Enable {
				ws := new(restful.WebService)
				ws.Path("/v1").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

				if err := h.discoverV1.GetClientAccessServer(ws, apiConfig.Include); err != nil {
					return nil, err
				}
				if err := h.configSvr.GetClientAccessServer(ws, apiConfig.Include); err != nil {
					return nil, err
				}
				wsContainer.Add(ws)
			}
		default:
			log.Warnf("api %s does not exist in httpserver", name)
		}
	}

	h.enablePprofAccess(wsContainer)
	h.enableSwaggerAPI(wsContainer)
	// 收集插件的 endpoint 数据
	h.enablePluginDebugAccess(wsContainer)
	h.enablePrometheusAccess(wsContainer)
	return wsContainer, nil
}

// enablePluginDebugAccess .
func (h *HTTPServer) enablePluginDebugAccess(wsContainer *restful.Container) {
	log.Infof("open http access for plugin")

	wsContainer.Handle("/debug/endpoints", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		endpints := make([]map[string]string, 0, 8)
		for _, checker := range h.healthCheckServer.Checkers() {
			handlers := checker.DebugHandlers()
			for i := range handlers {
				endpints = append(endpints, map[string]string{
					"path": handlers[i].Path,
					"desc": handlers[i].Desc,
				})
			}
		}

		for _, item := range h.apiserverSlots {
			if val, ok := item.(apiserver.EnrichApiserver); ok {
				handlers := val.DebugHandlers()
				for i := range handlers {
					endpints = append(endpints, map[string]string{
						"path": handlers[i].Path,
						"desc": handlers[i].Desc,
					})
				}
			}
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(utils.MustJson(endpints)))
	}))

	for _, checker := range h.healthCheckServer.Checkers() {
		handlers := checker.DebugHandlers()
		for i := range handlers {
			wsContainer.Handle(handlers[i].Path, handlers[i].Handler)
		}
	}

	for _, item := range h.apiserverSlots {
		if val, ok := item.(apiserver.EnrichApiserver); ok {
			handlers := val.DebugHandlers()
			for i := range handlers {
				wsContainer.Handle(handlers[i].Path, handlers[i].Handler)
			}
		}
	}
}

// process 在接收和回复时统一处理请求
func (h *HTTPServer) process(req *restful.Request, rsp *restful.Response, chain *restful.FilterChain) {
	func() {
		if err := h.preprocess(req, rsp); err != nil {
			return
		}

		chain.ProcessFilter(req, rsp)
	}()

	h.postProcess(req, rsp)
}

// preprocess 请求预处理
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
			utils.ZapRequestID(requestID),
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

// postProcess 请求后处理：统计
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

	recordApiCall := true
	if !ok {
		code = uint32(rsp.StatusCode())
		recordApiCall = code != http.StatusNotFound && code != http.StatusMethodNotAllowed
	}

	diff := now.Sub(startTime)
	// 打印耗时超过1s的请求
	if diff > time.Second {
		var scope *commonlog.Scope
		if strings.Contains(path, "naming") {
			scope = namingLog
		} else {
			scope = configLog
		}

		scope.Info("handling time > 1s",
			zap.String("client-address", req.Request.RemoteAddr),
			zap.String("user-agent", req.HeaderParameter("User-Agent")),
			utils.ZapRequestID(req.HeaderParameter("Request-Id")),
			zap.String("method", req.Request.Method),
			zap.String("url", req.Request.URL.String()),
			zap.Duration("handling-time", diff),
		)
	}

	if recordApiCall {
		h.statis.ReportCallMetrics(metrics.CallMetric{
			Type:     metrics.ServerCallMetric,
			API:      method,
			Protocol: "HTTP",
			Code:     int(code),
			Duration: diff,
		})
	}
}

// enterAuth 访问鉴权
func (h *HTTPServer) enterAuth(req *restful.Request, rsp *restful.Response) error {
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
		log.Error("http access is not allowed",
			zap.String("client", address),
			utils.ZapRequestID(rid))
		httpcommon.HTTPResponse(req, rsp, api.NotAllowedAccess)
		return errors.New("http access is not allowed")
	}
	return nil
}

// enterRateLimit 访问限制
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
			utils.ZapRequestID(rid))
		httpcommon.HTTPResponse(req, rsp, api.IPRateLimit)
		return errors.New("ip ratelimit is not allow")
	}

	// 接口级限流
	apiName := fmt.Sprintf("%s:%s", req.Request.Method,
		strings.TrimSuffix(req.Request.URL.Path, "/"))
	if ok := h.rateLimit.Allow(plugin.APIRatelimit, apiName); !ok {
		log.Error("api ratelimit is not allow", zap.String("client", address),
			utils.ZapRequestID(rid), zap.String("api", apiName))
		httpcommon.HTTPResponse(req, rsp, api.APIRateLimit)
		return errors.New("api ratelimit is not allow")
	}

	return nil
}

func (h *HTTPServer) recoverFunc(i interface{}, w http.ResponseWriter) {
	log.Errorf("panic %+v", i)
	obj := &service_manage.Response{}

	status := api.CalcCode(obj)
	if code := obj.GetCode().GetValue(); code != api.ExecuteSuccess {
		w.Header().Add(utils.PolarisCode, fmt.Sprintf("%d", code))
		w.Header().Add(utils.PolarisMessage, api.Code2Info(code))
	}
	w.WriteHeader(status)

	m := jsonpb.Marshaler{Indent: " ", EmitDefaults: true}
	_ = m.Marshal(w, obj)
}

func discoverCacheConvert(m interface{}) *httpcommon.CacheObject {
	resp, ok := m.(*apiservice.DiscoverResponse)
	if !ok {
		return nil
	}
	if resp.Code.GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
		return nil
	}
	if resp.GetService().GetRevision().GetValue() == "" {
		return nil
	}
	if _, ok := cacheTypes[resp.GetType().String()]; !ok {
		return nil
	}
	keyProto := fmt.Sprintf("%s-%s-%s", resp.GetService().GetNamespace().GetValue(),
		resp.GetService().GetName().GetValue(), resp.GetService().GetRevision().GetValue())

	return &httpcommon.CacheObject{
		OriginVal: resp,
		CacheType: resp.Type.String(),
		Key:       keyProto,
	}
}
