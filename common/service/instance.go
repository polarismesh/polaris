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

package service

import (
	"strings"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

type NamespaceSet struct {
	container map[string]*model.Namespace
}

func NewNamespaceSet() *NamespaceSet {
	return &NamespaceSet{
		container: make(map[string]*model.Namespace),
	}
}

func (set *NamespaceSet) Add(val *model.Namespace) {
	set.container[val.Name] = val
}

func (set *NamespaceSet) Remove(val *model.Namespace) {
	delete(set.container, val.Name)
}

func (set *NamespaceSet) ToSlice() []*model.Namespace {
	ret := make([]*model.Namespace, 0, len(set.container))

	for _, v := range set.container {
		ret = append(ret, v)
	}

	return ret
}

func (set *NamespaceSet) Range(fn func(val *model.Namespace) bool) {
	for _, v := range set.container {
		if !fn(v) {
			break
		}
	}
}

type ServiceSet struct {
	container map[string]*model.Service
}

func NewServiceSet() *ServiceSet {
	return &ServiceSet{
		container: make(map[string]*model.Service),
	}
}

func (set *ServiceSet) Add(val *model.Service) {
	set.container[val.ID] = val
}

func (set *ServiceSet) Remove(val *model.Service) {
	delete(set.container, val.ID)
}

func (set *ServiceSet) ToSlice() []*model.Service {
	ret := make([]*model.Service, 0, len(set.container))

	for _, v := range set.container {
		ret = append(ret, v)
	}

	return ret
}

func (set *ServiceSet) Range(fn func(val *model.Service) bool) {
	for _, v := range set.container {
		if !fn(v) {
			break
		}
	}
}

// CreateInstanceModel 创建存储层服务实例模型
func CreateInstanceModel(serviceID string, req *api.Instance) *model.Instance {
	// 默认为健康的
	healthy := true
	if req.GetHealthy() != nil {
		healthy = req.GetHealthy().GetValue()
	}

	// 默认为不隔离的
	isolate := false
	if req.GetIsolate() != nil {
		isolate = req.GetIsolate().GetValue()
	}

	// 权重默认是100
	var weight uint32 = 100
	if req.GetWeight() != nil {
		weight = req.GetWeight().GetValue()
	}

	instance := &model.Instance{
		ServiceID: serviceID,
	}

	protoIns := &api.Instance{
		Id:       req.GetId(),
		Host:     utils.NewStringValue(strings.TrimSpace(req.GetHost().GetValue())),
		VpcId:    req.GetVpcId(),
		Port:     req.GetPort(),
		Protocol: req.GetProtocol(),
		Version:  req.GetVersion(),
		Priority: req.GetPriority(),
		Weight:   utils.NewUInt32Value(weight),
		Healthy:  utils.NewBoolValue(healthy),
		Isolate:  utils.NewBoolValue(isolate),
		Location: req.Location,
		Metadata: req.Metadata,
		LogicSet: req.GetLogicSet(),
		Revision: utils.NewStringValue(utils.NewUUID()), // 更新版本号
	}

	// health Check，healthCheck不能为空，且没有显示把enable_health_check置为false
	// 如果create的时候，打开了healthCheck，那么实例模式是unhealthy，必须要一次心跳才会healthy
	if req.GetHealthCheck().GetHeartbeat() != nil &&
		(req.GetEnableHealthCheck() == nil || req.GetEnableHealthCheck().GetValue()) {
		protoIns.EnableHealthCheck = utils.NewBoolValue(true)
		protoIns.HealthCheck = req.HealthCheck
		protoIns.HealthCheck.Type = api.HealthCheck_HEARTBEAT
		// ttl range: (0, 60]
		ttl := protoIns.GetHealthCheck().GetHeartbeat().GetTtl().GetValue()
		if ttl == 0 || ttl > 60 {
			if protoIns.HealthCheck.Heartbeat.Ttl == nil {
				protoIns.HealthCheck.Heartbeat.Ttl = utils.NewUInt32Value(5)
			}
			protoIns.HealthCheck.Heartbeat.Ttl.Value = 5
		}
	}

	instance.Proto = protoIns
	return instance
}
