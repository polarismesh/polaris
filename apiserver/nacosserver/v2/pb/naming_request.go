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
	fmt "fmt"
	"strconv"
	"time"

	"github.com/polarismesh/polaris/apiserver/nacosserver/model"
)

// NamingRequest
type NamingRequest struct {
	*Request
	Namespace   string `json:"namespace"`
	ServiceName string `json:"serviceName"`
	GroupName   string `json:"groupName"`
	Module      string `json:"module"`
}

func (n *NamingRequest) RequestMeta() interface{} {
	return n
}

// NewNamingRequest
func NewNamingRequest() *NamingRequest {
	request := &Request{
		Headers:   make(map[string]string, 8),
		RequestId: "",
	}
	return &NamingRequest{
		Request:     request,
		Namespace:   model.DefaultNacosNamespace,
		ServiceName: "",
		GroupName:   model.DefaultServiceGroup,
		Module:      "naming",
	}
}

func NewBasicNamingRequest(requestId, namespace, serviceName, groupName string) *NamingRequest {
	request := &Request{
		Headers:   make(map[string]string, 8),
		RequestId: requestId,
	}
	return &NamingRequest{
		Request:     request,
		Namespace:   namespace,
		ServiceName: serviceName,
		GroupName:   groupName,
		Module:      "naming",
	}
}

func (r *NamingRequest) GetStringToSign() string {
	data := strconv.FormatInt(time.Now().Unix()*1000, 10)
	if r.ServiceName != "" || r.GroupName != "" {
		data = fmt.Sprintf("%s@@%s@@%s", data, r.GroupName, r.ServiceName)
	}
	return data
}

// InstanceRequest
type InstanceRequest struct {
	*NamingRequest
	Type     string         `json:"type"`
	Instance model.Instance `json:"instance"`
}

func (n *InstanceRequest) RequestMeta() interface{} {
	return n.NamingRequest
}

// NewInstanceRequest
func NewInstanceRequest() *InstanceRequest {
	return &InstanceRequest{
		NamingRequest: NewNamingRequest(),
		Type:          TypeInstanceRequest,
		Instance:      model.Instance{},
	}
}

func (r *InstanceRequest) GetRequestType() string {
	return TypeInstanceRequest
}

type PersistentInstanceRequest struct {
	*NamingRequest
	Type     string         `json:"type"`
	Instance model.Instance `json:"instance"`
}

func (n *PersistentInstanceRequest) RequestMeta() interface{} {
	return n.NamingRequest
}

// NewInstanceRequest
func NewPersistentInstanceRequest() *PersistentInstanceRequest {
	return &PersistentInstanceRequest{
		NamingRequest: NewNamingRequest(),
		Type:          TypePersistentInstanceRequest,
		Instance:      model.Instance{},
	}
}

func (r *PersistentInstanceRequest) GetRequestType() string {
	return TypePersistentInstanceRequest
}

// BatchInstanceRequest .
type BatchInstanceRequest struct {
	*NamingRequest
	Type      string            `json:"type"`
	Instances []*model.Instance `json:"instances"`
}

func (n *BatchInstanceRequest) RequestMeta() interface{} {
	return n.NamingRequest
}

func NewBatchInstanceRequest() *BatchInstanceRequest {
	return &BatchInstanceRequest{
		NamingRequest: NewNamingRequest(),
		Type:          TypeBatchInstanceRequest,
		Instances:     make([]*model.Instance, 0),
	}
}

func (r *BatchInstanceRequest) GetRequestType() string {
	return TypeBatchInstanceRequest
}

func (r *BatchInstanceRequest) Normalize() {
	for i := range r.Instances {
		ins := r.Instances[i]
		if len(ins.ServiceName) == 0 {
			ins.ServiceName = r.ServiceName
		}
	}
}

// NotifySubscriberRequest
type NotifySubscriberRequest struct {
	*NamingRequest
	ServiceInfo *model.ServiceInfo `json:"serviceInfo"`
}

func NewNotifySubscriberRequest() *NotifySubscriberRequest {
	return &NotifySubscriberRequest{
		NamingRequest: NewNamingRequest(),
		ServiceInfo:   &model.ServiceInfo{},
	}
}

func (r *NotifySubscriberRequest) GetRequestType() string {
	return TypeNotifySubscriberRequest
}

// SubscribeServiceRequest
type SubscribeServiceRequest struct {
	*NamingRequest
	Subscribe bool   `json:"subscribe"`
	Clusters  string `json:"clusters"`
}

func (n *SubscribeServiceRequest) RequestMeta() interface{} {
	return n.NamingRequest
}

// NewSubscribeServiceRequest .
func NewSubscribeServiceRequest() *SubscribeServiceRequest {
	return &SubscribeServiceRequest{
		NamingRequest: NewNamingRequest(),
		Subscribe:     true,
		Clusters:      "",
	}
}

func (r *SubscribeServiceRequest) GetRequestType() string {
	return TypeSubscribeServiceRequest
}

// ServiceListRequest
type ServiceListRequest struct {
	*NamingRequest
	PageNo   int    `json:"pageNo"`
	PageSize int    `json:"pageSize"`
	Selector string `json:"selector"`
}

func (n *ServiceListRequest) RequestMeta() interface{} {
	return n.NamingRequest
}

// NewServiceListRequest .
func NewServiceListRequest() *ServiceListRequest {
	return &ServiceListRequest{
		NamingRequest: NewNamingRequest(),
		PageNo:        0,
		PageSize:      10,
		Selector:      "",
	}
}

func (r *ServiceListRequest) GetRequestType() string {
	return TypeServiceListRequest
}

// ServiceQueryRequest
type ServiceQueryRequest struct {
	*NamingRequest
	Cluster     string `json:"cluster"`
	HealthyOnly bool   `json:"healthyOnly"`
	UdpPort     int    `json:"udpPort"`
}

func (n *ServiceQueryRequest) RequestMeta() interface{} {
	return n.NamingRequest
}

// NewServiceQueryRequest .
func NewServiceQueryRequest() *ServiceQueryRequest {
	return &ServiceQueryRequest{
		NamingRequest: NewNamingRequest(),
		Cluster:       "",
		HealthyOnly:   false,
		UdpPort:       0,
	}
}

func (r *ServiceQueryRequest) GetRequestType() string {
	return TypeServiceQueryRequest
}
