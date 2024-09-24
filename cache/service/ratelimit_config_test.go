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
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/duration"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	"github.com/stretchr/testify/assert"

	types "github.com/polarismesh/polaris/cache/api"
	cachemock "github.com/polarismesh/polaris/cache/mock"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store/mock"
)

/**
 * @brief 创建一个测试mock rateLimitCache
 */
func newTestRateLimitCache(t *testing.T) (*gomock.Controller, *mock.MockStore, *rateLimitCache) {
	ctl := gomock.NewController(t)

	storage := mock.NewMockStore(ctl)
	mockCacheMgr := cachemock.NewMockCacheManager(ctl)

	mockSvcCache := NewServiceCache(storage, mockCacheMgr)
	mockInstCache := NewInstanceCache(storage, mockCacheMgr)
	mockRateLimitCache := NewRateLimitCache(storage, mockCacheMgr)

	mockCacheMgr.EXPECT().GetCacher(types.CacheService).Return(mockSvcCache).AnyTimes()
	mockCacheMgr.EXPECT().GetCacher(types.CacheInstance).Return(mockInstCache).AnyTimes()
	mockCacheMgr.EXPECT().GetReportInterval().Return(time.Second).AnyTimes()
	mockCacheMgr.EXPECT().GetUpdateCacheInterval().Return(time.Second).AnyTimes()

	storage.EXPECT().GetUnixSecond(gomock.Any()).AnyTimes().Return(time.Now().Unix(), nil)
	var opt map[string]interface{}
	_ = mockRateLimitCache.Initialize(opt)
	_ = mockSvcCache.Initialize(opt)
	_ = mockInstCache.Initialize(opt)
	return ctl, storage, mockRateLimitCache.(*rateLimitCache)
}

func buildRateLimitRuleProtoWithLabels(name string, method string) *apitraffic.Rule {
	rule := &apitraffic.Rule{
		Priority: utils.NewUInt32Value(0),
		Resource: apitraffic.Rule_QPS,
		Type:     apitraffic.Rule_LOCAL,
		Labels: map[string]*apimodel.MatchString{"http.method": {
			Type:  apimodel.MatchString_EXACT,
			Value: utils.NewStringValue("post"),
		}},
		Amounts: []*apitraffic.Amount{{
			MaxAmount:     utils.NewUInt32Value(100),
			ValidDuration: &duration.Duration{Seconds: 1},
		}},
		Action:       utils.NewStringValue("reject"),
		Disable:      utils.NewBoolValue(false),
		RegexCombine: utils.NewBoolValue(false),
		Failover:     apitraffic.Rule_FAILOVER_LOCAL,
		Method: &apimodel.MatchString{
			Type:  apimodel.MatchString_EXACT,
			Value: utils.NewStringValue(method),
		},
		Name: utils.NewStringValue(name),
	}
	return rule
}

func buildRateLimitRuleProtoWithArguments(name string, method string) *apitraffic.Rule {
	rule := &apitraffic.Rule{
		Priority: utils.NewUInt32Value(0),
		Resource: apitraffic.Rule_QPS,
		Type:     apitraffic.Rule_LOCAL,
		Arguments: []*apitraffic.MatchArgument{
			{
				Type: apitraffic.MatchArgument_HEADER,
				Key:  "host",
				Value: &apimodel.MatchString{
					Type:  apimodel.MatchString_EXACT,
					Value: utils.NewStringValue("localhost"),
				},
			},
		},
		Amounts: []*apitraffic.Amount{{
			MaxAmount:     utils.NewUInt32Value(100),
			ValidDuration: &duration.Duration{Seconds: 1},
		}},
		Action:       utils.NewStringValue("reject"),
		Disable:      utils.NewBoolValue(false),
		RegexCombine: utils.NewBoolValue(false),
		Failover:     apitraffic.Rule_FAILOVER_LOCAL,
		Method: &apimodel.MatchString{
			Type:  apimodel.MatchString_EXACT,
			Value: utils.NewStringValue(method),
		},
		Name: utils.NewStringValue(name),
	}
	return rule
}

// genRateLimitsWithLabels 生成限流规则测试数据
func genRateLimits(
	beginNum, totalServices, totalRateLimits int, withLabels bool) []*model.RateLimit {
	rateLimits := make([]*model.RateLimit, 0, totalRateLimits)
	rulePerService := totalRateLimits / totalServices

	for i := beginNum; i < totalServices+beginNum; i++ {
		for j := 0; j < rulePerService; j++ {
			name := fmt.Sprintf("limit-rule-%d-%d", i, j)
			method := fmt.Sprintf("/test-%d", j)
			var rule *apitraffic.Rule
			if withLabels {
				rule = buildRateLimitRuleProtoWithLabels(name, method)
			} else {
				rule = buildRateLimitRuleProtoWithArguments(name, method)
			}
			rule.Service = utils.NewStringValue(fmt.Sprintf("service-%d", i))
			rule.Namespace = utils.NewStringValue("default")
			str, _ := json.Marshal(rule)
			labels, _ := json.Marshal(rule.GetLabels())
			rateLimit := &model.RateLimit{
				ID:        fmt.Sprintf("id-%d-%d", i, j),
				ServiceID: fmt.Sprintf("service-%d", i),
				Name:      name,
				Method:    method,
				Rule:      string(str),
				Revision:  fmt.Sprintf("revision-%d-%d", i, j),
				Labels:    string(labels),
				Valid:     true,
			}
			rateLimits = append(rateLimits, rateLimit)
		}
	}
	return rateLimits
}

/**
 * @brief 统计缓存中的限流数据
 */
func getRateLimitsCount(serviceKey model.ServiceKey, rlc *rateLimitCache) int {
	ret, _ := rlc.GetRateLimitRules(serviceKey)
	return len(ret)
}

/**
 * TestRateLimitUpdate 测试更新缓存操作
 */
func TestRateLimitUpdate(t *testing.T) {
	ctl, storage, rlc := newTestRateLimitCache(t)
	defer ctl.Finish()

	totalServices := 5
	totalRateLimits := 15
	rateLimits := genRateLimits(0, totalServices, totalRateLimits, false)

	t.Run("正常更新缓存，可以获取到数据", func(t *testing.T) {
		_ = rlc.Clear()
		storage.EXPECT().GetRateLimitsForCache(gomock.Any(), rlc.IsFirstUpdate()).Return(rateLimits, nil)
		if err := rlc.Update(); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		// 检查数目是否一致
		for i := 0; i < totalServices; i++ {
			count := getRateLimitsCount(model.ServiceKey{
				Namespace: "default",
				Name:      fmt.Sprintf("service-%d", i),
			}, rlc)
			if count == totalRateLimits/totalServices {
				t.Log("pass")
			} else {
				t.Fatalf("actual count is %d", count)
			}
		}

		count := rlc.GetRateLimitsCount()
		if count == totalRateLimits {
			t.Log("pass")
		} else {
			t.Fatalf("actual count is %d, expect : %d", count, len(rateLimits))
		}
	})

	t.Run("缓存数据为空", func(t *testing.T) {
		_ = rlc.Clear()
		storage.EXPECT().GetRateLimitsForCache(gomock.Any(), rlc.IsFirstUpdate()).
			Return(nil, nil)
		if err := rlc.Update(); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		if rlc.GetRateLimitsCount() == 0 {
			t.Log("pass")
		} else {
			t.Fatalf("actual rate limits count is %d",
				rlc.GetRateLimitsCount())
		}
	})

	t.Run("lastMtime正确更新", func(t *testing.T) {
		_ = rlc.Clear()

		currentTime := time.Now()
		rateLimits[0].ModifyTime = currentTime
		storage.EXPECT().GetRateLimitsForCache(gomock.Any(), rlc.IsFirstUpdate()).
			Return(rateLimits, nil)
		if err := rlc.Update(); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		if rlc.OriginLastFetchTime().Unix() == currentTime.Unix() {
			t.Log("pass")
		} else {
			t.Fatalf("last mtime error")
		}
	})

	t.Run("数据库返回错误，update错误", func(t *testing.T) {
		storage.EXPECT().GetRateLimitsForCache(gomock.Any(), rlc.IsFirstUpdate()).
			Return(nil, fmt.Errorf("stoarge error"))
		if err := rlc.Update(); err != nil {
			t.Log("pass")
		} else {
			t.Fatalf("error")
		}
	})
}

/**
 * TestRateLimitUpdate2 统计缓存中的限流数据
 */
func TestRateLimitUpdate2(t *testing.T) {
	ctl, storage, rlc := newTestRateLimitCache(t)
	defer ctl.Finish()

	totalServices := 5
	totalRateLimits := 15

	t.Run("更新缓存后，增加部分数据，缓存正常更新", func(t *testing.T) {
		_ = rlc.Clear()

		rateLimits := genRateLimits(0, totalServices, totalRateLimits, true)
		storage.EXPECT().GetRateLimitsForCache(gomock.Any(), rlc.IsFirstUpdate()).
			Return(rateLimits, nil)
		if err := rlc.Update(); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		rateLimits = genRateLimits(5, totalServices, totalRateLimits, true)
		storage.EXPECT().GetRateLimitsForCache(gomock.Any(), rlc.IsFirstUpdate()).
			Return(rateLimits, nil)
		if err := rlc.Update(); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		if rlc.GetRateLimitsCount() == totalRateLimits*2 {
			t.Log("pass")
		} else {
			t.Fatalf("actual rate limits count is %d", rlc.GetRateLimitsCount())
		}
	})

	t.Run("更新缓存后，删除部分数据，缓存正常更新", func(t *testing.T) {
		_ = rlc.Clear()

		rateLimits := genRateLimits(0, totalServices, totalRateLimits, true)
		storage.EXPECT().GetRateLimitsForCache(gomock.Any(), rlc.IsFirstUpdate()).
			Return(rateLimits, nil)
		if err := rlc.Update(); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		for i := 0; i < totalRateLimits; i += 2 {
			rateLimits[i].Valid = false
		}

		storage.EXPECT().GetRateLimitsForCache(gomock.Any(), rlc.IsFirstUpdate()).
			Return(rateLimits, nil)
		if err := rlc.Update(); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		if rlc.GetRateLimitsCount() == totalRateLimits/2 {
			t.Log("pass")
		} else {
			t.Fatalf("actual rate limits count is %d",
				rlc.GetRateLimitsCount())
		}
	})
}

/**
 * TestGetRateLimitsByServiceID 根据服务id获取限流数据和revision
 */
func TestGetRateLimitsByServiceID(t *testing.T) {
	ctl, storage, rlc := newTestRateLimitCache(t)
	defer ctl.Finish()

	t.Run("通过服务ID获取数据并检查labels", func(t *testing.T) {
		_ = rlc.Clear()

		totalServices := 5
		totalRateLimits := 15
		rateLimits := genRateLimits(0, totalServices, totalRateLimits, true)

		storage.EXPECT().GetRateLimitsForCache(gomock.Any(), rlc.IsFirstUpdate()).
			Return(rateLimits, nil)
		if err := rlc.Update(); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		rules, _ := rlc.GetRateLimitRules(model.ServiceKey{
			Namespace: "default",
			Name:      "service-1",
		})
		if len(rules) == totalRateLimits/totalServices {
			t.Log("pass")
		} else {
			t.Fatalf("expect num is %d, actual num is %d", totalRateLimits/totalServices, len(rateLimits))
		}

		for _, rateLimit := range rules {
			assert.Equal(t, 1, len(rateLimit.Proto.Labels))
			assert.Equal(t, 1, len(rateLimit.Proto.Arguments))
			for _, argument := range rateLimit.Proto.Arguments {
				assert.Equal(t, apitraffic.MatchArgument_CUSTOM, argument.Type)
				_, hasKey := rateLimit.Proto.Labels[argument.Key]
				assert.True(t, hasKey)
			}
		}
	})

	t.Run("通过服务ID获取数据并检查argument", func(t *testing.T) {
		_ = rlc.Clear()

		totalServices := 5
		totalRateLimits := 15
		rateLimits := genRateLimits(0, totalServices, totalRateLimits, false)

		storage.EXPECT().GetRateLimitsForCache(gomock.Any(), rlc.IsFirstUpdate()).
			Return(rateLimits, nil)
		if err := rlc.Update(); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		rateLimits, _ = rlc.GetRateLimitRules(model.ServiceKey{
			Namespace: "default",
			Name:      "service-1",
		})
		if len(rateLimits) == totalRateLimits/totalServices {
			t.Log("pass")
		} else {
			t.Fatalf("expect num is %d, actual num is %d", totalRateLimits/totalServices, len(rateLimits))
		}
		for _, rateLimit := range rateLimits {
			assert.Equal(t, 1, len(rateLimit.Proto.Arguments))
			assert.Equal(t, 1, len(rateLimit.Proto.Labels))
			labelValue, hasKey := rateLimit.Proto.Labels["$header.host"]
			assert.True(t, hasKey)
			assert.Equal(t, rateLimit.Proto.Arguments[0].Value.Value.GetValue(), labelValue.GetValue().GetValue())
		}
	})
}

func Test_QueryRateLimitRules(t *testing.T) {
	ctl, storage, rlc := newTestRateLimitCache(t)
	t.Cleanup(func() {
		ctl.Finish()
	})

	totalServices := 5
	totalRateLimits := 15
	rateLimits := genRateLimits(0, totalServices, totalRateLimits, true)

	storage.EXPECT().GetRateLimitsForCache(gomock.Any(), gomock.Any()).AnyTimes().
		Return(rateLimits, nil)
	if err := rlc.Update(); err != nil {
		t.Fatalf("error: %s", err.Error())
	}

	t.Run("根据ID进行查询", func(t *testing.T) {
		total, ret, err := rlc.QueryRateLimitRules(context.TODO(), types.RateLimitRuleArgs{
			ID:     rateLimits[0].ID,
			Offset: 0,
			Limit:  100,
		})

		assert.NoError(t, err)
		assert.Equal(t, int64(1), int64(total))
		assert.Equal(t, int64(1), int64(len(ret)))
		assert.Equal(t, rateLimits[0].ID, ret[0].ID)
	})

	t.Run("根据Name进行查询", func(t *testing.T) {
		total, ret, err := rlc.QueryRateLimitRules(context.TODO(), types.RateLimitRuleArgs{
			Name:   rateLimits[0].Name,
			Offset: 0,
			Limit:  100,
		})

		assert.NoError(t, err)
		assert.Equal(t, int64(1), int64(total))
		assert.Equal(t, int64(1), int64(len(ret)))
		assert.Equal(t, rateLimits[0].ID, ret[0].ID)
	})

	t.Run("根据Namespace&Service进行查询", func(t *testing.T) {
		total, ret, err := rlc.QueryRateLimitRules(context.TODO(), types.RateLimitRuleArgs{
			Service:   "service-0",
			Namespace: "default",
			Offset:    0,
			Limit:     100,
		})

		assert.NoError(t, err)
		assert.Equal(t, int64(3), int64(total))
		assert.Equal(t, int64(3), int64(len(ret)))
		for i := range ret {
			assert.Equal(t, "service-0", ret[i].Proto.Service.Value)
			assert.Equal(t, "default", ret[i].Proto.Namespace.Value)
		}
	})

	t.Run("根据分页进行查询", func(t *testing.T) {
		total, ret, err := rlc.QueryRateLimitRules(context.TODO(), types.RateLimitRuleArgs{
			Offset: 10,
			Limit:  5,
		})

		assert.NoError(t, err)
		assert.Equal(t, int64(total), int64(len(rateLimits)))
		assert.Equal(t, int64(5), int64(len(ret)))

		total, ret, err = rlc.QueryRateLimitRules(context.TODO(), types.RateLimitRuleArgs{
			Offset: 100,
			Limit:  5,
		})

		assert.NoError(t, err)
		assert.Equal(t, int64(total), int64(len(rateLimits)))
		assert.Equal(t, int64(0), int64(len(ret)))
	})

	t.Run("根据Disable进行查询", func(t *testing.T) {
		disable := true
		total, ret, err := rlc.QueryRateLimitRules(context.TODO(), types.RateLimitRuleArgs{
			Disable: &disable,
			Offset:  0,
			Limit:   100,
		})

		assert.NoError(t, err)
		assert.Equal(t, int64(0), int64(total))
		assert.Equal(t, int64(0), int64(len(ret)))

		disable = false
		total, ret, err = rlc.QueryRateLimitRules(context.TODO(), types.RateLimitRuleArgs{
			Disable: &disable,
			Offset:  0,
			Limit:   100,
		})

		assert.NoError(t, err)
		assert.Equal(t, int64(total), int64(len(rateLimits)))
		assert.Equal(t, int64(total), int64(len(ret)))
		for i := range ret {
			assert.Equal(t, disable, ret[i].Disable)
		}
	})

}
