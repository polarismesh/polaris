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

type (
	ServiceInfos               map[string]map[model.ServiceKey]*resource.ServiceInfo
	CurrentServiceInfoProvider func() ServiceInfos
)

// XdsResourceGenerator is the xDS resource generator
type XdsResourceGenerator struct {
	namingServer    service.DiscoverServer
	cache           *cache.ResourceCache
	versionNum      *atomic.Uint64
	xdsNodesMgr     *resource.XDSNodeManager
	svcInfoProvider CurrentServiceInfoProvider
}

// Generate 构建 XDS 资源缓存数据信息
func (x *XdsResourceGenerator) Generate(versionLocal string, needUpdate, needRemove ServiceInfos) {
	updateRequest := cache.NewUpdateResourcesRequest()

	deltaOp := func(runType resource.RunType, infos ServiceInfos, isRemove bool) {
		direction := corev3.TrafficDirection_OUTBOUND
		if runType == resource.RunTypeGateway {
			direction = corev3.TrafficDirection_INBOUND
		}
		generate := func(opt *resource.BuildOption) {
			opt.CloseEnvoyDemand()
			opt.TLSMode = resource.TLSModeNone
			// 默认构建没有设置 TLS 的 CDS 资源
			x.buildUpdateRequest(updateRequest, resource.CDS, opt, isRemove)
			// 构建设置了 TLS Mode == Strict 的 CDS 资源
			opt.TLSMode = resource.TLSModeStrict
			x.buildUpdateRequest(updateRequest, resource.CDS, opt, isRemove)
			// 构建设置了 TLS Mode == Permissive 的 CDS 资源
			opt.TLSMode = resource.TLSModePermissive
			x.buildUpdateRequest(updateRequest, resource.CDS, opt, isRemove)
			// 恢复 TLSMode
			opt.TLSMode = resource.TLSModeNone
			x.buildUpdateRequest(updateRequest, resource.EDS, opt, isRemove)
			x.buildUpdateRequest(updateRequest, resource.RDS, opt, isRemove)
			// 开启按需 Demand
			opt.OpenEnvoyDemand()
			x.buildUpdateRequest(updateRequest, resource.RDS, opt, isRemove)
		}

		// CDS/EDS/VHDS 一起构建
		for namespace, services := range infos {
			opt := &resource.BuildOption{
				RunType:          runType,
				Namespace:        namespace,
				Services:         services,
				TrafficDirection: direction,
			}
			// sidecar 和 gateway 大部份资源都是复用的，所以这里只需要构建一次即可，gateway 只有 RDS/LDS 存在特别，单独针对构建即可
			if runType == resource.RunTypeSidecar {
				generate(opt)
				x.buildUpdateRequest(updateRequest, resource.VHDS, opt, isRemove)
			}

			if runType == resource.RunTypeSidecar {
				for svcKey := range services {
					// 换成 INBOUND 构建 CDS、EDS、RDS
					opt.SelfService = svcKey
					opt.TrafficDirection = corev3.TrafficDirection_INBOUND
					generate(opt)
				}
			}
		}
	}

	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		// 处理 Sideacr
		deltaOp(resource.RunTypeSidecar, needUpdate, false)
		deltaOp(resource.RunTypeSidecar, needRemove, true)
	}()

	go func() {
		defer wg.Done()
		// 处理 Gateway
		deltaOp(resource.RunTypeGateway, needUpdate, false)
		deltaOp(resource.RunTypeGateway, needRemove, true)
	}()

	wg.Wait()

	if err := x.cache.UpdateResources(context.Background(), updateRequest); err != nil {
		log.Error("[XDS][Envoy] update xds resource fail", zap.Error(err))
	}
}

func (x *XdsResourceGenerator) buildOneEnvoyXDSCache(node *resource.XDSClient) error {
	opt := &resource.BuildOption{
		RunType:   node.RunType,
		Client:    node,
		TLSMode:   node.TLSMode,
		Namespace: node.GetSelfNamespace(),
		SelfService: model.ServiceKey{
			Namespace: node.GetSelfNamespace(),
			Name:      node.GetSelfService(),
		},
	}
	if node.OpenOnDemand {
		opt.OpenEnvoyDemand()
	}

	finalResources := make([]types.Resource, 0, 4)
	buildCache := func(xdsType resource.XDSType, opt *resource.BuildOption) {
		xxds, err := x.generateXDSResource(xdsType, opt)
		if err != nil {
			log.Error("[XDS][Envoy] generate envoy node resource fail", zap.Error(err))
			return
		}
		finalResources = append(finalResources, xxds...)
	}

	opt.TrafficDirection = corev3.TrafficDirection_OUTBOUND
	// 构建 OUTBOUND LDS 资源
	buildCache(resource.LDS, opt)
	opt.TrafficDirection = corev3.TrafficDirection_INBOUND
	// 构建 INBOUND LDS 资源
	buildCache(resource.LDS, opt)

	return x.cache.UpdateResources(context.Background(), &cache.UpdateResourcesRequest{
		Lds: map[string]map[string]types.Resource{
			opt.Client.ID: cachev3.IndexRawResourcesByName(finalResources),
		},
	})
}

func (x *XdsResourceGenerator) buildUpdateRequest(req *cache.UpdateResourcesRequest, xdsType resource.XDSType,
	opt *resource.BuildOption, isRemove bool) {

	opt.ForceDelete = isRemove
	xxds, err := x.generateXDSResource(xdsType, opt)
	if err != nil {
		log.Error("[XDS][Envoy] generate xds resource fail", zap.Error(err))
		return
	}

	switch opt.TLSMode {
	case resource.TLSModeNone:
		if opt.ForceDelete {
			req.RemoveNormalNamespaces(opt.Namespace, opt.TLSMode, xdsType, xxds)
		} else {
			req.AddNormalNamespaces(opt.Namespace, xdsType, xxds)
		}
	default:
		if opt.ForceDelete {
			req.RemoveTlsNamespaces(opt.Namespace, opt.TLSMode, xdsType, xxds)
		} else {
			req.AddTlsNamespaces(opt.Namespace, opt.TLSMode, xdsType, xxds)
		}
	}
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
