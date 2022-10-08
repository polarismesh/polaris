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
	"github.com/golang/protobuf/ptypes/wrappers"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/utils"
)

const (
	circuitBreakerName      = "test-name-%v"
	circuitBreakerNamespace = "test-namespace-%v"
)

/**
 * @brief 创建测试熔断规则
 */
func CreateCircuitBreakers(namespace *api.Namespace) []*api.CircuitBreaker {
	var circuitBreakers []*api.CircuitBreaker
	for index := 0; index < 2; index++ {
		circuitBreaker := &api.CircuitBreaker{
			Name:       utils.NewStringValue(fmt.Sprintf(circuitBreakerName, index)),
			Namespace:  namespace.GetName(),
			Business:   utils.NewStringValue("test"),
			Department: utils.NewStringValue("test"),
			Owners:     utils.NewStringValue("test"),
			Comment:    utils.NewStringValue("test"),
		}
		ruleNum := 2
		// 填充source规则
		sources := make([]*api.SourceMatcher, 0, ruleNum)
		for i := 0; i < ruleNum; i++ {
			source := &api.SourceMatcher{
				Service:   utils.NewStringValue(fmt.Sprintf("service-test-%d", i)),
				Namespace: utils.NewStringValue(fmt.Sprintf("namespace-test-%d", i)),
				Labels: map[string]*api.MatchString{
					fmt.Sprintf("name-%d", i): {
						Type:  api.MatchString_EXACT,
						Value: utils.NewStringValue(fmt.Sprintf("value-%d", i)),
					},
					fmt.Sprintf("name-%d", i+1): {
						Type:  api.MatchString_REGEX,
						Value: utils.NewStringValue(fmt.Sprintf("value-%d", i+1)),
					},
				},
			}
			sources = append(sources, source)
		}

		// 填充destination规则
		destinations := make([]*api.DestinationSet, 0, ruleNum)
		for i := 0; i < ruleNum; i++ {
			destination := &api.DestinationSet{
				Service:   utils.NewStringValue(fmt.Sprintf("service-test-%d", i)),
				Namespace: utils.NewStringValue(fmt.Sprintf("namespace-test-%d", i)),
				Metadata: map[string]*api.MatchString{
					fmt.Sprintf("name-%d", i): {
						Type:  api.MatchString_EXACT,
						Value: utils.NewStringValue(fmt.Sprintf("value-%d", i)),
					},
					fmt.Sprintf("name-%d", i+1): {
						Type:  api.MatchString_REGEX,
						Value: utils.NewStringValue(fmt.Sprintf("value-%d", i+1)),
					},
				},
				Resource: 0,
				Type:     0,
				Scope:    0,
				MetricWindow: &duration.Duration{
					Seconds: int64(i),
				},
				MetricPrecision: utils.NewUInt32Value(uint32(i)),
				UpdateInterval: &duration.Duration{
					Seconds: int64(i),
				},
			}
			destinations = append(destinations, destination)
		}

		// 填充inbound规则
		inbounds := make([]*api.CbRule, 0, ruleNum)
		for i := 0; i < ruleNum; i++ {
			inbound := &api.CbRule{
				Sources:      sources,
				Destinations: destinations,
			}
			inbounds = append(inbounds, inbound)
		}
		// 填充outbound规则
		outbounds := make([]*api.CbRule, 0, ruleNum)
		for i := 0; i < ruleNum; i++ {
			outbound := &api.CbRule{
				Sources:      sources,
				Destinations: destinations,
			}
			outbounds = append(outbounds, outbound)
		}
		circuitBreaker.Inbounds = inbounds
		circuitBreaker.Outbounds = outbounds
		circuitBreakers = append(circuitBreakers, circuitBreaker)
	}
	return circuitBreakers
}

/**
 * @brief 更新测试熔断规则
 */
func UpdateCircuitBreakers(circuitBreakers []*api.CircuitBreaker) {
	for _, item := range circuitBreakers {
		item.Inbounds = []*api.CbRule{
			{
				Sources: []*api.SourceMatcher{
					{
						Service: &wrappers.StringValue{
							Value: "testSvc",
						},
					},
				},
			},
		}
		item.Outbounds = []*api.CbRule{
			{
				Sources: []*api.SourceMatcher{
					{
						Service: &wrappers.StringValue{
							Value: "testSvc1",
						},
					},
				},
			},
		}
	}
}

/**
 * @brief 创建熔断规则版本
 */
func CreateCircuitBreakerVersions(circuitBreakers []*api.CircuitBreaker) []*api.CircuitBreaker {
	var newCircuitBreakers []*api.CircuitBreaker

	for index, item := range circuitBreakers {
		newCircuitBreaker := &api.CircuitBreaker{
			Id:        item.GetId(),
			Name:      item.GetName(),
			Namespace: item.GetNamespace(),
			Token:     item.GetToken(),
			Version:   utils.NewStringValue(fmt.Sprintf("test-version-%d", index)),
		}
		newCircuitBreakers = append(newCircuitBreakers, newCircuitBreaker)
	}

	return newCircuitBreakers
}

/**
 * @brief 创建测试发布熔断规则
 */
func CreateConfigRelease(services []*api.Service, circuitBreakers []*api.CircuitBreaker) []*api.ConfigRelease {
	var configReleases []*api.ConfigRelease
	for index := 0; index < 2; index++ {
		configRelease := &api.ConfigRelease{
			Service:        services[index],
			CircuitBreaker: circuitBreakers[index],
		}
		configReleases = append(configReleases, configRelease)
	}
	return configReleases
}
