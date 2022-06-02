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

package grpcserver

import (
	"context"
	"io"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

// initVirtualStream 对 VirtualStream 的一些初始化动作
type initVirtualStream func(vStream *VirtualStream)

// WithVirtualStreamMethod 设置 method
func WithVirtualStreamMethod(method string) initVirtualStream {
	return func(vStream *VirtualStream) {
		vStream.Method = method
	}
}

// WithVirtualStreamServerStream 设置 grpc.ServerStream
func WithVirtualStreamServerStream(stream grpc.ServerStream) initVirtualStream {
	return func(vStream *VirtualStream) {
		vStream.stream = stream
	}
}

// WithVirtualStreamPreProcessFunc 设置 PreProcessFunc
func WithVirtualStreamPreProcessFunc(preprocess PreProcessFunc) initVirtualStream {
	return func(vStream *VirtualStream) {
		vStream.preprocess = preprocess
	}
}

// WithVirtualStreamPostProcessFunc 设置 PostProcessFunc
func WithVirtualStreamPostProcessFunc(postprocess PostProcessFunc) initVirtualStream {
	return func(vStream *VirtualStream) {
		vStream.postprocess = postprocess
	}
}

// WithVirtualStreamBaseServer 设置 BaseGrpcServer
func WithVirtualStreamBaseServer(server *BaseGrpcServer) initVirtualStream {
	return func(vStream *VirtualStream) {
		vStream.server = server
	}
}

func newVirtualStream(ctx context.Context, initOptions ...initVirtualStream) *VirtualStream {
	var clientAddress string
	var clientIP string
	var userAgent string
	var requestID string

	peerAddress, exist := peer.FromContext(ctx)
	if exist {
		clientAddress = peerAddress.Addr.String()
		// 解析获取clientIP
		items := strings.Split(clientAddress, ":")
		if len(items) == 2 {
			clientIP = items[0]
		}
	}

	meta, exist := metadata.FromIncomingContext(ctx)
	if exist {
		agents := meta["user-agent"]
		if len(agents) > 0 {
			userAgent = agents[0]
		}

		ids := meta["request-id"]
		if len(ids) > 0 {
			requestID = ids[0]
		}
	}

	virtualStream := &VirtualStream{
		ClientAddress: clientAddress,
		ClientIP:      clientIP,
		UserAgent:     userAgent,
		RequestID:     requestID,
		server:        nil,
		stream:        nil,
		Code:          0,
	}

	for i := range initOptions {
		initOptions[i](virtualStream)
	}

	return virtualStream
}

// VirtualStream 虚拟Stream 继承ServerStream
type VirtualStream struct {
	server *BaseGrpcServer

	Method        string
	ClientAddress string
	ClientIP      string
	UserAgent     string
	RequestID     string

	stream grpc.ServerStream

	Code int

	preprocess  PreProcessFunc
	postprocess PostProcessFunc

	StartTime time.Time
}

// SetHeader sets the header metadata. It may be called multiple times.
// When call multiple times, all the provided metadata will be merged.
// All the metadata will be sent out when one of the following happens:
//  - ServerStream.SendHeader() is called;
//  - The first response is sent out;
//  - An RPC status is sent out (error or success).
func (v *VirtualStream) SetHeader(md metadata.MD) error {
	return v.stream.SetHeader(md)
}

// SendHeader sends the header metadata.
// The provided md and headers set by SetHeader() will be sent.
// It fails if called multiple times.
func (v *VirtualStream) SendHeader(md metadata.MD) error {
	return v.stream.SendHeader(md)
}

// SetTrailer sets the trailer metadata which will be sent with the RPC status.
// When called more than once, all the provided metadata will be merged.
func (v *VirtualStream) SetTrailer(md metadata.MD) {
	v.stream.SetTrailer(md)
}

// Context returns the context for this stream.
func (v *VirtualStream) Context() context.Context {
	return v.stream.Context()
}

// RecvMsg blocks until it receives a message into m or the stream is
// done. It returns io.EOF when the client has performed a CloseSend. On
// any non-EOF error, the stream is aborted and the error contains the
// RPC status.
//
// It is safe to have a goroutine calling SendMsg and another goroutine
// calling RecvMsg on the same stream at the same time, but it is not
// safe to call RecvMsg on the same stream in different goroutines.
func (v *VirtualStream) RecvMsg(m interface{}) error {
	err := v.stream.RecvMsg(m)
	if err == io.EOF {
		return err
	}

	if err == nil {
		err = v.preprocess(v, false)
	} else {
		v.Code = -1
	}

	return err
}

// SendMsg sends a message. On error, SendMsg aborts the stream and the
// error is returned directly.
//
// SendMsg blocks until:
//   - There is sufficient flow control to schedule m with the transport, or
//   - The stream is done, or
//   - The stream breaks.
//
// SendMsg does not wait until the message is received by the client. An
// untimely stream closure may result in lost messages.
//
// It is safe to have a goroutine calling SendMsg and another goroutine
// calling RecvMsg on the same stream at the same time, but it is not safe
// to call SendMsg on the same stream in different goroutines.
func (v *VirtualStream) SendMsg(m interface{}) error {
	v.postprocess(v, m)

	m = v.handleResponse(v.stream, m)

	err := v.stream.SendMsg(m)
	if err != nil {
		v.Code = -2
	}

	return err
}

func (v *VirtualStream) handleResponse(stream grpc.ServerStream, m interface{}) interface{} {
	if v.server.cache == nil {
		return m
	}

	cacheVal := v.server.convert(m)
	if cacheVal == nil {
		return m
	}

	if saveVal := v.server.cache.Get(cacheVal.CacheType, cacheVal.Key); saveVal != nil {
		return saveVal.GetPreparedMessage()
	}

	if err := cacheVal.PrepareMessage(stream); err != nil {
		return m
	}

	cacheVal = v.server.cache.Put(cacheVal)
	if cacheVal == nil {
		return m
	}

	return cacheVal.GetPreparedMessage()
}
