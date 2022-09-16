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

package v2

import (
	"time"

	"github.com/golang/protobuf/ptypes"
	apiv2 "github.com/polarismesh/polaris-server/common/api/v2"
	commontime "github.com/polarismesh/polaris-server/common/time"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	V2RuleIDKey         = "__routing_v2_id__"
	V1RuleIDKey         = "__routing_v1_id__"
	V1RuleRouteIndexKey = "__routing_v1_route_index__"
	V1RuleRouteTypeKey  = "__routing_v1_route_type__"

	V1RuleInRoute  = "in"
	V1RuleOutRoute = "out"
)

var (
	_ruleRoutingTypeUrl string
	_metaRoutingTypeUrl string
)

func init() {
	ruleAny, _ := ptypes.MarshalAny(&apiv2.RuleRoutingConfig{})
	metaAny, _ := ptypes.MarshalAny(&apiv2.MetadataRoutingConfig{})

	_ruleRoutingTypeUrl = ruleAny.GetTypeUrl()
	_metaRoutingTypeUrl = metaAny.GetTypeUrl()
}

// ExtendRoutingConfig 路由信息的扩展
type ExtendRoutingConfig struct {
	*RoutingConfig
	// MetadataRouting 元数据路由配置
	MetadataRouting *apiv2.MetadataRoutingConfig
	// RuleRouting 规则路由配置
	RuleRouting *apiv2.RuleRoutingConfig
	// ExtendInfo 额外信息数据
	ExtendInfo map[string]string
}

func (r *ExtendRoutingConfig) ToApi() (*apiv2.Routing, error) {

	var (
		any *anypb.Any
		err error
	)

	if r.GetRoutingPolicy() == apiv2.RoutingPolicy_MetadataPolicy {
		any, err = ptypes.MarshalAny(r.MetadataRouting)
		if err != nil {
			return nil, err
		}
	} else {
		any, err = ptypes.MarshalAny(r.RuleRouting)
		if err != nil {
			return nil, err
		}
	}

	return &apiv2.Routing{
		Id:            r.ID,
		Name:          r.Name,
		Namespace:     r.Namespace,
		Enable:        r.Enable,
		RoutingPolicy: r.GetRoutingPolicy(),
		RoutingConfig: any,
		Revision:      r.Revision,
		Ctime:         commontime.Time2String(r.CreateTime),
		Mtime:         commontime.Time2String(r.ModifyTime),
		Etime:         commontime.Time2String(r.EnableTime),
		Priority:      r.Priority,
		ExtendInfo:    r.ExtendInfo,
	}, nil
}

// RoutingConfig 路由规则
type RoutingConfig struct {
	// ID 规则唯一标识
	ID string `json:"id"`
	// namespace 所属的命名空间
	Namespace string `json:"namespace"`
	// name 规则名称
	Name string `json:"name"`
	// policy 规则类型
	Policy string `json:"policy"`
	// config 具体的路由规则内容
	Config string `json:"config"`
	// enable 路由规则是否启用
	Enable bool `json:"enable"`
	// priority 规则优先级
	Priority uint32 `json:"priority"`
	// revision 路由规则的版本信息
	Revision string `json:"revision"`
	// valid 路由规则是否有效，没有被逻辑删除
	Valid bool `json:"flag"`
	// createtime 规则创建时间
	CreateTime time.Time `json:"ctime"`
	// modifytime 规则修改时间
	ModifyTime time.Time `json:"mtime"`
	// enabletime 规则最近一次启用时间
	EnableTime time.Time `json:"etime"`
}

func (r *RoutingConfig) GetRoutingPolicy() apiv2.RoutingPolicy {
	v, ok := apiv2.RoutingPolicy_value[r.Policy]

	if !ok {
		return apiv2.RoutingPolicy(-1)
	}

	return apiv2.RoutingPolicy(v)
}

func (r *RoutingConfig) ToExpendRoutingConfig() (*ExtendRoutingConfig, error) {
	ret := &ExtendRoutingConfig{
		RoutingConfig: r,
	}

	policy := r.GetRoutingPolicy()

	if policy == apiv2.RoutingPolicy_RulePolicy {
		rule := &apiv2.RuleRoutingConfig{}
		if err := ptypes.UnmarshalAny(&anypb.Any{
			TypeUrl: _ruleRoutingTypeUrl,
			Value:   []byte(r.Config),
		}, rule); err != nil {
			return nil, err
		}
		ret.RuleRouting = rule
	}

	if policy == apiv2.RoutingPolicy_MetadataPolicy {
		rule := &apiv2.MetadataRoutingConfig{}
		if err := ptypes.UnmarshalAny(&anypb.Any{
			TypeUrl: _metaRoutingTypeUrl,
			Value:   []byte(r.Config),
		}, rule); err != nil {
			return nil, err
		}

		ret.MetadataRouting = rule
	}

	return ret, nil
}

func (r *RoutingConfig) ParseFromAPI(routing *apiv2.Routing) error {
	r.ID = routing.Id
	r.Revision = routing.Revision
	r.Name = routing.Name
	r.Namespace = routing.Name
	r.Enable = routing.Enable
	r.Policy = routing.GetRoutingPolicy().String()
	r.Priority = routing.Priority
	r.Config = string(routing.GetRoutingConfig().GetValue())
	return nil
}

// Arguments2Labels 将参数列表适配成旧的标签模型
func (r *RoutingConfig) Arguments2Labels() bool {
	if r.GetRoutingPolicy() == apiv2.RoutingPolicy_RulePolicy {

		return true
	}
	return false
}

// AdaptArgumentsAndLabels 对存量标签进行兼容
func (r *RoutingConfig) AdaptArgumentsAndLabels() error {
	// 新的限流规则，需要适配老的SDK使用场景
	if !r.Arguments2Labels() {
		// 存量限流规则，需要适配成新的规则
	}

	return nil
}

// Labels2Arguments 适配老的标签到新的参数列表
func (r *RoutingConfig) Labels2Arguments() (map[string]*apiv2.MatchString, error) {

	return nil, nil
}
