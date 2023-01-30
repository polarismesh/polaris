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
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"

	"github.com/polarismesh/polaris/common/utils"
)

/**
 * @brief 创建测试限流规则
 */
func CreateRateLimits(services []*apiservice.Service) []*apitraffic.Rule {
	var rateLimits []*apitraffic.Rule
	for index := 0; index < 2; index++ {
		rateLimit := &apitraffic.Rule{
			Name:      utils.NewStringValue(fmt.Sprintf("rlimit-%d", index)),
			Service:   services[index].GetName(),
			Namespace: services[index].GetNamespace(),
			Priority:  utils.NewUInt32Value(uint32(index)),
			Resource:  apitraffic.Rule_CONCURRENCY,
			Type:      apitraffic.Rule_LOCAL,
			Arguments: []*apitraffic.MatchArgument{{
				Type: apitraffic.MatchArgument_CUSTOM,
				Key:  fmt.Sprintf("name-%d", index),
				Value: &apimodel.MatchString{
					Type:  apimodel.MatchString_REGEX,
					Value: utils.NewStringValue(fmt.Sprintf("value-%d", index)),
				},
			}, {Type: apitraffic.MatchArgument_CUSTOM,
				Key: fmt.Sprintf("name-%d", index+1),
				Value: &apimodel.MatchString{
					Type:  apimodel.MatchString_EXACT,
					Value: utils.NewStringValue(fmt.Sprintf("value-%d", index+1)),
				}}},
			Amounts: []*apitraffic.Amount{
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
			Adjuster: &apitraffic.AmountAdjuster{
				Climb: &apitraffic.ClimbConfig{
					Enable: utils.NewBoolValue(true),
					Metric: &apitraffic.ClimbConfig_MetricConfig{
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
			AmountMode:   apitraffic.Rule_SHARE_EQUALLY,
			Failover:     apitraffic.Rule_FAILOVER_PASS,
		}
		rateLimits = append(rateLimits, rateLimit)
	}
	return rateLimits
}

/**
 * @brief 更新测试限流规则
 */
func UpdateRateLimits(rateLimits []*apitraffic.Rule) {
	for _, rateLimit := range rateLimits {
		rateLimit.Arguments = []*apitraffic.MatchArgument{
			{
				Type: apitraffic.MatchArgument_CUSTOM,
				Key:  "key1",
				Value: &apimodel.MatchString{
					Type:  apimodel.MatchString_REGEX,
					Value: utils.NewStringValue("value-1"),
				},
			},
		}
	}
}
