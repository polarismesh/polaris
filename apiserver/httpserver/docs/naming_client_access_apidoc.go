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

package docs

import (
	"github.com/emicklei/go-restful/v3"
	restfulspec "github.com/polarismesh/go-restful-openapi/v2"
	"github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
)

var (
	registerInstanceApiTags = []string{"Client"}
)

func EnrichReportClientApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("上报客户端信息").
		Metadata(restfulspec.KeyOpenAPITags, registerInstanceApiTags).
		Doc("上报客户端").
		Reads(apiservice.Client{}).
		Returns(0, "", struct {
			BaseResponse
			Client service_manage.Client `json:"client,omitempty"`
		}{})
}

func EnrichRegisterInstanceApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("注册实例").
		Metadata(restfulspec.KeyOpenAPITags, registerInstanceApiTags).
		Reads(apiservice.Instance{}).
		Returns(0, "", service_manage.Instance{})
}

func EnrichDeregisterInstanceApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("注销实例").
		Metadata(restfulspec.KeyOpenAPITags, registerInstanceApiTags).
		Reads(apiservice.Instance{}).
		Returns(0, "", service_manage.Instance{})
}

func EnrichHeartbeatApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("上报心跳").
		Metadata(restfulspec.KeyOpenAPITags, registerInstanceApiTags).
		Reads(apiservice.Instance{}).
		Returns(0, "", service_manage.Instance{})
}

func EnrichDiscoverApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("服务发现").
		Metadata(restfulspec.KeyOpenAPITags, registerInstanceApiTags).
		Reads(DiscoverRequest{},
			"Type 支持 [UNKNOWN/INSTANCE/ROUTING/RATE_LIMIT/CIRCUIT_BREAKER/SERVICES/NAMESPACES/FAULT_DETECTOR]").
		Returns(0, "", service_manage.DiscoverResponse{})
}
