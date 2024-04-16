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
	"archive/zip"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/apiserver/nacosserver/model"
	nacoshttp "github.com/polarismesh/polaris/apiserver/nacosserver/v1/http"
)

func (n *ConfigServer) GetClientServer() (*restful.WebService, error) {
	ws := new(restful.WebService)
	ws.Path("/nacos/v1/cs/configs").Consumes(restful.MIME_JSON, model.MIME).Produces(restful.MIME_JSON)
	n.addConfigFileAccess(ws)
	return ws, nil
}

func (n *ConfigServer) addConfigFileAccess(ws *restful.WebService) {
	ws.Route(ws.POST("/").To(n.Dispatch))
	ws.Route(ws.GET("/").To(n.GetConfig))
	ws.Route(ws.DELETE("/").To(n.DeleteConfig))
	ws.Route(ws.POST("/listener").To(n.WatchConfigs))
}

func (n *ConfigServer) Dispatch(req *restful.Request, rsp *restful.Response) {
	switch req.Request.URL.RawQuery {
	case "":
		n.PublishConfig(req, rsp)
	case "import=true":
		n.ConfigImport(req, rsp)
	}
}

func (n *ConfigServer) PublishConfig(req *restful.Request, rsp *restful.Response) {
	handler := nacoshttp.Handler{
		Request:  req,
		Response: rsp,
	}

	file, err := BuildConfigFile(req)
	if err != nil {
		nacoshttp.WrirteNacosErrorResponse(err, rsp)
		return
	}

	ret, err := n.handlePublishConfig(handler.ParseHeaderContext(), file)
	if err != nil {
		nacoshttp.WrirteNacosErrorResponse(err, rsp)
		return
	}
	nacoshttp.WrirteNacosResponse(ret, rsp)
}

func (n *ConfigServer) GetConfig(req *restful.Request, rsp *restful.Response) {
	handler := nacoshttp.Handler{
		Request:  req,
		Response: rsp,
	}

	baseInfo, err := parseConfigFileBase(req)
	if err != nil {
		nacoshttp.WrirteNacosErrorResponse(err, rsp)
		return
	}
	ret, err := n.handleGetConfig(handler.ParseHeaderContext(), &model.ConfigFile{
		ConfigFileBase: *baseInfo,
	}, rsp)
	if err != nil {
		nacoshttp.WrirteNacosErrorResponse(err, rsp)
		return
	}
	nacoshttp.WrirteSimpleResponse(ret, http.StatusOK, rsp)
}

func (n *ConfigServer) DeleteConfig(req *restful.Request, rsp *restful.Response) {
	handler := nacoshttp.Handler{
		Request:  req,
		Response: rsp,
	}
	baseInfo, err := parseConfigFileBase(req)
	if err != nil {
		nacoshttp.WrirteNacosErrorResponse(err, rsp)
		return
	}
	ret, err := n.handleDeleteConfig(handler.ParseHeaderContext(), &model.ConfigFile{
		ConfigFileBase: *baseInfo,
	})
	if err != nil {
		nacoshttp.WrirteNacosErrorResponse(err, rsp)
		return
	}
	nacoshttp.WrirteNacosResponse(ret, rsp)
}

func (n *ConfigServer) WatchConfigs(req *restful.Request, rsp *restful.Response) {
	handler := nacoshttp.Handler{
		Request:  req,
		Response: rsp,
	}

	probeModify := req.QueryParameter("Listening-Configs")
	if probeModify == "" {
		nacoshttp.WrirteNacosResponseWithCode(http.StatusBadRequest, "invalid probeModify", rsp)
		return
	}
	nacoslog.Info("[NACOS-V1][Config] receive client watch request.", zap.Any("listenCtx", probeModify))
	listenCtx, err := model.ParseConfigListenContext(req, probeModify)
	if err != nil {
		nacoshttp.WrirteNacosErrorResponse(err, rsp)
		return
	}

	n.handleWatch(handler.ParseHeaderContext(), listenCtx, rsp)
}

func (n *ConfigServer) ConfigImport(req *restful.Request, rsp *restful.Response) {
	handler := nacoshttp.Handler{
		Request:  req,
		Response: rsp,
	}

	var metaDataItem *ZipItem
	var items = make([]*ZipItem, 0, 32)

	handler.ProcessZip(func(f *zip.File, data []byte) {
		if (f.Name == ConfigExportMetadata || f.Name == ConfigExpotrMetadataV2) && metaDataItem == nil {
			metaDataItem = &ZipItem{
				Name: f.Name,
				Data: data,
			}
			return
		}
		items = append(items, &ZipItem{
			Name: f.Name,
			Data: data,
		})
	})

	policy := req.QueryParameter("policy")

	n.handleConfigImport(handler.ParseHeaderContext(), policy, &UnZipResult{
		Meta:  metaDataItem,
		Items: items,
	}, rsp)
}

func parseConfigFileBase(req *restful.Request) (*model.ConfigFileBase, error) {
	namespace := nacoshttp.Optional(req, model.ParamTenant, model.DefaultNacosConfigNamespace)
	dataId, err := nacoshttp.Required(req, "dataId")
	if err != nil {
		return nil, err
	}
	group, err := nacoshttp.Required(req, "group")
	if err != nil {
		return nil, err
	}

	return &model.ConfigFileBase{
		Namespace: namespace,
		Group:     group,
		DataId:    dataId,
	}, nil
}
