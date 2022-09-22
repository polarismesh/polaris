package xdsserverv3

import (
	"context"

	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/gogo/protobuf/jsonpb"
	commonlog "github.com/polarismesh/polaris-server/common/log"
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
		cb.log.Debugf("on stream %d request %s ", req.TypeUrl, str)
	}
	return nil
}

func (cb *Callbacks) OnStreamResponse(_ context.Context, id int64, req *discovery.DiscoveryRequest, resp *discovery.DiscoveryResponse) {
	if cb.log.DebugEnabled() {
		marshaler := jsonpb.Marshaler{}
		reqstr, _ := marshaler.MarshalToString(req)
		respstr, _ := marshaler.MarshalToString(resp)
		cb.log.Debugf("on stream %d request %s response %s", req.TypeUrl, reqstr, respstr)
	}
}

func (cb *Callbacks) OnStreamDeltaResponse(id int64, req *discovery.DeltaDiscoveryRequest, resp *discovery.DeltaDiscoveryResponse) {
	if cb.log.DebugEnabled() {
		marshaler := jsonpb.Marshaler{}
		reqstr, _ := marshaler.MarshalToString(req)
		respstr, _ := marshaler.MarshalToString(resp)
		cb.log.Debugf("on delta stream %d request %s response %s", req.TypeUrl, reqstr, respstr)
	}
}

func (cb *Callbacks) OnStreamDeltaRequest(id int64, req *discovery.DeltaDiscoveryRequest) error {
	if cb.log.DebugEnabled() {
		marshaler := jsonpb.Marshaler{}
		str, _ := marshaler.MarshalToString(req)
		cb.log.Debugf("on stream %d delta request %s ", req.TypeUrl, str)
	}
	return nil
}

func (cb *Callbacks) OnFetchRequest(_ context.Context, req *discovery.DiscoveryRequest) error {
	if cb.log.DebugEnabled() {
		marshaler := jsonpb.Marshaler{}
		str, _ := marshaler.MarshalToString(req)
		cb.log.Debugf("on fetch request %s ", req.TypeUrl, str)
	}
	return nil
}

func (cb *Callbacks) OnFetchResponse(req *discovery.DiscoveryRequest, resp *discovery.DiscoveryResponse) {
	if cb.log.DebugEnabled() {
		marshaler := jsonpb.Marshaler{}
		reqstr, _ := marshaler.MarshalToString(req)
		respstr, _ := marshaler.MarshalToString(resp)
		cb.log.Debugf("on fetch request %s response %s", req.TypeUrl, reqstr, respstr)
	}
}
