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

package eventhub

import "github.com/polarismesh/polaris/common/model"

// 事件主题
const (
	// InstanceEventTopic discover event
	InstanceEventTopic = "instance_event"
	// LeaderChangeEventTopic leader change
	LeaderChangeEventTopic = "leader_change_event"
	// ConfigFilePublishTopic config file release publish
	ConfigFilePublishTopic = "configfile_publish"
	// CacheInstanceEventTopic record cache occur instance add/update/del event
	CacheInstanceEventTopic = "cache_instance_event"
	// CacheClientEventTopic record cache occur client add/update/del event
	CacheClientEventTopic = "cache_client_event"
	// CacheNamespaceEventTopic record cache occur namespace add/update/del event
	CacheNamespaceEventTopic = "cache_namespace_event"
	// ClientEventTopic .
	ClientEventTopic = "client_event"
)

// PublishConfigFileEvent 事件对象，包含类型和事件消息
type PublishConfigFileEvent struct {
	Message *model.SimpleConfigFileRelease
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
)

type CacheInstanceEvent struct {
	Instance  *model.Instance
	EventType EventType
}

type CacheClientEvent struct {
	Client    *model.Client
	EventType EventType
}

type CacheNamespaceEvent struct {
	OldItem   *model.Namespace
	Item      *model.Namespace
	EventType EventType
}
