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

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	protoV2 "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
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
	ruleAny, _ := ptypes.MarshalAny(&apitraffic.RuleRoutingConfig{})
	metaAny, _ := ptypes.MarshalAny(&apitraffic.MetadataRoutingConfig{})

	RuleRoutingTypeUrl = ruleAny.GetTypeUrl()
	MetaRoutingTypeUrl = metaAny.GetTypeUrl()
}

// ExtendRouterConfig 路由信息的扩展
type ExtendRouterConfig struct {
	*RouterConfig
	// MetadataRouting 元数据路由配置
	MetadataRouting *apitraffic.MetadataRoutingConfig
	// RuleRouting 规则路由配置
	RuleRouting *apitraffic.RuleRoutingConfig
	// ExtendInfo 额外信息数据
	ExtendInfo map[string]string
}

// ToApi Turn to API object
func (r *ExtendRouterConfig) ToApi() (*apitraffic.RouteRule, error) {
	var (
		anyValue *anypb.Any
		err      error
	)

	if r.GetRoutingPolicy() == apitraffic.RoutingPolicy_MetadataPolicy {
		anyValue, err = ptypes.MarshalAny(r.MetadataRouting)
		if err != nil {
			return nil, err
		}
	} else {
		anyValue, err = ptypes.MarshalAny(r.RuleRouting)
		if err != nil {
			return nil, err
		}
	}

	return &apitraffic.RouteRule{
		Id:            r.ID,
		Name:          r.Name,
		Namespace:     r.Namespace,
		Enable:        r.Enable,
		RoutingPolicy: r.GetRoutingPolicy(),
		RoutingConfig: anyValue,
		Revision:      r.Revision,
		Ctime:         commontime.Time2String(r.CreateTime),
		Mtime:         commontime.Time2String(r.ModifyTime),
		Etime:         commontime.Time2String(r.EnableTime),
		Priority:      r.Priority,
		Description:   r.Description,
	}, nil
}

// RouterConfig Routing rules
type RouterConfig struct {
	// ID The unique id of the rules
	ID string `json:"id"`
	// namespace router config owner namespace
	Namespace string `json:"namespace"`
	// name router config name
	Name string `json:"name"`
	// policy Rules
	Policy string `json:"policy"`
	// config Specific routing rules content
	Config string `json:"config"`
	// enable Whether the routing rules are enabled
	Enable bool `json:"enable"`
	// priority Rules priority
	Priority uint32 `json:"priority"`
	// revision Edition information of routing rules
	Revision string `json:"revision"`
	// Description Simple description of rules
	Description string `json:"description"`
	// valid Whether the routing rules are valid and have not been deleted by logic
	Valid bool `json:"flag"`
	// createtime Rules creation time
	CreateTime time.Time `json:"ctime"`
	// modifytime Rules modify time
	ModifyTime time.Time `json:"mtime"`
	// enabletime The last time the rules enabled
	EnableTime time.Time `json:"etime"`
}

// GetRoutingPolicy Query routing rules type
func (r *RouterConfig) GetRoutingPolicy() apitraffic.RoutingPolicy {
	v, ok := apitraffic.RoutingPolicy_value[r.Policy]

	if !ok {
		return apitraffic.RoutingPolicy(-1)
	}

	return apitraffic.RoutingPolicy(v)
}

// ToExpendRoutingConfig Converted to an expansion object, serialize the corresponding PB Struct in advance
func (r *RouterConfig) ToExpendRoutingConfig() (*ExtendRouterConfig, error) {
	ret := &ExtendRouterConfig{
		RouterConfig: r,
	}

	configText := r.Config
	if len(configText) == 0 {
		return ret, nil
	}
	policy := r.GetRoutingPolicy()
	var err error
	if strings.HasPrefix(configText, "{") {
		// process with json
		switch policy {
		case apitraffic.RoutingPolicy_RulePolicy:
			rule := &apitraffic.RuleRoutingConfig{}
			if err = utils.UnmarshalFromJsonString(rule, configText); nil != err {
				return nil, err
			}
			parseSubRouteRule(rule)
			ret.RuleRouting = rule
			break
		case apitraffic.RoutingPolicy_MetadataPolicy:
			rule := &apitraffic.MetadataRoutingConfig{}
			if err = utils.UnmarshalFromJsonString(rule, configText); nil != err {
				return nil, err
			}
			ret.MetadataRouting = rule
			break
		}
		return ret, nil
	}

	if err := r.parseBinaryAnyMessage(policy, ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func (r *RouterConfig) parseBinaryAnyMessage(
	policy apitraffic.RoutingPolicy, ret *ExtendRouterConfig) error {
	// parse v1 binary
	switch policy {
	case apitraffic.RoutingPolicy_RulePolicy:
		rule := &apitraffic.RuleRoutingConfig{}
		anyMsg := &anypb.Any{
			TypeUrl: RuleRoutingTypeUrl,
			Value:   []byte(r.Config),
		}
		if err := unmarshalToAny(anyMsg, rule); nil != err {
			return err
		}
		parseSubRouteRule(rule)
		ret.RuleRouting = rule
	case apitraffic.RoutingPolicy_MetadataPolicy:
		rule := &apitraffic.MetadataRoutingConfig{}
		anyMsg := &anypb.Any{
			TypeUrl: MetaRoutingTypeUrl,
			Value:   []byte(r.Config),
		}
		if err := unmarshalToAny(anyMsg, rule); nil != err {
			return err
		}
		ret.MetadataRouting = rule
	}
	return nil
}

// ParseRouteRuleFromAPI Convert an internal object from the API object
func (r *RouterConfig) ParseRouteRuleFromAPI(routing *apitraffic.RouteRule) error {
	ruleMessage, err := ParseRouteRuleAnyToMessage(routing.RoutingPolicy, routing.RoutingConfig)
	if nil != err {
		return err
	}

	if r.Config, err = utils.MarshalToJsonString(ruleMessage); nil != err {
		return err
	}
	r.ID = routing.Id
	r.Revision = routing.Revision
	r.Name = routing.Name
	r.Namespace = routing.Namespace
	r.Enable = routing.Enable
	r.Policy = routing.GetRoutingPolicy().String()
	r.Priority = routing.Priority
	r.Description = routing.Description

	// Priority range range [0, 10]
	if r.Priority > 10 {
		r.Priority = 10
	}

	return nil
}

func unmarshalToAny(anyMessage *anypb.Any, message proto.Message) error {
	return anypb.UnmarshalTo(anyMessage, proto.MessageV2(message),
		protoV2.UnmarshalOptions{AllowPartial: true, DiscardUnknown: true})
}

// ParseRouteRuleAnyToMessage convert the any routing proto to message object
func ParseRouteRuleAnyToMessage(policy apitraffic.RoutingPolicy, anyMessage *anypb.Any) (proto.Message, error) {
	var rule proto.Message
	switch policy {
	case apitraffic.RoutingPolicy_RulePolicy:
		rule = &apitraffic.RuleRoutingConfig{}
		if err := unmarshalToAny(anyMessage, rule); err != nil {
			return nil, err
		}
		ruleRouting := rule.(*apitraffic.RuleRoutingConfig)
		parseSubRouteRule(ruleRouting)
		break
	case apitraffic.RoutingPolicy_MetadataPolicy:
		rule = &apitraffic.MetadataRoutingConfig{}
		if err := unmarshalToAny(anyMessage, rule); err != nil {
			return nil, err
		}
		break
	default:
		break
	}
	return rule, nil
}

func parseSubRouteRule(ruleRouting *apitraffic.RuleRoutingConfig) {
	if len(ruleRouting.Rules) == 0 {
		subRule := &apitraffic.SubRuleRouting{
			Name:         "",
			Sources:      ruleRouting.GetSources(),
			Destinations: ruleRouting.GetDestinations(),
		}
		ruleRouting.Rules = []*apitraffic.SubRuleRouting{
			subRule,
		}
	} else {
		// Abandon the value of the old field
		ruleRouting.Destinations = nil
		ruleRouting.Sources = nil
	}
}
