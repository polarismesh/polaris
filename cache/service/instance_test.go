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
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"github.com/stretchr/testify/assert"

	types "github.com/polarismesh/polaris/cache/api"
	cachemock "github.com/polarismesh/polaris/cache/mock"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
	"github.com/polarismesh/polaris/store/mock"
)

// 创建一个测试mock instanceCache
func newTestInstanceCache(t *testing.T) (*gomock.Controller, *mock.MockStore, *instanceCache) {
	ctl := gomock.NewController(t)

	storage := mock.NewMockStore(ctl)
	mockCacheMgr := cachemock.NewMockCacheManager(ctl)

	mockSvcCache := NewServiceCache(storage, mockCacheMgr)
	mockInstCache := NewInstanceCache(storage, mockCacheMgr)

	mockCacheMgr.EXPECT().GetCacher(types.CacheService).Return(mockSvcCache).AnyTimes()
	mockCacheMgr.EXPECT().GetCacher(types.CacheInstance).Return(mockInstCache).AnyTimes()
	mockCacheMgr.EXPECT().GetReportInterval().Return(time.Second).AnyTimes()
	mockCacheMgr.EXPECT().GetUpdateCacheInterval().Return(time.Second).AnyTimes()

	mockTx := mock.NewMockTx(ctl)
	mockTx.EXPECT().Commit().Return(nil).AnyTimes()
	mockTx.EXPECT().Rollback().Return(nil).AnyTimes()
	mockTx.EXPECT().CreateReadView().Return(nil).AnyTimes()
	storage.EXPECT().StartReadTx().Return(mockTx, nil).AnyTimes()

	storage.EXPECT().GetUnixSecond(gomock.Any()).AnyTimes().Return(time.Now().Unix(), nil)
	opt := map[string]interface{}{
		"disableBusiness": false,
		"needMeta":        true,
	}

	_ = mockSvcCache.Initialize(opt)
	_ = mockInstCache.Initialize(opt)

	return ctl, storage, mockInstCache.(*instanceCache)
}

// 生成测试数据
func genModelInstances(label string, total int) map[string]*model.Instance {
	out := make(map[string]*model.Instance)
	for i := 0; i < total; i++ {
		entry := &model.Instance{
			Proto: &apiservice.Instance{
				Id:   utils.NewStringValue(fmt.Sprintf("instanceID-%s-%d", label, i)),
				Host: utils.NewStringValue(fmt.Sprintf("host-%s-%d", label, i)),
				Port: utils.NewUInt32Value(uint32(i + 10)),
				Location: &apimodel.Location{
					Region: utils.NewStringValue("china"),
					Zone:   utils.NewStringValue("ap-shenzheng"),
					Campus: utils.NewStringValue("ap-shenzheng-1"),
				},
			},
			ServiceID: fmt.Sprintf("serviceID-%s", label),
			Valid:     true,
		}

		out[entry.Proto.Id.GetValue()] = entry
	}

	return out
}

func genModelInstancesConsole(label string, total int) map[string]*model.InstanceConsole {
	out := make(map[string]*model.InstanceConsole)
	for i := 0; i < total; i++ {
		entry := &model.InstanceConsole{
			Id:       fmt.Sprintf("InstanceConsole-%s-%d", label, i),
			Isolate:  false,
			Weight:   100,
			Metadata: "Metadata",
		}
		out[entry.Id] = entry
	}

	return out
}

// 对instanceCache的缓存数据进行计数统计
func iteratorInstances(ic *instanceCache) (int, int) {
	instancesCount := 0
	services := make(map[string]bool)
	_ = ic.IteratorInstances(func(key string, value *model.Instance) (b bool, e error) {
		instancesCount++
		if _, ok := services[value.ServiceID]; !ok {
			services[value.ServiceID] = true
		}
		return true, nil
	})

	return len(services), instancesCount
}

// TestInstanceCache_Update 测试正常的更新缓存操作
func TestInstanceCache_Update(t *testing.T) {
	ctl, storage, ic := newTestInstanceCache(t)
	defer ctl.Finish()
	t.Run("正常更新缓存，缓存数据符合预期", func(t *testing.T) {
		_ = ic.Clear()
		ret := make(map[string]*model.Instance)
		instances1 := genModelInstances("service1", 10) // 每次gen为一个服务的
		instances2 := genModelInstances("service2", 5)
		instanceConsoles := genModelInstancesConsole("console", 3)

		for id, instance := range instances1 {
			ret[id] = instance
		}
		for id, instance := range instances2 {
			ret[id] = instance
		}

		gomock.InOrder(storage.EXPECT().
			GetMoreInstances(gomock.Any(), gomock.Any(), ic.IsFirstUpdate(), ic.needMeta, ic.systemServiceID).
			Return(ret, nil))
		gomock.InOrder(storage.EXPECT().
			GetMoreInstanceConsoles(gomock.Any(), gomock.Any(), ic.IsFirstUpdate(), ic.needMeta, ic.systemServiceID).
			Return(instanceConsoles, nil))
		gomock.InOrder(storage.EXPECT().GetInstancesCountTx(gomock.Any()).Return(uint32(15), nil))
		if err := ic.Update(); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		servicesCount, instancesCount := iteratorInstances(ic)
		instanceConsoleCounts := ic.instanceConsoles.Len()
		if servicesCount == 2 && instancesCount == 10+5 && instanceConsoleCounts == 3 { // gen两次，有两个不同服务
			t.Logf("pass")
		} else {
			t.Fatalf("error: %d, %d", servicesCount, instancesCount)
		}
	})

	t.Run("数据为空，更新的内容为空", func(t *testing.T) {
		_ = ic.Clear()
		gomock.InOrder(storage.EXPECT().
			GetMoreInstances(gomock.Any(), gomock.Any(), ic.IsFirstUpdate(), ic.needMeta, ic.systemServiceID).
			Return(nil, nil))
		gomock.InOrder(storage.EXPECT().
			GetMoreInstanceConsoles(gomock.Any(), gomock.Any(), ic.IsFirstUpdate(), ic.needMeta, ic.systemServiceID).
			Return(nil, nil))
		if err := ic.Update(); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		servicesCount, instancesCount := iteratorInstances(ic)
		instanceConsoleCounts := ic.instanceConsoles.Len()
		if servicesCount != 0 || instancesCount != 0 || instanceConsoleCounts != 0 {
			t.Fatalf("error: %d %d", servicesCount, instancesCount)
		}
	})

	t.Run("lastMtime可以正常更新", func(t *testing.T) {
		_ = ic.Clear()
		instances := genModelInstances("services", 10)
		instanceConsoles := genModelInstancesConsole("console", 3)
		maxMtime := time.Now()
		instances[fmt.Sprintf("instanceID-%s-%d", "services", 5)].ModifyTime = maxMtime

		gomock.InOrder(
			storage.EXPECT().
				GetMoreInstances(gomock.Any(), gomock.Any(), gomock.Any(), ic.needMeta, ic.systemServiceID).
				Return(instances, nil),
			storage.EXPECT().
				GetMoreInstanceConsoles(gomock.Any(), gomock.Any(), ic.IsFirstUpdate(), ic.needMeta, ic.systemServiceID).
				Return(instanceConsoles, nil),
			storage.EXPECT().GetUnixSecond(gomock.Any()).Return(maxMtime.Unix(), nil).AnyTimes(),
		)
		if err := ic.Update(); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		if ic.LastMtime().Unix() != maxMtime.Unix() {
			t.Fatalf("error %d %d", ic.LastMtime().Unix(), maxMtime.Unix())
		}
	})
}

// TestInstanceCache_Update2 异常场景下的update测试
func TestInstanceCache_Update2(t *testing.T) {
	ctl, storage, ic := newTestInstanceCache(t)
	defer ctl.Finish()
	t.Run("数据库返回失败，update会返回失败", func(t *testing.T) {
		_ = ic.Clear()
		gomock.InOrder(storage.EXPECT().
			GetMoreInstances(gomock.Any(), gomock.Any(), ic.IsFirstUpdate(), ic.needMeta, ic.systemServiceID).
			Return(nil, fmt.Errorf("storage get error")))
		gomock.InOrder(storage.EXPECT().
			GetMoreInstanceConsoles(gomock.Any(), gomock.Any(), ic.IsFirstUpdate(), ic.needMeta, ic.systemServiceID).
			Return(nil, nil))
		gomock.InOrder(storage.EXPECT().GetInstancesCountTx(gomock.Any()).Return(uint32(0), fmt.Errorf("storage get error")))
		if err := ic.Update(); err != nil {
			t.Logf("pass: %s", err.Error())
		} else {
			t.Errorf("error")
		}
	})

	t.Run("更新数据，再删除部分数据，缓存正常", func(t *testing.T) {
		_ = ic.Clear()
		instances := genModelInstances("service-a", 20)
		instanceConsoles := genModelInstancesConsole("console", 3)
		gomock.InOrder(storage.EXPECT().
			GetMoreInstances(gomock.Any(), gomock.Any(), ic.IsFirstUpdate(), ic.needMeta, ic.systemServiceID).
			Return(instances, nil))
		gomock.InOrder(storage.EXPECT().
			GetMoreInstanceConsoles(gomock.Any(), gomock.Any(), ic.IsFirstUpdate(), ic.needMeta, ic.systemServiceID).
			Return(instanceConsoles, nil))
		if err := ic.Update(); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		idx := 0
		var invalidCount = 0
		for _, entry := range instances {
			if idx%2 == 0 {
				entry.Valid = false
				invalidCount++
			}
			idx++
		}
		gomock.InOrder(storage.EXPECT().
			GetMoreInstances(gomock.Any(), gomock.Any(), ic.IsFirstUpdate(), ic.needMeta, ic.systemServiceID).
			Return(instances, nil))
		gomock.InOrder(storage.EXPECT().
			GetMoreInstanceConsoles(gomock.Any(), gomock.Any(), ic.IsFirstUpdate(), ic.needMeta, ic.systemServiceID).
			Return(instanceConsoles, nil))
		if err := ic.Update(); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		servicesCount, instancesCount := iteratorInstances(ic)
		if servicesCount != 1 || instancesCount != 10 {
			t.Fatalf("error: %d %d", servicesCount, instancesCount)
		}
	})

	t.Run("对账发现缓存数据数量和存储层不一致", func(t *testing.T) {
		_ = ic.Clear()
		instances := genModelInstances("service-a", 20)

		queryCount := int32(0)
		storage.EXPECT().GetInstancesCountTx(gomock.Any()).Return(uint32(0), nil).AnyTimes()
		storage.EXPECT().
			GetMoreInstances(gomock.Any(), gomock.Any(), ic.IsFirstUpdate(), ic.needMeta, ic.systemServiceID).
			DoAndReturn(func(tx store.Tx, mtime time.Time, firstUpdate, needMeta bool, svcIds []string) (map[string]*model.Instance, error) {
				atomic.AddInt32(&queryCount, 1)
				if atomic.LoadInt32(&queryCount) == 2 {
					assert.Equal(t, time.Unix(0, 0), mtime)
				}
				return instances, nil
			}).AnyTimes()

		if err := ic.Update(); err != nil {
			t.Fatalf("error: %s", err.Error())
		}
	})
}

// 根据实例ID获取缓存内容
func TestInstanceCache_GetInstance(t *testing.T) {
	ctl, storage, ic := newTestInstanceCache(t)
	defer ctl.Finish()
	t.Run("缓存有数据，可以正常获取到数据", func(t *testing.T) {
		_ = ic.Clear()
		instances := genModelInstances("my-services", 10)
		instanceConsoles := genModelInstancesConsole("console", 3)

		gomock.InOrder(storage.EXPECT().
			GetMoreInstances(gomock.Any(), gomock.Any(), ic.IsFirstUpdate(), ic.needMeta, ic.systemServiceID).
			Return(instances, nil))
		gomock.InOrder(storage.EXPECT().
			GetMoreInstanceConsoles(gomock.Any(), gomock.Any(), ic.IsFirstUpdate(), ic.needMeta, ic.systemServiceID).
			Return(instanceConsoles, nil))
		gomock.InOrder(storage.EXPECT().GetInstancesCountTx(gomock.Any()).Return(uint32(10), nil))
		if err := ic.Update(); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		if instance := ic.GetInstance(instances[fmt.Sprintf("instanceID-%s-%d", "my-services", 6)].ID()); instance == nil {
			t.Fatalf("error")
		}

		if instance := ic.GetInstance("test-instance-xx"); instance != nil {
			t.Fatalf("error")
		}

		if instanceConsole := ic.GetInstanceConsole(instanceConsoles[fmt.Sprintf("InstanceConsole-%s-%d", "console", 2)].Id); instanceConsole == nil {
			t.Fatalf("error")
		}

		if instanceConsole := ic.GetInstance("test-instanceConsole-xx"); instanceConsole != nil {
			t.Fatalf("error")
		}
	})
}

func TestInstanceCache_GetServicePorts(t *testing.T) {
	ctl, storage, ic := newTestInstanceCache(t)
	defer ctl.Finish()
	t.Run("缓存有数据，可以正常获取到服务的端口列表", func(t *testing.T) {
		_ = ic.Clear()
		instances := genModelInstances("my-services", 10)
		instanceConsoles := genModelInstancesConsole("console", 3)

		ports := make(map[string][]*model.ServicePort)

		for i := range instances {
			ins := instances[i]
			if _, ok := ports[ins.ServiceID]; !ok {
				ports[ins.ServiceID] = make([]*model.ServicePort, 0, 4)
			}

			values := ports[ins.ServiceID]
			find := false

			for j := range values {
				if values[j].Port == ins.Port() {
					find = true
					break
				}
			}

			if !find {
				values = append(values, &model.ServicePort{
					Port: ins.Port(),
				})
			}

			ports[ins.ServiceID] = values
		}

		gomock.InOrder(storage.EXPECT().
			GetMoreInstances(gomock.Any(), gomock.Any(), ic.IsFirstUpdate(), ic.needMeta, ic.systemServiceID).
			Return(instances, nil))
		gomock.InOrder(storage.EXPECT().
			GetMoreInstanceConsoles(gomock.Any(), gomock.Any(), ic.IsFirstUpdate(), ic.needMeta, ic.systemServiceID).
			Return(instanceConsoles, nil))
		gomock.InOrder(storage.EXPECT().GetInstancesCountTx(gomock.Any()).Return(uint32(10), nil))
		if err := ic.Update(); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		for i := range instances {
			ins := instances[i]

			expectVal := ports[ins.ServiceID]
			targetVal := ic.GetServicePorts(ins.ServiceID)
			t.Logf("service-ports expectVal : %v, targetVal : %v", expectVal, targetVal)
			assert.ElementsMatch(t, expectVal, targetVal)
		}
	})
}

func TestInstanceCache_fillIntrnalLabels(t *testing.T) {
	ctl, storage, ic := newTestInstanceCache(t)
	defer ctl.Finish()
	t.Run("向实例Metadata中自动注入北极星默认label信息", func(t *testing.T) {
		_ = ic.Clear()
		instances := genModelInstances("inject-internal-label", 10)
		instanceConsoles := genModelInstancesConsole("console", 3)

		ports := make(map[string][]string)

		for i := range instances {
			ins := instances[i]
			if _, ok := ports[ins.ServiceID]; !ok {
				ports[ins.ServiceID] = make([]string, 0, 4)
			}

			values := ports[ins.ServiceID]
			find := false

			for j := range values {
				if values[j] == fmt.Sprintf("%d", ins.Port()) {
					find = true
					break
				}
			}

			if !find {
				values = append(values, fmt.Sprintf("%d", ins.Port()))
			}

			ports[ins.ServiceID] = values
		}

		gomock.InOrder(storage.EXPECT().
			GetMoreInstances(gomock.Any(), gomock.Any(), ic.IsFirstUpdate(), ic.needMeta, ic.systemServiceID).
			Return(instances, nil))
		gomock.InOrder(storage.EXPECT().
			GetMoreInstanceConsoles(gomock.Any(), gomock.Any(), ic.IsFirstUpdate(), ic.needMeta, ic.systemServiceID).
			Return(instanceConsoles, nil))
		gomock.InOrder(storage.EXPECT().GetInstancesCountTx(gomock.Any()).Return(uint32(10), nil))
		if err := ic.Update(); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		for i := range instances {
			ins := instances[i]

			assert.Equal(t, map[string]string{
				"region": "china",
				"zone":   "ap-shenzheng",
				"campus": "ap-shenzheng-1",
			}, ins.Proto.Metadata)
		}
	})
}

const (
	instances1Count = 50
	instances2Count = 30
)

// TestGetInstancesByServiceID 根据ServiceID获取缓存内容
func TestGetInstancesByServiceID(t *testing.T) {
	ctl, storage, ic := newTestInstanceCache(t)
	defer ctl.Finish()
	t.Run("可以通过serviceID获取实例信息", func(t *testing.T) {
		_ = ic.Clear()
		instances1 := genModelInstances("my-services", instances1Count)
		instances2 := genModelInstances("my-services-a", instances2Count)
		// instances2 = append(instances2, instances1...)
		instanceConsoles := genModelInstancesConsole("console", 3)

		ret := make(map[string]*model.Instance)
		for id, instance := range instances1 {
			ret[id] = instance
		}
		for id, instance := range instances2 {
			ret[id] = instance
		}

		gomock.InOrder(storage.EXPECT().
			GetMoreInstances(gomock.Any(), gomock.Any(), ic.IsFirstUpdate(), ic.needMeta, ic.systemServiceID).
			Return(ret, nil))
		gomock.InOrder(storage.EXPECT().
			GetMoreInstanceConsoles(gomock.Any(), gomock.Any(), ic.IsFirstUpdate(), ic.needMeta, ic.systemServiceID).
			Return(instanceConsoles, nil))
		gomock.InOrder(storage.EXPECT().
			GetInstancesCountTx(gomock.Any()).
			Return(uint32(instances1Count+instances2Count), nil))
		if err := ic.Update(); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		key := fmt.Sprintf("instanceID-%s-%d", "my-services-a", 1)
		if instances := ic.GetInstancesByServiceID(instances2[key].ServiceID); instances != nil {
			if len(instances) == instances2Count {
				t.Logf("pass")
			} else {
				t.Fatalf("error")
			}
		} else {
			t.Fatalf("error")
		}

		if instances := ic.GetInstancesByServiceID("aa"); instances != nil {
			t.Fatalf("error")
		}
	})
}
