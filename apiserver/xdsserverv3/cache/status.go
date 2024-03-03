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
	"sync"
	"time"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"

	"github.com/polarismesh/polaris/apiserver/xdsserverv3/resource"
)

// statusInfo tracks the server state for the remote Envoy node.
type statusInfo struct {
	// node is the constant Envoy node metadata.
	client *resource.XDSClient
	// watches are indexed channels for the response watches and the original requests.
	watches map[int64]cachev3.ResponseWatch
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

// newStatusInfo initializes a status info data structure.
func newStatusInfo(node *core.Node) *statusInfo {
	out := statusInfo{
		client:       resource.ParseXDSClient(node),
		watches:      make(map[int64]cachev3.ResponseWatch),
		deltaWatches: make(map[int64]cachev3.DeltaResponseWatch),
	}
	return &out
}

func (info *statusInfo) GetNode() *core.Node {
	info.mu.RLock()
	defer info.mu.RUnlock()
	return info.GetNode()
}

func (info *statusInfo) GetNumWatches() int {
	info.mu.RLock()
	defer info.mu.RUnlock()
	return len(info.watches)
}

func (info *statusInfo) GetNumDeltaWatches() int {
	info.mu.RLock()
	defer info.mu.RUnlock()
	return len(info.deltaWatches)
}

func (info *statusInfo) GetLastWatchRequestTime() time.Time {
	info.mu.RLock()
	defer info.mu.RUnlock()
	return info.lastWatchRequestTime
}

func (info *statusInfo) GetLastDeltaWatchRequestTime() time.Time {
	info.mu.RLock()
	defer info.mu.RUnlock()
	return info.lastDeltaWatchRequestTime
}

// setLastDeltaWatchRequestTime will set the current time of the last delta discovery watch request.
func (info *statusInfo) setLastDeltaWatchRequestTime(t time.Time) {
	info.mu.Lock()
	defer info.mu.Unlock()
	info.lastDeltaWatchRequestTime = t
}

// setDeltaResponseWatch will set the provided delta response watch for the associated watch ID.
func (info *statusInfo) setDeltaResponseWatch(id int64, drw cachev3.DeltaResponseWatch) {
	info.mu.Lock()
	defer info.mu.Unlock()
	info.deltaWatches[id] = drw
}
