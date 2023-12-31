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

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
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
		Caches *utils.SyncMap[string, cachev3.Cache]
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

// CleanEnvoyNodeCache 清理和 Envoy Node 强相关的缓存数据
func (sc *XDSCache) CleanEnvoyNodeCache(node *corev3.Node) {
	cacheKey := resource.LDS.ResourceType() + "~" + node.Id
	sc.Caches.Delete(cacheKey)
}

// CreateWatch returns a watch for an xDS request.
func (sc *XDSCache) CreateWatch(request *cachev3.Request, streamState stream.StreamState,
	value chan cachev3.Response) func() {
	if sc.hook != nil {
		sc.hook.OnCreateWatch(request, streamState, value)
	}
	item := sc.loadCache(request, streamState)
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
	item := sc.loadCache(request, state)
	if item == nil {
		value <- &NoReadyXdsResponse{}
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
		return cachev3.NewLinearCache(typeUrl, cachev3.WithLogger(log))
	})
	linearCache, _ := val.(*cachev3.LinearCache)
	return linearCache.UpdateResources(current, []string{})
}

// DeltaRemoveResource .
func (sc *XDSCache) DeltaRemoveResource(key, typeUrl string, current map[string]types.Resource) error {
	val, _ := sc.Caches.ComputeIfAbsent(key, func(_ string) cachev3.Cache {
		return cachev3.NewLinearCache(typeUrl, cachev3.WithLogger(log))
	})
	linearCache, _ := val.(*cachev3.LinearCache)

	waitRemove := make([]string, 0, len(current))
	for k := range current {
		waitRemove = append(waitRemove, k)
	}
	return linearCache.UpdateResources(nil, waitRemove)
}

func (sc *XDSCache) loadCache(req interface{}, streamState stream.StreamState) cachev3.Cache {
	var (
		typeUrl string
		client  *resource.XDSClient
	)
	switch args := req.(type) {
	case *cachev3.Request:
		client = resource.ParseXDSClient(args.GetNode())
		typeUrl = args.TypeUrl
	case *cachev3.DeltaRequest:
		client = resource.ParseXDSClient(args.GetNode())
		typeUrl = args.TypeUrl
	default:
		log.Error("[XDS][V3] no support client request type", zap.Any("req", args))
		return nil
	}
	cacheKey := BuildCacheKey(typeUrl, client.TLSMode, client)
	val, ok := sc.Caches.Load(cacheKey)
	if ok {
		log.Info("[XDS][V3] load cache to handle client request",
			zap.String("cache-key", cacheKey), zap.String("type", typeUrl),
			zap.String("client", client.Node.GetId()), zap.Bool("wildcard", streamState.IsWildcard()))
		return val
	}
	log.Error("[XDS][V3] cache not found to handle client request", zap.String("type", typeUrl),
		zap.String("client", client.Node.GetId()))
	return nil
}

func BuildCacheKey(typeUrl string, tlsMode resource.TLSMode, client *resource.XDSClient) string {
	xdsType := resource.FormatTypeUrl(typeUrl)
	if xdsType == resource.LDS {
		return typeUrl + "~" + client.GetNodeID()
	}
	key := typeUrl + "~" + client.GetSelfNamespace()
	if resource.SupportTLS(xdsType) && resource.EnableTLS(tlsMode) {
		key = key + "~" + string(tlsMode)
	}
	return key
}
