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
	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apifault "github.com/polarismesh/specification/source/go/api/v1/fault_tolerance"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	"google.golang.org/protobuf/types/known/anypb"
)

/**
 * @brief 回复消息接口
 */
type ResponseMessage interface {
	proto.Message
	GetCode() *wrappers.UInt32Value
	GetInfo() *wrappers.StringValue
}

type ResponseMessageV2 interface {
	proto.Message
	GetCode() uint32
	GetInfo() string
}

/**
 * @brief 获取返回码前三位
 * @note 返回码前三位和HTTP返回码定义一致
 */
func CalcCode(rm ResponseMessage) int {
	return int(rm.GetCode().GetValue() / 1000)
}

/**
 * @brief 获取返回码前三位
 * @note 返回码前三位和HTTP返回码定义一致
 */
func CalcCodeV2(rm ResponseMessageV2) int {
	return int(rm.GetCode() / 1000)
}

// IsSuccess .
func IsSuccess(rsp ResponseMessage) bool {
	return rsp.GetCode().GetValue() == uint32(apimodel.Code_ExecuteSuccess)
}

/**
 * @brief BatchWriteResponse添加Response
 */
func Collect(batchWriteResponse *apiservice.BatchWriteResponse, response *apiservice.Response) {
	// 非200的code，都归为异常
	if CalcCode(response) != 200 {
		if response.GetCode().GetValue() >= batchWriteResponse.GetCode().GetValue() {
			batchWriteResponse.Code.Value = response.GetCode().GetValue()
			batchWriteResponse.Info.Value = code2info[batchWriteResponse.GetCode().GetValue()]
		}
	}

	batchWriteResponse.Size.Value++
	batchWriteResponse.Responses = append(batchWriteResponse.Responses, response)
}

/**
 * @brief BatchWriteResponse添加Response
 */
func QueryCollect(resp *apiservice.BatchQueryResponse, response *apiservice.Response) {
	// 非200的code，都归为异常
	if CalcCode(response) != 200 {
		if response.GetCode().GetValue() >= resp.GetCode().GetValue() {
			resp.Code.Value = response.GetCode().GetValue()
			resp.Info.Value = code2info[resp.GetCode().GetValue()]
		}
	}
}

// AddNamespace BatchQueryResponse添加命名空间
func AddNamespace(b *apiservice.BatchQueryResponse, namespace *apimodel.Namespace) {
	b.Namespaces = append(b.Namespaces, namespace)
}

// AddNamespaceSummary 添加汇总信息
func AddNamespaceSummary(b *apiservice.BatchQueryResponse, summary *apimodel.Summary) {
	b.Summary = summary
}

// NewResponse 创建回复
func NewResponse(code apimodel.Code) *apiservice.Response {
	return &apiservice.Response{
		Code: &wrappers.UInt32Value{Value: uint32(code)},
		Info: &wrappers.StringValue{Value: code2info[uint32(code)]},
	}
}

// NewResponseWithMsg 带上具体的错误信息
func NewResponseWithMsg(code apimodel.Code, msg string) *apiservice.Response {
	resp := NewResponse(code)
	resp.Info.Value += ": " + msg
	return resp
}

/**
 * @brief 创建回复带客户端信息
 */
func NewClientResponse(code apimodel.Code, client *apiservice.Client) *apiservice.Response {
	return &apiservice.Response{
		Code:   &wrappers.UInt32Value{Value: uint32(code)},
		Info:   &wrappers.StringValue{Value: code2info[uint32(code)]},
		Client: client,
	}
}

/**
 * @brief 创建回复带命名空间信息
 */
func NewNamespaceResponse(code apimodel.Code, namespace *apimodel.Namespace) *apiservice.Response {
	return &apiservice.Response{
		Code:      &wrappers.UInt32Value{Value: uint32(code)},
		Info:      &wrappers.StringValue{Value: code2info[uint32(code)]},
		Namespace: namespace,
	}
}

/**
 * @brief 创建回复带服务信息
 */
func NewServiceResponse(code apimodel.Code, service *apiservice.Service) *apiservice.Response {
	return &apiservice.Response{
		Code:    &wrappers.UInt32Value{Value: uint32(code)},
		Info:    &wrappers.StringValue{Value: code2info[uint32(code)]},
		Service: service,
	}
}

// 创建带别名信息的答复
func NewServiceAliasResponse(code apimodel.Code, alias *apiservice.ServiceAlias) *apiservice.Response {
	resp := NewResponse(code)
	resp.Alias = alias
	return resp
}

/**
 * @brief 创建回复带服务实例信息
 */
func NewInstanceResponse(code apimodel.Code, instance *apiservice.Instance) *apiservice.Response {
	return &apiservice.Response{
		Code:     &wrappers.UInt32Value{Value: uint32(code)},
		Info:     &wrappers.StringValue{Value: code2info[uint32(code)]},
		Instance: instance,
	}
}

// 创建带自定义error的服务实例response
func NewInstanceRespWithError(code apimodel.Code, err error, instance *apiservice.Instance) *apiservice.Response {
	resp := NewInstanceResponse(code, instance)
	resp.Info.Value += " : " + err.Error()

	return resp
}

/**
 * @brief 创建回复带服务路由信息
 */
func NewRoutingResponse(code apimodel.Code, routing *apitraffic.Routing) *apiservice.Response {
	return &apiservice.Response{
		Code:    &wrappers.UInt32Value{Value: uint32(code)},
		Info:    &wrappers.StringValue{Value: code2info[uint32(code)]},
		Routing: routing,
	}
}

// NewAnyDataResponse create the response with data with any type
func NewAnyDataResponse(code apimodel.Code, msg proto.Message) *apiservice.Response {
	ret, err := anypb.New(proto.MessageV2(msg))
	if err != nil {
		return NewResponse(code)
	}
	return &apiservice.Response{
		Code: &wrappers.UInt32Value{Value: uint32(code)},
		Info: &wrappers.StringValue{Value: code2info[uint32(code)]},
		Data: ret,
	}
}

// NewRouterResponse 创建带新版本路由的返回
func NewRouterResponse(code apimodel.Code, router *apitraffic.RouteRule) *apiservice.Response {
	return NewAnyDataResponse(code, router)
}

// NewRateLimitResponse 创建回复带限流规则信息
func NewRateLimitResponse(code apimodel.Code, rule *apitraffic.Rule) *apiservice.Response {
	return &apiservice.Response{
		Code:      &wrappers.UInt32Value{Value: uint32(code)},
		Info:      &wrappers.StringValue{Value: code2info[uint32(code)]},
		RateLimit: rule,
	}
}

/**
 * @brief 创建回复带熔断规则信息
 */
func NewCircuitBreakerResponse(code apimodel.Code, circuitBreaker *apifault.CircuitBreaker) *apiservice.Response {
	return &apiservice.Response{
		Code:           &wrappers.UInt32Value{Value: uint32(code)},
		Info:           &wrappers.StringValue{Value: code2info[uint32(code)]},
		CircuitBreaker: circuitBreaker,
	}
}

/**
 * @brief 创建批量回复
 */
func NewBatchWriteResponse(code apimodel.Code) *apiservice.BatchWriteResponse {
	return &apiservice.BatchWriteResponse{
		Code: &wrappers.UInt32Value{Value: uint32(code)},
		Info: &wrappers.StringValue{Value: code2info[uint32(code)]},
		Size: &wrappers.UInt32Value{Value: 0},
	}
}

/**
 * @brief 创建带详细信息的批量回复
 */
func NewBatchWriteResponseWithMsg(code apimodel.Code, msg string) *apiservice.BatchWriteResponse {
	resp := NewBatchWriteResponse(code)
	resp.Info.Value += ": " + msg
	return resp
}

// NewBatchQueryResponse create the batch query responses
func NewBatchQueryResponse(code apimodel.Code) *apiservice.BatchQueryResponse {
	return &apiservice.BatchQueryResponse{
		Code:   &wrappers.UInt32Value{Value: uint32(code)},
		Info:   &wrappers.StringValue{Value: code2info[uint32(code)]},
		Amount: &wrappers.UInt32Value{Value: 0},
		Size:   &wrappers.UInt32Value{Value: 0},
	}
}

// NewBatchQueryResponseWithMsg create the batch query responses with message
func NewBatchQueryResponseWithMsg(code apimodel.Code, msg string) *apiservice.BatchQueryResponse {
	resp := NewBatchQueryResponse(code)
	resp.Info.Value += ": " + msg
	return resp
}

// AddAnyDataIntoBatchQuery add message as any data array
func AddAnyDataIntoBatchQuery(resp *apiservice.BatchQueryResponse, message proto.Message) error {
	ret, err := anypb.New(proto.MessageV2(message))
	if err != nil {
		return err
	}
	resp.Data = append(resp.Data, ret)
	return nil
}

// 创建一个空白的discoverResponse
func NewDiscoverResponse(code apimodel.Code) *apiservice.DiscoverResponse {
	return &apiservice.DiscoverResponse{
		Code: &wrappers.UInt32Value{Value: uint32(code)},
		Info: &wrappers.StringValue{Value: code2info[uint32(code)]},
	}
}

/**
 * @brief 创建查询服务回复
 */
func NewDiscoverServiceResponse(code apimodel.Code, service *apiservice.Service) *apiservice.DiscoverResponse {
	return &apiservice.DiscoverResponse{
		Code:    &wrappers.UInt32Value{Value: uint32(code)},
		Info:    &wrappers.StringValue{Value: code2info[uint32(code)]},
		Type:    apiservice.DiscoverResponse_SERVICES,
		Service: service,
	}
}

/**
 * @brief 创建查询服务实例回复
 */
func NewDiscoverInstanceResponse(code apimodel.Code, service *apiservice.Service) *apiservice.DiscoverResponse {
	return &apiservice.DiscoverResponse{
		Code:    &wrappers.UInt32Value{Value: uint32(code)},
		Info:    &wrappers.StringValue{Value: code2info[uint32(code)]},
		Type:    apiservice.DiscoverResponse_INSTANCE,
		Service: service,
	}
}

/**
 * @brief 创建查询服务路由回复
 */
func NewDiscoverRoutingResponse(code apimodel.Code, service *apiservice.Service) *apiservice.DiscoverResponse {
	return &apiservice.DiscoverResponse{
		Code:    &wrappers.UInt32Value{Value: uint32(code)},
		Info:    &wrappers.StringValue{Value: code2info[uint32(code)]},
		Type:    apiservice.DiscoverResponse_ROUTING,
		Service: service,
	}
}

/**
 * @brief 创建查询限流规则回复
 */
func NewDiscoverRateLimitResponse(code apimodel.Code, service *apiservice.Service) *apiservice.DiscoverResponse {
	return &apiservice.DiscoverResponse{
		Code:    &wrappers.UInt32Value{Value: uint32(code)},
		Info:    &wrappers.StringValue{Value: code2info[uint32(code)]},
		Type:    apiservice.DiscoverResponse_RATE_LIMIT,
		Service: service,
	}
}

/**
 * @brief 创建查询熔断规则回复
 */
func NewDiscoverCircuitBreakerResponse(code apimodel.Code, service *apiservice.Service) *apiservice.DiscoverResponse {
	return &apiservice.DiscoverResponse{
		Code:    &wrappers.UInt32Value{Value: uint32(code)},
		Info:    &wrappers.StringValue{Value: code2info[uint32(code)]},
		Type:    apiservice.DiscoverResponse_CIRCUIT_BREAKER,
		Service: service,
	}
}

// NewDiscoverLaneResponse .
func NewDiscoverLaneResponse(code apimodel.Code, service *apiservice.Service) *apiservice.DiscoverResponse {
	return &apiservice.DiscoverResponse{
		Code:    &wrappers.UInt32Value{Value: uint32(code)},
		Info:    &wrappers.StringValue{Value: code2info[uint32(code)]},
		Type:    apiservice.DiscoverResponse_LANE,
		Service: service,
	}
}

/**
 * @brief 创建查询探测规则回复
 */
func NewDiscoverFaultDetectorResponse(code apimodel.Code, service *apiservice.Service) *apiservice.DiscoverResponse {
	return &apiservice.DiscoverResponse{
		Code:    &wrappers.UInt32Value{Value: uint32(code)},
		Info:    &wrappers.StringValue{Value: code2info[uint32(code)]},
		Type:    apiservice.DiscoverResponse_FAULT_DETECTOR,
		Service: service,
	}
}

// 创建一个空白的 ConfigDiscoverResponse
func NewConfigDiscoverResponse(code apimodel.Code) *apiconfig.ConfigDiscoverResponse {
	return &apiconfig.ConfigDiscoverResponse{
		Code: uint32(code),
		Info: code2info[uint32(code)],
	}
}

// 格式化responses
// batch操作
// 如果所有子错误码一致，那么使用子错误码
// 如果包含任意一个5xx，那么返回500
func FormatBatchWriteResponse(response *apiservice.BatchWriteResponse) *apiservice.BatchWriteResponse {
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
