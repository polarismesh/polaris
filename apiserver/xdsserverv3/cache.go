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

	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/stream/v3"
)

type (
	snapshotCache struct {
		hook CacheHook
		// hash is the hashing function for Envoy nodes
		hash     cachev3.NodeHash
		xdsCache cachev3.SnapshotCache
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

// NewSnapshotCache create a XDS SnapshotCache to proxy cachev3.SnapshotCache
func NewSnapshotCache(xdsCache cachev3.SnapshotCache, hook CacheHook) cachev3.SnapshotCache {
	return newSnapshotCache(xdsCache, hook)
}

func newSnapshotCache(xdsCache cachev3.SnapshotCache, hook CacheHook) *snapshotCache {
	cache := &snapshotCache{
		hook:     hook,
		xdsCache: xdsCache,
	}
	return cache
}

// SetSnapshotCacheContext updates a snapshot for a node.
func (cache *snapshotCache) SetSnapshot(ctx context.Context, node string,
	snapshot cachev3.ResourceSnapshot) error {
	return cache.xdsCache.SetSnapshot(ctx, node, snapshot)
}

// GetSnapshots gets the snapshot for a node, and returns an error if not found.
func (cache *snapshotCache) GetSnapshot(node string) (cachev3.ResourceSnapshot, error) {
	return cache.xdsCache.GetSnapshot(node)
}

// ClearSnapshot clears snapshot and info for a node.
func (cache *snapshotCache) ClearSnapshot(node string) {
	cache.xdsCache.ClearSnapshot(node)
}

// CreateWatch returns a watch for an xDS request.
func (cache *snapshotCache) CreateWatch(request *cachev3.Request, streamState stream.StreamState,
	value chan cachev3.Response) func() {
	if cache.hook != nil {
		cache.hook.OnCreateWatch(request, streamState, value)
	}
	return cache.xdsCache.CreateWatch(request, streamState, value)
}

// CreateDeltaWatch returns a watch for a delta xDS request which implements the Simple SnapshotCache.
func (cache *snapshotCache) CreateDeltaWatch(request *cachev3.DeltaRequest, state stream.StreamState,
	value chan cachev3.DeltaResponse) func() {
	if cache.hook != nil {
		cache.hook.OnCreateDeltaWatch(request, state, value)
	}
	return cache.xdsCache.CreateDeltaWatch(request, state, value)
}

// Fetch implements the cache fetch function.
// Fetch is called on multiple streams, so responding to individual names with the same version works.
func (cache *snapshotCache) Fetch(ctx context.Context, request *cachev3.Request) (cachev3.Response, error) {
	if cache.hook != nil {
		cache.hook.OnFetch(ctx, request)
	}
	return cache.xdsCache.Fetch(ctx, request)
}

// GetStatusInfo retrieves the status info for the node.
func (cache *snapshotCache) GetStatusInfo(node string) cachev3.StatusInfo {
	return cache.xdsCache.GetStatusInfo(node)
}

// GetStatusKeys retrieves all node IDs in the status map.
func (cache *snapshotCache) GetStatusKeys() []string {
	return cache.xdsCache.GetStatusKeys()
}
