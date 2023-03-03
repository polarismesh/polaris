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

package service

import (
	"context"
	"time"

	"github.com/polarismesh/polaris/common/model"
)

type serviceNameResolver func(string) *model.Service

const maxRetryGetServiceName = 5

type BaseInstanceEventHandler struct {
	namingServer DiscoverServer
	svcResolver  serviceNameResolver
}

func NewBaseInstanceEventHandler(namingServer DiscoverServer) *BaseInstanceEventHandler {
	eventHandler := &BaseInstanceEventHandler{namingServer: namingServer}
	eventHandler.svcResolver = eventHandler.resolveService
	return eventHandler
}

func (b *BaseInstanceEventHandler) resolveService(svcId string) *model.Service {
	return b.namingServer.Cache().Service().GetServiceByID(svcId)
}

// PreProcess do preprocess logic for event
func (b *BaseInstanceEventHandler) PreProcess(ctx context.Context, value any) any {
	instEvent, ok := value.(model.InstanceEvent)
	if !ok {
		return value
	}
	b.resolveServiceName(&instEvent)
	return instEvent
}

func (b *BaseInstanceEventHandler) resolveServiceName(event *model.InstanceEvent) {
	if len(event.Service) == 0 && len(event.SvcId) > 0 {
		for i := 0; i < maxRetryGetServiceName; i++ {
			svcObject := b.svcResolver(event.SvcId)
			if nil == svcObject {
				time.Sleep(500 * time.Millisecond)
				continue
			}
			event.Service = svcObject.Name
			event.Namespace = svcObject.Namespace
			break
		}
	}
}
