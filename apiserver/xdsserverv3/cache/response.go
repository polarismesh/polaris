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

	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/stream/v3"
)

func createDeltaResponse(ctx context.Context, req *cachev3.DeltaRequest, state stream.StreamState,
	resources resourceContainer) *cachev3.RawDeltaResponse {
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
		for name, r := range resources.nodeResourceMap {
			// Since we've already precomputed the version hashes of the new snapshot,
			// we can just set it here to be used for comparison later
			version := resources.nodeVersionMap[name]
			nextVersionMap[name] = version
			prevVersion, found := state.GetResourceVersions()[name]
			if !found || (prevVersion != version) {
				filtered = append(filtered, r)
			}
		}

		for name, r := range resources.resourceMap {
			if _, exist := nextVersionMap[name]; exist {
				continue
			}
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
			_, commonOk := resources.resourceMap[name]
			_, nodeOk := resources.nodeResourceMap[name]
			if !commonOk && !nodeOk {
				toRemove = append(toRemove, name)
			}
		}
	default:
		nextVersionMap = make(map[string]string, len(state.GetSubscribedResourceNames()))
		// state.GetResourceVersions() may include resources no longer subscribed
		// In the current code this gets silently cleaned when updating the version map
		for name := range state.GetSubscribedResourceNames() {
			prevVersion, found := state.GetResourceVersions()[name]
			var commonOk, nodeOk bool
			if r, ok := resources.nodeResourceMap[name]; ok {
				nodeOk = true
				nextVersion := resources.nodeVersionMap[name]
				if prevVersion != nextVersion {
					filtered = append(filtered, r)
				}
				nextVersionMap[name] = nextVersion
			}
			if r, ok := resources.resourceMap[name]; ok {
				if _, exist := nextVersionMap[name]; !exist {
					commonOk = true
					nextVersion := resources.versionMap[name]
					if prevVersion != nextVersion {
						filtered = append(filtered, r)
					}
					nextVersionMap[name] = nextVersion
				}
			}
			if found && (!commonOk && !nodeOk) {
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

type NoReadyXdsResponse struct {
	cachev3.DeltaResponse
}

func (r *NoReadyXdsResponse) GetDeltaRequest() *discovery.DeltaDiscoveryRequest {
	return nil
}

func (r *NoReadyXdsResponse) GetDeltaDiscoveryResponse() (*discovery.DeltaDiscoveryResponse, error) {
	return nil, errors.New("node xds not created yet")
}
