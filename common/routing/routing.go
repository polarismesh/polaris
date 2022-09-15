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

	"github.com/golang/protobuf/ptypes"
	apiv1 "github.com/polarismesh/polaris-server/common/api/v1"
	apiv2 "github.com/polarismesh/polaris-server/common/api/v2"
	"github.com/polarismesh/polaris-server/common/model"
	v2 "github.com/polarismesh/polaris-server/common/model/v2"
	commontime "github.com/polarismesh/polaris-server/common/time"
	"github.com/polarismesh/polaris-server/common/utils"
)

const (
	LabelKeyPath     = "$path"
	LabelKeyMethod   = "$method"
	LabelKeyHeader   = "$header"
	LabelKeyQuery    = "$query"
	LabelKeyCallerIP = "$caller_ip"
)

// RoutingV1Config2API 把内部数据结构转换为API参数传递出去
func RoutingV1Config2API(req *model.RoutingConfig, service string, namespace string) (*apiv1.Routing, error) {
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
func CompositeRoutingV1AndV2(v1rule *apiv1.Routing, entries []*v2.ExtendRoutingConfig) (*apiv1.Routing, []string) {
	inRoutes, outRoutes, revisions := BuildV1RoutesFromV2(entries)

	// 全部认为是以默认位置进行创建，即追加的方式
	inBounds := v1rule.GetInbounds()
	outBounds := v1rule.GetOutbounds()

	if len(inRoutes) > 0 {
		v1rule.Inbounds = append(inRoutes, inBounds...)
	}
	if len(outRoutes) > 0 {
		v1rule.Outbounds = append(outRoutes, outBounds...)
	}

	if len(revisions) > 0 {
		revisions = append(revisions, v1rule.GetRevision().GetValue())
	} else {
		revisions = []string{v1rule.GetRevision().GetValue()}
	}

	return v1rule, revisions
}

// BuildV1RoutesFromV2 根据 v2 版本的路由规则适配成 v1 版本的路由规则，分为别 inBounds 以及 outBounds
func BuildV1RoutesFromV2(entries []*v2.ExtendRoutingConfig) ([]*apiv1.Route, []*apiv1.Route, []string) {
	// 将 v2rules 分为 inbound 以及 outbound
	revisions := make([]string, 0, len(entries))
	outRoutes := make([]*apiv1.Route, 0, 8)
	inRoutes := make([]*apiv1.Route, 0, 8)
	for i := range entries {
		if !entries[i].Enable {
			continue
		}
		outRoutes = append(outRoutes, BuildOutBoundsFromV2(entries[i])...)
		inRoutes = append(inRoutes, BuildInBoundsFromV2(entries[i])...)
		revisions = append(revisions, entries[i].Revision)
	}

	return inRoutes, outRoutes, revisions
}

// BuildOutBoundsFromV2 根据 v2 版本的路由规则适配成 v1 版本的路由规则中的 OutBounds
func BuildOutBoundsFromV2(item *v2.ExtendRoutingConfig) []*apiv1.Route {
	if item.GetRoutingPolicy() != apiv2.RoutingPolicy_RulePolicy {
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
		},
	}
}

// BuildInBoundsFromV2 将 v2 的路由规则转为 v1 的路由规则中的 InBounds
func BuildInBoundsFromV2(item *v2.ExtendRoutingConfig) []*apiv1.Route {
	if item.GetRoutingPolicy() != apiv2.RoutingPolicy_RulePolicy {
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
		},
	}
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
			key = LabelKeyMethod
		case apiv2.SourceMatch_HEADER:
			key = LabelKeyHeader + "." + argument.Key
		case apiv2.SourceMatch_QUERY:
			key = LabelKeyQuery + "." + argument.Key
		case apiv2.SourceMatch_CALLER_IP:
			key = LabelKeyCallerIP
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

func BuildV2Routing(req *apiv1.Routing, route *apiv1.Route) (*apiv2.Routing, error) {
	rule := ConvertV1RouteToV2Route(route)
	any, err := ptypes.MarshalAny(rule)
	if err != nil {
		return nil, err
	}

	var v2Id string
	if extendInfo := route.GetExtendInfo(); len(extendInfo) > 0 {
		v2Id = extendInfo[v2.V2RuleIDKey]
	}

	routing := &apiv2.Routing{
		Id:            v2Id,
		Name:          "",
		Namespace:     req.GetNamespace().GetValue(),
		Enable:        false,
		RoutingPolicy: apiv2.RoutingPolicy_RulePolicy,
		RoutingConfig: any,
		Revision:      utils.NewV2Revision(),
		Priority:      0,
	}

	return routing, nil
}

func BuildV2ExtendRouting(req *apiv1.Routing, route *apiv1.Route) (*v2.ExtendRoutingConfig, error) {
	rule := ConvertV1RouteToV2Route(route)

	var v2Id string
	if extendInfo := route.GetExtendInfo(); len(extendInfo) > 0 {
		v2Id = extendInfo[v2.V2RuleIDKey]
	}

	routing := &v2.ExtendRoutingConfig{
		RoutingConfig: &v2.RoutingConfig{
			ID:        v2Id,
			Name:      "",
			Namespace: req.GetNamespace().GetValue(),
			Enable:    true,
			Policy:    apiv2.RoutingPolicy_RulePolicy.String(),
			Revision:  utils.NewV2Revision(),
			Priority:  0,
		},
		RuleRouting: rule,
	}

	return routing, nil
}

func ConvertV1RouteToV2Route(route *apiv1.Route) *apiv2.RuleRoutingConfig {
	v2sources := make([]*apiv2.Source, 0, len(route.GetSources()))
	v1sources := route.GetSources()
	for i := range v1sources {
		entry := &apiv2.Source{
			Service:   v1sources[i].GetService().GetValue(),
			Namespace: v1sources[i].GetNamespace().GetValue(),
		}

		v2metadata := make([]*apiv2.SourceMatch, 0, 4)
		v1metedata := v1sources[i].GetMetadata()
		for index := range v1metedata {
			v2metadata = append(v2metadata, &apiv2.SourceMatch{
				Type: apiv2.SourceMatch_CUSTOM,
				Key:  index,
				Value: &apiv2.MatchString{
					Type:      apiv2.MatchString_MatchStringType(v1metedata[index].GetType()),
					Value:     v1metedata[index].GetValue(),
					ValueType: apiv2.MatchString_ValueType(v1metedata[index].GetValueType()),
				},
			})
		}
		entry.Arguments = v2metadata
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
