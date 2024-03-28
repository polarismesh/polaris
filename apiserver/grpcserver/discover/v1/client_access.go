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
	"context"
	"fmt"
	"io"
	"strings"

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	api "github.com/polarismesh/polaris/common/api/v1"
	commonlog "github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/metrics"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
)

var (
	accesslog = commonlog.GetScopeOrDefaultByName(commonlog.APIServerLoggerName)
)

// ReportClient 客户端上报
func (g *DiscoverServer) ReportClient(ctx context.Context, in *apiservice.Client) (*apiservice.Response, error) {
	return g.namingServer.ReportClient(utils.ConvertGRPCContext(ctx), in), nil
}

// RegisterInstance 注册服务实例
func (g *DiscoverServer) RegisterInstance(ctx context.Context, in *apiservice.Instance) (*apiservice.Response, error) {
	// 需要记录操作来源，提高效率，只针对特殊接口添加operator
	rCtx := utils.ConvertGRPCContext(ctx)
	rCtx = context.WithValue(rCtx, utils.StringContext("operator"), ParseGrpcOperator(ctx))

	// 客户端请求中带了 token 的，优先已请求中的为准
	if in.GetServiceToken().GetValue() != "" {
		rCtx = context.WithValue(rCtx, utils.ContextAuthTokenKey, in.GetServiceToken().GetValue())
	}

	grpcHeader := rCtx.Value(utils.ContextGrpcHeader).(metadata.MD)

	if _, ok := grpcHeader["async-regis"]; ok {
		rCtx = context.WithValue(rCtx, utils.ContextOpenAsyncRegis, true)
	}

	out := g.namingServer.RegisterInstance(rCtx, in)
	return out, nil
}

// DeregisterInstance 反注册服务实例
func (g *DiscoverServer) DeregisterInstance(
	ctx context.Context, in *apiservice.Instance) (*apiservice.Response, error) {
	// 需要记录操作来源，提高效率，只针对特殊接口添加operator
	rCtx := utils.ConvertGRPCContext(ctx)
	rCtx = context.WithValue(rCtx, utils.StringContext("operator"), ParseGrpcOperator(ctx))

	// 客户端请求中带了 token 的，优先已请求中的为准
	if in.GetServiceToken().GetValue() != "" {
		rCtx = context.WithValue(rCtx, utils.ContextAuthTokenKey, in.GetServiceToken().GetValue())
	}

	out := g.namingServer.DeregisterInstance(rCtx, in)
	return out, nil
}

// Discover 统一发现接口
func (g *DiscoverServer) Discover(server apiservice.PolarisGRPC_DiscoverServer) error {
	ctx := utils.ConvertGRPCContext(server.Context())
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
		accesslog.Info(msg,
			zap.String("type", apiservice.DiscoverRequest_DiscoverRequestType_name[int32(in.Type)]),
			zap.String("client-address", clientAddress),
			zap.String("user-agent", userAgent),
			utils.ZapRequestID(requestID),
		)

		// 是否允许访问
		if ok := g.allowAccess(method); !ok {
			resp := api.NewDiscoverResponse(apimodel.Code_ClientAPINotOpen)
			if sendErr := server.Send(resp); sendErr != nil {
				return sendErr
			}
			continue
		}

		// stream模式，需要对每个包进行检测
		if code := g.enterRateLimit(clientIP, method); code != uint32(apimodel.Code_ExecuteSuccess) {
			resp := api.NewDiscoverResponse(apimodel.Code(code))
			if err = server.Send(resp); err != nil {
				return err
			}
			continue
		}

		var out *apiservice.DiscoverResponse
		var action string
		startTime := commontime.CurrentMillisecond()
		defer func() {
			plugin.GetStatis().ReportDiscoverCall(metrics.ClientDiscoverMetric{
				Action:    action,
				ClientIP:  utils.ParseClientAddress(ctx),
				Namespace: in.GetService().GetNamespace().GetValue(),
				Resource:  in.GetType().String() + ":" + in.GetService().GetName().GetValue(),
				Timestamp: startTime,
				CostTime:  commontime.CurrentMillisecond() - startTime,
				Revision:  out.GetService().GetRevision().GetValue(),
				Success:   out.GetCode().GetValue() > uint32(apimodel.Code_DataNoChange),
			})
		}()

		// 兼容。如果请求中带了token，优先使用该token
		if in.GetService().GetToken().GetValue() != "" {
			ctx = context.WithValue(ctx, utils.ContextAuthTokenKey, in.GetService().GetToken().GetValue())
		}

		switch in.Type {
		case apiservice.DiscoverRequest_INSTANCE:
			action = metrics.ActionDiscoverInstance
			out = g.namingServer.ServiceInstancesCache(ctx, &apiservice.DiscoverFilter{}, in.Service)
		case apiservice.DiscoverRequest_ROUTING:
			action = metrics.ActionDiscoverRouterRule
			out = g.namingServer.GetRoutingConfigWithCache(ctx, in.Service)
		case apiservice.DiscoverRequest_RATE_LIMIT:
			action = metrics.ActionDiscoverRateLimit
			out = g.namingServer.GetRateLimitWithCache(ctx, in.Service)
		case apiservice.DiscoverRequest_CIRCUIT_BREAKER:
			action = metrics.ActionDiscoverCircuitBreaker
			out = g.namingServer.GetCircuitBreakerWithCache(ctx, in.Service)
		case apiservice.DiscoverRequest_SERVICES:
			action = metrics.ActionDiscoverServices
			out = g.namingServer.GetServiceWithCache(ctx, in.Service)
		case apiservice.DiscoverRequest_FAULT_DETECTOR:
			action = metrics.ActionDiscoverFaultDetect
			out = g.namingServer.GetFaultDetectWithCache(ctx, in.Service)
		default:
			out = api.NewDiscoverRoutingResponse(apimodel.Code_InvalidDiscoverResource, in.Service)
		}

		err = server.Send(out)
		if err != nil {
			return err
		}
	}
}

func (g *DiscoverServer) ReportServiceContract(ctx context.Context, in *apiservice.ServiceContract) (*apiservice.Response, error) {
	// 需要记录操作来源，提高效率，只针对特殊接口添加operator
	rCtx := utils.ConvertGRPCContext(ctx)
	rCtx = context.WithValue(rCtx, utils.StringContext("operator"), ParseGrpcOperator(ctx))

	out := g.namingServer.ReportServiceContract(rCtx, in)
	return out, nil
}

// 查询服务契约
func (g *DiscoverServer) GetServiceContract(ctx context.Context, req *apiservice.ServiceContract) (*apiservice.Response, error) {
	// 需要记录操作来源，提高效率，只针对特殊接口添加operator
	rCtx := utils.ConvertGRPCContext(ctx)
	rCtx = context.WithValue(rCtx, utils.StringContext("operator"), ParseGrpcOperator(ctx))

	out := g.namingServer.GetServiceContractWithCache(rCtx, req)
	return out, nil
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
