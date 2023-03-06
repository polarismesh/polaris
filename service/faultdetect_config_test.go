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

package service_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/polarismesh/polaris/service"
	apifault "github.com/polarismesh/specification/source/go/api/v1/fault_tolerance"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"github.com/stretchr/testify/assert"
)

func buildUnnamedFaultDetectRule() *apifault.FaultDetectRule {
	return &apifault.FaultDetectRule{
		Namespace:   service.DefaultNamespace,
		Description: "comment me",
		TargetService: &apifault.FaultDetectRule_DestinationService{
			Service:   "testDestService",
			Namespace: "test",
			Method:    &apimodel.MatchString{Type: apimodel.MatchString_IN, Value: &wrappers.StringValue{Value: "/foo"}},
		},
		Interval: 60,
		Timeout:  60,
		Port:     8888,
		Protocol: apifault.FaultDetectRule_HTTP,
	}
}

func buildFaultDetectRule(index int) *apifault.FaultDetectRule {
	return &apifault.FaultDetectRule{
		Name:        fmt.Sprintf("test-faultdetect-rule-%d", index),
		Namespace:   service.DefaultNamespace,
		Description: "comment me",
		TargetService: &apifault.FaultDetectRule_DestinationService{
			Service:   "testDestService",
			Namespace: "test",
			Method:    &apimodel.MatchString{Type: apimodel.MatchString_IN, Value: &wrappers.StringValue{Value: "/foo"}},
		},
		Interval: 60,
		Timeout:  60,
		Port:     8888,
		Protocol: apifault.FaultDetectRule_HTTP,
		HttpConfig: &apifault.HttpProtocolConfig{
			Method: "POST",
			Url:    "/health",
			Headers: []*apifault.HttpProtocolConfig_MessageHeader{
				{
					Key:   "Content-Type",
					Value: "application/json",
				},
			},
			Body: "<html>test</html>",
		},
		TcpConfig: &apifault.TcpProtocolConfig{Send: "0x1111", Receive: []string{"0x2223", "0x8981"}},
		UdpConfig: &apifault.UdpProtocolConfig{Send: "0x1111", Receive: []string{"0x2223", "0x8981"}},
	}
}

func createFaultDetectRules(discoverSuit *DiscoverTestSuit, count int) ([]*apifault.FaultDetectRule, *apiservice.BatchWriteResponse) {
	fdRules := make([]*apifault.FaultDetectRule, 0, count)
	for i := 0; i < count; i++ {
		fbRule := buildFaultDetectRule(i)
		fdRules = append(fdRules, fbRule)
	}
	resp := discoverSuit.DiscoverServer().CreateFaultDetectRules(discoverSuit.DefaultCtx, fdRules)
	return fdRules, resp
}

func cleanFaultDetectRules(discoverSuit *DiscoverTestSuit, response *apiservice.BatchWriteResponse) {
	fdRules := parseResponseToFaultDetectRules(response)
	if len(fdRules) > 0 {
		discoverSuit.DiscoverServer().DeleteFaultDetectRules(discoverSuit.DefaultCtx, fdRules)
	}
}

func checkFaultDetectRuleResponse(t *testing.T, requests []*apifault.FaultDetectRule, response *apiservice.BatchWriteResponse) {
	assertions := assert.New(t)
	assertions.Equal(len(requests), len(response.Responses))
	for _, resp := range response.Responses {
		assertions.Equal(uint32(apimodel.Code_ExecuteSuccess), resp.GetCode().GetValue())
		msg := &apifault.FaultDetectRule{}
		err := ptypes.UnmarshalAny(resp.GetData(), msg)
		assertions.Nil(err)
		assertions.True(len(msg.GetId()) > 0)
	}
}

func parseResponseToFaultDetectRules(response *apiservice.BatchWriteResponse) []*apifault.FaultDetectRule {
	fdRules := make([]*apifault.FaultDetectRule, 0, len(response.GetResponses()))
	for _, resp := range response.GetResponses() {
		if resp.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
			continue
		}
		msg := &apifault.FaultDetectRule{}
		_ = ptypes.UnmarshalAny(resp.GetData(), msg)
		fdRules = append(fdRules, msg)
	}
	return fdRules
}

// TestCreateFaultDetectRule test create faultdetect rule
func TestCreateFaultDetectRule(t *testing.T) {
	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	t.Run("正常创建探测规则，返回成功", func(t *testing.T) {
		fdRules, resp := createFaultDetectRules(discoverSuit, testCount)
		defer cleanFaultDetectRules(discoverSuit, resp)
		checkFaultDetectRuleResponse(t, fdRules, resp)
	})

	t.Run("重复创建探测规则，返回错误", func(t *testing.T) {
		fdRules, resp := createFaultDetectRules(discoverSuit, 1)
		defer cleanFaultDetectRules(discoverSuit, resp)
		checkFaultDetectRuleResponse(t, fdRules, resp)

		if resp := discoverSuit.DiscoverServer().CreateFaultDetectRules(discoverSuit.DefaultCtx, fdRules); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error, duplicate rule can not be passed")
		}
	})

	t.Run("创建探测规则，删除，再创建，返回成功", func(t *testing.T) {
		fdRules, resp := createFaultDetectRules(discoverSuit, 1)
		cleanFaultDetectRules(discoverSuit, resp)

		fdRules, resp = createFaultDetectRules(discoverSuit, 1)
		defer cleanFaultDetectRules(discoverSuit, resp)
		checkFaultDetectRuleResponse(t, fdRules, resp)
	})

	t.Run("创建探测规则时，没有传递规则名，返回错误", func(t *testing.T) {
		fdRule := buildUnnamedFaultDetectRule()
		if resp := discoverSuit.DiscoverServer().CreateFaultDetectRules(discoverSuit.DefaultCtx, []*apifault.FaultDetectRule{fdRule}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error, unnamed rule can not be passed")
		}
	})

	t.Run("并发创建探测规则，返回成功", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				fdRule := buildFaultDetectRule(index)
				fdRules := []*apifault.FaultDetectRule{fdRule}
				resp := discoverSuit.DiscoverServer().CreateFaultDetectRules(discoverSuit.DefaultCtx, fdRules)
				cleanFaultDetectRules(discoverSuit, resp)
			}(i)
		}
		wg.Wait()
	})
}
