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

package eurekaserver

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"

	"github.com/emicklei/go-restful/v3"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/utils"
)

const (
	ParamAppId      string = "appId"
	ParamInstId     string = "instId"
	ParamValue      string = "value"
	ParamVip        string = "vipAddress"
	ParamSVip       string = "svipAddress"
	HeaderNamespace string = "x-namespace"
)

// GetEurekaServer eureka web server
func (h *EurekaServer) GetEurekaServer() *restful.WebService {
	ws := new(restful.WebService)

	ws.Path("/eureka").Consumes(restful.MIME_JSON, restful.MIME_OCTET, restful.MIME_XML).Produces(restful.MIME_JSON,
		restful.MIME_XML)
	h.addDiscoverAccess(ws)
	return ws
}

// GetEurekaV1Server eureka v1 web server
func (h *EurekaServer) GetEurekaV1Server() *restful.WebService {
	ws := new(restful.WebService)

	ws.Path("/eureka/v1").Consumes(restful.MIME_JSON, restful.MIME_OCTET, restful.MIME_XML).Produces(restful.MIME_JSON,
		restful.MIME_XML)
	h.addDiscoverAccess(ws)
	return ws
}

// GetEurekaV2Server eureka v2 web server
func (h *EurekaServer) GetEurekaV2Server() *restful.WebService {
	ws := new(restful.WebService)

	ws.Path("/eureka/v2").Consumes(restful.MIME_JSON, restful.MIME_OCTET, restful.MIME_XML).Produces(restful.MIME_JSON,
		restful.MIME_XML)
	h.addDiscoverAccess(ws)
	return ws
}

// addDiscoverAccess 增加服务发现接口
func (h *EurekaServer) addDiscoverAccess(ws *restful.WebService) {
	// Register new application instance
	ws.Route(ws.POST(fmt.Sprintf("/apps/{%s}", ParamAppId)).To(h.RegisterApplication)).
		Param(ws.PathParameter(ParamAppId, "applicationId").DataType("string"))
	// De-register application instance
	ws.Route(ws.DELETE(fmt.Sprintf("/apps/{%s}/{%s}", ParamAppId, ParamInstId)).To(h.CancelInstance)).
		Param(ws.PathParameter(ParamAppId, "applicationId").DataType("string")).
		Param(ws.PathParameter(ParamInstId, "instanceId").DataType("string"))
	// Send application instance heartbeat
	ws.Route(ws.PUT(fmt.Sprintf("/apps/{%s}/{%s}", ParamAppId, ParamInstId)).To(h.RenewInstance)).
		Param(ws.PathParameter(ParamAppId, "applicationId").DataType("string")).
		Param(ws.PathParameter(ParamInstId, "instanceId").DataType("string"))
	// Query for all instances
	ws.Route(ws.GET("/apps").To(h.GetAllApplications))
	// Query for all instances(delta)
	ws.Route(ws.GET("/apps/delta").To(h.GetDeltaApplications))
	// Query for all appID instances
	ws.Route(ws.GET(fmt.Sprintf("/apps/{%s}", ParamAppId)).To(h.GetApplication)).
		Param(ws.PathParameter(ParamAppId, "applicationId").DataType("string"))
	// Query for a specific appID/instanceID
	ws.Route(ws.GET(fmt.Sprintf("/apps/{%s}/{%s}", ParamAppId, ParamInstId)).To(h.GetAppInstance)).
		Param(ws.PathParameter(ParamAppId, "applicationId").DataType("string")).
		Param(ws.PathParameter(ParamInstId, "instanceId").DataType("string"))
	// Take instance out of service
	ws.Route(ws.PUT(fmt.Sprintf("/apps/{%s}/{%s}/status", ParamAppId, ParamInstId)).To(h.UpdateStatus)).
		Param(ws.PathParameter(ParamAppId, "applicationId").DataType("string")).
		Param(ws.PathParameter(ParamInstId, "instanceId").DataType("string"))
	// Move instance back into service (remove override)
	ws.Route(ws.DELETE(fmt.Sprintf("/apps/{%s}/{%s}/status", ParamAppId, ParamInstId)).To(h.DeleteStatus)).
		Param(ws.PathParameter(ParamAppId, "applicationId").DataType("string")).
		Param(ws.PathParameter(ParamInstId, "instanceId").DataType("string"))
	// Query for a specific instanceID
	ws.Route(ws.GET(fmt.Sprintf("/instances/{%s}", ParamInstId)).To(h.GetInstance)).
		Param(ws.PathParameter(ParamInstId, "instanceId").DataType("string"))
	// Update metadata
	ws.Route(ws.PUT(fmt.Sprintf("/apps/{%s}/{%s}/metadata", ParamAppId, ParamInstId)).To(h.UpdateMetadata)).
		Param(ws.PathParameter(ParamAppId, "applicationId").DataType("string")).
		Param(ws.PathParameter(ParamInstId, "instanceId").DataType("string"))
	// Query for all instances under a particular vip address
	ws.Route(ws.GET(fmt.Sprintf("/vips/{%s}", ParamVip)).To(h.QueryByVipAddress)).
		Param(ws.PathParameter(ParamVip, "vipAddress").DataType("string"))
	// Query for all instances under a particular secure vip address
	ws.Route(ws.GET(fmt.Sprintf("/svips/{%s}", ParamSVip)).To(h.QueryBySVipAddress)).
		Param(ws.PathParameter(ParamSVip, "svipAddress").DataType("string"))
	// Query for handling batch replication request
	ws.Route(ws.POST("/peerreplication/batch").To(h.BatchReplication))
}

func parseAcceptValue(acceptValue string) map[string]bool {
	var values map[string]bool
	blankValues := strings.Split(acceptValue, ",")
	if len(blankValues) > 0 {
		values = make(map[string]bool, len(blankValues))
		for _, blankValue := range blankValues {
			values[strings.TrimSpace(blankValue)] = true
		}
	}
	return values
}

// GetAllApplications 全量拉取服务实例信息
func (h *EurekaServer) GetAllApplications(req *restful.Request, rsp *restful.Response) {
	namespace := readNamespaceFromRequest(req, h.namespace)
	appsRespCache := h.workers.Get(namespace).GetCachedAppsWithLoad()
	remoteAddr := req.Request.RemoteAddr
	acceptValue := getParamFromEurekaRequestHeader(req, restful.HEADER_Accept)
	if err := writeResponse(parseAcceptValue(acceptValue), appsRespCache, req, rsp); nil != err {
		eurekalog.Errorf("[EurekaServer]fail to write applications, client: %s, err: %v", remoteAddr, err)
	}
}

func writePolarisStatusCode(req *restful.Request, statusCode uint32) {
	req.SetAttribute(statusCodeHeader, statusCode)
}

// GetApplication 拉取单个服务实例信息
func (h *EurekaServer) GetApplication(req *restful.Request, rsp *restful.Response) {
	appId := readAppIdFromRequest(req)

	remoteAddr := req.Request.RemoteAddr
	namespace := readNamespaceFromRequest(req, h.namespace)
	appsRespCache := h.workers.Get(namespace).GetCachedAppsWithLoad()
	apps := appsRespCache.AppsResp.Applications
	app := apps.GetApplication(appId)
	if app == nil {
		eurekalog.Errorf("[EurekaServer]service %s not found, client: %s", appId, remoteAddr)
		writePolarisStatusCode(req, api.NotFoundService)
		writeHeader(http.StatusNotFound, rsp)
		return
	}

	appResp := ApplicationResponse{Application: app}
	var output interface{}
	output = appResp.Application

	acceptValue := getParamFromEurekaRequestHeader(req, restful.HEADER_Accept)
	if len(acceptValue) > 0 && acceptValue == restful.MIME_JSON {
		output = appResp
	}
	if err := writeEurekaResponse(acceptValue, output, req, rsp); nil != err {
		eurekalog.Errorf("[EurekaServer]fail to write application, client: %s, err: %v", remoteAddr, err)
	}
}

// GetAppInstance 拉取应用下某个实例的信息
func (h *EurekaServer) GetAppInstance(req *restful.Request, rsp *restful.Response) {
	remoteAddr := req.Request.RemoteAddr
	appId := readAppIdFromRequest(req)
	if len(appId) == 0 {
		eurekalog.Errorf("[EurekaServer] fail to parse request uri, uri: %s, client: %s, err: %s",
			req.Request.RequestURI, remoteAddr, "service name is empty")
		writePolarisStatusCode(req, api.InvalidServiceName)
		writeHeader(http.StatusBadRequest, rsp)
		return
	}
	instId := req.PathParameter(ParamInstId)
	if len(instId) == 0 {
		eurekalog.Errorf("[EUREKA-SERVER] fail to parse request uri, uri: %s, client: %s, err: %s",
			req.Request.RequestURI, remoteAddr, "instance id is required")
		writePolarisStatusCode(req, api.InvalidInstanceID)
		writeHeader(http.StatusBadRequest, rsp)
		return
	}
	namespace := readNamespaceFromRequest(req, h.namespace)
	appsRespCache := h.workers.Get(namespace).GetCachedAppsWithLoad()
	apps := appsRespCache.AppsResp.Applications
	app := apps.GetApplication(appId)
	if app == nil {
		eurekalog.Errorf("[EurekaServer]service %s not found, client: %s", appId, remoteAddr)
		writePolarisStatusCode(req, api.NotFoundService)
		writeHeader(http.StatusNotFound, rsp)
		return
	}
	ins := app.GetInstance(instId)
	if ins == nil {
		eurekalog.Errorf("[EurekaServer]instance %s not found, service: %s, client: %s", instId, appId, remoteAddr)
		writePolarisStatusCode(req, api.NotFoundInstance)
		writeHeader(http.StatusNotFound, rsp)
		return
	}

	insResp := InstanceResponse{InstanceInfo: ins}
	var output interface{}
	output = insResp.InstanceInfo
	acceptValue := getParamFromEurekaRequestHeader(req, restful.HEADER_Accept)
	if len(acceptValue) > 0 && acceptValue == restful.MIME_JSON {
		output = insResp
	}
	if err := writeEurekaResponse(acceptValue, output, req, rsp); nil != err {
		eurekalog.Errorf("[EurekaServer]fail to write instance, client: %s, err: %v", remoteAddr, err)
	}
}

func writeEurekaResponse(acceptValue string, output interface{}, req *restful.Request, rsp *restful.Response) error {
	return writeEurekaResponseWithCode(acceptValue, output, req, rsp, api.ExecuteSuccess)
}

func writeEurekaResponseWithCode(
	acceptValue string, output interface{}, req *restful.Request, rsp *restful.Response, statusCode uint32) error {
	writePolarisStatusCode(req, statusCode)
	var err error
	if len(acceptValue) > 0 && acceptValue == restful.MIME_JSON {
		err = rsp.WriteAsJson(output)
	} else {
		err = rsp.WriteAsXml(output)
	}

	return err
}

const (
	MimeJsonWild = "application/*+json"
)

func writeResponse(acceptValues map[string]bool, appsRespCache *ApplicationsRespCache,
	req *restful.Request, rsp *restful.Response) error {
	writePolarisStatusCode(req, api.ExecuteSuccess)
	var err error
	if len(acceptValues) > 0 && (hasKey(acceptValues, restful.MIME_JSON) || hasKey(acceptValues, MimeJsonWild)) {
		if len(appsRespCache.JsonBytes) > 0 {
			// 直接使用只读缓存返回
			rsp.Header().Set(restful.HEADER_ContentType, restful.MIME_JSON)
			rsp.WriteHeader(http.StatusOK)
			_, err = rsp.Write(appsRespCache.JsonBytes)
		} else {
			err = rsp.WriteAsJson(appsRespCache.AppsResp)
		}
	} else {
		if len(appsRespCache.XmlBytes) > 0 {
			rsp.Header().Set(restful.HEADER_ContentType, restful.MIME_XML)
			rsp.WriteHeader(http.StatusOK)
			_, err = rsp.Write([]byte(xml.Header))
			if err != nil {
				return err
			}
			_, err = rsp.Write(appsRespCache.XmlBytes)
			return err
		}
		err = rsp.WriteAsXml(appsRespCache.AppsResp.Applications)
	}
	return err
}

// GetDeltaApplications 增量拉取服务实例信息
func (h *EurekaServer) GetDeltaApplications(req *restful.Request, rsp *restful.Response) {
	namespace := readNamespaceFromRequest(req, h.namespace)
	work := h.workers.Get(namespace)
	appsRespCache := work.GetDeltaApps()
	if nil == appsRespCache {
		ctx := work.StartWorker()
		if nil != ctx {
			<-ctx.Done()
		}
		appsRespCache = work.GetDeltaApps()
	}
	remoteAddr := req.Request.RemoteAddr
	acceptValue := getParamFromEurekaRequestHeader(req, restful.HEADER_Accept)
	if err := writeResponse(parseAcceptValue(acceptValue), appsRespCache, req, rsp); nil != err {
		eurekalog.Errorf("[EurekaServer]fail to write delta applications, client: %s, err: %v", remoteAddr, err)
	}
}

func convertInstancePorts(instance *InstanceInfo) error {
	var err error
	if nil != instance.Port {
		if err = instance.Port.convertPortValue(); nil != err {
			return err
		}
		if err = instance.Port.convertEnableValue(); nil != err {
			return err
		}
	}
	if nil != instance.SecurePort {
		if err = instance.SecurePort.convertPortValue(); nil != err {
			return err
		}
		if err = instance.SecurePort.convertEnableValue(); nil != err {
			return err
		}
	}
	return nil
}

func checkRegisterRequest(registrationRequest *RegistrationRequest, req *restful.Request, rsp *restful.Response) bool {
	var err error
	remoteAddr := req.Request.RemoteAddr
	if nil == registrationRequest.Instance {
		eurekalog.Errorf("[EUREKA-SERVER] fail to parse register request, uri: %s, client: %s, err: %s",
			req.Request.RequestURI, remoteAddr, "instance content required")
		writePolarisStatusCode(req, api.EmptyRequest)
		writeHeader(http.StatusBadRequest, rsp)
		return false
	}
	if len(registrationRequest.Instance.InstanceId) == 0 && len(registrationRequest.Instance.HostName) == 0 {
		eurekalog.Errorf("[EUREKA-SERVER] fail to parse register request, uri: %s, client: %s, err: %s",
			req.Request.RequestURI, remoteAddr, "instance id required")
		writePolarisStatusCode(req, api.InvalidInstanceID)
		writeHeader(http.StatusBadRequest, rsp)
	}
	err = convertInstancePorts(registrationRequest.Instance)
	if nil != err {
		eurekalog.Errorf("[EUREKA-SERVER] fail to parse instance register request, "+
			"invalid port value, client: %s, err: %v", remoteAddr, err)
		writePolarisStatusCode(req, api.InvalidInstancePort)
		writeHeader(http.StatusBadRequest, rsp)
	}
	return true
}

// RegisterApplication 服务注册
func (h *EurekaServer) RegisterApplication(req *restful.Request, rsp *restful.Response) {
	remoteAddr := req.Request.RemoteAddr
	appId := readAppIdFromRequest(req)

	if len(appId) == 0 {
		eurekalog.Errorf("[EurekaServer] fail to parse request uri, uri: %s, client: %s, err: %s",
			req.Request.RequestURI, remoteAddr, "service name is empty")
		writePolarisStatusCode(req, api.InvalidServiceName)
		writeHeader(http.StatusBadRequest, rsp)
		return
	}
	registrationRequest := &RegistrationRequest{}
	acceptValue := getParamFromEurekaRequestHeader(req, restful.HEADER_ContentType)
	var err error
	if acceptValue == restful.MIME_XML {
		instance := &InstanceInfo{}
		registrationRequest.Instance = instance
		err = req.ReadEntity(&instance)
	} else {
		err = req.ReadEntity(registrationRequest)
	}
	if nil != err {
		eurekalog.Errorf("[EUREKA-SERVER] fail to parse instance register request, uri: %s, client: %s, err: %v",
			req.Request.RequestURI, remoteAddr, err)
		writePolarisStatusCode(req, api.ParseException)
		writeHeader(http.StatusBadRequest, rsp)
		return
	}

	if !checkRegisterRequest(registrationRequest, req, rsp) {
		return
	}

	token, err := getAuthFromEurekaRequestHeader(req)
	if err != nil {
		eurekalog.Infof("[EUREKA-SERVER]instance (instId=%s, appId=%s) get basic auth info fail, code is %d",
			registrationRequest.Instance.InstanceId, appId, api.ExecuteException)
		writePolarisStatusCode(req, api.ExecuteException)
		writeHeader(http.StatusUnauthorized, rsp)
		return
	}

	ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, token)

	namespace := readNamespaceFromRequest(req, h.namespace)
	eurekalog.Infof(
		"[EUREKA-SERVER]received instance register request, "+
			"client: %s, namespace: %s, instId: %s, appId: %s, ipAddr: %s",
		remoteAddr, namespace, registrationRequest.Instance.InstanceId, appId, registrationRequest.Instance.IpAddr)
	code := h.registerInstances(ctx, namespace, appId, registrationRequest.Instance, false)
	if code == api.ExecuteSuccess || code == api.ExistedResource || code == api.SameInstanceRequest {
		eurekalog.Infof(
			"[EUREKA-SERVER]instance (namespace=%s, instId=%s, appId=%s) has been registered successfully,"+
				" code is %d",
			namespace, registrationRequest.Instance.InstanceId, appId, code)
		writePolarisStatusCode(req, code)
		writeHeader(http.StatusNoContent, rsp)
		return
	}
	eurekalog.Errorf("[EUREKA-SERVER]instance (namespace=%s, instId=%s, appId=%s) has been registered failed, "+
		"code is %d",
		namespace, registrationRequest.Instance.InstanceId, appId, code)
	writePolarisStatusCode(req, code)
	writeHeader(int(code/1000), rsp)
}

// UpdateStatus 更新服务状态
func (h *EurekaServer) UpdateStatus(req *restful.Request, rsp *restful.Response) {
	remoteAddr := req.Request.RemoteAddr
	appId := readAppIdFromRequest(req)
	if len(appId) == 0 {
		eurekalog.Errorf("[EurekaServer] fail to parse request uri, uri: %s, client: %s, err: %s",
			req.Request.RequestURI, remoteAddr, "service name is empty")
		writePolarisStatusCode(req, api.InvalidServiceName)
		writeHeader(http.StatusBadRequest, rsp)
		return
	}
	instId := req.PathParameter(ParamInstId)
	if len(instId) == 0 {
		eurekalog.Errorf("[EUREKA-SERVER] fail to parse request uri, uri: %s, client: %s, err: %s",
			req.Request.RequestURI, remoteAddr, "instance id is required")
		writePolarisStatusCode(req, api.InvalidInstanceID)
		writeHeader(http.StatusBadRequest, rsp)
		return
	}
	status := req.QueryParameter(ParamValue)
	namespace := readNamespaceFromRequest(req, h.namespace)
	eurekalog.Infof("[EUREKA-SERVER]received instance updateStatus request, "+
		"client: %s, namespace: %s, instId: %s, appId: %s, status: %s",
		remoteAddr, namespace, instId, appId, status)
	// check status
	if status == StatusUnknown {
		writePolarisStatusCode(req, api.ExecuteSuccess)
		writeHeader(http.StatusOK, rsp)
		return
	}
	ctx := context.WithValue(context.Background(), sourceFromEureka{}, true)
	code := h.updateStatus(ctx, namespace, appId, instId, status, false)
	writePolarisStatusCode(req, code)
	if code == api.ExecuteSuccess || code == api.NoNeedUpdate {
		eurekalog.Infof("[EUREKA-SERVER] instance (namespace=%s, instId=%s, appId=%s) has been updated successfully",
			namespace, instId, appId)
		writeHeader(http.StatusOK, rsp)
		return
	}
	eurekalog.Errorf("[EUREKA-SERVER] instance (namespace=%s, instId=%s, appId=%s) has been updated failed, "+
		"code is %d",
		namespace, instId, appId, code)
	if code == api.NotFoundResource {
		writeHeader(http.StatusNotFound, rsp)
		return
	}
	writeHeader(int(code/1000), rsp)
}

// DeleteStatus 关闭强制隔离
func (h *EurekaServer) DeleteStatus(req *restful.Request, rsp *restful.Response) {
	remoteAddr := req.Request.RemoteAddr
	appId := readAppIdFromRequest(req)
	if len(appId) == 0 {
		eurekalog.Errorf("[EurekaServer] fail to parse request uri, uri: %s, client: %s, err: %s",
			req.Request.RequestURI, remoteAddr, "service name is empty")
		writePolarisStatusCode(req, api.InvalidServiceName)
		writeHeader(http.StatusBadRequest, rsp)
		return
	}
	instId := req.PathParameter(ParamInstId)
	if len(instId) == 0 {
		eurekalog.Errorf("[EUREKA-SERVER] fail to parse request uri, uri: %s, client: %s, err: %s",
			req.Request.RequestURI, remoteAddr, "instance id is required")
		writePolarisStatusCode(req, api.InvalidInstanceID)
		writeHeader(http.StatusBadRequest, rsp)
		return
	}

	namespace := readNamespaceFromRequest(req, h.namespace)

	eurekalog.Infof("[EUREKA-SERVER]received instance status delete request, "+
		"client: %s,namespace=%s, instId=%s, appId=%s",
		remoteAddr, namespace, instId, appId)

	ctx := context.WithValue(context.Background(), sourceFromEureka{}, true)
	code := h.updateStatus(ctx, namespace, appId, instId, StatusUp, false)
	writePolarisStatusCode(req, code)
	if code == api.ExecuteSuccess {
		eurekalog.Infof("[EUREKA-SERVER]instance status (namespace=%s, instId=%s, appId=%s) "+
			"has been deleted successfully",
			namespace, instId, appId)
		writeHeader(http.StatusOK, rsp)
		return
	}
	eurekalog.Errorf("[EUREKA-SERVER]instance status (namespace=%s, instId=%s, appId=%s) "+
		"has been deleted failed, code is %d",
		namespace, instId, appId, code)
	if code == api.NotFoundResource {
		writeHeader(http.StatusNotFound, rsp)
		return
	}
	writeHeader(int(code/1000), rsp)
}

// RenewInstance 更新实例状态
func (h *EurekaServer) RenewInstance(req *restful.Request, rsp *restful.Response) {
	remoteAddr := req.Request.RemoteAddr
	appId := readAppIdFromRequest(req)
	if len(appId) == 0 {
		eurekalog.Errorf("[EurekaServer] fail to parse request uri, uri: %s, client: %s, err: %s",
			req.Request.RequestURI, remoteAddr, "service name is empty")
		writePolarisStatusCode(req, api.InvalidServiceName)
		writeHeader(http.StatusBadRequest, rsp)
		return
	}
	instId := req.PathParameter(ParamInstId)
	if len(instId) == 0 {
		eurekalog.Errorf("[EUREKA-SERVER] fail to parse request uri, uri: %s, client: %s, err: %s",
			req.Request.RequestURI, remoteAddr, "instance id is required")
		writePolarisStatusCode(req, api.InvalidInstanceID)
		writeHeader(http.StatusBadRequest, rsp)
		return
	}
	namespace := readNamespaceFromRequest(req, h.namespace)
	code := h.renew(context.Background(), namespace, appId, instId, false)
	writePolarisStatusCode(req, code)
	if code == api.ExecuteSuccess || code == api.HeartbeatExceedLimit {
		writeHeader(http.StatusOK, rsp)
		return
	}
	eurekalog.Errorf("[EUREKA-SERVER]instance (namespace=%s, instId=%s, appId=%s) heartbeat failed, code is %d",
		namespace, instId, appId, code)
	if code == api.NotFoundResource {
		writeHeader(http.StatusNotFound, rsp)
		return
	}
	writeHeader(int(code/1000), rsp)
}

// CancelInstance 实例反注册
func (h *EurekaServer) CancelInstance(req *restful.Request, rsp *restful.Response) {
	appId := readAppIdFromRequest(req)
	remoteAddr := req.Request.RemoteAddr
	if len(appId) == 0 {
		eurekalog.Errorf("[EurekaServer] fail to parse request uri, uri: %s, client: %s, err: %s",
			req.Request.RequestURI, remoteAddr, "service name is empty")
		writePolarisStatusCode(req, api.InvalidServiceName)
		writeHeader(http.StatusBadRequest, rsp)
		return
	}
	instId := req.PathParameter(ParamInstId)
	if len(instId) == 0 {
		eurekalog.Errorf("[EUREKA-SERVER] fail to parse request uri, uri: %s, client: %s, err: %s",
			req.Request.RequestURI, remoteAddr, "instance id is required")
		writePolarisStatusCode(req, api.InvalidInstanceID)
		writeHeader(http.StatusBadRequest, rsp)
		return
	}
	namespace := readNamespaceFromRequest(req, h.namespace)
	eurekalog.Infof("[EUREKA-SERVER]received instance deregistered request, "+
		"client: %s, namespace: %s, instId: %s, appId: %s",
		remoteAddr, namespace, instId, appId)
	code := h.deregisterInstance(context.Background(), namespace, appId, instId, false)
	writePolarisStatusCode(req, code)
	if code == api.ExecuteSuccess || code == api.NotFoundResource || code == api.SameInstanceRequest {
		writeHeader(http.StatusOK, rsp)
		eurekalog.Infof("[EUREKA-SERVER]instance (namespace=%s, instId=%s, appId=%s) "+
			"has been deregistered successfully, code is %d",
			namespace, instId, appId, code)
		return
	}
	eurekalog.Errorf("[EUREKA-SERVER]instance (namespace=%s, instId=%s, appId=%s) has been deregistered failed,"+
		" code is %d",
		namespace, instId, appId, code)
	writeHeader(int(code/1000), rsp)
}

// GetInstance query instance by id
func (h *EurekaServer) GetInstance(req *restful.Request, rsp *restful.Response) {
	remoteAddr := req.Request.RemoteAddr
	instId := req.PathParameter(ParamInstId)
	if len(instId) == 0 {
		eurekalog.Errorf("[EUREKA-SERVER] fail to parse request uri, uri: %s, client: %s, err: %s",
			req.Request.RequestURI, remoteAddr, "instance id is required")
		writePolarisStatusCode(req, api.InvalidInstanceID)
		writeHeader(http.StatusBadRequest, rsp)
		return
	}
	namespace := readNamespaceFromRequest(req, h.namespace)
	appsRespCache := h.workers.Get(namespace).GetCachedAppsWithLoad()
	apps := appsRespCache.AppsResp.Applications
	instance := apps.GetInstance(instId)
	if nil == instance {
		writePolarisStatusCode(req, api.NotFoundInstance)
		writeHeader(http.StatusNotFound, rsp)
		return
	}
	insResp := InstanceResponse{InstanceInfo: instance}
	var output interface{}
	output = insResp.InstanceInfo
	acceptValue := getParamFromEurekaRequestHeader(req, restful.HEADER_Accept)
	if len(acceptValue) > 0 && acceptValue == restful.MIME_JSON {
		output = insResp
	}
	if err := writeEurekaResponse(acceptValue, output, req, rsp); nil != err {
		eurekalog.Errorf("[EurekaServer]fail to write instance, client: %s, err: %v", remoteAddr, err)
	}
}

// UpdateMetadata updateStatus instance metadata
func (h *EurekaServer) UpdateMetadata(req *restful.Request, rsp *restful.Response) {
	remoteAddr := req.Request.RemoteAddr
	appId := readAppIdFromRequest(req)
	if len(appId) == 0 {
		eurekalog.Errorf("[EurekaServer] fail to parse request uri, uri: %s, client: %s, err: %s",
			req.Request.RequestURI, remoteAddr, "service name is empty")
		writePolarisStatusCode(req, api.InvalidServiceName)
		writeHeader(http.StatusBadRequest, rsp)
		return
	}
	instId := req.PathParameter(ParamInstId)
	if len(instId) == 0 {
		eurekalog.Errorf("[EUREKA-SERVER] fail to parse request uri, uri: %s, client: %s, err: %s",
			req.Request.RequestURI, remoteAddr, "instance id is required")
		writePolarisStatusCode(req, api.InvalidInstanceID)
		writeHeader(http.StatusBadRequest, rsp)
		return
	}
	namespace := readNamespaceFromRequest(req, h.namespace)
	queryValues := req.Request.URL.Query()
	metadataMap := make(map[string]string, len(queryValues))
	for key, values := range queryValues {
		if len(values) == 0 {
			metadataMap[key] = ""
			continue
		}
		metadataMap[key] = values[0]
	}
	code := h.updateMetadata(context.Background(), namespace, appId, instId, metadataMap)
	writePolarisStatusCode(req, code)
	if code == api.ExecuteSuccess {
		eurekalog.Infof("[EUREKA-SERVER]instance metadata (namespace=%s, instId=%s, appId=%s) has been updated successfully",
			namespace, instId, appId)
		writeHeader(http.StatusOK, rsp)
		return
	}
	eurekalog.Errorf("[EUREKA-SERVER]instance metadata (namespace=%s, instId=%s, appId=%s) has been updated failed, "+
		"code is %d",
		namespace, instId, appId, code)
	if code == api.NotFoundResource {
		writeHeader(http.StatusNotFound, rsp)
		return
	}
	writeHeader(int(code/1000), rsp)
}

// QueryByVipAddress query for all instances under a particular vip address
func (h *EurekaServer) QueryByVipAddress(req *restful.Request, rsp *restful.Response) {
	remoteAddr := req.Request.RemoteAddr
	vipAddress := req.PathParameter(ParamVip)
	if len(vipAddress) == 0 {
		eurekalog.Errorf("[EurekaServer] fail to parse request uri, uri: %s, client: %s, err: %s",
			req.Request.RequestURI, remoteAddr, "vip address is empty")
		writePolarisStatusCode(req, api.InvalidParameter)
		writeHeader(http.StatusBadRequest, rsp)
		return
	}

	namespace := readNamespaceFromRequest(req, h.namespace)
	appsRespCache := h.workers.Get(namespace).GetVipApps(VipCacheKey{
		entityType:       entityTypeVip,
		targetVipAddress: formatReadName(vipAddress),
	})
	acceptValue := getParamFromEurekaRequestHeader(req, restful.HEADER_Accept)
	if err := writeResponse(parseAcceptValue(acceptValue), appsRespCache, req, rsp); nil != err {
		eurekalog.Errorf("[EurekaServer]fail to write vip applications, client: %s, err: %v", remoteAddr, err)
	}
}

// QueryBySVipAddress query for all instances under a particular secure vip address
func (h *EurekaServer) QueryBySVipAddress(req *restful.Request, rsp *restful.Response) {
	remoteAddr := req.Request.RemoteAddr
	vipAddress := req.PathParameter(ParamSVip)
	if len(vipAddress) == 0 {
		eurekalog.Errorf("[EurekaServer] fail to parse request uri, uri: %s, client: %s, err: %s",
			req.Request.RequestURI, remoteAddr, "svip address is empty")
		writePolarisStatusCode(req, api.InvalidParameter)
		writeHeader(http.StatusBadRequest, rsp)
		return
	}
	namespace := readNamespaceFromRequest(req, h.namespace)
	appsRespCache := h.workers.Get(namespace).GetVipApps(VipCacheKey{
		entityType:       entityTypeSVip,
		targetVipAddress: formatReadName(vipAddress),
	})
	acceptValue := getParamFromEurekaRequestHeader(req, restful.HEADER_Accept)
	if err := writeResponse(parseAcceptValue(acceptValue), appsRespCache, req, rsp); nil != err {
		eurekalog.Errorf("[EurekaServer]fail to write svip applications, client: %s, err: %v", remoteAddr, err)
	}
}
