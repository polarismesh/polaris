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
	"fmt"
	"io"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/polarismesh/polaris/apiserver/grpcserver"
	apiv1 "github.com/polarismesh/polaris/common/api/v1"
	apiv2 "github.com/polarismesh/polaris/common/api/v2"
	commonlog "github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/utils"
)

var (
	namingLog = commonlog.GetScopeOrDefaultByName(commonlog.NamingLoggerName)
)

// Discover 统一发现接口
func (g *DiscoverServer) Discover(server apiv2.PolarisGRPC_DiscoverServer) error {
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

		msg := fmt.Sprintf("receive grpc discover v2 request: %s", in.GetSerivce().String())
		namingLog.Info(msg,
			zap.String("type", apiv2.DiscoverRequest_DiscoverRequestType_name[int32(in.Type)]),
			zap.String("client-address", clientAddress),
			zap.String("user-agent", userAgent),
			utils.ZapRequestID(requestID),
		)

		// 是否允许访问
		if ok := g.allowAccess(method); !ok {
			resp := apiv2.NewDiscoverResponse(apiv1.ClientAPINotOpen)
			if sendErr := server.Send(resp); sendErr != nil {
				return sendErr
			}
			continue
		}

		// stream模式，需要对每个包进行检测
		if code := g.enterRateLimit(clientIP, method); code != apiv1.ExecuteSuccess {
			resp := apiv2.NewDiscoverResponse(code)
			if err = server.Send(resp); err != nil {
				return err
			}
			continue
		}

		var out *apiv2.DiscoverResponse
		switch in.Type {
		case apiv2.DiscoverRequest_ROUTING:
			out = g.namingServer.GetRoutingConfigV2WithCache(ctx, in.GetSerivce())
		case apiv2.DiscoverRequest_CIRCUIT_BREAKER:
			out = g.namingServer.GetCircuitBreakerV2WithCache(ctx, in.GetSerivce())
		default:
			out = apiv2.NewDiscoverRoutingResponse(apiv1.InvalidDiscoverResource, in.GetSerivce())
		}

		err = server.Send(out)
		if err != nil {
			return err
		}
	}
}
