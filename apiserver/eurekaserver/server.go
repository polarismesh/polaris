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
	"github.com/polarismesh/polaris-server/common/secure"
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
	tlsInfo                *secure.TLSInfo
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

// GetPort ????????????
func (h *EurekaServer) GetPort() uint32 {
	return h.listenPort
}

// GetProtocol ????????????
func (h *EurekaServer) GetProtocol() string {
	return ServerEureka
}

// Initialize ?????????HTTP API?????????
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

	// ????????????????????????
	if raw, _ := option[optionConnLimit].(map[interface{}]interface{}); raw != nil {
		connLimitConfig, err := connlimit.ParseConnLimitConfig(raw)
		if err != nil {
			return err
		}
		h.connLimitConfig = connLimitConfig
	}
	if raw, _ := option[optionTLS].(map[interface{}]interface{}); raw != nil {
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

// Run ??????HTTP API?????????
func (h *EurekaServer) Run(errCh chan error) {
	log.Infof("start eurekaserver")
	h.exitCh = make(chan struct{})
	h.start = true
	defer func() {
		close(h.exitCh)
		h.start = false
	}()
	var err error
	// ???????????????????????????
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
	// ?????????http server
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
	// ???????????????????????????
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

	// ??????????????????
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
	log.Infof("eurekaserver stop")
}

// ??????handler
func (h *EurekaServer) createRestfulContainer() (*restful.Container, error) {
	wsContainer := restful.NewContainer()
	wsContainer.Filter(h.process)
	wsContainer.Add(h.GetEurekaV2Server())
	wsContainer.Add(h.GetEurekaV1Server())
	wsContainer.Add(h.GetEurekaServer())
	return wsContainer, nil
}

// process ???????????????????????????????????????
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
 * @brief ???????????????
 */
func (h *EurekaServer) preprocess(req *restful.Request, rsp *restful.Response) error {
	// ??????????????????
	req.SetAttribute("start-time", time.Now())

	if isImportantRequest(req) {
		// ????????????
		log.Info("receive request",
			zap.String("client-address", req.Request.RemoteAddr),
			zap.String("user-agent", req.HeaderParameter("User-Agent")),
			zap.String("method", req.Request.Method),
			zap.String("url", req.Request.URL.String()),
		)
	}
	// ??????
	if err := h.enterRateLimit(req, rsp); err != nil {
		return err
	}

	return nil
}

// ????????????
func (h *EurekaServer) enterRateLimit(req *restful.Request, rsp *restful.Response) error {
	// ??????????????????????????????
	if h.rateLimit == nil {
		return nil
	}
	// IP?????????
	// ????????????????????????address
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

	// ???????????????
	apiName := fmt.Sprintf("%s:%s", req.Request.Method,
		strings.TrimSuffix(req.Request.URL.Path, "/"))
	if ok := h.rateLimit.Allow(plugin.APIRatelimit, apiName); !ok {
		log.Error("api ratelimit is not allow", zap.String("client", address), zap.String("api", apiName))
		RateLimitResponse(rsp)
		return errors.New("api ratelimit is not allow")
	}

	return nil
}

// RateLimitResponse http??????????????????
func RateLimitResponse(rsp *restful.Response) {
	rsp.WriteHeader(http.StatusTooManyRequests)
	rsp.Header().Add(restful.HEADER_ContentType, restful.MIME_JSON)
}

/**
 * @brief ????????????????????????
 */
func (h *EurekaServer) postproccess(req *restful.Request, rsp *restful.Response) {
	now := time.Now()
	// ??????????????????
	path := req.Request.URL.Path
	if path != "/" {
		// ??????????????????"/"
		path = strings.TrimSuffix(path, "/")
	}
	startTime := req.Attribute("start-time").(time.Time)

	recordApiCall := true
	code, ok := req.Attribute(statusCodeHeader).(uint32)
	if !ok {
		code = uint32(rsp.StatusCode())
		recordApiCall = code != http.StatusNotFound
	}
	diff := now.Sub(startTime)
	// ??????????????????1s?????????
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

	if recordApiCall {
		_ = h.statis.AddAPICall(method, "HTTP", int(code), diff.Nanoseconds())
	}
}

// getEurekaApi ?????? eureka ???????????????????????????????????? id
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

	// GET:/eureka/apps ??????????????????????????????????????????
	return method + ":" + path
}

// Stop ??????eurekaServer?????????
func (h *EurekaServer) Stop() {
	// ??????connLimit??????????????????????????????????????????????????????
	// ???????????????restart????????????connLimit??????
	connlimit.RemoveLimitListener(h.GetProtocol())
	if h.server != nil {
		_ = h.server.Close()
	}
	h.worker.Stop()
}

// Restart ??????eurekaServer
func (h *EurekaServer) Restart(
	option map[string]interface{}, api map[string]apiserver.APIConfig, errCh chan error) error {
	log.Infof("restart httpserver new config: %+v", option)
	// ????????????option
	backupOption := h.option
	// ????????????api
	backupAPI := h.openAPI

	// ??????restart???????????????stop????????????????????????
	h.restart = true
	// ??????httpserver
	h.Stop()
	// ??????httpserver??????
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
// ??????net/http
type tcpKeepAliveListener struct {
	*net.TCPListener
}

// Accept ?????????net/http
func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	_ = tc.SetKeepAlive(true)
	_ = tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}
