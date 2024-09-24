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

package discover

import (
	"net/http"

	"github.com/emicklei/go-restful/v3"

	"github.com/polarismesh/polaris/apiserver/nacosserver/model"
	nacoshttp "github.com/polarismesh/polaris/apiserver/nacosserver/v1/http"
)

func (n *DiscoverServer) GetClientServer() (*restful.WebService, error) {
	ws := new(restful.WebService)
	ws.Path("/nacos/v1/ns").Consumes(restful.MIME_JSON, model.MIME).Produces(restful.MIME_JSON)
	n.addInstanceAccess(ws)
	n.addSystemAccess(ws)
	n.AddServiceAccess(ws)
	return ws, nil
}

func (n *DiscoverServer) AddServiceAccess(ws *restful.WebService) {
	ws.Route(ws.GET("/service/list").To(n.ListServices))
}

func (n *DiscoverServer) addInstanceAccess(ws *restful.WebService) {
	ws.Route(ws.POST("/instance").To(n.RegisterInstance))
	ws.Route(ws.PUT("/instance").To(n.UpdateInstance))
	ws.Route(ws.DELETE("/instance").To(n.DeRegisterInstance))
	ws.Route(ws.PUT("/instance/beat").To(n.Heartbeat))
	ws.Route(ws.GET("/instance/list").To(n.ListInstances))
}

func (n *DiscoverServer) addSystemAccess(ws *restful.WebService) {
	ws.Route(ws.GET("/operator/metrics").To(n.ServerHealthStatus))
}

func (n *DiscoverServer) ListServices(req *restful.Request, rsp *restful.Response) {
	pageNo, err := nacoshttp.RequiredInt(req, model.ParamPageNo)
	if err != nil {
		nacoshttp.WrirteNacosErrorResponse(err, rsp)
		return
	}
	pageSize, err := nacoshttp.RequiredInt(req, model.ParamPageSize)
	if err != nil {
		nacoshttp.WrirteNacosErrorResponse(err, rsp)
		return
	}
	namespace := nacoshttp.Optional(req, model.ParamNamespaceID, model.DefaultNacosNamespace)
	namespace = model.ToPolarisNamespace(namespace)
	groupName := nacoshttp.Optional(req, model.ParamGroupName, model.DefaultServiceGroup)
	// selector := nacoshttp.Optional(req, model.ParamSelector, "")
	serviceList, count := model.HandleServiceListRequest(n.discoverSvr, namespace, groupName, pageNo, pageSize)
	resp := map[string]interface{}{
		"count": count,
		"doms":  serviceList,
	}
	nacoshttp.WrirteNacosResponse(resp, rsp)
}

func (n *DiscoverServer) RegisterInstance(req *restful.Request, rsp *restful.Response) {
	handler := nacoshttp.Handler{
		Request:  req,
		Response: rsp,
	}

	namespace := nacoshttp.Optional(req, model.ParamNamespaceID, model.DefaultNacosNamespace)
	namespace = model.ToPolarisNamespace(namespace)
	ins, err := BuildInstance(namespace, req, false)
	if err != nil {
		nacoshttp.WrirteNacosErrorResponse(err, rsp)
		return
	}

	ctx := handler.ParseHeaderContext()
	if err := n.handleRegister(ctx, namespace, ins.ServiceName, ins); err != nil {
		nacoshttp.WrirteNacosErrorResponse(err, rsp)
		return
	}
	nacoshttp.WrirteSimpleResponse("ok", http.StatusOK, rsp)
}

func (n *DiscoverServer) UpdateInstance(req *restful.Request, rsp *restful.Response) {
	handler := nacoshttp.Handler{
		Request:  req,
		Response: rsp,
	}

	namespace := nacoshttp.Optional(req, model.ParamNamespaceID, model.DefaultNacosNamespace)
	namespace = model.ToPolarisNamespace(namespace)
	ins, err := BuildInstance(namespace, req, false)
	if err != nil {
		nacoshttp.WrirteNacosErrorResponse(err, rsp)
		return
	}

	ctx := handler.ParseHeaderContext()
	if err := n.handleUpdate(ctx, namespace, ins.ServiceName, ins); err != nil {
		nacoshttp.WrirteNacosErrorResponse(err, rsp)
		return
	}
	nacoshttp.WrirteSimpleResponse("ok", http.StatusOK, rsp)
}

func (n *DiscoverServer) DeRegisterInstance(req *restful.Request, rsp *restful.Response) {
	handler := nacoshttp.Handler{
		Request:  req,
		Response: rsp,
	}

	namespace := nacoshttp.Optional(req, model.ParamNamespaceID, model.DefaultNacosNamespace)
	namespace = model.ToPolarisNamespace(namespace)
	ins, err := BuildInstance(namespace, req, true)
	if err != nil {
		nacoshttp.WrirteNacosErrorResponse(err, rsp)
		return
	}

	ctx := handler.ParseHeaderContext()
	if err := n.handleDeregister(ctx, namespace, ins.ServiceName, ins); err != nil {
		nacoshttp.WrirteNacosErrorResponse(err, rsp)
		return
	}
	nacoshttp.WrirteSimpleResponse("ok", http.StatusOK, rsp)
}

func (n *DiscoverServer) Heartbeat(req *restful.Request, rsp *restful.Response) {
	handler := nacoshttp.Handler{
		Request:  req,
		Response: rsp,
	}

	beat, err := BuildClientBeat(req)
	if err != nil {
		nacoshttp.WrirteNacosErrorResponse(err, rsp)
		return
	}

	ctx := handler.ParseHeaderContext()
	data, err := n.handleBeat(ctx, beat.Namespace, beat.ServiceName, beat)
	if err != nil {
		nacoshttp.WrirteNacosErrorResponse(err, rsp)
		return
	}
	nacoshttp.WrirteNacosResponse(data, rsp)
}

func (n *DiscoverServer) ListInstances(req *restful.Request, rsp *restful.Response) {
	handler := nacoshttp.Handler{
		Request:  req,
		Response: rsp,
	}

	ctx := handler.ParseHeaderContext()
	params := nacoshttp.ParseQueryParams(req)

	params[model.ParamNamespaceID] = model.ToPolarisNamespace(params[model.ParamNamespaceID])
	data, err := n.handleQueryInstances(ctx, params)
	if err != nil {
		nacoshttp.WrirteNacosErrorResponse(err, rsp)
		return
	}
	nacoshttp.WrirteNacosResponse(data, rsp)
}

func (n *DiscoverServer) ServerHealthStatus(req *restful.Request, rsp *restful.Response) {
	nacoshttp.WrirteNacosResponse(map[string]interface{}{
		"status": "UP",
	}, rsp)
}
