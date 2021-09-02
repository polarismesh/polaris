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

package model

import (
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/golang/protobuf/ptypes/wrappers"
	"time"
)

// 组合了api的Instance对象
type Instance struct {
	Proto             *api.Instance
	ServiceID         string
	ServicePlatformID string
	Valid             bool
	ModifyTime        time.Time
}

// get id
func (i *Instance) ID() string {
	if i.Proto == nil {
		return ""
	}
	return i.Proto.GetId().GetValue()
}

// get service
func (i *Instance) Service() string {
	if i.Proto == nil {
		return ""
	}
	return i.Proto.GetService().GetValue()
}

// get namespace
func (i *Instance) Namespace() string {
	if i.Proto == nil {
		return ""
	}
	return i.Proto.GetNamespace().GetValue()
}

// get vpcid
func (i *Instance) VpcID() string {
	if i.Proto == nil {
		return ""
	}
	return i.Proto.GetVpcId().GetValue()
}

// get host
func (i *Instance) Host() string {
	if i.Proto == nil {
		return ""
	}
	return i.Proto.GetHost().GetValue()
}

// get port
func (i *Instance) Port() uint32 {
	if i.Proto == nil {
		return 0
	}
	return i.Proto.GetPort().GetValue()
}

// get protocol
func (i *Instance) Protocol() string {
	if i.Proto == nil {
		return ""
	}
	return i.Proto.GetProtocol().GetValue()
}

// get version
func (i *Instance) Version() string {
	if i.Proto == nil {
		return ""
	}
	return i.Proto.GetVersion().GetValue()
}

// get priority
func (i *Instance) Priority() uint32 {
	if i.Proto == nil {
		return 0
	}
	return i.Proto.GetPriority().GetValue()
}

// get weight
func (i *Instance) Weight() uint32 {
	if i.Proto == nil {
		return 0
	}
	return i.Proto.GetWeight().GetValue()
}

// get enable health check
func (i *Instance) EnableHealthCheck() bool {
	if i.Proto == nil {
		return false
	}
	return i.Proto.GetEnableHealthCheck().GetValue()
}

// get health check
func (i *Instance) HealthCheck() *api.HealthCheck {
	if i.Proto == nil {
		return nil
	}
	return i.Proto.GetHealthCheck()
}

// get healthy
func (i *Instance) Healthy() bool {
	if i.Proto == nil {
		return false
	}
	return i.Proto.GetHealthy().GetValue()
}

// get isolate
func (i *Instance) Isolate() bool {
	if i.Proto == nil {
		return false
	}
	return i.Proto.GetIsolate().GetValue()
}

// get location
func (i *Instance) Location() *api.Location {
	if i.Proto == nil {
		return nil
	}
	return i.Proto.GetLocation()
}

// get metadata
func (i *Instance) Metadata() map[string]string {
	if i.Proto == nil {
		return nil
	}
	return i.Proto.GetMetadata()
}

// get logic set
func (i *Instance) LogicSet() string {
	if i.Proto == nil {
		return ""
	}
	return i.Proto.GetLogicSet().GetValue()
}

// get ctime
func (i *Instance) Ctime() string {
	if i.Proto == nil {
		return ""
	}
	return i.Proto.GetCtime().GetValue()
}

// get mtime
func (i *Instance) Mtime() string {
	if i.Proto == nil {
		return ""
	}
	return i.Proto.GetMtime().GetValue()
}

// get revision
func (i *Instance) Revision() string {
	if i.Proto == nil {
		return ""
	}
	return i.Proto.GetRevision().GetValue()
}

// get service token
func (i *Instance) ServiceToken() string {
	if i.Proto == nil {
		return ""
	}
	return i.Proto.GetServiceToken().GetValue()
}

// malloc proto if proto is null
func (i *Instance) MallocProto() {
	if i.Proto == nil {
		i.Proto = &api.Instance{}
	}
}

// 对应store层（database）的对象
type InstanceStore struct {
	ID                string
	ServiceID         string
	Host              string
	VpcID             string
	Port              uint32
	Protocol          string
	Version           string
	HealthStatus      int
	Isolate           int
	Weight            uint32
	EnableHealthCheck int
	CheckType         int32
	TTL               uint32
	Priority          uint32
	Revision          string
	LogicSet          string
	Region            string
	Zone              string
	Campus            string
	Meta              map[string]string
	Flag              int
	CreateTime        int64
	ModifyTime        int64
}

// 包含服务名的store信息
type ExpandInstanceStore struct {
	ServiceName       string
	Namespace         string
	ServiceToken      string
	ServicePlatformID string
	ServiceInstance   *InstanceStore
}

// store的数据转换为组合了api的数据结构
func Store2Instance(is *InstanceStore) *Instance {
	ins := &Instance{
		Proto: &api.Instance{
			Id:                &wrappers.StringValue{Value: is.ID},
			VpcId:             &wrappers.StringValue{Value: is.VpcID},
			Host:              &wrappers.StringValue{Value: is.Host},
			Port:              &wrappers.UInt32Value{Value: is.Port},
			Protocol:          &wrappers.StringValue{Value: is.Protocol},
			Version:           &wrappers.StringValue{Value: is.Version},
			Priority:          &wrappers.UInt32Value{Value: is.Priority},
			Weight:            &wrappers.UInt32Value{Value: is.Weight},
			EnableHealthCheck: &wrappers.BoolValue{Value: int2bool(is.EnableHealthCheck)},
			Healthy:           &wrappers.BoolValue{Value: int2bool(is.HealthStatus)},
			Isolate:           &wrappers.BoolValue{Value: int2bool(is.Isolate)},
			Metadata:          is.Meta,
			LogicSet:          &wrappers.StringValue{Value: is.LogicSet},
			Ctime:             &wrappers.StringValue{Value: int64Time2String(is.CreateTime)},
			Mtime:             &wrappers.StringValue{Value: int64Time2String(is.ModifyTime)},
			Revision:          &wrappers.StringValue{Value: is.Revision},
		},
		ServiceID:  is.ServiceID,
		Valid:      flag2valid(is.Flag),
		ModifyTime: time.Unix(is.ModifyTime, 0),
	}
	// 如果不存在checkType，即checkType==-1。HealthCheck置为nil
	if is.CheckType != -1 {
		ins.Proto.HealthCheck = &api.HealthCheck{
			Type: api.HealthCheck_HealthCheckType(is.CheckType),
			Heartbeat: &api.HeartbeatHealthCheck{
				Ttl: &wrappers.UInt32Value{Value: is.TTL},
			},
		}
	}
	// 如果location不为空，那么填充一下location
	if is.Region != "" {
		ins.Proto.Location = &api.Location{
			Region: &wrappers.StringValue{Value: is.Region},
			Zone:   &wrappers.StringValue{Value: is.Zone},
			Campus: &wrappers.StringValue{Value: is.Campus},
		}
	}

	return ins
}

// 扩展store转换
func ExpandStore2Instance(es *ExpandInstanceStore) *Instance {
	out := Store2Instance(es.ServiceInstance)
	out.Proto.Service = &wrappers.StringValue{Value: es.ServiceName}
	out.Proto.Namespace = &wrappers.StringValue{Value: es.Namespace}
	if es.ServiceToken != "" {
		out.Proto.ServiceToken = &wrappers.StringValue{Value: es.ServiceToken}
	}
	out.ServicePlatformID = es.ServicePlatformID
	return out
}
