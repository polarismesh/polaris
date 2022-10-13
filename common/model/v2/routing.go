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
	"google.golang.org/protobuf/types/known/anypb"

	apiv2 "github.com/polarismesh/polaris/common/api/v2"
	commontime "github.com/polarismesh/polaris/common/time"
)

const (
	// V2RuleIDKey v2 版本的规则路由 ID
	V2RuleIDKey = "__routing_v2_id__"
	// V1RuleIDKey v1 版本的路由规则 ID
	V1RuleIDKey = "__routing_v1_id__"
	// V1RuleRouteIndexKey v1 版本 route 规则在自己 route 链中的 index 信息
	V1RuleRouteIndexKey = "__routing_v1_route_index__"
	// V1RuleRouteTypeKey 标识当前 v2 路由规则在 v1 的 inBound 还是 outBound
	V1RuleRouteTypeKey = "__routing_v1_route_type__"
	// V1RuleInRoute inBound 类型
	V1RuleInRoute = "in"
	// V1RuleOutRoute outBound 类型
	V1RuleOutRoute = "out"
)

var (
	// RuleRoutingTypeUrl 记录 anypb.Any 中关于 RuleRoutingConfig 的 url 信息
	RuleRoutingTypeUrl string
	// MetaRoutingTypeUrl 记录 anypb.Any 中关于 MetadataRoutingConfig 的 url 信息
	MetaRoutingTypeUrl string
)

func init() {
	ruleAny, _ := ptypes.MarshalAny(&apiv2.RuleRoutingConfig{})
	metaAny, _ := ptypes.MarshalAny(&apiv2.MetadataRoutingConfig{})

	RuleRoutingTypeUrl = ruleAny.GetTypeUrl()
	MetaRoutingTypeUrl = metaAny.GetTypeUrl()
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

// ToApi 转为 api 对象
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
		Description:   r.Description,
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
	// Description 规则简单描述
	Description string `json:"description"`
	// valid 路由规则是否有效，没有被逻辑删除
	Valid bool `json:"flag"`
	// createtime 规则创建时间
	CreateTime time.Time `json:"ctime"`
	// modifytime 规则修改时间
	ModifyTime time.Time `json:"mtime"`
	// enabletime 规则最近一次启用时间
	EnableTime time.Time `json:"etime"`
}

// GetRoutingPolicy 查询路由规则类型
func (r *RoutingConfig) GetRoutingPolicy() apiv2.RoutingPolicy {
	v, ok := apiv2.RoutingPolicy_value[r.Policy]

	if !ok {
		return apiv2.RoutingPolicy(-1)
	}

	return apiv2.RoutingPolicy(v)
}

// ToExpendRoutingConfig 转为扩展对象，提前序列化出相应的 pb struct
func (r *RoutingConfig) ToExpendRoutingConfig() (*ExtendRoutingConfig, error) {
	ret := &ExtendRoutingConfig{
		RoutingConfig: r,
	}

	policy := r.GetRoutingPolicy()

	if policy == apiv2.RoutingPolicy_RulePolicy {
		rule := &apiv2.RuleRoutingConfig{}
		if err := ptypes.UnmarshalAny(&anypb.Any{
			TypeUrl: RuleRoutingTypeUrl,
			Value:   []byte(r.Config),
		}, rule); err != nil {
			return nil, err
		}
		ret.RuleRouting = rule
	}

	if policy == apiv2.RoutingPolicy_MetadataPolicy {
		rule := &apiv2.MetadataRoutingConfig{}
		if err := ptypes.UnmarshalAny(&anypb.Any{
			TypeUrl: MetaRoutingTypeUrl,
			Value:   []byte(r.Config),
		}, rule); err != nil {
			return nil, err
		}

		ret.MetadataRouting = rule
	}

	return ret, nil
}

// ParseFromAPI 从 API 对象中转换出内部对象
func (r *RoutingConfig) ParseFromAPI(routing *apiv2.Routing) error {
	r.ID = routing.Id
	r.Revision = routing.Revision
	r.Name = routing.Name
	r.Namespace = routing.Namespace
	r.Enable = routing.Enable
	r.Policy = routing.GetRoutingPolicy().String()
	r.Priority = routing.Priority
	r.Config = string(routing.GetRoutingConfig().GetValue())
	r.Description = routing.Description

	// 优先级区间范围 [0, 10]
	if r.Priority > 10 {
		r.Priority = 10
	}
	if r.Priority < 0 {
		r.Priority = 0
	}

	return nil
}
