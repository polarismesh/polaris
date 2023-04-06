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

	"github.com/polarismesh/polaris/apiserver/httpserver/docs"
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
	ws.Path("/config/v1").Consumes(restful.MIME_JSON, "multipart/form-data").Produces(restful.MIME_JSON, "application/zip")

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
	ws.Route(docs.EnrichCreateConfigFileGroupApiDocs(ws.POST("/configfilegroups").To(h.CreateConfigFileGroup)))
	ws.Route(docs.EnrichQueryConfigFileGroupsApiDocs(ws.GET("/configfilegroups").To(h.QueryConfigFileGroups)))
	ws.Route(docs.EnrichDeleteConfigFileGroupApiDocs(ws.DELETE("/configfilegroups").To(h.DeleteConfigFileGroup)))
	ws.Route(docs.EnrichUpdateConfigFileGroupApiDocs(ws.PUT("/configfilegroups").To(h.UpdateConfigFileGroup)))

	// 配置文件
	ws.Route(docs.EnrichCreateConfigFileApiDocs(ws.POST("/configfiles").To(h.CreateConfigFile)))
	ws.Route(docs.EnrichGetConfigFileApiDocs(ws.GET("/configfiles").To(h.GetConfigFile)))
	ws.Route(docs.EnrichQueryConfigFilesByGroupApiDocs(ws.GET("/configfiles/by-group").To(h.QueryConfigFilesByGroup)))
	ws.Route(docs.EnrichSearchConfigFileApiDocs(ws.GET("/configfiles/search").To(h.SearchConfigFile)))
	ws.Route(docs.EnrichUpdateConfigFileApiDocs(ws.PUT("/configfiles").To(h.UpdateConfigFile)))
	ws.Route(docs.EnrichDeleteConfigFileApiDocs(ws.DELETE("/configfiles").To(h.DeleteConfigFile)))
	ws.Route(docs.EnrichBatchDeleteConfigFileApiDocs(ws.POST("/configfiles/batchdelete").To(h.BatchDeleteConfigFile)))
	ws.Route(docs.EnrichExportConfigFileApiDocs(ws.POST("/configfiles/export").To(h.ExportConfigFile)))
	ws.Route(docs.EnrichImportConfigFileApiDocs(ws.POST("/configfiles/import").To(h.ImportConfigFile)))

	// 配置文件发布
	ws.Route(docs.EnrichPublishConfigFileApiDocs(ws.POST("/configfiles/release").To(h.PublishConfigFile)))
	ws.Route(docs.EnrichGetConfigFileReleaseApiDocs(ws.GET("/configfiles/release").To(h.GetConfigFileRelease)))

	// 配置文件发布历史
	ws.Route(docs.EnrichGetConfigFileReleaseHistoryApiDocs(ws.GET("/configfiles/releasehistory").
		To(h.GetConfigFileReleaseHistory)))

	// config file template
	ws.Route(docs.EnrichGetAllConfigFileTemplatesApiDocs(ws.GET("/configfiletemplates").To(h.GetAllConfigFileTemplates)))
	ws.Route(docs.EnrichCreateConfigFileTemplateApiDocs(ws.POST("/configfiletemplates").To(h.CreateConfigFileTemplate)))
}

func (h *HTTPServer) bindConfigClientEndpoint(ws *restful.WebService) {
	ws.Route(docs.EnrichGetConfigFileForClientApiDocs(ws.GET("/GetConfigFile").To(h.getConfigFile)))
	ws.Route(docs.EnrichWatchConfigFileForClientApiDocs(ws.POST("/WatchConfigFile").To(h.watchConfigFile)))
}

// StopConfigServer 停止配置中心模块
func (h *HTTPServer) StopConfigServer() {
}
