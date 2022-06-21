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

	v1 "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/store/mock"
)

// 创建一个测试mock instanceCache
func newTestInstanceCache(t *testing.T) (*gomock.Controller, *mock.MockStore, *instanceCache) {
	ctl := gomock.NewController(t)

	storage := mock.NewMockStore(ctl)
	ic := newInstanceCache(storage, make(chan *revisionNotify, 1024))
	storage.EXPECT().GetUnixSecond().AnyTimes().Return(time.Now().Unix(), nil)
	opt := map[string]interface{}{
		"disableBusiness": false,
		"needMeta":        true,
	}
	_ = ic.initialize(opt)

	return ctl, storage, ic
}

// 生成测试数据
func genModelInstances(label string, total int) map[string]*model.Instance {
	out := make(map[string]*model.Instance)
	for i := 0; i < total; i++ {
		entry := &model.Instance{
			Proto: &v1.Instance{
				Id:   utils.NewStringValue(fmt.Sprintf("instanceID-%s-%d", label, i)),
				Host: utils.NewStringValue(fmt.Sprintf("host-%s-%d", label, i)),
				Port: utils.NewUInt32Value(uint32(i + 10)),
			},
			ServiceID: fmt.Sprintf("serviceID-%s", label),
			Valid:     true,
		}

		out[entry.Proto.Id.GetValue()] = entry
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
		_ = ic.clear()
		ret := make(map[string]*model.Instance)
		instances1 := genModelInstances("service1", 10) // 每次gen为一个服务的
		instances2 := genModelInstances("service2", 5)

		for id, instance := range instances1 {
			ret[id] = instance
		}
		for id, instance := range instances2 {
			ret[id] = instance
		}

		gomock.InOrder(storage.EXPECT().
			GetMoreInstances(gomock.Any(), ic.firstUpdate, ic.needMeta, ic.systemServiceID).
			Return(ret, nil))
		gomock.InOrder(storage.EXPECT().GetInstancesCount().Return(uint32(15), nil))
		if err := ic.update(0); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		servicesCount, instancesCount := iteratorInstances(ic)
		if servicesCount == 2 && instancesCount == 10+5 { // gen两次，有两个不同服务
			t.Logf("pass")
		} else {
			t.Fatalf("error: %d, %d", servicesCount, instancesCount)
		}
	})

	t.Run("数据为空，更新的内容为空", func(t *testing.T) {
		_ = ic.clear()
		gomock.InOrder(storage.EXPECT().
			GetMoreInstances(gomock.Any(), ic.firstUpdate, ic.needMeta, ic.systemServiceID).
			Return(nil, nil))
		if err := ic.update(0); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		servicesCount, instancesCount := iteratorInstances(ic)
		if servicesCount != 0 || instancesCount != 0 {
			t.Fatalf("error: %d %d", servicesCount, instancesCount)
		}
	})

	t.Run("lastMtime可以正常更新", func(t *testing.T) {
		_ = ic.clear()
		instances := genModelInstances("services", 10)
		maxMtime := time.Unix(1000, 0)
		instances[fmt.Sprintf("instanceID-%s-%d", "services", 5)].ModifyTime = maxMtime

		gomock.InOrder(storage.EXPECT().
			GetMoreInstances(gomock.Any(), ic.firstUpdate, ic.needMeta, ic.systemServiceID).
			Return(instances, nil))
		if err := ic.update(0); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		if ic.lastMtime != maxMtime.Unix() {
			t.Fatalf("error %d %d", ic.lastMtime, maxMtime.Unix())
		}
	})
}

// TestInstanceCache_Update2 异常场景下的update测试
func TestInstanceCache_Update2(t *testing.T) {
	ctl, storage, ic := newTestInstanceCache(t)
	defer ctl.Finish()
	t.Run("数据库返回失败，update会返回失败", func(t *testing.T) {
		_ = ic.clear()
		gomock.InOrder(storage.EXPECT().
			GetMoreInstances(gomock.Any(), ic.firstUpdate, ic.needMeta, ic.systemServiceID).
			Return(nil, fmt.Errorf("storage get error")))
		gomock.InOrder(storage.EXPECT().GetInstancesCount().Return(uint32(0), fmt.Errorf("storage get error")))
		if err := ic.update(0); err != nil {
			t.Logf("pass: %s", err.Error())
		} else {
			t.Errorf("error")
		}
	})
	t.Run("更新数据，再删除部分数据，缓存正常", func(t *testing.T) {
		_ = ic.clear()
		instances := genModelInstances("service-a", 20)
		gomock.InOrder(storage.EXPECT().
			GetMoreInstances(gomock.Any(), ic.firstUpdate, ic.needMeta, ic.systemServiceID).
			Return(instances, nil))
		if err := ic.update(0); err != nil {
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
			GetMoreInstances(gomock.Any(), ic.firstUpdate, ic.needMeta, ic.systemServiceID).
			Return(instances, nil))
		if err := ic.update(0); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		servicesCount, instancesCount := iteratorInstances(ic)
		if servicesCount != 1 || instancesCount != 10 {
			t.Fatalf("error: %d %d", servicesCount, instancesCount)
		}
	})
}

// 根据实例ID获取缓存内容
func TestInstanceCache_GetInstance(t *testing.T) {
	ctl, storage, ic := newTestInstanceCache(t)
	defer ctl.Finish()
	t.Run("缓存有数据，可以正常获取到数据", func(t *testing.T) {
		_ = ic.clear()
		instances := genModelInstances("my-services", 10)
		gomock.InOrder(storage.EXPECT().
			GetMoreInstances(gomock.Any(), ic.firstUpdate, ic.needMeta, ic.systemServiceID).
			Return(instances, nil))
		gomock.InOrder(storage.EXPECT().GetInstancesCount().Return(uint32(10), nil))
		if err := ic.update(0); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		if instance := ic.GetInstance(instances[fmt.Sprintf("instanceID-%s-%d", "my-services", 6)].ID()); instance == nil {
			t.Fatalf("error")
		}

		if instance := ic.GetInstance("test-instance-xx"); instance != nil {
			t.Fatalf("error")
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
		_ = ic.clear()
		instances1 := genModelInstances("my-services", instances1Count)
		instances2 := genModelInstances("my-services-a", instances2Count)
		// instances2 = append(instances2, instances1...)

		ret := make(map[string]*model.Instance)
		for id, instance := range instances1 {
			ret[id] = instance
		}
		for id, instance := range instances2 {
			ret[id] = instance
		}

		gomock.InOrder(storage.EXPECT().
			GetMoreInstances(gomock.Any(), ic.firstUpdate, ic.needMeta, ic.systemServiceID).
			Return(ret, nil))
		gomock.InOrder(storage.EXPECT().
			GetInstancesCount().
			Return(uint32(instances1Count+instances2Count), nil))
		if err := ic.update(0); err != nil {
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
