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

	api "github.com/polarismesh/polaris/common/api/v1"
)

var (
	configConsoleApiTags = []string{"ConfigConsole"}
	configClientApiTags  = []string{"ConfigClient"}
)

func enrichCreateConfigFileGroupApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("创建配置文件组").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Reads(api.ConfigFileGroup{}, "开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header\nHeader X-Polaris-Token: {访问凭据}\n ```\n{\n    \"name\":\"someGroup\",\n    \"namespace\":\"someNamespace\",\n    \"comment\":\"some comment\",\n    \"createBy\":\"ledou\"\n}\n```")
}

func enrichQueryConfigFileGroupsApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("搜索配置文件组").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Param(restful.QueryParameter("namespace", "命名空间，不填表示全部命名空间").DataType("string").Required(false)).
		Param(restful.QueryParameter("group", "配置文件分组名，模糊搜索").DataType("string").Required(false)).
		Param(restful.QueryParameter("fileName", "配置文件名称，模糊搜索").DataType("string").Required(false)).
		Param(restful.QueryParameter("offset", "翻页偏移量 默认为 0").DataType("integer").Required(false).DefaultValue("0")).
		Param(restful.QueryParameter("limit", "一页大小，最大为 100").DataType("integer").Required(true).DefaultValue("100"))
}

func enrichDeleteConfigFileGroupApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("删除配置文件").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Param(restful.QueryParameter("namespace", "命名空间").DataType("string").Required(true)).
		Param(restful.QueryParameter("group", "配置文件分组").DataType("string").Required(true))
}

func enrichUpdateConfigFileGroupApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("更新配置文件组").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Reads(api.ConfigFileGroup{}, "开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header\nHeader X-Polaris-Token: {访问凭据}\n ```\n{\n    \"name\":\"someGroup\",\n    \"namespace\":\"someNamespace\",\n    \"comment\":\"some comment\",\n    \"createBy\":\"ledou\"\n}\n```")
}

func enrichCreateConfigFileApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("创建配置文件").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Reads(api.ConfigFile{}, "开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header\nHeader X-Polaris-Token: {访问凭据}\n ```{\n    \"name\":\"application.properties\",\n    \"namespace\":\"someNamespace\",\n    \"group\":\"someGroup\",\n    \"content\":\"redis.cache.age=10\",\n    \"comment\":\"第一个配置文件\",\n    \"tags\":[{\"key\":\"service\", \"value\":\"helloService\"}],\n    \"createBy\":\"ledou\",\n    \"format\":\"properties\"\n}\n```\n")
}

func enrichGetConfigFileApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("拉取配置").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Param(restful.QueryParameter("namespace", "命名空间").DataType("string").Required(true)).
		Param(restful.QueryParameter("group", "配置文件分组").DataType("string").Required(true)).
		Param(restful.QueryParameter("name", "配置文件名").DataType("string").Required(true))
}

func enrichQueryConfigFilesByGroupApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("搜索配置文件").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Param(restful.QueryParameter("namespace", "命名空间").DataType("string").Required(false)).
		Param(restful.QueryParameter("group", "配置文件分组").DataType("string").Required(false)).
		Param(restful.QueryParameter("offset", "翻页偏移量 默认为 0").DataType("integer").Required(false).DefaultValue("0")).
		Param(restful.QueryParameter("limit", "一页大小，最大为 100").DataType("integer").Required(true).DefaultValue("100"))
}

func enrichSearchConfigFileApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("搜索配置文件").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Param(restful.QueryParameter("namespace", "命名空间").DataType("string").Required(false)).
		Param(restful.QueryParameter("group", "配置文件分组").DataType("string").Required(false)).
		Param(restful.QueryParameter("name", "配置文件").DataType("string").Required(false)).
		Param(restful.QueryParameter("tags", "格式：key1,value1,key2,value2").DataType("string").Required(false)).
		Param(restful.QueryParameter("offset", "翻页偏移量 默认为 0").DataType("integer").Required(false).DefaultValue("0")).
		Param(restful.QueryParameter("limit", "一页大小，最大为 100").DataType("integer").Required(true).DefaultValue("100"))
}

func enrichUpdateConfigFileApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("创建配置文件").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Reads(api.ConfigFile{}, "开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header\nHeader X-Polaris-Token: {访问凭据}\n ```{\n    \"name\":\"application.properties\",\n    \"namespace\":\"someNamespace\",\n    \"group\":\"someGroup\",\n    \"content\":\"redis.cache.age=10\",\n    \"comment\":\"第一个配置文件\",\n    \"tags\":[{\"key\":\"service\", \"value\":\"helloService\"}],\n    \"createBy\":\"ledou\",\n    \"format\":\"properties\"\n}\n```\n")
}

func enrichDeleteConfigFileApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("创建配置文件").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Param(restful.QueryParameter("namespace", "命名空间").DataType("string").Required(true)).
		Param(restful.QueryParameter("group", "配置文件分组").DataType("string").Required(true)).
		Param(restful.QueryParameter("name", "配置文件").DataType("string").Required(true)).
		Param(restful.QueryParameter("deleteBy", "操作人").DataType("string").Required(false))
}

func enrichBatchDeleteConfigFileApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("批量删除配置文件").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Param(restful.QueryParameter("deleteBy", "操作人").DataType("string").Required(false)).
		Reads(api.ConfigFile{}, "开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header\nHeader X-Polaris-Token: {访问凭据}\n```[\n     {\n         \"name\":\"application.properties\",\n         \"namespace\":\"someNamespace\",\n         \"group\":\"someGroup\"\n     }\n]\n```")
}

func enrichPublishConfigFileApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("发布配置文件").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Reads(api.ConfigFileRelease{}, "开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header\nHeader X-Polaris-Token: {访问凭据}\n```{\n    \"name\":\"release-002\",\n    \"fileName\":\"application.properties\",\n    \"namespace\":\"someNamespace\",\n    \"group\":\"someGroup\",\n    \"comment\":\"发布第一个配置文件\",\n    \"createBy\":\"ledou\"\n}\n```")
}

func enrichGetConfigFileReleaseApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("获取配置文件最后一次全量发布信息").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Param(restful.QueryParameter("namespace", "命名空间").DataType("string").Required(true)).
		Param(restful.QueryParameter("group", "配置文件分组").DataType("string").Required(true)).
		Param(restful.QueryParameter("name", "配置文件").DataType("string").Required(true))
}

func enrichGetConfigFileReleaseHistoryApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("获取配置文件发布历史记录").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Param(restful.QueryParameter("namespace", "命名空间").DataType("string").Required(true)).
		Param(restful.QueryParameter("group", "配置文件分组").DataType("string").Required(false)).
		Param(restful.QueryParameter("name", "配置文件").DataType("string").Required(false)).
		Param(restful.QueryParameter("offset", "翻页偏移量 默认为 0").DataType("integer").Required(false).DefaultValue("0")).
		Param(restful.QueryParameter("limit", "一页大小，最大为 100").DataType("integer").Required(true).DefaultValue("100"))
}

func enrichGetAllConfigFileTemplatesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("获取配置模板").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags)
}

func enrichCreateConfigFileTemplateApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("创建配置模板").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags)
}

func enrichGetConfigFileForClientApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("拉取配置").
		Metadata(restfulspec.KeyOpenAPITags, configClientApiTags).
		Param(restful.QueryParameter("namespace", "命名空间").DataType("string").Required(true)).
		Param(restful.QueryParameter("group", "配置文件分组").DataType("string").Required(true)).
		Param(restful.QueryParameter("fileName", "配置文件名").DataType("string").Required(true)).
		Param(restful.QueryParameter("version", "配置文件客户端版本号，刚启动时设置为 0").DataType("integer").Required(true))
}

func enrichWatchConfigFileForClientApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("监听配置").
		Metadata(restfulspec.KeyOpenAPITags, configClientApiTags).
		Reads(api.ClientWatchConfigFileRequest{}, "通过 Http LongPolling 机制订阅配置变更。")
}
