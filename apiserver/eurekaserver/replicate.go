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
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/emicklei/go-restful/v3"
	"github.com/golang/protobuf/ptypes/wrappers"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/service"
)

const (
	actionRegister             = "Register"
	actionHeartbeat            = "Heartbeat"
	actionCancel               = "Cancel"
	actionStatusUpdate         = "StatusUpdate"
	actionDeleteStatusOverride = "DeleteStatusOverride"
)

const (
	headerIdentityName    = "DiscoveryIdentity-Name"
	headerIdentityVersion = "DiscoveryIdentity-Version"
	headerIdentityId      = "DiscoveryIdentity-Id"
	valueIdentityName     = "PolarisServer"
)

// BatchReplication do the server request replication
func (h *EurekaServer) BatchReplication(req *restful.Request, rsp *restful.Response) {
	eurekalog.Infof("[EUREKA-SERVER] received replicate request %+v", req)
	sourceSvrName := req.HeaderParameter(headerIdentityName)
	remoteAddr := req.Request.RemoteAddr
	if sourceSvrName == valueIdentityName {
		// we should not process the replication from polaris
		batchResponse := &ReplicationListResponse{ResponseList: []*ReplicationInstanceResponse{}}
		if err := writeEurekaResponse(restful.MIME_JSON, batchResponse, req, rsp); nil != err {
			eurekalog.Errorf("[EurekaServer]fail to write replicate response, client: %s, err: %v", remoteAddr, err)
		}
		return
	}
	replicateRequest := &ReplicationList{}
	var err error
	err = req.ReadEntity(replicateRequest)
	if nil != err {
		eurekalog.Errorf("[EUREKA-SERVER] fail to parse peer replicate request, uri: %s, client: %s, err: %v",
			req.Request.RequestURI, remoteAddr, err)
		writePolarisStatusCode(req, api.ParseException)
		writeHeader(http.StatusBadRequest, rsp)
		return
	}
	token, err := getAuthFromEurekaRequestHeader(req)
	if err != nil {
		eurekalog.Infof("[EUREKA-SERVER]replicate request get basic auth info fail, code is %d", api.ExecuteException)
		writePolarisStatusCode(req, api.ExecuteException)
		writeHeader(http.StatusForbidden, rsp)
		return
	}
	namespace := readNamespaceFromRequest(req, h.namespace)
	batchResponse, resultCode := h.doBatchReplicate(replicateRequest, token, namespace)
	if err := writeEurekaResponseWithCode(restful.MIME_JSON, batchResponse, req, rsp, resultCode); nil != err {
		eurekalog.Errorf("[EurekaServer]fail to write replicate response, client: %s, err: %v", remoteAddr, err)
	}
}

func (h *EurekaServer) doBatchReplicate(
	replicateRequest *ReplicationList, token string, namespace string) (*ReplicationListResponse, uint32) {
	batchResponse := &ReplicationListResponse{}
	var resultCode = api.ExecuteSuccess
	itemCount := len(replicateRequest.ReplicationList)
	if itemCount == 0 {
		return batchResponse, resultCode
	}
	batchResponse.ResponseList = make([]*ReplicationInstanceResponse, itemCount)
	wg := &sync.WaitGroup{}
	wg.Add(itemCount)
	mutex := &sync.Mutex{}
	for i, inst := range replicateRequest.ReplicationList {
		go func(idx int, instanceInfo *ReplicationInstance) {
			defer wg.Done()
			resp, code := h.dispatch(instanceInfo, token, namespace)
			if code != api.ExecuteSuccess {
				atomic.CompareAndSwapUint32(&resultCode, api.ExecuteSuccess, code)
				eurekalog.Warnf("[EUREKA-SERVER] fail to process replicate instance request, code is %d, "+
					"action %s, instance %s, app %s",
					code, instanceInfo.Action, instanceInfo.Id, instanceInfo.AppName)
			}
			mutex.Lock()
			batchResponse.ResponseList[idx] = resp
			mutex.Unlock()
		}(i, inst)
	}
	wg.Wait()
	return batchResponse, resultCode
}

func (h *EurekaServer) dispatch(
	replicationInstance *ReplicationInstance, token string, namespace string) (*ReplicationInstanceResponse, uint32) {
	appName := formatReadName(replicationInstance.AppName)
	ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, token)
	var retCode = api.ExecuteSuccess
	eurekalog.Debugf("[EurekaServer]dispatch replicate request %+v", replicationInstance)
	if nil != replicationInstance.InstanceInfo {
		_ = convertInstancePorts(replicationInstance.InstanceInfo)
		eurekalog.Debugf("[EurekaServer]dispatch replicate instance %+v, port %+v, sport %+v",
			replicationInstance.InstanceInfo, replicationInstance.InstanceInfo.Port, replicationInstance.InstanceInfo.SecurePort)
	}
	switch replicationInstance.Action {
	case actionRegister:
		instanceInfo := replicationInstance.InstanceInfo
		retCode = h.registerInstances(ctx, namespace, appName, instanceInfo, true)
		if retCode == api.ExecuteSuccess || retCode == api.ExistedResource || retCode == api.SameInstanceRequest {
			retCode = api.ExecuteSuccess
		}
	case actionHeartbeat:
		instanceId := replicationInstance.Id
		retCode = h.renew(ctx, namespace, appName, instanceId, true)
		if retCode == api.ExecuteSuccess || retCode == api.HeartbeatExceedLimit {
			retCode = api.ExecuteSuccess
		}
	case actionCancel:
		instanceId := replicationInstance.Id
		retCode = h.deregisterInstance(ctx, namespace, appName, instanceId, true)
		if retCode == api.ExecuteSuccess || retCode == api.NotFoundResource || retCode == api.SameInstanceRequest {
			retCode = api.ExecuteSuccess
		}
	case actionStatusUpdate:
		status := replicationInstance.Status
		instanceId := replicationInstance.Id
		retCode = h.updateStatus(ctx, namespace, appName, instanceId, status, true)
	case actionDeleteStatusOverride:
		instanceId := replicationInstance.Id
		retCode = h.updateStatus(ctx, namespace, appName, instanceId, StatusUp, true)
	}

	statusCode := http.StatusOK
	if retCode == api.NotFoundResource {
		statusCode = http.StatusNotFound
	}
	return &ReplicationInstanceResponse{
		StatusCode: statusCode,
	}, retCode
}

func eventToInstance(event *model.InstanceEvent, appName string, curTimeMilli int64) *InstanceInfo {
	instance := &apiservice.Instance{
		Id:                &wrappers.StringValue{Value: event.Id},
		Host:              &wrappers.StringValue{Value: event.Instance.GetHost().GetValue()},
		Port:              &wrappers.UInt32Value{Value: event.Instance.GetPort().GetValue()},
		Protocol:          &wrappers.StringValue{Value: event.Instance.GetProtocol().GetValue()},
		Version:           &wrappers.StringValue{Value: event.Instance.GetVersion().GetValue()},
		Priority:          &wrappers.UInt32Value{Value: event.Instance.GetPriority().GetValue()},
		Weight:            &wrappers.UInt32Value{Value: event.Instance.GetWeight().GetValue()},
		EnableHealthCheck: &wrappers.BoolValue{Value: event.Instance.GetEnableHealthCheck().GetValue()},
		HealthCheck:       event.Instance.GetHealthCheck(),
		Healthy:           &wrappers.BoolValue{Value: event.Instance.GetHealthy().GetValue()},
		Isolate:           &wrappers.BoolValue{Value: event.Instance.GetIsolate().GetValue()},
		Location:          event.Instance.GetLocation(),
		Metadata:          event.Instance.GetMetadata(),
	}
	if event.EType == model.EventInstanceTurnHealth {
		instance.Healthy = &wrappers.BoolValue{Value: true}
	} else if event.EType == model.EventInstanceTurnUnHealth {
		instance.Healthy = &wrappers.BoolValue{Value: false}
	} else if event.EType == model.EventInstanceOpenIsolate {
		instance.Isolate = &wrappers.BoolValue{Value: true}
	} else if event.EType == model.EventInstanceCloseIsolate {
		instance.Isolate = &wrappers.BoolValue{Value: false}
	}
	return buildInstance(appName, instance, curTimeMilli)
}

func (h *EurekaServer) shouldReplicate(e model.InstanceEvent) bool {
	if h.replicateWorkers == nil {
		return false
	}
	if _, exist := h.replicateWorkers.Get(e.Namespace); !exist {
		return false
	}

	if len(e.Service) == 0 {
		eurekalog.Warnf("[EUREKA]fail to replicate, service name is empty for event %s", e)
		return false
	}
	metadata := e.MetaData
	if len(metadata) > 0 {
		if value, ok := metadata[MetadataReplicate]; ok {
			// we should not replicate around
			isReplicate, _ := strconv.ParseBool(value)
			return !isReplicate
		}
	}
	return true
}

type EurekaInstanceEventHandler struct {
	*service.BaseInstanceEventHandler
	svr *EurekaServer
}

func (e *EurekaInstanceEventHandler) OnEvent(ctx context.Context, any2 any) error {
	return e.svr.handleInstanceEvent(ctx, any2)
}

func (h *EurekaServer) handleInstanceEvent(ctx context.Context, i interface{}) error {
	e := i.(model.InstanceEvent)
	if !h.shouldReplicate(e) {
		return nil
	}
	namespace := e.Namespace
	appName := formatReadName(e.Service)
	curTimeMilli := time.Now().UnixMilli()
	eurekaInstanceId := e.Id
	if e.Instance.Metadata != nil {
		if _, ok := e.Instance.Metadata[MetadataInstanceId]; ok {
			eurekaInstanceId = e.Instance.Metadata[MetadataInstanceId]
		}
	}
	if _, ok := e.MetaData[MetadataInstanceId]; ok {
		eurekaInstanceId = e.MetaData[MetadataInstanceId]
	}
	replicateWorker, _ := h.replicateWorkers.Get(namespace)
	switch e.EType {
	case model.EventInstanceOnline, model.EventInstanceUpdate, model.EventInstanceTurnHealth:
		instanceInfo := eventToInstance(&e, appName, curTimeMilli)
		replicateWorker.AddReplicateTask(&ReplicationInstance{
			AppName:            appName,
			Id:                 eurekaInstanceId,
			LastDirtyTimestamp: curTimeMilli,
			Status:             instanceInfo.Status,
			InstanceInfo:       instanceInfo,
			Action:             actionRegister,
		})
	case model.EventInstanceOffline, model.EventInstanceTurnUnHealth:
		replicateWorker.AddReplicateTask(&ReplicationInstance{
			AppName: appName,
			Id:      eurekaInstanceId,
			Action:  actionCancel,
		})
	case model.EventInstanceSendHeartbeat:
		instanceInfo := eventToInstance(&e, appName, curTimeMilli)
		rInstance := &ReplicationInstance{
			AppName:      appName,
			Id:           eurekaInstanceId,
			Status:       instanceInfo.Status,
			InstanceInfo: instanceInfo,
			Action:       actionHeartbeat,
		}
		replicateWorker.AddReplicateTask(rInstance)
	case model.EventInstanceOpenIsolate, model.EventInstanceCloseIsolate:
		replicateWorker.AddReplicateTask(&ReplicationInstance{
			AppName:            appName,
			Id:                 eurekaInstanceId,
			LastDirtyTimestamp: curTimeMilli,
			Status:             parseStatus(e.Instance),
			Action:             actionStatusUpdate,
		})

	}
	return nil
}
