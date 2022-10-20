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
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/golang/protobuf/proto"
	restfulspec "github.com/polarismesh/go-restful-openapi/v2"

	httpcommon "github.com/polarismesh/polaris/apiserver/httpserver/http"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/utils"
)

const (
	defaultReadAccess string = "default-read"
	defaultAccess     string = "default"
)

// GetNamingConsoleAccessServer 注册管理端接口
func (h *HTTPServerV1) GetNamingConsoleAccessServer(include []string) (*restful.WebService, error) {
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
			log.Errorf("method %s does not exist in HTTPServerV1 console access", item)
			return nil, fmt.Errorf("method %s does not exist in HTTPServerV1 console access", item)
		}
	}
	return ws, nil
}

// addDefaultReadAccess 增加默认读接口
func (h *HTTPServerV1) addDefaultReadAccess(ws *restful.WebService) {
	// 管理端接口：只包含读接口
	nsTags := []string{"Namespaces"}
	ws.Route(ws.GET("/namespaces").To(h.GetNamespaces).
		Doc("get namespaces").
		Metadata(restfulspec.KeyOpenAPITags, nsTags))

	ws.Route(ws.GET("/namespace/token").To(h.GetNamespaceToken).
		Doc("get namespaces token").
		Metadata(restfulspec.KeyOpenAPITags, nsTags))

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
}

// addDefaultAccess 增加默认接口
func (h *HTTPServerV1) addDefaultAccess(ws *restful.WebService) {
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
	ws.Route(ws.PUT("/service/alias").To(h.UpdateServiceAlias))
	ws.Route(ws.GET("/service/aliases").To(h.GetServiceAliases))
	ws.Route(ws.POST("/service/aliases/delete").To(h.DeleteServiceAliases))
	ws.Route(ws.GET("/service/circuitbreaker").To(h.GetCircuitBreakerByService))
	ws.Route(ws.POST("/service/owner").To(h.GetServiceOwner))

	ws.Route(ws.POST("/instances").To(h.CreateInstances))
	ws.Route(ws.POST("/instances/delete").To(h.DeleteInstances))
	ws.Route(ws.POST("/instances/delete/host").To(h.DeleteInstancesByHost))
	ws.Route(ws.PUT("/instances").To(h.UpdateInstances))
	ws.Route(ws.PUT("/instances/isolate/host").To(h.UpdateInstancesIsolate))
	ws.Route(ws.GET("/instances").To(h.GetInstances))
	ws.Route(ws.GET("/instances/count").To(h.GetInstancesCount))
	ws.Route(ws.GET("/instances/labels").To(h.GetInstanceLabels))

	ws.Route(ws.POST("/routings").To(h.CreateRoutings))
	ws.Route(ws.POST("/routings/delete").To(h.DeleteRoutings))
	ws.Route(ws.PUT("/routings").To(h.UpdateRoutings))
	ws.Route(ws.GET("/routings").To(h.GetRoutings))

	ws.Route(ws.POST("/ratelimits").To(h.CreateRateLimits))
	ws.Route(ws.POST("/ratelimits/delete").To(h.DeleteRateLimits))
	ws.Route(ws.PUT("/ratelimits").To(h.UpdateRateLimits))
	ws.Route(ws.GET("/ratelimits").To(h.GetRateLimits))
	ws.Route(ws.PUT("/ratelimits/enable").To(h.EnableRateLimits))

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
}

// CreateNamespaces 创建命名空间
func (h *HTTPServerV1) CreateNamespaces(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var namespaces NamespaceArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.Namespace{}
		namespaces = append(namespaces, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.namespaceServer.CreateNamespaces(ctx, namespaces))
}

// DeleteNamespaces 删除命名空间
func (h *HTTPServerV1) DeleteNamespaces(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var namespaces NamespaceArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.Namespace{}
		namespaces = append(namespaces, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	ret := h.namespaceServer.DeleteNamespaces(ctx, namespaces)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}

	handler.WriteHeaderAndProto(ret)
}

// UpdateNamespaces 修改命名空间
func (h *HTTPServerV1) UpdateNamespaces(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var namespaces NamespaceArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.Namespace{}
		namespaces = append(namespaces, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	ret := h.namespaceServer.UpdateNamespaces(ctx, namespaces)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}

	handler.WriteHeaderAndProto(ret)
}

// GetNamespaces 查询命名空间
func (h *HTTPServerV1) GetNamespaces(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	ret := h.namespaceServer.GetNamespaces(handler.ParseHeaderContext(), req.Request.URL.Query())
	handler.WriteHeaderAndProto(ret)
}

// GetNamespaceToken 命名空间token的获取
func (h *HTTPServerV1) GetNamespaceToken(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	token := req.HeaderParameter("Polaris-Token")
	ctx := context.WithValue(context.Background(), utils.StringContext("polaris-token"), token)

	queryParams := httpcommon.ParseQueryParams(req)
	namespace := &api.Namespace{
		Name:  utils.NewStringValue(queryParams["name"]),
		Token: utils.NewStringValue(queryParams["token"]),
	}

	ret := h.namespaceServer.GetNamespaceToken(ctx, namespace)
	handler.WriteHeaderAndProto(ret)
}

// UpdateNamespaceToken 更新命名空间的token
func (h *HTTPServerV1) UpdateNamespaceToken(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var namespace api.Namespace
	ctx, err := handler.Parse(&namespace)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	ret := h.namespaceServer.UpdateNamespaceToken(ctx, &namespace)
	handler.WriteHeaderAndProto(ret)
}

// CreateServices 创建服务
func (h *HTTPServerV1) CreateServices(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var services ServiceArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.Service{}
		services = append(services, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.namingServer.CreateServices(ctx, services))
}

// DeleteServices 删除服务
func (h *HTTPServerV1) DeleteServices(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var services ServiceArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.Service{}
		services = append(services, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	ret := h.namingServer.DeleteServices(ctx, services)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}

	handler.WriteHeaderAndProto(ret)
}

// UpdateServices 修改服务
func (h *HTTPServerV1) UpdateServices(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var services ServiceArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.Service{}
		services = append(services, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	ret := h.namingServer.UpdateServices(ctx, services)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}
	handler.WriteHeaderAndProto(ret)
}

// GetServices 查询服务
func (h *HTTPServerV1) GetServices(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	queryParams := httpcommon.ParseQueryParams(req)
	ctx := handler.ParseHeaderContext()
	ret := h.namingServer.GetServices(ctx, queryParams)
	handler.WriteHeaderAndProto(ret)
}

// GetServicesCount 查询服务总数
func (h *HTTPServerV1) GetServicesCount(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}
	ret := h.namingServer.GetServicesCount(handler.ParseHeaderContext())
	handler.WriteHeaderAndProto(ret)
}

// GetServiceToken 获取服务token
func (h *HTTPServerV1) GetServiceToken(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}
	token := req.HeaderParameter("Polaris-Token")
	ctx := context.WithValue(context.Background(), utils.StringContext("polaris-token"), token)

	queryParams := httpcommon.ParseQueryParams(req)
	service := &api.Service{
		Name:      utils.NewStringValue(queryParams["name"]),
		Namespace: utils.NewStringValue(queryParams["namespace"]),
		Token:     utils.NewStringValue(queryParams["token"]),
	}

	ret := h.namingServer.GetServiceToken(ctx, service)
	handler.WriteHeaderAndProto(ret)
}

// UpdateServiceToken 更新服务token
func (h *HTTPServerV1) UpdateServiceToken(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var service api.Service
	ctx, err := handler.Parse(&service)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.namingServer.UpdateServiceToken(ctx, &service))
}

// CreateServiceAlias service alias
func (h *HTTPServerV1) CreateServiceAlias(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var alias api.ServiceAlias
	ctx, err := handler.Parse(&alias)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.namingServer.CreateServiceAlias(ctx, &alias))
}

// UpdateServiceAlias 修改服务别名
func (h *HTTPServerV1) UpdateServiceAlias(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var alias api.ServiceAlias
	ctx, err := handler.Parse(&alias)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	ret := h.namingServer.UpdateServiceAlias(ctx, &alias)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}

	handler.WriteHeaderAndProto(ret)
}

// DeleteServiceAliases 删除服务别名
func (h *HTTPServerV1) DeleteServiceAliases(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var aliases ServiceAliasArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.ServiceAlias{}
		aliases = append(aliases, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}
	ret := h.namingServer.DeleteServiceAliases(ctx, aliases)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}

	handler.WriteHeaderAndProto(ret)
}

// GetServiceAliases 根据源服务获取服务别名
func (h *HTTPServerV1) GetServiceAliases(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	queryParams := httpcommon.ParseQueryParams(req)
	ret := h.namingServer.GetServiceAliases(handler.ParseHeaderContext(), queryParams)
	handler.WriteHeaderAndProto(ret)
}

// CreateInstances 创建服务实例
func (h *HTTPServerV1) CreateInstances(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var instances InstanceArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.Instance{}
		instances = append(instances, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.namingServer.CreateInstances(ctx, instances))
}

// DeleteInstances 删除服务实例
func (h *HTTPServerV1) DeleteInstances(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var instances InstanceArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.Instance{}
		instances = append(instances, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	ret := h.namingServer.DeleteInstances(ctx, instances)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}

	handler.WriteHeaderAndProto(ret)
}

// DeleteInstancesByHost 根据host删除服务实例
func (h *HTTPServerV1) DeleteInstancesByHost(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var instances InstanceArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.Instance{}
		instances = append(instances, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	ret := h.namingServer.DeleteInstancesByHost(ctx, instances)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}

	handler.WriteHeaderAndProto(ret)
}

// UpdateInstances 修改服务实例
func (h *HTTPServerV1) UpdateInstances(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var instances InstanceArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.Instance{}
		instances = append(instances, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	ret := h.namingServer.UpdateInstances(ctx, instances)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}

	handler.WriteHeaderAndProto(ret)
}

// UpdateInstancesIsolate 修改服务实例的隔离状态
func (h *HTTPServerV1) UpdateInstancesIsolate(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var instances InstanceArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.Instance{}
		instances = append(instances, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	ret := h.namingServer.UpdateInstancesIsolate(ctx, instances)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}

	handler.WriteHeaderAndProto(ret)
}

// GetInstances 查询服务实例
func (h *HTTPServerV1) GetInstances(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	queryParams := httpcommon.ParseQueryParams(req)
	ret := h.namingServer.GetInstances(handler.ParseHeaderContext(), queryParams)
	handler.WriteHeaderAndProto(ret)
}

// GetInstancesCount 查询服务实例
func (h *HTTPServerV1) GetInstancesCount(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	ret := h.namingServer.GetInstancesCount(handler.ParseHeaderContext())
	handler.WriteHeaderAndProto(ret)
}

// GetInstanceLabels 查询某个服务下所有实例的标签信息
func (h *HTTPServerV1) GetInstanceLabels(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	ret := h.namingServer.GetInstanceLabels(handler.ParseHeaderContext(), httpcommon.ParseQueryParams(req))
	handler.WriteHeaderAndProto(ret)
}

// CreateRoutings 创建规则路由
func (h *HTTPServerV1) CreateRoutings(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var routings RoutingArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.Routing{}
		routings = append(routings, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	ret := h.namingServer.CreateRoutingConfigs(ctx, routings)
	handler.WriteHeaderAndProto(ret)
}

// DeleteRoutings 删除规则路由
func (h *HTTPServerV1) DeleteRoutings(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var routings RoutingArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.Routing{}
		routings = append(routings, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	ret := h.namingServer.DeleteRoutingConfigs(ctx, routings)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}

	handler.WriteHeaderAndProto(ret)
}

// UpdateRoutings 修改规则路由
func (h *HTTPServerV1) UpdateRoutings(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var routings RoutingArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.Routing{}
		routings = append(routings, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	ret := h.namingServer.UpdateRoutingConfigs(ctx, routings)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}

	handler.WriteHeaderAndProto(ret)
}

// GetRoutings 查询规则路由
func (h *HTTPServerV1) GetRoutings(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	queryParams := httpcommon.ParseQueryParams(req)
	ret := h.namingServer.GetRoutingConfigs(handler.ParseHeaderContext(), queryParams)
	handler.WriteHeaderAndProto(ret)
}

// CreateRateLimits 创建限流规则
func (h *HTTPServerV1) CreateRateLimits(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var rateLimits RateLimitArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.Rule{}
		rateLimits = append(rateLimits, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.namingServer.CreateRateLimits(ctx, rateLimits))
}

// DeleteRateLimits 删除限流规则
func (h *HTTPServerV1) DeleteRateLimits(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var rateLimits RateLimitArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.Rule{}
		rateLimits = append(rateLimits, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	ret := h.namingServer.DeleteRateLimits(ctx, rateLimits)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}
	handler.WriteHeaderAndProto(ret)
}

// EnableRateLimits 激活限流规则
func (h *HTTPServerV1) EnableRateLimits(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}
	var rateLimits RateLimitArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.Rule{}
		rateLimits = append(rateLimits, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}
	ret := h.namingServer.EnableRateLimits(ctx, rateLimits)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}

	handler.WriteHeaderAndProto(ret)
}

// UpdateRateLimits 修改限流规则
func (h *HTTPServerV1) UpdateRateLimits(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var rateLimits RateLimitArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.Rule{}
		rateLimits = append(rateLimits, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	ret := h.namingServer.UpdateRateLimits(ctx, rateLimits)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}

	handler.WriteHeaderAndProto(ret)
}

// GetRateLimits 查询限流规则
func (h *HTTPServerV1) GetRateLimits(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	queryParams := httpcommon.ParseQueryParams(req)
	ret := h.namingServer.GetRateLimits(handler.ParseHeaderContext(), queryParams)
	handler.WriteHeaderAndProto(ret)
}

// CreateCircuitBreakers 创建熔断规则
func (h *HTTPServerV1) CreateCircuitBreakers(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var circuitBreakers CircuitBreakerArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.CircuitBreaker{}
		circuitBreakers = append(circuitBreakers, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	ret := h.namingServer.CreateCircuitBreakers(ctx, circuitBreakers)
	handler.WriteHeaderAndProto(ret)
}

// CreateCircuitBreakerVersions 创建熔断规则版本
func (h *HTTPServerV1) CreateCircuitBreakerVersions(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var circuitBreakers CircuitBreakerArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.CircuitBreaker{}
		circuitBreakers = append(circuitBreakers, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchQueryResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.namingServer.CreateCircuitBreakerVersions(ctx, circuitBreakers))
}

// DeleteCircuitBreakers 删除熔断规则
func (h *HTTPServerV1) DeleteCircuitBreakers(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var circuitBreakers CircuitBreakerArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.CircuitBreaker{}
		circuitBreakers = append(circuitBreakers, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	ret := h.namingServer.DeleteCircuitBreakers(ctx, circuitBreakers)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}
	handler.WriteHeaderAndProto(ret)
}

// UpdateCircuitBreakers 修改熔断规则
func (h *HTTPServerV1) UpdateCircuitBreakers(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var circuitBreakers CircuitBreakerArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.CircuitBreaker{}
		circuitBreakers = append(circuitBreakers, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	ret := h.namingServer.UpdateCircuitBreakers(ctx, circuitBreakers)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}
	handler.WriteHeaderAndProto(ret)
}

// ReleaseCircuitBreakers 发布熔断规则
func (h *HTTPServerV1) ReleaseCircuitBreakers(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var configRelease ConfigReleaseArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.ConfigRelease{}
		configRelease = append(configRelease, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	ret := h.namingServer.ReleaseCircuitBreakers(ctx, configRelease)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}
	handler.WriteHeaderAndProto(ret)
}

// UnBindCircuitBreakers 解绑熔断规则
func (h *HTTPServerV1) UnBindCircuitBreakers(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var configRelease ConfigReleaseArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.ConfigRelease{}
		configRelease = append(configRelease, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	ret := h.namingServer.UnBindCircuitBreakers(ctx, configRelease)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}
	handler.WriteHeaderAndProto(ret)
}

// GetCircuitBreaker 根据id和version获取熔断规则
func (h *HTTPServerV1) GetCircuitBreaker(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	queryParams := httpcommon.ParseQueryParams(req)
	ret := h.namingServer.GetCircuitBreaker(handler.ParseHeaderContext(), queryParams)
	handler.WriteHeaderAndProto(ret)
}

// GetCircuitBreakerVersions 查询熔断规则的所有版本
func (h *HTTPServerV1) GetCircuitBreakerVersions(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	queryParams := httpcommon.ParseQueryParams(req)
	ret := h.namingServer.GetCircuitBreakerVersions(handler.ParseHeaderContext(), queryParams)
	handler.WriteHeaderAndProto(ret)
}

// GetMasterCircuitBreakers 查询master熔断规则
func (h *HTTPServerV1) GetMasterCircuitBreakers(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	queryParams := httpcommon.ParseQueryParams(req)
	ret := h.namingServer.GetMasterCircuitBreakers(handler.ParseHeaderContext(), queryParams)
	handler.WriteHeaderAndProto(ret)
}

// GetReleaseCircuitBreakers 根据规则id查询已发布的熔断规则
func (h *HTTPServerV1) GetReleaseCircuitBreakers(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	queryParams := httpcommon.ParseQueryParams(req)
	ret := h.namingServer.GetReleaseCircuitBreakers(handler.ParseHeaderContext(), queryParams)
	handler.WriteHeaderAndProto(ret)
}

// GetCircuitBreakerByService 根据服务查询绑定熔断规则
func (h *HTTPServerV1) GetCircuitBreakerByService(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	queryParams := httpcommon.ParseQueryParams(req)
	ret := h.namingServer.GetCircuitBreakerByService(handler.ParseHeaderContext(), queryParams)
	handler.WriteHeaderAndProto(ret)
}

// GetServiceOwner 根据服务获取服务负责人
func (h *HTTPServerV1) GetServiceOwner(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var services ServiceArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.Service{}
		services = append(services, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.namingServer.GetServiceOwner(ctx, services))
}

// GetCircuitBreakerToken 获取熔断规则token
func (h *HTTPServerV1) GetCircuitBreakerToken(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}
	token := req.HeaderParameter("Polaris-Token")
	ctx := context.WithValue(context.Background(), utils.StringContext("polaris-token"), token)

	queryParams := httpcommon.ParseQueryParams(req)
	circuitBreaker := &api.CircuitBreaker{
		Id:      utils.NewStringValue(queryParams["id"]),
		Version: utils.NewStringValue("master"),
		Token:   utils.NewStringValue(queryParams["token"]),
	}
	ret := h.namingServer.GetCircuitBreakerToken(ctx, circuitBreaker)
	handler.WriteHeaderAndProto(ret)
}
