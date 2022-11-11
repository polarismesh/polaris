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

	apiv2 "github.com/polarismesh/polaris/common/api/v2"
	v2 "github.com/polarismesh/polaris/common/model/v2"
	"github.com/polarismesh/polaris/common/utils"
)

// RoutingArgs 路由规则查询参数
type RoutingArgs struct {
	// Filter
	Filter map[string]string
	// ID 路由规则 ID
	ID string
	// Name 条件中的服务名
	Name string
	// Service 主调 or 被调服务名称
	Service string
	// Namespace 主调 or 被调服务所在命名空间
	Namespace string
	// SourceService 主调服务
	SourceService string
	// SourceNamespace 主调服务所在命名空间
	SourceNamespace string
	// DestinationService 被调服务
	DestinationService string
	// DestinationNamespace 被调服务所在命名空间
	DestinationNamespace string
	// Enable
	Enable *bool
	// Offset
	Offset uint32
	// Limit
	Limit uint32
	// OrderField 排序字段
	OrderField string
	// OrderType 排序规则
	OrderType string
}

// forceUpdate 更新配置
func (rc *routingConfigCache) forceUpdate() error {
	if err := rc.update(0); err != nil {
		return err
	}
	return nil
}

func queryRoutingRuleV2ByService(rule *v2.ExtendRoutingConfig, sourceNamespace, sourceService,
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

	sources := rule.RuleRouting.GetSources()
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

	destinations := rule.RuleRouting.GetDestinations()
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
		return sourceFind && destFind
	}
	return sourceFind || destFind
}

// GetRoutingConfigsV2 查询路由配置列表
func (rc *routingConfigCache) GetRoutingConfigsV2(args *RoutingArgs) (uint32, []*v2.ExtendRoutingConfig, error) {
	if err := rc.forceUpdate(); err != nil {
		return 0, nil, err
	}
	hasSourceQuery := len(args.SourceService) != 0 || len(args.SourceNamespace) != 0
	hasDestQuery := len(args.DestinationService) != 0 || len(args.DestinationNamespace) != 0

	res := make([]*v2.ExtendRoutingConfig, 0, 8)

	var process = func(_ string, svc *v2.ExtendRoutingConfig) {
		if args.ID != "" && args.ID != svc.ID {
			return
		}

		if svc.GetRoutingPolicy() == apiv2.RoutingPolicy_RulePolicy {
			// 主被调服务都查询
			if args.Namespace != "" && args.Service != "" {
				if !queryRoutingRuleV2ByService(svc, args.Namespace, args.Service,
					args.Namespace, args.Service, false) {
					return
				}
			}

			if hasSourceQuery || hasDestQuery {
				if !queryRoutingRuleV2ByService(svc, args.SourceNamespace, args.SourceService, args.DestinationNamespace,
					args.DestinationService, hasSourceQuery && hasDestQuery) {
					return
				}
			}
		}

		if args.Name != "" {
			name, fuzzy := utils.ParseWildName(args.Name)
			if fuzzy {
				if !strings.Contains(svc.Name, name) {
					return
				}
			} else if args.Name != svc.Name {
				return
			}
		}

		if args.Enable != nil && *args.Enable != svc.Enable {
			return
		}

		res = append(res, svc)
	}

	rc.IteratorRoutings(func(key string, value *v2.ExtendRoutingConfig) {
		process(key, value)
	})

	amount, routings := rc.sortBeforeTrim(res, args)
	return amount, routings, nil
}

func (rc *routingConfigCache) sortBeforeTrim(routings []*v2.ExtendRoutingConfig,
	args *RoutingArgs) (uint32, []*v2.ExtendRoutingConfig) {
	// 所有符合条件的路由规则数量
	amount := uint32(len(routings))
	// 判断 offset 和 limit 是否允许返回对应的服务
	if args.Offset >= amount || args.Limit == 0 {
		return amount, nil
	}
	// 将路由规则按照修改时间和 id 进行排序
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

func orderByRoutingModifyTime(a, b *v2.ExtendRoutingConfig, asc bool) bool {
	if a.Priority < b.Priority {
		return asc
	}

	if a.Priority > b.Priority {
		// false && asc always false
		return false
	}

	return strings.Compare(a.ID, b.ID) < 0 && asc
}

func orderByRoutingPriority(a, b *v2.ExtendRoutingConfig, asc bool) bool {
	if a.ModifyTime.After(b.ModifyTime) {
		return asc
	}

	if a.ModifyTime.Before(b.ModifyTime) {
		// false && asc always false
		return false
	}

	return strings.Compare(a.ID, b.ID) < 0 && asc
}
