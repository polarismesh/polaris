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
	restfulspec "github.com/polarismesh/go-restful-openapi/v2"
	apifault "github.com/polarismesh/specification/source/go/api/v1/fault_tolerance"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
)

var (
	namespacesApiTags          = []string{"Namespaces"}
	servicesApiTags            = []string{"Services"}
	instancesApiTags           = []string{"Instances"}
	routingRulesApiTags        = []string{"RoutingRules"}
	rateLimitsApiTags          = []string{"RateLimits"}
	circuitBreakersApiTags     = []string{"CircuitBreakers"}
	circuitBreakerRulesApiTags = []string{"CircuitBreakerRules"}
	faultDetectsApiTags        = []string{"FaultDetects"}
)

func enrichGetNamespacesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("获取命名空间列表").
		Metadata(restfulspec.KeyOpenAPITags, namespacesApiTags).
		Param(restful.QueryParameter("name", "命名空间唯一名称").
			DataType("string").Required(true)).
		Param(restful.QueryParameter("offset", "查询偏移量").
			DataType("integer").Required(false).DefaultValue("0")).
		Param(restful.QueryParameter("limit", "查询条数，**最多查询100条**").
			DataType("integer").Required(false)).
		Notes(enrichGetNamespacesApiNotes)
}

func enrichCreateNamespacesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("创建命名空间").
		Metadata(restfulspec.KeyOpenAPITags, namespacesApiTags).
		Reads([]apimodel.Namespace{}, "create namespaces").
		Notes(enrichCreateNamespacesApiNotes)
}

func enrichDeleteNamespacesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("删除命名空间").
		Metadata(restfulspec.KeyOpenAPITags, namespacesApiTags).
		Reads([]apimodel.Namespace{}, "delete namespaces").
		Notes(enrichDeleteNamespacesApiNotes)
}

func enrichUpdateNamespacesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("更新命名空间").
		Metadata(restfulspec.KeyOpenAPITags, namespacesApiTags).
		Reads([]apimodel.Namespace{}, "update namespaces").
		Notes(enrichUpdateNamespacesApiNotes)
}

func enrichGetNamespaceTokenApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("查询命名空间Token").
		Metadata(restfulspec.KeyOpenAPITags, namespacesApiTags).Deprecate()
}

func enrichUpdateNamespaceTokenApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("更新命名空间Token").
		Metadata(restfulspec.KeyOpenAPITags, namespacesApiTags).Deprecate()
}

func enrichGetServicesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("获取服务列表").
		Metadata(restfulspec.KeyOpenAPITags, servicesApiTags).
		Param(restful.QueryParameter("name", "服务名").DataType("string").
			Required(false).
			DefaultValue("demo-service")).
		Param(restful.QueryParameter("namespace", "命名空间").DataType("string").
			Required(false).
			DefaultValue("default")).
		Param(restful.QueryParameter("business", "业务，默认模糊查询").DataType("string").
			Required(false)).
		Param(restful.QueryParameter("department", "部门").DataType("string").
			Required(false)).
		Param(restful.QueryParameter("host", "实例IP，**多个IP以英文逗号分隔**").
			DataType("string").Required(false)).
		Param(restful.QueryParameter("port", "**实例端口**，**多个端口以英文逗号分隔** ").
			DataType("string").Required(false)).
		Param(restful.QueryParameter("keys", "服务元数据名，keys和values需要同时填写，"+
			"目前只支持查询一组元数据。").DataType("string").Required(false)).
		Param(restful.QueryParameter("values", "服务元数据名，keys和values需要同时填写，"+
			"目前只支持查询一组元数据。").DataType("string").Required(false)).
		Param(restful.QueryParameter("offset", "查询偏移量").DataType("integer").
			Required(false).DefaultValue("0")).
		Param(restful.QueryParameter("limit", "查询条数，**最多查询100条**").DataType("integer").
			Required(false)).
		Notes(enrichGetServicesApiNotes)
}

func enrichCreateServicesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("创建服务").
		Metadata(restfulspec.KeyOpenAPITags, servicesApiTags).
		Reads([]apiservice.Service{}, "create services").
		Notes(enrichCreateServicesApiNotes)
}

func enrichDeleteServicesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("删除服务").
		Metadata(restfulspec.KeyOpenAPITags, servicesApiTags).
		Reads([]apiservice.Service{}, "delete services").
		Notes(enrichDeleteServicesApiNotes)
}

func enrichUpdateServicesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("更新服务").
		Metadata(restfulspec.KeyOpenAPITags, servicesApiTags).
		Reads([]apiservice.Service{}, "update services").
		Notes(enrichUpdateServicesApiNotes)
}

func enrichGetServicesCountApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("获取服务数量").
		Metadata(restfulspec.KeyOpenAPITags, servicesApiTags).
		Notes(enrichGetServicesCountApiNotes)
}

func enrichGetServiceTokenApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("查询服务Token").
		Metadata(restfulspec.KeyOpenAPITags, servicesApiTags).Deprecate()
}

func enrichUpdateServiceTokenApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("更新服务Token").
		Metadata(restfulspec.KeyOpenAPITags, servicesApiTags).Deprecate()
}

func enrichCreateServiceAliasApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("创建服务别名").
		Metadata(restfulspec.KeyOpenAPITags, servicesApiTags).
		Reads(apiservice.ServiceAlias{}, "create service alias").
		Notes(enrichCreateServiceAliasApiNotes)
}

func enrichUpdateServiceAliasApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("更新服务别名").
		Metadata(restfulspec.KeyOpenAPITags, servicesApiTags).
		Reads(apiservice.ServiceAlias{}, "update service alias").
		Notes(enrichUpdateServiceAliasApiNotes)
}

func enrichGetServiceAliasesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("查询服务别名").
		Metadata(restfulspec.KeyOpenAPITags, servicesApiTags).
		Notes(enrichGetServiceAliasesApiNotes)
}

func enrichDeleteServiceAliasesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("删除服务别名").
		Metadata(restfulspec.KeyOpenAPITags, servicesApiTags).
		Reads([]apiservice.ServiceAlias{}, "delete service aliases").
		Notes(enrichDeleteServiceAliasesApiNotes)
}

func enrichGetCircuitBreakerByServiceApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("根据服务查询熔断规则").
		Metadata(restfulspec.KeyOpenAPITags, servicesApiTags).
		Param(restful.PathParameter("service", "服务名").DataType("string").
			Required(true)).
		Param(restful.PathParameter("namespace", "命名空间").DataType("string").
			Required(true)).
		Notes(enrichGetCircuitBreakerByServiceApiNotes)
}

func enrichGetServiceOwnerApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("根据服务获取服务负责人").
		Metadata(restfulspec.KeyOpenAPITags, servicesApiTags).
		Notes(enrichGetServiceOwnerApiNotes)
}

func enrichCreateInstancesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("创建实例").
		Metadata(restfulspec.KeyOpenAPITags, instancesApiTags).
		Reads([]apiservice.Instance{}, "create instances").
		Notes(enrichCreateInstancesApiNotes)
}

func enrichDeleteInstancesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("删除实例(根据实例ID)").
		Metadata(restfulspec.KeyOpenAPITags, instancesApiTags).
		Reads([]apiservice.Instance{}, "delete instances").
		Notes(enrichDeleteInstancesApiNotes)
}

func enrichDeleteInstancesByHostApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("删除实例(根据主机)").
		Metadata(restfulspec.KeyOpenAPITags, instancesApiTags).
		Reads([]apiservice.Instance{}, "delete instances").
		Notes(enrichDeleteInstancesByHostApiNotes)
}

func enrichUpdateInstancesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("更新实例").
		Metadata(restfulspec.KeyOpenAPITags, instancesApiTags).
		Reads([]apiservice.Instance{}, "update instances").
		Notes(enrichUpdateInstancesApiNotes)
}

func enrichUpdateInstancesIsolateApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("修改服务实例的隔离状态").
		Metadata(restfulspec.KeyOpenAPITags, instancesApiTags).
		Reads([]apiservice.Instance{}, "update instances").
		Notes(enrichUpdateInstancesIsolateApiNotes)
}

func enrichGetInstancesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("查询服务实例").
		Metadata(restfulspec.KeyOpenAPITags, instancesApiTags).
		Param(restful.PathParameter("service", "服务名称").
			DataType("string").Required(true)).
		Param(restful.PathParameter("namespace", "命名空间").
			DataType("string").Required(true)).
		Param(restful.PathParameter("host", "实例IP").
			DataType("string").Required(true)).
		Param(restful.PathParameter("keys", "标签key").
			DataType("string").Required(false)).
		Param(restful.PathParameter("values", "标签value").
			DataType("string").Required(false)).
		Param(restful.PathParameter("healthy", "实例健康状态").
			DataType("string").Required(false)).
		Param(restful.PathParameter("isolate", "实例隔离状态").
			DataType("string").Required(false)).
		Param(restful.PathParameter("protocol", "实例端口协议状态").
			DataType("string").Required(false)).
		Param(restful.PathParameter("version", "实例版本").
			DataType("string").Required(false)).
		Param(restful.PathParameter("cmdb_region", "实例region信息").
			DataType("string").Required(false)).
		Param(restful.PathParameter("cmdb_zone", "实例zone信息").
			DataType("string").Required(false)).
		Param(restful.PathParameter("cmdb_idc", "实例idc信息").
			DataType("string").Required(false)).
		Param(restful.PathParameter("offset", "查询偏移量").
			DataType("integer").Required(false)).
		Param(restful.PathParameter("limit", "查询条数").
			DataType("integer").Required(false)).
		Notes(enrichGetInstancesApiNotes)
}

func enrichGetInstancesCountApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("查询服务实例数量").
		Metadata(restfulspec.KeyOpenAPITags, instancesApiTags).
		Notes(enrichGetInstancesCountApiNotes)
}

func enrichGetInstanceLabelsApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("查询某个服务下所有实例的标签信息").
		Metadata(restfulspec.KeyOpenAPITags, instancesApiTags).
		Param(restful.QueryParameter("service", "服务名称").
			DataType("string").Required(true)).
		Param(restful.QueryParameter("namespace", "命名空间").
			DataType("string").Required(true)).
		Notes(enrichGetInstanceLabelsApiNotes)
}

func enrichCreateRoutingsApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("创建路由规则").
		Metadata(restfulspec.KeyOpenAPITags, routingRulesApiTags).
		Reads([]apitraffic.Routing{}, "create routing rules").
		Notes(enrichCreateRoutingsApiNotes)
}

func enrichDeleteRoutingsApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("删除路由规则").
		Metadata(restfulspec.KeyOpenAPITags, routingRulesApiTags).
		Reads([]apitraffic.Routing{}, "delete routing rules").
		Notes(enrichDeleteRoutingsApiNotes)
}

func enrichUpdateRoutingsApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("更新路由规则").
		Metadata(restfulspec.KeyOpenAPITags, routingRulesApiTags).
		Reads([]apitraffic.Routing{}, "update routing rules").
		Notes(enrichUpdateRoutingsApiNotes)
}

func enrichGetRoutingsApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("查询路由规则").
		Metadata(restfulspec.KeyOpenAPITags, routingRulesApiTags).
		Param(restful.PathParameter("service", "服务名称").DataType("string").
			Required(false)).
		Param(restful.PathParameter("namespace", "命名空间").DataType("string").
			Required(false)).
		Param(restful.PathParameter("offset", "分页的起始位置，默认为0").DataType("integer").
			Required(false).
			DefaultValue("0")).
		Param(restful.PathParameter("limit", "每页行数，默认100").DataType("integer").
			Required(false).
			DefaultValue("100")).
		Notes(enrichGetRoutingsApiNotes)
}

func enrichCreateRateLimitsApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("创建限流规则").
		Metadata(restfulspec.KeyOpenAPITags, rateLimitsApiTags).
		Reads([]apitraffic.RateLimit{}, "create rate limits").
		Notes(enrichCreateRateLimitsApiNotes)
}

func enrichDeleteRateLimitsApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("删除限流规则").
		Metadata(restfulspec.KeyOpenAPITags, rateLimitsApiTags).
		Reads([]apitraffic.RateLimit{}, "delete rate limits").
		Notes(enrichDeleteRateLimitsApiNotes)
}

func enrichUpdateRateLimitsApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("更新限流规则").
		Metadata(restfulspec.KeyOpenAPITags, rateLimitsApiTags).
		Reads([]apitraffic.RateLimit{}, "update rate limits").
		Notes(enrichUpdateRateLimitsApiNotes)
}

func enrichGetRateLimitsApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("查询限流规则").
		Metadata(restfulspec.KeyOpenAPITags, rateLimitsApiTags).
		Param(restful.PathParameter("id", "规则ID").
			DataType("string").Required(false)).
		Param(restful.PathParameter("name", "规则名称").
			DataType("string").Required(false)).
		Param(restful.PathParameter("service", "服务名称").
			DataType("string").Required(false)).
		Param(restful.PathParameter("namespace", "命名空间").
			DataType("string").Required(false)).
		Param(restful.PathParameter("method", "限流接口名，默认为模糊匹配 ").
			DataType("string").Required(false)).
		Param(restful.PathParameter("disable", "规则是否启用，true为不启用，false为启用").
			DataType("boolean").Required(false)).
		Param(restful.PathParameter("brief",
			"是否只显示概要信息，brief=true时，只返回规则列表概要信息，默认为false").
			DataType("boolean").Required(false).DefaultValue("false")).
		Param(restful.PathParameter("offset", "分页的起始位置，默认为0").DataType("integer").
			Required(false).DefaultValue("0")).
		Param(restful.PathParameter("limit", "每页行数，默认100  ").DataType("integer").
			Required(false).DefaultValue("100")).
		Notes(enrichGetRateLimitsApiNotes)
}

func enrichEnableRateLimitsApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("启用限流规则").
		Metadata(restfulspec.KeyOpenAPITags, rateLimitsApiTags).
		Reads([]apitraffic.RateLimit{}, "enable rate limits").
		Notes(enrichEnableRateLimitsApiNotes)
}

func enrichCreateCircuitBreakersApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("创建熔断规则").
		Metadata(restfulspec.KeyOpenAPITags, circuitBreakersApiTags).
		Reads([]apifault.CircuitBreaker{}, "create circuit breakers").
		Notes(enrichCreateCircuitBreakersApiNotes)
}

func enrichCreateCircuitBreakerVersionsApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("创建熔断规则版本").
		Metadata(restfulspec.KeyOpenAPITags, circuitBreakersApiTags).
		Reads([]apifault.CircuitBreaker{}, "create circuit breaker versions").
		Notes(enrichCreateCircuitBreakerVersionsApiNotes)
}

func enrichDeleteCircuitBreakersApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("删除熔断规则").
		Metadata(restfulspec.KeyOpenAPITags, circuitBreakersApiTags).
		Reads([]apifault.CircuitBreaker{}, "delete circuit breakers").
		Notes(enrichDeleteCircuitBreakersApiNotes)
}

func enrichUpdateCircuitBreakersApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("更新熔断规则").
		Metadata(restfulspec.KeyOpenAPITags, circuitBreakersApiTags).
		Reads([]apifault.CircuitBreaker{}, "update circuit breakers").
		Notes(enrichUpdateCircuitBreakersApiNotes)
}

func enrichReleaseCircuitBreakersApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("发布熔断规则").
		Metadata(restfulspec.KeyOpenAPITags, circuitBreakersApiTags).
		Reads([]apiservice.ConfigRelease{}, "release circuit breakers").
		Notes(enrichReleaseCircuitBreakersApiNotes)
}

func enrichUnBindCircuitBreakersApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("解绑熔断规则").
		Metadata(restfulspec.KeyOpenAPITags, circuitBreakersApiTags).
		Reads([]apiservice.ConfigRelease{}, "unbind circuit breakers").
		Notes(enrichUnBindCircuitBreakersApiNotes)
}

func enrichGetCircuitBreakersApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("查询熔断规则").
		Metadata(restfulspec.KeyOpenAPITags, circuitBreakersApiTags).
		Param(restful.PathParameter("id", "规则ID").
			DataType("string").Required(true)).
		Param(restful.PathParameter("version", "版本").
			DataType("string").Required(true)).
		Notes(enrichGetCircuitBreakersApiNotes)
}

func enrichGetCircuitBreakerVersionsApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("查询熔断规则版本").
		Metadata(restfulspec.KeyOpenAPITags, circuitBreakersApiTags).
		Param(restful.PathParameter("id", "规则ID").
			DataType("string").Required(true)).
		Notes(enrichGetCircuitBreakerVersionsApiNotes)
}

func enrichGetMasterCircuitBreakersApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("查询熔断规则Master版本").
		Metadata(restfulspec.KeyOpenAPITags, circuitBreakersApiTags).
		Param(restful.PathParameter("id", "规则ID").
			DataType("string").Required(true)).
		Notes(enrichGetMasterCircuitBreakersApiNotes)
}

func enrichGetReleaseCircuitBreakersApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("根据规则id查询已发布的熔断规则").
		Metadata(restfulspec.KeyOpenAPITags, circuitBreakersApiTags).
		Param(restful.PathParameter("id", "规则ID").
			DataType("string").Required(true)).
		Notes(enrichGetReleaseCircuitBreakersApiNotes)
}

func enrichGetCircuitBreakerTokensApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("查询熔断规则Token").
		Metadata(restfulspec.KeyOpenAPITags, circuitBreakersApiTags).Deprecate()
}

func enrichCreateCircuitBreakerRulesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("创建熔断规则").
		Metadata(restfulspec.KeyOpenAPITags, circuitBreakerRulesApiTags).
		Reads([]apifault.CircuitBreakerRule{}, "create circuitbreaker rules").
		Notes(enrichCreateCircuitBreakerRulesApiNotes)
}

func enrichDeleteCircuitBreakerRulesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("删除熔断规则").
		Metadata(restfulspec.KeyOpenAPITags, circuitBreakerRulesApiTags).
		Reads([]apifault.CircuitBreakerRule{}, "delete circuitbreaker rules").
		Notes(enrichDeleteCircuitBreakerRulesApiNotes)
}

func enrichUpdateCircuitBreakerRulesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("更新熔断规则").
		Metadata(restfulspec.KeyOpenAPITags, circuitBreakerRulesApiTags).
		Reads([]apifault.CircuitBreakerRule{}, "update circuitbreaker rules").
		Notes(enrichUpdateCircuitBreakerRulesApiNotes)
}

func enrichGetCircuitBreakerRulesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("查询熔断规则").
		Metadata(restfulspec.KeyOpenAPITags, circuitBreakerRulesApiTags).
		Param(restful.PathParameter("brief", "是否只显示概要信息，brief=true时，则不返回规则详情，"+
			"只返回规则列表概要信息，默认为false").DataType("boolean").
			Required(false).DefaultValue("false")).
		Param(restful.PathParameter("offset", "分页的起始位置，默认为0").DataType("integer").
			Required(false).DefaultValue("0")).
		Param(restful.PathParameter("limit", "每页行数，默认100  ").DataType("integer").
			Required(false).DefaultValue("100")).
		Param(restful.PathParameter("id", "规则ID").DataType("string").
			Required(false)).
		Param(restful.PathParameter("name", "规则名称").DataType("string").
			Required(false)).
		Param(restful.PathParameter("enable", "规则是否启用，true为启用，false为不启用").
			DataType("boolean").Required(false)).
		Param(restful.PathParameter("namespace", "命名空间").DataType("string").
			Required(false)).
		Param(restful.PathParameter("level", "熔断级别，可输入多个，逗号分割：1服务，2接口，3分组，4实例").
			DataType("string").Required(true)).
		Param(restful.PathParameter("service", "规则所关联服务，必须和serviceNamespace一起用").
			DataType("string").Required(true)).
		Param(restful.PathParameter("serviceNamespace", "规则所关联服务的命名空间，必须和service一起用").
			DataType("string").Required(true)).
		Param(restful.PathParameter("srcService", "规则的源服务名，模糊匹配").
			DataType("string").Required(true)).
		Param(restful.PathParameter("srcNamespace", "规则的源命名空间，模糊匹配").
			DataType("string").Required(true)).
		Param(restful.PathParameter("dstService", "规则的目标服务名，模糊匹配").
			DataType("string").Required(true)).
		Param(restful.PathParameter("dstNamespace", "规则的目标命名空间，模糊匹配").
			DataType("string").Required(true)).
		Param(restful.PathParameter("dstMethod", "规则的目标方法名，模糊匹配").
			DataType("string").Required(true)).
		Param(restful.PathParameter("description", "规则描述，模糊匹配").
			DataType("string").Required(true)).
		Notes(enrichGetCircuitBreakerRulesApiNotes)
}

func enrichEnableCircuitBreakerRulesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("启用限流规则").
		Metadata(restfulspec.KeyOpenAPITags, circuitBreakerRulesApiTags).
		Reads([]apitraffic.RateLimit{}, "enable rate limits").
		Notes(enrichEnableCircuitBreakerRulesApiNotes)
}

func enrichCreateFaultDetectRulesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("创建健康检查规则").
		Metadata(restfulspec.KeyOpenAPITags, faultDetectsApiTags).
		Reads([]apifault.FaultDetectRule{}, "create fault detect rules").
		Notes(enrichCreateFaultDetectRulesApiNotes)
}

func enrichDeleteFaultDetectRulesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("删除健康检查规则").
		Metadata(restfulspec.KeyOpenAPITags, faultDetectsApiTags).
		Reads([]apifault.FaultDetectRule{}, "delete fault detect rules").
		Notes(enrichDeleteFaultDetectRulesApiNotes)
}

func enrichUpdateFaultDetectRulesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("更新熔断规则").
		Metadata(restfulspec.KeyOpenAPITags, faultDetectsApiTags).
		Reads([]apifault.FaultDetectRule{}, "update fault detect rules").
		Notes(enrichUpdateFaultDetectRulesApiNotes)
}

func enrichGetFaultDetectRulesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("查询熔断规则").
		Metadata(restfulspec.KeyOpenAPITags, faultDetectsApiTags).
		Param(restful.PathParameter("brief", "是否只显示概要信息，brief=true时，"+
			"则不返回规则详情，只返回规则列表概要信息，默认为false").DataType("boolean").
			Required(false).DefaultValue("false")).
		Param(restful.PathParameter("offset", "分页的起始位置，默认为0").DataType("integer").
			Required(false).DefaultValue("0")).
		Param(restful.PathParameter("limit", "每页行数，默认100  ").DataType("integer").
			Required(false).DefaultValue("100")).
		Param(restful.PathParameter("id", "规则ID").DataType("string").
			Required(false)).
		Param(restful.PathParameter("name", "规则名称").DataType("string").
			Required(false)).
		Param(restful.PathParameter("enable", "规则是否启用，true为启用，false为不启用").
			DataType("boolean").Required(false)).
		Param(restful.PathParameter("namespace", "命名空间").DataType("string").
			Required(false)).
		Param(restful.PathParameter("service", "规则所关联服务，必须和serviceNamespace一起用").
			DataType("string").Required(true)).
		Param(restful.PathParameter("serviceNamespace", "规则所关联服务的命名空间，必须和service一起用").
			DataType("string").Required(true)).
		Param(restful.PathParameter("dstService", "规则的目标服务名，模糊匹配").
			DataType("string").Required(true)).
		Param(restful.PathParameter("dstNamespace", "规则的目标命名空间，模糊匹配").
			DataType("string").Required(true)).
		Param(restful.PathParameter("dstMethod", "规则的目标方法名，模糊匹配").
			DataType("string").Required(true)).
		Param(restful.PathParameter("description", "规则描述，模糊匹配").
			DataType("string").Required(true)).
		Notes(enrichGetFaultDetectRulesApiNotes)
}
