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

// Listener listener for value changes
type Listener interface {
	// OnCreated callback when cache value created
	OnCreated(value interface{})
	// OnUpdated callback when cache value updated
	OnUpdated(value interface{})
	// OnDeleted callback when cache value deleted
	OnDeleted(value interface{})
	// OnBatchCreated callback when cache value created
	OnBatchCreated(value interface{})
	// OnBatchUpdated callback when cache value updated
	OnBatchUpdated(value interface{})
	// OnBatchDeleted callback when cache value deleted
	OnBatchDeleted(value interface{})
}

// EventType common event type
type EventType int

const (
	// EventCreated value create event
	EventCreated EventType = iota
	// EventUpdated value update event
	EventUpdated
	// EventDeleted value delete event
	EventDeleted
	// EventInstanceReload value instances reload
	EventInstanceReload
	// EventPrincipalRemove value principal batch remove
	EventPrincipalRemove
)

type listenerManager struct {
	listeners []Listener
}

func newListenerManager(listeners []Listener) *listenerManager {
	return &listenerManager{
		listeners: listeners,
	}
}

func (l *listenerManager) onEvent(value interface{}, event EventType) {
	if len(l.listeners) == 0 {
		return
	}
	for _, listener := range l.listeners {
		switch event {
		case EventCreated:
			listener.OnCreated(value)
		case EventUpdated:
			listener.OnUpdated(value)
		case EventDeleted:
			listener.OnDeleted(value)
		case EventInstanceReload:
			listener.OnBatchUpdated(value)
		case EventPrincipalRemove:
			listener.OnBatchDeleted(value)
		}
	}
}
