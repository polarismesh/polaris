package cache

import (
	"sync"
	"time"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
)

// statusInfo tracks the server state for the remote Envoy node.
type statusInfo struct {
	// node is the constant Envoy node metadata.
	node *core.Node
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
		node:         node,
		watches:      make(map[int64]cachev3.ResponseWatch),
		deltaWatches: make(map[int64]cachev3.DeltaResponseWatch),
	}
	return &out
}

func (info *statusInfo) GetNode() *core.Node {
	info.mu.RLock()
	defer info.mu.RUnlock()
	return info.node
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
