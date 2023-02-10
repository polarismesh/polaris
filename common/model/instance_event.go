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

package model

import (
	"context"
	"fmt"
	"time"

	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
)

// InstanceEventType 探测事件类型
type InstanceEventType string

const (
	// EventDiscoverNone empty discover event
	EventDiscoverNone InstanceEventType = "EventDiscoverNone"
	// EventInstanceOnline instance becoming online
	EventInstanceOnline InstanceEventType = "InstanceOnline"
	// EventInstanceTurnUnHealth Instance becomes unhealthy
	EventInstanceTurnUnHealth InstanceEventType = "InstanceTurnUnHealth"
	// EventInstanceTurnHealth Instance becomes healthy
	EventInstanceTurnHealth InstanceEventType = "InstanceTurnHealth"
	// EventInstanceOpenIsolate Instance is in isolation
	EventInstanceOpenIsolate InstanceEventType = "InstanceOpenIsolate"
	// EventInstanceCloseIsolate Instance shutdown isolation state
	EventInstanceCloseIsolate InstanceEventType = "InstanceCloseIsolate"
	// EventInstanceOffline Instance offline
	EventInstanceOffline InstanceEventType = "InstanceOffline"
	// EventInstanceSendHeartbeat Instance send heartbeat package to server
	EventInstanceSendHeartbeat InstanceEventType = "InstanceSendHeartbeat"
)

// CtxEventKeyMetadata 用于将metadata从Context中传入并取出
const CtxEventKeyMetadata = "ctx_event_metadata"

// InstanceEvent 服务实例事件
type InstanceEvent struct {
	Id         string
	Namespace  string
	Service    string
	Instance   *apiservice.Instance
	EType      InstanceEventType
	CreateTime time.Time
	MetaData   map[string]string
}

// InjectMetadata 从context中获取metadata并注入到事件对象
func (i *InstanceEvent) InjectMetadata(ctx context.Context) {
	value := ctx.Value(CtxEventKeyMetadata)
	if nil == value {
		return
	}
	i.MetaData = value.(map[string]string)
}

func (i *InstanceEvent) String() string {
	if nil == i {
		return "nil"
	}
	hostPortStr := fmt.Sprintf("%s:%d", i.Instance.GetHost().GetValue(), i.Instance.GetPort().GetValue())
	return fmt.Sprintf("InstanceEvent(id=%s, namespace=%s, service=%s, type=%v, instance=%s, healthy=%v)",
		i.Id, i.Namespace, i.Service, i.EType, hostPortStr, i.Instance.GetHealthy().GetValue())
}
