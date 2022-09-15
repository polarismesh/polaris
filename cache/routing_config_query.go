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

	v2 "github.com/polarismesh/polaris-server/common/model/v2"
)

// RoutingArgs
type RoutingArgs struct {
	// Filter
	Filter map[string]string
	// Name 条件中的服务名
	Name string
	// FuzzyName
	FuzzyName bool
	// Namesapce
	Namespace string
	// Enable
	Enable *bool
	// Offset
	Offset uint32
	// Limit
	Limit uint32
}

// Update 更新配置
func (rc *routingConfigCache) ForceUpdate() error {
	if err := rc.update(0); err != nil {
		return err
	}
	return nil
}

// GetRoutingConfigsV2 查询路由配置列表
func (rc *routingConfigCache) GetRoutingConfigsV2(args *RoutingArgs) (uint32, []*v2.ExtendRoutingConfig, error) {

	res := make([]*v2.ExtendRoutingConfig, 0, 8)
	var process = func(_ string, svc *v2.ExtendRoutingConfig) {
		if args.Namespace != "" && svc.Namespace != args.Namespace {
			return
		}

		if args.Name != "" {
			if args.FuzzyName && !strings.Contains(svc.Name, args.Name[0:len(args.Name)-1]) {
				return
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

	amount, routings := rc.sortBeforeTrim(res, args.Offset, args.Offset)
	return amount, routings, nil
}

func (rc *routingConfigCache) sortBeforeTrim(routings []*v2.ExtendRoutingConfig,
	offset, limit uint32) (uint32, []*v2.ExtendRoutingConfig) {
	// 所有符合条件的路由规则数量
	amount := uint32(len(routings))
	// 判断 offset 和 limit 是否允许返回对应的服务
	if offset >= amount || limit == 0 {
		return amount, nil
	}
	// 将路由规则按照修改时间和 id 进行排序
	sort.Slice(routings, func(i, j int) bool {
		if routings[i].ModifyTime.After(routings[j].ModifyTime) {
			return true
		} else if routings[i].ModifyTime.Before(routings[j].ModifyTime) {
			return false
		} else {
			return strings.Compare(routings[i].ID, routings[j].ID) < 0
		}
	})
	endIdx := offset + limit
	if endIdx > amount {
		endIdx = amount
	}
	return amount, routings[offset:endIdx]
}
