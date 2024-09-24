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

package config

import (
	"github.com/emicklei/go-restful/v3"

	"github.com/polarismesh/polaris/admin"
	"github.com/polarismesh/polaris/apiserver"
	"github.com/polarismesh/polaris/apiserver/httpserver/docs"
	"github.com/polarismesh/polaris/config"
	"github.com/polarismesh/polaris/namespace"
)

// HTTPServer
type HTTPServer struct {
	maintainServer  admin.AdminOperateServer
	namespaceServer namespace.NamespaceOperateServer
	configServer    config.ConfigCenterServer
}

// NewServer 创建配置中心的 HttpServer
func NewServer(
	maintainServer admin.AdminOperateServer,
	namespaceServer namespace.NamespaceOperateServer,
	configServer config.ConfigCenterServer) *HTTPServer {
	return &HTTPServer{
		maintainServer:  maintainServer,
		namespaceServer: namespaceServer,
		configServer:    configServer,
	}
}

const (
	defaultReadAccess   string = "default-read"
	defaultAccess       string = "default"
	configConsoleAccess string = "config"
)

// GetConfigAccessServer 获取配置中心接口
func (h *HTTPServer) GetConsoleAccessServer(include []string) (*restful.WebService, error) {
	consoleAccess := []string{defaultAccess}

	ws := new(restful.WebService)
	ws.Path("/config/v1").Consumes(restful.MIME_JSON, "multipart/form-data").Produces(restful.MIME_JSON, "application/zip")

	if len(include) == 0 {
		include = consoleAccess
	}

	for _, item := range include {
		switch item {
		case defaultReadAccess:
			h.addDefaultReadAccess(ws)
			// 仅为了兼容老的客户端发现路径
			h.addDiscover(ws)
		case configConsoleAccess, defaultAccess:
			// 仅为了兼容老的客户端发现路径
			h.addDiscover(ws)
			h.addDefaultAccess(ws)
		}
	}

	return ws, nil
}

func (h *HTTPServer) addDefaultReadAccess(ws *restful.WebService) {
	ws.Route(docs.EnrichQueryConfigFileGroupsApiDocs(ws.GET("/configfilegroups").To(h.QueryConfigFileGroups)))
	ws.Route(docs.EnrichGetConfigFileApiDocs(ws.GET("/configfiles").To(h.GetConfigFile)))
	ws.Route(docs.EnrichQueryConfigFilesByGroupApiDocs(ws.GET("/configfiles/by-group").To(h.SearchConfigFile)))
	ws.Route(docs.EnrichSearchConfigFileApiDocs(ws.GET("/configfiles/search").To(h.SearchConfigFile)))
	ws.Route(docs.EnrichGetAllConfigEncryptAlgorithms(ws.GET("/configfiles/encryptalgorithm").
		To(h.GetAllConfigEncryptAlgorithms)))
	ws.Route(docs.EnrichGetConfigFileReleaseApiDocs(ws.GET("/configfiles/release").To(h.GetConfigFileRelease)))
	ws.Route(docs.EnrichGetConfigFileReleaseHistoryApiDocs(ws.GET("/configfiles/releasehistory").
		To(h.GetConfigFileReleaseHistory)))
	ws.Route(docs.EnrichGetAllConfigFileTemplatesApiDocs(ws.GET("/configfiletemplates").To(h.GetAllConfigFileTemplates)))
}

func (h *HTTPServer) addDefaultAccess(ws *restful.WebService) {
	// 配置文件组
	ws.Route(docs.EnrichCreateConfigFileGroupApiDocs(ws.POST("/configfilegroups").To(h.CreateConfigFileGroup)))
	ws.Route(docs.EnrichUpdateConfigFileGroupApiDocs(ws.PUT("/configfilegroups").To(h.UpdateConfigFileGroup)))
	ws.Route(docs.EnrichDeleteConfigFileGroupApiDocs(ws.DELETE("/configfilegroups").To(h.DeleteConfigFileGroup)))
	ws.Route(docs.EnrichQueryConfigFileGroupsApiDocs(ws.GET("/configfilegroups").To(h.QueryConfigFileGroups)))

	// 配置文件
	ws.Route(docs.EnrichCreateConfigFileApiDocs(ws.POST("/configfiles").To(h.CreateConfigFile)))
	ws.Route(docs.EnrichGetConfigFileApiDocs(ws.GET("/configfiles").To(h.GetConfigFile)))
	ws.Route(docs.EnrichQueryConfigFilesByGroupApiDocs(ws.GET("/configfiles/by-group").To(h.SearchConfigFile)))
	ws.Route(docs.EnrichSearchConfigFileApiDocs(ws.GET("/configfiles/search").To(h.SearchConfigFile)))
	ws.Route(docs.EnrichUpdateConfigFileApiDocs(ws.PUT("/configfiles").To(h.UpdateConfigFile)))
	ws.Route(docs.EnrichDeleteConfigFileApiDocs(ws.DELETE("/configfiles").To(h.DeleteConfigFile)))
	ws.Route(docs.EnrichBatchDeleteConfigFileApiDocs(ws.POST("/configfiles/batchdelete").To(h.BatchDeleteConfigFile)))
	ws.Route(docs.EnrichExportConfigFileApiDocs(ws.POST("/configfiles/export").To(h.ExportConfigFile)))
	ws.Route(docs.EnrichImportConfigFileApiDocs(ws.POST("/configfiles/import").To(h.ImportConfigFile)))
	ws.Route(docs.EnrichGetAllConfigEncryptAlgorithms(ws.GET("/configfiles/encryptalgorithm").
		To(h.GetAllConfigEncryptAlgorithms)))

	// 配置文件发布
	ws.Route(docs.EnrichPublishConfigFileApiDocs(ws.POST("/configfiles/release").To(h.PublishConfigFile)))
	ws.Route(docs.EnrichGetConfigFileReleaseApiDocs(ws.PUT("/configfiles/releases/rollback").To(h.RollbackConfigFileReleases)))
	ws.Route(docs.EnrichGetConfigFileReleaseApiDocs(ws.GET("/configfiles/release").To(h.GetConfigFileRelease)))
	ws.Route(docs.EnrichGetConfigFileReleaseApiDocs(ws.GET("/configfiles/releases").To(h.GetConfigFileReleases)))
	ws.Route(docs.EnrichGetConfigFileReleaseApiDocs(ws.POST("/configfiles/releases/delete").To(h.DeleteConfigFileReleases)))
	ws.Route(docs.EnrichGetConfigFileReleaseApiDocs(ws.GET("/configfiles/release/versions").To(h.GetConfigFileReleaseVersions)))
	ws.Route(docs.EnrichUpsertAndReleaseConfigFileApiDocs(ws.POST("/configfiles/createandpub").To(h.UpsertAndReleaseConfigFile)))
	ws.Route(docs.EnrichStopBetaReleaseConfigFileApiDocs(ws.POST("/configfiles/releases/stopbeta").To(h.StopGrayConfigFileReleases)))

	// 配置文件发布历史
	ws.Route(docs.EnrichGetConfigFileReleaseHistoryApiDocs(ws.GET("/configfiles/releasehistory").
		To(h.GetConfigFileReleaseHistory)))

	// config file template
	ws.Route(docs.EnrichGetAllConfigFileTemplatesApiDocs(ws.GET("/configfiletemplates").To(h.GetAllConfigFileTemplates)))
	ws.Route(docs.EnrichCreateConfigFileTemplateApiDocs(ws.POST("/configfiletemplates").To(h.CreateConfigFileTemplate)))
}

// GetClientAccessServer 获取配置中心接口
func (h *HTTPServer) GetClientAccessServer(ws *restful.WebService, include []string) error {
	clientAccess := []string{apiserver.DiscoverAccess, apiserver.CreateFileAccess}

	if len(include) == 0 {
		include = clientAccess
	}

	for _, item := range include {
		switch item {
		case apiserver.CreateFileAccess:
			h.addCreateFile(ws)
		case apiserver.DiscoverAccess:
			h.addDiscover(ws)
		}
	}

	return nil
}

func (h *HTTPServer) addDiscover(ws *restful.WebService) {
	ws.Route(docs.EnrichConfigDiscoverApiDocs(ws.POST("/ConfigDiscover").To(h.Discover)))
	ws.Route(docs.EnrichGetConfigFileForClientApiDocs(ws.GET("/GetConfigFile").To(h.ClientGetConfigFile)))
	ws.Route(docs.EnrichWatchConfigFileForClientApiDocs(ws.POST("/WatchConfigFile").To(h.ClientWatchConfigFile)))
	ws.Route(docs.EnrichGetConfigFileMetadataList(ws.POST("/GetConfigFileMetadataList").To(h.GetConfigFileMetadataList)))
}

func (h *HTTPServer) addCreateFile(ws *restful.WebService) {
}
