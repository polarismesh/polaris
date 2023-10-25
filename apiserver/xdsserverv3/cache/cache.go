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

package cache

import (
	"context"
	"errors"
	"fmt"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/stream/v3"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/apiserver/xdsserverv3/resource"
	"github.com/polarismesh/polaris/common/utils"
)

type (
	XDSCache struct {
		hook CacheHook
		// hash is the hashing function for Envoy nodes
		hash cachev3.NodeHash
		// Muxed caches.
		Caches *utils.SyncMap[string, cache.Cache]
	}

	// CacheHook
	CacheHook interface {
		// OnCreateWatch
		OnCreateWatch(request *cachev3.Request, streamState stream.StreamState,
			value chan cachev3.Response)
		// OnCreateDeltaWatch
		OnCreateDeltaWatch(request *cachev3.DeltaRequest, state stream.StreamState,
			value chan cachev3.DeltaResponse)
		// OnFetch
		OnFetch(ctx context.Context, request *cachev3.Request)
	}
)

// NewCache create a XDS SnapshotCache to proxy cachev3.SnapshotCache
func NewCache(hook CacheHook) *XDSCache {
	sc := &XDSCache{
		hook:   hook,
		Caches: utils.NewSyncMap[string, cachev3.Cache](),
	}
	return sc
}

// CreateWatch returns a watch for an xDS request.
func (sc *XDSCache) CreateWatch(request *cachev3.Request, streamState stream.StreamState,
	value chan cachev3.Response) func() {
	if sc.hook != nil {
		sc.hook.OnCreateWatch(request, streamState, value)
	}
	item := sc.loadCache(request)
	if item == nil {
		value <- nil
		return func() {}
	}
	return item.CreateWatch(request, streamState, value)
}

// CreateDeltaWatch returns a watch for a delta xDS request which implements the Simple SnapshotCache.
func (sc *XDSCache) CreateDeltaWatch(request *cachev3.DeltaRequest, state stream.StreamState,
	value chan cachev3.DeltaResponse) func() {
	if sc.hook != nil {
		sc.hook.OnCreateDeltaWatch(request, state, value)
	}
	item := sc.loadCache(request)
	if item == nil {
		value <- nil
		return func() {}
	}
	return item.CreateDeltaWatch(request, state, value)
}

// Fetch implements the cache fetch function.
// Fetch is called on multiple streams, so responding to individual names with the same version works.
func (sc *XDSCache) Fetch(ctx context.Context, request *cachev3.Request) (cachev3.Response, error) {
	return nil, errors.New("not implemented")
}

// DeltaUpdateResource .
func (sc *XDSCache) DeltaUpdateResource(key, typeUrl string, current map[string]types.Resource) error {
	val, _ := sc.Caches.ComputeIfAbsent(key, func(_ string) cachev3.Cache {
		return NewLinearCache(typeUrl)
	})
	linearCache, _ := val.(*LinearCache)
	return linearCache.UpdateResources(current, []string{})
}

func classify(typeUrl string, resources []string, client *resource.XDSClient) []string {
	isAllowNode := false
	_, isAllowTls := allowTlsResource[typeUrl]
	if isAllowNodeFunc, exist := allowEachNodeResource[typeUrl]; exist {
		isAllowNode = isAllowNodeFunc(typeUrl, resources, client)
	}
	first := typeUrl + "~" + client.Node.GetId()
	second := typeUrl + "~" + client.GetSelfNamespace()
	tlsMode, exist := client.Metadata[resource.TLSModeTag]

	// 没有设置 TLS 开关
	if !exist || tlsMode == string(resource.TLSModeNone) {
		if isAllowNode {
			return []string{first, second}
		}
		return []string{second}
	}
	if isAllowNode {
		if isAllowTls {
			return []string{first, second, second + "~" + tlsMode}
		}
		return []string{first, second}
	}
	if isAllowTls {
		return []string{first, second, second + "~" + tlsMode}
	}
	return []string{second}
}

type PredicateNodeResource func(typeUrl string, resources []string, client *resource.XDSClient) bool

var (
	allowEachNodeResource = map[string]PredicateNodeResource{
		resourcev3.ListenerType: func(typeUrl string, resources []string, client *resource.XDSClient) bool {
			return true
		},
		resourcev3.EndpointType: func(typeUrl string, resources []string, client *resource.XDSClient) bool {
			selfSvc := fmt.Sprintf("INBOUND|%s|%s", client.GetSelfNamespace(), client.GetSelfService())
			for i := range resources {
				if resources[i] == selfSvc {
					return true
				}
			}
			return false
		},
		resourcev3.RouteType: func(typeUrl string, resources []string, client *resource.XDSClient) bool {
			for i := range resources {
				if resources[i] == resource.InBoundRouteConfigName {
					return true
				}
			}
			return false
		},
	}
	allowTlsResource = map[string]struct{}{
		resourcev3.ListenerType: {},
		resourcev3.ClusterType:  {},
	}
)

func (sc *XDSCache) loadCache(req interface{}) cachev3.Cache {
	var (
		keys               []string
		typeUrl            string
		client             *resource.XDSClient
		subscribeResources []string
	)
	switch args := req.(type) {
	case *cache.Request:
		client = resource.ParseXDSClient(args.GetNode())
		subscribeResources = args.GetResourceNames()
		keys = classify(args.TypeUrl, subscribeResources, client)
		typeUrl = args.TypeUrl
	case *cache.DeltaRequest:
		client = resource.ParseXDSClient(args.GetNode())
		subscribeResources = args.GetResourceNamesSubscribe()
		keys = classify(args.TypeUrl, subscribeResources, client)
		typeUrl = args.TypeUrl
	default:
		log.Error("[XDS][V3] no support client request type", zap.Any("req", args))
		return nil
	}
	for i := range keys {
		val, ok := sc.Caches.Load(keys[i])
		if ok {
			log.Info("[XDS][V3] load cache to handle client request", zap.Strings("keys", keys),
				zap.String("hit-key", keys[i]), zap.Strings("subscribe", subscribeResources),
				zap.String("type", typeUrl), zap.String("client", client.Node.GetId()))
			return val
		}
	}
	log.Error("[XDS][V3] cache not found to handle client request", zap.String("type", typeUrl),
		zap.String("client", client.Node.GetId()))
	return nil
}
