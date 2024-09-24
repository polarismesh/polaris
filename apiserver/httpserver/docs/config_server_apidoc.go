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
	"github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
)

var (
	configConsoleApiTags = []string{"ConfigConsole"}
	configClientApiTags  = []string{"Client"}
)

func EnrichCreateConfigFileGroupApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("创建配置文件组").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Reads(apiconfig.ConfigFileGroup{}).
		Returns(0, "", BaseResponse{})
}

func EnrichQueryConfigFileGroupsApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("搜索配置文件组").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Param(restful.QueryParameter("namespace", "命名空间，不填表示全部命名空间").
			DataType(typeNameString).Required(false)).
		Param(restful.QueryParameter("group", "配置文件分组名，模糊搜索").
			DataType(typeNameString).Required(false)).
		Param(restful.QueryParameter("fileName", "配置文件名称，模糊搜索").
			DataType(typeNameString).Required(false)).
		Param(restful.QueryParameter("offset", "翻页偏移量 默认为 0").
			DataType(typeNameInteger).
			Required(false).DefaultValue("0")).
		Param(restful.QueryParameter("limit", "一页大小，最大为 100").
			DataType(typeNameInteger).
			Required(true).DefaultValue("100")).
		Returns(0, "", struct {
			BatchQueryResponse
			ConfigFileGroups []config_manage.ConfigFileGroup `json:"configFileGroups,omitempty"`
		}{})
}

func EnrichDeleteConfigFileGroupApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("删除配置文件组").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Param(restful.QueryParameter("namespace", "命名空间").DataType(typeNameString).Required(true)).
		Param(restful.QueryParameter("group", "配置文件分组").DataType(typeNameString).Required(true)).
		Returns(0, "", BaseResponse{})
}

func EnrichUpdateConfigFileGroupApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("更新配置文件组").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Reads(apiconfig.ConfigFileGroup{}).
		Returns(0, "", BaseResponse{})
}

func EnrichCreateConfigFileApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("创建配置文件").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Reads(apiconfig.ConfigFile{}).
		Returns(0, "", BaseResponse{})
}

func EnrichUpsertAndReleaseConfigFileApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("创建/更新并发布配置文件").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Reads(apiconfig.ConfigFilePublishInfo{}).
		Returns(0, "", BaseResponse{})
}

func EnrichStopBetaReleaseConfigFileApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("停止灰度发布配置文件").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Reads(apiconfig.ConfigFileRelease{}).
		Returns(0, "", BaseResponse{})
}

func EnrichGetConfigFileApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("拉取配置").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Param(restful.QueryParameter("namespace", "命名空间").DataType(typeNameString).Required(true)).
		Param(restful.QueryParameter("group", "配置文件分组").DataType(typeNameString).Required(true)).
		Param(restful.QueryParameter("name", "配置文件名").DataType(typeNameString).Required(true)).
		Returns(0, "", struct {
			BaseResponse
			ConfigFile config_manage.ConfigFile `json:"configFile,omitempty"`
		}{})
}

func EnrichQueryConfigFilesByGroupApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("搜索配置文件").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Param(restful.QueryParameter("namespace", "命名空间").DataType(typeNameString).Required(false)).
		Param(restful.QueryParameter("group", "配置文件分组").DataType(typeNameString).Required(false)).
		Param(restful.QueryParameter("offset", "翻页偏移量 默认为 0").DataType(typeNameInteger).
			Required(false).DefaultValue("0")).
		Param(restful.QueryParameter("limit", "一页大小，最大为 100").DataType(typeNameInteger).
			Required(true).DefaultValue("100")).
		Returns(0, "", struct {
			BatchQueryResponse
			ConfigFiles []config_manage.ConfigFile `json:"configFiles,omitempty"`
		}{})
}

func EnrichSearchConfigFileApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("搜索配置文件").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Param(restful.QueryParameter("namespace", "命名空间").DataType(typeNameString).Required(false)).
		Param(restful.QueryParameter("group", "配置文件分组").DataType(typeNameString).Required(false)).
		Param(restful.QueryParameter("name", "配置文件").DataType(typeNameString).Required(false)).
		Param(restful.QueryParameter("tags", "格式：key1,value1,key2,value2").DataType(typeNameString).Required(false)).
		Param(restful.QueryParameter("offset", "翻页偏移量 默认为 0").DataType(typeNameInteger).
			Required(false).DefaultValue("0")).
		Param(restful.QueryParameter("limit", "一页大小，最大为 100").DataType(typeNameInteger).
			Required(true).DefaultValue("100")).
		Returns(0, "", struct {
			BatchQueryResponse
			ConfigFiles []config_manage.ConfigFile `json:"configFiles,omitempty"`
		}{})
}

func EnrichUpdateConfigFileApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("更新配置文件").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Reads(apiconfig.ConfigFile{}).
		Returns(0, "", BaseResponse{})
}

func EnrichDeleteConfigFileApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("删除配置文件").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Param(restful.QueryParameter("namespace", "命名空间").DataType(typeNameString).Required(true)).
		Param(restful.QueryParameter("group", "配置文件分组").DataType(typeNameString).Required(true)).
		Param(restful.QueryParameter("name", "配置文件").DataType(typeNameString).Required(true)).
		Param(restful.QueryParameter("deleteBy", "操作人").DataType(typeNameString).Required(false)).
		Returns(0, "", BaseResponse{})
}

func EnrichBatchDeleteConfigFileApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("批量删除配置文件").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Param(restful.QueryParameter("deleteBy", "操作人").DataType(typeNameString).Required(false)).
		Reads([]apiconfig.ConfigFile{}).
		Returns(0, "", BaseResponse{})
}

func EnrichExportConfigFileApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("导出配置文件").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Reads(apiconfig.ConfigFileExportRequest{}).
		ReturnsWithHeaders(0, "", nil, map[string]restful.Header{
			"Content-Type": {
				Items: &restful.Items{
					Type:    "string",
					Default: "application/zip",
				},
			},
			"Content-Disposition": {
				Items: &restful.Items{
					Type:    "string",
					Default: "attachment; filename=config.zip",
				},
			},
		})
}

func EnrichImportConfigFileApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("导入配置文件").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Param(restful.QueryParameter("namespace", "命名空间").DataType(typeNameString).Required(true)).
		Param(restful.QueryParameter("group", "配置文件分组").DataType(typeNameString).Required(false)).
		Param(restful.MultiPartFormParameter("conflict_handling",
			"配置文件冲突处理，跳过skip，覆盖overwrite").DataType(typeNameString).Required(true)).
		Param(restful.MultiPartFormParameter("config", "配置文件").DataType("file").Required(true)).
		Returns(0, "", config_manage.ConfigImportResponse{})
}

func EnrichPublishConfigFileApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("发布配置文件").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Reads(apiconfig.ConfigFileRelease{}).
		Returns(0, "", BaseResponse{})
}

func EnrichGetConfigFileReleaseApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("获取配置文件最后一次全量发布信息").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Param(restful.QueryParameter("namespace", "命名空间").DataType(typeNameString).Required(true)).
		Param(restful.QueryParameter("group", "配置文件分组").DataType(typeNameString).Required(true)).
		Param(restful.QueryParameter("name", "配置文件").DataType(typeNameString).Required(true)).
		Returns(0, "", struct {
			BaseResponse
			ConfigFileRelease config_manage.ConfigFileRelease `json:"configFileRelease,omitempty"`
		}{})
}

func EnrichGetConfigFileReleaseHistoryApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("获取配置文件发布历史记录").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Param(restful.QueryParameter("namespace", "命名空间").DataType(typeNameString).Required(true)).
		Param(restful.QueryParameter("group", "配置文件分组").DataType(typeNameString).Required(false)).
		Param(restful.QueryParameter("name", "配置文件").DataType(typeNameString).Required(false)).
		Param(restful.QueryParameter("offset", "翻页偏移量 默认为 0").DataType(typeNameInteger).
			Required(false).DefaultValue("0")).
		Param(restful.QueryParameter("limit", "一页大小，最大为 100").DataType(typeNameInteger).
			Required(true).DefaultValue("100")).
		Returns(0, "", struct {
			BatchQueryResponse
			ConfigFileReleaseHistories []config_manage.ConfigFileReleaseHistory `json:"configFileReleaseHistories,omitempty"`
		}{})
}

func EnrichGetAllConfigFileTemplatesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("获取配置模板").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Returns(0, "", struct {
			BatchQueryResponse
			ConfigFileTemplates []config_manage.ConfigFileTemplate `json:"configFileTemplates,omitempty"`
		}{})
}

func EnrichCreateConfigFileTemplateApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("创建配置模板").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Reads(config_manage.ConfigFileTemplate{}).
		Returns(0, "", BaseResponse{})
}

func EnrichConfigDiscoverApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("配置数据发现").
		Metadata(restfulspec.KeyOpenAPITags, configClientApiTags).
		Reads(config_manage.ConfigDiscoverResponse{}).
		Returns(0, "", apiconfig.ConfigDiscoverResponse{})
}

func EnrichGetConfigFileForClientApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("拉取配置").
		Metadata(restfulspec.KeyOpenAPITags, configClientApiTags).
		Param(restful.QueryParameter("namespace", "命名空间").DataType(typeNameString).Required(true)).
		Param(restful.QueryParameter("group", "配置文件分组").DataType(typeNameString).Required(true)).
		Param(restful.QueryParameter("fileName", "配置文件名").DataType(typeNameString).Required(true)).
		Param(restful.QueryParameter("version", "配置文件客户端版本号，刚启动时设置为 0").
			DataType(typeNameInteger).Required(true)).
		Returns(0, "", config_manage.ConfigClientResponse{})
}

func EnrichWatchConfigFileForClientApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("监听配置").
		Metadata(restfulspec.KeyOpenAPITags, configClientApiTags).
		Reads(apiconfig.ClientWatchConfigFileRequest{}, "通过 Http LongPolling 机制订阅配置变更。").
		Returns(0, "", config_manage.ConfigClientResponse{})
}

func EnrichGetConfigFileMetadataList(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("监听配置").
		Metadata(restfulspec.KeyOpenAPITags, configClientApiTags).
		Reads(apiconfig.ClientWatchConfigFileRequest{}, "通过 Http LongPolling 机制订阅配置变更。").
		Returns(0, "", config_manage.ConfigClientResponse{})
}

func EnrichGetAllConfigEncryptAlgorithms(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("返回当前配置加解密的算法").
		Metadata(restfulspec.KeyOpenAPITags, configConsoleApiTags).
		Returns(0, "", config_manage.ConfigEncryptAlgorithmResponse{})
}
