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

package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/polarismesh/polaris/cache"
	types "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/store/mock"
)

// TestCacheManager_Start 测试cache函数是否正常
func TestCacheManager_Start(t *testing.T) {
	ctl := gomock.NewController(t)
	storage := mock.NewMockStore(ctl)
	storage.EXPECT().GetUnixSecond(gomock.Any()).AnyTimes().Return(time.Now().Unix(), nil)
	defer ctl.Finish()

	conf := &cache.Config{}
	entries := []cache.ConfigEntry{
		{
			Name: "service",
		},
		{
			Name: "instance",
		},
		{
			Name: "routingConfig",
		},
		{
			Name: "rateLimitConfig",
		},
		{
			Name: "circuitBreakerConfig",
		},
		{
			Name: "l5",
		},
	}
	cache.SetCacheConfig(conf)

	t.Run("测试正常的更新缓存逻辑", func(t *testing.T) {
		c, err := cache.TestCacheInitialize(context.Background(), &cache.Config{}, storage)
		assert.Nil(t, err)
		assert.NotNil(t, c)
		err = c.OpenResourceCache(entries...)
		assert.NotNil(t, c)
		time.Sleep(time.Second)
		beg := time.Unix(0, 0).Add(types.DefaultTimeDiff)
		storage.EXPECT().GetUnixSecond(gomock.Any()).AnyTimes().Return(time.Now().Unix(), nil)
		storage.EXPECT().GetMoreInstances(gomock.Any(), beg, true, false, nil).Return(nil, nil).MaxTimes(1)
		storage.EXPECT().GetMoreInstances(gomock.Any(), beg, false, false, nil).Return(nil, nil).MaxTimes(3)
		storage.EXPECT().GetMoreServices(beg, true, false, false).Return(nil, nil).MaxTimes(1)
		storage.EXPECT().GetMoreServices(beg, false, false, false).Return(nil, nil).MaxTimes(3)
		storage.EXPECT().GetRoutingConfigsForCache(beg, true).Return(nil, nil).MaxTimes(3)
		storage.EXPECT().GetRoutingConfigsForCache(beg, false).Return(nil, nil).MaxTimes(3)
		storage.EXPECT().GetMoreL5Routes(uint32(0)).Return(nil, nil).MaxTimes(3)
		storage.EXPECT().GetMoreL5Policies(uint32(0)).Return(nil, nil).MaxTimes(3)
		storage.EXPECT().GetMoreL5Sections(uint32(0)).Return(nil, nil).MaxTimes(3)
		storage.EXPECT().GetMoreL5IPConfigs(uint32(0)).Return(nil, nil).MaxTimes(3)
		storage.EXPECT().GetRateLimitsForCache(beg, true).Return(nil, nil).MaxTimes(1)
		storage.EXPECT().GetRateLimitsForCache(beg, false).Return(nil, nil).MaxTimes(3)
		storage.EXPECT().GetCircuitBreakerRulesForCache(beg, false).Return(nil, nil).MaxTimes(3)
		storage.EXPECT().GetInstancesCountTx(gomock.Any()).Return(uint32(0), nil).MaxTimes(1)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err = c.Initialize()
		assert.Nil(t, err)

		err = c.Start(ctx)
		assert.Nil(t, err)

		// 等待cache更新
		time.Sleep(c.GetUpdateCacheInterval() + time.Second)
	})

}
