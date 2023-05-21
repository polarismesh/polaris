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
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
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

	rule := &apitraffic.RouteRule{
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
	}
	if r.EnableTime.Year() > 2000 {
		rule.Etime = commontime.Time2String(r.EnableTime)
	} else {
		rule.Etime = ""
	}
	return rule, nil
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

const (
	_labelKeyPath     = "$path"
	_labelKeyMethod   = "$method"
	_labelKeyHeader   = "$header"
	_labelKeyQuery    = "$query"
	_labelKeyCallerIP = "$caller_ip"
	_labelKeyCookie   = "$cookie"

	MatchAll = "*"
)

// RoutingConfigV1ToAPI Convert the internal data structure to API parameter to pass out
func RoutingConfigV1ToAPI(req *RoutingConfig, service string, namespace string) (*apitraffic.Routing, error) {
	if req == nil {
		return nil, nil
	}

	out := &apitraffic.Routing{
		Service:   utils.NewStringValue(service),
		Namespace: utils.NewStringValue(namespace),
		Revision:  utils.NewStringValue(req.Revision),
		Ctime:     utils.NewStringValue(commontime.Time2String(req.CreateTime)),
		Mtime:     utils.NewStringValue(commontime.Time2String(req.ModifyTime)),
	}

	if req.InBounds != "" {
		var inBounds []*apitraffic.Route
		if err := json.Unmarshal([]byte(req.InBounds), &inBounds); err != nil {
			return nil, err
		}
		out.Inbounds = inBounds
	}
	if req.OutBounds != "" {
		var outBounds []*apitraffic.Route
		if err := json.Unmarshal([]byte(req.OutBounds), &outBounds); err != nil {
			return nil, err
		}
		out.Outbounds = outBounds
	}

	return out, nil
}

// CompositeRoutingV1AndV2 The routing rules of the V1 version and the rules of the V2 version
func CompositeRoutingV1AndV2(v1rule *apitraffic.Routing, level1, level2,
	level3 []*ExtendRouterConfig) (*apitraffic.Routing, []string) {
	sort.Slice(level1, func(i, j int) bool {
		return CompareRoutingV2(level1[i], level1[j])
	})

	sort.Slice(level2, func(i, j int) bool {
		return CompareRoutingV2(level2[i], level2[j])
	})

	sort.Slice(level3, func(i, j int) bool {
		return CompareRoutingV2(level3[i], level3[j])
	})

	level1inRoutes, level1outRoutes, level1Revisions :=
		BuildV1RoutesFromV2(v1rule.Service.Value, v1rule.Namespace.Value, level1)
	level2inRoutes, level2outRoutes, level2Revisions :=
		BuildV1RoutesFromV2(v1rule.Service.Value, v1rule.Namespace.Value, level2)
	level3inRoutes, level3outRoutes, level3Revisions :=
		BuildV1RoutesFromV2(v1rule.Service.Value, v1rule.Namespace.Value, level3)

	inBounds := v1rule.GetInbounds()
	outBounds := v1rule.GetOutbounds()

	// Processing inbounds rules，level1 cache -> v1rules -> level2 cache -> level3 cache
	if len(level1inRoutes) > 0 {
		v1rule.Inbounds = append(level1inRoutes, inBounds...)
	}
	if len(level2inRoutes) > 0 {
		v1rule.Inbounds = append(v1rule.Inbounds, level2inRoutes...)
	}
	if len(level3inRoutes) > 0 {
		v1rule.Inbounds = append(v1rule.Inbounds, level3inRoutes...)
	}

	// Processing OutBounds rules，level1 cache -> v1rules -> level2 cache -> level3 cache
	if len(level1outRoutes) > 0 {
		v1rule.Outbounds = append(level1outRoutes, outBounds...)
	}
	if len(level2outRoutes) > 0 {
		v1rule.Outbounds = append(v1rule.Outbounds, level2outRoutes...)
	}
	if len(level3outRoutes) > 0 {
		v1rule.Outbounds = append(v1rule.Outbounds, level3outRoutes...)
	}

	revisions := make([]string, 0, 1+len(level1Revisions)+len(level2Revisions)+len(level3Revisions))
	revisions = append(revisions, v1rule.GetRevision().GetValue())
	if len(level1Revisions) > 0 {
		revisions = append(revisions, level1Revisions...)
	}
	if len(level2Revisions) > 0 {
		revisions = append(revisions, level2Revisions...)
	}
	if len(level3Revisions) > 0 {
		revisions = append(revisions, level3Revisions...)
	}

	return v1rule, revisions
}

// BuildV1RoutesFromV2 According to the routing rules of the V2 version, it is adapted to the V1 version
// of the routing rules.
// return inBound outBound revisions
func BuildV1RoutesFromV2(service, namespace string,
	entries []*ExtendRouterConfig) ([]*apitraffic.Route, []*apitraffic.Route, []string) {
	if len(entries) == 0 {
		return []*apitraffic.Route{}, []*apitraffic.Route{}, []string{}
	}

	revisions := make([]string, 0, len(entries))
	outRoutes := make([]*apitraffic.Route, 0, 8)
	inRoutes := make([]*apitraffic.Route, 0, 8)
	for i := range entries {
		if !entries[i].Enable {
			continue
		}
		outRoutes = append(outRoutes, BuildOutBoundsFromV2(service, namespace, entries[i])...)
		inRoutes = append(inRoutes, BuildInBoundsFromV2(service, namespace, entries[i])...)
		revisions = append(revisions, entries[i].Revision)
	}

	return inRoutes, outRoutes, revisions
}

// BuildOutBoundsFromV2 According to the routing rules of the V2 version, it is adapted to the
// outbounds in the routing rule of V1 version
func BuildOutBoundsFromV2(service, namespace string, item *ExtendRouterConfig) []*apitraffic.Route {
	if item.GetRoutingPolicy() != apitraffic.RoutingPolicy_RulePolicy {
		return []*apitraffic.Route{}
	}

	var find bool

	matchService := func(source *apitraffic.SourceService) bool {
		if source.Service == service && source.Namespace == namespace {
			return true
		}
		if source.Namespace == namespace && source.Service == MatchAll {
			return true
		}
		if source.Namespace == MatchAll && source.Service == MatchAll {
			return true
		}
		return false
	}

	routes := make([]*apitraffic.Route, 0, 8)
	for i := range item.RuleRouting.Rules {
		subRule := item.RuleRouting.Rules[i]
		sources := item.RuleRouting.Rules[i].Sources
		v1sources := make([]*apitraffic.Source, 0, len(sources))
		for i := range sources {
			if matchService(sources[i]) {
				find = true
				entry := &apitraffic.Source{
					Service:   utils.NewStringValue(service),
					Namespace: utils.NewStringValue(namespace),
				}
				entry.Metadata = RoutingArguments2Labels(sources[i].GetArguments())
				v1sources = append(v1sources, entry)
			}
		}

		if !find {
			break
		}

		destinations := item.RuleRouting.Rules[i].Destinations
		v1destinations := make([]*apitraffic.Destination, 0, len(destinations))
		for i := range destinations {
			name := fmt.Sprintf("%s.%s.%s", item.Name, subRule.Name, destinations[i].Name)
			entry := &apitraffic.Destination{
				Name:      utils.NewStringValue(name),
				Service:   utils.NewStringValue(destinations[i].Service),
				Namespace: utils.NewStringValue(destinations[i].Namespace),
				Priority:  utils.NewUInt32Value(destinations[i].GetPriority()),
				Weight:    utils.NewUInt32Value(destinations[i].GetWeight()),
				Transfer:  utils.NewStringValue(destinations[i].GetTransfer()),
				Isolate:   utils.NewBoolValue(destinations[i].GetIsolate()),
			}

			v1labels := make(map[string]*apimodel.MatchString)
			v2labels := destinations[i].GetLabels()
			for index := range v2labels {
				v1labels[index] = &apimodel.MatchString{
					Type:      v2labels[index].GetType(),
					Value:     v2labels[index].GetValue(),
					ValueType: v2labels[index].GetValueType(),
				}
			}

			entry.Metadata = v1labels
			v1destinations = append(v1destinations, entry)
		}

		routes = append(routes, &apitraffic.Route{
			Sources:      v1sources,
			Destinations: v1destinations,
			ExtendInfo: map[string]string{
				V2RuleIDKey: item.ID,
			},
		})
	}

	return routes
}

// BuildInBoundsFromV2 Convert the routing rules of V2 to the inbounds in the routing rule of V1
func BuildInBoundsFromV2(service, namespace string, item *ExtendRouterConfig) []*apitraffic.Route {
	if item.GetRoutingPolicy() != apitraffic.RoutingPolicy_RulePolicy {
		return []*apitraffic.Route{}
	}

	var find bool

	matchService := func(destination *apitraffic.DestinationGroup) bool {
		if destination.Service == service && destination.Namespace == namespace {
			return true
		}
		if destination.Namespace == namespace && destination.Service == MatchAll {
			return true
		}
		if destination.Namespace == MatchAll && destination.Service == MatchAll {
			return true
		}
		return false
	}

	routes := make([]*apitraffic.Route, 0, 8)

	for i := range item.RuleRouting.Rules {
		subRule := item.RuleRouting.Rules[i]
		destinations := item.RuleRouting.Rules[i].Destinations
		v1destinations := make([]*apitraffic.Destination, 0, len(destinations))
		for i := range destinations {
			if matchService(destinations[i]) {
				find = true
				name := fmt.Sprintf("%s.%s.%s", item.Name, subRule.Name, destinations[i].Name)
				entry := &apitraffic.Destination{
					Name:      utils.NewStringValue(name),
					Service:   utils.NewStringValue(service),
					Namespace: utils.NewStringValue(namespace),
					Priority:  utils.NewUInt32Value(destinations[i].GetPriority()),
					Weight:    utils.NewUInt32Value(destinations[i].GetWeight()),
					Transfer:  utils.NewStringValue(destinations[i].GetTransfer()),
					Isolate:   utils.NewBoolValue(destinations[i].GetIsolate()),
				}

				v1labels := make(map[string]*apimodel.MatchString)
				v2labels := destinations[i].GetLabels()
				for index := range v2labels {
					v1labels[index] = &apimodel.MatchString{
						Type:      v2labels[index].GetType(),
						Value:     v2labels[index].GetValue(),
						ValueType: v2labels[index].GetValueType(),
					}
				}

				entry.Metadata = v1labels
				v1destinations = append(v1destinations, entry)
			}
		}

		if !find {
			break
		}

		sources := item.RuleRouting.Rules[i].Sources
		v1sources := make([]*apitraffic.Source, 0, len(sources))
		for i := range sources {
			entry := &apitraffic.Source{
				Service:   utils.NewStringValue(sources[i].Service),
				Namespace: utils.NewStringValue(sources[i].Namespace),
			}

			entry.Metadata = RoutingArguments2Labels(sources[i].GetArguments())
			v1sources = append(v1sources, entry)
		}

		routes = append(routes, &apitraffic.Route{
			Sources:      v1sources,
			Destinations: v1destinations,
			ExtendInfo: map[string]string{
				V2RuleIDKey: item.ID,
			},
		})
	}

	return routes
}

// RoutingLabels2Arguments Adapting the old label model into a list of parameters
func RoutingLabels2Arguments(labels map[string]*apimodel.MatchString) []*apitraffic.SourceMatch {
	if len(labels) == 0 {
		return []*apitraffic.SourceMatch{}
	}

	arguments := make([]*apitraffic.SourceMatch, 0, 4)
	for index := range labels {
		arguments = append(arguments, &apitraffic.SourceMatch{
			Type: apitraffic.SourceMatch_CUSTOM,
			Key:  index,
			Value: &apimodel.MatchString{
				Type:      labels[index].GetType(),
				Value:     labels[index].GetValue(),
				ValueType: labels[index].GetValueType(),
			},
		})
	}

	return arguments
}

// RoutingArguments2Labels Adapt the parameter list to the old label model
func RoutingArguments2Labels(args []*apitraffic.SourceMatch) map[string]*apimodel.MatchString {
	labels := make(map[string]*apimodel.MatchString)
	for i := range args {
		argument := args[i]
		var key string
		switch argument.Type {
		case apitraffic.SourceMatch_CUSTOM:
			key = argument.Key
		case apitraffic.SourceMatch_METHOD:
			key = _labelKeyMethod
		case apitraffic.SourceMatch_HEADER:
			key = _labelKeyHeader + "." + argument.Key
		case apitraffic.SourceMatch_QUERY:
			key = _labelKeyQuery + "." + argument.Key
		case apitraffic.SourceMatch_CALLER_IP:
			key = _labelKeyCallerIP
		case apitraffic.SourceMatch_COOKIE:
			key = _labelKeyCookie + "." + argument.Key
		case apitraffic.SourceMatch_PATH:
			key = _labelKeyPath
		default:
			continue
		}

		labels[key] = &apimodel.MatchString{
			Type:      argument.GetValue().GetType(),
			Value:     argument.GetValue().GetValue(),
			ValueType: argument.GetValue().GetValueType(),
		}
	}

	return labels
}

// BuildV2RoutingFromV1Route Build a V2 version of API data object routing rules
func BuildV2RoutingFromV1Route(req *apitraffic.Routing, route *apitraffic.Route) (*apitraffic.RouteRule, error) {
	var v2Id string
	if extendInfo := route.GetExtendInfo(); len(extendInfo) > 0 {
		v2Id = extendInfo[V2RuleIDKey]
	} else {
		v2Id = utils.NewRoutingV2UUID()
	}

	rule := convertV1RouteToV2Route(route)
	any, err := ptypes.MarshalAny(rule)
	if err != nil {
		return nil, err
	}

	routing := &apitraffic.RouteRule{
		Id:            v2Id,
		Name:          "",
		Enable:        false,
		RoutingPolicy: apitraffic.RoutingPolicy_RulePolicy,
		RoutingConfig: any,
		Revision:      utils.NewV2Revision(),
		Priority:      0,
	}

	return routing, nil
}

// BuildV2ExtendRouting Build the internal data object routing rules of V2 version
func BuildV2ExtendRouting(req *apitraffic.Routing, route *apitraffic.Route) (*ExtendRouterConfig, error) {
	var v2Id string
	if extendInfo := route.GetExtendInfo(); len(extendInfo) > 0 {
		v2Id = extendInfo[V2RuleIDKey]
	}
	if v2Id == "" {
		v2Id = utils.NewRoutingV2UUID()
	}

	routing := &ExtendRouterConfig{
		RouterConfig: &RouterConfig{
			ID:       v2Id,
			Name:     v2Id,
			Enable:   true,
			Policy:   apitraffic.RoutingPolicy_RulePolicy.String(),
			Revision: req.GetRevision().GetValue(),
			Priority: 0,
		},
		RuleRouting: convertV1RouteToV2Route(route),
	}

	return routing, nil
}

// convertV1RouteToV2Route Turn the routing rules of the V1 version to the routing rules of V2 version
func convertV1RouteToV2Route(route *apitraffic.Route) *apitraffic.RuleRoutingConfig {
	v2sources := make([]*apitraffic.SourceService, 0, len(route.GetSources()))
	v1sources := route.GetSources()
	for i := range v1sources {
		entry := &apitraffic.SourceService{
			Service:   v1sources[i].GetService().GetValue(),
			Namespace: v1sources[i].GetNamespace().GetValue(),
		}

		entry.Arguments = RoutingLabels2Arguments(v1sources[i].GetMetadata())
		v2sources = append(v2sources, entry)
	}

	v2destinations := make([]*apitraffic.DestinationGroup, 0, len(route.GetDestinations()))
	v1destinations := route.GetDestinations()
	for i := range v1destinations {
		entry := &apitraffic.DestinationGroup{
			Service:   v1destinations[i].GetService().GetValue(),
			Namespace: v1destinations[i].GetNamespace().GetValue(),
			Priority:  v1destinations[i].GetPriority().GetValue(),
			Weight:    v1destinations[i].GetWeight().GetValue(),
			Transfer:  v1destinations[i].GetTransfer().GetValue(),
			Isolate:   v1destinations[i].GetIsolate().GetValue(),
		}

		v2labels := make(map[string]*apimodel.MatchString)
		v1labels := v1destinations[i].GetMetadata()
		for index := range v1labels {
			v2labels[index] = &apimodel.MatchString{
				Type:      v1labels[index].GetType(),
				Value:     v1labels[index].GetValue(),
				ValueType: v1labels[index].GetValueType(),
			}
		}

		entry.Labels = v2labels
		v2destinations = append(v2destinations, entry)
	}

	return &apitraffic.RuleRoutingConfig{
		Rules: []*apitraffic.SubRuleRouting{
			{
				Sources:      v2sources,
				Destinations: v2destinations,
			},
		},
	}
}

// CompareRoutingV2 Compare the priority of two routing.
func CompareRoutingV2(a, b *ExtendRouterConfig) bool {
	if a.Priority != b.Priority {
		return a.Priority < b.Priority
	}
	return a.CreateTime.Before(b.CreateTime)
}

// ConvertRoutingV1ToExtendV2 The routing rules of the V1 version are converted to V2 version for storage
// TODO Reduce duplicate code logic
func ConvertRoutingV1ToExtendV2(svcName, svcNamespace string,
	rule *RoutingConfig) ([]*ExtendRouterConfig, []*ExtendRouterConfig, error) {
	inRet := make([]*ExtendRouterConfig, 0, 4)
	outRet := make([]*ExtendRouterConfig, 0, 4)

	if rule.InBounds != "" {
		var inBounds []*apitraffic.Route
		if err := json.Unmarshal([]byte(rule.InBounds), &inBounds); err != nil {
			return nil, nil, err
		}

		priorityMax := 0

		for i := range inBounds {
			routing, err := BuildV2ExtendRouting(&apitraffic.Routing{
				Namespace: utils.NewStringValue(svcNamespace),
			}, inBounds[i])
			if err != nil {
				return nil, nil, err
			}
			routing.ID = fmt.Sprintf("%sin%d", rule.ID, i)
			routing.Revision = rule.Revision
			routing.Enable = true
			routing.CreateTime = rule.CreateTime
			routing.ModifyTime = rule.ModifyTime
			routing.EnableTime = rule.CreateTime
			routing.ExtendInfo = map[string]string{
				V1RuleIDKey:         rule.ID,
				V1RuleRouteIndexKey: fmt.Sprintf("%d", i),
				V1RuleRouteTypeKey:  V1RuleInRoute,
			}

			if priorityMax > 10 {
				priorityMax = 10
			}

			routing.Priority = uint32(priorityMax)
			priorityMax++

			inRet = append(inRet, routing)
		}
	}
	if rule.OutBounds != "" {
		var outBounds []*apitraffic.Route
		if err := json.Unmarshal([]byte(rule.OutBounds), &outBounds); err != nil {
			return nil, nil, err
		}

		priorityMax := 0

		for i := range outBounds {
			routing, err := BuildV2ExtendRouting(&apitraffic.Routing{
				Namespace: utils.NewStringValue(svcNamespace),
			}, outBounds[i])
			if err != nil {
				return nil, nil, err
			}
			routing.ID = fmt.Sprintf("%sout%d", rule.ID, i)
			routing.Revision = rule.Revision
			routing.CreateTime = rule.CreateTime
			routing.ModifyTime = rule.ModifyTime
			routing.EnableTime = rule.CreateTime
			routing.ExtendInfo = map[string]string{
				V1RuleIDKey:         rule.ID,
				V1RuleRouteIndexKey: fmt.Sprintf("%d", i),
				V1RuleRouteTypeKey:  V1RuleOutRoute,
			}

			if priorityMax > 10 {
				priorityMax = 10
			}

			routing.Priority = uint32(priorityMax)
			priorityMax++

			outRet = append(outRet, routing)
		}
	}

	return inRet, outRet, nil
}
