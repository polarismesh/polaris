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

package xdsserverv3

import (
	"context"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	resource "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
)

// makeGatewaySnapshot nodeId must be like gateway~namespace
func (x *XDSServer) makeGatewaySnapshot(nodeId, version string, services []*ServiceInfo) (err error) {
	resources := make(map[resource.Type][]types.Resource)
	resources[resource.EndpointType] = makeEndpoints(services)
	resources[resource.ClusterType] = x.makeClusters(services)
	resources[resource.RouteType] = x.; (services)
	resources[resource.ListenerType] = makeListeners()
	snapshot, err := cachev3.NewSnapshot(version, resources)
	if err != nil {
		log.Errorf("fail to create snapshot for %s, err is %v", nodeId, err)
		return err
	}
	err = snapshot.Consistent()
	if err != nil {
		return err
	}
	log.Infof("will serve ns: %s ,snapshot: %+v", nodeId, string(dumpSnapShotJSON(snapshot)))
	// 为每个 nodeId 刷写 cache ，推送 xds 更新
	if err := x.cache.SetSnapshot(context.Background(), nodeId, snapshot); err != nil {
		log.Errorf("snapshot error %q for %+v", err, snapshot)
		return err
	}
	return
}
