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
	"sync"
	"time"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"go.uber.org/atomic"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/apiserver/xdsserverv3/cache"
	"github.com/polarismesh/polaris/apiserver/xdsserverv3/resource"
	"github.com/polarismesh/polaris/common/metrics"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/plugin"
	"github.com/polarismesh/polaris/service"
)

var (
	ErrorNoSupportXDSType = errors.New("unsupport xds build type")
)

type XDSGenerate func(xdsType resource.XDSType, opt *resource.BuildOption)

// XdsResourceGenerator is the xDS resource generator
type XdsResourceGenerator struct {
	namingServer service.DiscoverServer
	cache        *cache.XDSCache
	versionNum   *atomic.Uint64
	xdsNodesMgr  *resource.XDSNodeManager
}

func (x *XdsResourceGenerator) Generate(versionLocal string,
	needUpdate, needRemove map[string]map[model.ServiceKey]*resource.ServiceInfo) {

	// 如果没有任何一个 XDS Node 接入则不会生成与 Node 有关的 XDS Resource
	if x.xdsNodesMgr.HasEnvoyNodes() {
		// 只构建 Sidecar 特有的 XDS 数据
		_ = x.buildEnvoyXDSCache(needUpdate, needRemove)
	}

	deltaOp := func(runType resource.RunType, infos map[string]map[model.ServiceKey]*resource.ServiceInfo, f XDSGenerate) {
		direction := corev3.TrafficDirection_OUTBOUND
		if runType == resource.RunTypeGateway {
			direction = corev3.TrafficDirection_INBOUND
		}
		// CDS/EDS/VHDS 一起构建
		for namespace, services := range infos {
			opt := &resource.BuildOption{
				RunType:          runType,
				Namespace:        namespace,
				Services:         services,
				TrafficDirection: direction,
				TLSMode:          resource.TLSModeNone,
			}
			f(resource.RDS, opt)
			f(resource.EDS, opt)
			f(resource.VHDS, opt)
			// 默认构建没有设置 TLS 的 CDS 资源
			f(resource.CDS, opt)

			// 构建设置了 TLS Mode == Strict 的 CDS 资源
			opt.TLSMode = resource.TLSModeStrict
			f(resource.CDS, opt)
			// 构建设置了 TLS Mode == Permissive 的 CDS 资源
			opt.TLSMode = resource.TLSModePermissive
			f(resource.CDS, opt)
		}
	}

	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()

		// 处理 Sideacr
		deltaOp(resource.RunTypeSidecar, needUpdate, x.buildAndDeltaUpdate)
		deltaOp(resource.RunTypeSidecar, needRemove, x.buildAndDeltaRemove)
	}()

	go func() {
		defer wg.Done()

		// 处理 Gateway
		deltaOp(resource.RunTypeGateway, needUpdate, x.buildAndDeltaUpdate)
		deltaOp(resource.RunTypeGateway, needRemove, x.buildAndDeltaRemove)
	}()

	wg.Wait()
}

func (x *XdsResourceGenerator) buildAndDeltaRemove(xdsType resource.XDSType, opt *resource.BuildOption) {
	xxds, err := x.generateXDSResource(xdsType, opt)
	if err != nil {
		log.Error("[XDS][Envoy] generate xds resource fail", zap.String("type", xdsType.String()), zap.Error(err))
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

	if err := x.cache.DeltaRemoveResource(cacheKey, typeUrl, cachev3.IndexRawResourcesByName(xxds)); err != nil {
		log.Error("[XDS][Envoy] delta update fail", zap.String("cache-key", cacheKey),
			zap.String("type", xdsType.String()), zap.Error(err))
		return
	}
}

func (x *XdsResourceGenerator) buildAndDeltaUpdate(xdsType resource.XDSType, opt *resource.BuildOption) {
	xxds, err := x.generateXDSResource(xdsType, opt)
	if err != nil {
		log.Error("[XDS][Envoy] generate xds resource fail", zap.String("type", xdsType.String()), zap.Error(err))
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
		log.Error("[XDS][Envoy] delta update fail", zap.String("cache-key", cacheKey),
			zap.String("type", xdsType.String()), zap.Error(err))
		return
	}
}

func (x *XdsResourceGenerator) buildEnvoyXDSCache(needUpdate, needRemove map[string]map[model.ServiceKey]*resource.ServiceInfo) error {
	nodes := x.xdsNodesMgr.ListEnvoyNodes()
	if len(nodes) == 0 || len(needUpdate) == 0 || len(needRemove) == 0 {
		// 如果没有任何一个 XDS Sidecar Node 客户端，不做任何操作
		log.Info("[XDS][Envoy] xds nodes or update/remove info is empty", zap.Int("nodes", len(nodes)),
			zap.Int("need-update", len(needUpdate)), zap.Int("need-remove", len(needRemove)))
		return nil
	}

	deltaOp := func(infos map[string]map[model.ServiceKey]*resource.ServiceInfo, f func(xdsType resource.XDSType, opt *resource.BuildOption)) {
		for i := range nodes {
			node := nodes[i]
			opt := &resource.BuildOption{
				RunType:        node.RunType,
				Client:         node,
				TLSMode:        node.TLSMode,
				Namespace:      node.GetSelfNamespace(),
				Services:       infos[node.GetSelfNamespace()],
				OpenOnDemand:   node.OpenOnDemand,
				OnDemandServer: node.OnDemandServer,
				SelfService: model.ServiceKey{
					Namespace: node.GetSelfNamespace(),
					Name:      node.GetSelfService(),
				},
			}

			opt.TrafficDirection = corev3.TrafficDirection_OUTBOUND
			// 构建 OUTBOUND LDS 资源
			f(resource.LDS, opt)
			// 构建 OUTBOUND RDS 资源
			f(resource.RDS, opt)
			opt.TrafficDirection = corev3.TrafficDirection_INBOUND
			// 构建 INBOUND LDS 资源
			f(resource.LDS, opt)
			// 构建 INBOUND EDS 资源
			f(resource.EDS, opt)
			// 构建 INBOUND RDS 资源
			f(resource.RDS, opt)
		}
	}

	deltaOp(needUpdate, x.buildAndDeltaUpdate)
	deltaOp(needRemove, x.buildAndDeltaRemove)
	return nil
}

func (x *XdsResourceGenerator) generateXDSResource(xdsType resource.XDSType,
	opt *resource.BuildOption) ([]types.Resource, error) {

	// 需要预埋相关 XDS 资源生成时间开销
	start := time.Now()
	defer func() {
		plugin.GetStatis().ReportCallMetrics(metrics.CallMetric{
			Type:     metrics.XDSResourceBuildCallMetric,
			API:      xdsType.String(),
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
		return nil, ErrorNoSupportXDSType
	}

	// 构建 XDS 资源缓存数据
	xdsBuilder.Init(x.namingServer)
	resources, err := xdsBuilder.Generate(opt)
	if err != nil {
		return nil, err
	}
	return resources.([]types.Resource), nil
}
