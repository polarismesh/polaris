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

	"github.com/emicklei/go-restful"

	api "github.com/polarismesh/polaris-server/common/api/v1"
)

const (
	ParamAppId  string = "appId"
	ParamInstId string = "instId"
	ParamValue  string = "value"
)

/**
 * @brief 注册管理端接口
 */
func (h *EurekaServer) GetEurekaAccessServer() *restful.WebService {
	ws := new(restful.WebService)

	ws.Path("/eureka").Consumes(restful.MIME_JSON, restful.MIME_OCTET, restful.MIME_XML).Produces(restful.MIME_JSON,
		restful.MIME_XML)
	h.addDiscoverAccess(ws)
	return ws
}

/**
 * @brief 增加服务发现接口
 */
func (h *EurekaServer) addDiscoverAccess(ws *restful.WebService) {
	// 应用实例注册
	ws.Route(ws.POST(fmt.Sprintf("/apps/{%s}", ParamAppId)).To(h.RegisterApplication)).
		Param(ws.PathParameter(ParamAppId, "applicationId").DataType("string"))
	// 获取全量服务
	ws.Route(ws.GET("/apps").To(h.GetAllApplications))
	// 获取全量的服务的增量信息
	ws.Route(ws.GET("/apps/delta").To(h.GetDeltaApplications))
	// 获取单个服务的详情
	ws.Route(ws.GET(fmt.Sprintf("/apps/{%s}", ParamAppId)).To(h.GetApplication)).
		Param(ws.PathParameter(ParamAppId, "applicationId").DataType("string"))
	// 获取单个实例的详情
	ws.Route(ws.GET(fmt.Sprintf("/apps/{%s}/{%s}", ParamAppId, ParamInstId)).To(h.GetInstance)).
		Param(ws.PathParameter(ParamAppId, "applicationId").DataType("string")).
		Param(ws.PathParameter(ParamInstId, "instanceId").DataType("string"))
	// 心跳上报
	ws.Route(ws.PUT(fmt.Sprintf("/apps/{%s}/{%s}", ParamAppId, ParamInstId)).To(h.RenewInstance)).
		Param(ws.PathParameter(ParamAppId, "applicationId").DataType("string")).
		Param(ws.PathParameter(ParamInstId, "instanceId").DataType("string"))
		//Param(ws.QueryParameter(ParamStatus, "status").DataType("string"))
	// 实例反注册
	ws.Route(ws.DELETE(fmt.Sprintf("/apps/{%s}/{%s}", ParamAppId, ParamInstId)).To(h.CancelInstance)).
		Param(ws.PathParameter(ParamAppId, "applicationId").DataType("string")).
		Param(ws.PathParameter(ParamInstId, "instanceId").DataType("string"))
	//状态变更
	ws.Route(ws.PUT(fmt.Sprintf("/apps/{%s}/{%s}/status", ParamAppId, ParamInstId)).To(h.UpdateStatus)).
		Param(ws.PathParameter(ParamAppId, "applicationId").DataType("string")).
		Param(ws.PathParameter(ParamInstId, "instanceId").DataType("string"))
	// 删除状态变更
	ws.Route(ws.DELETE(fmt.Sprintf("/apps/{%s}/{%s}/status", ParamAppId, ParamInstId)).To(h.DeleteStatus)).
		Param(ws.PathParameter(ParamAppId, "applicationId").DataType("string")).
		Param(ws.PathParameter(ParamInstId, "instanceId").DataType("string"))
}

//全量拉取服务实例信息
func (h *EurekaServer) GetAllApplications(req *restful.Request, rsp *restful.Response) {
	appsRespCache := h.worker.GetCachedAppsWithLoad()

	acceptValue := getParamFromEurekaRequestHeader(req, restful.HEADER_Accept)
	if err := writeResponse(acceptValue, appsRespCache, req, rsp); nil != err {
		log.Errorf("[EurekaServer]fail to write applications, err is %v", err)
	}
}

func writePolarisStatusCode(req *restful.Request, statusCode uint32) {
	req.SetAttribute(statusCodeHeader, statusCode)
}

// GetApplication 拉取单个服务实例信息
func (h *EurekaServer) GetApplication(req *restful.Request, rsp *restful.Response) {
	appId := strings.ToUpper(req.PathParameter(ParamAppId))

	appsRespCache := h.worker.GetCachedAppsWithLoad()
	apps := appsRespCache.AppsResp.Applications
	app := apps.GetApplication(appId)
	if app == nil {
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
		log.Errorf("[EurekaServer]fail to write application, err is %v", err)
	}
}

// GetInstance 拉取应用下某个实例的信息
func (h *EurekaServer) GetInstance(req *restful.Request, rsp *restful.Response) {
	appId := strings.ToUpper(req.PathParameter(ParamAppId))
	insId := req.PathParameter(ParamInstId)

	appsRespCache := h.worker.GetCachedAppsWithLoad()
	apps := appsRespCache.AppsResp.Applications
	app := apps.GetApplication(appId)
	if app == nil {
		writePolarisStatusCode(req, api.NotFoundService)
		writeHeader(http.StatusNotFound, rsp)
		return
	}
	ins := app.GetInstance(insId)
	if ins == nil {
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
		log.Errorf("[EurekaServer]fail to write instance, err is %v", err)
	}
}

func writeEurekaResponse(acceptValue string, output interface{}, req *restful.Request, rsp *restful.Response) error {
	writePolarisStatusCode(req, api.ExecuteSuccess)
	var err error
	if len(acceptValue) > 0 && acceptValue == restful.MIME_JSON {
		err = rsp.WriteAsJson(output)
	} else {
		err = rsp.WriteAsXml(output)
	}

	return err
}

func writeResponse(
	acceptValue string, appsRespCache *ApplicationsRespCache, req *restful.Request, rsp *restful.Response) error {
	writePolarisStatusCode(req, api.ExecuteSuccess)
	var err error
	if len(acceptValue) > 0 && acceptValue == restful.MIME_JSON {
		if len(appsRespCache.JsonBytes) > 0 {
			//直接使用只读缓存返回
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
		} else {
			err = rsp.WriteAsXml(appsRespCache.AppsResp.Applications)
		}
	}
	return err
}

//增量拉取服务实例信息
func (h *EurekaServer) GetDeltaApplications(req *restful.Request, rsp *restful.Response) {
	appsRespCache := h.worker.GetDeltaApps()
	if nil == appsRespCache {
		ctx := h.worker.StartWorker()
		if nil != ctx {
			<-ctx.Done()
		}
		appsRespCache = h.worker.GetDeltaApps()
	}
	acceptValue := getParamFromEurekaRequestHeader(req, restful.HEADER_Accept)
	if err := writeResponse(acceptValue, appsRespCache, req, rsp); nil != err {
		log.Errorf("[EurekaServer]fail to write delta applications, err is %v", err)
	}
}

func writeHeader(httpStatus int, rsp *restful.Response) {
	rsp.AddHeader(restful.HEADER_ContentType, restful.MIME_XML)
	rsp.WriteHeader(httpStatus)
}

//服务注册
func (h *EurekaServer) RegisterApplication(req *restful.Request, rsp *restful.Response) {
	appId := strings.ToUpper(req.PathParameter(ParamAppId))
	if len(appId) == 0 {
		writePolarisStatusCode(req, api.InvalidServiceName)
		writeHeader(http.StatusBadRequest, rsp)
		return
	}
	remoteAddr := req.Request.RemoteAddr
	registrationRequest := &RegistrationRequest{}
	acceptValue := getParamFromEurekaRequestHeader(req, restful.HEADER_Accept)
	var err error
	if acceptValue == restful.MIME_XML {
		instance := &InstanceInfo{}
		registrationRequest.Instance = instance
		err = req.ReadEntity(&instance)
	} else {
		err = req.ReadEntity(registrationRequest)
	}
	if nil != err {
		log.Errorf("[EUREKA-SERVER] fail to parse instance register request, err is %v", err)
		writePolarisStatusCode(req, api.ParseException)
		writeHeader(http.StatusBadRequest, rsp)
		return
	}
	if nil == registrationRequest.Instance {
		log.Errorf("[EUREKA-SERVER] fail to parse instance register request, instance content required")
		writePolarisStatusCode(req, api.EmptyRequest)
		writeHeader(http.StatusBadRequest, rsp)
		return
	}
	if nil != registrationRequest.Instance.Port {
		if err = registrationRequest.Instance.Port.convertPortValue(); nil != err {
			log.Errorf("[EUREKA-SERVER] fail to parse instance register request from %s, "+
				"invalid insecure port value, err is %v", remoteAddr, err)
			writePolarisStatusCode(req, api.InvalidInstancePort)
			writeHeader(http.StatusBadRequest, rsp)
			return
		}
		if err = registrationRequest.Instance.Port.convertEnableValue(); nil != err {
			log.Errorf("[EUREKA-SERVER] fail to parse instance register request from %s, "+
				"invalid insecure enable value, err is %v", remoteAddr, err)
			writePolarisStatusCode(req, api.InvalidInstancePort)
			writeHeader(http.StatusBadRequest, rsp)
			return
		}
	}
	if nil != registrationRequest.Instance.SecurePort {
		if err = registrationRequest.Instance.SecurePort.convertPortValue(); nil != err {
			log.Errorf("[EUREKA-SERVER] fail to parse instance register request from %s, "+
				"invalid secure port value, err is %v", remoteAddr, err)
			writePolarisStatusCode(req, api.InvalidInstancePort)
			writeHeader(http.StatusBadRequest, rsp)
			return
		}
		if err = registrationRequest.Instance.SecurePort.convertEnableValue(); nil != err {
			log.Errorf("[EUREKA-SERVER] fail to parse instance register request from %s, "+
				"invalid secure enable value, err is %v", remoteAddr, err)
			writePolarisStatusCode(req, api.InvalidInstancePort)
			writeHeader(http.StatusBadRequest, rsp)
			return
		}
	}
	log.Infof("[EUREKA-SERVER]received instance register request from %s, instId=%s, appId=%s, ipAddr is %s",
		remoteAddr, registrationRequest.Instance.InstanceId, appId, registrationRequest.Instance.IpAddr)
	code := h.registerInstances(context.Background(), appId, registrationRequest.Instance)
	if code == api.ExecuteSuccess || code == api.ExistedResource || code == api.SameInstanceRequest {
		log.Infof("[EUREKA-SERVER]instance (instId=%s, appId=%s) has been registered successfully, code is %d",
			registrationRequest.Instance.InstanceId, appId, code)
		writePolarisStatusCode(req, code)
		writeHeader(http.StatusNoContent, rsp)
		return
	}
	log.Errorf("[EUREKA-SERVER]instance (instId=%s, appId=%s) has been registered failed, code is %d",
		registrationRequest.Instance.InstanceId, appId, code)
	writePolarisStatusCode(req, code)
	writeHeader(int(code/1000), rsp)
}

//人工进行状态更新
func (h *EurekaServer) UpdateStatus(req *restful.Request, rsp *restful.Response) {
	appId := req.PathParameter(ParamAppId)
	if len(appId) == 0 {
		writePolarisStatusCode(req, api.InvalidServiceName)
		writeHeader(http.StatusBadRequest, rsp)
		return
	}
	instId := req.PathParameter(ParamInstId)
	if len(instId) == 0 {
		writePolarisStatusCode(req, api.InvalidInstanceID)
		writeHeader(http.StatusBadRequest, rsp)
		return
	}
	remoteAddr := req.Request.RemoteAddr
	status := req.QueryParameter(ParamValue)
	log.Infof("[EUREKA-SERVER]received instance update request from %s, instId=%s, appId=%s, status=%s",
		remoteAddr, instId, appId, status)
	//check status
	if status == StatusUnknown {
		writePolarisStatusCode(req, api.ExecuteSuccess)
		writeHeader(http.StatusOK, rsp)
		return
	}
	code := h.update(context.Background(), appId, instId, status)
	writePolarisStatusCode(req, code)
	if code == api.ExecuteSuccess {
		log.Infof("[EUREKA-SERVER]instance (instId=%s, appId=%s) has been updated successfully", instId, appId)
		writeHeader(http.StatusOK, rsp)
		return
	}
	log.Errorf("[EUREKA-SERVER]instance (instId=%s, appId=%s) has been updated failed, code is %d",
		instId, appId, code)
	if code == api.NotFoundResource {
		writeHeader(http.StatusNotFound, rsp)
		return
	}
	writeHeader(int(code/1000), rsp)
}

// UpdateStatus 关闭强制隔离
func (h *EurekaServer) DeleteStatus(req *restful.Request, rsp *restful.Response) {
	appId := req.PathParameter(ParamAppId)
	if len(appId) == 0 {
		writePolarisStatusCode(req, api.InvalidServiceName)
		writeHeader(http.StatusBadRequest, rsp)
		return
	}
	instId := req.PathParameter(ParamInstId)
	if len(instId) == 0 {
		writePolarisStatusCode(req, api.InvalidInstanceID)
		writeHeader(http.StatusBadRequest, rsp)
		return
	}
	remoteAddr := req.Request.RemoteAddr

	log.Infof("[EUREKA-SERVER]received instance status delete request from %s, instId=%s, appId=%s",
		remoteAddr, instId, appId)

	code := h.update(context.Background(), appId, instId, StatusUp)
	writePolarisStatusCode(req, code)
	if code == api.ExecuteSuccess {
		log.Infof("[EUREKA-SERVER]instance status (instId=%s, appId=%s) has been deleted successfully", instId, appId)
		writeHeader(http.StatusOK, rsp)
		return
	}
	log.Errorf("[EUREKA-SERVER]instance status (instId=%s, appId=%s) has been deleted failed, code is %d",
		instId, appId, code)
	if code == api.NotFoundResource {
		writeHeader(http.StatusNotFound, rsp)
		return
	}
	writeHeader(int(code/1000), rsp)
}

//心跳上报
func (h *EurekaServer) RenewInstance(req *restful.Request, rsp *restful.Response) {
	appId := req.PathParameter(ParamAppId)
	if len(appId) == 0 {
		writePolarisStatusCode(req, api.InvalidServiceName)
		writeHeader(http.StatusBadRequest, rsp)
		return
	}
	instId := req.PathParameter(ParamInstId)
	if len(instId) == 0 {
		writePolarisStatusCode(req, api.InvalidInstanceID)
		writeHeader(http.StatusBadRequest, rsp)
		return
	}
	code := h.renew(context.Background(), appId, instId)
	writePolarisStatusCode(req, code)
	if code == api.ExecuteSuccess || code == api.HeartbeatExceedLimit {
		writeHeader(http.StatusOK, rsp)
		return
	}
	log.Errorf("[EUREKA-SERVER]instance (instId=%s, appId=%s) heartbeat failed, code is %d",
		instId, appId, code)
	if code == api.NotFoundResource {
		writeHeader(http.StatusNotFound, rsp)
		return
	}
	writeHeader(int(code/1000), rsp)
}

//实例反注册
func (h *EurekaServer) CancelInstance(req *restful.Request, rsp *restful.Response) {
	appId := req.PathParameter(ParamAppId)
	if len(appId) == 0 {
		writePolarisStatusCode(req, api.InvalidServiceName)
		writeHeader(http.StatusBadRequest, rsp)
		return
	}
	instId := req.PathParameter(ParamInstId)
	if len(instId) == 0 {
		writePolarisStatusCode(req, api.InvalidInstanceID)
		writeHeader(http.StatusBadRequest, rsp)
		return
	}
	remoteAddr := req.Request.RemoteAddr
	log.Infof("[EUREKA-SERVER]received instance deregistered request from %s, instId=%s, appId=%s",
		remoteAddr, instId, appId)
	code := h.deregisterInstance(context.Background(), appId, instId)
	writePolarisStatusCode(req, code)
	if code == api.ExecuteSuccess || code == api.NotFoundResource || code == api.SameInstanceRequest {
		writeHeader(http.StatusOK, rsp)
		log.Infof(
			"[EUREKA-SERVER]instance (instId=%s, appId=%s) has been deregistered successfully, code is %d",
			instId, appId, code)
		return
	}
	log.Errorf("[EUREKA-SERVER]instance (instId=%s, appId=%s) has been deregistered failed, code is %d",
		instId, appId, code)
	writeHeader(int(code/1000), rsp)
}

func getParamFromEurekaRequestHeader(req *restful.Request, headerName string) string {
	headerValue := req.HeaderParameter(headerName)
	if len(headerValue) > 0 {
		return headerValue
	} else {
		headerValue = req.HeaderParameter(strings.ToLower(headerName))
		return headerValue
	}

}
