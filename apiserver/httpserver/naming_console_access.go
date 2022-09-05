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
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/golang/protobuf/proto"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/utils"
)

const (
	defaultReadAccess string = "default-read"
	defaultAccess     string = "default"
)

// GetNamingConsoleAccessServer 注册管理端接口
func (h *HTTPServer) GetNamingConsoleAccessServer(include []string) (*restful.WebService, error) {
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

// addDefaultReadAccess 增加默认读接口
func (h *HTTPServer) addDefaultReadAccess(ws *restful.WebService) {
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

// addDefaultAccess 增加默认接口
func (h *HTTPServer) addDefaultAccess(ws *restful.WebService) {
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

	ws.Route(ws.POST("/platforms").To(h.CreatePlatforms))
	ws.Route(ws.POST("/platforms/delete").To(h.DeletePlatforms))
	ws.Route(ws.PUT("/platforms").To(h.UpdatePlatforms))
	ws.Route(ws.GET("/platforms").To(h.GetPlatforms))
	ws.Route(ws.GET("/platform/token").To(h.GetPlatformToken))

}

// CreateNamespaces 创建命名空间
func (h *HTTPServer) CreateNamespaces(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

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
func (h *HTTPServer) DeleteNamespaces(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

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
func (h *HTTPServer) UpdateNamespaces(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

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
func (h *HTTPServer) GetNamespaces(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	ret := h.namespaceServer.GetNamespaces(handler.ParseHeaderContext(), req.Request.URL.Query())
	handler.WriteHeaderAndProto(ret)
}

// GetNamespaceToken 命名空间token的获取
func (h *HTTPServer) GetNamespaceToken(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	token := req.HeaderParameter("Polaris-Token")
	ctx := context.WithValue(context.Background(), utils.StringContext("polaris-token"), token)

	queryParams := utils.ParseQueryParams(req)
	namespace := &api.Namespace{
		Name:  utils.NewStringValue(queryParams["name"]),
		Token: utils.NewStringValue(queryParams["token"]),
	}

	ret := h.namespaceServer.GetNamespaceToken(ctx, namespace)
	handler.WriteHeaderAndProto(ret)
}

// UpdateNamespaceToken 更新命名空间的token
func (h *HTTPServer) UpdateNamespaceToken(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

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
func (h *HTTPServer) CreateServices(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

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
func (h *HTTPServer) DeleteServices(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

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
func (h *HTTPServer) UpdateServices(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

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
func (h *HTTPServer) GetServices(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := utils.ParseQueryParams(req)
	ctx := handler.ParseHeaderContext()
	ret := h.namingServer.GetServices(ctx, queryParams)
	handler.WriteHeaderAndProto(ret)
}

// GetServicesCount 查询服务总数
func (h *HTTPServer) GetServicesCount(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}
	ret := h.namingServer.GetServicesCount(handler.ParseHeaderContext())
	handler.WriteHeaderAndProto(ret)
}

// GetServiceToken 获取服务token
func (h *HTTPServer) GetServiceToken(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}
	token := req.HeaderParameter("Polaris-Token")
	ctx := context.WithValue(context.Background(), utils.StringContext("polaris-token"), token)

	queryParams := utils.ParseQueryParams(req)
	service := &api.Service{
		Name:      utils.NewStringValue(queryParams["name"]),
		Namespace: utils.NewStringValue(queryParams["namespace"]),
		Token:     utils.NewStringValue(queryParams["token"]),
	}

	ret := h.namingServer.GetServiceToken(ctx, service)
	handler.WriteHeaderAndProto(ret)
}

// UpdateServiceToken 更新服务token
func (h *HTTPServer) UpdateServiceToken(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var service api.Service
	ctx, err := handler.Parse(&service)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.namingServer.UpdateServiceToken(ctx, &service))
}

// CreateServiceAlias service alias
func (h *HTTPServer) CreateServiceAlias(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var alias api.ServiceAlias
	ctx, err := handler.Parse(&alias)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.namingServer.CreateServiceAlias(ctx, &alias))
}

// UpdateServiceAlias 修改服务别名
func (h *HTTPServer) UpdateServiceAlias(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

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
func (h *HTTPServer) DeleteServiceAliases(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

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
func (h *HTTPServer) GetServiceAliases(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := utils.ParseQueryParams(req)
	ret := h.namingServer.GetServiceAliases(handler.ParseHeaderContext(), queryParams)
	handler.WriteHeaderAndProto(ret)
}

// CreateInstances 创建服务实例
func (h *HTTPServer) CreateInstances(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

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
func (h *HTTPServer) DeleteInstances(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

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
func (h *HTTPServer) DeleteInstancesByHost(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

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
func (h *HTTPServer) UpdateInstances(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

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
func (h *HTTPServer) UpdateInstancesIsolate(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

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
func (h *HTTPServer) GetInstances(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := utils.ParseQueryParams(req)
	ret := h.namingServer.GetInstances(handler.ParseHeaderContext(), queryParams)
	handler.WriteHeaderAndProto(ret)
}

// GetInstancesCount 查询服务实例
func (h *HTTPServer) GetInstancesCount(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	ret := h.namingServer.GetInstancesCount(handler.ParseHeaderContext())
	handler.WriteHeaderAndProto(ret)
}

// CreateRoutings 创建规则路由
func (h *HTTPServer) CreateRoutings(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

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
func (h *HTTPServer) DeleteRoutings(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

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
func (h *HTTPServer) UpdateRoutings(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

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
func (h *HTTPServer) GetRoutings(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := utils.ParseQueryParams(req)
	ret := h.namingServer.GetRoutingConfigs(handler.ParseHeaderContext(), queryParams)
	handler.WriteHeaderAndProto(ret)
}

// CreateRateLimits 创建限流规则
func (h *HTTPServer) CreateRateLimits(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

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
func (h *HTTPServer) DeleteRateLimits(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

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
func (h *HTTPServer) EnableRateLimits(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}
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
func (h *HTTPServer) UpdateRateLimits(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

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
func (h *HTTPServer) GetRateLimits(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := utils.ParseQueryParams(req)
	ret := h.namingServer.GetRateLimits(handler.ParseHeaderContext(), queryParams)
	handler.WriteHeaderAndProto(ret)
}

// CreateCircuitBreakers 创建熔断规则
func (h *HTTPServer) CreateCircuitBreakers(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

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
func (h *HTTPServer) CreateCircuitBreakerVersions(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

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
func (h *HTTPServer) DeleteCircuitBreakers(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

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
func (h *HTTPServer) UpdateCircuitBreakers(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

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
func (h *HTTPServer) ReleaseCircuitBreakers(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

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
func (h *HTTPServer) UnBindCircuitBreakers(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

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
func (h *HTTPServer) GetCircuitBreaker(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := utils.ParseQueryParams(req)
	ret := h.namingServer.GetCircuitBreaker(handler.ParseHeaderContext(), queryParams)
	handler.WriteHeaderAndProto(ret)
}

// GetCircuitBreakerVersions 查询熔断规则的所有版本
func (h *HTTPServer) GetCircuitBreakerVersions(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := utils.ParseQueryParams(req)
	ret := h.namingServer.GetCircuitBreakerVersions(handler.ParseHeaderContext(), queryParams)
	handler.WriteHeaderAndProto(ret)
}

// GetMasterCircuitBreakers 查询master熔断规则
func (h *HTTPServer) GetMasterCircuitBreakers(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := utils.ParseQueryParams(req)
	ret := h.namingServer.GetMasterCircuitBreakers(handler.ParseHeaderContext(), queryParams)
	handler.WriteHeaderAndProto(ret)
}

// GetReleaseCircuitBreakers 根据规则id查询已发布的熔断规则
func (h *HTTPServer) GetReleaseCircuitBreakers(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := utils.ParseQueryParams(req)
	ret := h.namingServer.GetReleaseCircuitBreakers(handler.ParseHeaderContext(), queryParams)
	handler.WriteHeaderAndProto(ret)
}

// GetCircuitBreakerByService 根据服务查询绑定熔断规则
func (h *HTTPServer) GetCircuitBreakerByService(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := utils.ParseQueryParams(req)
	ret := h.namingServer.GetCircuitBreakerByService(handler.ParseHeaderContext(), queryParams)
	handler.WriteHeaderAndProto(ret)
}

// GetServiceOwner 根据服务获取服务负责人
func (h *HTTPServer) GetServiceOwner(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

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
func (h *HTTPServer) GetCircuitBreakerToken(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}
	token := req.HeaderParameter("Polaris-Token")
	ctx := context.WithValue(context.Background(), utils.StringContext("polaris-token"), token)

	queryParams := utils.ParseQueryParams(req)
	circuitBreaker := &api.CircuitBreaker{
		Id:      utils.NewStringValue(queryParams["id"]),
		Version: utils.NewStringValue("master"),
		Token:   utils.NewStringValue(queryParams["token"]),
	}
	ret := h.namingServer.GetCircuitBreakerToken(ctx, circuitBreaker)
	handler.WriteHeaderAndProto(ret)
}

// CreatePlatforms 创建平台
func (h *HTTPServer) CreatePlatforms(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var platforms PlatformArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.Platform{}
		platforms = append(platforms, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.namingServer.CreatePlatforms(ctx, platforms))
}

// UpdatePlatforms 修改平台
func (h *HTTPServer) UpdatePlatforms(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var platforms PlatformArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.Platform{}
		platforms = append(platforms, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	ret := h.namingServer.UpdatePlatforms(ctx, platforms)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}

	handler.WriteHeaderAndProto(ret)
}

// DeletePlatforms 删除平台
func (h *HTTPServer) DeletePlatforms(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var platforms PlatformArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.Platform{}
		platforms = append(platforms, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchQueryResponseWithMsg(api.ParseException, err.Error()))
		return
	}
	ret := h.namingServer.DeletePlatforms(ctx, platforms)
	if code := api.CalcCode(ret); code != http.StatusOK {
		handler.WriteHeaderAndProto(ret)
		return
	}

	handler.WriteHeaderAndProto(ret)
}

// GetPlatforms 查询平台
func (h *HTTPServer) GetPlatforms(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := utils.ParseQueryParams(req)
	ret := h.namingServer.GetPlatforms(handler.ParseHeaderContext(), queryParams)
	handler.WriteHeaderAndProto(ret)
}

// GetPlatformToken 查询平台Token
func (h *HTTPServer) GetPlatformToken(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}
	token := req.HeaderParameter("Polaris-Token")
	ctx := context.WithValue(context.Background(), utils.StringContext("polaris-token"), token)

	queryParams := utils.ParseQueryParams(req)
	platform := &api.Platform{
		Id:    utils.NewStringValue(queryParams["id"]),
		Token: utils.NewStringValue(queryParams["token"]),
	}

	ret := h.namingServer.GetPlatformToken(ctx, platform)
	handler.WriteHeaderAndProto(ret)
}
