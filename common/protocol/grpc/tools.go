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

package grpc

import (
	"context"
	"github.com/polarismesh/polaris-server/common/utils"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"strings"
)

/**
 * @brief 将GRPC上下文转换成内部上下文
 */
func ConvertContext(ctx context.Context) context.Context {
	requestID := ""
	userAgent := ""
	meta, exist := metadata.FromIncomingContext(ctx)
	if exist {
		ids := meta["request-id"]
		if len(ids) > 0 {
			requestID = ids[0]
		}
		agents := meta["user-agent"]
		if len(agents) > 0 {
			userAgent = agents[0]
		}
	}

	clientIP := ""
	address := ""
	if pr, ok := peer.FromContext(ctx); ok && pr.Addr != nil {
		address = pr.Addr.String()
		addrSlice := strings.Split(address, ":")
		if len(addrSlice) == 2 {
			clientIP = addrSlice[0]
		}
	}

	ctx = context.Background()
	ctx = context.WithValue(ctx, utils.StringContext("request-id"), requestID)
	ctx = context.WithValue(ctx, utils.StringContext("client-ip"), clientIP)
	ctx = context.WithValue(ctx, utils.StringContext("client-address"), address)
	ctx = context.WithValue(ctx, utils.StringContext("user-agent"), userAgent)
	return ctx
}