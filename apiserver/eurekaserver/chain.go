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

package eurekaserver

import (
	"context"

	"go.uber.org/zap"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/service"
	"github.com/polarismesh/polaris/store"
)

type (
	sourceFromEureka struct{}
)

func (h *EurekaServer) registerInstanceChain() {
	svr := h.originDiscoverSvr.(*service.Server)
	svr.AddInstanceChain(&EurekaInstanceChain{
		s: svr.Store(),
	})
}

type EurekaInstanceChain struct {
	s store.Store
}

func (c *EurekaInstanceChain) AfterUpdate(ctx context.Context, instances ...*model.Instance) {
	isFromEureka, _ := ctx.Value(sourceFromEureka{}).(bool)
	if isFromEureka {
		return
	}

	// TODO：这里要注意避免 eureka -> polaris -> notify -> eureka 带来的重复操作，后续会在 context 中携带信息做判断处理
	for i := range instances {
		ins := instances[i]
		metadata := ins.Proto.GetMetadata()
		if _, ok := metadata[InternalMetadataStatus]; !ok {
			continue
		}
		if ins.Isolate() {
			metadata[InternalMetadataStatus] = StatusOutOfService
		} else {
			metadata[InternalMetadataStatus] = StatusUp
		}
		if err := c.s.BatchAppendInstanceMetadata([]*store.InstanceMetadataRequest{
			{
				InstanceID: ins.ID(),
				Revision:   utils.NewUUID(),
				Metadata: map[string]string{
					InternalMetadataStatus: metadata[InternalMetadataStatus],
				},
			},
		}); err != nil {
			eurekalog.Error("[EUREKA-SERVER] after update instance isolate fail", zap.Error(err))
		}
	}
}
