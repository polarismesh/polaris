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
	"github.com/emicklei/go-restful/v3"
	"github.com/golang/protobuf/ptypes/wrappers"
	"net/http"
	"strings"
	"time"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
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
	log.Infof("[EUREKA-SERVER] received replicate request %+v", req)
	sourceSvrName := req.HeaderParameter(headerIdentityName)
	if sourceSvrName == valueIdentityName {
		// we should not process the replication from polaris
		writeHeader(http.StatusOK, rsp)
		return
	}
	remoteAddr := req.Request.RemoteAddr
	replicateRequest := &ReplicationList{}
	var err error
	err = req.ReadEntity(replicateRequest)
	if nil != err {
		log.Errorf("[EUREKA-SERVER] fail to parse peer replicate request, uri: %s, client: %s, err: %v",
			req.Request.RequestURI, remoteAddr, err)
		writePolarisStatusCode(req, api.ParseException)
		writeHeader(http.StatusOK, rsp)
		return
	}
	token, err := getAuthFromEurekaRequestHeader(req)
	if err != nil {
		log.Infof("[EUREKA-SERVER]replicate request get basic auth info fail, code is %d", api.ExecuteException)
		writePolarisStatusCode(req, api.ExecuteException)
		writeHeader(http.StatusOK, rsp)
		return
	}
	batchResponse := &ReplicationListResponse{}
	var resultCode uint32
	for _, instanceInfo := range replicateRequest.ReplicationList {
		resp, code := h.dispatch(instanceInfo, token)
		if code != api.ExecuteSuccess {
			resultCode = code
			log.Warnf("[EUREKA-SERVER] fail to process replicate instance request, code is %d, action %s, instance %s, app %s",
				code, instanceInfo.Action, instanceInfo.Id, instanceInfo.AppName)
		}
		batchResponse.ResponseList = append(batchResponse.ResponseList, resp)
	}
	writePolarisStatusCode(req, resultCode)
	if err := writeEurekaResponse(restful.MIME_JSON, batchResponse, req, rsp); nil != err {
		log.Errorf("[EurekaServer]fail to write replicate response, client: %s, err: %v", remoteAddr, err)
	}
}

func (h *EurekaServer) dispatch(replicationInstance *ReplicationInstance, token string) (*ReplicationInstanceResponse, uint32) {
	appName := replicationInstance.AppName
	ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, token)
	var code uint32
	log.Debugf("[EurekaServer]dispatch replicate request %+v", replicationInstance)
	if nil != replicationInstance.InstanceInfo {
		_ = convertInstancePorts(replicationInstance.InstanceInfo)
		log.Debugf("[EurekaServer]dispatch replicate instance %+v, port %+v, sport %+v",
			replicationInstance.InstanceInfo, replicationInstance.InstanceInfo.Port, replicationInstance.InstanceInfo.SecurePort)
	}
	switch replicationInstance.Action {
	case actionRegister:
		instanceInfo := replicationInstance.InstanceInfo
		code = h.registerInstances(ctx, appName, instanceInfo, true)
		if code == api.ExecuteSuccess || code == api.ExistedResource || code == api.SameInstanceRequest {
			code = api.ExecuteSuccess
		}
	case actionHeartbeat:
		instanceId := replicationInstance.Id
		code = h.renew(ctx, appName, instanceId)
		if code == api.ExecuteSuccess || code == api.HeartbeatExceedLimit {
			code = api.ExecuteSuccess
		}
	case actionCancel:
		instanceId := replicationInstance.Id
		code = h.deregisterInstance(ctx, appName, instanceId)
		if code == api.ExecuteSuccess || code == api.NotFoundResource || code == api.SameInstanceRequest {
			code = api.ExecuteSuccess
		}
	case actionStatusUpdate:
		status := replicationInstance.Status
		instanceId := replicationInstance.Id
		code = h.updateStatus(ctx, appName, instanceId, status)
	case actionDeleteStatusOverride:
		instanceId := replicationInstance.Id
		code = h.updateStatus(ctx, appName, instanceId, StatusUp)
	}
	return &ReplicationInstanceResponse{
		StatusCode: http.StatusOK,
	}, code
}

func eventToInstance(event *model.InstanceEvent, appName string, curTimeMilli int64) *InstanceInfo {
	instance := &api.Instance{
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
	if e.Namespace != h.namespace {
		// only process the service in same namespace
		return false
	}
	metadata := e.Instance.GetMetadata()
	if len(metadata) > 0 {
		if _, ok := metadata[MetadataReplicate]; ok {
			// we should not replicate around
			return false
		}
	}
	return true
}

func (h *EurekaServer) handleInstanceEvent(ctx context.Context, i interface{}) error {
	e := i.(model.InstanceEvent)
	if !h.shouldReplicate(e) {
		return nil
	}
	appName := strings.ToUpper(e.Service)
	curTimeMilli := time.Now().UnixMilli()
	switch e.EType {
	case model.EventInstanceOnline:
		instanceInfo := eventToInstance(&e, appName, curTimeMilli)
		h.replicateWorker.AddReplicateTask(&ReplicationInstance{
			AppName:            appName,
			Id:                 e.Id,
			LastDirtyTimestamp: curTimeMilli,
			Status:             StatusUp,
			InstanceInfo:       instanceInfo,
			Action:             actionRegister,
		})
	case model.EventInstanceOffline:
		h.replicateWorker.AddReplicateTask(&ReplicationInstance{
			AppName: appName,
			Id:      e.Id,
			Action:  actionCancel,
		})
	case model.EventInstanceSendHeartbeat:
		instanceInfo := eventToInstance(&e, appName, curTimeMilli)
		rInstance := &ReplicationInstance{
			AppName:      appName,
			Id:           e.Id,
			Status:       StatusUp,
			InstanceInfo: instanceInfo,
			Action:       actionHeartbeat,
		}
		if e.Instance.GetIsolate().GetValue() {
			rInstance.OverriddenStatus = StatusOutOfService
		}
		h.replicateWorker.AddReplicateTask(rInstance)
	case model.EventInstanceTurnHealth:
		h.replicateWorker.AddReplicateTask(&ReplicationInstance{
			AppName:            appName,
			Id:                 e.Id,
			LastDirtyTimestamp: curTimeMilli,
			Status:             StatusUp,
			Action:             actionStatusUpdate,
		})
	case model.EventInstanceTurnUnHealth:
		h.replicateWorker.AddReplicateTask(&ReplicationInstance{
			AppName:            appName,
			Id:                 e.Id,
			LastDirtyTimestamp: curTimeMilli,
			Status:             StatusDown,
			Action:             actionStatusUpdate,
		})
	case model.EventInstanceOpenIsolate:
		h.replicateWorker.AddReplicateTask(&ReplicationInstance{
			AppName:            appName,
			Id:                 e.Id,
			LastDirtyTimestamp: curTimeMilli,
			OverriddenStatus:   StatusOutOfService,
			Action:             actionHeartbeat,
		})
	case model.EventInstanceCloseIsolate:
		h.replicateWorker.AddReplicateTask(&ReplicationInstance{
			AppName:            appName,
			Id:                 e.Id,
			LastDirtyTimestamp: curTimeMilli,
			Action:             actionDeleteStatusOverride,
		})

	}
	return nil
}
