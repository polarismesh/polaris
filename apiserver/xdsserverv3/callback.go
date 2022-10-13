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

	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/gogo/protobuf/jsonpb"

	commonlog "github.com/polarismesh/polaris/common/log"
)

type Callbacks struct {
	log *commonlog.Scope
}

func (cb *Callbacks) Report() {

}

func (cb *Callbacks) OnStreamOpen(_ context.Context, id int64, typ string) error {
	if cb.log.DebugEnabled() {
		cb.log.Debugf("stream %d open for %s", id, typ)
	}
	return nil
}

func (cb *Callbacks) OnStreamClosed(id int64) {
	if cb.log.DebugEnabled() {
		cb.log.Debugf("stream %d closed", id)
	}
}

func (cb *Callbacks) OnDeltaStreamOpen(_ context.Context, id int64, typ string) error {
	if cb.log.DebugEnabled() {
		cb.log.Debugf("delta stream %d open for %s", id, typ)
	}
	return nil
}

func (cb *Callbacks) OnDeltaStreamClosed(id int64) {
	if cb.log.DebugEnabled() {
		cb.log.Debugf("delta stream %d closed", id)
	}
}

func (cb *Callbacks) OnStreamRequest(id int64, req *discovery.DiscoveryRequest) error {
	if cb.log.DebugEnabled() {
		marshaler := jsonpb.Marshaler{}
		str, _ := marshaler.MarshalToString(req)
		cb.log.Debugf("on stream %d type %s request %s ", id, req.TypeUrl, str)
	}
	return nil
}

func (cb *Callbacks) OnStreamResponse(_ context.Context, id int64, req *discovery.DiscoveryRequest, resp *discovery.DiscoveryResponse) {
	if cb.log.DebugEnabled() {
		marshaler := jsonpb.Marshaler{}
		reqstr, _ := marshaler.MarshalToString(req)
		respstr, _ := marshaler.MarshalToString(resp)
		cb.log.Debugf("on stream %d type %s request %s response %s", id, req.TypeUrl, reqstr, respstr)
	}
}

func (cb *Callbacks) OnStreamDeltaResponse(id int64, req *discovery.DeltaDiscoveryRequest, resp *discovery.DeltaDiscoveryResponse) {
	if cb.log.DebugEnabled() {
		marshaler := jsonpb.Marshaler{}
		reqstr, _ := marshaler.MarshalToString(req)
		respstr, _ := marshaler.MarshalToString(resp)
		cb.log.Debugf("on delta stream %d type %s request %s response %s", id, req.TypeUrl, reqstr, respstr)
	}
}

func (cb *Callbacks) OnStreamDeltaRequest(id int64, req *discovery.DeltaDiscoveryRequest) error {
	if cb.log.DebugEnabled() {
		marshaler := jsonpb.Marshaler{}
		str, _ := marshaler.MarshalToString(req)
		cb.log.Debugf("on stream %d delta type %s request %s", id, req.TypeUrl, str)
	}
	return nil
}

func (cb *Callbacks) OnFetchRequest(_ context.Context, req *discovery.DiscoveryRequest) error {
	if cb.log.DebugEnabled() {
		marshaler := jsonpb.Marshaler{}
		str, _ := marshaler.MarshalToString(req)
		cb.log.Debugf("on fetch type %s request %s ", req.TypeUrl, str)
	}
	return nil
}

func (cb *Callbacks) OnFetchResponse(req *discovery.DiscoveryRequest, resp *discovery.DiscoveryResponse) {
	if cb.log.DebugEnabled() {
		marshaler := jsonpb.Marshaler{}
		reqstr, _ := marshaler.MarshalToString(req)
		respstr, _ := marshaler.MarshalToString(resp)
		cb.log.Debugf("on fetch type %s request %s response %s", req.TypeUrl, reqstr, respstr)
	}
}
