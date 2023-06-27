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

package v1

import (
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"go.uber.org/atomic"

	"github.com/polarismesh/polaris/apiserver/xdsserverv3/resource"
	"github.com/polarismesh/polaris/service"
)

type options func(svr *XDSServer)

func WithDiscoverServer(discoverSvr service.DiscoverServer) options {
	return func(svr *XDSServer) {
		svr.namingServer = discoverSvr
	}
}

func WithSnapshot(cache cachev3.SnapshotCache) options {
	return func(svr *XDSServer) {
		svr.cache = cache
	}
}

func WithVersion(versionNum *atomic.Uint64) options {
	return func(svr *XDSServer) {
		svr.versionNum = versionNum
	}
}

func WithXDSNodeMgr(nodeMgr *resource.XDSNodeManager) options {
	return func(svr *XDSServer) {
		svr.xdsNodesMgr = nodeMgr
	}
}
