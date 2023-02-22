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

package cache

import (
	"sort"
	"strings"

	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

// RoutingArgs Routing rules query parameters
type RoutingArgs struct {
	// Filter extend filter params
	Filter map[string]string
	// ID route rule id
	ID string
	// Name route rule name
	Name string
	// Service service name
	Service string
	// Namespace namesapce
	Namespace string
	// SourceService source service name
	SourceService string
	// SourceNamespace source service namespace
	SourceNamespace string
	// DestinationService destination service name
	DestinationService string
	// DestinationNamespace destination service namespace
	DestinationNamespace string
	// Enable
	Enable *bool
	// Offset
	Offset uint32
	// Limit
	Limit uint32
	// OrderField Sort field
	OrderField string
	// OrderType Sorting rules
	OrderType string
}

// forceUpdate 更新配置
func (rc *routingConfigCache) forceUpdate() error {
	if err := rc.update(); err != nil {
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

	for i := range rule.RuleRouting.Rules {
		subRule := rule.RuleRouting.Rules[i]
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

// GetRoutingConfigsV2 Query Route Configuration List
func (rc *routingConfigCache) QueryRoutingConfigsV2(args *RoutingArgs) (uint32, []*model.ExtendRouterConfig, error) {
	if err := rc.forceUpdate(); err != nil {
		return 0, nil, err
	}
	hasSourceQuery := len(args.SourceService) != 0 || len(args.SourceNamespace) != 0
	hasDestQuery := len(args.DestinationService) != 0 || len(args.DestinationNamespace) != 0

	res := make([]*model.ExtendRouterConfig, 0, 8)

	var process = func(_ string, routeRule *model.ExtendRouterConfig) {
		if args.ID != "" && args.ID != routeRule.ID {
			return
		}

		if routeRule.GetRoutingPolicy() == apitraffic.RoutingPolicy_RulePolicy {
			if args.Namespace != "" && args.Service != "" {
				if !queryRoutingRuleV2ByService(routeRule, args.Namespace, args.Service,
					args.Namespace, args.Service, false) {
					return
				}
			}

			if hasSourceQuery || hasDestQuery {
				if !queryRoutingRuleV2ByService(routeRule, args.SourceNamespace, args.SourceService, args.DestinationNamespace,
					args.DestinationService, hasSourceQuery && hasDestQuery) {
					return
				}
			}
		}

		if args.Name != "" {
			name, fuzzy := utils.ParseWildName(args.Name)
			if fuzzy {
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

	rc.IteratorRoutings(func(key string, value *model.ExtendRouterConfig) {
		process(key, value)
	})

	amount, routings := rc.sortBeforeTrim(res, args)
	return amount, routings, nil
}

func (rc *routingConfigCache) sortBeforeTrim(routings []*model.ExtendRouterConfig,
	args *RoutingArgs) (uint32, []*model.ExtendRouterConfig) {
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

func orderByRoutingModifyTime(a, b *model.ExtendRouterConfig, asc bool) bool {
	if a.Priority < b.Priority {
		return asc
	}

	if a.Priority > b.Priority {
		// false && asc always false
		return false
	}

	return strings.Compare(a.ID, b.ID) < 0 && asc
}

func orderByRoutingPriority(a, b *model.ExtendRouterConfig, asc bool) bool {
	if a.ModifyTime.After(b.ModifyTime) {
		return asc
	}

	if a.ModifyTime.Before(b.ModifyTime) {
		// false && asc always false
		return false
	}

	return strings.Compare(a.ID, b.ID) < 0 && asc
}
