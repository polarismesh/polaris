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
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/wrappers"
	apifault "github.com/polarismesh/specification/source/go/api/v1/fault_tolerance"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"google.golang.org/protobuf/types/known/wrapperspb"

	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
)

var (
	// ErrorNoNamespace 没有找到对应的命名空间
	ErrorNoNamespace error = errors.New("no such namespace")
	// ErrorNoService 没有找到对应的服务
	ErrorNoService error = errors.New("no such service")
)

func ExportToMap(exportTo []*wrappers.StringValue) map[string]struct{} {
	ret := make(map[string]struct{})
	for _, v := range exportTo {
		ret[v.Value] = struct{}{}
	}
	return ret
}

// Namespace 命名空间结构体
type Namespace struct {
	Name       string
	Comment    string
	Token      string
	Owner      string
	Valid      bool
	CreateTime time.Time
	ModifyTime time.Time
	// ServiceExportTo 服务可见性设置
	ServiceExportTo map[string]struct{}
	// Metadata 命名空间元数据
	Metadata map[string]string
}

func (n *Namespace) ListServiceExportTo() []*wrappers.StringValue {
	ret := make([]*wrappers.StringValue, 0, len(n.ServiceExportTo))
	for i := range n.ServiceExportTo {
		ret = append(ret, &wrappers.StringValue{Value: i})
	}
	return ret
}

type ServicePort struct {
	Port     uint32
	Protocol string
}

// Service 服务数据
type Service struct {
	ID           string
	Name         string
	Namespace    string
	Business     string
	Ports        string
	Meta         map[string]string
	Comment      string
	Department   string
	CmdbMod1     string
	CmdbMod2     string
	CmdbMod3     string
	Token        string
	Owner        string
	Revision     string
	Reference    string
	ReferFilter  string
	PlatformID   string
	Valid        bool
	CreateTime   time.Time
	ModifyTime   time.Time
	Mtime        int64
	Ctime        int64
	ServicePorts []*ServicePort
	// ExportTo 服务可见性暴露设置
	ExportTo    map[string]struct{}
	OldExportTo map[string]struct{}
}

func (s *Service) ToSpec() *apiservice.Service {
	return &apiservice.Service{
		Name:       wrapperspb.String(s.Name),
		Namespace:  wrapperspb.String(s.Namespace),
		Metadata:   s.CopyMeta(),
		Ports:      wrapperspb.String(s.Ports),
		Business:   wrapperspb.String(s.Business),
		Department: wrapperspb.String(s.Department),
		CmdbMod1:   wrapperspb.String(s.CmdbMod1),
		CmdbMod2:   wrapperspb.String(s.CmdbMod2),
		CmdbMod3:   wrapperspb.String(s.CmdbMod3),
		Comment:    wrapperspb.String(s.Comment),
		Owners:     wrapperspb.String(s.Owner),
		Token:      wrapperspb.String(s.Token),
		Ctime:      wrapperspb.String(commontime.Time2String(s.CreateTime)),
		Mtime:      wrapperspb.String(commontime.Time2String(s.ModifyTime)),
		Revision:   wrapperspb.String(s.Revision),
		Id:         wrapperspb.String(s.ID),
		ExportTo:   s.ListExportTo(),
	}
}

func (s *Service) CopyMeta() map[string]string {
	ret := make(map[string]string)
	for k, v := range s.Meta {
		ret[k] = v
	}
	return ret
}

func (s *Service) ProtectThreshold() float32 {
	if len(s.Meta) == 0 {
		return 0
	}
	val := s.Meta[MetadataServiceProtectThreshold]
	threshold, _ := strconv.ParseFloat(val, 32)
	return float32(threshold)
}

func (s *Service) ListExportTo() []*wrappers.StringValue {
	ret := make([]*wrappers.StringValue, 0, len(s.ExportTo))
	for i := range s.ExportTo {
		ret = append(ret, &wrappers.StringValue{Value: i})
	}
	return ret
}

// EnhancedService 服务增强数据
type EnhancedService struct {
	*Service
	TotalInstanceCount   uint32
	HealthyInstanceCount uint32
}

// ServiceKey 服务名
type ServiceKey struct {
	Namespace string
	Name      string
}

func (s *ServiceKey) Equal(o *ServiceKey) bool {
	if s == nil {
		return false
	}
	if o == nil {
		return false
	}
	return s.Name == o.Name && s.Namespace == o.Namespace
}

func (s *ServiceKey) IsExact() bool {
	return s.Namespace != "" && s.Namespace != MatchAll && s.Name != "" && s.Name != MatchAll
}

func (s *ServiceKey) Domain() string {
	return s.Name + "." + s.Namespace
}

// IsAlias 便捷函数封装
func (s *Service) IsAlias() bool {
	return s.Reference != ""
}

// ServiceAlias 服务别名结构体
type ServiceAlias struct {
	ID             string
	Alias          string
	AliasNamespace string
	ServiceID      string
	Service        string
	Namespace      string
	Owner          string
	Comment        string
	CreateTime     time.Time
	ModifyTime     time.Time
	ExportTo       map[string]struct{}
}

func (s *ServiceAlias) ListExportTo() []*wrappers.StringValue {
	ret := make([]*wrappers.StringValue, 0, len(s.ExportTo))
	for i := range s.ExportTo {
		ret = append(ret, &wrappers.StringValue{Value: i})
	}
	return ret
}

// WeightType 服务下实例的权重类型
type WeightType uint32

const (
	// WEIGHTDYNAMIC 动态权重
	WEIGHTDYNAMIC WeightType = iota

	// WEIGHTSTATIC 静态权重
	WEIGHTSTATIC
)

// WeightString weight string map
var WeightString = map[WeightType]string{
	WEIGHTDYNAMIC: "dynamic",
	WEIGHTSTATIC:  "static",
}

// WeightEnum weight enum map
var WeightEnum = map[string]WeightType{
	"dynamic": WEIGHTDYNAMIC,
	"static":  WEIGHTSTATIC,
}

// LocationStore 地域信息，对应数据库字段
type LocationStore struct {
	IP         string
	Region     string
	Zone       string
	Campus     string
	RegionID   uint32
	ZoneID     uint32
	CampusID   uint32
	Flag       int
	ModifyTime int64
}

// Location cmdb信息，对应内存结构体
type Location struct {
	Proto    *apimodel.Location
	RegionID uint32
	ZoneID   uint32
	CampusID uint32
	Valid    bool
}

// LocationView cmdb信息，对应内存结构体
type LocationView struct {
	IP       string
	Region   string
	Zone     string
	Campus   string
	RegionID uint32
	ZoneID   uint32
	CampusID uint32
}

// Store2Location 转成内存数据结构
func Store2Location(s *LocationStore) *Location {
	return &Location{
		Proto: &apimodel.Location{
			Region: &wrappers.StringValue{Value: s.Region},
			Zone:   &wrappers.StringValue{Value: s.Zone},
			Campus: &wrappers.StringValue{Value: s.Campus},
		},
		RegionID: s.RegionID,
		ZoneID:   s.ZoneID,
		CampusID: s.CampusID,
		Valid:    flag2valid(s.Flag),
	}
}

// CircuitBreaker 熔断规则
type CircuitBreaker struct {
	ID         string
	Version    string
	Name       string
	Namespace  string
	Business   string
	Department string
	Comment    string
	Inbounds   string
	Outbounds  string
	Token      string
	Owner      string
	Revision   string
	Valid      bool
	CreateTime time.Time
	ModifyTime time.Time
}

// ServiceWithCircuitBreaker 与服务关系绑定的熔断规则
type ServiceWithCircuitBreaker struct {
	ServiceID      string
	CircuitBreaker *CircuitBreaker
	Valid          bool
	CreateTime     time.Time
	ModifyTime     time.Time
}

// ServiceWithCircuitBreakerRules 与服务关系绑定的熔断规则
type ServiceWithCircuitBreakerRules struct {
	mutex               sync.RWMutex
	Service             ServiceKey
	circuitBreakerRules map[string]*CircuitBreakerRule
	Revision            string
}

func NewServiceWithCircuitBreakerRules(svcKey ServiceKey) *ServiceWithCircuitBreakerRules {
	return &ServiceWithCircuitBreakerRules{
		Service:             svcKey,
		circuitBreakerRules: make(map[string]*CircuitBreakerRule),
	}
}

func (s *ServiceWithCircuitBreakerRules) AddCircuitBreakerRule(rule *CircuitBreakerRule) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.circuitBreakerRules[rule.ID] = rule
}

func (s *ServiceWithCircuitBreakerRules) DelCircuitBreakerRule(id string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.circuitBreakerRules, id)
}

func (s *ServiceWithCircuitBreakerRules) IterateCircuitBreakerRules(callback func(*CircuitBreakerRule)) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	for _, rule := range s.circuitBreakerRules {
		callback(rule)
	}
}

func (s *ServiceWithCircuitBreakerRules) CountCircuitBreakerRules() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return len(s.circuitBreakerRules)
}

func (s *ServiceWithCircuitBreakerRules) Clear() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.circuitBreakerRules = make(map[string]*CircuitBreakerRule)
	s.Revision = ""
}

// ServiceWithFaultDetectRules 与服务关系绑定的探测规则
type ServiceWithFaultDetectRules struct {
	mutex            sync.RWMutex
	Service          ServiceKey
	faultDetectRules map[string]*FaultDetectRule
	Revision         string
}

func NewServiceWithFaultDetectRules(svcKey ServiceKey) *ServiceWithFaultDetectRules {
	return &ServiceWithFaultDetectRules{
		Service:          svcKey,
		faultDetectRules: make(map[string]*FaultDetectRule),
	}
}

func (s *ServiceWithFaultDetectRules) AddFaultDetectRule(rule *FaultDetectRule) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.faultDetectRules[rule.ID] = rule
}

func (s *ServiceWithFaultDetectRules) DelFaultDetectRule(id string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.faultDetectRules, id)
}

func (s *ServiceWithFaultDetectRules) IterateFaultDetectRules(callback func(*FaultDetectRule)) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	for _, rule := range s.faultDetectRules {
		callback(rule)
	}
}

func (s *ServiceWithFaultDetectRules) CountFaultDetectRules() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return len(s.faultDetectRules)
}

func (s *ServiceWithFaultDetectRules) Clear() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.faultDetectRules = make(map[string]*FaultDetectRule)
	s.Revision = ""
}

// CircuitBreakerRelation 熔断规则绑定关系
type CircuitBreakerRelation struct {
	ServiceID   string
	RuleID      string
	RuleVersion string
	Valid       bool
	CreateTime  time.Time
	ModifyTime  time.Time
}

// CircuitBreakerDetail 返回给控制台的熔断规则及服务数据
type CircuitBreakerDetail struct {
	Total               uint32
	CircuitBreakerInfos []*CircuitBreakerInfo
}

// CircuitBreakerInfo 熔断规则及绑定服务
type CircuitBreakerInfo struct {
	CircuitBreaker *CircuitBreaker
	Services       []*Service
}

// Int2bool 整数转换为bool值
func Int2bool(entry int) bool {
	return entry != 0
}

// StatusBoolToInt 状态bool转int
func StatusBoolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

// store的flag转换为valid
// flag==1为无效，其他情况为有效
func flag2valid(flag int) bool {
	return flag != 1
}

// InstanceCount Service instance statistics
type InstanceCount struct {
	// IsolateInstanceCount 隔离状态的实例
	IsolateInstanceCount uint32
	// HealthyInstanceCount 健康实例数
	HealthyInstanceCount uint32
	// TotalInstanceCount 总实例数
	TotalInstanceCount uint32
	// VersionCounts 按照实例的版本进行统计计算
	VersionCounts map[string]*InstanceVersionCount
}

// InstanceVersionCount instance version metrics count
type InstanceVersionCount struct {
	// IsolateInstanceCount 隔离状态的实例
	IsolateInstanceCount uint32
	// HealthyInstanceCount 健康实例数
	HealthyInstanceCount uint32
	// TotalInstanceCount 总实例数
	TotalInstanceCount uint32
}

// NamespaceServiceCount Namespace service data
type NamespaceServiceCount struct {
	// ServiceCount 服务数量
	ServiceCount uint32
	// InstanceCnt 实例健康数/实例总数
	InstanceCnt *InstanceCount
}

// CircuitBreakerRule 熔断规则
type CircuitBreakerRule struct {
	Proto        *apifault.CircuitBreakerRule
	ID           string
	Name         string
	Namespace    string
	Description  string
	Level        int
	SrcService   string
	SrcNamespace string
	DstService   string
	DstNamespace string
	DstMethod    string
	Rule         string
	Revision     string
	Enable       bool
	Valid        bool
	CreateTime   time.Time
	ModifyTime   time.Time
	EnableTime   time.Time
}

func (c *CircuitBreakerRule) IsServiceChange(other *CircuitBreakerRule) bool {
	srcSvcEqual := c.SrcService == other.SrcService && c.SrcNamespace == other.SrcNamespace
	dstSvcEqual := c.DstService == other.DstService && c.DstNamespace == other.DstNamespace
	return !srcSvcEqual || !dstSvcEqual
}

// FaultDetectRule 故障探测规则
type FaultDetectRule struct {
	Proto        *apifault.FaultDetectRule
	ID           string
	Name         string
	Namespace    string
	Description  string
	DstService   string
	DstNamespace string
	DstMethod    string
	Rule         string
	Revision     string
	Valid        bool
	CreateTime   time.Time
	ModifyTime   time.Time
}

func (c *FaultDetectRule) IsServiceChange(other *FaultDetectRule) bool {
	dstSvcEqual := c.DstService == other.DstService && c.DstNamespace == other.DstNamespace
	return !dstSvcEqual
}

const (
	MetadataInstanceLastHeartbeatTime   = "internal-lastheartbeat"
	MetadataServiceProtectThreshold     = "internal-service-protectthreshold"
	MetadataRegisterFrom                = "internal-register-from"
	MetadataInternalMetaHealthCheckPath = "internal-healthcheck_path"
	MetadataInternalMetaTraceSampling   = "internal-trace_sampling"
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

// InstanceEventType 探测事件类型
type InstanceEventType string

const (
	// EventDiscoverNone empty discover event
	EventDiscoverNone InstanceEventType = "EventDiscoverNone"
	// EventInstanceOnline instance becoming online
	EventInstanceOnline InstanceEventType = "InstanceOnline"
	// EventInstanceTurnUnHealth Instance becomes unhealthy
	EventInstanceTurnUnHealth InstanceEventType = "InstanceTurnUnHealth"
	// EventInstanceTurnHealth Instance becomes healthy
	EventInstanceTurnHealth InstanceEventType = "InstanceTurnHealth"
	// EventInstanceOpenIsolate Instance is in isolation
	EventInstanceOpenIsolate InstanceEventType = "InstanceOpenIsolate"
	// EventInstanceCloseIsolate Instance shutdown isolation state
	EventInstanceCloseIsolate InstanceEventType = "InstanceCloseIsolate"
	// EventInstanceOffline Instance offline
	EventInstanceOffline InstanceEventType = "InstanceOffline"
	// EventInstanceSendHeartbeat Instance send heartbeat package to server
	EventInstanceSendHeartbeat InstanceEventType = "InstanceSendHeartbeat"
	// EventInstanceUpdate Instance metadata and info update event
	EventInstanceUpdate InstanceEventType = "InstanceUpdate"
	// EventClientOffline .
	EventClientOffline InstanceEventType = "ClientOffline"
)

// CtxEventKeyMetadata 用于将metadata从Context中传入并取出
const CtxEventKeyMetadata = "ctx_event_metadata"

// InstanceEvent 服务实例事件
type InstanceEvent struct {
	Id         string
	SvcId      string
	Namespace  string
	Service    string
	Instance   *apiservice.Instance
	EType      InstanceEventType
	CreateTime time.Time
	MetaData   map[string]string
}

// InjectMetadata 从context中获取metadata并注入到事件对象
func (i *InstanceEvent) InjectMetadata(ctx context.Context) {
	value := ctx.Value(CtxEventKeyMetadata)
	if nil == value {
		return
	}
	i.MetaData = value.(map[string]string)
}

func (i *InstanceEvent) String() string {
	if nil == i {
		return "nil"
	}
	hostPortStr := fmt.Sprintf("%s:%d", i.Instance.GetHost().GetValue(), i.Instance.GetPort().GetValue())
	return fmt.Sprintf("InstanceEvent(id=%s, namespace=%s, svcId=%s, service=%s, type=%v, instance=%s, healthy=%v)",
		i.Id, i.Namespace, i.SvcId, i.Service, i.EType, hostPortStr, i.Instance.GetHealthy().GetValue())
}

type ClientEvent struct {
	EType InstanceEventType
	Id    string
}
