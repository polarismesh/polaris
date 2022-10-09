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

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
)

// checkHeartbeatInstance 检查心跳实例请求参数
// 检查是否存在token，以及 id或者四元组
// 注意：心跳上报只允许从client上报，因此token只会存在req中
func checkHeartbeatInstance(req *api.Instance) (string, *api.Response) {
	if req == nil {
		return "", api.NewInstanceResponse(api.EmptyRequest, req)
	}
	if req.GetId() != nil {
		if req.GetId().GetValue() == "" {
			return "", api.NewInstanceResponse(api.InvalidInstanceID, req)
		}
		return req.GetId().GetValue(), nil
	}
	return utils.CheckInstanceTetrad(req)
}

func (s *Server) doReport(ctx context.Context, instance *api.Instance) *api.Response {
	if len(s.checkers) == 0 {
		return api.NewResponse(api.HealthCheckNotOpen)
	}
	id, errRsp := checkHeartbeatInstance(instance)
	if errRsp != nil {
		return errRsp
	}

	ins := s.instanceCache.GetInstance(id)
	if ins == nil {
		return api.NewResponse(api.NotFoundResource)
	}

	instance.Id = utils.NewStringValue(id)
	insCache := s.cacheProvider.GetInstance(id)
	if insCache == nil {
		return api.NewInstanceResponse(api.HeartbeatOnDisabledIns, instance)
	}
	checker, ok := s.checkers[int32(insCache.HealthCheck().GetType())]
	if !ok {
		return api.NewInstanceResponse(api.HeartbeatTypeNotFound, instance)
	}
	request := &plugin.ReportRequest{
		QueryRequest: plugin.QueryRequest{
			InstanceId: id,
			Host:       insCache.Host(),
			Port:       insCache.Port(),
		},
		LocalHost:  s.localHost,
		CurTimeSec: time.Now().Unix() - s.timeAdjuster.GetDiff(),
	}
	err := checker.Report(request)
	if err != nil {
		log.Errorf("[Heartbeat][Server]fail to do report for %s:%d, id is %s, err is %v",
			insCache.Host(), insCache.Port(), id, err)
		return api.NewInstanceResponse(api.HeartbeatException, instance)
	}
	return api.NewInstanceResponse(api.ExecuteSuccess, instance)
}
