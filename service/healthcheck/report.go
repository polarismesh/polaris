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

package healthcheck

import (
	"context"
	"time"

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
)

// checkHeartbeatInstance 检查心跳实例请求参数
// 检查是否存在token，以及 id或者四元组
// 注意：心跳上报只允许从client上报，因此token只会存在req中
func checkHeartbeatInstance(req *apiservice.Instance) (string, *apiservice.Response) {
	if req == nil {
		return "", api.NewInstanceResponse(apimodel.Code_EmptyRequest, req)
	}
	if req.GetId() != nil {
		if req.GetId().GetValue() == "" {
			return "", api.NewInstanceResponse(apimodel.Code_InvalidInstanceID, req)
		}
		return req.GetId().GetValue(), nil
	}
	return utils.CheckInstanceTetrad(req)
}

const max404Count = 3

func (s *Server) checkInstanceExists(ctx context.Context, id string) (int64, *model.Instance, apimodel.Code) {
	ins := s.instanceCache.GetInstance(id)
	if ins != nil {
		return -1, ins, apimodel.Code_ExecuteSuccess
	}
	resp, err := s.defaultChecker.Query(ctx, &plugin.QueryRequest{
		InstanceId: id,
	})
	if nil != err {
		log.Errorf("[healthcheck]fail to query report count by id %s, err: %v", id, err)
		return -1, nil, apimodel.Code_ExecuteSuccess
	}
	if resp.Count > max404Count {
		log.Errorf("[healthcheck] not found heartbeat record by id %s, count: %v", id, resp.Count)
		return resp.Count, nil, apimodel.Code_NotFoundResource
	}
	return resp.Count, nil, apimodel.Code_ExecuteSuccess
}

func (s *Server) getHealthChecker(id string) plugin.HealthChecker {
	insCache := s.cacheProvider.GetInstance(id)
	if insCache == nil {
		insCache = s.cacheProvider.GetSelfServiceInstance(id)
	}
	if insCache == nil {
		return s.defaultChecker
	}
	checker, ok := s.checkers[int32(insCache.HealthCheck().GetType())]
	if !ok {
		return s.defaultChecker
	}
	return checker
}

func (s *Server) doReport(ctx context.Context, instance *apiservice.Instance) *apiservice.Response {
	if !s.hcOpt.IsOpen() || len(s.checkers) == 0 {
		return api.NewResponse(apimodel.Code_HealthCheckNotOpen)
	}
	id, errRsp := checkHeartbeatInstance(instance)
	if errRsp != nil {
		return errRsp
	}
	request := &plugin.ReportRequest{
		QueryRequest: plugin.QueryRequest{
			InstanceId: id,
			Host:       instance.GetHost().GetValue(),
			Port:       instance.GetPort().GetValue(),
		},
		LocalHost:  s.localHost,
		CurTimeSec: time.Now().Unix() - s.timeAdjuster.GetDiff(),
	}
	code, err := s.baseReport(ctx, id, request)
	if err != nil {
		log.Errorf("[Heartbeat][Server] fail to do report for %s:%d, id is %s, err is %v",
			instance.GetHost().GetValue(), instance.GetPort().GetValue(), id, err)
		return api.NewInstanceResponse(apimodel.Code_HeartbeatException, instance)
	}
	return api.NewInstanceResponse(code, instance)
}

func (s *Server) doReports(ctx context.Context, beats []*apiservice.InstanceHeartbeat) *apiservice.Response {
	if !s.hcOpt.IsOpen() || len(s.checkers) == 0 {
		return api.NewResponse(apimodel.Code_HealthCheckNotOpen)
	}
	for i := range beats {
		beat := beats[i]
		request := &plugin.ReportRequest{
			QueryRequest: plugin.QueryRequest{
				InstanceId: beat.InstanceId,
				Host:       beat.Host,
				Port:       beat.Port,
			},
			LocalHost:  s.localHost,
			CurTimeSec: time.Now().Unix() - s.timeAdjuster.GetDiff(),
		}
		code, err := s.baseReport(ctx, beat.InstanceId, request)
		if err != nil {
			log.Errorf("[Heartbeat][Server]fail to do report for %s:%d, id is %s, err is %v",
				beat.GetHost(), beat.GetPort(), beat.GetInstanceId(), err)
			return api.NewInstanceResponse(apimodel.Code_HeartbeatException, nil)
		}
		if code != apimodel.Code_ExecuteSuccess {
			log.Warnf("[Heartbeat][Server] do report for %s:%d, id is %s, code is %v",
				beat.GetHost(), beat.GetPort(), beat.GetInstanceId(), code)
		}
	}
	return api.NewResponse(apimodel.Code_ExecuteSuccess)
}

func (s *Server) baseReport(ctx context.Context, id string, reportReq *plugin.ReportRequest) (apimodel.Code, error) {
	count, ins, code := s.checkInstanceExists(ctx, id)
	checker := s.getHealthChecker(id)
	reportReq.Count = count + 1
	err := checker.Report(ctx, reportReq)
	if nil != ins {
		event := &model.InstanceEvent{
			Id:       id,
			Instance: ins.Proto,
			EType:    model.EventInstanceSendHeartbeat,
		}
		event.InjectMetadata(ctx)
		s.publishInstanceEvent(ins.ServiceID, *event)
	}
	return code, err
}
