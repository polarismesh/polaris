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
	. "github.com/smartystreets/goconvey/convey"

	v1 "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/store/mock"
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

	Convey("测试正常的更新缓存逻辑", t, func() {
		err := TestCacheInitialize(context.Background(), &Config{Open: true}, storage)
		So(err, ShouldBeNil)

		c := cacheMgn
		So(c, ShouldNotBeNil)

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
		So(err, ShouldBeNil)

		err = c.Start(ctx)
		So(err, ShouldBeNil)

		// 等待cache更新
		time.Sleep(c.GetUpdateCacheInterval() + time.Second)
	})
}

// TestRevisionWorker 测试revision的管道是否正常
func TestRevisionWorker(t *testing.T) {
	ctl := gomock.NewController(t)
	storage := mock.NewMockStore(ctl)
	storage.EXPECT().GetUnixSecond().AnyTimes().Return(time.Now().Unix(), nil)
	defer ctl.Finish()

	Convey("revision计算，chan可以正常收发", t, func() {
		err := TestCacheInitialize(context.TODO(), &Config{Open: true}, storage)
		nc := cacheMgn
		defer func() { _ = nc.Clear() }()
		So(err, ShouldBeNil)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go nc.revisionWorker(ctx)

		Convey("revision计算，实例有增加有减少，计算正常", func() {
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
			storage.EXPECT().GetMoreServices(gomock.Any(), true, false, false).Return(services, nil)
			// 触发计算
			_ = nc.caches[CacheService].update(0)
			time.Sleep(time.Second * 10)
			So(nc.GetServiceRevisionCount(), ShouldEqual, maxTotal)

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
			storage.EXPECT().GetMoreServices(gomock.Any(), false, false, false).Return(services, nil)
			// 触发计算
			_ = nc.caches[CacheService].update(0)
			time.Sleep(time.Second * 20)
			// 检查是否有正常计算
			So(nc.GetServiceRevisionCount(), ShouldEqual, maxTotal/2)
		})
	})
}

// TestComputeRevision 测试计算revision的函数
func TestComputeRevision(t *testing.T) {
	Convey("instances为空，可以正常计算", t, func() {
		out, err := ComputeRevision("123", nil)
		So(err, ShouldBeNil)
		So(out, ShouldNotBeEmpty)
	})

	Convey("instances内容一样，不同顺序，计算出的revision一样", t, func() {
		instances := make([]*model.Instance, 0, 6)
		for i := 0; i < 6; i++ {
			instances = append(instances, &model.Instance{
				Proto: &v1.Instance{
					Revision: utils.NewStringValue(fmt.Sprintf("revision-%d", i)),
				},
			})
		}

		lhs, err := ComputeRevision("123", nil)
		So(err, ShouldBeNil)
		So(lhs, ShouldNotBeEmpty)

		// 交换一下数据，数据内容不变，revision应该保证不变
		tmp := instances[0]
		instances[0] = instances[1]
		instances[1] = instances[3]
		instances[3] = tmp

		rhs, err := ComputeRevision("123", nil)
		So(err, ShouldBeNil)
		So(lhs, ShouldEqual, rhs)
	})

	Convey("serviceRevision发生改变，返回改变", t, func() {
		lhs, err := ComputeRevision("123", nil)
		So(err, ShouldBeNil)
		So(lhs, ShouldNotBeEmpty)

		rhs, err := ComputeRevision("456", nil)
		So(err, ShouldBeNil)
		So(lhs, ShouldNotEqual, rhs)
	})

	Convey("instances内容改变，返回改变", t, func() {
		instance := &model.Instance{Proto: &v1.Instance{Revision: utils.NewStringValue("123456")}}
		lhs, err := ComputeRevision("123", []*model.Instance{instance})
		So(err, ShouldBeNil)
		So(lhs, ShouldNotBeEmpty)

		instance.Proto.Revision.Value = "654321"
		rhs, err := ComputeRevision("456", []*model.Instance{instance})
		So(err, ShouldBeNil)
		So(lhs, ShouldNotEqual, rhs)
	})
}
