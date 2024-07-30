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

package model

import (
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/polarismesh/polaris/common/utils"
)

func TestParseRouteRuleFromAPI(t *testing.T) {
	ruleRouting := &apitraffic.RuleRoutingConfig{
		Sources: []*apitraffic.SourceService{
			{
				Service:   "testSvc",
				Namespace: "test",
			},
		},
	}
	anyValue, err := anypb.New(proto.MessageV2(ruleRouting))
	assert.Nil(t, err)
	routing := &apitraffic.RouteRule{
		Id: "ssss", RoutingPolicy: apitraffic.RoutingPolicy_RulePolicy, RoutingConfig: anyValue}
	rConfig := &RouterConfig{}
	err = rConfig.ParseRouteRuleFromAPI(routing)
	assert.Nil(t, err)
	assert.True(t, true, strings.HasPrefix(rConfig.Config, "{"))
}

func TestToExpendRoutingConfig(t *testing.T) {
	ruleRouting := &apitraffic.RuleRoutingConfig{
		Sources: []*apitraffic.SourceService{
			{
				Service:   "testSvc",
				Namespace: "test",
			},
		},
	}
	// 1. check json text
	text, err := utils.MarshalToJsonString(ruleRouting)
	assert.Nil(t, err)
	rConfig := &RouterConfig{}
	rConfig.Config = text
	rConfig.Policy = "RulePolicy"
	erConfig, err := rConfig.ToExpendRoutingConfig()
	assert.Nil(t, err)
	assert.Equal(t, ruleRouting.Sources[0].Service, erConfig.RuleRouting.RuleRouting.Sources[0].Service)

	// 2. check v1 binary
	anyValue, err := anypb.New(proto.MessageV2(ruleRouting))
	assert.Nil(t, err)
	v1AnyStr := string(anyValue.GetValue())
	rConfig.Config = v1AnyStr
	erConfig, err = rConfig.ToExpendRoutingConfig()
	assert.Nil(t, err)
	assert.Equal(t, ruleRouting.Sources[0].Service, erConfig.RuleRouting.RuleRouting.Sources[0].Service)

	// 3. check v2 binary
	//ruleRoutingV2 := &v2.RuleRoutingConfig{
	//	Sources: []*v2.Source{
	//		{
	//			Service:   "testSvc",
	//			Namespace: "test",
	//			Arguments: []*v2.SourceMatch{},
	//		},
	//	},
	//}
	//anyValue, err = anypb.New(proto.MessageV2(ruleRoutingV2))
	//assert.Nil(t, err)
	//v2AnyStr := string(anyValue.GetValue())
	//rConfig.Config = v2AnyStr
	//erConfig, err = rConfig.ToExpendRoutingConfig()
	//assert.Nil(t, err)
	//assert.Equal(t, ruleRoutingV2.Sources[0].Service, erConfig.RuleRouting.Sources[0].Service)
	//assert.Equal(t, v1AnyStr, v2AnyStr)
}
