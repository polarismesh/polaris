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

package xdsserverv3

import (
	"errors"
	"strconv"
	"time"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"go.uber.org/atomic"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/apiserver/xdsserverv3/cache"
	"github.com/polarismesh/polaris/apiserver/xdsserverv3/resource"
	"github.com/polarismesh/polaris/common/metrics"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/plugin"
	"github.com/polarismesh/polaris/service"
)

// XdsResourceGenerator is the xDS resource generator
type XdsResourceGenerator struct {
	namingServer service.DiscoverServer
	cache        *cache.XDSCache
	versionNum   *atomic.Uint64
	xdsNodesMgr  *resource.XDSNodeManager
}

func (x *XdsResourceGenerator) Generate(versionLocal string,
	registryInfo map[string]map[model.ServiceKey]*resource.ServiceInfo) {

	// 如果没有任何一个 XDS Node 接入则不会生成与 Node 有关的 XDS Resource
	if x.xdsNodesMgr.HasEnvoyNodes() {
		// 只构建 Sidecar 特有的 XDS 数据
		_ = x.buildSidecarXDSCache(registryInfo)
	}

	// CDS/EDS/VHDS 一起构建
	for namespace, services := range registryInfo {
		opt := &resource.BuildOption{
			RunType:          resource.RunTypeSidecar,
			Namespace:        namespace,
			Services:         services,
			TrafficDirection: corev3.TrafficDirection_OUTBOUND,
			TLSMode:          resource.TLSModeNone,
		}
		x.buildAndDeltaUpdate(resource.RDS, opt)
		x.buildAndDeltaUpdate(resource.EDS, opt)
		x.buildAndDeltaUpdate(resource.VHDS, opt)
		// 默认构建没有设置 TLS 的 CDS 资源
		x.buildAndDeltaUpdate(resource.CDS, opt)

		// 构建设置了 TLS Mode == Strict 的 CDS 资源
		opt.TLSMode = resource.TLSModeStrict
		x.buildAndDeltaUpdate(resource.CDS, opt)
		// 构建设置了 TLS Mode == Permissive 的 CDS 资源
		opt.TLSMode = resource.TLSModePermissive
		x.buildAndDeltaUpdate(resource.CDS, opt)
	}
}

func (x *XdsResourceGenerator) buildAndDeltaUpdate(xdsType resource.XDSType, opt *resource.BuildOption) {
	xxds, err := x.generateXDSResource(xdsType, opt)
	if err != nil {
		log.Error("[XDS][Sidecar] build common fail", zap.String("type", xdsType.String()), zap.Error(err))
		return
	}

	typeUrl := xdsType.ResourceType()
	cacheKey := xdsType.ResourceType() + "~" + opt.Namespace
	if opt.TLSMode != resource.TLSModeNone {
		cacheKey = cacheKey + "~" + string(opt.TLSMode)
	}
	// 与 XDS Node 有关的全部都有单独的 Cache 缓存处理
	if opt.Client != nil {
		cacheKey = xdsType.ResourceType() + "~" + opt.Client.Node.Id
	}

	if err := x.cache.DeltaUpdateResource(cacheKey, typeUrl, cachev3.IndexRawResourcesByName(xxds)); err != nil {
		log.Error("[XDS][Sidecar] delta update fail", zap.String("cache-key", cacheKey),
			zap.String("type", xdsType.String()), zap.Error(err))
		return
	}
}

func (x *XdsResourceGenerator) buildSidecarXDSCache(registryInfo map[string]map[model.ServiceKey]*resource.ServiceInfo) error {

	nodes := x.xdsNodesMgr.ListSidecarNodes()
	if len(nodes) == 0 || len(registryInfo) == 0 {
		// 如果没有任何一个 XDS Sidecar Node 客户端，不做任何操作
		log.Info("[XDS][Sidecar] xds nodes or registryInfo is empty", zap.Int("nodes", len(nodes)),
			zap.Int("register", len(registryInfo)))
		return nil
	}

	for i := range nodes {
		node := nodes[i]
		xdsNode := node
		opt := &resource.BuildOption{
			RunType:        resource.RunTypeSidecar,
			Client:         xdsNode,
			TLSMode:        node.TLSMode,
			Namespace:      xdsNode.GetSelfNamespace(),
			OpenOnDemand:   xdsNode.OpenOnDemand,
			OnDemandServer: xdsNode.OnDemandServer,
			SelfService: model.ServiceKey{
				Namespace: xdsNode.GetSelfNamespace(),
				Name:      xdsNode.GetSelfService(),
			},
		}

		opt.TrafficDirection = corev3.TrafficDirection_OUTBOUND
		// 构建 INBOUND LDS 资源
		x.buildAndDeltaUpdate(resource.LDS, opt)
		// 构建 INBOUND RDS 资源
		x.buildAndDeltaUpdate(resource.RDS, opt)
		opt.TrafficDirection = corev3.TrafficDirection_INBOUND
		// 构建 INBOUND LDS 资源
		x.buildAndDeltaUpdate(resource.LDS, opt)
		// 构建 INBOUND EDS 资源
		x.buildAndDeltaUpdate(resource.EDS, opt)
		// 构建 INBOUND RDS 资源
		x.buildAndDeltaUpdate(resource.RDS, opt)
	}
	return nil
}

// buildGatewayXDSCache 网关场景是允许跨命名空间直接进行访问
func (x *XdsResourceGenerator) buildGatewayXDSCache(versionLocal string,
	registryInfo map[string]map[model.ServiceKey]*resource.ServiceInfo) error {

	nodes := x.xdsNodesMgr.ListGatewayNodes()
	if len(nodes) == 0 || len(registryInfo) == 0 {
		// 如果没有任何一个 XDS Gateway Node 客户端，不做任何操作
		log.Info("[XDS][Gateway][V2] xds nodes or registryInfo is empty", zap.Int("nodes", len(nodes)),
			zap.Int("registr", len(registryInfo)))
		return nil
	}

	alreadyMakeCache := map[string]struct{}{}
	for i := range nodes {
		node := nodes[i]
		cacheKey := (resource.PolarisNodeHash{}).ID(node.Node)
		if _, exist := alreadyMakeCache[cacheKey]; exist {
			continue
		}
		alreadyMakeCache[cacheKey] = struct{}{}
		if err := x.makeGatewaySnapshot(node, node.TLSMode, versionLocal, registryInfo); err != nil {
			log.Error("[XDS][Gateway][V2] make snapshot fail", zap.String("cacheKey", cacheKey),
				zap.Error(err))
		}
	}
	return nil
}

// makeGatewaySnapshot nodeId must be like gateway~namespace
func (x *XdsResourceGenerator) makeGatewaySnapshot(xdsNode *resource.XDSClient, tlsMode resource.TLSMode,
	version string, registryInfo map[string]map[model.ServiceKey]*resource.ServiceInfo) error {

	opt := &resource.BuildOption{
		TLSMode: tlsMode,
	}
	var (
		allEndpoints []types.Resource
		allClusters  []types.Resource
		allRouters   []types.Resource
	)
	for namespace, services := range registryInfo {
		opt.Services = services
		opt.Namespace = namespace
		// 构建 endpoints XDS 资源缓存数据，这里不需要下发网关的自己的
		endpoints, err := x.generateXDSResource(resource.EDS, opt)
		if err != nil {
			return err
		}
		allEndpoints = append(allEndpoints, endpoints...)
		// 构建 cluster XDS 资源缓存数据
		clusters, err := x.generateXDSResource(resource.CDS, opt)
		if err != nil {
			return err
		}
		allClusters = append(allClusters, clusters...)
		// 构建 route XDS 资源缓存
		routers, err := x.generateXDSResource(resource.RDS, opt)
		if err != nil {
			return err
		}
		allRouters = append(allRouters, routers...)
	}

	// 构建 listener XDS 资源缓存
	listeners, err := x.generateXDSResource(resource.LDS, opt)
	if err != nil {
		return err
	}

	resources := make(map[resourcev3.Type][]types.Resource)
	resources[resourcev3.EndpointType] = allEndpoints
	resources[resourcev3.ClusterType] = allClusters
	resources[resourcev3.RouteType] = allRouters
	resources[resourcev3.ListenerType] = listeners
	cacheKey := (resource.PolarisNodeHash{}).ID(xdsNode.Node)

	for typeUrl, resources := range resources {
		if err := x.cache.DeltaUpdateResource(xdsNode.Node.Id, typeUrl, cachev3.IndexRawResourcesByName(resources)); err != nil {
			// TODO: need log
		}
	}
	// 为每个 nodeId 刷写 cache ，推送 xds 更新
	log.Info("[XDS][Gateway] upsert xds resource success", zap.String("cacheKey", cacheKey))
	return nil
}

func (x *XdsResourceGenerator) generateXDSResource(xdsType resource.XDSType,
	opt *resource.BuildOption) ([]types.Resource, error) {

	// TODO 需要预埋相关 XDS 资源生成时间开销
	start := time.Now()
	defer func() {
		plugin.GetStatis().ReportCallMetrics(metrics.CallMetric{
			Type:     metrics.XDSResourceBuildCallMetric,
			API:      string(xdsType),
			Protocol: "XDS",
			Times:    1,
			Duration: time.Since(start),
			Labels: map[string]string{
				"service_count": strconv.FormatInt(int64(len(opt.Services)), 10),
				"tls_mode":      string(opt.TLSMode),
			},
		})
	}()

	var (
		xdsBuilder resource.XDSBuilder
	)
	switch xdsType {
	case resource.CDS:
		xdsBuilder = &CDSBuilder{}
	case resource.EDS:
		xdsBuilder = &EDSBuilder{}
	case resource.LDS:
		xdsBuilder = &LDSBuilder{}
	case resource.RDS:
		xdsBuilder = &RDSBuilder{}
	case resource.VHDS:
		xdsBuilder = &VHDSBuilder{}
	default:
		return nil, errors.New("unsupport xds build type")
	}

	// 构建 XDS 资源缓存数据
	xdsBuilder.Init(x.namingServer)
	resources, err := xdsBuilder.Generate(opt)
	if err != nil {
		return nil, err
	}
	return resources.([]types.Resource), nil
}
