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
	apifault "github.com/polarismesh/specification/source/go/api/v1/fault_tolerance"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	"github.com/polarismesh/polaris/common/utils"
)

const (
	circuitBreakerName      = "test-name-%v"
	circuitBreakerNamespace = "test-namespace-%v"
)

/**
 * @brief 创建测试熔断规则
 */
func CreateCircuitBreakers(namespace *apimodel.Namespace) []*apifault.CircuitBreaker {
	var circuitBreakers []*apifault.CircuitBreaker
	for index := 0; index < 2; index++ {
		circuitBreaker := &apifault.CircuitBreaker{
			Name:       utils.NewStringValue(fmt.Sprintf(circuitBreakerName, index)),
			Namespace:  namespace.GetName(),
			Business:   utils.NewStringValue("test"),
			Department: utils.NewStringValue("test"),
			Owners:     utils.NewStringValue("test"),
			Comment:    utils.NewStringValue("test"),
		}
		ruleNum := 2
		// 填充source规则
		sources := make([]*apifault.SourceMatcher, 0, ruleNum)
		for i := 0; i < ruleNum; i++ {
			source := &apifault.SourceMatcher{
				Service:   utils.NewStringValue(fmt.Sprintf("service-test-%d", i)),
				Namespace: utils.NewStringValue(fmt.Sprintf("namespace-test-%d", i)),
				Labels: map[string]*apimodel.MatchString{
					fmt.Sprintf("name-%d", i): {
						Type:  apimodel.MatchString_EXACT,
						Value: utils.NewStringValue(fmt.Sprintf("value-%d", i)),
					},
					fmt.Sprintf("name-%d", i+1): {
						Type:  apimodel.MatchString_REGEX,
						Value: utils.NewStringValue(fmt.Sprintf("value-%d", i+1)),
					},
				},
			}
			sources = append(sources, source)
		}

		// 填充destination规则
		destinations := make([]*apifault.DestinationSet, 0, ruleNum)
		for i := 0; i < ruleNum; i++ {
			destination := &apifault.DestinationSet{
				Service:   utils.NewStringValue(fmt.Sprintf("service-test-%d", i)),
				Namespace: utils.NewStringValue(fmt.Sprintf("namespace-test-%d", i)),
				Metadata: map[string]*apimodel.MatchString{
					fmt.Sprintf("name-%d", i): {
						Type:  apimodel.MatchString_EXACT,
						Value: utils.NewStringValue(fmt.Sprintf("value-%d", i)),
					},
					fmt.Sprintf("name-%d", i+1): {
						Type:  apimodel.MatchString_REGEX,
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
		inbounds := make([]*apifault.CbRule, 0, ruleNum)
		for i := 0; i < ruleNum; i++ {
			inbound := &apifault.CbRule{
				Sources:      sources,
				Destinations: destinations,
			}
			inbounds = append(inbounds, inbound)
		}
		// 填充outbound规则
		outbounds := make([]*apifault.CbRule, 0, ruleNum)
		for i := 0; i < ruleNum; i++ {
			outbound := &apifault.CbRule{
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
func UpdateCircuitBreakers(circuitBreakers []*apifault.CircuitBreaker) {
	for _, item := range circuitBreakers {
		item.Inbounds = []*apifault.CbRule{
			{
				Sources: []*apifault.SourceMatcher{
					{
						Service: &wrappers.StringValue{
							Value: "testSvc",
						},
					},
				},
			},
		}
		item.Outbounds = []*apifault.CbRule{
			{
				Sources: []*apifault.SourceMatcher{
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
func CreateCircuitBreakerVersions(circuitBreakers []*apifault.CircuitBreaker) []*apifault.CircuitBreaker {
	var newCircuitBreakers []*apifault.CircuitBreaker

	for index, item := range circuitBreakers {
		newCircuitBreaker := &apifault.CircuitBreaker{
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
func CreateConfigRelease(services []*apiservice.Service, circuitBreakers []*apifault.CircuitBreaker) []*apiservice.ConfigRelease {
	var configReleases []*apiservice.ConfigRelease
	for index := 0; index < 2; index++ {
		configRelease := &apiservice.ConfigRelease{
			Service:        services[index],
			CircuitBreaker: circuitBreakers[index],
		}
		configReleases = append(configReleases, configRelease)
	}
	return configReleases
}
