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

package service

import (
	"context"
	"sort"
	"strings"

	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"

	types "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

// forceUpdate 更新配置
func (rc *RouteRuleCache) forceUpdate() error {
	if err := rc.Update(); err != nil {
		return err
	}
	return nil
}

func queryRoutingRuleV2ByService(rule *model.ExtendRouterConfig, sourceNamespace, sourceService,
	destNamespace, destService string, both bool) bool {
	var (
		sourceFind bool
		destFind   bool
	)

	hasSourceSvc := len(sourceService) != 0
	hasSourceNamespace := len(sourceNamespace) != 0
	hasDestSvc := len(destService) != 0
	hasDestNamespace := len(destNamespace) != 0

	sourceService, isWildSourceSvc := utils.ParseWildName(sourceService)
	sourceNamespace, isWildSourceNamespace := utils.ParseWildName(sourceNamespace)
	destService, isWildDestSvc := utils.ParseWildName(destService)
	destNamespace, isWildDestNamespace := utils.ParseWildName(destNamespace)

	for i := range rule.RuleRouting.RuleRouting.Rules {
		subRule := rule.RuleRouting.RuleRouting.Rules[i]
		sources := subRule.GetSources()
		if hasSourceNamespace || hasSourceSvc {
			for i := range sources {
				item := sources[i]
				if hasSourceSvc {
					if isWildSourceSvc {
						if !strings.Contains(item.Service, sourceService) {
							continue
						}
					} else if item.Service != sourceService {
						continue
					}
				}
				if hasSourceNamespace {
					if isWildSourceNamespace {
						if !strings.Contains(item.Namespace, sourceNamespace) {
							continue
						}
					} else if item.Namespace != sourceNamespace {
						continue
					}
				}
				sourceFind = true
				break
			}
		}

		destinations := subRule.GetDestinations()
		if hasDestNamespace || hasDestSvc {
			for i := range destinations {
				item := destinations[i]
				if hasDestSvc {
					if isWildDestSvc && !strings.Contains(item.Service, destService) {
						continue
					}
					if item.Service != destService {
						continue
					}
				}
				if hasDestNamespace {
					if isWildDestNamespace && !strings.Contains(item.Namespace, destNamespace) {
						continue
					}
					if item.Namespace != destNamespace {
						continue
					}
				}
				destFind = true
				break
			}
		}

		if both {
			if sourceFind && destFind {
				return true
			}
		} else if sourceFind || destFind {
			return true
		}
	}
	return false
}

// QueryRoutingConfigsV2 Query Route Configuration List
func (rc *RouteRuleCache) QueryRoutingConfigsV2(ctx context.Context, args *types.RoutingArgs) (uint32, []*model.ExtendRouterConfig, error) {
	if err := rc.forceUpdate(); err != nil {
		return 0, nil, err
	}
	hasSvcQuery := len(args.Service) != 0 || len(args.Namespace) != 0
	hasSourceQuery := len(args.SourceService) != 0 || len(args.SourceNamespace) != 0
	hasDestQuery := len(args.DestinationService) != 0 || len(args.DestinationNamespace) != 0
	needBoth := hasSourceQuery && hasDestQuery

	res := make([]*model.ExtendRouterConfig, 0, 8)

	var process = func(_ string, routeRule *model.ExtendRouterConfig) {
		if args.ID != "" && args.ID != routeRule.ID {
			return
		}

		if routeRule.GetRoutingPolicy() == apitraffic.RoutingPolicy_RulePolicy {
			if args.Namespace != "" {
				if args.SourceNamespace == "" {
					args.SourceNamespace = args.Namespace
				}
				if args.DestinationNamespace == "" {
					args.DestinationNamespace = args.Namespace
				}
			}
			if args.Service != "" {
				if args.SourceService == "" {
					args.SourceService = args.Service
				}
				if args.DestinationService == "" {
					args.DestinationService = args.Service
				}
			}
			if hasSvcQuery || hasSourceQuery || hasDestQuery {
				if !queryRoutingRuleV2ByService(routeRule,
					args.SourceNamespace, args.SourceService,
					args.DestinationNamespace, args.DestinationService,
					needBoth) {
					return
				}
			}
		}

		if args.Name != "" {
			name, isWild := utils.ParseWildName(args.Name)
			if isWild {
				if !strings.Contains(routeRule.Name, name) {
					return
				}
			} else if args.Name != routeRule.Name {
				return
			}
		}

		if args.Enable != nil && *args.Enable != routeRule.Enable {
			return
		}

		res = append(res, routeRule)
	}

	rc.IteratorRouterRule(func(key string, value *model.ExtendRouterConfig) {
		process(key, value)
	})

	amount, routings := rc.sortBeforeTrim(res, args)
	return amount, routings, nil
}

func (rc *RouteRuleCache) sortBeforeTrim(routings []*model.ExtendRouterConfig,
	args *types.RoutingArgs) (uint32, []*model.ExtendRouterConfig) {
	amount := uint32(len(routings))
	if args.Offset >= amount || args.Limit == 0 {
		return amount, nil
	}
	sort.Slice(routings, func(i, j int) bool {
		asc := strings.ToLower(args.OrderType) == "asc" || args.OrderType == ""
		if strings.ToLower(args.OrderField) == "priority" {
			return orderByRoutingPriority(routings[i], routings[j], asc)
		}
		return orderByRoutingModifyTime(routings[i], routings[j], asc)
	})
	endIdx := args.Offset + args.Limit
	if endIdx > amount {
		endIdx = amount
	}
	return amount, routings[args.Offset:endIdx]
}

func orderByRoutingPriority(a, b *model.ExtendRouterConfig, asc bool) bool {
	if a.Priority < b.Priority {
		return asc
	}
	if a.Priority > b.Priority {
		// false && asc always false
		return false
	}
	return strings.Compare(a.ID, b.ID) < 0 && asc
}

func orderByRoutingModifyTime(a, b *model.ExtendRouterConfig, asc bool) bool {
	if a.ModifyTime.After(b.ModifyTime) {
		return asc
	}
	if a.ModifyTime.Before(b.ModifyTime) {
		// false && asc always false
		return false
	}
	return strings.Compare(a.ID, b.ID) < 0 && asc
}
