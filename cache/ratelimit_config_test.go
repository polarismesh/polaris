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
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store/mock"
)

/**
 * @brief 创建一个测试mock rateLimitCache
 */
func newTestRateLimitCache(t *testing.T) (*gomock.Controller, *mock.MockStore, *rateLimitCache) {
	ctl := gomock.NewController(t)

	storage := mock.NewMockStore(ctl)
	storage.EXPECT().GetUnixSecond().AnyTimes().Return(time.Now().Unix(), nil)
	rlc := newRateLimitCache(storage)
	var opt map[string]interface{}
	_ = rlc.initialize(opt)
	return ctl, storage, rlc
}

/**
 * @brief 生成限流规则测试数据
 */
func genModelRateLimits(beginNum, totalServices, totalRateLimits int) ([]*model.RateLimit, []*model.RateLimitRevision) {
	rateLimits := make([]*model.RateLimit, 0, totalRateLimits)
	revisions := make([]*model.RateLimitRevision, 0, totalServices)
	rulePerService := totalRateLimits / totalServices

	for i := beginNum; i < totalServices+beginNum; i++ {
		revision := &model.RateLimitRevision{
			ServiceID:    fmt.Sprintf("service-%d", i),
			LastRevision: fmt.Sprintf("last-revision-%d", i),
		}
		revisions = append(revisions, revision)
		for j := 0; j < rulePerService; j++ {
			rateLimit := &model.RateLimit{
				ID:        fmt.Sprintf("id-%d-%d", i, j),
				ServiceID: fmt.Sprintf("service-%d", i),
				ClusterID: fmt.Sprintf("cluster-%d", j),
				Rule:      fmt.Sprintf("rule-%d-%d", i, j),
				Revision:  fmt.Sprintf("revision-%d-%d", i, j),
				Valid:     true,
			}
			rateLimits = append(rateLimits, rateLimit)
		}
	}
	return rateLimits, revisions
}

/**
 * @brief 统计缓存中的限流数据
 */
func getRateLimitsCount(serviceID string, rlc *rateLimitCache) int {
	rateLimitsCount := 0
	rateLimitIterProc := func(key string, value *model.RateLimit) (bool, error) {
		rateLimitsCount++
		return true, nil
	}
	_ = rlc.GetRateLimit(serviceID, rateLimitIterProc)
	return rateLimitsCount
}

/**
 * TestRateLimitUpdate 测试更新缓存操作
 */
func TestRateLimitUpdate(t *testing.T) {
	ctl, storage, rlc := newTestRateLimitCache(t)
	defer ctl.Finish()

	totalServices := 5
	totalRateLimits := 15
	rateLimits, revisions := genModelRateLimits(0, totalServices, totalRateLimits)

	t.Run("正常更新缓存，可以获取到数据", func(t *testing.T) {
		_ = rlc.clear()

		storage.EXPECT().GetRateLimitsForCache(gomock.Any(), rlc.firstUpdate).
			Return(rateLimits, revisions, nil)
		if err := rlc.update(0); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		// 检查数目是否一致
		for i := 0; i < totalServices; i++ {
			count := getRateLimitsCount(fmt.Sprintf("service-%d", i), rlc)
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
			t.Fatalf("actual count is %d", count)
		}

		count = rlc.GetRevisionsCount()
		if count == totalServices {
			t.Log("pass")
		} else {
			t.Fatalf("actual count is %d", count)
		}
	})

	t.Run("缓存数据为空", func(t *testing.T) {
		_ = rlc.clear()

		storage.EXPECT().GetRateLimitsForCache(gomock.Any(), rlc.firstUpdate).
			Return(nil, nil, nil)
		if err := rlc.update(0); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		if rlc.GetRateLimitsCount() == 0 && rlc.GetRevisionsCount() == 0 {
			t.Log("pass")
		} else {
			t.Fatalf("actual rate limits count is %d, revisions count is %d",
				rlc.GetRateLimitsCount(), rlc.GetRevisionsCount())
		}
	})

	t.Run("lastMtime正确更新", func(t *testing.T) {
		_ = rlc.clear()

		currentTime := time.Unix(100, 0)
		rateLimits[0].ModifyTime = currentTime
		storage.EXPECT().GetRateLimitsForCache(gomock.Any(), rlc.firstUpdate).
			Return(rateLimits, revisions, nil)
		if err := rlc.update(0); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		if rlc.lastTime.Unix() == currentTime.Unix() {
			t.Log("pass")
		} else {
			t.Fatalf("last mtime error")
		}
	})

	t.Run("数据库返回错误，update错误", func(t *testing.T) {
		storage.EXPECT().GetRateLimitsForCache(gomock.Any(), rlc.firstUpdate).
			Return(nil, nil, fmt.Errorf("stoarge error"))
		if err := rlc.update(0); err != nil {
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
		_ = rlc.clear()

		rateLimits, revisions := genModelRateLimits(0, totalServices, totalRateLimits)
		storage.EXPECT().GetRateLimitsForCache(gomock.Any(), rlc.firstUpdate).
			Return(rateLimits, revisions, nil)
		if err := rlc.update(0); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		rateLimits, revisions = genModelRateLimits(5, totalServices, totalRateLimits)
		storage.EXPECT().GetRateLimitsForCache(gomock.Any(), rlc.firstUpdate).
			Return(rateLimits, revisions, nil)
		if err := rlc.update(0); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		if rlc.GetRateLimitsCount() == totalRateLimits*2 && rlc.GetRevisionsCount() == totalServices*2 {
			t.Log("pass")
		} else {
			t.Fatalf("actual rate limits count is %d, revisions count is %d", rlc.GetRateLimitsCount(), rlc.GetRevisionsCount())
		}
	})

	t.Run("更新缓存后，删除部分数据，缓存正常更新", func(t *testing.T) {
		_ = rlc.clear()

		rateLimits, revisions := genModelRateLimits(0, totalServices, totalRateLimits)
		storage.EXPECT().GetRateLimitsForCache(gomock.Any(), rlc.firstUpdate).
			Return(rateLimits, revisions, nil)
		if err := rlc.update(0); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		for i := 0; i < totalRateLimits; i += 2 {
			rateLimits[i].Valid = false
		}

		storage.EXPECT().GetRateLimitsForCache(gomock.Any(), rlc.firstUpdate).
			Return(rateLimits, revisions, nil)
		if err := rlc.update(0); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		if rlc.GetRateLimitsCount() == totalRateLimits/2 && rlc.GetRevisionsCount() == totalServices {
			t.Log("pass")
		} else {
			t.Fatalf("actual rate limits count is %d, revisions count is %d",
				rlc.GetRateLimitsCount(), rlc.GetRevisionsCount())
		}
	})
}

/**
 * TestGetRateLimitsByServiceID 根据服务id获取限流数据和revision
 */
func TestGetRateLimitsByServiceID(t *testing.T) {
	ctl, storage, rlc := newTestRateLimitCache(t)
	defer ctl.Finish()

	t.Run("通过服务ID获取数据和revision", func(t *testing.T) {
		_ = rlc.clear()

		totalServices := 5
		totalRateLimits := 15
		rateLimits, revisions := genModelRateLimits(0, totalServices, totalRateLimits)

		storage.EXPECT().GetRateLimitsForCache(gomock.Any(), rlc.firstUpdate).
			Return(rateLimits, revisions, nil)
		if err := rlc.update(0); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		rateLimits = rlc.GetRateLimitByServiceID("service-1")
		if len(rateLimits) == totalRateLimits/totalServices {
			t.Log("pass")
		} else {
			t.Fatalf("expect num is %d, actual num is %d", totalRateLimits/totalServices, len(rateLimits))
		}
		lastRevision := rlc.GetLastRevision("service-1")
		if lastRevision == "last-revision-1" {
			t.Log("pass")
		} else {
			t.Fatalf("actual last revision is %s", lastRevision)
		}

		rateLimits = rlc.GetRateLimitByServiceID("service-11")
		if len(rateLimits) == 0 {
			t.Log("pass")
		} else {
			t.Fatalf("expect num is 0, actual num is %d", len(rateLimits))
		}

		lastRevision = rlc.GetLastRevision("service-11")
		if lastRevision == "" {
			t.Log("pass")
		} else {
			t.Fatalf("actual last revision is %s", lastRevision)
		}
	})
}
