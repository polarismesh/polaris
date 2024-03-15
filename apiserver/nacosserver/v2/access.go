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

package v2

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"go.uber.org/zap"

	nacosmodel "github.com/polarismesh/polaris/apiserver/nacosserver/model"
	nacospb "github.com/polarismesh/polaris/apiserver/nacosserver/v2/pb"
	"github.com/polarismesh/polaris/apiserver/nacosserver/v2/remote"
	"github.com/polarismesh/polaris/common/utils"
)

var (
	ErrorNoSuchPayloadType      = errors.New("not such payload type")
	ErrorInvalidRequestBodyType = errors.New("invalid request body type")
)

type (
	// RequestHandler
	RequestHandler func(context.Context, nacospb.BaseRequest, nacospb.RequestMeta) (nacospb.BaseResponse, error)
	// RequestHandlerWarrper
	RequestHandlerWarrper struct {
		Handler        RequestHandler
		PayloadBuilder func() nacospb.CustomerPayload
	}
)

var (
	debugLevel = map[string]struct{}{
		"HealthCheckRequest": {},
	}
)

func (h *NacosV2Server) Request(ctx context.Context, payload *nacospb.Payload) (*nacospb.Payload, error) {
	ctx = h.ConvertContext(ctx)
	h.connectionManager.RefreshClient(ctx)
	ctx = injectPayloadHeader(ctx, payload)
	handle, val, err := h.UnmarshalPayload(payload)
	if err != nil {
		return nil, err
	}
	msg, ok := val.(nacospb.BaseRequest)
	if !ok {
		return nil, ErrorInvalidRequestBodyType
	}
	nacoslog.Debug("[NACOS-V2] handler client request", zap.String("conn-id", remote.ValueConnID(ctx)),
		utils.ZapRequestID(msg.GetRequestId()), zap.String("type", msg.GetRequestType()))
	connMeta := remote.ValueConnMeta(ctx)

	startTime := time.Now()
	resp, err := handle(ctx, msg, nacospb.RequestMeta{
		ConnectionID:  remote.ValueConnID(ctx),
		ClientIP:      payload.GetMetadata().GetClientIp(),
		ClientVersion: connMeta.Version,
		Labels:        connMeta.Labels,
	})
	// 打印耗时超过1s的请求
	if diff := time.Since(startTime); diff > time.Second {
		nacoslog.Info("[NACOS-V2] handler client request", zap.String("conn-id", remote.ValueConnID(ctx)),
			utils.ZapRequestID(msg.GetRequestId()),
			zap.String("type", msg.GetRequestType()),
			zap.Duration("handling-time", diff),
		)
	}

	if err != nil {
		resp = toNacosErrorResp(err)
	}

	resp.SetRequestId(msg.GetRequestId())
	return remote.MarshalPayload(resp)
}

func toNacosErrorResp(err error) nacospb.BaseResponse {
	if nacosErr, ok := err.(*nacosmodel.NacosError); ok {
		return &nacospb.ErrorResponse{
			Response: &nacospb.Response{
				ResultCode: int(nacosmodel.Response_Fail.Code),
				ErrorCode:  int(nacosErr.ErrCode),
				Success:    false,
				Message:    nacosErr.ErrMsg,
			},
		}
	} else if nacosErr, ok := err.(*nacosmodel.NacosApiError); ok {
		return &nacospb.ErrorResponse{
			Response: &nacospb.Response{
				ResultCode: int(nacosmodel.Response_Fail.Code),
				ErrorCode:  int(nacosErr.DetailErrCode),
				Success:    false,
				Message:    nacosErr.ErrAbstract,
			},
		}
	}
	return &nacospb.ErrorResponse{
		Response: &nacospb.Response{
			ResultCode: int(nacosmodel.Response_Fail.Code),
			ErrorCode:  int(nacosmodel.ErrorCode_ServerError.Code),
			Success:    false,
			Message:    err.Error(),
		},
	}
}

func (h *NacosV2Server) RequestBiStream(svr nacospb.BiRequestStream_RequestBiStreamServer) error {
	ctx := h.ConvertContext(svr.Context())
	connID := remote.ValueConnID(ctx)
	client, ok := h.connectionManager.GetClient(connID)
	if ok {
		client.SetStreamRef(&remote.SyncServerStream{Stream: svr})
	}
	nacoslog.Info("[NACOS-V2] client use birequest to register stream", zap.String("conn-id", remote.ValueConnID(ctx)))

	for {
		req, err := svr.Recv()
		if err != nil {
			if io.EOF == err {
				return nil
			}
			return err
		}
		ctx = injectPayloadHeader(ctx, req)
		_, val, err := h.UnmarshalPayload(req)
		if err != nil {
			return err
		}
		switch msg := val.(type) {
		case *nacospb.ConnectionSetupRequest:
			nacoslog.Info("[NACOS-V2] handler client birequest", zap.String("conn-id", remote.ValueConnID(ctx)),
				utils.ZapRequestID(msg.GetRequestId()),
				zap.String("type", msg.GetRequestType()),
			)
			if err := h.connectionManager.RegisterConnection(ctx, req, msg); err != nil {
				return err
			}
		case nacospb.BaseResponse:
			nacoslog.Info("[NACOS-V2] handler client birequest", zap.String("conn-id", remote.ValueConnID(ctx)),
				utils.ZapRequestID(msg.GetRequestId()),
				zap.String("resp-type", msg.GetResponseType()),
			)
			// 刷新链接的最近一次更新时间
			h.connectionManager.RefreshClient(ctx)
			if _, ok := msg.(*nacospb.NotifySubscriberResponse); !ok {
				continue
			}
			// notify ack msg to callback
			h.connectionManager.InFlights().NotifyInFlight(connID, msg)
		}
	}
}

// UnmarshalPayload .
func (h *NacosV2Server) UnmarshalPayload(payload *nacospb.Payload) (remote.RequestHandler, nacospb.CustomerPayload, error) {
	t := payload.GetMetadata().GetType()
	nacoslog.Debug("[API-Server][NACOS-V2] unmarshal payload info", zap.String("type", t))
	handler, ok := h.handleRegistry[t]
	if !ok {
		return nil, nil, ErrorNoSuchPayloadType
	}
	msg := handler.PayloadBuilder()
	if err := json.Unmarshal(payload.GetBody().GetValue(), msg); err != nil {
		return nil, nil, err
	}
	return handler.Handler, msg, nil
}

func injectPayloadHeader(ctx context.Context, payload *nacospb.Payload) context.Context {
	metadata := payload.GetMetadata()
	if metadata == nil {
		return ctx
	}
	if len(metadata.Headers) == 0 {
		return ctx
	}
	token, exist := metadata.Headers[nacosmodel.NacosClientAuthHeader]
	if exist {
		ctx = context.WithValue(ctx, utils.ContextAuthTokenKey, token)
	}
	for k, v := range metadata.Headers {
		ctx = context.WithValue(ctx, utils.StringContext(k), v)
	}
	return ctx
}
