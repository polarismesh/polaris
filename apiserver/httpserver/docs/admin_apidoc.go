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
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	"github.com/polarismesh/polaris/admin"
)

var (
	maintainApiTags = []string{"Maintain"}
)

func EnrichGetServerConnectionsApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("获取服务端连接数").
		Metadata(restfulspec.KeyOpenAPITags, maintainApiTags).
		Param(restful.QueryParameter("protocol", "查看指定协议").DataType(typeNameString).Required(true)).
		Param(restful.QueryParameter("host", "查看指定host").DataType(typeNameString).Required(false)).
		Notes(enrichGetServerConnectionsApiNotes)
}

func EnrichGetServerConnStatsApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("获取服务端连接统计信息").
		Metadata(restfulspec.KeyOpenAPITags, maintainApiTags).
		Param(restful.QueryParameter("protocol", "查看指定协议").DataType(typeNameString).Required(true)).
		Param(restful.QueryParameter("host", "查看指定host").DataType(typeNameString).Required(false)).
		Param(restful.QueryParameter("amount", "总数").DataType(typeNameInteger).Required(false)).
		Notes(enrichGetServerConnStatsApiNotes)
}

func EnrichCloseConnectionsApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("关闭指定client ip的连接").
		Metadata(restfulspec.KeyOpenAPITags, maintainApiTags).
		Reads([]admin.ConnReq{}).
		Notes(enrichCloseConnectionsApiNotes)
}

func EnrichFreeOSMemoryApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("释放系统内存").
		Metadata(restfulspec.KeyOpenAPITags, maintainApiTags).
		Notes(enrichFreeOSMemoryApiNotes)
}

func EnrichCleanInstanceApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("彻底清理flag=1的实例").
		Metadata(restfulspec.KeyOpenAPITags, maintainApiTags).
		Reads(apiservice.Instance{}).
		Notes(enrichCleanInstanceApiNotes)
}

func EnrichBatchCleanInstancesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("彻底清理flag=1的实例").
		Metadata(restfulspec.KeyOpenAPITags, maintainApiTags).
		Reads(apiservice.Instance{}).
		Notes(enrichBatchCleanInstancesApiNotes)
}

func EnrichGetLastHeartbeatApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("获取上一次心跳的时间").
		Metadata(restfulspec.KeyOpenAPITags, maintainApiTags).
		Param(restful.QueryParameter("id", "实例ID 如果存在则其它参数可不填").DataType(typeNameString).Required(false)).
		Param(restful.QueryParameter("service", "服务名").DataType(typeNameString).Required(false)).
		Param(restful.QueryParameter("namespace", "命名空间").DataType(typeNameString).Required(false)).
		Param(restful.QueryParameter("host", "主机名").DataType(typeNameString).Required(false)).
		Param(restful.QueryParameter("port", "端口").DataType(typeNameInteger).Required(false)).
		Param(restful.QueryParameter("vpv_id", "VPC ID").DataType(typeNameString).Required(false)).
		Notes(enrichGetLastHeartbeatApiNotes)
}

func EnrichGetLogOutputLevelApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("获取日志输出级别").
		Metadata(restfulspec.KeyOpenAPITags, maintainApiTags).
		Notes(enrichGetLogOutputLevelApiNotes)
}

func EnrichSetLogOutputLevelApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("设置日志输出级别").
		Metadata(restfulspec.KeyOpenAPITags, maintainApiTags).
		Notes(enrichSetLogOutputLevelApiNotes)
}

func EnrichListLeaderElectionsApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("获取选主的结果").
		Metadata(restfulspec.KeyOpenAPITags, maintainApiTags).
		Notes(enrichListLeaderElectionsApiNotes)
}

func EnrichReleaseLeaderElectionApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("主动放弃主身份").
		Metadata(restfulspec.KeyOpenAPITags, maintainApiTags).
		Notes(enrichReleaseLeaderElectionApiNotes)
}

func EnrichGetCMDBInfoApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("查询CMDB信息").
		Metadata(restfulspec.KeyOpenAPITags, maintainApiTags).
		Notes(enrichReleaseLeaderElectionApiNotes)
}

func EnrichGetReportClientsApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("查询SDK实例列表").
		Metadata(restfulspec.KeyOpenAPITags, maintainApiTags).
		Notes(enrichReleaseLeaderElectionApiNotes)
}
