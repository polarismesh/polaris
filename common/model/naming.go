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
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/wrappers"
	apifault "github.com/polarismesh/specification/source/go/api/v1/fault_tolerance"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
)

// Namespace 命名空间结构体
type Namespace struct {
	Name       string
	Comment    string
	Token      string
	Owner      string
	Valid      bool
	CreateTime time.Time
	ModifyTime time.Time
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
	return srcSvcEqual && dstSvcEqual
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
	return dstSvcEqual
}
