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
	"context"
	"errors"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"go.uber.org/atomic"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/apiserver/xdsserverv3/resource"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/service"
)

// XdsResourceGenerator is the xDS resource generator
type XdsResourceGenerator struct {
	namingServer service.DiscoverServer
	cache        cachev3.SnapshotCache
	versionNum   *atomic.Uint64
	xdsNodesMgr  *resource.XDSNodeManager
}

func (x *XdsResourceGenerator) Generate(versionLocal string,
	registryInfo map[string]map[model.ServiceKey]*resource.ServiceInfo) {

	_ = x.buildSidecarXDSCache(versionLocal, registryInfo)
	_ = x.buildGatewayXDSCache(versionLocal, registryInfo)
}

func (x *XdsResourceGenerator) buildSidecarXDSCache(versionLocal string,
	registryInfo map[string]map[model.ServiceKey]*resource.ServiceInfo) error {

	nodes := x.xdsNodesMgr.ListSidecarNodes()
	if len(nodes) == 0 || len(registryInfo) == 0 {
		// 如果没有任何一个 XDS Sidecar Node 客户端，不做任何操作
		log.Info("[XDS][Sidecar][V2] xds nodes or registryInfo is empty", zap.Int("nodes", len(nodes)),
			zap.Int("register", len(registryInfo)))
		return nil
	}

	alreadyMakeCache := map[string]struct{}{}
	for i := range nodes {
		node := nodes[i]
		if node.IsGateway() {
			log.Error("[XDS][Sidecar][V2] run type not sidecar or info is invalid",
				zap.String("node", node.Node.Id))
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
			log.Info("[XDS][Sidecar][V2] service is empty, maybe not update",
				zap.String("namespace", selfNamespace), zap.String("cacheKey", cacheKey))
			continue
		}
		if err := x.makeSidecarSnapshot(cacheKey, node, node.TLSMode, versionLocal, services); err != nil {
			log.Error("[XDS][Sidecar][V2] make snapshot fail", zap.String("cacheKey", cacheKey),
				zap.Error(err))
		}
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
		if !node.IsGateway() {
			log.Error("[XDS][Gateway][V2] run type not gateway or info is invalid",
				zap.String("node", node.Node.Id))
			continue
		}

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

// makeSidecarSnapshot nodeId must be like sideacr~namespace or namespace
func (x *XdsResourceGenerator) makeSidecarSnapshot(cacheKey string, xdsNode *resource.XDSClient,
	tlsMode resource.TLSMode, version string, services map[model.ServiceKey]*resource.ServiceInfo) error {

	// 构建所有 XDS 的 Snapshot Resource
	opt := &resource.BuildOption{
		Services:     services,
		TLSMode:      tlsMode,
		VersionLocal: version,
		Namespace:    xdsNode.GetSelfNamespace(),
	}
	// 构建 endpoints XDS 资源缓存数据
	boundEndpoints, err := x.generateXDSResource(resource.EDS, xdsNode, opt)
	if err != nil {
		return err
	}
	// 构建 cluster XDS 资源缓存数据
	boundClusters, err := x.generateXDSResource(resource.CDS, xdsNode, opt)
	if err != nil {
		return err
	}
	// 构建 route XDS 资源缓存
	boundRouters, err := x.generateXDSResource(resource.RDS, xdsNode, opt)
	if err != nil {
		return err
	}
	// 构建 listener XDS 资源缓存
	boundListeners, err := x.generateXDSResource(resource.LDS, xdsNode, opt)
	if err != nil {
		return err
	}

	resources := make(map[resourcev3.Type][]types.Resource)
	resources[resourcev3.EndpointType] = boundEndpoints
	resources[resourcev3.ClusterType] = boundClusters
	resources[resourcev3.RouteType] = boundRouters
	resources[resourcev3.ListenerType] = boundListeners

	snapshot, err := cachev3.NewSnapshot(version, resources)
	if err != nil {
		log.Error("[XDS][Sidecar][V2] fail to create snapshot", zap.Any("envoy-node", xdsNode.Node.Id),
			zap.String("cacheKey", cacheKey), zap.String("snapshot", string(resource.DumpSnapShotJSON(snapshot))),
			zap.Error(err))
		return err
	}
	if err := snapshot.Consistent(); err != nil {
		log.Error("[XDS][Sidecar][V2] verify snapshot consistent", zap.Any("envoy-node", xdsNode.Node.Id),
			zap.String("cacheKey", cacheKey), zap.Error(err))
		return err
	}

	// 为每个 nodeId 刷写 cache ，推送 xds 更新
	if err := x.cache.SetSnapshot(context.Background(), cacheKey, snapshot); err != nil {
		log.Error("[XDS][Sidecar][V2] upsert snapshot error", zap.Any("envoy-node", xdsNode.Node.Id),
			zap.String("cacheKey", cacheKey), zap.Error(err))
		return err
	}
	log.Info("[XDS][Sidecar][V2] upsert snapshot success", zap.Any("envoy-node", xdsNode.Node.Id),
		zap.String("cacheKey", cacheKey), zap.String("snapshot", string(resource.DumpSnapShotJSON(snapshot))))
	return nil
}

// makeGatewaySnapshot nodeId must be like gateway~namespace
func (x *XdsResourceGenerator) makeGatewaySnapshot(xdsNode *resource.XDSClient, tlsMode resource.TLSMode,
	version string, registryInfo map[string]map[model.ServiceKey]*resource.ServiceInfo) error {

	nodeId := xdsNode.Node.Id

	opt := &resource.BuildOption{
		TLSMode:      tlsMode,
		VersionLocal: version,
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
		log.Error("[XDS][Gateway][V2] upsert snapshot error",
			zap.String("cacheKey", cacheKey), zap.Error(err))
		return err
	}
	log.Info("[XDS][Gateway][V2] upsert snapshot success", zap.String("cacheKey", cacheKey),
		zap.String("snapshot", string(resource.DumpSnapShotJSON(snapshot))))
	return nil
}

func (x *XdsResourceGenerator) generateXDSResource(xdsType resource.XDSType, xdsNode *resource.XDSClient,
	opt *resource.BuildOption) ([]types.Resource, error) {

	var xdsBuilder resource.XDSBuilder
	switch xdsType {
	case resource.CDS:
		xdsBuilder = &CDSBuilder{}
	case resource.EDS:
		xdsBuilder = &EDSBuilder{}
	case resource.LDS:
		xdsBuilder = &LDSBuilder{}
	case resource.RDS:
		xdsBuilder = &RDSBuilder{}
	default:
		return nil, errors.New("unsupport xds build type")
	}

	// 构建 XDS 资源缓存数据
	xdsBuilder.Init(xdsNode, x.namingServer)
	resources, err := xdsBuilder.Generate(opt)
	if err != nil {
		return nil, err
	}
	return resources.([]types.Resource), nil
}
