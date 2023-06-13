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
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
)

var (
	namespaceApiTags = []string{"Namespaces"}
)

func EnrichGetNamespacesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("查询命名空间列表(New)").
		Metadata(restfulspec.KeyOpenAPITags, namespaceApiTags).
		Param(restful.QueryParameter("name", "命名空间唯一名称").
			DataType(typeNameString).Required(true)).
		Param(restful.QueryParameter("offset", "查询偏移量").
			DataType(typeNameInteger).Required(false).DefaultValue("0")).
		Param(restful.QueryParameter("limit", "查询条数，**最多查询100条**").
			DataType(typeNameInteger).Required(false)).
		Returns(0, "", struct {
			BatchQueryResponse
			Namespaces []apimodel.Namespace `json:"namespaces"`
		}{})
}

func EnrichCreateNamespacesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("创建命名空间(New)").
		Metadata(restfulspec.KeyOpenAPITags, namespaceApiTags).
		Reads([]apimodel.Namespace{}, "create namespaces").
		Returns(0, "", struct {
			BatchWriteResponse
			Responses []struct {
				BaseResponse
				Namespace apimodel.Namespace `json:"namespace"`
			} `json:"responses"`
		}{})
}

func EnrichDeleteNamespacesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("删除命名空间(New)").
		Metadata(restfulspec.KeyOpenAPITags, namespaceApiTags).
		Reads([]apimodel.Namespace{}, "delete namespaces").
		Returns(0, "", struct {
			BatchWriteResponse
			Responses []struct {
				BaseResponse
				Namespace apimodel.Namespace `json:"namespace"`
			} `json:"responses"`
		}{})
}

func EnrichUpdateNamespacesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("更新命名空间(New)").
		Metadata(restfulspec.KeyOpenAPITags, namespaceApiTags).
		Reads([]apimodel.Namespace{}, "update namespaces").
		Returns(0, "", struct {
			BatchWriteResponse
			Responses []struct {
				BaseResponse
				Namespace apimodel.Namespace `json:"namespace"`
			} `json:"responses"`
		}{})
}
