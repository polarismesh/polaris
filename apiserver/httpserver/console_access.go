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
	"github.com/polarismesh/polaris-server/common/log"
	"net/http"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/emicklei/go-restful"
)

const (
	defaultReadAccess string = "default-read"
	defaultAccess     string = "default"
)

/**
 * @brief 注册管理端接口
 */
func (h *Httpserver) GetConsoleAccessServer(include []string) (*restful.WebService, error) {
	consoleAccess := []string{defaultAccess}

	ws := new(restful.WebService)

	ws.Path("/naming/v1").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

	// 如果为空，则开启全部接口
	if len(include) == 0 {
		include = consoleAccess
	}

	var hasDefault = false
	for _, item := range include {
		if item == defaultAccess {
			hasDefault = true
			break
		}
	}
	for _, item := range include {
		switch item {
		case defaultReadAccess:
			if !hasDefault {
				h.addDefaultReadAccess(ws)
			}
		case defaultAccess:
			h.addDefaultAccess(ws)
		default:
			log.Errorf("method %s does not exist in httpserver console access", item)
			return nil, fmt.Errorf("method %s does not exist in httpserver console access", item)
		}
	}
	return ws, nil
}

/**
 * @brief 增加默认读接口
 */
func (h *Httpserver) addDefaultReadAccess(ws *restful.WebService) {
	// 管理端接口：只包含读接口
	ws.Route(ws.GET("/namespaces").To(h.GetNamespaces))
	ws.Route(ws.GET("/namespace/token").To(h.GetNamespaceToken))

	ws.Route(ws.GET("/services").To(h.GetServices))
	ws.Route(ws.GET("/services/count").To(h.GetServicesCount))
	ws.Route(ws.GET("/service/token").To(h.GetServiceToken))
	ws.Route(ws.POST("/service/alias").To(h.CreateServiceAlias))
	ws.Route(ws.GET("/service/aliases").To(h.GetServiceAliases))
	ws.Route(ws.GET("/service/circuitbreaker").To(h.GetCircuitBreakerByService))
	ws.Route(ws.POST("/service/owner").To(h.GetServiceOwner))

	ws.Route(ws.GET("/instances").To(h.GetInstances))
	ws.Route(ws.GET("/instances/count").To(h.GetInstancesCount))

	ws.Route(ws.POST("/routings").To(h.CreateRoutings))
	ws.Route(ws.GET("/routings").To(h.GetRoutings))

	ws.Route(ws.POST("/ratelimits").To(h.CreateRateLimits))
	ws.Route(ws.GET("/ratelimits").To(h.GetRateLimits))

	ws.Route(ws.GET("/circuitbreaker").To(h.GetCircuitBreaker))
	ws.Route(ws.GET("/circuitbreaker/versions").To(h.GetCircuitBreakerVersions))
	ws.Route(ws.GET("/circuitbreakers/master").To(h.GetMasterCircuitBreakers))
	ws.Route(ws.GET("/circuitbreakers/release").To(h.GetReleaseCircuitBreakers))
	ws.Route(ws.GET("/circuitbreaker/token").To(h.GetCircuitBreakerToken))

	ws.Route(ws.GET("/platforms").To(h.GetPlatforms))
	ws.Route(ws.GET("/platform/token").To(h.GetPlatformToken))
}

/**
 * @brief 增加默认接口
 */
func (h *Httpserver) addDefaultAccess(ws *restful.WebService) {
	// 管理端接口：增删改查请求全部操作存储层
	ws.Route(ws.POST("/namespaces").To(h.CreateNamespaces))
	ws.Route(ws.POST("/namespaces/delete").To(h.DeleteNamespaces))
	ws.Route(ws.PUT("/namespaces").To(h.UpdateNamespaces))
	ws.Route(ws.GET("/namespaces").To(h.GetNamespaces))
	ws.Route(ws.GET("/namespace/token").To(h.GetNamespaceToken))
	ws.Route(ws.PUT("/namespace/token").To(h.UpdateNamespaceToken))

	ws.Route(ws.POST("/services").To(h.CreateServices))
	ws.Route(ws.POST("/services/delete").To(h.DeleteServices))
	ws.Route(ws.PUT("/services").To(h.UpdateServices))
	ws.Route(ws.GET("/services").To(h.GetServices))
	ws.Route(ws.GET("/services/count").To(h.GetServicesCount))
	ws.Route(ws.GET("/service/token").To(h.GetServiceToken))
	ws.Route(ws.PUT("/service/token").To(h.UpdateServiceToken))
	ws.Route(ws.POST("/service/alias").To(h.CreateServiceAlias))
	ws.Route(ws.POST("/service/alias/no-auth").To(h.CreateServiceAliasNoAuth))
	ws.Route(ws.PUT("/service/alias").To(h.UpdateServiceAlias))
	ws.Route(ws.GET("/service/aliases").To(h.GetServiceAliases))
	ws.Route(ws.GET("/service/circuitbreaker").To(h.GetCircuitBreakerByService))
	ws.Route(ws.POST("/service/owner").To(h.GetServiceOwner))

	ws.Route(ws.POST("/instances").To(h.CreateInstances))
	ws.Route(ws.POST("/instances/delete").To(h.DeleteInstances))
	ws.Route(ws.POST("/instances/delete/host").To(h.DeleteInstancesByHost))
	ws.Route(ws.PUT("/instances").To(h.UpdateInstances))
	ws.Route(ws.PUT("/instances/isolate/host").To(h.UpdateInstancesIsolate))
	ws.Route(ws.GET("/instances").To(h.GetInstances))
	ws.Route(ws.GET("/instances/count").To(h.GetInstancesCount))

	ws.Route(ws.POST("/routings").To(h.CreateRoutings))
	ws.Route(ws.POST("/routings/delete").To(h.DeleteRoutings))
	ws.Route(ws.PUT("/routings").To(h.UpdateRoutings))
	ws.Route(ws.GET("/routings").To(h.GetRoutings))

	ws.Route(ws.POST("/ratelimits").To(h.CreateRateLimits))
	ws.Route(ws.POST("/ratelimits/delete").To(h.DeleteRateLimits))
	ws.Route(ws.PUT("/ratelimits").To(h.UpdateRateLimits))
	ws.Route(ws.GET("/ratelimits").To(h.GetRateLimits))

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

	ws.Route(ws.POST("/platforms").To(h.CreatePlatforms))
	ws.Route(ws.POST("/platforms/delete").To(h.DeletePlatforms))
	ws.Route(ws.PUT("/platforms").To(h.UpdatePlatforms))
	ws.Route(ws.GET("/platforms").To(h.GetPlatforms))
	ws.Route(ws.GET("/platform/token").To(h.GetPlatformToken))

}

/**
 * @brief 创建命名空间
 */
func (h *Httpserver) CreateNamespaces(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var namespaces NamespaceArr
	ctx, err := handler.Parse(&namespaces)
	if ctx == nil {
		ret := api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	ret := h.namingServer.CreateNamespaces(ctx, namespaces)
	handler.WriteHeaderAndProto(ret)
}

/**
 * @brief 删除命名空间
 */
func (h *Httpserver) DeleteNamespaces(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var namespaces NamespaceArr
	ctx, err := handler.Parse(&namespaces)
	if ctx == nil {
		ret := api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	ret := h.namingServer.DeleteNamespaces(ctx, namespaces)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}

	handler.WriteHeader(ret.GetCode().GetValue(), http.StatusOK)
}

/**
 * @brief 修改命名空间
 */
func (h *Httpserver) UpdateNamespaces(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var namespaces NamespaceArr
	ctx, err := handler.Parse(&namespaces)
	if ctx == nil {
		ret := api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	ret := h.namingServer.UpdateNamespaces(ctx, namespaces)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}

	handler.WriteHeader(ret.GetCode().GetValue(), http.StatusOK)
}

/**
 * @brief 查询命名空间
 */
func (h *Httpserver) GetNamespaces(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	ret := h.namingServer.GetNamespaces(req.Request.URL.Query())
	handler.WriteHeaderAndProto(ret)
}

// 命名空间token的获取
func (h *Httpserver) GetNamespaceToken(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	token := req.HeaderParameter("Polaris-Token")
	ctx := context.WithValue(context.Background(), utils.StringContext("polaris-token"), token)

	queryParams := parseQueryParams(req)
	namespace := &api.Namespace{
		Name:  utils.NewStringValue(queryParams["name"]),
		Token: utils.NewStringValue(queryParams["token"]),
	}

	ret := h.namingServer.GetNamespaceToken(ctx, namespace)
	handler.WriteHeaderAndProto(ret)
}

// 更新命名空间的token
func (h *Httpserver) UpdateNamespaceToken(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var namespace api.Namespace
	ctx, err := handler.Parse(&namespace)
	if ctx == nil {
		ret := api.NewResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	ret := h.namingServer.UpdateNamespaceToken(ctx, &namespace)
	handler.WriteHeaderAndProto(ret)
}

/**
 * @brief 创建服务
 */
func (h *Httpserver) CreateServices(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var services ServiceArr
	ctx, err := handler.Parse(&services)
	if ctx == nil {
		ret := api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	ret := h.namingServer.CreateServices(ctx, services)
	handler.WriteHeaderAndProto(ret)
}

/**
 * @brief 删除服务
 */
func (h *Httpserver) DeleteServices(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var services ServiceArr
	ctx, err := handler.Parse(&services)
	if ctx == nil {
		ret := api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	ret := h.namingServer.DeleteServices(ctx, services)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}

	handler.WriteHeader(ret.GetCode().GetValue(), http.StatusOK)
}

/**
 * @brief 修改服务
 */
func (h *Httpserver) UpdateServices(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var services ServiceArr
	ctx, err := handler.Parse(&services)
	if ctx == nil {
		ret := api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	ret := h.namingServer.UpdateServices(ctx, services)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}
	handler.WriteHeader(ret.GetCode().GetValue(), http.StatusOK)
}

/**
 * @brief 查询服务
 */
func (h *Httpserver) GetServices(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := parseQueryParams(req)
	ret := h.namingServer.GetServices(queryParams)
	handler.WriteHeaderAndProto(ret)
}

/**
 * @brief 查询服务总数
 */
func (h *Httpserver) GetServicesCount(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}
	ret := h.namingServer.GetServicesCount()
	handler.WriteHeaderAndProto(ret)
}

// 获取服务token
func (h *Httpserver) GetServiceToken(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}
	token := req.HeaderParameter("Polaris-Token")
	ctx := context.WithValue(context.Background(), utils.StringContext("polaris-token"), token)

	queryParams := parseQueryParams(req)
	service := &api.Service{
		Name:      utils.NewStringValue(queryParams["name"]),
		Namespace: utils.NewStringValue(queryParams["namespace"]),
		Token:     utils.NewStringValue(queryParams["token"]),
	}

	ret := h.namingServer.GetServiceToken(ctx, service)
	handler.WriteHeaderAndProto(ret)
}

// 更新服务token
func (h *Httpserver) UpdateServiceToken(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var service api.Service
	ctx, err := handler.Parse(&service)
	if ctx == nil {
		ret := api.NewResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	ret := h.namingServer.UpdateServiceToken(ctx, &service)
	handler.WriteHeaderAndProto(ret)
}

// service alias
func (h *Httpserver) CreateServiceAlias(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var alias api.ServiceAlias
	ctx, err := handler.Parse(&alias)
	if ctx == nil {
		ret := api.NewResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	out := h.namingServer.CreateServiceAlias(ctx, &alias)
	handler.WriteHeaderAndProto(out)
	return
}

/**
 * @brief 创建服务别名
 * @note 不需要鉴权
 */
func (h *Httpserver) CreateServiceAliasNoAuth(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var alias api.ServiceAlias
	ctx, err := handler.Parse(&alias)
	if ctx == nil {
		ret := api.NewResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	out := h.namingServer.CreateServiceAliasNoAuth(ctx, &alias)
	handler.WriteHeaderAndProto(out)
	return
}

// 修改服务别名
func (h *Httpserver) UpdateServiceAlias(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var alias api.ServiceAlias
	ctx, err := handler.Parse(&alias)
	if ctx == nil {
		ret := api.NewResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	ret := h.namingServer.UpdateServiceAlias(ctx, &alias)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}
	handler.WriteHeader(ret.GetCode().GetValue(), http.StatusOK)
	return
}

// 根据源服务获取服务别名
func (h *Httpserver) GetServiceAliases(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := parseQueryParams(req)
	ret := h.namingServer.GetServiceAliases(queryParams)
	handler.WriteHeaderAndProto(ret)
}

/**
 * @brief 创建服务实例
 */
func (h *Httpserver) CreateInstances(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var instances InstanceArr
	ctx, err := handler.Parse(&instances)
	if ctx == nil {
		ret := api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	ret := h.namingServer.CreateInstances(ctx, instances)
	handler.WriteHeaderAndProto(ret)
}

/**
 * @brief 删除服务实例
 */
func (h *Httpserver) DeleteInstances(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var instances InstanceArr
	ctx, err := handler.Parse(&instances)
	if ctx == nil {
		ret := api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	ret := h.namingServer.DeleteInstances(ctx, instances)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}

	handler.WriteHeader(ret.GetCode().GetValue(), http.StatusOK)
}

/**
 * @brief 根据host删除服务实例
 */
func (h *Httpserver) DeleteInstancesByHost(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var instances InstanceArr
	ctx, err := handler.Parse(&instances)
	if ctx == nil {
		ret := api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	ret := h.namingServer.DeleteInstancesByHost(ctx, instances)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}

	handler.WriteHeader(ret.GetCode().GetValue(), http.StatusOK)
}

/**
 * @brief 修改服务实例
 */
func (h *Httpserver) UpdateInstances(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var instances InstanceArr
	ctx, err := handler.Parse(&instances)
	if ctx == nil {
		ret := api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	ret := h.namingServer.UpdateInstances(ctx, instances)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}

	handler.WriteHeader(ret.GetCode().GetValue(), http.StatusOK)
}

/**
 * @brief 修改服务实例的隔离状态
 */
func (h *Httpserver) UpdateInstancesIsolate(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var instances InstanceArr
	ctx, err := handler.Parse(&instances)
	if ctx == nil {
		ret := api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	ret := h.namingServer.UpdateInstancesIsolate(ctx, instances)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}

	handler.WriteHeader(ret.GetCode().GetValue(), http.StatusOK)
}

/**
 * @brief 查询服务实例
 */
func (h *Httpserver) GetInstances(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := parseQueryParams(req)
	ret := h.namingServer.GetInstances(queryParams)
	handler.WriteHeaderAndProto(ret)
}

/**
 * @brief 查询服务实例
 */
func (h *Httpserver) GetInstancesCount(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	ret := h.namingServer.GetInstancesCount()
	handler.WriteHeaderAndProto(ret)
}

/**
 * @brief 创建规则路由
 */
func (h *Httpserver) CreateRoutings(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var routings RoutingArr
	ctx, err := handler.Parse(&routings)
	if ctx == nil {
		ret := api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	ret := h.namingServer.CreateRoutingConfigs(ctx, routings)
	handler.WriteHeaderAndProto(ret)
}

/**
 * @brief 删除规则路由
 */
func (h *Httpserver) DeleteRoutings(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var routings RoutingArr
	ctx, err := handler.Parse(&routings)
	if ctx == nil {
		ret := api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	ret := h.namingServer.DeleteRoutingConfigs(ctx, routings)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}

	handler.WriteHeader(ret.GetCode().GetValue(), http.StatusOK)
}

/**
 * @brief 修改规则路由
 */
func (h *Httpserver) UpdateRoutings(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var routings RoutingArr
	ctx, err := handler.Parse(&routings)
	if ctx == nil {
		ret := api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	ret := h.namingServer.UpdateRoutingConfigs(ctx, routings)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}

	handler.WriteHeader(ret.GetCode().GetValue(), http.StatusOK)
}

/**
 * @brief 查询规则路由
 */
func (h *Httpserver) GetRoutings(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := parseQueryParams(req)
	ret := h.namingServer.GetRoutingConfigs(nil, queryParams)
	handler.WriteHeaderAndProto(ret)
}

/**
 * @brief 创建限流规则
 */
func (h *Httpserver) CreateRateLimits(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var rateLimits RateLimitArr
	ctx, err := handler.Parse(&rateLimits)
	if ctx == nil {
		ret := api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	ret := h.namingServer.CreateRateLimits(ctx, rateLimits)
	handler.WriteHeaderAndProto(ret)
}

/**
 * @brief 删除限流规则
 */
func (h *Httpserver) DeleteRateLimits(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var rateLimits RateLimitArr
	ctx, err := handler.Parse(&rateLimits)
	if ctx == nil {
		ret := api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	ret := h.namingServer.DeleteRateLimits(ctx, rateLimits)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}
	handler.WriteHeader(ret.GetCode().GetValue(), http.StatusOK)
}

/**
 * @brief 修改限流规则
 */
func (h *Httpserver) UpdateRateLimits(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var rateLimits RateLimitArr
	ctx, err := handler.Parse(&rateLimits)
	if ctx == nil {
		ret := api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	ret := h.namingServer.UpdateRateLimits(ctx, rateLimits)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}

	handler.WriteHeader(ret.GetCode().GetValue(), http.StatusOK)
}

/**
 * @brief 查询限流规则
 */
func (h *Httpserver) GetRateLimits(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := parseQueryParams(req)
	ret := h.namingServer.GetRateLimits(queryParams)
	handler.WriteHeaderAndProto(ret)
}

/**
 * @brief 创建熔断规则
 */
func (h *Httpserver) CreateCircuitBreakers(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var circuitBreakers CircuitBreakerArr
	ctx, err := handler.Parse(&circuitBreakers)
	if ctx == nil {
		ret := api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	ret := h.namingServer.CreateCircuitBreakers(ctx, circuitBreakers)
	handler.WriteHeaderAndProto(ret)
}

/**
 * @brief 创建熔断规则版本
 */
func (h *Httpserver) CreateCircuitBreakerVersions(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var circuitBreakers CircuitBreakerArr
	ctx, err := handler.Parse(&circuitBreakers)
	if ctx == nil {
		ret := api.NewBatchQueryResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	ret := h.namingServer.CreateCircuitBreakerVersions(ctx, circuitBreakers)
	handler.WriteHeaderAndProto(ret)
}

/**
 * @brief 删除熔断规则
 */
func (h *Httpserver) DeleteCircuitBreakers(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var circuitBreakers CircuitBreakerArr
	ctx, err := handler.Parse(&circuitBreakers)
	if ctx == nil {
		ret := api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	ret := h.namingServer.DeleteCircuitBreakers(ctx, circuitBreakers)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}
	handler.WriteHeader(ret.GetCode().GetValue(), http.StatusOK)
}

/**
 * @brief 修改熔断规则
 */
func (h *Httpserver) UpdateCircuitBreakers(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var circuitBreakers CircuitBreakerArr
	ctx, err := handler.Parse(&circuitBreakers)
	if ctx == nil {
		ret := api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	ret := h.namingServer.UpdateCircuitBreakers(ctx, circuitBreakers)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}
	handler.WriteHeader(ret.GetCode().GetValue(), http.StatusOK)
}

/**
 * @brief 发布熔断规则
 */
func (h *Httpserver) ReleaseCircuitBreakers(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var configRelease ConfigReleaseArr
	ctx, err := handler.Parse(&configRelease)
	if ctx == nil {
		ret := api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	ret := h.namingServer.ReleaseCircuitBreakers(ctx, configRelease)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}
	handler.WriteHeader(ret.GetCode().GetValue(), http.StatusOK)
}

/**
 * @brief 解绑熔断规则
 */
func (h *Httpserver) UnBindCircuitBreakers(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var configRelease ConfigReleaseArr
	ctx, err := handler.Parse(&configRelease)
	if ctx == nil {
		ret := api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	ret := h.namingServer.UnBindCircuitBreakers(ctx, configRelease)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}
	handler.WriteHeader(ret.GetCode().GetValue(), http.StatusOK)
}

/**
 * @brief 根据id和version获取熔断规则
 */
func (h *Httpserver) GetCircuitBreaker(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := parseQueryParams(req)
	ret := h.namingServer.GetCircuitBreaker(queryParams)
	handler.WriteHeaderAndProto(ret)
}

/**
 * @brief 查询熔断规则的所有版本
 */
func (h *Httpserver) GetCircuitBreakerVersions(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := parseQueryParams(req)
	ret := h.namingServer.GetCircuitBreakerVersions(queryParams)
	handler.WriteHeaderAndProto(ret)
}

/**
 * @brief 查询master熔断规则
 */
func (h *Httpserver) GetMasterCircuitBreakers(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := parseQueryParams(req)
	ret := h.namingServer.GetMasterCircuitBreakers(queryParams)
	handler.WriteHeaderAndProto(ret)
}

/**
 * @brief 根据规则id查询已发布的熔断规则
 */
func (h *Httpserver) GetReleaseCircuitBreakers(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := parseQueryParams(req)
	ret := h.namingServer.GetReleaseCircuitBreakers(queryParams)
	handler.WriteHeaderAndProto(ret)
}

/**
 * @brief 根据服务查询绑定熔断规则
 */
func (h *Httpserver) GetCircuitBreakerByService(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := parseQueryParams(req)
	ret := h.namingServer.GetCircuitBreakerByService(queryParams)
	handler.WriteHeaderAndProto(ret)
}

/**
 * @brief 根据服务获取服务负责人
 */
func (h *Httpserver) GetServiceOwner(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var services ServiceArr
	ctx, err := handler.Parse(&services)
	if ctx == nil {
		ret := api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	ret := h.namingServer.GetServiceOwner(ctx, services)
	handler.WriteHeaderAndProto(ret)
}

/**
 * @brief 获取熔断规则token
 */
func (h *Httpserver) GetCircuitBreakerToken(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}
	token := req.HeaderParameter("Polaris-Token")
	ctx := context.WithValue(context.Background(), utils.StringContext("polaris-token"), token)

	queryParams := parseQueryParams(req)
	circuitBreaker := &api.CircuitBreaker{
		Id:      utils.NewStringValue(queryParams["id"]),
		Version: utils.NewStringValue("master"),
		Token:   utils.NewStringValue(queryParams["token"]),
	}
	ret := h.namingServer.GetCircuitBreakerToken(ctx, circuitBreaker)
	handler.WriteHeaderAndProto(ret)
}

/*
 * @brief 创建平台
 */
func (h *Httpserver) CreatePlatforms(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var platforms PlatformArr
	ctx, err := handler.Parse(&platforms)

	if ctx == nil {
		ret := api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	ret := h.namingServer.CreatePlatforms(ctx, platforms)
	handler.WriteHeaderAndProto(ret)
}

/**
 * @brief 修改平台
 */
func (h *Httpserver) UpdatePlatforms(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var platforms PlatformArr
	ctx, err := handler.Parse(&platforms)

	if ctx == nil {
		ret := api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	ret := h.namingServer.UpdatePlatforms(ctx, platforms)

	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}

	handler.WriteHeader(ret.GetCode().GetValue(), http.StatusOK)
}

/**
 * @brief 删除平台
 */
func (h *Httpserver) DeletePlatforms(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var platforms PlatformArr
	ctx, err := handler.Parse(&platforms)
	if ctx == nil {
		ret := api.NewBatchQueryResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}
	ret := h.namingServer.DeletePlatforms(ctx, platforms)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}

	handler.WriteHeader(ret.GetCode().GetValue(), http.StatusOK)
}

/**
 * @brief 查询平台
 */
func (h *Httpserver) GetPlatforms(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := parseQueryParams(req)
	ret := h.namingServer.GetPlatforms(queryParams)
	handler.WriteHeaderAndProto(ret)
}

/**
 * @brief 查询平台Token
 */
func (h *Httpserver) GetPlatformToken(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}
	token := req.HeaderParameter("Polaris-Token")
	ctx := context.WithValue(context.Background(), utils.StringContext("polaris-token"), token)

	queryParams := parseQueryParams(req)
	platform := &api.Platform{
		Id:    utils.NewStringValue(queryParams["id"]),
		Token: utils.NewStringValue(queryParams["token"]),
	}

	ret := h.namingServer.GetPlatformToken(ctx, platform)
	handler.WriteHeaderAndProto(ret)
}

/**
 * @brief 解析并获取HTTP的query params
 */
func parseQueryParams(req *restful.Request) map[string]string {
	queryParams := make(map[string]string)
	for key, value := range req.Request.URL.Query() {
		if len(value) > 0 {
			queryParams[key] = value[0] // 暂时默认只支持一个查询
		}
	}

	return queryParams
}
