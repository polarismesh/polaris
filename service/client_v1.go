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

	"github.com/golang/protobuf/ptypes/wrappers"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/wrapperspb"

	cachetypes "github.com/polarismesh/polaris/cache/api"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/metrics"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

// RegisterInstance create one instance
func (s *Server) RegisterInstance(ctx context.Context, req *apiservice.Instance) *apiservice.Response {
	return s.CreateInstance(ctx, req)
}

// DeregisterInstance delete one instance
func (s *Server) DeregisterInstance(ctx context.Context, req *apiservice.Instance) *apiservice.Response {
	return s.DeleteInstance(ctx, req)
}

// ReportServiceContract report client service interface info
func (s *Server) ReportServiceContract(ctx context.Context, req *apiservice.ServiceContract) *apiservice.Response {
	cacheData := s.caches.ServiceContract().Get(ctx, &model.ServiceContract{
		Namespace: req.GetNamespace(),
		Service:   req.GetService(),
		Type:      req.GetName(),
		Version:   req.GetVersion(),
		Protocol:  req.GetProtocol(),
	})
	// 通过 Cache 模块减少无意义的 CreateServiceContract 逻辑
	if cacheData == nil || cacheData.Content != req.GetContent() {
		rsp := s.CreateServiceContract(ctx, req)
		if !isSuccessReportContract(rsp) {
			return rsp
		}
	}

	rsp := s.CreateServiceContractInterfaces(ctx, req, apiservice.InterfaceDescriptor_Client)
	return rsp
}

func isSuccessReportContract(rsp *apiservice.Response) bool {
	code := rsp.GetCode().GetValue()
	if code == uint32(apimodel.Code_ExecuteSuccess) {
		return true
	}
	if code == uint32(apimodel.Code_NoNeedUpdate) {
		return true
	}
	return false
}

// ReportClient 客户端上报信息
func (s *Server) ReportClient(ctx context.Context, req *apiservice.Client) *apiservice.Response {
	// 客户端信息不写入到DB中
	host := req.GetHost().GetValue()
	// 从CMDB查询地理位置信息
	if s.cmdb != nil {
		location, err := s.cmdb.GetLocation(host)
		if err != nil {
			log.Errora(utils.RequestID(ctx), zap.Error(err))
		}
		if location != nil {
			req.Location = location.Proto
		}
	}

	// save the client with unique id into store
	if len(req.GetId().GetValue()) > 0 {
		return s.checkAndStoreClient(ctx, req)
	}
	out := &apiservice.Client{
		Host:     req.GetHost(),
		Location: req.Location,
	}
	return api.NewClientResponse(apimodel.Code_ExecuteSuccess, out)
}

// GetPrometheusTargets Used for client acquisition service information
func (s *Server) GetPrometheusTargets(ctx context.Context,
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

	// 加入北极星集群自身
	checkers := s.healthServer.ListCheckerServer()
	for i := range checkers {
		checker := checkers[i]
		target := model.PrometheusTarget{
			Targets: []string{fmt.Sprintf("%s:%d", checker.Host(), metrics.GetMetricsPort())},
			Labels: map[string]string{
				"__metrics_path__":         "/metrics",
				"__scheme__":               "http",
				"__meta_polaris_client_id": checker.ID(),
			},
		}
		targets = append(targets, target)
	}

	return &model.PrometheusDiscoveryResponse{
		Code:     api.ExecuteSuccess,
		Response: targets,
	}
}

// GetServiceWithCache 查询服务列表
func (s *Server) GetServiceWithCache(ctx context.Context, req *apiservice.Service) *apiservice.DiscoverResponse {
	resp := api.NewDiscoverServiceResponse(apimodel.Code_ExecuteSuccess, req)
	var (
		revision string
		svcs     []*model.Service
	)

	if req.GetNamespace().GetValue() != "" {
		revision, svcs = s.Cache().Service().ListServices(req.GetNamespace().GetValue())
	} else {
		revision, svcs = s.Cache().Service().ListAllServices()
	}
	if revision == "" {
		return resp
	}

	log.Debug("[Service][Discover] list servies", zap.Int("size", len(svcs)), zap.String("revision", revision))
	if revision == req.GetRevision().GetValue() {
		return api.NewDiscoverServiceResponse(apimodel.Code_DataNoChange, req)
	}

	ret := make([]*apiservice.Service, 0, len(svcs))
	for i := range svcs {
		ret = append(ret, &apiservice.Service{
			Namespace: utils.NewStringValue(svcs[i].Namespace),
			Name:      utils.NewStringValue(svcs[i].Name),
			Metadata:  svcs[i].Meta,
		})
	}

	resp.Services = ret
	resp.Service = &apiservice.Service{
		Namespace: utils.NewStringValue(req.GetNamespace().GetValue()),
		Name:      utils.NewStringValue(req.GetName().GetValue()),
		Revision:  utils.NewStringValue(revision),
	}

	return resp
}

// ServiceInstancesCache 根据服务名查询服务实例列表
func (s *Server) ServiceInstancesCache(ctx context.Context, filter *apiservice.DiscoverFilter,
	req *apiservice.Service) *apiservice.DiscoverResponse {

	resp := createCommonDiscoverResponse(req, apiservice.DiscoverResponse_INSTANCE)
	serviceName := req.GetName().GetValue()
	namespaceName := req.GetNamespace().GetValue()

	// 数据源都来自Cache，这里拿到的service，已经是源服务
	aliasFor, visibleServices := s.findVisibleServices(serviceName, namespaceName, req)
	if len(visibleServices) == 0 {
		log.Infof("[Server][Service][Instance] not found name(%s) namespace(%s) service",
			serviceName, namespaceName)
		return api.NewDiscoverInstanceResponse(apimodel.Code_NotFoundResource, req)
	}

	revisions := make([]string, 0, len(visibleServices)+1)
	finalInstances := make(map[string]*apiservice.Instance, 128)
	for _, svc := range visibleServices {
		revision := s.caches.Service().GetRevisionWorker().GetServiceInstanceRevision(svc.ID)
		if revision == "" {
			revision = utils.NewUUID()
		}
		revisions = append(revisions, revision)
	}
	aggregateRevision, err := cachetypes.CompositeComputeRevision(revisions)
	if err != nil {
		log.Errorf("[Server][Service][Instance] compute multi revision service(%s) err: %s",
			aliasFor.ID, err.Error())
		return api.NewDiscoverInstanceResponse(apimodel.Code_ExecuteException, req)
	}
	if aggregateRevision == req.GetRevision().GetValue() {
		return api.NewDiscoverInstanceResponse(apimodel.Code_DataNoChange, req)
	}

	for _, svc := range visibleServices {
		specSvc := &apiservice.Service{
			Id:        utils.NewStringValue(svc.ID),
			Name:      utils.NewStringValue(svc.Name),
			Namespace: utils.NewStringValue(svc.Namespace),
		}
		ret := s.caches.Instance().DiscoverServiceInstances(specSvc.GetId().GetValue(), filter.GetOnlyHealthyInstance())
		for i := range ret {
			copyIns := s.getInstance(specSvc, ret[i].Proto)
			// 注意：这里的value是cache的，不修改cache的数据，通过getInstance，浅拷贝一份数据
			finalInstances[copyIns.GetId().GetValue()] = copyIns
		}
	}

	// 填充service数据
	resp.Service = service2Api(aliasFor)
	// 这里需要把服务信息改为用户请求的服务名以及命名空间
	resp.Service.Name = req.GetName()
	resp.Service.Namespace = req.GetNamespace()
	resp.Service.Revision = utils.NewStringValue(aggregateRevision)
	// 塞入源服务信息数据
	resp.AliasFor = service2Api(aliasFor)
	// 填充instance数据
	resp.Instances = make([]*apiservice.Instance, 0, len(finalInstances))
	for i := range finalInstances {
		// 注意：这里的value是cache的，不修改cache的数据，通过getInstance，浅拷贝一份数据
		resp.Instances = append(resp.Instances, finalInstances[i])
	}
	return resp
}

func (s *Server) findVisibleServices(serviceName, namespaceName string, req *apiservice.Service) (*model.Service, []*model.Service) {
	visibleServices := make([]*model.Service, 0, 4)
	// 数据源都来自Cache，这里拿到的service，已经是源服务
	aliasFor := s.getServiceCache(serviceName, namespaceName)
	if aliasFor == nil {
		aliasFor = &model.Service{
			Name:      serviceName,
			Namespace: namespaceName,
		}
		ret := s.caches.Service().GetVisibleServicesInOtherNamespace(serviceName, namespaceName)
		if len(ret) == 0 {
			return nil, nil
		}
		visibleServices = append(visibleServices, ret...)
	} else {
		visibleServices = append(visibleServices, aliasFor)
	}

	return aliasFor, visibleServices
}

// GetRoutingConfigWithCache 获取缓存中的路由配置信息
func (s *Server) GetRoutingConfigWithCache(ctx context.Context, req *apiservice.Service) *apiservice.DiscoverResponse {
	resp := createCommonDiscoverResponse(req, apiservice.DiscoverResponse_ROUTING)
	aliasFor := s.findServiceAlias(req)

	out, err := s.caches.RoutingConfig().GetRouterConfig(aliasFor.ID, aliasFor.Name, aliasFor.Namespace)
	if err != nil {
		log.Error("[Server][Service][Routing] discover routing", utils.RequestID(ctx), zap.Error(err))
		return api.NewDiscoverRoutingResponse(apimodel.Code_ExecuteException, req)
	}
	if out == nil {
		return resp
	}

	// 获取路由数据，并对比revision
	if out.GetRevision().GetValue() == req.GetRevision().GetValue() {
		return api.NewDiscoverRoutingResponse(apimodel.Code_DataNoChange, req)
	}

	// 数据不一致，发生了改变
	// 数据格式转换，service只需要返回二元组与routing的revision
	resp.Service.Revision = out.GetRevision()
	resp.Routing = out
	resp.AliasFor = &apiservice.Service{
		Name:      utils.NewStringValue(aliasFor.Name),
		Namespace: utils.NewStringValue(aliasFor.Namespace),
	}
	return resp
}

// GetRateLimitWithCache 获取缓存中的限流规则信息
func (s *Server) GetRateLimitWithCache(ctx context.Context, req *apiservice.Service) *apiservice.DiscoverResponse {
	resp := createCommonDiscoverResponse(req, apiservice.DiscoverResponse_RATE_LIMIT)
	aliasFor := s.findServiceAlias(req)

	rules, revision := s.caches.RateLimit().GetRateLimitRules(model.ServiceKey{
		Namespace: aliasFor.Namespace,
		Name:      aliasFor.Name,
	})
	if len(rules) == 0 || revision == "" {
		return resp
	}
	if req.GetRevision().GetValue() == revision {
		return api.NewDiscoverRateLimitResponse(apimodel.Code_DataNoChange, req)
	}
	resp.RateLimit = &apitraffic.RateLimit{
		Revision: utils.NewStringValue(revision),
		Rules:    []*apitraffic.Rule{},
	}
	for i := range rules {
		rateLimit, err := rateLimit2Client(req.GetName().GetValue(), req.GetNamespace().GetValue(), rules[i])
		if rateLimit == nil || err != nil {
			continue
		}
		resp.RateLimit.Rules = append(resp.RateLimit.Rules, rateLimit)
	}

	// 塞入源服务信息数据
	resp.AliasFor = &apiservice.Service{
		Namespace: utils.NewStringValue(aliasFor.Namespace),
		Name:      utils.NewStringValue(aliasFor.Name),
	}
	// 服务名和request保持一致
	resp.Service = &apiservice.Service{
		Name:      req.GetName(),
		Namespace: req.GetNamespace(),
		Revision:  utils.NewStringValue(revision),
	}
	return resp
}

func (s *Server) GetFaultDetectWithCache(ctx context.Context, req *apiservice.Service) *apiservice.DiscoverResponse {
	resp := createCommonDiscoverResponse(req, apiservice.DiscoverResponse_FAULT_DETECTOR)
	aliasFor := s.findServiceAlias(req)

	out := s.caches.FaultDetector().GetFaultDetectConfig(aliasFor.Name, aliasFor.Namespace)
	if out == nil || out.Revision == "" {
		return resp
	}

	if req.GetRevision().GetValue() == out.Revision {
		return api.NewDiscoverFaultDetectorResponse(apimodel.Code_DataNoChange, req)
	}

	// 数据不一致，发生了改变
	var err error
	resp.AliasFor = &apiservice.Service{
		Name:      utils.NewStringValue(aliasFor.Name),
		Namespace: utils.NewStringValue(aliasFor.Namespace),
	}
	resp.Service.Revision = utils.NewStringValue(out.Revision)
	resp.FaultDetector, err = faultDetectRule2ClientAPI(out)
	if err != nil {
		log.Error(err.Error(), utils.RequestID(ctx))
		return api.NewDiscoverFaultDetectorResponse(apimodel.Code_ExecuteException, req)
	}
	return resp
}

// GetCircuitBreakerWithCache 获取缓存中的熔断规则信息
func (s *Server) GetCircuitBreakerWithCache(ctx context.Context, req *apiservice.Service) *apiservice.DiscoverResponse {
	resp := createCommonDiscoverResponse(req, apiservice.DiscoverResponse_CIRCUIT_BREAKER)
	// 获取源服务
	aliasFor := s.findServiceAlias(req)
	out := s.caches.CircuitBreaker().GetCircuitBreakerConfig(aliasFor.Name, aliasFor.Namespace)
	if out == nil || out.Revision == "" {
		return resp
	}

	// 获取熔断规则数据，并对比revision
	if len(req.GetRevision().GetValue()) > 0 && req.GetRevision().GetValue() == out.Revision {
		return api.NewDiscoverCircuitBreakerResponse(apimodel.Code_DataNoChange, req)
	}

	// 数据不一致，发生了改变
	var err error
	resp.AliasFor = &apiservice.Service{
		Name:      utils.NewStringValue(aliasFor.Name),
		Namespace: utils.NewStringValue(aliasFor.Namespace),
	}
	resp.Service.Revision = utils.NewStringValue(out.Revision)
	resp.CircuitBreaker, err = circuitBreaker2ClientAPI(out, req.GetName().GetValue(), req.GetNamespace().GetValue())
	if err != nil {
		log.Error(err.Error(), utils.RequestID(ctx))
		return api.NewDiscoverCircuitBreakerResponse(apimodel.Code_ExecuteException, req)
	}
	return resp
}

// GetServiceContractWithCache User Client Get ServiceContract Rule Information
func (s *Server) GetServiceContractWithCache(ctx context.Context,
	req *apiservice.ServiceContract) *apiservice.Response {
	resp := api.NewResponse(apimodel.Code_ExecuteSuccess)
	// 服务名和request保持一致
	resp.Service = &apiservice.Service{
		Name:      wrapperspb.String(req.GetService()),
		Namespace: wrapperspb.String(req.GetNamespace()),
	}

	// 获取源服务
	aliasFor := s.findServiceAlias(resp.Service)

	out := s.caches.ServiceContract().Get(ctx, &model.ServiceContract{
		Namespace: aliasFor.Namespace,
		Service:   aliasFor.Name,
		Version:   req.Version,
		Type:      req.Name,
		Protocol:  req.Protocol,
	})
	if out == nil {
		resp.Code = wrapperspb.UInt32(uint32(apimodel.Code_NotFoundResource))
		resp.Info = wrapperspb.String(api.Code2Info(uint32(apimodel.Code_NotFoundResource)))
		return resp
	}

	// 获取熔断规则数据，并对比revision
	if len(req.GetRevision()) > 0 && req.GetRevision() == out.Revision {
		resp.Code = wrapperspb.UInt32(uint32(apimodel.Code_DataNoChange))
		resp.Info = wrapperspb.String(api.Code2Info(uint32(apimodel.Code_DataNoChange)))
		return resp
	}

	resp.Service.Revision = wrapperspb.String(out.Revision)
	resp.ServiceContract = out.ToSpec()
	return resp
}

// GetLaneRuleWithCache fetch lane rule by client
func (s *Server) GetLaneRuleWithCache(ctx context.Context, req *apiservice.Service) *apiservice.DiscoverResponse {
	resp := createCommonDiscoverResponse(req, apiservice.DiscoverResponse_LANE)
	// 获取源服务
	aliasFor := s.findServiceAlias(req)
	out, revision := s.caches.LaneRule().GetLaneRules(aliasFor)
	if out == nil || revision == "" {
		return resp
	}

	// 获取泳道规则数据，并对比revision
	if len(req.GetRevision().GetValue()) > 0 && req.GetRevision().GetValue() == revision {
		return api.NewDiscoverLaneResponse(apimodel.Code_DataNoChange, req)
	}

	resp.AliasFor = &apiservice.Service{
		Name:      utils.NewStringValue(aliasFor.Name),
		Namespace: utils.NewStringValue(aliasFor.Namespace),
	}
	resp.Service.Revision = utils.NewStringValue(revision)
	resp.Lanes = make([]*apitraffic.LaneGroup, 0, len(out))
	for i := range out {
		resp.Lanes = append(resp.Lanes, out[i].Proto)
	}
	return resp
}

func (s *Server) findServiceAlias(req *apiservice.Service) *model.Service {
	// 获取源服务
	aliasFor := s.getServiceCache(req.GetName().GetValue(), req.GetNamespace().GetValue())
	if aliasFor == nil {
		aliasFor = &model.Service{
			Namespace: req.GetNamespace().GetValue(),
			Name:      req.GetName().GetValue(),
		}
	}
	return aliasFor
}

func CreateCommonDiscoverResponse(req *apiservice.Service,
	dT apiservice.DiscoverResponse_DiscoverResponseType) *apiservice.DiscoverResponse {
	return createCommonDiscoverResponse(req, dT)
}

func createCommonDiscoverResponse(req *apiservice.Service,
	dT apiservice.DiscoverResponse_DiscoverResponseType) *apiservice.DiscoverResponse {
	return &apiservice.DiscoverResponse{
		Code: &wrappers.UInt32Value{Value: uint32(apimodel.Code_ExecuteSuccess)},
		Info: &wrappers.StringValue{Value: api.Code2Info(uint32(apimodel.Code_ExecuteSuccess))},
		Type: dT,
		Service: &apiservice.Service{
			Name:      req.GetName(),
			Namespace: req.GetNamespace(),
		},
	}
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
