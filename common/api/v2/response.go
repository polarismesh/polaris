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
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"

	v1 "github.com/polarismesh/polaris/common/api/v1"
)

/**
 * @brief 回复消息接口
 */
type ResponseMessage interface {
	proto.Message
	GetCode() uint32
	GetInfo() string
}

/**
 * @brief 获取返回码前三位
 * @note 返回码前三位和HTTP返回码定义一致
 */
func CalcCode(rm ResponseMessage) int {
	return int(rm.GetCode() / 1000)
}

/**
 * @brief BatchWriteResponse添加Response
 */
func (b *BatchWriteResponse) Collect(response *Response) {
	// 非200的code，都归为异常
	if CalcCode(response) != 200 {
		if response.GetCode() >= b.GetCode() {
			b.Code = response.GetCode()
			b.Info = v1.Code2Info(b.GetCode())
		}
	}

	b.Size++
	b.Responses = append(b.Responses, response)
}

/**
 * @brief BatchWriteResponse添加Response
 */
func (b *BatchWriteResponse) CollectBatch(response []*Response) {
	for _, resp := range response {
		b.Collect(resp)
	}
}

/**
 * @brief 创建简单回复
 */
func NewSimpleResponse(code uint32) *SimpleResponse {
	return &SimpleResponse{
		Code: code,
		Info: v1.Code2Info(code),
	}
}

/**
 * @brief 创建回复
 */
func NewResponse(code uint32) *Response {
	return &Response{
		Code: code,
		Info: v1.Code2Info(code),
	}
}

// 带上具体的错误信息
func NewResponseWithMsg(code uint32, msg string) *Response {
	resp := NewResponse(code)
	resp.Info += ": " + msg
	return resp
}

/**
 * @brief 创建回复带服务路由信息
 */
func NewRoutingResponse(code uint32, routing *Routing) *Response {
	ret, err := ptypes.MarshalAny(routing)
	if err != nil {
		return &Response{
			Code: code,
			Info: v1.Code2Info(code),
		}
	}

	return &Response{
		Code: code,
		Info: v1.Code2Info(code),
		Data: ret,
	}
}

/**
 * @brief 创建批量回复
 */
func NewBatchWriteResponse(code uint32) *BatchWriteResponse {
	return &BatchWriteResponse{
		Code: code,
		Info: v1.Code2Info(code),
		Size: 0,
	}
}

/**
 * @brief 创建带详细信息的批量回复
 */
func NewBatchWriteResponseWithMsg(code uint32, msg string) *BatchWriteResponse {
	resp := NewBatchWriteResponse(code)
	resp.Info += ": " + msg
	return resp
}

/**
 * @brief 创建批量查询回复
 */
func NewBatchQueryResponse(code uint32) *BatchQueryResponse {
	return &BatchQueryResponse{
		Code:   code,
		Info:   v1.Code2Info(code),
		Amount: 0,
		Size:   0,
	}
}

/**
 * @brief 创建带详细信息的批量查询回复
 */
func NewBatchQueryResponseWithMsg(code uint32, msg string) *BatchQueryResponse {
	resp := NewBatchQueryResponse(code)
	resp.Info += ": " + msg
	return resp
}

// 格式化responses
// batch操作
// 如果所有子错误码一致，那么使用子错误码
// 如果包含任意一个5xx，那么返回500
func FormatBatchWriteResponse(response *BatchWriteResponse) *BatchWriteResponse {
	var code uint32
	for _, resp := range response.Responses {
		if code == 0 {
			code = resp.GetCode()
			continue
		}
		if code == resp.GetCode() {
			continue
		}
		// 发现不一样
		code = 0
		break
	}
	// code不等于0，意味着所有的resp都是一样的错误码，则合并为同一个错误码
	if code != 0 {
		response.Code = code
		response.Info = v1.Code2Info(code)
		return response
	}

	// 错误都不一致
	// 存在5XX，则返回500
	// 不存在5XX，但是存在4XX，则返回4XX
	// 除去以上两个情况，不修改返回值
	hasBadRequest := false
	for _, resp := range response.Responses {
		httpStatus := CalcCode(resp)
		if httpStatus >= 500 {
			response.Code = v1.ExecuteException
			response.Info = v1.Code2Info(response.Code)
			return response
		} else if httpStatus >= 400 {
			hasBadRequest = true
		}
	}

	if hasBadRequest {
		response.Code = v1.BadRequest
		response.Info = v1.Code2Info(response.Code)
	}
	return response
}

/**
 * @brief 创建查询服务路由回复
 */
func NewDiscoverRoutingResponse(code uint32, service *Service) *DiscoverResponse {
	return &DiscoverResponse{
		Code:    code,
		Info:    v1.Code2Info(code),
		Type:    DiscoverResponse_ROUTING,
		Service: service,
	}
}

// 创建一个空白的discoverResponse
func NewDiscoverResponse(code uint32) *DiscoverResponse {
	return &DiscoverResponse{
		Code: code,
		Info: v1.Code2Info(code),
	}
}
