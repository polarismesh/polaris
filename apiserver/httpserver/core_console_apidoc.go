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

package httpserver

import (
	"github.com/emicklei/go-restful/v3"
	restfulspec "github.com/polarismesh/go-restful-openapi/v2"

	v1 "github.com/polarismesh/polaris/common/api/v1"
)

var (
	namespaceApiTags = []string{"Namespaces"}
)

func enrichGetNamespacesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("查询命名空间列表").
		Metadata(restfulspec.KeyOpenAPITags, namespaceApiTags).
		Param(restful.QueryParameter("name", "命名空间唯一名称").DataType("string").Required(true)).
		Param(restful.QueryParameter("offset", "查询偏移量").DataType("integer").Required(false).DefaultValue("0")).
		Param(restful.QueryParameter("limit", "查询条数，**最多查询100条**").DataType("integer").Required(false)).
		Notes(enrichGetNamespacesApiNotes)
}

func enrichCreateNamespacesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("创建命名空间").
		Metadata(restfulspec.KeyOpenAPITags, namespaceApiTags).
		Reads([]v1.Namespace{}, "create namespaces").
		Notes(enrichCreateNamespacesApiNotes)
}

func enrichDeleteNamespacesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("删除命名空间").
		Metadata(restfulspec.KeyOpenAPITags, namespaceApiTags).
		Reads([]v1.Namespace{}, "delete namespaces").
		Notes(enrichDeleteNamespacesApiNotes)
}

func enrichUpdateNamespacesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("更新命名空间").
		Metadata(restfulspec.KeyOpenAPITags, namespaceApiTags).
		Reads([]v1.Namespace{}, "update namespaces").
		Notes(enrichUpdateNamespacesApiNotes)
}

func enrichGetNamespaceTokenApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("查询命名空间Token").
		Metadata(restfulspec.KeyOpenAPITags, namespaceApiTags).Deprecate()
}

func enrichUpdateNamespaceTokenApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("更新命名空间Token").
		Metadata(restfulspec.KeyOpenAPITags, namespaceApiTags).Deprecate()
}
