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

package discover

import (
	"context"
	"fmt"
	"io"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"

	"github.com/polarismesh/polaris-server/apiserver/grpcserver"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/utils"
)

// ReportClient 客户端上报
func (g *GRPCServer) ReportClient(ctx context.Context, in *api.Client) (*api.Response, error) {
	return g.namingServer.ReportClient(grpcserver.ConvertContext(ctx), in), nil
}

// RegisterInstance 注册服务实例
func (g *GRPCServer) RegisterInstance(ctx context.Context, in *api.Instance) (*api.Response, error) {
	// 需要记录操作来源，提高效率，只针对特殊接口添加operator
	rCtx := grpcserver.ConvertContext(ctx)
	rCtx = context.WithValue(rCtx, utils.StringContext("operator"), ParseGrpcOperator(ctx))

	// 客户端请求中带了 token 的，优先已请求中的为准
	if in.GetServiceToken().GetValue() != "" {
		rCtx = context.WithValue(rCtx, utils.ContextAuthTokenKey, in.GetServiceToken().GetValue())
	}

	if in.GetHealthCheck().GetHeartbeat().GetTtl() != nil {
		rCtx = context.WithValue(rCtx, utils.ContextOpenAsyncRegis, true)
	}

	out := g.namingServer.RegisterInstance(rCtx, in)
	return out, nil
}

// DeregisterInstance 反注册服务实例
func (g *GRPCServer) DeregisterInstance(ctx context.Context, in *api.Instance) (*api.Response, error) {
	// 需要记录操作来源，提高效率，只针对特殊接口添加operator
	rCtx := grpcserver.ConvertContext(ctx)
	rCtx = context.WithValue(rCtx, utils.StringContext("operator"), ParseGrpcOperator(ctx))

	// 客户端请求中带了 token 的，优先已请求中的为准
	if in.GetServiceToken().GetValue() != "" {
		rCtx = context.WithValue(rCtx, utils.ContextAuthTokenKey, in.GetServiceToken().GetValue())
	}

	out := g.namingServer.DeregisterInstance(rCtx, in)
	return out, nil
}

// Discover 统一发现接口
func (g *GRPCServer) Discover(server api.PolarisGRPC_DiscoverServer) error {
	ctx := grpcserver.ConvertContext(server.Context())
	clientIP, _ := ctx.Value(utils.StringContext("client-ip")).(string)
	clientAddress, _ := ctx.Value(utils.StringContext("client-address")).(string)
	requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)
	userAgent, _ := ctx.Value(utils.StringContext("user-agent")).(string)
	method, _ := grpc.MethodFromServerStream(server)

	for {
		in, err := server.Recv()
		if err != nil {
			if io.EOF == err {
				return nil
			}
			return err
		}

		msg := fmt.Sprintf("receive grpc discover request: %s", in.Service.String())
		log.Info(msg,
			zap.String("type", api.DiscoverRequest_DiscoverRequestType_name[int32(in.Type)]),
			zap.String("client-address", clientAddress),
			zap.String("user-agent", userAgent),
			zap.String("request-id", requestID),
		)

		// 是否允许访问
		if ok := g.allowAccess(method); !ok {
			resp := api.NewDiscoverResponse(api.ClientAPINotOpen)
			if sendErr := server.Send(resp); sendErr != nil {
				return sendErr
			}
			continue
		}

		// stream模式，需要对每个包进行检测
		if code := g.enterRateLimit(clientIP, method); code != api.ExecuteSuccess {
			resp := api.NewDiscoverResponse(code)
			if err = server.Send(resp); err != nil {
				return err
			}
			continue
		}

		var out *api.DiscoverResponse
		switch in.Type {
		case api.DiscoverRequest_INSTANCE:
			out = g.namingServer.ServiceInstancesCache(ctx, in.Service)
		case api.DiscoverRequest_ROUTING:
			out = g.namingServer.GetRoutingConfigWithCache(ctx, in.Service)
		case api.DiscoverRequest_RATE_LIMIT:
			out = g.namingServer.GetRateLimitWithCache(ctx, in.Service)
		case api.DiscoverRequest_CIRCUIT_BREAKER:
			out = g.namingServer.GetCircuitBreakerWithCache(ctx, in.Service)
		case api.DiscoverRequest_SERVICES:
			out = g.namingServer.GetServiceWithCache(ctx, in.Service)
		default:
			out = api.NewDiscoverRoutingResponse(api.InvalidDiscoverResource, in.Service)
		}

		err = server.Send(out)
		if err != nil {
			return err
		}
	}
}

// Heartbeat 上报心跳
func (g *GRPCServer) Heartbeat(ctx context.Context, in *api.Instance) (*api.Response, error) {
	return g.healthCheckServer.Report(grpcserver.ConvertContext(ctx), in), nil
}

// ParseGrpcOperator 构造请求源
func ParseGrpcOperator(ctx context.Context) string {
	// 获取请求源
	operator := "GRPC"
	if pr, ok := peer.FromContext(ctx); ok && pr.Addr != nil {
		addrSlice := strings.Split(pr.Addr.String(), ":")
		if len(addrSlice) == 2 {
			operator += ":" + addrSlice[0]
		}
	}

	return operator
}
