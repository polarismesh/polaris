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
	"strconv"
	"sync"
	"testing"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/wrappers"
	apifault "github.com/polarismesh/specification/source/go/api/v1/fault_tolerance"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"github.com/stretchr/testify/assert"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/service"
)

func buildUnnamedCircuitBreakerRule() *apifault.CircuitBreakerRule {
	return &apifault.CircuitBreakerRule{
		Namespace:   service.DefaultNamespace,
		Enable:      true,
		Description: "comment me",
		Level:       apifault.Level_GROUP,
		RuleMatcher: &apifault.RuleMatcher{
			Source: &apifault.RuleMatcher_SourceService{
				Service:   "testSrcService",
				Namespace: "test",
			},
			Destination: &apifault.RuleMatcher_DestinationService{
				Service:   "testDestService",
				Namespace: "test",
				Method:    &apimodel.MatchString{Type: apimodel.MatchString_IN, Value: &wrappers.StringValue{Value: "/foo"}},
			},
		},
	}
}

func buildCircuitBreakerRule(index int) *apifault.CircuitBreakerRule {
	return &apifault.CircuitBreakerRule{
		Name:        fmt.Sprintf("test-circuitbreaker-rule-%d", index),
		Namespace:   service.DefaultNamespace,
		Enable:      true,
		Description: "comment me",
		Level:       apifault.Level_GROUP,
		RuleMatcher: &apifault.RuleMatcher{
			Source: &apifault.RuleMatcher_SourceService{
				Service:   "testSrcService",
				Namespace: "test",
			},
			Destination: &apifault.RuleMatcher_DestinationService{
				Service:   "testDestService",
				Namespace: "test",
				Method:    &apimodel.MatchString{Type: apimodel.MatchString_IN, Value: &wrappers.StringValue{Value: "/foo"}},
			},
		},
		ErrorConditions: []*apifault.ErrorCondition{
			{
				InputType: apifault.ErrorCondition_RET_CODE,
				Condition: &apimodel.MatchString{Type: apimodel.MatchString_IN, Value: &wrappers.StringValue{Value: "400, 500"}},
			},
			{
				InputType: apifault.ErrorCondition_DELAY,
				Condition: &apimodel.MatchString{Type: apimodel.MatchString_EXACT, Value: &wrappers.StringValue{Value: "500"}},
			},
		},
		TriggerCondition: []*apifault.TriggerCondition{
			{
				TriggerType: apifault.TriggerCondition_CONSECUTIVE_ERROR,
				ErrorCount:  10,
			},
			{
				TriggerType:  apifault.TriggerCondition_ERROR_RATE,
				ErrorPercent: 50,
				Interval:     30,
			},
		},
		MaxEjectionPercent: 90,
		RecoverCondition: &apifault.RecoverCondition{
			ConsecutiveSuccess: 3,
			SleepWindow:        60,
		},
		FaultDetectConfig: &apifault.FaultDetectConfig{Enable: true},
		FallbackConfig: &apifault.FallbackConfig{
			Enable: true,
			Response: &apifault.FallbackResponse{
				Code: 500,
				Headers: []*apifault.FallbackResponse_MessageHeader{
					{
						Key:   "x-redirect",
						Value: "test.com",
					},
				},
				Body: "<h>Too many request</h>",
			},
		},
	}
}

const testCount = 5

func createCircuitBreakerRules(discoverSuit *DiscoverTestSuit, count int) ([]*apifault.CircuitBreakerRule, *apiservice.BatchWriteResponse) {
	cbRules := make([]*apifault.CircuitBreakerRule, 0, count)
	for i := 0; i < count; i++ {
		cbRule := buildCircuitBreakerRule(i)
		cbRules = append(cbRules, cbRule)
	}
	resp := discoverSuit.DiscoverServer().CreateCircuitBreakerRules(discoverSuit.DefaultCtx, cbRules)
	return cbRules, resp
}

func queryCircuitBreakerRules(discoverSuit *DiscoverTestSuit, query map[string]string) *apiservice.BatchQueryResponse {
	return discoverSuit.DiscoverServer().GetCircuitBreakerRules(discoverSuit.DefaultCtx, query)
}

func cleanCircuitBreakerRules(discoverSuit *DiscoverTestSuit, response *apiservice.BatchWriteResponse) {
	cbRules := parseResponseToCircuitBreakerRules(response)
	if len(cbRules) == 0 {
		return
	}
	discoverSuit.DiscoverServer().DeleteCircuitBreakerRules(discoverSuit.DefaultCtx, cbRules)
}

func checkCircuitBreakerRuleResponse(t *testing.T, requests []*apifault.CircuitBreakerRule, response *apiservice.BatchWriteResponse) {
	assertions := assert.New(t)
	assertions.Equal(len(requests), len(response.Responses))
	for _, resp := range response.Responses {
		assertions.Equal(uint32(apimodel.Code_ExecuteSuccess), resp.GetCode().GetValue())
		msg := &apifault.CircuitBreakerRule{}
		err := ptypes.UnmarshalAny(resp.GetData(), msg)
		assertions.Nil(err)
		assertions.True(len(msg.GetId()) > 0)
	}
}

func parseResponseToCircuitBreakerRules(response *apiservice.BatchWriteResponse) []*apifault.CircuitBreakerRule {
	cbRules := make([]*apifault.CircuitBreakerRule, 0, len(response.GetResponses()))
	for _, resp := range response.GetResponses() {
		if resp.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
			continue
		}
		msg := &apifault.CircuitBreakerRule{}
		_ = ptypes.UnmarshalAny(resp.GetData(), msg)
		cbRules = append(cbRules, msg)
	}
	return cbRules
}

// TestCreateCircuitBreakerRule test create circuitbreaker rule
func TestCreateCircuitBreakerRule(t *testing.T) {
	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	t.Run("abnormal_scene", func(t *testing.T) {
		t.Run("empty_request", func(t *testing.T) {
			resp := discoverSuit.DiscoverServer().CreateCircuitBreakerRules(discoverSuit.DefaultCtx, []*apifault.CircuitBreakerRule{})
			assert.False(t, api.IsSuccess(resp), resp.GetInfo().GetValue())
			assert.Equal(t, uint32(apimodel.Code_EmptyRequest), resp.GetCode().GetValue())
		})

		t.Run("too_many_request", func(t *testing.T) {
			requests := []*apifault.CircuitBreakerRule{}
			for i := 0; i < utils.MaxBatchSize+10; i++ {
				requests = append(requests, &apifault.CircuitBreakerRule{
					Id: "123123",
				})
			}
			resp := discoverSuit.DiscoverServer().CreateCircuitBreakerRules(discoverSuit.DefaultCtx, requests)
			assert.False(t, api.IsSuccess(resp), resp.GetInfo().GetValue())
			assert.Equal(t, uint32(apimodel.Code_BatchSizeOverLimit), resp.GetCode().GetValue(), resp.GetInfo().GetValue())
		})
	})

	t.Run("正常创建熔断规则，返回成功", func(t *testing.T) {
		cbRules, resp := createCircuitBreakerRules(discoverSuit, testCount)
		defer cleanCircuitBreakerRules(discoverSuit, resp)
		qResp := queryCircuitBreakerRules(discoverSuit, map[string]string{"level": strconv.Itoa(int(apifault.Level_GROUP))})
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), qResp.GetCode().GetValue())
		checkCircuitBreakerRuleResponse(t, cbRules, resp)

	})

	t.Run("重复创建熔断规则，返回错误", func(t *testing.T) {
		cbRules, firstResp := createCircuitBreakerRules(discoverSuit, 1)
		defer cleanCircuitBreakerRules(discoverSuit, firstResp)
		checkCircuitBreakerRuleResponse(t, cbRules, firstResp)

		if resp := discoverSuit.DiscoverServer().CreateCircuitBreakerRules(discoverSuit.DefaultCtx, cbRules); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error, duplicate rule can not be passed")
		}
	})

	t.Run("创建熔断规则，删除，再创建，返回成功", func(t *testing.T) {
		cbRules, firstResp := createCircuitBreakerRules(discoverSuit, 1)
		cleanCircuitBreakerRules(discoverSuit, firstResp)
		cbRules, resp := createCircuitBreakerRules(discoverSuit, 1)
		defer cleanCircuitBreakerRules(discoverSuit, resp)
		checkCircuitBreakerRuleResponse(t, cbRules, resp)
	})

	t.Run("创建熔断规则时，没有传递规则名，返回错误", func(t *testing.T) {
		cbRule := buildUnnamedCircuitBreakerRule()
		if resp := discoverSuit.DiscoverServer().CreateCircuitBreakerRules(discoverSuit.DefaultCtx, []*apifault.CircuitBreakerRule{cbRule}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error, unnamed rule can not be passed")
		}
	})

	t.Run("并发创建熔断规则，返回成功", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				cbRule := buildCircuitBreakerRule(index)
				cbRules := []*apifault.CircuitBreakerRule{cbRule}
				resp := discoverSuit.DiscoverServer().CreateCircuitBreakerRules(discoverSuit.DefaultCtx, cbRules)
				cleanCircuitBreakerRules(discoverSuit, resp)
			}(i)
		}
		wg.Wait()
	})

	t.Run("创建多个熔断规则，并进行查询，返回查询结果", func(t *testing.T) {
		cbRules, firstResp := createCircuitBreakerRules(discoverSuit, 5)
		defer cleanCircuitBreakerRules(discoverSuit, firstResp)
		checkCircuitBreakerRuleResponse(t, cbRules, firstResp)
		batchResp := discoverSuit.DiscoverServer().GetCircuitBreakerRules(
			discoverSuit.DefaultCtx, map[string]string{"name": "test-circuitbreaker-rule"})
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), batchResp.GetCode().GetValue())
		anyValues := batchResp.GetData()
		assert.Equal(t, len(cbRules), len(anyValues))
		batchResp = discoverSuit.DiscoverServer().GetCircuitBreakerRules(
			discoverSuit.DefaultCtx, map[string]string{"name": "test-circuitbreaker-rule", "srcNamespace": "test1"})
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), batchResp.GetCode().GetValue())
		anyValues = batchResp.GetData()
		assert.Equal(t, 0, len(anyValues))
		batchResp = discoverSuit.DiscoverServer().GetCircuitBreakerRules(
			discoverSuit.DefaultCtx, map[string]string{"name": "test-circuitbreaker-rule", "dstService": "test1"})
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), batchResp.GetCode().GetValue())
		anyValues = batchResp.GetData()
		assert.Equal(t, 0, len(anyValues))
	})
}

func TestEnableCircuitBreakerRule(t *testing.T) {
	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	t.Run("正常创建熔断规则，返回成功", func(t *testing.T) {
		cbRules, resp := createCircuitBreakerRules(discoverSuit, testCount)
		defer cleanCircuitBreakerRules(discoverSuit, resp)
		qResp := queryCircuitBreakerRules(discoverSuit, map[string]string{"level": strconv.Itoa(int(apifault.Level_GROUP))})
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), qResp.GetCode().GetValue())
		checkCircuitBreakerRuleResponse(t, cbRules, resp)

		testRule := cbRules[0]
		testRule.Enable = false
		resp = discoverSuit.DiscoverServer().EnableCircuitBreakerRules(discoverSuit.DefaultCtx, []*apifault.CircuitBreakerRule{testRule})
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), resp.GetCode().GetValue())
		qResp = queryCircuitBreakerRules(discoverSuit, map[string]string{"id": testRule.Id})
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), qResp.GetCode().GetValue())

		for _, resp := range qResp.GetData() {
			msg := &apifault.CircuitBreakerRule{}
			err := ptypes.UnmarshalAny(resp, msg)
			assert.Nil(t, err)
			assert.False(t, msg.Enable)
		}
	})
}

func TestUpdateCircuitBreakerRule(t *testing.T) {
	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	t.Run("正常更新熔断规则，返回成功", func(t *testing.T) {
		cbRules, resp := createCircuitBreakerRules(discoverSuit, testCount)
		defer cleanCircuitBreakerRules(discoverSuit, resp)
		qResp := queryCircuitBreakerRules(discoverSuit, map[string]string{"level": strconv.Itoa(int(apifault.Level_GROUP))})
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), qResp.GetCode().GetValue())
		checkCircuitBreakerRuleResponse(t, cbRules, resp)

		mockDescr := "update circuitbreaker rule info"
		testRule := cbRules[0]
		testRule.Description = mockDescr
		resp = discoverSuit.DiscoverServer().UpdateCircuitBreakerRules(discoverSuit.DefaultCtx, []*apifault.CircuitBreakerRule{testRule})
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), resp.GetCode().GetValue())
		qResp = queryCircuitBreakerRules(discoverSuit, map[string]string{"id": testRule.Id})
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), qResp.GetCode().GetValue())

		for _, resp := range qResp.GetData() {
			msg := &apifault.CircuitBreakerRule{}
			err := ptypes.UnmarshalAny(resp, msg)
			assert.Nil(t, err)
			assert.Equal(t, mockDescr, msg.Description)
		}
	})
}
