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

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"

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
	ws.Route(ws.GET("/configfilegroups").To(h.QueryConfigFileGroups))
	ws.Route(ws.DELETE("/configfilegroups").To(h.DeleteConfigFileGroup))
	ws.Route(ws.PUT("/configfilegroups").To(h.UpdateConfigFileGroup))

	// 配置文件
	ws.Route(ws.POST("/configfiles").To(h.CreateConfigFile))
	ws.Route(ws.GET("/configfiles").To(h.GetConfigFile).
		Doc("拉取配置").
		Metadata(restfulspec.KeyOpenAPITags, tags))
	ws.Route(ws.GET("/configfiles/by-group").To(h.QueryConfigFilesByGroup))
	ws.Route(ws.GET("/configfiles/search").To(h.SearchConfigFile))
	ws.Route(ws.PUT("/configfiles").To(h.UpdateConfigFile))
	ws.Route(ws.DELETE("/configfiles").To(h.DeleteConfigFile))
	ws.Route(ws.POST("/configfiles/batchdelete").To(h.BatchDeleteConfigFile))

	// 配置文件发布
	ws.Route(ws.POST("/configfiles/release").To(h.PublishConfigFile))
	ws.Route(ws.GET("/configfiles/release").To(h.GetConfigFileRelease))

	// 配置文件发布历史
	ws.Route(ws.GET("/configfiles/releasehistory").To(h.GetConfigFileReleaseHistory))

	// config file template
	ws.Route(ws.GET("/configfiletemplates").To(h.GetAllConfigFileTemplates))
	ws.Route(ws.POST("/configfiletemplates").To(h.CreateConfigFileTemplate))
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
