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
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/wrappers"
)

/**
 * @brief 回复消息接口
 */
type ResponseMessage interface {
	proto.Message
	GetCode() *wrappers.UInt32Value
	GetInfo() *wrappers.StringValue
}

/**
 * @brief 获取返回码前三位
 * @note 返回码前三位和HTTP返回码定义一致
 */
func CalcCode(rm ResponseMessage) int {
	return int(rm.GetCode().GetValue() / 1000)
}

/**
 * @brief BatchWriteResponse添加Response
 */
func (b *BatchWriteResponse) Collect(response *Response) {
	// 非200的code，都归为异常
	if CalcCode(response) != 200 {
		if response.GetCode().GetValue() >= b.GetCode().GetValue() {
			b.Code.Value = response.GetCode().GetValue()
			b.Info.Value = code2info[b.GetCode().GetValue()]
		}
	}

	b.Size.Value++
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
 * @brief BatchQueryResponse添加命名空间
 */
func (b *BatchQueryResponse) AddNamespace(namespace *Namespace) {
	b.Namespaces = append(b.Namespaces, namespace)
}

/**
 * @brief 创建简单回复
 */
func NewSimpleResponse(code uint32) *SimpleResponse {
	return &SimpleResponse{
		Code: &wrappers.UInt32Value{Value: code},
		Info: &wrappers.StringValue{Value: code2info[code]},
	}
}

/**
 * @brief 创建回复
 */
func NewResponse(code uint32) *Response {
	return &Response{
		Code: &wrappers.UInt32Value{Value: code},
		Info: &wrappers.StringValue{Value: code2info[code]},
	}
}

// 带上具体的错误信息
func NewResponseWithMsg(code uint32, msg string) *Response {
	resp := NewResponse(code)
	resp.Info.Value += ": " + msg
	return resp
}

/**
 * @brief 创建回复带客户端信息
 */
func NewClientResponse(code uint32, client *Client) *Response {
	return &Response{
		Code:   &wrappers.UInt32Value{Value: code},
		Info:   &wrappers.StringValue{Value: code2info[code]},
		Client: client,
	}
}

/**
 * @brief 创建回复带命名空间信息
 */
func NewNamespaceResponse(code uint32, namespace *Namespace) *Response {
	return &Response{
		Code:      &wrappers.UInt32Value{Value: code},
		Info:      &wrappers.StringValue{Value: code2info[code]},
		Namespace: namespace,
	}
}

/**
 * @brief 创建回复带服务信息
 */
func NewServiceResponse(code uint32, service *Service) *Response {
	return &Response{
		Code:    &wrappers.UInt32Value{Value: code},
		Info:    &wrappers.StringValue{Value: code2info[code]},
		Service: service,
	}
}

// 创建带别名信息的答复
func NewServiceAliasResponse(code uint32, alias *ServiceAlias) *Response {
	resp := NewResponse(code)
	resp.Alias = alias
	return resp
}

/**
 * @brief 创建回复带服务实例信息
 */
func NewInstanceResponse(code uint32, instance *Instance) *Response {
	return &Response{
		Code:     &wrappers.UInt32Value{Value: code},
		Info:     &wrappers.StringValue{Value: code2info[code]},
		Instance: instance,
	}
}

// 创建带自定义error的服务实例response
func NewInstanceRespWithError(code uint32, err error, instance *Instance) *Response {
	resp := NewInstanceResponse(code, instance)
	resp.Info.Value += " : " + err.Error()

	return resp
}

/**
 * @brief 创建回复带服务路由信息
 */
func NewRoutingResponse(code uint32, routing *Routing) *Response {
	return &Response{
		Code:    &wrappers.UInt32Value{Value: code},
		Info:    &wrappers.StringValue{Value: code2info[code]},
		Routing: routing,
	}
}

/**
 * @brief 创建回复带限流规则信息
 */
func NewRateLimitResponse(code uint32, rule *Rule) *Response {
	return &Response{
		Code:      &wrappers.UInt32Value{Value: code},
		Info:      &wrappers.StringValue{Value: code2info[code]},
		RateLimit: rule,
	}
}

/**
 * @brief 创建回复带熔断规则信息
 */
func NewCircuitBreakerResponse(code uint32, circuitBreaker *CircuitBreaker) *Response {
	return &Response{
		Code:           &wrappers.UInt32Value{Value: code},
		Info:           &wrappers.StringValue{Value: code2info[code]},
		CircuitBreaker: circuitBreaker,
	}
}

/**
 * @brief 创建回复带发布信息
 */
func NewConfigResponse(code uint32, configRelease *ConfigRelease) *Response {
	return &Response{
		Code:          &wrappers.UInt32Value{Value: code},
		Info:          &wrappers.StringValue{Value: code2info[code]},
		ConfigRelease: configRelease,
	}
}

/**
 * @brief 创建批量回复
 */
func NewBatchWriteResponse(code uint32) *BatchWriteResponse {
	return &BatchWriteResponse{
		Code: &wrappers.UInt32Value{Value: code},
		Info: &wrappers.StringValue{Value: code2info[code]},
		Size: &wrappers.UInt32Value{Value: 0},
	}
}

/**
 * @brief 创建带详细信息的批量回复
 */
func NewBatchWriteResponseWithMsg(code uint32, msg string) *BatchWriteResponse {
	resp := NewBatchWriteResponse(code)
	resp.Info.Value += ": " + msg
	return resp
}

/**
 * @brief 创建批量查询回复
 */
func NewBatchQueryResponse(code uint32) *BatchQueryResponse {
	return &BatchQueryResponse{
		Code:   &wrappers.UInt32Value{Value: code},
		Info:   &wrappers.StringValue{Value: code2info[code]},
		Amount: &wrappers.UInt32Value{Value: 0},
		Size:   &wrappers.UInt32Value{Value: 0},
	}
}

/**
 * @brief 创建带详细信息的批量查询回复
 */
func NewBatchQueryResponseWithMsg(code uint32, msg string) *BatchQueryResponse {
	resp := NewBatchQueryResponse(code)
	resp.Info.Value += ": " + msg
	return resp
}

// 创建一个空白的discoverResponse
func NewDiscoverResponse(code uint32) *DiscoverResponse {
	return &DiscoverResponse{
		Code: &wrappers.UInt32Value{Value: code},
		Info: &wrappers.StringValue{Value: code2info[code]},
	}
}

/**
 * @brief 创建查询服务回复
 */
func NewDiscoverServiceResponse(code uint32, service *Service) *DiscoverResponse {
	return &DiscoverResponse{
		Code:    &wrappers.UInt32Value{Value: code},
		Info:    &wrappers.StringValue{Value: code2info[code]},
		Type:    DiscoverResponse_SERVICES,
		Service: service,
	}
}

/**
 * @brief 创建查询服务实例回复
 */
func NewDiscoverInstanceResponse(code uint32, service *Service) *DiscoverResponse {
	return &DiscoverResponse{
		Code:    &wrappers.UInt32Value{Value: code},
		Info:    &wrappers.StringValue{Value: code2info[code]},
		Type:    DiscoverResponse_INSTANCE,
		Service: service,
	}
}

/**
 * @brief 创建查询服务路由回复
 */
func NewDiscoverRoutingResponse(code uint32, service *Service) *DiscoverResponse {
	return &DiscoverResponse{
		Code:    &wrappers.UInt32Value{Value: code},
		Info:    &wrappers.StringValue{Value: code2info[code]},
		Type:    DiscoverResponse_ROUTING,
		Service: service,
	}
}

/**
 * @brief 创建查询限流规则回复
 */
func NewDiscoverRateLimitResponse(code uint32, service *Service) *DiscoverResponse {
	return &DiscoverResponse{
		Code:    &wrappers.UInt32Value{Value: code},
		Info:    &wrappers.StringValue{Value: code2info[code]},
		Type:    DiscoverResponse_RATE_LIMIT,
		Service: service,
	}
}

/**
 * @brief 创建查询熔断规则回复
 */
func NewDiscoverCircuitBreakerResponse(code uint32, service *Service) *DiscoverResponse {
	return &DiscoverResponse{
		Code:    &wrappers.UInt32Value{Value: code},
		Info:    &wrappers.StringValue{Value: code2info[code]},
		Type:    DiscoverResponse_CIRCUIT_BREAKER,
		Service: service,
	}
}

// 格式化responses
// batch操作
// 如果所有子错误码一致，那么使用子错误码
// 如果包含任意一个5xx，那么返回500
func FormatBatchWriteResponse(response *BatchWriteResponse) *BatchWriteResponse {
	var code uint32
	for _, resp := range response.Responses {
		if code == 0 {
			code = resp.GetCode().GetValue()
			continue
		}
		if code == resp.GetCode().GetValue() {
			continue
		}
		// 发现不一样
		code = 0
		break
	}
	// code不等于0，意味着所有的resp都是一样的错误码，则合并为同一个错误码
	if code != 0 {
		response.Code.Value = code
		response.Info.Value = code2info[code]
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
			response.Code.Value = ExecuteException
			response.Info.Value = code2info[response.Code.Value]
			return response
		} else if httpStatus >= 400 {
			hasBadRequest = true
		}
	}

	if hasBadRequest {
		response.Code.Value = BadRequest
		response.Info.Value = code2info[response.Code.Value]
	}
	return response
}
