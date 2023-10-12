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
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/stream/v3"
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
	item := sc.loadOrStore(request)
	return item.CreateWatch(request, streamState, value)
}

// CreateDeltaWatch returns a watch for a delta xDS request which implements the Simple SnapshotCache.
func (sc *XDSCache) CreateDeltaWatch(request *cachev3.DeltaRequest, state stream.StreamState,
	value chan cachev3.DeltaResponse) func() {
	if sc.hook != nil {
		sc.hook.OnCreateDeltaWatch(request, state, value)
	}
	item := sc.loadOrStore(request)
	return item.CreateDeltaWatch(request, state, value)
}

// Fetch implements the cache fetch function.
// Fetch is called on multiple streams, so responding to individual names with the same version works.
func (sc *XDSCache) Fetch(ctx context.Context, request *cachev3.Request) (cachev3.Response, error) {
	return nil, errors.New("not implemented")
}

// DeltaUpdateResource .
func (sc *XDSCache) DeltaUpdateResource(typeUrl string, current map[string]types.Resource) error {
	key := typeUrl
	val, _ := sc.Caches.ComputeIfAbsent(key, func(_ string) cachev3.Cache {
		return cache.NewLinearCache(typeUrl, cache.WithLogger(log))
	})
	linearCache, _ := val.(*cache.LinearCache)
	return linearCache.UpdateResources(current, []string{})
}

func classify(typeUrl string, node *corev3.Node) string {
	tlsMode, exist := resource.GetEnvoyMetaField(node.GetMetadata(), resource.TLSModeTag, "")
	if !exist || tlsMode == string(resource.TLSModeNone) {
		return typeUrl
	}
	return typeUrl + "~" + tlsMode
}

func (sc *XDSCache) loadOrStore(req interface{}) cachev3.Cache {
	var (
		key     string
		typeUrl string
	)
	switch args := req.(type) {
	case *cache.Request:
		key = classify(args.TypeUrl, args.GetNode())
		typeUrl = args.TypeUrl
	case *cache.DeltaRequest:
		key = classify(args.TypeUrl, args.GetNode())
		typeUrl = args.TypeUrl
	}
	val, _ := sc.Caches.ComputeIfAbsent(key, func(_ string) cachev3.Cache {
		return cache.NewLinearCache(typeUrl, cache.WithLogger(log))
	})
	return val
}
