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
	"github.com/emicklei/go-restful/v3"

	"github.com/polarismesh/polaris/apiserver"
	"github.com/polarismesh/polaris/apiserver/httpserver/docs"
	"github.com/polarismesh/polaris/namespace"
	"github.com/polarismesh/polaris/service"
	"github.com/polarismesh/polaris/service/healthcheck"
)

type HTTPServerV1 struct {
	namespaceServer   namespace.NamespaceOperateServer
	namingServer      service.DiscoverServer
	healthCheckServer *healthcheck.Server
}

func NewV1Server(
	namespaceServer namespace.NamespaceOperateServer,
	namingServer service.DiscoverServer,
	healthCheckServer *healthcheck.Server) *HTTPServerV1 {
	return &HTTPServerV1{
		namespaceServer:   namespaceServer,
		namingServer:      namingServer,
		healthCheckServer: healthCheckServer,
	}
}

const (
	defaultReadAccess    string = "default-read"
	defaultAccess        string = "default"
	serviceAccess        string = "service"
	circuitBreakerAccess string = "circuitbreaker"
	routingAccess        string = "router"
	rateLimitAccess      string = "ratelimit"
)

// GetConsoleAccessServer 注册管理端接口
func (h *HTTPServerV1) GetConsoleAccessServer(include []string) (*restful.WebService, error) {
	consoleAccess := []string{defaultAccess}

	ws := new(restful.WebService)

	ws.Path("/naming/v1").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

	// 如果为空，则开启全部接口
	if len(include) == 0 {
		include = consoleAccess
	}
	oldInclude := include

	for _, item := range oldInclude {
		if item == defaultReadAccess {
			include = []string{defaultReadAccess}
			break
		}
	}

	for _, item := range oldInclude {
		if item == defaultAccess {
			include = consoleAccess
			break
		}
	}

	for _, item := range include {
		switch item {
		case defaultReadAccess:
			h.addDefaultReadAccess(ws)
		case defaultAccess:
			h.addDefaultAccess(ws)
		case serviceAccess:
			h.addServiceAccess(ws)
		case circuitBreakerAccess:
			h.addCircuitBreakerRuleAccess(ws)
		case routingAccess:
			h.addRoutingRuleAccess(ws)
		case rateLimitAccess:
			h.addRateLimitRuleAccess(ws)
		}
	}
	return ws, nil
}

// addDefaultReadAccess 增加默认读接口
func (h *HTTPServerV1) addDefaultReadAccess(ws *restful.WebService) {
	// 管理端接口：只包含读接口
	ws.Route(docs.EnrichGetNamespacesApiDocs(ws.GET("/namespaces").To(h.GetNamespaces)))
	ws.Route(docs.EnrichGetServicesApiDocs(ws.GET("/services").To(h.GetServices)))
	ws.Route(docs.EnrichGetServicesCountApiDocs(ws.GET("/services/count").To(h.GetServicesCount)))
	ws.Route(docs.EnrichGetServiceAliasesApiDocs(ws.GET("/service/aliases").To(h.GetServiceAliases)))

	ws.Route(docs.EnrichGetInstancesApiDocs(ws.GET("/instances").To(h.GetInstances)))
	ws.Route(docs.EnrichGetInstancesCountApiDocs(ws.GET("/instances/count").To(h.GetInstancesCount)))
	ws.Route(docs.EnrichGetRateLimitsApiDocs(ws.GET("/ratelimits").To(h.GetRateLimits)))
	ws.Route(docs.EnrichGetCircuitBreakerRulesApiDocs(
		ws.GET("/circuitbreaker/rules").To(h.GetCircuitBreakerRules)))
	ws.Route(docs.EnrichGetFaultDetectRulesApiDocs(ws.GET("/faultdetectors").To(h.GetFaultDetectRules)))

	ws.Route(docs.EnrichGetServiceContractsApiDocs(
		ws.GET("/service/contracts").To(h.GetServiceContracts)))
	ws.Route(docs.EnrichGetServiceContractsApiDocs(
		ws.GET("/service/contract/versions").To(h.GetServiceContractVersions)))

	// Deprecate -- start
	ws.Route(ws.GET("/namespace/token").To(h.GetNamespaceToken))
	ws.Route(ws.GET("/service/token").To(h.GetServiceToken))
	ws.Route(ws.POST("/service/owner").To(h.GetServiceOwner))
	ws.Route(ws.GET("/service/circuitbreaker").To(h.GetCircuitBreakerByService))
	ws.Route(ws.GET("/circuitbreaker").To(h.GetCircuitBreaker))
	ws.Route(ws.GET("/circuitbreaker/versions").To(h.GetCircuitBreakerVersions))
	ws.Route(ws.GET("/circuitbreakers/master").To(h.GetMasterCircuitBreakers))
	ws.Route(ws.GET("/circuitbreakers/release").To(h.GetReleaseCircuitBreakers))
	ws.Route(ws.GET("/circuitbreaker/token").To(h.GetCircuitBreakerToken))
	ws.Route(ws.GET("/routings").To(h.GetRoutings))
	// Deprecate -- end
}

// addDefaultAccess 增加默认接口
func (h *HTTPServerV1) addDefaultAccess(ws *restful.WebService) {
	// 管理端接口：增删改查请求全部操作存储层
	h.addServiceAccess(ws)
	h.addRoutingRuleAccess(ws)
	h.addRateLimitRuleAccess(ws)
	h.addCircuitBreakerRuleAccess(ws)
}

// addServiceAccess .
func (h *HTTPServerV1) addServiceAccess(ws *restful.WebService) {
	ws.Route(docs.EnrichCreateNamespacesApiDocsOld(ws.POST("/namespaces").To(h.CreateNamespaces)))
	ws.Route(docs.EnrichDeleteNamespacesApiDocsOld(ws.POST("/namespaces/delete").To(h.DeleteNamespaces)))
	ws.Route(docs.EnrichUpdateNamespacesApiDocsOld(ws.PUT("/namespaces").To(h.UpdateNamespaces)))
	ws.Route(docs.EnrichGetNamespacesApiDocsOld(ws.GET("/namespaces").To(h.GetNamespaces)))
	ws.Route(docs.EnrichGetNamespaceTokenApiDocsOld(ws.GET("/namespace/token").To(h.GetNamespaceToken)))
	ws.Route(docs.EnrichUpdateNamespaceTokenApiDocsOld(
		ws.PUT("/namespace/token").To(h.UpdateNamespaceToken)))

	ws.Route(docs.EnrichCreateServicesApiDocs(ws.POST("/services").To(h.CreateServices)))
	ws.Route(docs.EnrichDeleteServicesApiDocs(ws.POST("/services/delete").To(h.DeleteServices)))
	ws.Route(docs.EnrichUpdateServicesApiDocs(ws.PUT("/services").To(h.UpdateServices)))
	ws.Route(docs.EnrichGetServicesApiDocs(ws.GET("/services").To(h.GetServices)))
	ws.Route(docs.EnrichGetAllServicesApiDocs(ws.GET("/services/all").To(h.GetAllServices)))
	ws.Route(docs.EnrichGetServicesCountApiDocs(ws.GET("/services/count").To(h.GetServicesCount)))
	ws.Route(docs.EnrichGetServiceTokenApiDocs(ws.GET("/service/token").To(h.GetServiceToken)))
	ws.Route(docs.EnrichUpdateServiceTokenApiDocs(ws.PUT("/service/token").To(h.UpdateServiceToken)))
	ws.Route(docs.EnrichCreateServiceAliasApiDocs(ws.POST("/service/alias").To(h.CreateServiceAlias)))
	ws.Route(docs.EnrichUpdateServiceAliasApiDocs(ws.PUT("/service/alias").To(h.UpdateServiceAlias)))
	ws.Route(docs.EnrichGetServiceAliasesApiDocs(ws.GET("/service/aliases").To(h.GetServiceAliases)))
	ws.Route(docs.EnrichDeleteServiceAliasesApiDocs(
		ws.POST("/service/aliases/delete").To(h.DeleteServiceAliases)))

	ws.Route(docs.EnrichCreateInstancesApiDocs(ws.POST("/instances").To(h.CreateInstances)))
	ws.Route(docs.EnrichDeleteInstancesApiDocs(ws.POST("/instances/delete").To(h.DeleteInstances)))
	ws.Route(docs.EnrichDeleteInstancesByHostApiDocs(
		ws.POST("/instances/delete/host").To(h.DeleteInstancesByHost)))
	ws.Route(docs.EnrichUpdateInstancesApiDocs(ws.PUT("/instances").To(h.UpdateInstances)))
	ws.Route(docs.EnrichUpdateInstancesIsolateApiDocs(
		ws.PUT("/instances/isolate/host").To(h.UpdateInstancesIsolate)))
	ws.Route(docs.EnrichGetInstancesApiDocs(ws.GET("/instances").To(h.GetInstances)))
	ws.Route(docs.EnrichGetInstancesCountApiDocs(ws.GET("/instances/count").To(h.GetInstancesCount)))
	ws.Route(docs.EnrichGetInstanceLabelsApiDocs(ws.GET("/instances/labels").To(h.GetInstanceLabels)))

	// 服务契约相关
	ws.Route(docs.EnrichCreateServiceContractsApiDocs(
		ws.POST("/service/contracts").To(h.CreateServiceContract)))
	ws.Route(docs.EnrichGetServiceContractsApiDocs(
		ws.GET("/service/contracts").To(h.GetServiceContracts)))
	ws.Route(docs.EnrichDeleteServiceContractsApiDocs(
		ws.POST("/service/contracts/delete").To(h.DeleteServiceContracts)))
	ws.Route(docs.EnrichGetServiceContractsApiDocs(
		ws.GET("/service/contract/versions").To(h.GetServiceContractVersions)))
	ws.Route(docs.EnrichAddServiceContractInterfacesApiDocs(
		ws.POST("/service/contract/methods").To(h.CreateServiceContractInterfaces)))
	ws.Route(docs.EnrichAppendServiceContractInterfacesApiDocs(
		ws.PUT("/service/contract/methods/append").To(h.AppendServiceContractInterfaces)))
	ws.Route(docs.EnrichDeleteServiceContractsApiDocs(
		ws.POST("/service/contract/methods/delete").To(h.DeleteServiceContractInterfaces)))

	ws.Route(ws.POST("/service/owner").To(h.GetServiceOwner))
}

// addRoutingRuleAccess 增加默认接口
func (h *HTTPServerV1) addRoutingRuleAccess(ws *restful.WebService) {
	// Deprecate -- start
	ws.Route(ws.POST("/routings").To(h.CreateRoutings))
	ws.Route(ws.POST("/routings/delete").To(h.DeleteRoutings))
	ws.Route(ws.PUT("/routings").To(h.UpdateRoutings))
	ws.Route(ws.GET("/routings").To(h.GetRoutings))
	// Deprecate -- end
}

func (h *HTTPServerV1) addRateLimitRuleAccess(ws *restful.WebService) {
	ws.Route(docs.EnrichCreateRateLimitsApiDocs(ws.POST("/ratelimits").To(h.CreateRateLimits)))
	ws.Route(docs.EnrichDeleteRateLimitsApiDocs(ws.POST("/ratelimits/delete").To(h.DeleteRateLimits)))
	ws.Route(docs.EnrichUpdateRateLimitsApiDocs(ws.PUT("/ratelimits").To(h.UpdateRateLimits)))
	ws.Route(docs.EnrichGetRateLimitsApiDocs(ws.GET("/ratelimits").To(h.GetRateLimits)))
	ws.Route(docs.EnrichEnableRateLimitsApiDocs(ws.PUT("/ratelimits/enable").To(h.EnableRateLimits)))
}

func (h *HTTPServerV1) addCircuitBreakerRuleAccess(ws *restful.WebService) {
	// Deprecate -- start
	ws.Route(ws.POST("/circuitbreakers").To(h.CreateCircuitBreakers))
	ws.Route(ws.POST("/circuitbreakers/version").To(h.CreateCircuitBreakerVersions))
	ws.Route(ws.POST("/circuitbreakers/delete").To(h.DeleteCircuitBreakers))
	ws.Route(ws.PUT("/circuitbreakers").To(h.UpdateCircuitBreakers))
	ws.Route(ws.POST("/circuitbreakers/release").To(h.ReleaseCircuitBreakers))
	ws.Route(ws.POST("/circuitbreakers/unbind").To(h.UnBindCircuitBreakers))
	ws.Route(ws.GET("/circuitbreaker").To(h.GetCircuitBreaker))
	ws.Route(ws.GET("/circuitbreaker/versions").To(h.GetCircuitBreakerVersions))
	ws.Route(ws.GET("/circuitbreakers/master").To(h.GetMasterCircuitBreakers))
	ws.Route(ws.GET("/circuitbreakers/release").To(h.GetReleaseCircuitBreakers))
	ws.Route(ws.GET("/circuitbreaker/token").To(h.GetCircuitBreakerToken))
	// Deprecate -- end

	ws.Route(docs.EnrichGetCircuitBreakerRulesApiDocs(
		ws.GET("/circuitbreaker/rules").To(h.GetCircuitBreakerRules)))
	ws.Route(docs.EnrichCreateCircuitBreakerRulesApiDocs(
		ws.POST("/circuitbreaker/rules").To(h.CreateCircuitBreakerRules)))
	ws.Route(docs.EnrichUpdateCircuitBreakerRulesApiDocs(
		ws.PUT("/circuitbreaker/rules").To(h.UpdateCircuitBreakerRules)))
	ws.Route(docs.EnrichDeleteCircuitBreakerRulesApiDocs(
		ws.POST("/circuitbreaker/rules/delete").To(h.DeleteCircuitBreakerRules)))
	ws.Route(docs.EnrichEnableCircuitBreakerRulesApiDocs(
		ws.PUT("/circuitbreaker/rules/enable").To(h.EnableCircuitBreakerRules)))
	ws.Route(docs.EnrichGetFaultDetectRulesApiDocs(
		ws.GET("/faultdetectors").To(h.GetFaultDetectRules)))
	ws.Route(docs.EnrichCreateFaultDetectRulesApiDocs(
		ws.POST("/faultdetectors").To(h.CreateFaultDetectRules)))
	ws.Route(docs.EnrichUpdateFaultDetectRulesApiDocs(
		ws.PUT("/faultdetectors").To(h.UpdateFaultDetectRules)))
	ws.Route(docs.EnrichDeleteFaultDetectRulesApiDocs(
		ws.POST("/faultdetectors/delete").To(h.DeleteFaultDetectRules)))
}

// GetClientAccessServer get client access server
func (h *HTTPServerV1) GetClientAccessServer(ws *restful.WebService, include []string) error {
	clientAccess := []string{apiserver.DiscoverAccess, apiserver.RegisterAccess, apiserver.HealthcheckAccess}

	// 如果为空，则开启全部接口
	if len(include) == 0 {
		include = clientAccess
	}

	// 客户端接口：增删改请求操作存储层，查请求访问缓存
	for _, item := range include {
		switch item {
		case apiserver.DiscoverAccess:
			h.addDiscoverAccess(ws)
		case apiserver.RegisterAccess:
			h.addRegisterAccess(ws)
		case apiserver.HealthcheckAccess:
			h.addHealthCheckAccess(ws)
		}
	}
	return nil
}

// addDiscoverAccess 增加服务发现接口
func (h *HTTPServerV1) addDiscoverAccess(ws *restful.WebService) {
	ws.Route(docs.EnrichReportClientApiDocs(ws.POST("/ReportClient").To(h.ReportClient)))
	ws.Route(docs.EnrichDiscoverApiDocs(ws.POST("/Discover").To(h.Discover)))
}

// addRegisterAccess 增加注册/反注册接口
func (h *HTTPServerV1) addRegisterAccess(ws *restful.WebService) {
	ws.Route(docs.EnrichRegisterInstanceApiDocs(ws.POST("/RegisterInstance").To(h.RegisterInstance)))
	ws.Route(docs.EnrichDeregisterInstanceApiDocs(ws.POST("/DeregisterInstance").To(h.DeregisterInstance)))
}

// addHealthCheckAccess 增加健康检查接口
func (h *HTTPServerV1) addHealthCheckAccess(ws *restful.WebService) {
	ws.Route(docs.EnrichHeartbeatApiDocs(ws.POST("/Heartbeat").To(h.Heartbeat)))
}
