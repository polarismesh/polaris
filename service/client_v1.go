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

package service

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

// RegisterInstance create one instance
func (s *Server) RegisterInstance(ctx context.Context, req *api.Instance) *api.Response {
	return s.CreateInstance(ctx, req)
}

// DeregisterInstance delete one instance
func (s *Server) DeregisterInstance(ctx context.Context, req *api.Instance) *api.Response {
	return s.DeleteInstance(ctx, req)
}

// ReportClient 客户端上报信息
func (s *Server) ReportClient(ctx context.Context, req *api.Client) *api.Response {
	if s.caches == nil {
		return api.NewResponse(api.ClientAPINotOpen)
	}

	// 客户端信息不写入到DB中
	host := req.GetHost().GetValue()

	// 从CMDB查询地理位置信息
	if s.cmdb != nil {
		location, err := s.cmdb.GetLocation(host)
		if err != nil {
			log.Error(err.Error(), utils.ZapRequestIDByCtx(ctx))
			return api.NewClientResponse(api.CMDBPluginException, req)
		}

		if location == nil {
			return api.NewClientResponse(api.CMDBNotFindHost, req)
		}
		req.Location = location.Proto
	}

	// save the client with unique id into store
	if len(req.GetId().GetValue()) > 0 {
		return s.checkAndStoreClient(ctx, req)
	}
	out := &api.Client{
		Host:     req.GetHost(),
		Location: req.Location,
	}
	return api.NewClientResponse(api.ExecuteSuccess, out)
}

// GetReportClientWithCache Used for client acquisition service information
func (s *Server) GetReportClientWithCache(ctx context.Context,
	query map[string]string) *model.PrometheusDiscoveryResponse {

	if s.caches == nil {
		return &model.PrometheusDiscoveryResponse{
			Code:     api.NotFoundInstance,
			Response: make([]model.PrometheusTarget, 0),
		}
	}

	targets := make([]model.PrometheusTarget, 0, 8)

	expectSchema := map[string]struct{}{
		"http":  {},
		"https": {},
	}

	s.Cache().Client().IteratorClients(func(key string, value *model.Client) bool {
		for i := range value.Proto().Stat {
			stat := value.Proto().Stat[i]
			if stat.Target.GetValue() != model.StatReportPrometheus {
				continue
			}
			_, ok := expectSchema[strings.ToLower(stat.Protocol.GetValue())]
			if !ok {
				continue
			}

			target := model.PrometheusTarget{
				Targets: []string{fmt.Sprintf("%s:%d", value.Proto().Host.GetValue(), stat.Port.GetValue())},
				Labels: map[string]string{
					"__metrics_path__":         stat.Path.GetValue(),
					"__scheme__":               stat.Protocol.GetValue(),
					"__meta_polaris_client_id": value.Proto().Id.GetValue(),
				},
			}
			targets = append(targets, target)
		}

		return true
	})

	return &model.PrometheusDiscoveryResponse{
		Code:     api.ExecuteSuccess,
		Response: targets,
	}
}

// GetServiceWithCache 根据元数据查询服务
func (s *Server) GetServiceWithCache(ctx context.Context, req *api.Service) *api.DiscoverResponse {
	if s.caches == nil {
		return api.NewDiscoverServiceResponse(api.ClientAPINotOpen, req)
	}

	if req == nil {
		return api.NewDiscoverServiceResponse(api.EmptyRequest, req)
	}
	// 可根据business查询服务
	// if len(req.GetMetadata()) == 0 && len(req.Business.GetValue()) == 0 {
	// 	return api.NewDiscoverServiceResponse(api.InvalidServiceMetadata, req)
	// }

	requestID := utils.ParseRequestID(ctx)

	resp := api.NewDiscoverServiceResponse(api.ExecuteSuccess, req)

	resp.Services = []*api.Service{}

	serviceIterProc := func(key string, value *model.Service) (bool, error) {
		if checkServiceMetadata(req.GetMetadata(), value, req.Business.GetValue(), req.Namespace.GetValue()) {
			service := &api.Service{
				Name:      utils.NewStringValue(value.Name),
				Namespace: utils.NewStringValue(value.Namespace),
			}
			resp.Services = append(resp.Services, service)
		}
		return true, nil
	}

	if err := s.caches.Service().IteratorServices(serviceIterProc); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return api.NewDiscoverServiceResponse(api.ExecuteException, req)
	}

	return resp
}

// checkServiceMetadata 判断请求元数据是否属于服务的元数据
func checkServiceMetadata(requestMeta map[string]string, service *model.Service, business, namespace string) bool {
	// if len(service.Meta) == 0 && len(business) == 0 {
	// 	return false
	// }
	if len(business) > 0 && business != service.Business {
		return false
	}

	if len(namespace) > 0 && namespace != service.Namespace {
		return false
	}

	if len(requestMeta) != 0 {
		for key, requestValue := range requestMeta {
			value, ok := service.Meta[key]
			if !ok || requestValue != value {
				return false
			}
		}
	}

	return true
}

// ServiceInstancesCache 根据服务名查询服务实例列表
func (s *Server) ServiceInstancesCache(ctx context.Context, req *api.Service) *api.DiscoverResponse {
	if req == nil {
		return api.NewDiscoverInstanceResponse(api.EmptyRequest, req)
	}
	if s.caches == nil {
		return api.NewDiscoverInstanceResponse(api.ClientAPINotOpen, req)
	}

	serviceName := req.GetName().GetValue()
	namespaceName := req.GetNamespace().GetValue()

	if serviceName == "" {
		return api.NewDiscoverInstanceResponse(api.InvalidServiceName, req)
	}

	// 消费服务为了兼容，可以不带namespace，server端使用默认的namespace
	if namespaceName == "" {
		namespaceName = DefaultNamespace
	}

	// 数据源都来自Cache，这里拿到的service，已经是源服务
	service := s.getServiceCache(serviceName, namespaceName)
	if service == nil {
		log.Errorf("[Server][Service][Instance] not found name(%s) namespace(%s) service", serviceName, namespaceName)
		return api.NewDiscoverInstanceResponse(api.NotFoundResource, req)
	}
	s.RecordDiscoverStatis(service.Name, service.Namespace)
	// 获取revision，如果revision一致，则不返回内容，直接返回一个状态码
	revision := s.caches.GetServiceInstanceRevision(service.ID)
	if revision == "" {
		// 不能直接获取，则需要重新计算，大部分情况都可以直接获取的
		// 获取instance数据，service已经是源服务，可以直接查找cache
		instances := s.caches.Instance().GetInstancesByServiceID(service.ID)
		var revisionErr error
		revision, revisionErr = s.GetServiceInstanceRevision(service.ID, instances)
		if revisionErr != nil {
			log.Errorf("[Server][Service][Instance] compute revision service(%s) err: %s",
				service.ID, revisionErr.Error())
			return api.NewDiscoverInstanceResponse(api.ExecuteException, req)
		}
	}
	if revision == req.GetRevision().GetValue() {
		return api.NewDiscoverInstanceResponse(api.DataNoChange, req)
	}

	// revision不一致，重新获取数据
	// 填充service数据
	resp := api.NewDiscoverInstanceResponse(api.ExecuteSuccess, service2Api(service))
	// 填充新的revision TODO
	resp.Service.Revision.Value = revision
	resp.Service.Namespace = req.GetNamespace()
	resp.Service.Name = req.GetName() // 别名场景，response需要保持和request的服务名一致
	// 填充instance数据
	resp.Instances = make([]*api.Instance, 0) // TODO
	_ = s.caches.Instance().
		IteratorInstancesWithService(service.ID, // service已经是源服务
			func(key string, value *model.Instance) (b bool, e error) {
				// 注意：这里的value是cache的，不修改cache的数据，通过getInstance，浅拷贝一份数据
				resp.Instances = append(resp.Instances, s.getInstance(req, value.Proto))
				return true, nil
			})

	return resp
}

// GetRoutingConfigWithCache 获取缓存中的路由配置信息
func (s *Server) GetRoutingConfigWithCache(ctx context.Context, req *api.Service) *api.DiscoverResponse {
	if s.caches == nil {
		return api.NewDiscoverRoutingResponse(api.ClientAPINotOpen, req)
	}
	if req == nil {
		return api.NewDiscoverRoutingResponse(api.EmptyRequest, req)
	}

	if req.GetName().GetValue() == "" {
		return api.NewDiscoverRoutingResponse(api.InvalidServiceName, req)
	}
	if req.GetNamespace().GetValue() == "" {
		return api.NewDiscoverRoutingResponse(api.InvalidNamespaceName, req)
	}

	resp := api.NewDiscoverRoutingResponse(api.ExecuteSuccess, nil)
	resp.Service = &api.Service{
		Name:      req.GetName(),
		Namespace: req.GetNamespace(),
	}

	// 先从缓存获取ServiceID，这里返回的是源服务
	svc := s.getServiceCache(req.GetName().GetValue(), req.GetNamespace().GetValue())
	if svc == nil {
		return api.NewDiscoverRoutingResponse(api.NotFoundService, req)
	}
	out, err := s.caches.RoutingConfig().GetRoutingConfigV1(svc.ID, svc.Name, svc.Namespace)
	if err != nil {
		log.Error("[Server][Service][Routing] discover routing", utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return api.NewDiscoverRoutingResponse(api.ExecuteException, req)
	}

	if out == nil {
		return resp
	}

	// 获取路由数据，并对比revision
	if out.GetRevision().GetValue() == req.GetRevision().GetValue() {
		return api.NewDiscoverRoutingResponse(api.DataNoChange, req)
	}

	// 数据不一致，发生了改变
	// 数据格式转换，service只需要返回二元组与routing的revision
	resp.Service.Revision = out.GetRevision()
	resp.Routing = out
	return resp
}

// GetRateLimitWithCache 获取缓存中的限流规则信息
func (s *Server) GetRateLimitWithCache(ctx context.Context, req *api.Service) *api.DiscoverResponse {
	if s.caches == nil {
		return api.NewDiscoverRoutingResponse(api.ClientAPINotOpen, req)
	}

	requestID := utils.ParseRequestID(ctx)

	if req == nil {
		return api.NewDiscoverRateLimitResponse(api.EmptyRequest, req)
	}

	if req.GetName().GetValue() == "" {
		return api.NewDiscoverRateLimitResponse(api.InvalidServiceName, req)
	}
	if req.GetNamespace().GetValue() == "" {
		return api.NewDiscoverRateLimitResponse(api.InvalidNamespaceName, req)
	}

	// 获取源服务
	service := s.getServiceCache(req.GetName().GetValue(), req.GetNamespace().GetValue())
	if service == nil {
		return api.NewDiscoverRateLimitResponse(api.NotFoundService, req)
	}

	resp := api.NewDiscoverRateLimitResponse(api.ExecuteSuccess, nil)
	// 服务名和request保持一致
	resp.Service = &api.Service{
		Name:      req.GetName(),
		Namespace: req.GetNamespace(),
	}

	// 获取最新的revision
	lastRevision := s.caches.RateLimit().GetLastRevision(service.ID)

	// 缓存中无此服务的限流规则
	if lastRevision == "" {
		return resp
	}

	if req.GetRevision().GetValue() == lastRevision {
		return api.NewDiscoverRateLimitResponse(api.DataNoChange, req)
	}

	// 获取限流规则数据
	resp.Service.Revision = utils.NewStringValue(lastRevision) // 更新revision

	resp.RateLimit = &api.RateLimit{
		Revision: utils.NewStringValue(lastRevision),
		Rules:    []*api.Rule{},
	}

	rateLimitIterProc := func(key string, value *model.RateLimit) (bool, error) {
		rateLimit, err := rateLimit2Client(req.GetName().GetValue(), req.GetNamespace().GetValue(), value)
		if err != nil {
			return false, err
		}
		resp.RateLimit.Rules = append(resp.RateLimit.Rules, rateLimit)
		return true, nil
	}

	err := s.caches.RateLimit().GetRateLimit(service.ID, rateLimitIterProc)
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return api.NewDiscoverRateLimitResponse(api.ExecuteException, req)
	}

	return resp
}

// GetCircuitBreakerWithCache 获取缓存中的熔断规则信息
func (s *Server) GetCircuitBreakerWithCache(ctx context.Context, req *api.Service) *api.DiscoverResponse {
	if s.caches == nil {
		return api.NewDiscoverCircuitBreakerResponse(api.ClientAPINotOpen, req)
	}
	requestID := utils.ParseRequestID(ctx)
	if req == nil {
		return api.NewDiscoverCircuitBreakerResponse(api.EmptyRequest, req)
	}

	if req.GetName().GetValue() == "" {
		return api.NewDiscoverCircuitBreakerResponse(api.InvalidServiceName, req)
	}
	if req.GetNamespace().GetValue() == "" {
		return api.NewDiscoverCircuitBreakerResponse(api.InvalidNamespaceName, req)
	}

	// 获取源服务
	service := s.getServiceCache(req.GetName().GetValue(), req.GetNamespace().GetValue())
	if service == nil {
		return api.NewDiscoverCircuitBreakerResponse(api.NotFoundService, req)
	}

	resp := api.NewDiscoverCircuitBreakerResponse(api.ExecuteSuccess, nil)
	// 服务名和request保持一致
	resp.Service = &api.Service{
		Name:      req.GetName(),
		Namespace: req.GetNamespace(),
	}

	out := s.caches.CircuitBreaker().GetCircuitBreakerConfig(service.ID)
	if out == nil {
		return resp
	}

	// 获取熔断规则数据，并对比revision
	if req.GetRevision().GetValue() == out.CircuitBreaker.Revision {
		return api.NewDiscoverCircuitBreakerResponse(api.DataNoChange, req)
	}

	// 数据不一致，发生了改变
	var err error
	resp.Service.Revision = utils.NewStringValue(out.CircuitBreaker.Revision)
	resp.CircuitBreaker, err = circuitBreaker2ClientAPI(out, req.GetName().GetValue(), req.GetNamespace().GetValue())
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return api.NewDiscoverCircuitBreakerResponse(api.ExecuteException, req)
	}
	return resp
}

// 根据ServiceID获取instances
func (s *Server) getInstancesCache(service *model.Service) []*model.Instance {
	id := s.getSourceServiceID(service)
	// TODO refer_filter还要处理一下
	return s.caches.Instance().GetInstancesByServiceID(id)
}

// 获取顶级服务ID
// 没有顶级ID，则返回自身
func (s *Server) getSourceServiceID(service *model.Service) string {
	if service == nil || service.ID == "" {
		return ""
	}
	// 找到parent服务，最多两级，因此不用递归查找
	if service.IsAlias() {
		return service.Reference
	}

	return service.ID

}

// 根据服务名获取服务缓存数据
// 注意，如果是服务别名查询，这里会返回别名的源服务，不会返回别名
func (s *Server) getServiceCache(name string, namespace string) *model.Service {
	sc := s.caches.Service()
	service := sc.GetServiceByName(name, namespace)
	if service == nil {
		return nil
	}
	// 如果是服务别名，继续查找一下
	if service.IsAlias() {
		service = sc.GetServiceByID(service.Reference)
		if service == nil {
			return nil
		}
	}

	if service.Meta == nil {
		service.Meta = make(map[string]string)
	}
	return service
}
