// Copyright 2018 Envoyproxy Authors
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package cache

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/stream/v3"
	"github.com/polarismesh/polaris/apiserver/xdsserverv3/resource"
	"github.com/polarismesh/polaris/common/utils"
)

// CacheHook
type CacheHook interface {
	// OnCreateWatch
	OnCreateWatch(request *cachev3.Request, streamState stream.StreamState,
		value chan cachev3.Response)
	// OnCreateDeltaWatch
	OnCreateDeltaWatch(request *cachev3.DeltaRequest, state stream.StreamState,
		value chan cachev3.DeltaResponse)
	// OnFetch
	OnFetch(ctx context.Context, request *cachev3.Request)
}

type TypeResources struct {
	// UpsertResources 新增/更新的 XDS resource 信息
	UpsertResources map[string]types.Resource
	// RemoveResources 准备删除的 XDS resource 信息
	RemoveResources map[string]struct{}
}

// UpdateResourcesRequest 更新 XDS 资源请求
type UpdateResourcesRequest struct {
	// Lds LDS 相关的资源
	Lds map[string]map[string]types.Resource
	// Types CDS/EDS/RDS/VHDS 相关的资源
	Types map[resource.XDSType]TypeResources
}

// ResourcesContainer
type ResourcesContainer struct {
	// GlobalVersion 当前整体 typeUrl 下的所有 resource 的全局共用版本，主要是用于非 Delta 场景下的 Watch
	GlobalVersion string
	// Resources name -> resource
	Resources map[string]types.Resource
	// VersionMap holds the current hash map of all resources in the snapshot.
	// This field should remain nil until it is used, at which point should be
	// instantiated by calling ConstructVersionMap().
	// VersionMap is only to be used with delta xDS.
	VersionMap map[string]string
}

func (s *ResourcesContainer) updateGlobalRevision() {
	s.GlobalVersion = utils.NewUUID()
}

// ConstructVersionMap will construct a version map based on the current state of a snapshot
func (s *ResourcesContainer) ConstructVersionMap(modified []string) error {
	if s == nil {
		return fmt.Errorf("missing resource container")
	}

	if s.VersionMap == nil {
		s.VersionMap = map[string]string{}
	}

	if len(modified) == 0 {
		// construct all resource
		for _, res := range s.Resources {
			// Hash our version in here and build the version map.
			marshaledResource, err := cachev3.MarshalResource(res)
			if err != nil {
				return err
			}
			v := cachev3.HashResource(marshaledResource)
			if v == "" {
				return fmt.Errorf("failed to build resource version: %w", err)
			}
			s.VersionMap[cachev3.GetResourceName(res)] = v
		}
		return nil
	}
	for _, name := range modified {
		res, ok := s.Resources[name]
		if !ok {
			continue
		}
		// Hash our version in here and build the version map.
		marshaledResource, err := cachev3.MarshalResource(res)
		if err != nil {
			return err
		}
		v := cachev3.HashResource(marshaledResource)
		if v == "" {
			return fmt.Errorf("failed to build resource version: %w", err)
		}
		s.VersionMap[name] = v
	}
	return nil
}

type ResourceCache struct {
	hook CacheHook
	// watchCount and deltaWatchCount are atomic counters incremented for each watch respectively. They need to
	// be the first fields in the struct to guarantee 64-bit alignment,
	// which is a requirement for atomic operations on 64-bit operands to work on
	// 32-bit machines.
	watchCount      int64
	deltaWatchCount int64
	// ads flag to hold responses until all resources are named
	ads bool
	// ldsResources 记录 Envoy Node LDS 的资源记录信息
	ldsResources map[string]*ResourcesContainer
	// resourcesContainer are cached resources indexed by node IDs
	resourcesContainer map[resource.XDSType]*ResourcesContainer
	// status information for all nodes indexed by node IDs
	status map[string]*statusInfo

	mu sync.RWMutex
}

// NewResourceCache initializes a simple sc.
//
// ADS flag forces a delay in responding to streaming requests until all
// resources are explicitly named in the request. This avoids the problem of a
// partial request over a single stream for a subset of resources which would
// require generating a fresh version for acknowledgement. ADS flag requires
// snapshot consistency. For non-ADS case (and fetch), multiple partial
// requests are sent across multiple streams and re-using the snapshot version
// is OK.
//
// Logger is optional.
func NewResourceCache(hook CacheHook) *ResourceCache {
	cache := &ResourceCache{
		hook:               hook,
		ads:                true,
		ldsResources:       make(map[string]*ResourcesContainer),
		resourcesContainer: make(map[resource.XDSType]*ResourcesContainer),
		status:             make(map[string]*statusInfo),
	}
	return cache
}

// CleanEnvoyNodeCache 清理 Envoy Node 强关联的 XDS 规则数据
func (sc *ResourceCache) CleanEnvoyNodeCache(node *corev3.Node) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	delete(sc.ldsResources, node.GetId())
	return nil
}

// UpdateResources updates a snapshot for a node.
func (sc *ResourceCache) UpdateResources(ctx context.Context, req *UpdateResourcesRequest) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// 更新 LDS 资源信息
	for nodeId, resources := range req.Lds {
		if _, ok := sc.ldsResources[nodeId]; !ok {
			sc.ldsResources[nodeId] = &ResourcesContainer{
				Resources: resources,
			}
			sc.ldsResources[nodeId].updateGlobalRevision()
			sc.ldsResources[nodeId].ConstructVersionMap(nil)
		}
	}

	// 更新非 LDS 的 XDS 规则到 ResourceContainer 中
	for typeUrl, resources := range req.Types {
		if container, ok := sc.resourcesContainer[typeUrl]; !ok {
			container = &ResourcesContainer{
				Resources: resources.UpsertResources,
			}
			sc.resourcesContainer[typeUrl] = container
		} else {
			for name := range resources.RemoveResources {
				delete(container.Resources, name)
				delete(container.VersionMap, name)
			}
			modified := make([]string, 0, len(resources.UpsertResources))
			for name, res := range resources.UpsertResources {
				container.Resources[name] = res
				modified = append(modified, name)
			}
		}
		sc.resourcesContainer[typeUrl].updateGlobalRevision()
	}

	for _, info := range sc.status {
		info.mu.Lock()
		defer info.mu.Unlock()
		for id, watch := range info.watches {
			watchType := resource.FormatTypeUrl(watch.Request.TypeUrl)
			var container *ResourcesContainer
			switch watchType {
			case resource.LDS:
				// 获取到 Envoy Node 对应希望看到的 ldsRes 资源
				container = sc.ldsResources[info.node.Id]
			default:
				container = sc.resourcesContainer[watchType]
			}
			curVersion := container.GlobalVersion
			if curVersion != watch.Request.VersionInfo {
				log.Debugf("respond open watch %d %s%v with new version %q", id, watch.Request.TypeUrl,
					watch.Request.ResourceNames, curVersion)

				resources := container.Resources
				if err := sc.respond(ctx, watch.Request, watch.Response, resources, curVersion, false); err != nil {
					return err
				}
				// discard the watch
				delete(info.watches, id)
			}
		}

		// We only calculate version hashes when using delta. We don't
		// want to do this when using SOTW so we can avoid unnecessary
		// computational cost if not using delta.
		if len(info.deltaWatches) > 0 {
			for _, container := range sc.resourcesContainer {
				if err := container.ConstructVersionMap(nil); err != nil {
					log.Errorf("failed to compute version for snapshot resources inline: %s", err)
					return err
				}
			}
		}

		// process our delta watches
		for id, watch := range info.deltaWatches {
			watchType := resource.FormatTypeUrl(watch.Request.TypeUrl)
			var container *ResourcesContainer
			switch watchType {
			case resource.LDS:
				// 获取到 Envoy Node 对应希望看到的 ldsRes 资源
				container = sc.ldsResources[info.node.Id]
			default:
				container = sc.resourcesContainer[watchType]
			}
			res, err := sc.respondDelta(
				ctx,
				container,
				watch.Request,
				watch.Response,
				watch.StreamState,
			)
			if err != nil {
				return err
			}
			// If we detect a nil response here, that means there has been no state change
			// so we don't want to respond or remove any existing resource watches
			if res != nil {
				delete(info.deltaWatches, id)
			}
		}
	}
	return nil
}

// nameSet creates a map from a string slice to value true.
func nameSet(names []string) map[string]bool {
	set := make(map[string]bool, len(names))
	for _, name := range names {
		set[name] = true
	}
	return set
}

// superset checks that all resources are listed in the names set.
func superset(names map[string]bool, resources map[string]types.Resource) error {
	for resourceName := range resources {
		if _, exists := names[resourceName]; !exists {
			return fmt.Errorf("%q not listed", resourceName)
		}
	}
	return nil
}

// CreateWatch returns a watch for an xDS request.  A nil function may be
// returned if an error occurs.
func (sc *ResourceCache) CreateWatch(request *cachev3.Request, streamState stream.StreamState, value chan cachev3.Response) func() {
	if sc.hook != nil {
		sc.hook.OnCreateWatch(request, streamState, value)
	}

	nodeID := request.Node.GetId()

	sc.mu.Lock()
	defer sc.mu.Unlock()

	info, ok := sc.status[nodeID]
	if !ok {
		info = newStatusInfo(request.Node)
		sc.status[nodeID] = info
	}

	// update last watch request time
	info.mu.Lock()
	info.lastWatchRequestTime = time.Now()
	info.mu.Unlock()

	watchType := resource.FormatTypeUrl(request.GetTypeUrl())

	var container *ResourcesContainer
	var version string
	var exists bool

	switch watchType {
	case resource.LDS:
		// 获取到 Envoy Node 对应希望看到的 ldsRes 资源
		container, exists = sc.ldsResources[info.node.Id]
	default:
		container, exists = sc.resourcesContainer[watchType]
	}
	if exists {
		version = container.GlobalVersion
	}

	if exists {
		knownResourceNames := streamState.GetKnownResourceNames(request.TypeUrl)
		diff := []string{}
		for _, r := range request.ResourceNames {
			if _, ok := knownResourceNames[r]; !ok {
				diff = append(diff, r)
			}
		}

		log.Debugf("nodeID %q requested %s%v and known %v. Diff %v", nodeID,
			request.TypeUrl, request.ResourceNames, knownResourceNames, diff)

		if len(diff) > 0 {
			resources := container.Resources
			for _, name := range diff {
				if _, exists := resources[name]; exists {
					if err := sc.respond(context.Background(), request, value, resources, version, false); err != nil {
						log.Errorf("failed to send a response for %s%v to nodeID %q: %s", request.TypeUrl,
							request.ResourceNames, nodeID, err)
						return nil
					}
					return func() {}
				}
			}
		}
	}

	// if the requested version is up-to-date or missing a response, leave an open watch
	if !exists || request.VersionInfo == version {
		watchID := sc.nextWatchID()
		log.Debugf("open watch %d for %s%v from nodeID %q, version %q", watchID, request.TypeUrl, request.ResourceNames, nodeID, request.VersionInfo)
		info.mu.Lock()
		info.watches[watchID] = cachev3.ResponseWatch{Request: request, Response: value}
		info.mu.Unlock()
		return sc.cancelWatch(nodeID, watchID)
	}

	// otherwise, the watch may be responded immediately
	resources := container.Resources
	if err := sc.respond(context.Background(), request, value, resources, version, false); err != nil {
		log.Errorf("failed to send a response for %s%v to nodeID %q: %s", request.TypeUrl,
			request.ResourceNames, nodeID, err)
		return nil
	}

	return func() {}
}

func (sc *ResourceCache) nextWatchID() int64 {
	return atomic.AddInt64(&sc.watchCount, 1)
}

// cancellation function for cleaning stale watches
func (sc *ResourceCache) cancelWatch(nodeID string, watchID int64) func() {
	return func() {
		// uses the cache mutex
		sc.mu.RLock()
		defer sc.mu.RUnlock()
		if info, ok := sc.status[nodeID]; ok {
			info.mu.Lock()
			delete(info.watches, watchID)
			info.mu.Unlock()
		}
	}
}

// Respond to a watch with the snapshot value. The value channel should have capacity not to block.
// TODO(kuat) do not respond always, see issue https://github.com/envoyproxy/go-control-plane/issues/46
func (sc *ResourceCache) respond(ctx context.Context, request *cachev3.Request,
	value chan cachev3.Response,
	resources map[string]types.Resource,
	version string,
	heartbeat bool,
) error {

	// for ADS, the request names must match the snapshot names
	// if they do not, then the watch is never responded, and it is expected that envoy makes another request
	if len(request.ResourceNames) != 0 && sc.ads {
		if err := superset(nameSet(request.ResourceNames), resources); err != nil {
			log.Warnf("ADS mode: not responding to request: %v", err)
			return nil
		}
	}

	log.Debugf("respond %s%v version %q with version %q", request.TypeUrl, request.ResourceNames, request.VersionInfo, version)

	select {
	case value <- createResponse(ctx, request, resources, version, heartbeat):
		return nil
	case <-ctx.Done():
		return context.Canceled
	}
}

func createResponse(ctx context.Context, request *cachev3.Request, resources map[string]types.Resource,
	version string, heartbeat bool) cachev3.Response {
	filtered := make([]types.ResourceWithTTL, 0, len(resources))

	// Reply only with the requested resources. Envoy may ask each resource
	// individually in a separate stream. It is ok to reply with the same version
	// on separate streams since requests do not share their response versions.
	if len(request.ResourceNames) != 0 {
		set := nameSet(request.ResourceNames)
		for name, resource := range resources {
			if set[name] {
				filtered = append(filtered, types.ResourceWithTTL{Resource: resource})
			}
		}
	} else {
		for _, resource := range resources {
			filtered = append(filtered, types.ResourceWithTTL{Resource: resource})
		}
	}

	return &cachev3.RawResponse{
		Request:   request,
		Version:   version,
		Resources: filtered,
		Heartbeat: heartbeat,
		Ctx:       ctx,
	}
}

// CreateDeltaWatch returns a watch for a delta xDS request which implements the Simple Snapshotsc.
func (sc *ResourceCache) CreateDeltaWatch(request *cachev3.DeltaRequest, state stream.StreamState,
	value chan cachev3.DeltaResponse) func() {

	if sc.hook != nil {
		sc.hook.OnCreateDeltaWatch(request, state, value)
	}

	nodeID := request.Node.GetId()
	t := request.GetTypeUrl()

	sc.mu.Lock()
	defer sc.mu.Unlock()

	info, ok := sc.status[nodeID]
	if !ok {
		info = newStatusInfo(request.Node)
		sc.status[nodeID] = info
	}

	// update last watch request time
	info.setLastDeltaWatchRequestTime(time.Now())

	watchType := resource.FormatTypeUrl(request.GetTypeUrl())

	var container *ResourcesContainer
	var exists bool

	switch watchType {
	case resource.LDS:
		// 获取到 Envoy Node 对应希望看到的 ldsRes 资源
		container, exists = sc.ldsResources[info.node.Id]
	default:
		container, exists = sc.resourcesContainer[watchType]
	}

	// There are three different cases that leads to a delayed watch trigger:
	// - no snapshot exists for the requested nodeID
	// - a snapshot exists, but we failed to initialize its version map
	// - we attempted to issue a response, but the caller is already up to date
	delayedResponse := !exists
	if exists {
		if err := container.ConstructVersionMap(nil); err != nil {
			log.Errorf("failed to compute version for snapshot resources inline: %s", err)
		}
		response, err := sc.respondDelta(context.Background(), container, request, value, state)
		if err != nil {
			log.Errorf("failed to respond with delta response: %s", err)
		}

		delayedResponse = response == nil
	}

	if delayedResponse {
		watchID := sc.nextDeltaWatchID()

		if exists {
			log.Infof("open delta watch ID:%d for %s Resources:%v from nodeID: %q,  version %q", watchID, t,
				state.GetSubscribedResourceNames(), nodeID, container.GlobalVersion)
		} else {
			log.Infof("open delta watch ID:%d for %s Resources:%v from nodeID: %q", watchID, t,
				state.GetSubscribedResourceNames(), nodeID)
		}

		info.setDeltaResponseWatch(watchID, cachev3.DeltaResponseWatch{
			Request:     request,
			Response:    value,
			StreamState: state,
		})
		return sc.cancelDeltaWatch(nodeID, watchID)
	}

	return nil
}

// Respond to a delta watch with the provided snapshot value. If the response is nil, there has been no state change.
func (sc *ResourceCache) respondDelta(ctx context.Context, container *ResourcesContainer,
	request *cachev3.DeltaRequest,
	value chan cachev3.DeltaResponse,
	state stream.StreamState,
) (*cachev3.RawDeltaResponse, error) {

	resp := createDeltaResponse(ctx, request, state, resourceContainer{
		resourceMap:   container.Resources,
		versionMap:    container.VersionMap,
		systemVersion: container.GlobalVersion,
	})

	// Only send a response if there were changes
	// We want to respond immediately for the first wildcard request in a stream, even if the response is empty
	// otherwise, envoy won't complete initialization
	if len(resp.Resources) > 0 || len(resp.RemovedResources) > 0 || (state.IsWildcard() && state.IsFirst()) {
		if log != nil {
			log.Debugf("node: %s, sending delta response for typeURL %s with resources: "+
				" %v removed resources: %v with wildcard: %t",
				request.GetNode().GetId(), request.TypeUrl, cachev3.GetResourceNames(resp.Resources),
				resp.RemovedResources, state.IsWildcard())
		}
		select {
		case value <- resp:
			return resp, nil
		case <-ctx.Done():
			return resp, context.Canceled
		}
	}
	return nil, nil
}

func (sc *ResourceCache) nextDeltaWatchID() int64 {
	return atomic.AddInt64(&sc.deltaWatchCount, 1)
}

// cancellation function for cleaning stale delta watches
func (sc *ResourceCache) cancelDeltaWatch(nodeID string, watchID int64) func() {
	return func() {
		sc.mu.RLock()
		defer sc.mu.RUnlock()
		if info, ok := sc.status[nodeID]; ok {
			info.mu.Lock()
			delete(info.deltaWatches, watchID)
			info.mu.Unlock()
		}
	}
}

// Fetch implements the cache fetch function.
// Fetch is called on multiple streams, so responding to individual names with the same version works.
func (sc *ResourceCache) Fetch(ctx context.Context, request *cachev3.Request) (cachev3.Response, error) {

	nodeID := request.Node.GetId()

	sc.mu.RLock()
	defer sc.mu.RUnlock()

	if container, exists := sc.resourcesContainer[resource.FormatTypeUrl(request.GetTypeUrl())]; exists {
		// Respond only if the request version is distinct from the current snapshot state.
		// It might be beneficial to hold the request since Envoy will re-attempt the refresh.
		version := container.GlobalVersion
		if request.VersionInfo == version {
			log.Warnf("skip fetch: version up to date")
			return nil, &types.SkipFetchError{}
		}

		out := createResponse(ctx, request, container.Resources, version, false)
		return out, nil
	}

	return nil, fmt.Errorf("missing snapshot for %q", nodeID)
}

// GetStatusInfo retrieves the status info for the node.
func (sc *ResourceCache) GetStatusInfo(node string) cachev3.StatusInfo {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	info, exists := sc.status[node]
	if !exists {
		log.Warnf("node does not exist")
		return nil
	}

	return info
}

// GetStatusKeys retrieves all node IDs in the status map.
func (sc *ResourceCache) GetStatusKeys() []string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	out := make([]string, 0, len(sc.status))
	for id := range sc.status {
		out = append(out, id)
	}

	return out
}
