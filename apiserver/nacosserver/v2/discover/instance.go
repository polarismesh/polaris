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
	"context"
	"fmt"

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/polarismesh/polaris/apiserver/nacosserver/core"
	nacosmodel "github.com/polarismesh/polaris/apiserver/nacosserver/model"
	nacospb "github.com/polarismesh/polaris/apiserver/nacosserver/v2/pb"
	"github.com/polarismesh/polaris/apiserver/nacosserver/v2/remote"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

func (h *DiscoverServer) handleInstanceRequest(ctx context.Context, req nacospb.BaseRequest,
	meta nacospb.RequestMeta) (nacospb.BaseResponse, error) {
	insReq, ok := req.(*nacospb.InstanceRequest)
	if !ok {
		return nil, remote.ErrorInvalidRequestBodyType
	}

	namespace := nacosmodel.DefaultNacosNamespace
	if len(insReq.Namespace) != 0 {
		namespace = insReq.Namespace
	}
	namespace = nacosmodel.ToPolarisNamespace(namespace)
	svcName := nacosmodel.BuildServiceName(insReq.ServiceName, insReq.GroupName)
	ins := nacosmodel.PrepareSpecInstance(namespace, svcName, &insReq.Instance)
	// 设置连接 ID 作为实例的 metadata 属性信息
	ins.Metadata[nacosmodel.InternalNacosClientConnectionID] = remote.ValueConnID(ctx)

	// Nacos2.x 显示关闭实例的健康检查能力，实例的健康状态和 Grpc Connection 绑定在一起
	ins.EnableHealthCheck = wrapperspb.Bool(false)
	ins.HealthCheck = nil

	var resp *service_manage.Response
	var respType string

	switch insReq.Type {
	case "registerInstance":
		respType = "registerInstance"
		resp = h.discoverSvr.RegisterInstance(ctx, ins)
		insID := resp.GetInstance().GetId().GetValue()
		h.clientManager.addServiceInstance(meta.ConnectionID, model.ServiceKey{
			Namespace: ins.GetNamespace().GetValue(),
			Name:      ins.GetService().GetValue(),
		}, insID)
	case "deregisterInstance":
		respType = "deregisterInstance"
		insID, errRsp := utils.CheckInstanceTetrad(ins)
		if errRsp != nil {
			return nil, &nacosmodel.NacosError{
				ErrCode: int32(errRsp.GetCode().GetValue()),
				ErrMsg:  errRsp.GetInfo().GetValue(),
			}
		}
		ins.Id = utils.NewStringValue(insID)
		resp = h.discoverSvr.DeregisterInstance(ctx, ins)
		h.clientManager.delServiceInstance(meta.ConnectionID, model.ServiceKey{
			Namespace: ins.GetNamespace().GetValue(),
			Name:      ins.GetService().GetValue(),
		}, insID)
	default:
		return nil, &nacosmodel.NacosError{
			ErrCode: int32(nacosmodel.ExceptionCode_InvalidParam),
			ErrMsg:  fmt.Sprintf("Unsupported request type %s", insReq.Type),
		}
	}

	errCode := int(nacosmodel.ErrorCode_Success.Code)
	resultCode := int(nacosmodel.Response_Success.Code)
	success := true

	if resp.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
		success = false
		errCode = int(nacosmodel.ErrorCode_ServerError.Code)
		resultCode = int(nacosmodel.Response_Fail.Code)
	}

	return &nacospb.InstanceResponse{
		Response: &nacospb.Response{
			ResultCode: resultCode,
			ErrorCode:  errCode,
			Success:    success,
			Message:    resp.GetInfo().GetValue(),
		},
		Type: respType,
	}, nil
}

func (h *DiscoverServer) handlePersistentInstanceRequest(ctx context.Context, req nacospb.BaseRequest,
	meta nacospb.RequestMeta) (nacospb.BaseResponse, error) {
	insReq, ok := req.(*nacospb.PersistentInstanceRequest)
	if !ok {
		return nil, remote.ErrorInvalidRequestBodyType
	}

	namespace := nacosmodel.DefaultNacosNamespace
	if len(insReq.Namespace) != 0 {
		namespace = insReq.Namespace
	}
	namespace = nacosmodel.ToPolarisNamespace(namespace)
	svcName := nacosmodel.BuildServiceName(insReq.ServiceName, insReq.GroupName)
	ins := nacosmodel.PrepareSpecInstance(namespace, svcName, &insReq.Instance)
	ins.EnableHealthCheck = wrapperspb.Bool(false)
	ins.HealthCheck = nil

	var resp *service_manage.Response
	var respType string

	errCode := int(nacosmodel.ErrorCode_Success.Code)
	resultCode := int(nacosmodel.Response_Success.Code)
	success := true

	switch insReq.Type {
	case "registerInstance":
		respType = "registerInstance"
		resp = h.discoverSvr.RegisterInstance(ctx, ins)
	case "deregisterInstance":
		respType = "deregisterInstance"
		insID, errRsp := utils.CheckInstanceTetrad(ins)
		if errRsp != nil {
			return nil, &nacosmodel.NacosError{
				ErrCode: int32(errRsp.GetCode().GetValue()),
				ErrMsg:  errRsp.GetInfo().GetValue(),
			}
		}
		ins.Id = utils.NewStringValue(insID)
		resp = h.discoverSvr.DeregisterInstance(ctx, ins)
	default:
		return nil, &nacosmodel.NacosError{
			ErrCode: int32(nacosmodel.ExceptionCode_InvalidParam),
			ErrMsg:  fmt.Sprintf("Unsupported request type %s", insReq.Type),
		}
	}

	return &nacospb.InstanceResponse{
		Response: &nacospb.Response{
			ResultCode: resultCode,
			ErrorCode:  errCode,
			Success:    success,
			Message:    resp.GetInfo().GetValue(),
		},
		Type: respType,
	}, nil
}

func (h *DiscoverServer) handleBatchInstanceRequest(ctx context.Context, req nacospb.BaseRequest,
	meta nacospb.RequestMeta) (nacospb.BaseResponse, error) {
	batchInsReq, ok := req.(*nacospb.BatchInstanceRequest)
	if !ok {
		return nil, remote.ErrorInvalidRequestBodyType
	}

	namespace := nacosmodel.DefaultNacosNamespace
	if len(batchInsReq.Namespace) != 0 {
		namespace = batchInsReq.Namespace
	}
	namespace = nacosmodel.ToPolarisNamespace(namespace)
	var (
		svcName   = nacosmodel.BuildServiceName(batchInsReq.ServiceName, batchInsReq.GroupName)
		batchResp = api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	)

	ctx = context.WithValue(ctx, utils.ContextOpenAsyncRegis, true)
	switch batchInsReq.Type {
	case "batchRegisterInstance":
		for i := range batchInsReq.Instances {
			insReq := batchInsReq.Instances[i]
			ins := nacosmodel.PrepareSpecInstance(namespace, svcName, insReq)
			ins.Metadata[nacosmodel.InternalNacosClientConnectionID] = remote.ValueConnID(ctx)
			// 显示关闭实例的健康检查能力
			ins.EnableHealthCheck = wrapperspb.Bool(false)
			ins.HealthCheck = nil
			resp := h.discoverSvr.RegisterInstance(ctx, ins)
			api.Collect(batchResp, resp)
			if resp.GetCode().GetValue() == uint32(apimodel.Code_ExecuteSuccess) {
				insID := resp.GetInstance().GetId().GetValue()
				h.clientManager.addServiceInstance(meta.ConnectionID, model.ServiceKey{
					Namespace: ins.GetNamespace().GetValue(),
					Name:      ins.GetService().GetValue(),
				}, insID)
			} else {
				nacoslog.Error("[NACOS-V2][Instance] batch register fail", zap.String("namespace", namespace),
					zap.String("service", ins.GetService().GetValue()), zap.String("ip", insReq.IP),
					zap.Int32("port", insReq.Port), zap.String("msg", resp.GetInfo().GetValue()))
			}
		}
	default:
		return nil, &nacosmodel.NacosError{
			ErrCode: int32(nacosmodel.ExceptionCode_InvalidParam),
			ErrMsg:  fmt.Sprintf("Unsupported request type %s", batchInsReq.Type),
		}
	}

	errCode := int(nacosmodel.ErrorCode_Success.Code)
	resultCode := int(nacosmodel.Response_Success.Code)
	success := true

	if batchResp.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
		success = false
		errCode = int(nacosmodel.ErrorCode_ServerError.Code)
		resultCode = int(nacosmodel.Response_Fail.Code)
	}

	return &nacospb.BatchInstanceResponse{
		Response: &nacospb.Response{
			ResultCode: resultCode,
			ErrorCode:  errCode,
			Success:    success,
			Message:    batchResp.GetInfo().GetValue(),
		},
		Type: "batchRegisterInstance",
	}, nil
}

func (h *DiscoverServer) HandleClientConnect(ctx context.Context, client *remote.Client) {
	h.clientManager.addConnectionClientIfAbsent(client.ID)
}

func (h *DiscoverServer) HandleClientDisConnect(ctx context.Context, client *remote.Client) {
	nacoslog.Info("[NACOS-CORE][PushCenter] remove WatchClient", zap.String("id", client.ID))
	grpcPushSvr := h.pushCenter.(*GrpcPushCenter)
	grpcPushSvr.RemoveClientIf(func(s string, wc *core.WatchClient) bool {
		return wc.ID() == client.ID
	})

	connClient, ok := h.clientManager.delClient(client.ID)
	if !ok {
		nacoslog.Info("[NACOS-V2][Connection] not found target ConnectionClient, skip dis-connect event")
		return
	}

	connClient.RangePublishInstance(func(svc model.ServiceKey, ids []string) {
		req := make([]*service_manage.Instance, 0, len(ids))
		for i := range ids {
			req = append(req, &service_manage.Instance{
				Id: utils.NewStringValue(ids[i]),
			})
		}
		if len(req) == 0 {
			return
		}
		nacoslog.Info("[NACOS-V2][Connection] receive client disconnect event, do deregist all publish instance",
			zap.String("conn-id", connClient.ConnID), zap.Any("svc", svc), zap.Strings("instance-ids", ids))
		h.clientManager.delServiceInstance(connClient.ConnID, model.ServiceKey{
			Namespace: svc.Namespace,
			Name:      svc.Name,
		}, ids...)
		resp := h.originDiscoverSvr.DeleteInstances(ctx, req)
		if resp.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
			nacoslog.Error("[NACOS-V2][Connection] deregister all instance fail", zap.String("conn-id", connClient.ConnID),
				zap.Any("svc", svc), zap.String("msg", resp.GetInfo().GetValue()))
		}
	})

	connClient.Destroy()
}
