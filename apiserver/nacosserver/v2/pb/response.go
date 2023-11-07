/*
 * Copyright 1999-2020 Alibaba Group Holding Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package nacos_grpc_service

import (
	"encoding/json"

	"github.com/polarismesh/polaris/apiserver/nacosserver/model"
)

const (
	TypeConnectResetResponse       = "ConnectResetResponse"
	TypeClientDetectionResponse    = "ClientDetectionResponse"
	TypeServerCheckResponse        = "ServerCheckResponse"
	TypeInstanceResponse           = "InstanceResponse"
	TypeBatchInstanceResponse      = "BatchInstanceResponse"
	TypeQueryServiceResponse       = "QueryServiceResponse"
	TypeSubscribeServiceResponse   = "SubscribeServiceResponse"
	TypeServiceListResponse        = "ServiceListResponse"
	TypeNotifySubscriberResponse   = "NotifySubscriberResponse"
	TypeHealthCheckResponse        = "HealthCheckResponse"
	TypeErrorResponse              = "ErrorResponse"
	TypeConfigChangeNotifyResponse = "ConfigChangeNotifyResponse"
)

// BaseResponse
type BaseResponse interface {
	GetResponseType() string
	SetRequestId(requestId string)
	GetRequestId() string
	GetBody() string
	GetErrorCode() int
	IsSuccess() bool
	GetResultCode() int
	GetMessage() string
}

// Response
type Response struct {
	ResultCode int    `json:"resultCode"`
	ErrorCode  int    `json:"errorCode"`
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	RequestId  string `json:"requestId"`
}

func (r *Response) GetRequestId() string {
	return r.RequestId
}

func (r *Response) SetRequestId(requestId string) {
	r.RequestId = requestId
}

func (r *Response) GetBody() string {
	//nolint:errchkjson
	data, _ := json.Marshal(r)
	return string(data)
}

func (r *Response) IsSuccess() bool {
	return r.Success
}

func (r *Response) GetErrorCode() int {
	return r.ErrorCode
}

func (r *Response) GetResultCode() int {
	return r.ResultCode
}

func (r *Response) GetMessage() string {
	return r.Message
}

// ConnectResetResponse
type ConnectResetResponse struct {
	*Response
}

func (c *ConnectResetResponse) GetResponseType() string {
	return "ConnectResetResponse"
}

// ClientDetectionResponse
type ClientDetectionResponse struct {
	*Response
}

func (c *ClientDetectionResponse) GetResponseType() string {
	return "ClientDetectionResponse"
}

// NewServerCheckResponse
func NewServerCheckResponse() *ServerCheckResponse {
	return &ServerCheckResponse{
		Response: &Response{
			ResultCode: int(model.Response_Success.Code),
			ErrorCode:  int(model.ErrorCode_Success.Code),
			Success:    true,
			Message:    "success",
			RequestId:  "",
		},
		ConnectionId: "",
	}
}

// ServerCheckResponse
type ServerCheckResponse struct {
	*Response
	ConnectionId string `json:"connectionId"`
}

func (c *ServerCheckResponse) GetResponseType() string {
	return TypeServerCheckResponse
}

// InstanceResponse
type InstanceResponse struct {
	*Response
	Type string `json:"type"`
}

func (c *InstanceResponse) GetResponseType() string {
	return TypeInstanceResponse
}

// BatchInstanceResponse
type BatchInstanceResponse struct {
	*Response
	Type string `json:"type"`
}

func (c *BatchInstanceResponse) GetResponseType() string {
	return TypeBatchInstanceResponse
}

// QueryServiceResponse
type QueryServiceResponse struct {
	*Response
	ServiceInfo model.Service `json:"serviceInfo"`
}

func (c *QueryServiceResponse) GetResponseType() string {
	return TypeQueryServiceResponse
}

// SubscribeServiceResponse
type SubscribeServiceResponse struct {
	*Response
	ServiceInfo model.ServiceInfo `json:"serviceInfo"`
}

func (c *SubscribeServiceResponse) GetResponseType() string {
	return TypeSubscribeServiceResponse
}

// ServiceListResponse
type ServiceListResponse struct {
	*Response
	Count        int      `json:"count"`
	ServiceNames []string `json:"serviceNames"`
}

func (c *ServiceListResponse) GetResponseType() string {
	return TypeServiceListResponse
}

// NotifySubscriberResponse
type NotifySubscriberResponse struct {
	*Response
}

func (c *NotifySubscriberResponse) GetResponseType() string {
	return TypeNotifySubscriberResponse
}

// NewHealthCheckResponse
func NewHealthCheckResponse() *HealthCheckResponse {
	return &HealthCheckResponse{
		Response: &Response{
			ResultCode: int(model.Response_Success.Code),
			ErrorCode:  int(model.ErrorCode_Success.Code),
			Success:    true,
			Message:    "success",
			RequestId:  "",
		},
	}
}

// HealthCheckResponse
type HealthCheckResponse struct {
	*Response
}

func (c *HealthCheckResponse) GetResponseType() string {
	return TypeHealthCheckResponse
}

// ErrorResponse
type ErrorResponse struct {
	*Response
}

func (c *ErrorResponse) GetResponseType() string {
	return TypeErrorResponse
}

type ConfigChangeBatchListenResponse struct {
	*Response
	ChangedConfigs []ConfigContext `json:"changedConfigs"`
}

func NewConfigChangeBatchListenResponse() *ConfigChangeBatchListenResponse {
	return &ConfigChangeBatchListenResponse{
		Response: &Response{
			ResultCode: int(model.Response_Success.Code),
			Success:    true,
			Message:    model.Response_Success.Desc,
		},
		ChangedConfigs: make([]ConfigContext, 0, 4),
	}
}

func (c *ConfigChangeBatchListenResponse) GetResponseType() string {
	return "ConfigChangeBatchListenResponse"
}

type ConfigQueryResponse struct {
	*Response
	Content          string `json:"content"`
	EncryptedDataKey string `json:"encryptedDataKey"`
	ContentType      string `json:"contentType"`
	Md5              string `json:"md5"`
	LastModified     int64  `json:"lastModified"`
	IsBeta           bool   `json:"isBeta"`
	Tag              bool   `json:"tag"`
}

func (c *ConfigQueryResponse) GetResponseType() string {
	return "ConfigQueryResponse"
}

type ConfigPublishResponse struct {
	*Response
}

func (c *ConfigPublishResponse) GetResponseType() string {
	return "ConfigPublishResponse"
}

type ConfigRemoveResponse struct {
	*Response
}

func (c *ConfigRemoveResponse) GetResponseType() string {
	return "ConfigRemoveResponse"
}

type ConfigChangeNotifyResponse struct {
	*Response
}

func (c *ConfigChangeNotifyResponse) GetResponseType() string {
	return "ConfigChangeNotifyResponse"
}
