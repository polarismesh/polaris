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

	types "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

// forceUpdate 更新配置
func (rlc *rateLimitCache) forceUpdate() error {
	if err := rlc.Update(); err != nil {
		return err
	}
	return nil
}

// QueryRateLimitRules
func (rlc *rateLimitCache) QueryRateLimitRules(ctx context.Context, args types.RateLimitRuleArgs) (uint32, []*model.RateLimit, error) {
	if err := rlc.forceUpdate(); err != nil {
		return 0, nil, err
	}

	hasService := len(args.Service) != 0
	hasNamespace := len(args.Namespace) != 0

	res := make([]*model.RateLimit, 0, 8)
	process := func(rule *model.RateLimit) {
		if hasService && args.Service != rule.Proto.GetService().GetValue() {
			return
		}
		if hasNamespace && args.Namespace != rule.Proto.GetNamespace().GetValue() {
			return
		}
		if args.ID != "" && args.ID != rule.ID {
			return
		}
		if args.Name != "" {
			name, _ := utils.ParseWildName(args.Name)
			if !strings.Contains(rule.Name, name) {
				return
			}
		}

		if args.Disable != nil && *args.Disable != rule.Disable {
			return
		}
		res = append(res, rule)
	}
	rlc.IteratorRateLimit(process)
	amount, routings := rlc.sortBeforeTrim(res, args)
	return amount, routings, nil
}

func (rlc *rateLimitCache) sortBeforeTrim(rules []*model.RateLimit,
	args types.RateLimitRuleArgs) (uint32, []*model.RateLimit) {
	amount := uint32(len(rules))
	if args.Offset >= amount || args.Limit == 0 {
		return amount, nil
	}
	sort.Slice(rules, func(i, j int) bool {
		asc := strings.ToLower(args.OrderType) == "asc" || args.OrderType == ""
		if strings.ToLower(args.OrderField) == "priority" {
			return orderByRateLimitPriority(rules[i], rules[j], asc)
		}
		return orderByRateLimitModifyTime(rules[i], rules[j], asc)
	})
	endIdx := args.Offset + args.Limit
	if endIdx > amount {
		endIdx = amount
	}
	return amount, rules[args.Offset:endIdx]
}

func orderByRateLimitPriority(a, b *model.RateLimit, asc bool) bool {
	if a.Priority < b.Priority {
		return asc
	}
	if a.Priority > b.Priority {
		// false && asc always false
		return false
	}
	return strings.Compare(a.ID, b.ID) < 0 && asc
}

func orderByRateLimitModifyTime(a, b *model.RateLimit, asc bool) bool {
	if a.ModifyTime.After(b.ModifyTime) {
		return asc
	}
	if a.ModifyTime.Before(b.ModifyTime) {
		return false
	}
	return strings.Compare(a.ID, b.ID) < 0 && asc
}
