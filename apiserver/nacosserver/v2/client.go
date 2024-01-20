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

	nacospb "github.com/polarismesh/polaris/apiserver/nacosserver/v2/pb"
	"github.com/polarismesh/polaris/apiserver/nacosserver/v2/remote"
)

// handleServerCheckRequest 客户端首次发起请求，用于向 server 获取当前长连接的 ID 信息
func (h *NacosV2Server) handleServerCheckRequest(ctx context.Context, req nacospb.BaseRequest,
	meta nacospb.RequestMeta) (nacospb.BaseResponse, error) {
	if _, ok := req.(*nacospb.ServerCheckRequest); !ok {
		return nil, ErrorInvalidRequestBodyType
	}
	resp := nacospb.NewServerCheckResponse()
	resp.ConnectionId = remote.ValueConnID(ctx)
	return resp, nil
}

// handleHealthCheckRequest 主要用于客户端检查自己和 server 长连接的健康状态
func (h *NacosV2Server) handleHealthCheckRequest(ctx context.Context, req nacospb.BaseRequest,
	meta nacospb.RequestMeta) (nacospb.BaseResponse, error) {
	if _, ok := req.(*nacospb.HealthCheckRequest); !ok {
		return nil, ErrorInvalidRequestBodyType
	}
	return nacospb.NewHealthCheckResponse(), nil
}
