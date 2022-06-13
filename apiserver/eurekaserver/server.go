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

package eurekaserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/emicklei/go-restful"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris-server/apiserver"
	"github.com/polarismesh/polaris-server/common/connlimit"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/plugin"
	"github.com/polarismesh/polaris-server/service"
	"github.com/polarismesh/polaris-server/service/healthcheck"
)

const (
	SecureProtocol   = "HTTPS"
	InsecureProtocol = "HTTP"

	MetadataRegisterFrom        = "internal-register-from"
	MetadataAppGroupName        = "internal-eureka-app-group"
	MetadataCountryId           = "internal-eureka-country-id"
	MetadataDataCenterInfoClazz = "internal-eureka-dci-clazz"
	MetadataDataCenterInfoName  = "internal-eureka-dci-name"
	MetadataHostName            = "internal-eureka-hostname"
	MetadataRenewalInterval     = "internal-eureka-renewal-interval"
	MetadataDuration            = "internal-eureka-duration"
	MetadataHomePageUrl         = "internal-eureka-home-url"
	MetadataStatusPageUrl       = "internal-eureka-status-url"
	MetadataHealthCheckUrl      = "internal-eureka-health-url"
	MetadataVipAddress          = "internal-eureka-vip"
	MetadataSecureVipAddress    = "internal-eureka-secure-vip"
	MetadataInsecurePort        = "internal-eureka-insecure-port"
	MetadataInsecurePortEnabled = "internal-eureka-insecure-port-enabled"
	MetadataSecurePort          = "internal-eureka-secure-port"
	MetadataSecurePortEnabled   = "internal-eureka-secure-port-enabled"

	ServerEureka = "eureka"

	KeyRegion = "region"
	keyZone   = "zone"
	keyCampus = "campus"

	StatusOutOfService = "OUT_OF_SERVICE"
	StatusUp           = "UP"
	StatusDown         = "DOWN"
	StatusUnknown      = "UNKNOWN"

	ActionAdded    = "ADDED"
	ActionModified = "MODIFIED"
	ActionDeleted  = "DELETED"

	DefaultCountryIdInt            = 1
	DefaultDciClazz                = "com.netflix.appinfo.InstanceInfo$DefaultDataCenterInfo"
	DefaultDciName                 = "MyOwn"
	DefaultRenewInterval           = 30
	DefaultDuration                = 90
	DefaultUnhealthyExpireInterval = 180

	DefaultOwner        = "polaris"
	DefaultSSLPort      = 443
	DefaultInsecurePort = 8080

	operationRegister           = "POST:/eureka/apps/{application}"
	operationDeregister         = "DELETE:/eureka/apps/{application}/{instanceId}"
	operationHeartbeat          = "PUT:/eureka/apps/{application}/{instanceId}"
	operationAllInstances       = "GET:/eureka/apps"
	operationDelta              = "GET:/eureka/apps/delta"
	operationAllAppIDInstances  = "GET:/eureka/apps/{application}"
	operationAppIDInstance      = "GET:/eureka/apps/{application}/{instanceId}"
	operationStatusChange       = "PUT:/eureka/apps/{application}/{instanceId}/status"
	operationDeleteStatusChange = "DELETE:/eureka/apps/{application}/{instanceId}/status"

	pathPrefix   = "/eureka/apps"
	statusSuffix = "/status"

	statusCodeHeader = utils.PolarisCode
)

var (
	DefaultDataCenterInfo = &DataCenterInfo{
		Clazz: DefaultDciClazz,
		Name:  DefaultDciName,
	}
	DefaultCountryId = strconv.Itoa(DefaultCountryIdInt)
)

// EurekaServer is the Eureka server
type EurekaServer struct {
	server                 *http.Server
	namingServer           service.DiscoverServer
	healthCheckServer      *healthcheck.Server
	connLimitConfig        *connlimit.Config
	option                 map[string]interface{}
	openAPI                map[string]apiserver.APIConfig
	worker                 *ApplicationsWorker
	listenPort             uint32
	listenIP               string
	exitCh                 chan struct{}
	start                  bool
	restart                bool
	rateLimit              plugin.Ratelimit
	statis                 plugin.Statis
	namespace              string
	refreshInterval        time.Duration
	deltaExpireInterval    time.Duration
	enableSelfPreservation bool
}

// GetPort 获取端口
func (h *EurekaServer) GetPort() uint32 {
	return h.listenPort
}

// GetProtocol 获取协议
func (h *EurekaServer) GetProtocol() string {
	return ServerEureka
}

// Initialize 初始化HTTP API服务器
func (h *EurekaServer) Initialize(ctx context.Context, option map[string]interface{},
	api map[string]apiserver.APIConfig) error {
	h.listenIP = option[optionListenIP].(string)
	h.listenPort = uint32(option[optionListenPort].(int))
	h.option = option
	h.openAPI = api

	var namespace = DefaultNamespace
	if namespaceValue, ok := option[optionNamespace]; ok {
		theNamespace := namespaceValue.(string)
		if len(theNamespace) > 0 {
			namespace = theNamespace
		}
	}
	h.namespace = namespace

	var refreshInterval int
	if value, ok := option[optionRefreshInterval]; ok {
		refreshInterval = value.(int)
	}
	if refreshInterval < DefaultRefreshInterval {
		refreshInterval = DefaultRefreshInterval
	}

	var deltaExpireInterval int
	if value, ok := option[optionDeltaExpireInterval]; ok {
		deltaExpireInterval = value.(int)
	}
	if deltaExpireInterval < DefaultDetailExpireInterval {
		deltaExpireInterval = DefaultDetailExpireInterval
	}

	// 连接数限制的配置
	if raw, _ := option[optionConnLimit].(map[interface{}]interface{}); raw != nil {
		connLimitConfig, err := connlimit.ParseConnLimitConfig(raw)
		if err != nil {
			return err
		}
		h.connLimitConfig = connLimitConfig
	}
	h.refreshInterval = time.Duration(refreshInterval) * time.Second
	h.deltaExpireInterval = time.Duration(deltaExpireInterval) * time.Second

	var enableSelfPreservation bool
	if value, ok := option[optionEnableSelfPreservation]; ok {
		enableSelfPreservation = value.(bool)
	} else {
		enableSelfPreservation = DefaultEnableSelfPreservation
	}
	h.enableSelfPreservation = enableSelfPreservation
	return nil
}

// Run 启动HTTP API服务器
func (h *EurekaServer) Run(errCh chan error) {
	log.Infof("start eurekaserver")
	h.exitCh = make(chan struct{})
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
	h.healthCheckServer, err = healthcheck.GetServer()
	if err != nil {
		log.Errorf("%v", err)
		errCh <- err
		return
	}
	h.worker = NewApplicationsWorker(h.refreshInterval, h.deltaExpireInterval, h.enableSelfPreservation,
		h.namingServer, h.healthCheckServer, h.namespace)
	h.statis = plugin.GetStatis()
	// 初始化http server
	address := fmt.Sprintf("%v:%v", h.listenIP, h.listenPort)

	wsContainer, err := h.createRestfulContainer()
	if err != nil {
		errCh <- err
		return
	}

	server := http.Server{Addr: address, Handler: wsContainer, WriteTimeout: 2 * time.Minute}

	ln, err := net.Listen("tcp", address)
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
	if err := server.Serve(ln); err != nil {
		log.Errorf("%+v", err)
		if !h.restart {
			log.Infof("not in restart progress, broadcast error")
			errCh <- err
		}

		return
	}

	log.Infof("eurekaserver stop")
}

// 创建handler
func (h *EurekaServer) createRestfulContainer() (*restful.Container, error) {
	wsContainer := restful.NewContainer()
	wsContainer.Filter(h.process)
	wsContainer.Add(h.GetEurekaV2Server())
	wsContainer.Add(h.GetEurekaV1Server())
	wsContainer.Add(h.GetEurekaServer())
	return wsContainer, nil
}

// process 在接收和回复时统一处理请求
func (h *EurekaServer) process(req *restful.Request, rsp *restful.Response, chain *restful.FilterChain) {
	func() {
		if err := h.preprocess(req, rsp); err != nil {
			return
		}

		chain.ProcessFilter(req, rsp)
	}()

	h.postproccess(req, rsp)
}

func isImportantRequest(req *restful.Request) bool {
	if req.Request.Method == "POST" || req.Request.Method == "DELETE" {
		return true
	}
	urlStr := req.Request.URL.String()
	if req.Request.Method == "PUT" && strings.Contains(urlStr, "/status") {
		return true
	}
	return false
}

/**
 * @brief 请求预处理
 */
func (h *EurekaServer) preprocess(req *restful.Request, rsp *restful.Response) error {
	// 设置开始时间
	req.SetAttribute("start-time", time.Now())

	if isImportantRequest(req) {
		// 打印请求
		log.Info("receive request",
			zap.String("client-address", req.Request.RemoteAddr),
			zap.String("user-agent", req.HeaderParameter("User-Agent")),
			zap.String("method", req.Request.Method),
			zap.String("url", req.Request.URL.String()),
		)
	}
	// 限流
	if err := h.enterRateLimit(req, rsp); err != nil {
		return err
	}

	return nil
}

// 访问限制
func (h *EurekaServer) enterRateLimit(req *restful.Request, rsp *restful.Response) error {
	// 检查限流插件是否开启
	if h.rateLimit == nil {
		return nil
	}
	// IP级限流
	// 先获取当前请求的address
	address := req.Request.RemoteAddr
	segments := strings.Split(address, ":")
	if len(segments) != 2 {
		return nil
	}
	if ok := h.rateLimit.Allow(plugin.IPRatelimit, segments[0]); !ok {
		log.Error("ip ratelimit is not allow", zap.String("client", address))
		RateLimitResponse(rsp)
		return errors.New("ip ratelimit is not allow")
	}

	// 接口级限流
	apiName := fmt.Sprintf("%s:%s", req.Request.Method,
		strings.TrimSuffix(req.Request.URL.Path, "/"))
	if ok := h.rateLimit.Allow(plugin.APIRatelimit, apiName); !ok {
		log.Error("api ratelimit is not allow", zap.String("client", address), zap.String("api", apiName))
		RateLimitResponse(rsp)
		return errors.New("api ratelimit is not allow")
	}

	return nil
}

// RateLimitResponse http答复简单封装
func RateLimitResponse(rsp *restful.Response) {
	rsp.WriteHeader(http.StatusTooManyRequests)
	rsp.Header().Add(restful.HEADER_ContentType, restful.MIME_JSON)
}

/**
 * @brief 请求后处理：统计
 */
func (h *EurekaServer) postproccess(req *restful.Request, rsp *restful.Response) {
	now := time.Now()
	// 接口调用统计
	path := req.Request.URL.Path
	if path != "/" {
		// 去掉最后一个"/"
		path = strings.TrimSuffix(path, "/")
	}
	startTime := req.Attribute("start-time").(time.Time)
	code := rsp.StatusCode()
	polarisCode, ok := req.Attribute(statusCodeHeader).(uint32)
	if ok {
		code = int(polarisCode)
	}
	diff := now.Sub(startTime)
	// 打印耗时超过1s的请求
	if diff > time.Second {
		log.Info("handling time > 1s",
			zap.String("client-address", req.Request.RemoteAddr),
			zap.String("user-agent", req.HeaderParameter("User-Agent")),
			zap.String("method", req.Request.Method),
			zap.String("url", req.Request.URL.String()),
			zap.Duration("handling-time", diff),
		)
	}
	method := getEurekaApi(req.Request.Method, path)
	_ = h.statis.AddAPICall(method, "HTTP", code, diff.Nanoseconds())
}

// getEurekaApi 聚合 eureka 接口，不暴露服务名和实例 id
func getEurekaApi(method, path string) string {

	if path == "" {
		return ""
	}
	if !strings.HasPrefix(path, pathPrefix) {
		return path
	}

	pathSlashCount := strings.Count(path, "/")

	switch method {
	case http.MethodPost:
		if pathSlashCount == 3 {
			// POST:/eureka/apps/{application}
			return operationRegister
		}
	case http.MethodGet:
		if path == "/eureka/apps/delta" {
			return operationDelta
		}
		if pathSlashCount == 3 {
			// GET:/eureka/apps/{application}
			return operationAllAppIDInstances
		} else if pathSlashCount == 4 {
			// GET:/eureka/apps/{application}/{instanceid}
			return operationAppIDInstance
		}
	case http.MethodDelete:
		if pathSlashCount == 4 {
			// DELETE:/eureka/apps/{application}/{instanceid}
			return operationDeregister
		} else if strings.HasSuffix(path, statusSuffix) && pathSlashCount == 5 {
			// DELETE:/eureka/apps/{application}/{instanceid}/status
			return operationDeleteStatusChange
		}
	case http.MethodPut:
		if pathSlashCount == 4 {
			// PUT:/eureka/apps/{application}/{instanceid}
			return operationHeartbeat
		} else if strings.HasSuffix(path, statusSuffix) && pathSlashCount == 5 {
			// PUT:/eureka/apps/{application}/{instanceid}/status
			return operationStatusChange
		}
	}

	// GET:/eureka/apps 和其他无法识别的接口直接返回
	return method + ":" + path
}

// Stop 结束eurekaServer的运行
func (h *EurekaServer) Stop() {
	// 释放connLimit的数据，如果没有开启，也需要执行一下
	// 目的：防止restart的时候，connLimit冲突
	connlimit.RemoveLimitListener(h.GetProtocol())
	if h.server != nil {
		_ = h.server.Close()
	}
	h.worker.Stop()
}

// Restart 重启eurekaServer
func (h *EurekaServer) Restart(
	option map[string]interface{}, api map[string]apiserver.APIConfig, errCh chan error) error {
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

	if err := h.Initialize(context.Background(), option, api); err != nil {
		h.restart = false
		if initErr := h.Initialize(context.Background(), backupOption, backupAPI); initErr != nil {
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

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
// 来自net/http
type tcpKeepAliveListener struct {
	*net.TCPListener
}

// Accept 来自于net/http
func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	_ = tc.SetKeepAlive(true)
	_ = tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}
