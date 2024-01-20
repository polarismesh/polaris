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

	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/stream/v3"
)

// OnCreateWatch before call cachev3.SnapshotCache CreateWatch
func (x *XDSServer) OnCreateWatch(request *cachev3.Request, streamState stream.StreamState,
	value chan cachev3.Response) {
	x.activeUpdateTask()

	client := x.nodeMgr.GetNode(request.GetNode().Id)
	if client == nil {
		return
	}
	_ = x.resourceGenerator.buildOneEnvoyXDSCache(client)
}

// OnCreateDeltaWatch before call cachev3.SnapshotCache OnCreateDeltaWatch
func (x *XDSServer) OnCreateDeltaWatch(request *cachev3.DeltaRequest, state stream.StreamState,
	value chan cachev3.DeltaResponse) {
	x.activeUpdateTask()
	client := x.nodeMgr.GetNode(request.GetNode().Id)
	if client == nil {
		return
	}
	_ = x.resourceGenerator.buildOneEnvoyXDSCache(client)
}

// OnFetch before call cachev3.SnapshotCache OnFetch
func (x *XDSServer) OnFetch(ctx context.Context, request *cachev3.Request) {
	x.activeUpdateTask()
	client := x.nodeMgr.GetNode(request.GetNode().Id)
	if client == nil {
		return
	}
	_ = x.resourceGenerator.buildOneEnvoyXDSCache(client)
}
