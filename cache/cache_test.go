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
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"github.com/stretchr/testify/assert"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store/mock"
)

// TestCacheManager_Start 测试cache函数是否正常
func TestCacheManager_Start(t *testing.T) {
	ctl := gomock.NewController(t)
	storage := mock.NewMockStore(ctl)
	storage.EXPECT().GetUnixSecond().AnyTimes().Return(time.Now().Unix(), nil)
	defer ctl.Finish()

	conf := &Config{
		Open: true,
		Resources: []ConfigEntry{
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
		},
	}
	SetCacheConfig(conf)

	t.Run("测试正常的更新缓存逻辑", func(t *testing.T) {
		c, err := TestCacheInitialize(context.Background(), &Config{Open: true}, storage)
		assert.Nil(t, err)
		assert.NotNil(t, c)
		beg := time.Unix(0, 0).Add(DefaultTimeDiff)
		storage.EXPECT().GetUnixSecond().AnyTimes().Return(time.Now().Unix(), nil)
		storage.EXPECT().GetMoreInstances(beg, true, false, nil).Return(nil, nil).MaxTimes(1)
		storage.EXPECT().GetMoreInstances(beg, false, false, nil).Return(nil, nil).MaxTimes(3)
		storage.EXPECT().GetMoreServices(beg, true, false, false).Return(nil, nil).MaxTimes(1)
		storage.EXPECT().GetMoreServices(beg, false, false, false).Return(nil, nil).MaxTimes(3)
		storage.EXPECT().GetRoutingConfigsForCache(beg, true).Return(nil, nil).MaxTimes(3)
		storage.EXPECT().GetRoutingConfigsForCache(beg, false).Return(nil, nil).MaxTimes(3)
		storage.EXPECT().GetMoreL5Routes(uint32(0)).Return(nil, nil).MaxTimes(3)
		storage.EXPECT().GetMoreL5Policies(uint32(0)).Return(nil, nil).MaxTimes(3)
		storage.EXPECT().GetMoreL5Sections(uint32(0)).Return(nil, nil).MaxTimes(3)
		storage.EXPECT().GetMoreL5IPConfigs(uint32(0)).Return(nil, nil).MaxTimes(3)
		storage.EXPECT().GetRateLimitsForCache(beg, true).Return(nil, nil, nil).MaxTimes(1)
		storage.EXPECT().GetRateLimitsForCache(beg, false).Return(nil, nil, nil).MaxTimes(3)
		storage.EXPECT().GetCircuitBreakerForCache(beg, true).Return(nil, nil).MaxTimes(1)
		storage.EXPECT().GetCircuitBreakerForCache(beg, false).Return(nil, nil).MaxTimes(3)
		storage.EXPECT().GetInstancesCount().Return(uint32(0), nil).MaxTimes(1)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err = c.initialize()
		assert.Nil(t, err)

		err = c.Start(ctx)
		assert.Nil(t, err)

		// 等待cache更新
		time.Sleep(c.GetUpdateCacheInterval() + time.Second)
	})

	Convey("测试TestRefresh", t, func() {
		c, err := TestCacheInitialize(context.Background(), &Config{Open: true}, storage)
		So(err, ShouldBeNil)
		So(c, ShouldNotBeNil)

		err = c.TestRefresh()
		So(err, ShouldBeNil)
	})
}

// TestRevisionWorker 测试revision的管道是否正常
func TestRevisionWorker(t *testing.T) {
	ctl := gomock.NewController(t)
	storage := mock.NewMockStore(ctl)
	storage.EXPECT().GetUnixSecond().AnyTimes().Return(time.Now().Unix(), nil)
	defer ctl.Finish()

	t.Run("revision计算，chan可以正常收发", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		nc, err := TestCacheInitialize(ctx, &Config{Open: true}, storage)
		assert.Nil(t, err)
		t.Cleanup(func ()  {
			cancel()
			_ = nc.Clear()
		})
		go nc.revisionWorker(ctx)

		t.Run("revision计算，实例有增加有减少，计算正常", func(t *testing.T) {
			_ = nc.Clear()
			// mock一下cache中服务的数据
			maxTotal := 20480
			services := make(map[string]*model.Service)
			for i := 0; i < maxTotal; i++ {
				item := &model.Service{
					ID:       fmt.Sprintf("service-id-%d", i),
					Revision: fmt.Sprintf("revision-%d", i),
					Valid:    true,
				}
				services[item.ID] = item
			}
			storage.EXPECT().GetServicesCount().Return(uint32(maxTotal), nil).AnyTimes()
			storage.EXPECT().GetMoreServices(gomock.Any(), true, false, false).Return(services, nil)
			// 触发计算
			_ = nc.caches[CacheService].update()
			time.Sleep(time.Second * 10)
			assert.Equal(t, maxTotal, nc.GetServiceRevisionCount())

			services = make(map[string]*model.Service)
			for i := 0; i < maxTotal; i++ {
				if i%2 == 0 {
					item := &model.Service{
						ID:       fmt.Sprintf("service-id-%d", i),
						Revision: fmt.Sprintf("revision-%d", i),
						Valid:    false,
					}
					services[item.ID] = item
				}
			}
			storage.EXPECT().GetServicesCount().Return(uint32(maxTotal), nil).AnyTimes()
			storage.EXPECT().GetMoreServices(gomock.Any(), false, false, false).Return(services, nil)
			// 触发计算
			_ = nc.caches[CacheService].update()
			time.Sleep(time.Second * 20)
			// 检查是否有正常计算
			assert.Equal(t, maxTotal/2, nc.GetServiceRevisionCount())
		})
	})
}

// TestComputeRevision 测试计算revision的函数
func TestComputeRevision(t *testing.T) {
	t.Run("instances为空，可以正常计算", func(t *testing.T) {
		out, err := ComputeRevision("123", nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, out)
	})

	t.Run("instances内容一样，不同顺序，计算出的revision一样", func(t *testing.T) {
		instances := make([]*model.Instance, 0, 6)
		for i := 0; i < 6; i++ {
			instances = append(instances, &model.Instance{
				Proto: &apiservice.Instance{
					Revision: utils.NewStringValue(fmt.Sprintf("revision-%d", i)),
				},
			})
		}

		lhs, err := ComputeRevision("123", nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, lhs)

		// 交换一下数据，数据内容不变，revision应该保证不变
		tmp := instances[0]
		instances[0] = instances[1]
		instances[1] = instances[3]
		instances[3] = tmp

		rhs, err := ComputeRevision("123", nil)
		assert.NoError(t, err)
		assert.Equal(t, lhs, rhs)
	})

	t.Run("serviceRevision发生改变，返回改变", func(t *testing.T) {
		lhs, err := ComputeRevision("123", nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, lhs)

		rhs, err := ComputeRevision("456", nil)
		assert.NoError(t, err)
		assert.NotEqual(t, lhs, rhs)
	})

	t.Run("instances内容改变，返回改变", func(t *testing.T) {
		instance := &model.Instance{Proto: &apiservice.Instance{Revision: utils.NewStringValue("123456")}}
		lhs, err := ComputeRevision("123", []*model.Instance{instance})
		assert.NoError(t, err)
		assert.NotEmpty(t, lhs)

		instance.Proto.Revision.Value = "654321"
		rhs, err := ComputeRevision("456", []*model.Instance{instance})
		assert.NoError(t, err)
		assert.NotEqual(t, lhs, rhs)
	})
}
