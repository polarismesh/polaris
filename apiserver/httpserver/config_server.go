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
	// 配置文件组
	ws.Route(enrichCreateConfigFileGroupApiDocs(ws.POST("/configfilegroups").To(h.CreateConfigFileGroup)))
	ws.Route(enrichQueryConfigFileGroupsApiDocs(ws.GET("/configfilegroups").To(h.QueryConfigFileGroups)))
	ws.Route(enrichDeleteConfigFileGroupApiDocs(ws.DELETE("/configfilegroups").To(h.DeleteConfigFileGroup)))
	ws.Route(enrichUpdateConfigFileGroupApiDocs(ws.PUT("/configfilegroups").To(h.UpdateConfigFileGroup)))

	// 配置文件
	ws.Route(enrichCreateConfigFileApiDocs(ws.POST("/configfiles").To(h.CreateConfigFile)))
	ws.Route(enrichGetConfigFileApiDocs(ws.GET("/configfiles").To(h.GetConfigFile)))
	ws.Route(enrichQueryConfigFilesByGroupApiDocs(ws.GET("/configfiles/by-group").To(h.QueryConfigFilesByGroup)))
	ws.Route(enrichSearchConfigFileApiDocs(ws.GET("/configfiles/search").To(h.SearchConfigFile)))
	ws.Route(enrichUpdateConfigFileApiDocs(ws.PUT("/configfiles").To(h.UpdateConfigFile)))
	ws.Route(enrichDeleteConfigFileApiDocs(ws.DELETE("/configfiles").To(h.DeleteConfigFile)))
	ws.Route(enrichBatchDeleteConfigFileApiDocs(ws.POST("/configfiles/batchdelete").To(h.BatchDeleteConfigFile)))

	// 配置文件发布
	ws.Route(enrichPublishConfigFileApiDocs(ws.POST("/configfiles/release").To(h.PublishConfigFile)))
	ws.Route(enrichGetConfigFileReleaseApiDocs(ws.GET("/configfiles/release").To(h.GetConfigFileRelease)))

	// 配置文件发布历史
	ws.Route(enrichGetConfigFileReleaseHistoryApiDocs(ws.GET("/configfiles/releasehistory").To(h.GetConfigFileReleaseHistory)))

	// config file template
	ws.Route(enrichGetAllConfigFileTemplatesApiDocs(ws.GET("/configfiletemplates").To(h.GetAllConfigFileTemplates)))
	ws.Route(enrichCreateConfigFileTemplateApiDocs(ws.POST("/configfiletemplates").To(h.CreateConfigFileTemplate)))
}

func (h *HTTPServer) bindConfigClientEndpoint(ws *restful.WebService) {
	ws.Route(enrichGetConfigFileForClientApiDocs(ws.GET("/GetConfigFile").To(h.getConfigFile)))
	ws.Route(enrichWatchConfigFileForClientApiDocs(ws.POST("/WatchConfigFile").To(h.watchConfigFile)))
}

// StopConfigServer 停止配置中心模块
func (h *HTTPServer) StopConfigServer() {
}
