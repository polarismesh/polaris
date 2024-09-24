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
)

const (
	TypeConnectionSetupRequest    = "ConnectionSetupRequest"
	TypeConnectResetRequest       = "ConnectResetRequest"
	TypeClientDetectionRequest    = "ClientDetectionRequest"
	TypeHealthCheckRequest        = "HealthCheckRequest"
	TypeServerCheckRequest        = "ServerCheckRequest"
	TypeInstanceRequest           = "InstanceRequest"
	TypePersistentInstanceRequest = "PersistentInstanceRequest"
	TypeBatchInstanceRequest      = "BatchInstanceRequest"
	TypeNotifySubscriberRequest   = "NotifySubscriberRequest"
	TypeSubscribeServiceRequest   = "SubscribeServiceRequest"
	TypeServiceListRequest        = "ServiceListRequest"
	TypeServiceQueryRequest       = "ServiceQueryRequest"
	TypeConfigPublishRequest      = "ConfigPublishRequest"
	TypeConfigQueryRequest        = "ConfigQueryRequest"
	TypeConfigRemoveRequest       = "ConfigRemoveRequest"
	TypeConfigBatchListenRequest  = "ConfigBatchListenRequest"
)

// CustomerPayload
type CustomerPayload interface{}

// RequestMeta
type RequestMeta struct {
	ConnectionID  string
	ClientIP      string
	ClientVersion string
	Labels        map[string]string
}

// Request
type Request struct {
	Headers   map[string]string `json:"-"`
	RequestId string            `json:"requestId"`
}

// BaseRequest
type BaseRequest interface {
	GetHeaders() map[string]string
	GetRequestType() string
	GetBody(request BaseRequest) string
	PutAllHeaders(headers map[string]string)
	GetRequestId() string
	GetStringToSign() string
}

func (r *Request) PutAllHeaders(headers map[string]string) {
	if r.Headers == nil {
		r.Headers = make(map[string]string)
	}
	for k, v := range headers {
		r.Headers[k] = v
	}
}

func (r *Request) ClearHeaders() {
	r.Headers = make(map[string]string)
}

func (r *Request) GetHeaders() map[string]string {
	if r.Headers == nil {
		r.Headers = make(map[string]string)
	}
	return r.Headers
}

func (r *Request) GetBody(request BaseRequest) string {
	//nolint:errchkjson
	js, _ := json.Marshal(request)
	return string(js)
}

func (r *Request) GetRequestId() string {
	return r.RequestId
}

func (r *Request) GetStringToSign() string {
	return ""
}
