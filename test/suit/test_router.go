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

package testsuit

import (
	"fmt"
	"testing"

	"github.com/golang/protobuf/ptypes"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"

	"github.com/polarismesh/polaris/common/utils"
)

func MockRoutingV2(t *testing.T, cnt int32) []*apitraffic.RouteRule {
	rules := make([]*apitraffic.RouteRule, 0, cnt)
	for i := int32(0); i < cnt; i++ {
		matchString := &apimodel.MatchString{
			Type:  apimodel.MatchString_EXACT,
			Value: utils.NewStringValue(fmt.Sprintf("in-meta-value-%d", i)),
		}
		source := &apitraffic.SourceService{
			Service:   fmt.Sprintf("in-source-service-%d", i),
			Namespace: fmt.Sprintf("in-source-service-%d", i),
			Arguments: []*apitraffic.SourceMatch{
				{},
			},
		}
		destination := &apitraffic.DestinationGroup{
			Service:   fmt.Sprintf("in-destination-service-%d", i),
			Namespace: fmt.Sprintf("in-destination-service-%d", i),
			Labels: map[string]*apimodel.MatchString{
				fmt.Sprintf("in-metadata-%d", i): matchString,
			},
			Priority: 120,
			Weight:   100,
			Transfer: "abcdefg",
		}

		entry := &apitraffic.RuleRoutingConfig{
			Sources:      []*apitraffic.SourceService{source},
			Destinations: []*apitraffic.DestinationGroup{destination},
		}

		any, err := ptypes.MarshalAny(entry)
		if err != nil {
			t.Fatal(err)
		}

		item := &apitraffic.RouteRule{
			Id:            "",
			Name:          fmt.Sprintf("test-routing-name-%d", i),
			Namespace:     "",
			Enable:        false,
			RoutingPolicy: apitraffic.RoutingPolicy_RulePolicy,
			RoutingConfig: any,
			Revision:      "",
			Etime:         "",
			Priority:      0,
			Description:   "",
		}

		rules = append(rules, item)
	}

	return rules
}
