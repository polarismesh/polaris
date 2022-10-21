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

package routing

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/golang/protobuf/ptypes"

	apiv1 "github.com/polarismesh/polaris/common/api/v1"
	apiv2 "github.com/polarismesh/polaris/common/api/v2"
	"github.com/polarismesh/polaris/common/model"
	v2 "github.com/polarismesh/polaris/common/model/v2"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
)

const (
	_labelKeyPath     = "$path"
	_labelKeyMethod   = "$method"
	_labelKeyHeader   = "$header"
	_labelKeyQuery    = "$query"
	_labelKeyCallerIP = "$caller_ip"
	_labelKeyCookie   = "$cookie"

	MatchAll = "*"
)

// RoutingConfigV1ToAPI 把内部数据结构转换为API参数传递出去
func RoutingConfigV1ToAPI(req *model.RoutingConfig, service string, namespace string) (*apiv1.Routing, error) {
	if req == nil {
		return nil, nil
	}

	out := &apiv1.Routing{
		Service:   utils.NewStringValue(service),
		Namespace: utils.NewStringValue(namespace),
		Revision:  utils.NewStringValue(req.Revision),
		Ctime:     utils.NewStringValue(commontime.Time2String(req.CreateTime)),
		Mtime:     utils.NewStringValue(commontime.Time2String(req.ModifyTime)),
	}

	if req.InBounds != "" {
		var inBounds []*apiv1.Route
		if err := json.Unmarshal([]byte(req.InBounds), &inBounds); err != nil {
			return nil, err
		}
		out.Inbounds = inBounds
	}
	if req.OutBounds != "" {
		var outBounds []*apiv1.Route
		if err := json.Unmarshal([]byte(req.OutBounds), &outBounds); err != nil {
			return nil, err
		}
		out.Outbounds = outBounds
	}

	return out, nil
}

// CompositeRoutingV1AndV2 合并 v1 版本的路由规则以及 v2 版本的规则路由
func CompositeRoutingV1AndV2(v1rule *apiv1.Routing, level1, level2,
	level3 []*v2.ExtendRoutingConfig) (*apiv1.Routing, []string) {

	// 先确保规则的排序是从最高优先级开始排序
	sort.Slice(level1, func(i, j int) bool {
		return CompareRoutingV2(level1[i], level1[j])
	})

	// 先确保规则的排序是从最高优先级开始排序
	sort.Slice(level2, func(i, j int) bool {
		return CompareRoutingV2(level2[i], level2[j])
	})

	// 先确保规则的排序是从最高优先级开始排序
	sort.Slice(level3, func(i, j int) bool {
		return CompareRoutingV2(level3[i], level3[j])
	})

	level1inRoutes, level1outRoutes, level1Revisions := BuildV1RoutesFromV2(v1rule.Service.Value, v1rule.Namespace.Value, level1)
	level2inRoutes, level2outRoutes, level2Revisions := BuildV1RoutesFromV2(v1rule.Service.Value, v1rule.Namespace.Value, level2)
	level3inRoutes, level3outRoutes, level3Revisions := BuildV1RoutesFromV2(v1rule.Service.Value, v1rule.Namespace.Value, level3)

	inBounds := v1rule.GetInbounds()
	outBounds := v1rule.GetOutbounds()

	// 处理 inbounds 规则，level1 cache -> v1rules -> level2 cache -> level3 cache
	if len(level1inRoutes) > 0 {
		v1rule.Inbounds = append(level1inRoutes, inBounds...)
	}
	if len(level2inRoutes) > 0 {
		v1rule.Inbounds = append(v1rule.Inbounds, level2inRoutes...)
	}
	if len(level3inRoutes) > 0 {
		v1rule.Inbounds = append(v1rule.Inbounds, level3inRoutes...)
	}

	// 处理 outbounds 规则，level1 cache -> v1rules -> level2 cache -> level3 cache
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

// BuildV1RoutesFromV2 根据 v2 版本的路由规则适配成 v1 版本的路由规则，分为别 inBounds 以及 outBounds
// retuen inBound outBound revisions
func BuildV1RoutesFromV2(service, namespace string, entries []*v2.ExtendRoutingConfig) ([]*apiv1.Route, []*apiv1.Route, []string) {
	if len(entries) == 0 {
		return []*apiv1.Route{}, []*apiv1.Route{}, []string{}
	}

	// 将 v2rules 分为 inbound 以及 outbound
	revisions := make([]string, 0, len(entries))
	outRoutes := make([]*apiv1.Route, 0, 8)
	inRoutes := make([]*apiv1.Route, 0, 8)
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

// BuildOutBoundsFromV2 根据 v2 版本的路由规则适配成 v1 版本的路由规则中的 OutBounds
func BuildOutBoundsFromV2(service, namespace string, item *v2.ExtendRoutingConfig) []*apiv1.Route {
	if item.GetRoutingPolicy() != apiv2.RoutingPolicy_RulePolicy {
		return []*apiv1.Route{}
	}

	var find bool

	matchService := func(source *apiv2.Source) bool {
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

	v1sources := make([]*apiv1.Source, 0, len(item.RuleRouting.Sources))
	sources := item.RuleRouting.Sources
	for i := range sources {
		if matchService(sources[i]) {
			find = true
			entry := &apiv1.Source{
				Service:   utils.NewStringValue(service),
				Namespace: utils.NewStringValue(namespace),
			}
			entry.Metadata = RoutingArguments2Labels(sources[i].GetArguments())
			v1sources = append(v1sources, entry)
		}
	}

	if !find {
		return []*apiv1.Route{}
	}

	v1destinations := make([]*apiv1.Destination, 0, len(item.RuleRouting.Destinations))
	destinations := item.RuleRouting.Destinations
	for i := range destinations {
		entry := &apiv1.Destination{
			Service:   utils.NewStringValue(destinations[i].Service),
			Namespace: utils.NewStringValue(destinations[i].Namespace),
			Priority:  utils.NewUInt32Value(destinations[i].GetPriority()),
			Weight:    utils.NewUInt32Value(destinations[i].GetWeight()),
			Transfer:  utils.NewStringValue(destinations[i].GetTransfer()),
			Isolate:   utils.NewBoolValue(destinations[i].GetIsolate()),
		}

		v1labels := make(map[string]*apiv1.MatchString)
		v2labels := destinations[i].GetLabels()
		for index := range v2labels {
			v1labels[index] = &apiv1.MatchString{
				Type:      apiv1.MatchString_MatchStringType(v2labels[index].GetType()),
				Value:     v2labels[index].GetValue(),
				ValueType: apiv1.MatchString_ValueType(v2labels[index].GetValueType()),
			}
		}

		entry.Metadata = v1labels
		v1destinations = append(v1destinations, entry)
	}

	return []*apiv1.Route{
		{
			Sources:      v1sources,
			Destinations: v1destinations,
			ExtendInfo: map[string]string{
				v2.V2RuleIDKey: item.ID,
			},
		},
	}
}

// BuildInBoundsFromV2 将 v2 的路由规则转为 v1 的路由规则中的 InBounds
func BuildInBoundsFromV2(service, namespace string, item *v2.ExtendRoutingConfig) []*apiv1.Route {
	if item.GetRoutingPolicy() != apiv2.RoutingPolicy_RulePolicy {
		return []*apiv1.Route{}
	}

	var find bool

	matchService := func(destination *apiv2.Destination) bool {
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

	v1destinations := make([]*apiv1.Destination, 0, len(item.RuleRouting.Destinations))
	destinations := item.RuleRouting.Destinations
	for i := range destinations {
		if matchService(destinations[i]) {
			find = true
			entry := &apiv1.Destination{
				Service:   utils.NewStringValue(service),
				Namespace: utils.NewStringValue(namespace),
				Priority:  utils.NewUInt32Value(destinations[i].GetPriority()),
				Weight:    utils.NewUInt32Value(destinations[i].GetWeight()),
				Transfer:  utils.NewStringValue(destinations[i].GetTransfer()),
				Isolate:   utils.NewBoolValue(destinations[i].GetIsolate()),
			}

			v1labels := make(map[string]*apiv1.MatchString)
			v2labels := destinations[i].GetLabels()
			for index := range v2labels {
				v1labels[index] = &apiv1.MatchString{
					Type:      apiv1.MatchString_MatchStringType(v2labels[index].GetType()),
					Value:     v2labels[index].GetValue(),
					ValueType: apiv1.MatchString_ValueType(v2labels[index].GetValueType()),
				}
			}

			entry.Metadata = v1labels
			v1destinations = append(v1destinations, entry)
		}
	}

	if !find {
		return []*apiv1.Route{}
	}

	v1sources := make([]*apiv1.Source, 0, len(item.RuleRouting.Sources))
	sources := item.RuleRouting.Sources
	for i := range sources {
		entry := &apiv1.Source{
			Service:   utils.NewStringValue(sources[i].Service),
			Namespace: utils.NewStringValue(sources[i].Namespace),
		}

		entry.Metadata = RoutingArguments2Labels(sources[i].GetArguments())
		v1sources = append(v1sources, entry)
	}

	return []*apiv1.Route{
		{
			Sources:      v1sources,
			Destinations: v1destinations,
		},
	}
}

// RoutingLabels2Arguments 将旧的标签模型适配成参数列表
func RoutingLabels2Arguments(labels map[string]*apiv1.MatchString) []*apiv2.SourceMatch {
	if len(labels) == 0 {
		return []*apiv2.SourceMatch{}
	}

	arguments := make([]*apiv2.SourceMatch, 0, 4)
	for index := range labels {
		arguments = append(arguments, &apiv2.SourceMatch{
			Type: apiv2.SourceMatch_CUSTOM,
			Key:  index,
			Value: &apiv2.MatchString{
				Type:      apiv2.MatchString_MatchStringType(labels[index].GetType()),
				Value:     labels[index].GetValue(),
				ValueType: apiv2.MatchString_ValueType(labels[index].GetValueType()),
			},
		})
	}

	return arguments
}

// RoutingArguments2Labels 将参数列表适配成旧的标签模型
func RoutingArguments2Labels(args []*apiv2.SourceMatch) map[string]*apiv1.MatchString {
	labels := make(map[string]*apiv1.MatchString)
	for i := range args {
		argument := args[i]
		var key string
		switch argument.Type {
		case apiv2.SourceMatch_CUSTOM:
			key = argument.Key
		case apiv2.SourceMatch_METHOD:
			key = _labelKeyMethod
		case apiv2.SourceMatch_HEADER:
			key = _labelKeyHeader + "." + argument.Key
		case apiv2.SourceMatch_QUERY:
			key = _labelKeyQuery + "." + argument.Key
		case apiv2.SourceMatch_CALLER_IP:
			key = _labelKeyCallerIP
		case apiv2.SourceMatch_COOKIE:
			key = _labelKeyCookie + "." + argument.Key
		case apiv2.SourceMatch_PATH:
			key = _labelKeyPath + "." + argument.Key
		default:
			continue
		}

		labels[key] = &apiv1.MatchString{
			Type:      apiv1.MatchString_MatchStringType(argument.GetValue().GetType()),
			Value:     argument.GetValue().GetValue(),
			ValueType: apiv1.MatchString_ValueType(argument.GetValue().GetValueType()),
		}
	}

	return labels
}

// BuildV2RoutingFromV1Route 构建 v2 版本的API数据对象路由规则
func BuildV2RoutingFromV1Route(req *apiv1.Routing, route *apiv1.Route) (*apiv2.Routing, error) {
	rule := ConvertV1RouteToV2Route(route)
	any, err := ptypes.MarshalAny(rule)
	if err != nil {
		return nil, err
	}

	var v2Id string
	if extendInfo := route.GetExtendInfo(); len(extendInfo) > 0 {
		v2Id = extendInfo[v2.V2RuleIDKey]
	} else {
		v2Id = utils.NewRoutingV2UUID()
	}

	routing := &apiv2.Routing{
		Id:            v2Id,
		Name:          "",
		Enable:        false,
		RoutingPolicy: apiv2.RoutingPolicy_RulePolicy,
		RoutingConfig: any,
		Revision:      utils.NewV2Revision(),
		Priority:      0,
	}

	return routing, nil
}

// BuildV2ExtendRouting 构建 v2 版本的内部数据对象路由规则
func BuildV2ExtendRouting(req *apiv1.Routing, route *apiv1.Route) (*v2.ExtendRoutingConfig, error) {
	rule := ConvertV1RouteToV2Route(route)

	var v2Id string
	if extendInfo := route.GetExtendInfo(); len(extendInfo) > 0 {
		v2Id = extendInfo[v2.V2RuleIDKey]
	}
	if v2Id == "" {
		v2Id = utils.NewRoutingV2UUID()
	}

	routing := &v2.ExtendRoutingConfig{
		RoutingConfig: &v2.RoutingConfig{
			ID:       v2Id,
			Name:     "",
			Enable:   true,
			Policy:   apiv2.RoutingPolicy_RulePolicy.String(),
			Revision: req.GetRevision().GetValue(),
			Priority: 0,
		},
		RuleRouting: rule,
	}

	return routing, nil
}

// ConvertV1RouteToV2Route 将 v1 版本的路由规则转为 v2 版本的路由规则
func ConvertV1RouteToV2Route(route *apiv1.Route) *apiv2.RuleRoutingConfig {
	v2sources := make([]*apiv2.Source, 0, len(route.GetSources()))
	v1sources := route.GetSources()
	for i := range v1sources {
		entry := &apiv2.Source{
			Service:   v1sources[i].GetService().GetValue(),
			Namespace: v1sources[i].GetNamespace().GetValue(),
		}

		entry.Arguments = RoutingLabels2Arguments(v1sources[i].GetMetadata())
		v2sources = append(v2sources, entry)
	}

	v2destinations := make([]*apiv2.Destination, 0, len(route.GetDestinations()))
	v1destinations := route.GetDestinations()
	for i := range v1destinations {
		entry := &apiv2.Destination{
			Service:   v1destinations[i].GetService().GetValue(),
			Namespace: v1destinations[i].GetNamespace().GetValue(),
			Priority:  v1destinations[i].GetPriority().GetValue(),
			Weight:    v1destinations[i].GetWeight().GetValue(),
			Transfer:  v1destinations[i].GetTransfer().GetValue(),
			Isolate:   v1destinations[i].GetIsolate().GetValue(),
		}

		v2labels := make(map[string]*apiv2.MatchString)
		v1labels := v1destinations[i].GetMetadata()
		for index := range v1labels {
			v2labels[index] = &apiv2.MatchString{
				Type:      apiv2.MatchString_MatchStringType(v1labels[index].GetType()),
				Value:     v1labels[index].GetValue(),
				ValueType: apiv2.MatchString_ValueType(v1labels[index].GetValueType()),
			}
		}

		entry.Labels = v2labels
		v2destinations = append(v2destinations, entry)
	}

	return &apiv2.RuleRoutingConfig{
		Sources:      v2sources,
		Destinations: v2destinations,
	}
}

// CompareRoutingV2
func CompareRoutingV2(a, b *v2.ExtendRoutingConfig) bool {
	if a.Priority != b.Priority {
		return a.Priority < b.Priority
	}
	return a.CreateTime.Before(b.CreateTime)
}

// ConvertRoutingV1ToExtendV2 v1 版本的路由规则转为 v2 版本进行存储
func ConvertRoutingV1ToExtendV2(svcName, svcNamespace string, rule *model.RoutingConfig) ([]*v2.ExtendRoutingConfig, []*v2.ExtendRoutingConfig, error) {
	inRet := make([]*v2.ExtendRoutingConfig, 0, 4)
	outRet := make([]*v2.ExtendRoutingConfig, 0, 4)

	if rule.InBounds != "" {
		var inBounds []*apiv1.Route
		if err := json.Unmarshal([]byte(rule.InBounds), &inBounds); err != nil {
			return nil, nil, err
		}

		priorityMax := 0

		for i := range inBounds {
			routing, err := BuildV2ExtendRouting(&apiv1.Routing{
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
				v2.V1RuleIDKey:         rule.ID,
				v2.V1RuleRouteIndexKey: fmt.Sprintf("%d", i),
				v2.V1RuleRouteTypeKey:  v2.V1RuleInRoute,
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
		var outBounds []*apiv1.Route
		if err := json.Unmarshal([]byte(rule.OutBounds), &outBounds); err != nil {
			return nil, nil, err
		}

		priorityMax := 0

		for i := range outBounds {
			routing, err := BuildV2ExtendRouting(&apiv1.Routing{
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
				v2.V1RuleIDKey:         rule.ID,
				v2.V1RuleRouteIndexKey: fmt.Sprintf("%d", i),
				v2.V1RuleRouteTypeKey:  v2.V1RuleOutRoute,
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
