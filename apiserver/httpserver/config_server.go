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
	"fmt"

	"github.com/emicklei/go-restful/v3"

	restfulspec "github.com/polarismesh/go-restful-openapi/v2"

	api "github.com/polarismesh/polaris-server/common/api/v1"
)

const (
	configDefaultAccess string = "default"
	configConsoleAccess string = "console"
	configClientAccess  string = "client"
)

// GetConfigAccessServer 获取配置中心接口
func (h *HTTPServer) GetConfigAccessServer(include []string) (*restful.WebService, error) {
	consoleAccess := []string{configDefaultAccess}

	ws := new(restful.WebService)
	ws.Path("/config/v1").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

	if len(include) == 0 {
		include = consoleAccess
	}

	for _, item := range include {
		switch item {
		case configDefaultAccess:
			h.bindConfigConsoleEndpoint(ws)
			h.bindConfigClientEndpoint(ws)
		case configConsoleAccess:
			h.bindConfigConsoleEndpoint(ws)
		case configClientAccess:
			h.bindConfigClientEndpoint(ws)
		default:
			log.Errorf("[Config][HttpServer] the patch of config endpoint [%s] does not exist", item)
			return nil, fmt.Errorf("[Config][HttpServer] the patch of config endpoint [%s] does not exist", item)
		}
	}

	return ws, nil
}

func (h *HTTPServer) bindConfigConsoleEndpoint(ws *restful.WebService) {
	tags := []string{"ConfigConsole"}
	// 配置文件组
	ws.Route(ws.POST("/configfilegroups").To(h.CreateConfigFileGroup).
		Doc("创建配置文件组").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(api.ConfigFileGroup{}, "开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header\nHeader X-Polaris-Token: {访问凭据}\n ```\n{\n    \"name\":\"someGroup\",\n    \"namespace\":\"someNamespace\",\n    \"comment\":\"some comment\",\n    \"createBy\":\"ledou\"\n}\n```"))
	ws.Route(ws.GET("/configfilegroups").To(h.QueryConfigFileGroups).
		Doc("搜索配置文件组").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.QueryParameter("namespace", "命名空间，不填表示全部命名空间").DataType("string").Required(false)).
		Param(ws.QueryParameter("group", "配置文件分组名，模糊搜索").DataType("string").Required(false)).
		Param(ws.QueryParameter("fileName", "配置文件名称，模糊搜索").DataType("string").Required(false)).
		Param(ws.QueryParameter("offset", "翻页偏移量 默认为 0").DataType("int").Required(false).DefaultValue("0")).
		Param(ws.QueryParameter("limit", "一页大小，最大为 100").DataType("int").Required(true).DefaultValue("100")))
	ws.Route(ws.DELETE("/configfilegroups").To(h.DeleteConfigFileGroup).
		Doc("删除配置文件").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.QueryParameter("namespace", "命名空间").DataType("string").Required(true)).
		Param(ws.QueryParameter("group", "配置文件分组").DataType("string").Required(true)))
	ws.Route(ws.PUT("/configfilegroups").To(h.UpdateConfigFileGroup).
		Doc("更新配置文件组").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(api.ConfigFileGroup{}, "开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header\nHeader X-Polaris-Token: {访问凭据}\n ```\n{\n    \"name\":\"someGroup\",\n    \"namespace\":\"someNamespace\",\n    \"comment\":\"some comment\",\n    \"createBy\":\"ledou\"\n}\n```"))

	// 配置文件
	ws.Route(ws.POST("/configfiles").To(h.CreateConfigFile).
		Doc("创建配置文件").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(api.ConfigFile{}, "开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header\nHeader X-Polaris-Token: {访问凭据}\n ```{\n    \"name\":\"application.properties\",\n    \"namespace\":\"someNamespace\",\n    \"group\":\"someGroup\",\n    \"content\":\"redis.cache.age=10\",\n    \"comment\":\"第一个配置文件\",\n    \"tags\":[{\"key\":\"service\", \"value\":\"helloService\"}],\n    \"createBy\":\"ledou\",\n    \"format\":\"properties\"\n}\n```\n"))
	ws.Route(ws.GET("/configfiles").To(h.GetConfigFile).
		Doc("拉取配置").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.QueryParameter("namespace", "命名空间").DataType("string").Required(true)).
		Param(ws.QueryParameter("group", "配置文件分组").DataType("string").Required(true)).
		Param(ws.QueryParameter("name", "配置文件名").DataType("string").Required(true)))
	ws.Route(ws.GET("/configfiles/by-group").To(h.QueryConfigFilesByGroup).
		Doc("搜索配置文件").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.QueryParameter("namespace", "命名空间").DataType("string").Required(false)).
		Param(ws.QueryParameter("group", "配置文件分组").DataType("string").Required(false)).
		Param(ws.QueryParameter("offset", "翻页偏移量 默认为 0").DataType("int").Required(false).DefaultValue("0")).
		Param(ws.QueryParameter("limit", "一页大小，最大为 100").DataType("int").Required(true).DefaultValue("100")))
	ws.Route(ws.GET("/configfiles/search").To(h.SearchConfigFile).
		Doc("搜索配置文件").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.QueryParameter("namespace", "命名空间").DataType("string").Required(false)).
		Param(ws.QueryParameter("group", "配置文件分组").DataType("string").Required(false)).
		Param(ws.QueryParameter("name", "配置文件").DataType("string").Required(false)).
		Param(ws.QueryParameter("tags", "格式：key1,value1,key2,value2").DataType("string").Required(false)).
		Param(ws.QueryParameter("offset", "翻页偏移量 默认为 0").DataType("int").Required(false).DefaultValue("0")).
		Param(ws.QueryParameter("limit", "一页大小，最大为 100").DataType("int").Required(true).DefaultValue("100")))
	ws.Route(ws.PUT("/configfiles").To(h.UpdateConfigFile).
		Doc("创建配置文件").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(api.ConfigFile{}, "开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header\nHeader X-Polaris-Token: {访问凭据}\n ```{\n    \"name\":\"application.properties\",\n    \"namespace\":\"someNamespace\",\n    \"group\":\"someGroup\",\n    \"content\":\"redis.cache.age=10\",\n    \"comment\":\"第一个配置文件\",\n    \"tags\":[{\"key\":\"service\", \"value\":\"helloService\"}],\n    \"createBy\":\"ledou\",\n    \"format\":\"properties\"\n}\n```\n"))
	ws.Route(ws.DELETE("/configfiles").To(h.DeleteConfigFile).
		Doc("创建配置文件").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.QueryParameter("namespace", "命名空间").DataType("string").Required(true)).
		Param(ws.QueryParameter("group", "配置文件分组").DataType("string").Required(true)).
		Param(ws.QueryParameter("name", "配置文件").DataType("string").Required(true)).
		Param(ws.QueryParameter("deleteBy", "操作人").DataType("string").Required(false)))
	ws.Route(ws.POST("/configfiles/batchdelete").To(h.BatchDeleteConfigFile).
		Doc("批量删除配置文件").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.QueryParameter("deleteBy", "操作人").DataType("string").Required(false)).
		Reads(api.ConfigFile{}, "开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header\nHeader X-Polaris-Token: {访问凭据}\n```[\n     {\n         \"name\":\"application.properties\",\n         \"namespace\":\"someNamespace\",\n         \"group\":\"someGroup\"\n     }\n]\n```"))

	// 配置文件发布
	ws.Route(ws.POST("/configfiles/release").To(h.PublishConfigFile).
		Doc("发布配置文件").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(api.ConfigFileRelease{}, "开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header\nHeader X-Polaris-Token: {访问凭据}\n```{\n    \"name\":\"release-002\",\n    \"fileName\":\"application.properties\",\n    \"namespace\":\"someNamespace\",\n    \"group\":\"someGroup\",\n    \"comment\":\"发布第一个配置文件\",\n    \"createBy\":\"ledou\"\n}\n```"))
	ws.Route(ws.GET("/configfiles/release").To(h.GetConfigFileRelease).
		Doc("获取配置文件最后一次全量发布信息").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.QueryParameter("namespace", "命名空间").DataType("string").Required(true)).
		Param(ws.QueryParameter("group", "配置文件分组").DataType("string").Required(true)).
		Param(ws.QueryParameter("name", "配置文件").DataType("string").Required(true)))

	// 配置文件发布历史
	ws.Route(ws.GET("/configfiles/releasehistory").To(h.GetConfigFileReleaseHistory).
		Doc("获取配置文件发布历史记录").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.QueryParameter("namespace", "命名空间").DataType("string").Required(true)).
		Param(ws.QueryParameter("group", "配置文件分组").DataType("string").Required(false)).
		Param(ws.QueryParameter("name", "配置文件").DataType("string").Required(false)).
		Param(ws.QueryParameter("offset", "翻页偏移量 默认为 0").DataType("int").Required(false).DefaultValue("0")).
		Param(ws.QueryParameter("limit", "一页大小，最大为 100").DataType("int").Required(true).DefaultValue("100")))

	// config file template
	ws.Route(ws.GET("/configfiletemplates").To(h.GetAllConfigFileTemplates).
		Doc("获取配置模板").
		Metadata(restfulspec.KeyOpenAPITags, tags))
	ws.Route(ws.POST("/configfiletemplates").To(h.CreateConfigFileTemplate).
		Doc("创建配置模板").
		Metadata(restfulspec.KeyOpenAPITags, tags))
}

func (h *HTTPServer) bindConfigClientEndpoint(ws *restful.WebService) {
	tags := []string{"ConfigClient"}
	ws.Route(ws.GET("/GetConfigFile").To(h.getConfigFile).
		Doc("拉取配置").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.QueryParameter("namespace", "命名空间").DataType("string").Required(true)).
		Param(ws.QueryParameter("group", "配置文件分组").DataType("string").Required(true)).
		Param(ws.QueryParameter("fileName", "配置文件名").DataType("string").Required(true)).
		Param(ws.QueryParameter("version", "配置文件客户端版本号，刚启动时设置为 0").DataType("integer").Required(true)))

	ws.Route(ws.POST("/WatchConfigFile").To(h.watchConfigFile).
		Doc("监听配置").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(api.ClientWatchConfigFileRequest{}, "通过 Http LongPolling 机制订阅配置变更。"))

}

// StopConfigServer 停止配置中心模块
func (h *HTTPServer) StopConfigServer() {
}
