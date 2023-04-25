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
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/wrappers"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
)

// Instance 组合了api的Instance对象
type Instance struct {
	Proto             *apiservice.Instance
	ServiceID         string
	ServicePlatformID string
	// Valid Whether it is deleted by logic
	Valid bool
	// ModifyTime Update time of instance
	ModifyTime time.Time
}

// ID get id
func (i *Instance) ID() string {
	if i.Proto == nil {
		return ""
	}
	return i.Proto.GetId().GetValue()
}

// Service get service
func (i *Instance) Service() string {
	if i.Proto == nil {
		return ""
	}
	return i.Proto.GetService().GetValue()
}

// Namespace get namespace
func (i *Instance) Namespace() string {
	if i.Proto == nil {
		return ""
	}
	return i.Proto.GetNamespace().GetValue()
}

// VpcID get vpcid
func (i *Instance) VpcID() string {
	if i.Proto == nil {
		return ""
	}
	return i.Proto.GetVpcId().GetValue()
}

// Host get host
func (i *Instance) Host() string {
	if i.Proto == nil {
		return ""
	}
	return i.Proto.GetHost().GetValue()
}

// Port get port
func (i *Instance) Port() uint32 {
	if i.Proto == nil {
		return 0
	}
	return i.Proto.GetPort().GetValue()
}

// Protocol get protocol
func (i *Instance) Protocol() string {
	if i.Proto == nil {
		return ""
	}
	return i.Proto.GetProtocol().GetValue()
}

// Version get version
func (i *Instance) Version() string {
	if i.Proto == nil {
		return ""
	}
	return i.Proto.GetVersion().GetValue()
}

// Priority gets priority
func (i *Instance) Priority() uint32 {
	if i.Proto == nil {
		return 0
	}
	return i.Proto.GetPriority().GetValue()
}

// Weight get weight
func (i *Instance) Weight() uint32 {
	if i.Proto == nil {
		return 0
	}
	return i.Proto.GetWeight().GetValue()
}

// EnableHealthCheck get enables health check
func (i *Instance) EnableHealthCheck() bool {
	if i.Proto == nil {
		return false
	}
	return i.Proto.GetEnableHealthCheck().GetValue()
}

// HealthCheck get health check
func (i *Instance) HealthCheck() *apiservice.HealthCheck {
	if i.Proto == nil {
		return nil
	}
	return i.Proto.GetHealthCheck()
}

// Healthy get healthy
func (i *Instance) Healthy() bool {
	if i.Proto == nil {
		return false
	}
	return i.Proto.GetHealthy().GetValue()
}

// Isolate get isolate
func (i *Instance) Isolate() bool {
	if i.Proto == nil {
		return false
	}
	return i.Proto.GetIsolate().GetValue()
}

// Location gets location
func (i *Instance) Location() *apimodel.Location {
	if i.Proto == nil {
		return nil
	}
	return i.Proto.GetLocation()
}

// Metadata get metadata
func (i *Instance) Metadata() map[string]string {
	if i.Proto == nil {
		return nil
	}
	return i.Proto.GetMetadata()
}

// LogicSet get logic set
func (i *Instance) LogicSet() string {
	if i.Proto == nil {
		return ""
	}
	return i.Proto.GetLogicSet().GetValue()
}

// Ctime get ctime
func (i *Instance) Ctime() string {
	if i.Proto == nil {
		return ""
	}
	return i.Proto.GetCtime().GetValue()
}

// Mtime get mtime
func (i *Instance) Mtime() string {
	if i.Proto == nil {
		return ""
	}
	return i.Proto.GetMtime().GetValue()
}

// Revision get revision
func (i *Instance) Revision() string {
	if i.Proto == nil {
		return ""
	}
	return i.Proto.GetRevision().GetValue()
}

// ServiceToken get service token
func (i *Instance) ServiceToken() string {
	if i.Proto == nil {
		return ""
	}
	return i.Proto.GetServiceToken().GetValue()
}

// MallocProto malloc proto if proto is null
func (i *Instance) MallocProto() {
	if i.Proto == nil {
		i.Proto = &apiservice.Instance{}
	}
}

// InstanceStore 对应store层（database）的对象
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

// ExpandInstanceStore 包含服务名的store信息
type ExpandInstanceStore struct {
	ServiceName       string
	Namespace         string
	ServiceToken      string
	ServicePlatformID string
	ServiceInstance   *InstanceStore
}

// Store2Instance store的数据转换为组合了api的数据结构
func Store2Instance(is *InstanceStore) *Instance {
	ins := &Instance{
		Proto: &apiservice.Instance{
			Id:                &wrappers.StringValue{Value: is.ID},
			VpcId:             &wrappers.StringValue{Value: is.VpcID},
			Host:              &wrappers.StringValue{Value: is.Host},
			Port:              &wrappers.UInt32Value{Value: is.Port},
			Protocol:          &wrappers.StringValue{Value: is.Protocol},
			Version:           &wrappers.StringValue{Value: is.Version},
			Priority:          &wrappers.UInt32Value{Value: is.Priority},
			Weight:            &wrappers.UInt32Value{Value: is.Weight},
			EnableHealthCheck: &wrappers.BoolValue{Value: Int2bool(is.EnableHealthCheck)},
			Healthy:           &wrappers.BoolValue{Value: Int2bool(is.HealthStatus)},
			Location: &apimodel.Location{
				Region: &wrappers.StringValue{Value: is.Region},
				Zone:   &wrappers.StringValue{Value: is.Zone},
				Campus: &wrappers.StringValue{Value: is.Campus},
			},
			Isolate:  &wrappers.BoolValue{Value: Int2bool(is.Isolate)},
			Metadata: is.Meta,
			LogicSet: &wrappers.StringValue{Value: is.LogicSet},
			Ctime:    &wrappers.StringValue{Value: commontime.Int64Time2String(is.CreateTime)},
			Mtime:    &wrappers.StringValue{Value: commontime.Int64Time2String(is.ModifyTime)},
			Revision: &wrappers.StringValue{Value: is.Revision},
		},
		ServiceID:  is.ServiceID,
		Valid:      flag2valid(is.Flag),
		ModifyTime: time.Unix(is.ModifyTime, 0),
	}
	// 如果不存在checkType，即checkType==-1。HealthCheck置为nil
	if is.CheckType != -1 {
		ins.Proto.HealthCheck = &apiservice.HealthCheck{
			Type: apiservice.HealthCheck_HealthCheckType(is.CheckType),
			Heartbeat: &apiservice.HeartbeatHealthCheck{
				Ttl: &wrappers.UInt32Value{Value: is.TTL},
			},
		}
	}
	// 如果location不为空，那么填充一下location
	if is.Region != "" {
		ins.Proto.Location = &apimodel.Location{
			Region: &wrappers.StringValue{Value: is.Region},
			Zone:   &wrappers.StringValue{Value: is.Zone},
			Campus: &wrappers.StringValue{Value: is.Campus},
		}
	}

	return ins
}

// ExpandStore2Instance 扩展store转换
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

type NamespaceSet struct {
	container map[string]*Namespace
}

func NewNamespaceSet() *NamespaceSet {
	return &NamespaceSet{
		container: make(map[string]*Namespace),
	}
}

func (set *NamespaceSet) Add(val *Namespace) {
	set.container[val.Name] = val
}

func (set *NamespaceSet) Remove(val *Namespace) {
	delete(set.container, val.Name)
}

func (set *NamespaceSet) ToSlice() []*Namespace {
	ret := make([]*Namespace, 0, len(set.container))

	for _, v := range set.container {
		ret = append(ret, v)
	}

	return ret
}

func (set *NamespaceSet) Range(fn func(val *Namespace) bool) {
	for _, v := range set.container {
		if !fn(v) {
			break
		}
	}
}

type ServiceSet struct {
	container map[string]*Service
}

func NewServiceSet() *ServiceSet {
	return &ServiceSet{
		container: make(map[string]*Service),
	}
}

func (set *ServiceSet) Add(val *Service) {
	set.container[val.ID] = val
}

func (set *ServiceSet) Remove(val *Service) {
	delete(set.container, val.ID)
}

func (set *ServiceSet) ToSlice() []*Service {
	ret := make([]*Service, 0, len(set.container))

	for _, v := range set.container {
		ret = append(ret, v)
	}

	return ret
}

func (set *ServiceSet) Range(fn func(val *Service) bool) {
	for _, v := range set.container {
		if !fn(v) {
			break
		}
	}
}

// CreateInstanceModel 创建存储层服务实例模型
func CreateInstanceModel(serviceID string, req *apiservice.Instance) *Instance {
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

	instance := &Instance{
		ServiceID: serviceID,
	}

	protoIns := &apiservice.Instance{
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
		protoIns.HealthCheck.Type = apiservice.HealthCheck_HEARTBEAT
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
