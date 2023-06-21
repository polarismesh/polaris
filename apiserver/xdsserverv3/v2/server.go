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

package v2

import (
	"context"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"go.uber.org/atomic"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/apiserver/xdsserverv3/resource"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/service"
)

func New(opt ...options) *XDSServer {
	svr := &XDSServer{}
	for i := range opt {
		opt[i](svr)
	}
	return svr
}

// XDSServer is the xDS server
type XDSServer struct {
	namingServer service.DiscoverServer
	cache        cachev3.SnapshotCache
	versionNum   *atomic.Uint64
	xdsNodesMgr  *resource.XDSNodeManager
}

func (x *XDSServer) Generate(versionLocal string,
	registryInfo map[string]map[model.ServiceKey]*resource.ServiceInfo) {

	x.buildSidecarXDSCache(versionLocal, registryInfo)
	x.buildGatewayXDSCache(versionLocal, registryInfo)
}

func (x *XDSServer) buildSidecarXDSCache(versionLocal string,
	registryInfo map[string]map[model.ServiceKey]*resource.ServiceInfo) error {

	nodes := x.xdsNodesMgr.ListSidecarNodes()
	if len(nodes) == 0 || len(registryInfo) == 0 {
		// 如果没有任何一个 XDS Sidecar Node 客户端，不做任何操作
		return nil
	}

	alreadyMakeCache := map[string]struct{}{}
	for i := range nodes {
		node := nodes[i]
		if node.IsGateway() {
			log.Errorf("[XDS][Sidecar][V2] xds node=%s run type not sidecar or info is invalid", node.Node.Id)
			continue
		}

		cacheKey := (resource.PolarisNodeHash{}).ID(node.Node)
		if _, exist := alreadyMakeCache[cacheKey]; exist {
			continue
		}
		alreadyMakeCache[cacheKey] = struct{}{}

		selfNamespace := node.GetSelfNamespace()
		services := registryInfo[selfNamespace]
		if len(services) == 0 {
			continue
		}
		x.makeSidecarSnapshot(node, node.TLSMode, versionLocal, services)
	}
	return nil
}

// buildGatewayXDSCache 网关场景是允许跨命名空间直接进行访问
func (x *XDSServer) buildGatewayXDSCache(versionLocal string,
	registryInfo map[string]map[model.ServiceKey]*resource.ServiceInfo) error {

	nodes := x.xdsNodesMgr.ListSidecarNodes()
	if len(nodes) == 0 || len(registryInfo) == 0 {
		// 如果没有任何一个 XDS Gateway Node 客户端，不做任何操作
		return nil
	}

	alreadyMakeCache := map[string]struct{}{}
	for i := range nodes {
		node := nodes[i]
		if !node.IsGateway() {
			log.Errorf("[XDS][Gateway][V2] xds node=%s run type not gateway or info is invalid", node.Node.Id)
			continue
		}

		cacheKey := (resource.PolarisNodeHash{}).ID(node.Node)
		if _, exist := alreadyMakeCache[cacheKey]; exist {
			continue
		}
		alreadyMakeCache[cacheKey] = struct{}{}
		x.makeGatewaySnapshot(node, node.TLSMode, versionLocal, registryInfo)
	}
	return nil
}

// makeSidecarSnapshot nodeId must be like sideacr~namespace or namespace
func (x *XDSServer) makeSidecarSnapshot(xdsNode *resource.XDSClient, tlsMode resource.TLSMode,
	version string, services map[model.ServiceKey]*resource.ServiceInfo) error {

	// 构建所有 XDS 的 INBOUND Snapshot Resource
	opt := &resource.BuildOption{
		Services:         services,
		TLSMode:          tlsMode,
		TrafficDirection: corev3.TrafficDirection_INBOUND,
		VersionLocal:     version,
		Namespace:        xdsNode.GetSelfNamespace(),
	}
	// 构建 endpoints XDS 资源缓存数据
	inBoundEndpoints, err := x.generateXDSResource(resource.EDS, xdsNode, opt)
	if err != nil {
		return err
	}
	// 构建 cluster XDS 资源缓存数据
	inBoundClusters, err := x.generateXDSResource(resource.CDS, xdsNode, opt)
	if err != nil {
		return err
	}
	// 构建 route XDS 资源缓存
	inBoundRouters, err := x.generateXDSResource(resource.RDS, xdsNode, opt)
	if err != nil {
		return err
	}
	// 构建 listener XDS 资源缓存
	inBoundListeners, err := x.generateXDSResource(resource.LDS, xdsNode, opt)
	if err != nil {
		return err
	}

	// 构建所有 XDS 的 INBOUND Snapshot Resource
	opt = &resource.BuildOption{
		Services:         services,
		TLSMode:          tlsMode,
		TrafficDirection: corev3.TrafficDirection_OUTBOUND,
	}
	// 构建 endpoints XDS 资源缓存数据
	outBoundEndpoints, err := x.generateXDSResource(resource.EDS, xdsNode, opt)
	if err != nil {
		return err
	}
	// 构建 cluster XDS 资源缓存数据
	outBoundClusters, err := x.generateXDSResource(resource.CDS, xdsNode, opt)
	if err != nil {
		return err
	}
	// 构建 route XDS 资源缓存
	outBoundRouters, err := x.generateXDSResource(resource.RDS, xdsNode, opt)
	if err != nil {
		return err
	}
	// 构建 listener XDS 资源缓存
	outBoundListeners, err := x.generateXDSResource(resource.LDS, xdsNode, opt)
	if err != nil {
		return err
	}

	resources := make(map[resourcev3.Type][]types.Resource)
	resources[resourcev3.EndpointType] = append(outBoundEndpoints, inBoundEndpoints...)
	resources[resourcev3.ClusterType] = append(outBoundClusters, inBoundClusters...)
	resources[resourcev3.RouteType] = append(outBoundRouters, inBoundRouters...)
	resources[resourcev3.ListenerType] = append(outBoundListeners, inBoundListeners...)

	cacheKey := (resource.PolarisNodeHash{}).ID(xdsNode.Node)
	snapshot, err := cachev3.NewSnapshot(version, resources)
	if err != nil {
		log.Errorf("[XDS][Sidecar][V2] fail to create snapshot", zap.String("cacheKey", cacheKey), zap.Error(err))
		return err
	}
	if err := snapshot.Consistent(); err != nil {
		return err
	}

	// 为每个 nodeId 刷写 cache ，推送 xds 更新
	if err := x.cache.SetSnapshot(context.Background(), cacheKey, snapshot); err != nil {
		log.Error("[XDS][Sidecar][V2] upsert snapshot error", zap.String("cacheKey", cacheKey), zap.Error(err))
		return err
	}
	log.Info("[XDS][Sidecar][V2] upsert snapshot success", zap.String("cacheKey", cacheKey),
		zap.String("snapshot", string(resource.DumpSnapShotJSON(snapshot))))
	return nil
}

// makeGatewaySnapshot nodeId must be like gateway~namespace
func (x *XDSServer) makeGatewaySnapshot(xdsNode *resource.XDSClient, tlsMode resource.TLSMode,
	version string, registryInfo map[string]map[model.ServiceKey]*resource.ServiceInfo) error {

	nodeId := xdsNode.Node.Id

	opt := &resource.BuildOption{
		TLSMode:          tlsMode,
		TrafficDirection: corev3.TrafficDirection_OUTBOUND,
		VersionLocal:     version,
	}
	var (
		allEndpoints []types.Resource
		allClusters  []types.Resource
		allRouters   []types.Resource
	)
	for namespace, services := range registryInfo {
		opt.Services = services
		opt.Namespace = namespace
		// 构建 endpoints XDS 资源缓存数据
		endpoints, err := x.generateXDSResource(resource.EDS, xdsNode, opt)
		if err != nil {
			return err
		}
		allEndpoints = append(allEndpoints, endpoints...)
		// 构建 cluster XDS 资源缓存数据
		clusters, err := x.generateXDSResource(resource.CDS, xdsNode, opt)
		if err != nil {
			return err
		}
		allClusters = append(allClusters, clusters...)
		// 构建 route XDS 资源缓存
		routers, err := x.generateXDSResource(resource.RDS, xdsNode, opt)
		if err != nil {
			return err
		}
		allRouters = append(allRouters, routers...)
	}

	// 注意: 网关这里 LDS 需要设置的 INBOUND 类型参数
	opt.TrafficDirection = corev3.TrafficDirection_INBOUND
	// 构建 listener XDS 资源缓存
	listeners, err := x.generateXDSResource(resource.LDS, xdsNode, opt)
	if err != nil {
		return err
	}

	resources := make(map[resourcev3.Type][]types.Resource)
	resources[resourcev3.EndpointType] = allEndpoints
	resources[resourcev3.ClusterType] = allClusters
	resources[resourcev3.RouteType] = allRouters
	resources[resourcev3.ListenerType] = listeners
	snapshot, err := cachev3.NewSnapshot(version, resources)
	if err != nil {
		log.Errorf("[XDS][Gateway][V2] fail to create snapshot for %s, err is %v", nodeId, err)
		return err
	}
	if err := snapshot.Consistent(); err != nil {
		return err
	}

	cacheKey := (resource.PolarisNodeHash{}).ID(xdsNode.Node)
	// 为每个 nodeId 刷写 cache ，推送 xds 更新
	if err := x.cache.SetSnapshot(context.Background(), cacheKey, snapshot); err != nil {
		log.Error("[XDS][Gateway][V2] upsert snapshot error", zap.String("cacheKey", cacheKey), zap.Error(err))
		return err
	}
	log.Info("[XDS][Gateway][V2] upsert snapshot success", zap.String("cacheKey", cacheKey),
		zap.String("snapshot", string(resource.DumpSnapShotJSON(snapshot))))
	return nil
}

func (x *XDSServer) generateXDSResource(xdsType resource.XDSType, xdsNode *resource.XDSClient,
	opt *resource.BuildOption) ([]types.Resource, error) {

	// 构建 XDS 资源缓存数据
	xdsBuilder := resource.GetBuilder(xdsType)
	xdsBuilder.Init(xdsNode, x.namingServer)
	resources, err := xdsBuilder.Generate(opt)
	if err != nil {
		return nil, err
	}
	return resources.([]types.Resource), nil
}
