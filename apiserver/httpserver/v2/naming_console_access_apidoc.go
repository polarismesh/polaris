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
	"github.com/emicklei/go-restful/v3"
	restfulspec "github.com/polarismesh/go-restful-openapi/v2"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
)

var (
	routingRulesApiTags = []string{"RoutingRules"}
)

func enrichCreateRoutingsApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("创建路由规则(V2)").
		Metadata(restfulspec.KeyOpenAPITags, routingRulesApiTags).
		Operation("v2CreateRoutings").
		Reads([]apitraffic.RouteRule{}).
		Notes(enrichCreateRoutingsApiNotes)
}

func enrichDeleteRoutingsApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("删除路由规则(V2)").
		Metadata(restfulspec.KeyOpenAPITags, routingRulesApiTags).
		Operation("v2DeleteRoutings").
		Notes(enrichDeleteRoutingsApiNotes)
}

func enrichUpdateRoutingsApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("更新路由规则(V2)").
		Metadata(restfulspec.KeyOpenAPITags, routingRulesApiTags).
		Operation("v2UpdateRoutings").
		Reads([]apitraffic.RouteRule{}).
		Notes(enrichUpdateRoutingsApiNotes)
}

func enrichGetRoutingsApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("获取路由规则(V2)").
		Metadata(restfulspec.KeyOpenAPITags, routingRulesApiTags).
		Operation("v2GetRoutings").
		Notes(enrichGetRoutingsApiNotes)
}

func enrichEnableRoutingsApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.Doc("启用路由规则(V2)").
		Metadata(restfulspec.KeyOpenAPITags, routingRulesApiTags).
		Operation("v2EnableRoutings").
		Notes(enrichEnableRoutingsApiNotes)
}
