// Copyright 2020 Envoyproxy Authors
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
	"errors"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/stream/v3"

	"github.com/polarismesh/polaris/apiserver/xdsserverv3/resource"
)

type watches = map[chan cachev3.Response]struct{}

// LinearCache supports collections of opaque resources. This cache has a
// single collection indexed by resource names and manages resource versions
// internally. It implements the cache interface for a single type URL and
// should be combined with other caches via type URL muxing. It can be used to
// supply EDS entries, for example, uniformly across a fleet of proxies.
type LinearCache struct {
	// Type URL specific to the cache.
	typeURL string
	// --- XDS Watcher ---
	// Watchers collection by node id
	watchClients map[string]*watchClient
	// Continuously incremented counter used to index delta watches.
	deltaWatchCount int64
	// --- common xds resource for target type_url ---
	// Collection of resources indexed by name.
	resources map[string]types.Resource
	// versionMap holds the current hash map of all resources in the cache when delta watches are present.
	// versionMap is only to be used with delta xDS.
	versionMap map[string]string
	// Versions for each resource by name.
	versionVector map[string]uint64
	// --- node xds resource for target type_url ---
	// Collection of resource indexed by node-id -> name
	nodeResources map[string]map[string]types.Resource
	// nodeVersionMap holds the current hash map of all resources in the cache when delta watches are present.
	// nodeVersionMap is only to be used with delta xDS.
	nodeVersionMap map[string]map[string]string
	// nodeVersionVector for each resource by name.
	nodeVersionVector map[string]map[string]uint64

	versionPrefix string

	// Continuously incremented version.
	version uint64
	// Continuously incremented version for per envoy node.
	nodeVersion map[string]uint64
	mu          sync.RWMutex
}

var _ cachev3.Cache = &LinearCache{}

// NewLinearCache creates a new cache. See the comments on the struct definition.
func NewLinearCache(typeURL string) *LinearCache {
	out := &LinearCache{
		typeURL:       typeURL,
		resources:     make(map[string]types.Resource),
		watchClients:  make(map[string]*watchClient),
		versionMap:    nil,
		version:       0,
		versionVector: make(map[string]uint64),
	}
	return out
}

func (cache *LinearCache) respond(value chan<- cachev3.Response, staleResources []string) {
	var resources []types.ResourceWithTTL
	// TODO: optimize the resources slice creations across different clients
	if len(staleResources) == 0 {
		resources = make([]types.ResourceWithTTL, 0, len(cache.resources))
		for _, resource := range cache.resources {
			resources = append(resources, types.ResourceWithTTL{Resource: resource})
		}
	} else {
		resources = make([]types.ResourceWithTTL, 0, len(staleResources))
		for _, name := range staleResources {
			resource := cache.resources[name]
			if resource != nil {
				resources = append(resources, types.ResourceWithTTL{Resource: resource})
			}
		}
	}
	value <- &cachev3.RawResponse{
		Request:   &cachev3.Request{TypeUrl: cache.typeURL},
		Resources: resources,
		// Version:   cache.getVersion(),
		Ctx: context.Background(),
	}
}

func (cache *LinearCache) notifyAll(watchClients map[string]*watchClient, modified map[string]struct{}) {
	// de-duplicate watches that need to be responded
	notifyList := make(map[chan<- cachev3.Response][]string)
	for name := range modified {
		for _, client := range watchClients {
			watchChs := client.popResourceWatchChans(name)
			for _, ch := range watchChs {
				notifyList[ch] = append(notifyList[ch], name)
			}
		}
	}
	for value, stale := range notifyList {
		cache.respond(value, stale)
	}
	for _, client := range watchClients {
		if !client.hasWatchAll() {
			continue
		}
		client.foreachWatchAll(func(resp chan<- cachev3.Response) {
			cache.respond(resp, nil)
		})
		client.cleanWatchAll()
	}

	// Building the version map has a very high cost when using SetResources to do full updates.
	// As it is only used with delta watches, it is only maintained when applicable.
	if cache.versionMap != nil {
		if err := cache.updateVersionMap(modified); err != nil {
			log.Errorf("failed to update version map: %v", err)
		}

		for _, client := range watchClients {
			watchs, ok := client.listDeltaWatchs(modified)
			if !ok {
				continue
			}
			for id, watch := range watchs {
				if res := cache.respondDelta(watch.Request, watch.Response, watch.StreamState); res != nil {
					cache.cancelDeltaWatch(client.client.GetNodeID(), id)()
				}
			}
		}
	}
}

func (cache *LinearCache) notifyClient(node *resource.XDSClient, modified map[string]struct{}) {
	watchC, ok := cache.watchClients[node.GetNodeID()]
	if !ok {
		return
	}
	waitNotifier := map[string]*watchClient{
		node.GetNodeID(): watchC,
	}
	cache.notifyAll(waitNotifier, modified)
}

func (cache *LinearCache) respondDelta(request *cachev3.DeltaRequest, value chan cachev3.DeltaResponse, state stream.StreamState) *cachev3.RawDeltaResponse {
	resp := createDeltaResponse(context.Background(), request, state, resourceContainer{
		resourceMap:     cache.resources,
		nodeResourceMap: cache.nodeResources[request.Node.Id],
		versionMap:      cache.versionMap,
		nodeVersionMap:  cache.nodeVersionMap[request.Node.Id],
		systemVersion:   cache.getVersion(request.Node.Id),
	})

	// Only send a response if there were changes
	if len(resp.Resources) > 0 || len(resp.RemovedResources) > 0 {
		log.Infof("[linear cache] node: %s, sending delta response for typeURL %s with resources: %v removed resources: %v with wildcard: %t",
			request.GetNode().GetId(), request.TypeUrl, cachev3.GetResourceNames(resp.Resources), resp.RemovedResources, state.IsWildcard())
		value <- resp
		return resp
	}
	return nil
}

// DeleteNodeResources clean target node link all resource in the collection.
func (cache *LinearCache) DeleteNodeResources(client *resource.XDSClient) error {
	if client == nil {
		return errors.New("nil resource")
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()

	delete(cache.watchClients, client.GetNodeID())
	delete(cache.nodeVersion, client.GetNodeID())
	delete(cache.nodeResources, client.GetNodeID())
	delete(cache.nodeVersionMap, client.GetNodeID())
	delete(cache.nodeVersionVector, client.GetNodeID())
	return nil
}

// UpdateResource updates a resource in the collection.
func (cache *LinearCache) UpdateNodeResource(client *resource.XDSClient, name string, res types.Resource) error {
	if res == nil {
		return errors.New("nil resource")
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()

	if _, ok := cache.nodeVersion[client.GetNodeID()]; !ok {
		cache.nodeVersion[client.GetNodeID()] = 0
	}
	cache.nodeVersion[client.GetNodeID()] = cache.nodeVersion[client.GetNodeID()] + 1

	if _, ok := cache.nodeVersionVector[client.GetNodeID()]; !ok {
		cache.nodeVersionVector[client.GetNodeID()] = map[string]uint64{}
	}
	cache.nodeVersionVector[client.GetNodeID()][name] = cache.nodeVersion[client.GetNodeID()]

	if _, ok := cache.nodeResources[client.GetNodeID()]; !ok {
		cache.nodeResources[client.GetNodeID()] = map[string]types.Resource{}
	}
	cache.nodeResources[client.GetNodeID()][name] = res

	// TODO: batch watch closures to prevent rapid updates
	cache.notifyClient(client, map[string]struct{}{name: {}})
	return nil
}

// DeleteResource removes a resource in the collection.
func (cache *LinearCache) DeleteResource(name string) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.version++
	delete(cache.versionVector, name)
	delete(cache.resources, name)

	// TODO: batch watch closures to prevent rapid updates
	cache.notifyAll(cache.watchClients, map[string]struct{}{name: {}})
	return nil
}

// UpdateResources updates/deletes a list of resources in the cache.
// Calling UpdateResources instead of iterating on UpdateResource and DeleteResource
// is significantly more efficient when using delta or wildcard watches.
func (cache *LinearCache) UpdateResources(toUpdate map[string]types.Resource, toDelete []string) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.version++

	modified := make(map[string]struct{}, len(toUpdate)+len(toDelete))
	for name, resource := range toUpdate {
		cache.versionVector[name] = cache.version
		cache.resources[name] = resource
		modified[name] = struct{}{}
	}
	for _, name := range toDelete {
		delete(cache.versionVector, name)
		delete(cache.resources, name)
		modified[name] = struct{}{}
	}

	cache.notifyAll(cache.watchClients, modified)
	return nil
}

// SetResources replaces current resources with a new set of resources.
// This function is useful for wildcard xDS subscriptions.
// This way watches that are subscribed to all resources are triggered only once regardless of how many resources are changed.
func (cache *LinearCache) SetResources(resources map[string]types.Resource) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.version++

	modified := map[string]struct{}{}
	// Collect deleted resource names.
	for name := range cache.resources {
		if _, found := resources[name]; !found {
			delete(cache.versionVector, name)
			modified[name] = struct{}{}
		}
	}

	cache.resources = resources

	// Collect changed resource names.
	// We assume all resources passed to SetResources are changed.
	// Otherwise we would have to do proto.Equal on resources which is pretty expensive operation
	for name := range resources {
		cache.versionVector[name] = cache.version
		modified[name] = struct{}{}
	}

	cache.notifyAll(cache.watchClients, modified)
}

// GetResources returns current resources stored in the cache
func (cache *LinearCache) GetResources() map[string]types.Resource {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	// create a copy of our internal storage to avoid data races
	// involving mutations of our backing map
	resources := make(map[string]types.Resource, len(cache.resources))
	for k, v := range cache.resources {
		resources[k] = v
	}
	return resources
}

func parseReceiveVersion(version string) (string, string) {
	ret := strings.Split(version, "~")
	resVer, nodeVer := ret[0], ret[1]
	return resVer, nodeVer
}

func (cache *LinearCache) CreateWatch(request *cachev3.Request, _ stream.StreamState, value chan cachev3.Response) func() {
	if request.TypeUrl != cache.typeURL {
		value <- nil
		return nil
	}

	nodeId := request.Node.Id
	watchClient, ok := cache.watchClients[nodeId]
	if !ok {
		watchClient = newWatchClient(request.Node)
		cache.watchClients[nodeId] = watchClient
	}

	// If the version is not up to date, check whether any requested resource has
	// been updated between the last version and the current version. This avoids the problem
	// of sending empty updates whenever an irrelevant resource changes.
	stale := false
	staleResources := []string{} // empty means all

	// strip version prefix if it is present
	var (
		lastVersion, lastNodeVersion uint64
		err                          error
	)
	if strings.HasPrefix(request.VersionInfo, cache.versionPrefix) {
		var lastVerErr, lastNodeVerErr error
		resVer, nodeVer := parseReceiveVersion(request.VersionInfo[len(cache.versionPrefix):])
		lastVersion, lastVerErr = strconv.ParseUint(resVer, 0, 64)
		lastNodeVersion, lastNodeVerErr = strconv.ParseUint(nodeVer, 0, 64)
		err = errors.Join(lastVerErr, lastNodeVerErr)
	} else {
		err = errors.New("mis-matched version prefix")
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	if err != nil {
		stale = true
		staleResources = request.ResourceNames
	} else if len(request.ResourceNames) == 0 {
		stale = lastVersion != cache.version
	} else {
		for _, name := range request.ResourceNames {
			if saveVer, ok := cache.versionVector[name]; ok {
				// When a resource is removed, its version defaults 0 and it is not considered stale.
				if lastVersion < saveVer {
					stale = true
					staleResources = append(staleResources, name)
				}
			}
			if saveVer, ok := cache.nodeVersionVector[nodeId][name]; ok {
				// When a resource is removed, its version defaults 0 and it is not considered stale.
				if lastNodeVersion < saveVer {
					stale = true
					staleResources = append(staleResources, name)
				}
			}
		}
	}
	if stale {
		cache.respond(value, staleResources)
		return nil
	}
	// Create open watches since versions are up to date.
	if len(request.ResourceNames) == 0 {
		watchClient.setWatchAll(value)
		return func() {
			watchClient.removeWatchAll(value)
		}
	}
	watchClient.addResourcesWatch(request.GetResourceNames(), value)
	return func() {
		watchClient.removeResourcesWatch(request.GetResourceNames(), value)
	}
}

func (cache *LinearCache) CreateDeltaWatch(request *cachev3.DeltaRequest, state stream.StreamState, value chan cachev3.DeltaResponse) func() {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	nodeId := request.Node.Id
	watchClient, ok := cache.watchClients[nodeId]
	if !ok {
		watchClient = newWatchClient(request.Node)
		cache.watchClients[nodeId] = watchClient
	}
	// update last watch request time
	watchClient.setLastDeltaWatchRequestTime(time.Now())

	if cache.versionMap == nil {
		// If we had no previously open delta watches, we need to build the version map for the first time.
		// The version map will not be destroyed when the last delta watch is removed.
		// This avoids constantly rebuilding when only a few delta watches are open.
		modified := map[string]struct{}{}
		for name := range cache.resources {
			modified[name] = struct{}{}
		}
		err := cache.updateVersionMap(modified)
		if err != nil {
			log.Errorf("failed to update version map: %v", err)
		}
	}
	response := cache.respondDelta(request, value, state)

	// if respondDelta returns nil this means that there is no change in any resource version
	// create a new watch accordingly
	if response == nil {
		watchID := cache.nextDeltaWatchID()
		log.Infof("[linear cache] open delta watch ID:%d for %s Resources:%v, system version %q", watchID,
			cache.typeURL, state.GetSubscribedResourceNames(), cache.getVersion(nodeId))

		watchClient.setDeltaResponseWatch(watchID, cachev3.DeltaResponseWatch{Request: request, Response: value, StreamState: state})
		return cache.cancelDeltaWatch(nodeId, watchID)
	}
	return nil
}

func (cache *LinearCache) updateVersionMap(modified map[string]struct{}) error {
	if cache.versionMap == nil {
		cache.versionMap = make(map[string]string, len(modified))
	}
	for name := range modified {
		r, ok := cache.resources[name]
		if !ok {
			// The resource was deleted
			delete(cache.versionMap, name)
			continue
		}
		// hash our version in here and build the version map
		marshaledResource, err := cachev3.MarshalResource(r)
		if err != nil {
			return err
		}
		v := cachev3.HashResource(marshaledResource)
		if v == "" {
			return errors.New("failed to build resource version")
		}

		cache.versionMap[name] = v
	}
	return nil
}

func (cache *LinearCache) getVersion(nodeId string) string {
	return cache.versionPrefix + strconv.FormatUint(cache.version, 10) + "~" + strconv.FormatUint(cache.nodeVersion[nodeId], 10)
}

// cancellation function for cleaning stale watches
func (cache *LinearCache) cancelDeltaWatch(nodeID string, watchID int64) func() {
	return func() {
		cache.mu.RLock()
		defer cache.mu.RUnlock()
		if info, ok := cache.watchClients[nodeID]; ok {
			info.mu.Lock()
			delete(info.deltaWatches, watchID)
			info.mu.Unlock()
		}
	}
}

func (cache *LinearCache) nextDeltaWatchID() int64 {
	return atomic.AddInt64(&cache.deltaWatchCount, 1)
}

func (cache *LinearCache) Fetch(context.Context, *cachev3.Request) (cachev3.Response, error) {
	return nil, errors.New("not implemented")
}

// Number of resources currently on the cache.
// As GetResources is building a clone it is expensive to get metrics otherwise.
func (cache *LinearCache) NumResources() int {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return len(cache.resources)
}

// groups together resource-related arguments for the createDeltaResponse function
type resourceContainer struct {
	resourceMap     map[string]types.Resource
	nodeResourceMap map[string]types.Resource
	versionMap      map[string]string
	nodeVersionMap  map[string]string
	systemVersion   string
}

func createDeltaResponse(ctx context.Context, req *cachev3.DeltaRequest, state stream.StreamState, resources resourceContainer) *cachev3.RawDeltaResponse {
	// variables to build our response with
	var nextVersionMap map[string]string
	var filtered []types.Resource
	var toRemove []string

	// If we are handling a wildcard request, we want to respond with all resources
	switch {
	case state.IsWildcard():
		if len(state.GetResourceVersions()) == 0 {
			filtered = make([]types.Resource, 0, len(resources.resourceMap))
		}
		nextVersionMap = make(map[string]string, len(resources.resourceMap))
		for name, r := range resources.resourceMap {
			// Since we've already precomputed the version hashes of the new snapshot,
			// we can just set it here to be used for comparison later
			version := resources.versionMap[name]
			nextVersionMap[name] = version
			prevVersion, found := state.GetResourceVersions()[name]
			if !found || (prevVersion != version) {
				filtered = append(filtered, r)
			}
		}

		// Compute resources for removal
		// The resource version can be set to "" here to trigger a removal even if never returned before
		for name := range state.GetResourceVersions() {
			if _, ok := resources.resourceMap[name]; !ok {
				toRemove = append(toRemove, name)
			}
		}
	default:
		nextVersionMap = make(map[string]string, len(state.GetSubscribedResourceNames()))
		// state.GetResourceVersions() may include resources no longer subscribed
		// In the current code this gets silently cleaned when updating the version map
		for name := range state.GetSubscribedResourceNames() {
			prevVersion, found := state.GetResourceVersions()[name]
			if r, ok := resources.resourceMap[name]; ok {
				nextVersion := resources.versionMap[name]
				if prevVersion != nextVersion {
					filtered = append(filtered, r)
				}
				nextVersionMap[name] = nextVersion
			} else if found {
				toRemove = append(toRemove, name)
			}
		}
	}

	return &cachev3.RawDeltaResponse{
		DeltaRequest:      req,
		Resources:         filtered,
		RemovedResources:  toRemove,
		NextVersionMap:    nextVersionMap,
		SystemVersionInfo: resources.systemVersion,
		Ctx:               ctx,
	}
}

// newWatchClient initializes a status info data structure.
func newWatchClient(node *corev3.Node) *watchClient {
	out := watchClient{
		watchAll:     map[chan<- cachev3.Response]struct{}{},
		client:       resource.ParseXDSClient(node),
		watches:      make(map[string]map[chan<- cachev3.Response]struct{}),
		deltaWatches: make(map[int64]cachev3.DeltaResponseWatch),
	}
	return &out
}

// watchClient tracks the server state for the remote Envoy node.
type watchClient struct {
	watchAll map[chan<- cachev3.Response]struct{}
	// node is the constant Envoy node metadata.
	client *resource.XDSClient
	// watches are indexed channels for the response watches and the original requests.
	watches map[string]map[chan<- cachev3.Response]struct{}
	// deltaWatches are indexed channels for the delta response watches and the original requests
	deltaWatches map[int64]cachev3.DeltaResponseWatch
	// the timestamp of the last watch request
	lastWatchRequestTime time.Time
	// the timestamp of the last delta watch request
	lastDeltaWatchRequestTime time.Time
	// mutex to protect the status fields.
	// should not acquire mutex of the parent cache after acquiring this mutex.
	mu sync.RWMutex
}

func (info *watchClient) GetNumWatches() int {
	info.mu.RLock()
	defer info.mu.RUnlock()
	return len(info.watches)
}

func (info *watchClient) GetNumDeltaWatches() int {
	info.mu.RLock()
	defer info.mu.RUnlock()
	return len(info.deltaWatches)
}

func (info *watchClient) GetLastWatchRequestTime() time.Time {
	info.mu.RLock()
	defer info.mu.RUnlock()
	return info.lastWatchRequestTime
}

func (info *watchClient) GetLastDeltaWatchRequestTime() time.Time {
	info.mu.RLock()
	defer info.mu.RUnlock()
	return info.lastDeltaWatchRequestTime
}

// setLastDeltaWatchRequestTime will set the current time of the last delta discovery watch request.
func (info *watchClient) setLastDeltaWatchRequestTime(t time.Time) {
	info.mu.Lock()
	defer info.mu.Unlock()
	info.lastDeltaWatchRequestTime = t
}

// setDeltaResponseWatch will set the provided delta response watch for the associated watch ID.
func (info *watchClient) setDeltaResponseWatch(id int64, drw cachev3.DeltaResponseWatch) {
	info.mu.Lock()
	defer info.mu.Unlock()
	info.deltaWatches[id] = drw
}

func (info *watchClient) listDeltaWatchs(modified map[string]struct{}) (map[int64]cachev3.DeltaResponseWatch, bool) {
	return nil, true
}

// setDeltaResponseWatch will set the provided delta response watch for the associated watch ID.
func (info *watchClient) setWatchAll(respCh chan<- cachev3.Response) {
	info.mu.Lock()
	defer info.mu.Unlock()
	info.watchAll[respCh] = struct{}{}
}

func (info *watchClient) removeWatchAll(respCh chan<- cachev3.Response) {
	info.mu.Lock()
	defer info.mu.Unlock()
	delete(info.watchAll, respCh)
}

func (info *watchClient) hasWatchAll() bool {
	info.mu.RLock()
	defer info.mu.RUnlock()
	return len(info.watchAll) > 0
}

func (info *watchClient) foreachWatchAll(consumer func(resp chan<- cachev3.Response)) {
	info.mu.RLock()
	defer info.mu.RUnlock()
	for v := range info.watchAll {
		consumer(v)
	}
}

func (info *watchClient) cleanWatchAll() {
	info.mu.Lock()
	defer info.mu.Unlock()
	info.watchAll = make(map[chan<- cachev3.Response]struct{})
}

func (info *watchClient) addResourcesWatch(resources []string, resp chan<- cachev3.Response) {
	info.mu.Lock()
	defer info.mu.Unlock()

	for _, res := range resources {
		if _, ok := info.watches[res]; !ok {
			info.watches[res] = make(map[chan<- cachev3.Response]struct{})
		}
		info.watches[res][resp] = struct{}{}
	}
}

func (info *watchClient) popResourceWatchChans(resource string) []chan<- cachev3.Response {
	return nil
}

func (info *watchClient) removeResourcesWatch(resources []string, resp chan<- cachev3.Response) {
	info.mu.Lock()
	defer info.mu.Unlock()

	for _, res := range resources {
		if _, ok := info.watches[res]; !ok {
			continue
		}
		delete(info.watches[res], resp)
		if len(info.watches[res]) == 0 {
			delete(info.watches, res)
		}
	}
}
