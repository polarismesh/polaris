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

package resource

import (
	"context"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	commonlog "github.com/polarismesh/polaris/common/log"
)

func NewCallback(log *commonlog.Scope, nodeMgr *XDSNodeManager) *Callbacks {
	return &Callbacks{
		log:     log,
		nodeMgr: nodeMgr,
	}
}

type Callbacks struct {
	log     *commonlog.Scope
	nodeMgr *XDSNodeManager
}

func (cb *Callbacks) Report() {

}

func (cb *Callbacks) OnStreamOpen(_ context.Context, id int64, typ string) error {
	return nil
}

func (cb *Callbacks) OnStreamClosed(id int64, node *corev3.Node) {
	cb.nodeMgr.DelNode(id)
}

func (cb *Callbacks) OnDeltaStreamOpen(_ context.Context, id int64, typ string) error {
	return nil
}

func (cb *Callbacks) OnDeltaStreamClosed(id int64, node *corev3.Node) {
}

func (cb *Callbacks) OnStreamRequest(id int64, req *discovery.DiscoveryRequest) error {
	cb.nodeMgr.AddNodeIfAbsent(id, req.GetNode())
	return nil
}

func (cb *Callbacks) OnStreamResponse(_ context.Context, id int64, req *discovery.DiscoveryRequest,
	resp *discovery.DiscoveryResponse) {
}

func (cb *Callbacks) OnStreamDeltaRequest(id int64, req *discovery.DeltaDiscoveryRequest) error {
	cb.nodeMgr.AddNodeIfAbsent(id, req.GetNode())
	return nil
}

func (cb *Callbacks) OnStreamDeltaResponse(id int64, req *discovery.DeltaDiscoveryRequest,
	resp *discovery.DeltaDiscoveryResponse) {
}

func (cb *Callbacks) OnFetchRequest(_ context.Context, req *discovery.DiscoveryRequest) error {
	return nil
}

func (cb *Callbacks) OnFetchResponse(req *discovery.DiscoveryRequest, resp *discovery.DiscoveryResponse) {
}
