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
	"go.uber.org/atomic"

	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"

	"github.com/polarismesh/polaris/apiserver/xdsserverv3/resource"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/service"
)

func New(opt ...options) *XDSServer {
	svr := &XDSServer{}
	for i := range opt {
		opt[i](svr)
	}
	return svr
}

// XDSServer is the xDS server
type XDSServer struct {
	namingServer service.DiscoverServer
	cache        cachev3.SnapshotCache
	versionNum   *atomic.Uint64
	xdsNodesMgr  *resource.XDSNodeManager
}

func (x *XDSServer) Generate(versionLocal string,
	registryInfo map[string]map[model.ServiceKey]*resource.ServiceInfo) {
	for ns, services := range registryInfo {
		_ = x.makeSnapshot(ns, versionLocal, resource.TLSModeNone, services)
		_ = x.makeSnapshot(ns, versionLocal, resource.TLSModePermissive, services)
		_ = x.makeSnapshot(ns, versionLocal, resource.TLSModeStrict, services)
	}
	return
}
