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

package resource

import (
	"fmt"

	"github.com/golang/protobuf/ptypes/duration"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/utils"
)

/**
 * @brief 创建测试限流规则
 */
func CreateRateLimits(services []*api.Service) []*api.Rule {
	var rateLimits []*api.Rule
	for index := 0; index < 2; index++ {
		rateLimit := &api.Rule{
			Name:      utils.NewStringValue(fmt.Sprintf("rlimit-%d", index)),
			Service:   services[index].GetName(),
			Namespace: services[index].GetNamespace(),
			Priority:  utils.NewUInt32Value(uint32(index)),
			Resource:  api.Rule_CONCURRENCY,
			Type:      api.Rule_LOCAL,
			Arguments: []*api.MatchArgument{{
				Type: api.MatchArgument_CUSTOM,
				Key:  fmt.Sprintf("name-%d", index),
				Value: &api.MatchString{
					Type:  api.MatchString_REGEX,
					Value: utils.NewStringValue(fmt.Sprintf("value-%d", index)),
				},
			}, {Type: api.MatchArgument_CUSTOM,
				Key: fmt.Sprintf("name-%d", index+1),
				Value: &api.MatchString{
					Type:  api.MatchString_EXACT,
					Value: utils.NewStringValue(fmt.Sprintf("value-%d", index+1)),
				}}},
			Amounts: []*api.Amount{
				{
					MaxAmount: utils.NewUInt32Value(uint32(index)),
					ValidDuration: &duration.Duration{
						Seconds: int64(index),
						Nanos:   int32(index),
					},
				},
			},
			Action:  utils.NewStringValue("REJECT"),
			Disable: utils.NewBoolValue(true),
			Adjuster: &api.AmountAdjuster{
				Climb: &api.ClimbConfig{
					Enable: utils.NewBoolValue(true),
					Metric: &api.ClimbConfig_MetricConfig{
						Window: &duration.Duration{
							Seconds: int64(index),
							Nanos:   int32(index),
						},
						Precision: utils.NewUInt32Value(uint32(index)),
						ReportInterval: &duration.Duration{
							Seconds: int64(index),
							Nanos:   int32(index),
						},
					},
				},
			},
			RegexCombine: utils.NewBoolValue(true),
			AmountMode:   api.Rule_SHARE_EQUALLY,
			Failover:     api.Rule_FAILOVER_PASS,
		}
		rateLimits = append(rateLimits, rateLimit)
	}
	return rateLimits
}

/**
 * @brief 更新测试限流规则
 */
func UpdateRateLimits(rateLimits []*api.Rule) {
	for _, rateLimit := range rateLimits {
		rateLimit.Arguments = []*api.MatchArgument{
			{
				Type: api.MatchArgument_CUSTOM,
				Key:  "key1",
				Value: &api.MatchString{
					Type:  api.MatchString_REGEX,
					Value: utils.NewStringValue("value-1"),
				},
			},
		}
	}
}
