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
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	v1 "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/store"
	"github.com/polarismesh/polaris-server/store/mock"
)

var (
	testSvcCacheMap map[string]Cache = make(map[string]Cache)
)

// 生成一个测试的serviceCache和对应的mock对象
func newTestServiceCache(t *testing.T) (*gomock.Controller, *mock.MockStore, *serviceCache, *instanceCache) {
	ctl := gomock.NewController(t)

	storage := mock.NewMockStore(ctl)
	storage.EXPECT().GetUnixSecond().AnyTimes().Return(time.Now().Unix(), nil)
	notifier := make(chan *revisionNotify, 1024)
	ic := newInstanceCache(storage, notifier)
	sc := newServiceCache(storage, notifier, ic)
	opt := map[string]interface{}{
		"disableBusiness": false,
		"needMeta":        true,
	}
	_ = ic.initialize(opt)
	_ = sc.initialize(opt)

	ic.addListener([]Listener{
		&WatchInstanceReload{
			Handler: func(val interface{}) {
				if svcIds, ok := val.(map[string]bool); ok {
					testSvcCacheMap["serviceCache"].(*serviceCache).notifyServiceCountReload(svcIds)
				}
			},
		},
	})

	testSvcCacheMap["serviceCache"] = sc

	return ctl, storage, sc, ic
}

// 获取当前缓存中的services总数
func getServiceCacheCount(sc *serviceCache) int {
	sum := 0
	_ = sc.IteratorServices(func(key string, value *model.Service) (bool, error) {
		sum++
		return true, nil
	})
	return sum
}

// 生成一些测试的services
func genModelService(total int) map[string]*model.Service {
	out := make(map[string]*model.Service)
	for i := 0; i < total; i++ {
		item := &model.Service{
			ID:         fmt.Sprintf("ID-%d", i),
			Namespace:  fmt.Sprintf("Namespace-%d", i),
			Name:       fmt.Sprintf("Name-%d", i),
			Valid:      true,
			ModifyTime: time.Unix(int64(i), 0),
		}
		out[item.ID] = item
	}

	return out
}

// 生成一些测试的services
func genModelServiceByNamespace(total int, namespace string) map[string]*model.Service {
	out := make(map[string]*model.Service)
	for i := 0; i < total; i++ {
		item := &model.Service{
			ID:         fmt.Sprintf("ID-%d", i),
			Namespace:  namespace,
			Name:       fmt.Sprintf("Name-%d", i),
			Valid:      true,
			ModifyTime: time.Unix(int64(i), 0),
		}
		out[item.ID] = item
	}

	return out
}

func genModelInstancesByServicesWithInsId(
	services map[string]*model.Service, instCount int, insIdPrefix string) (map[string][]*model.Instance, map[string]*model.Instance) {
	var svcToInstances = make(map[string][]*model.Instance, len(services))
	var allInstances = make(map[string]*model.Instance, len(services)*instCount)
	var idx int
	for id, svc := range services {
		label := svc.Name
		instancesSvc := make([]*model.Instance, 0, instCount)
		for i := 0; i < instCount; i++ {
			entry := &model.Instance{
				Proto: &v1.Instance{
					Id:   utils.NewStringValue(fmt.Sprintf("%s-instanceID-%s-%d", insIdPrefix, label, idx)),
					Host: utils.NewStringValue(fmt.Sprintf("host-%s-%d", label, idx)),
					Port: utils.NewUInt32Value(uint32(idx + 10)),
				},
				ServiceID: svc.ID,
				Valid:     true,
			}
			idx++
			instancesSvc = append(instancesSvc, entry)
			allInstances[entry.ID()] = entry
		}
		svcToInstances[id] = instancesSvc
	}
	return svcToInstances, allInstances
}

// 生成一些测试的services
func genModelServiceByNamespaces(total int, namespace []string) map[string]*model.Service {
	out := make(map[string]*model.Service)
	for i := 0; i < total; i++ {
		item := &model.Service{
			ID:         fmt.Sprintf("ID-%d", i),
			Namespace:  namespace[rand.Intn(len(namespace))],
			Name:       fmt.Sprintf("Name-%d", i),
			Valid:      true,
			ModifyTime: time.Unix(int64(i), 0),
		}
		out[item.ID] = item
	}

	return out
}

// TestServiceUpdate 测试缓存更新函数
func TestServiceUpdate(t *testing.T) {
	ctl, storage, sc, _ := newTestServiceCache(t)
	defer ctl.Finish()

	t.Run("所有数据为空，可以正常获取数据", func(t *testing.T) {
		gomock.InOrder(
			storage.EXPECT().
				GetMoreServices(gomock.Any(), sc.firstUpdate, sc.disableBusiness, sc.needMeta).
				Return(nil, nil),
		)

		if err := sc.update(0); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		if sum := getServiceCacheCount(sc); sum != 0 {
			t.Fatalf("error: %d", sum)
		}
	})
	t.Run("有数据更新，数据正常", func(t *testing.T) {
		_ = sc.clear()
		services := genModelService(100)
		gomock.InOrder(
			storage.EXPECT().GetMoreServices(gomock.Any(), sc.firstUpdate, sc.disableBusiness, sc.needMeta).
				Return(services, nil),
		)

		if err := sc.update(0); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		if sum := getServiceCacheCount(sc); sum != 100 {
			t.Fatalf("error: %d", sum)
		}
	})
	t.Run("有数据更新，重复更新，数据更新正常", func(t *testing.T) {
		_ = sc.clear()
		services1 := genModelService(100)
		services2 := genModelService(300)
		gomock.InOrder(
			storage.EXPECT().GetMoreServices(gomock.Any(), sc.firstUpdate, sc.disableBusiness, sc.needMeta).
				Return(services1, nil),
		)

		if err := sc.update(0); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		gomock.InOrder(
			storage.EXPECT().GetMoreServices(gomock.Any(), sc.firstUpdate, sc.disableBusiness, sc.needMeta).
				Return(services2, nil),
		)
		_ = sc.update(0)
		if sum := getServiceCacheCount(sc); sum != 300 {
			t.Fatalf("error: %d", sum)
		}
	})
}

// TestServiceUpdate1 测试缓存更新函数1
func TestServiceUpdate1(t *testing.T) {
	ctl, storage, sc, _ := newTestServiceCache(t)
	defer ctl.Finish()

	t.Run("服务全部被删除，会被清除掉", func(t *testing.T) {
		_ = sc.clear()
		services := genModelService(100)
		gomock.InOrder(storage.EXPECT().
			GetMoreServices(gomock.Any(), sc.firstUpdate, sc.disableBusiness, sc.needMeta).Return(services, nil))
		_ = sc.update(0)

		// 把所有的都置为false
		for _, service := range services {
			service.Valid = false
		}

		gomock.InOrder(storage.EXPECT().
			GetMoreServices(gomock.Any(), sc.firstUpdate, sc.disableBusiness, sc.needMeta).Return(services, nil))
		_ = sc.update(0)

		if sum := getServiceCacheCount(sc); sum != 0 {
			t.Fatalf("error: %d", sum)
		}
	})

	t.Run("服务部分被删除，缓存内容正常", func(t *testing.T) {
		_ = sc.clear()
		services := genModelService(100)
		gomock.InOrder(storage.EXPECT().
			GetMoreServices(gomock.Any(), sc.firstUpdate, sc.disableBusiness, sc.needMeta).Return(services, nil))
		_ = sc.update(0)

		// 把所有的都置为false
		idx := 0
		for _, service := range services {
			if idx%2 == 0 {
				service.Valid = false
			}
			idx++
		}

		gomock.InOrder(storage.EXPECT().
			GetMoreServices(gomock.Any(), sc.firstUpdate, sc.disableBusiness, sc.needMeta).Return(services, nil))
		_ = sc.update(0)

		if sum := getServiceCacheCount(sc); sum != 50 { // remain half
			t.Fatalf("error: %d", sum)
		}
	})
}

// TestServiceUpdate2 测试缓存更新
func TestServiceUpdate2(t *testing.T) {
	ctl, storage, sc, _ := newTestServiceCache(t)
	defer ctl.Finish()

	t.Run("store返回失败，update会返回失败", func(t *testing.T) {
		_ = sc.clear()
		gomock.InOrder(
			storage.EXPECT().GetMoreServices(gomock.Any(), sc.firstUpdate, sc.disableBusiness, sc.needMeta).
				Return(nil, fmt.Errorf("store error")),
		)

		if err := sc.update(0); err != nil {
			t.Logf("pass: %s", err.Error())
		} else {
			t.Fatalf("error")
		}
	})
}

// TestGetServiceByName 根据服务名获取服务缓存信息
func TestGetServiceByName(t *testing.T) {
	ctl, _, sc, _ := newTestServiceCache(t)
	defer ctl.Finish()
	t.Run("可以根据服务名和命名空间，正常获取缓存服务信息", func(t *testing.T) {
		_ = sc.clear()
		services := genModelService(20)
		sc.setServices(services)

		for _, entry := range services {
			service := sc.GetServiceByName(entry.Name, entry.Namespace)
			if service == nil {
				t.Fatalf("error")
			}
		}
	})
	t.Run("服务不存在，返回为空", func(t *testing.T) {
		_ = sc.clear()
		services := genModelService(20)
		sc.setServices(services)
		if service := sc.GetServiceByName("aaa", "bbb"); service != nil {
			t.Fatalf("error")
		}
	})
}

// TestServiceCache_GetServiceByID 根据服务ID获取服务缓存信息
func TestServiceCache_GetServiceByID(t *testing.T) {
	ctl, _, sc, _ := newTestServiceCache(t)
	defer ctl.Finish()

	t.Run("可以根据服务ID，正常获取缓存的服务信息", func(t *testing.T) {
		_ = sc.clear()
		services := genModelService(30)
		sc.setServices(services)

		for _, entry := range services {
			service := sc.GetServiceByID(entry.ID)
			if service == nil {
				t.Fatalf("error")
			}
		}
	})

	t.Run("缓存内容为空，根据ID获取数据，会返回为空", func(t *testing.T) {
		_ = sc.clear()
		services := genModelService(30)
		sc.setServices(services)

		if service := sc.GetServiceByID("123456789"); service != nil {
			t.Fatalf("error")
		}
	})
}

func genModelInstancesByServices(
	services map[string]*model.Service, instCount int) (map[string][]*model.Instance, map[string]*model.Instance) {
	var svcToInstances = make(map[string][]*model.Instance, len(services))
	var allInstances = make(map[string]*model.Instance, len(services)*instCount)
	var idx int
	for id, svc := range services {
		label := svc.Name
		instancesSvc := make([]*model.Instance, 0, instCount)
		for i := 0; i < instCount; i++ {
			entry := &model.Instance{
				Proto: &v1.Instance{
					Id:   utils.NewStringValue(fmt.Sprintf("instanceID-%s-%d", label, idx)),
					Host: utils.NewStringValue(fmt.Sprintf("host-%s-%d", label, idx)),
					Port: utils.NewUInt32Value(uint32(idx + 10)),
				},
				ServiceID: svc.ID,
				Valid:     true,
			}
			idx++
			instancesSvc = append(instancesSvc, entry)
			allInstances[entry.ID()] = entry
		}
		svcToInstances[id] = instancesSvc
	}
	return svcToInstances, allInstances
}

// TestServiceCache_GetServicesByFilter 根据实例的host查询对应的服务列表
func TestServiceCache_GetServicesByFilter(t *testing.T) {
	ctl, _, sc, _ := newTestServiceCache(t)
	defer ctl.Finish()

	t.Run("可以根据服务host，正常获取缓存的服务信息", func(t *testing.T) {
		_ = sc.clear()
		services := genModelServiceByNamespace(100, "default")
		sc.setServices(services)

		svcInstances, instances := genModelInstancesByServices(services, 2)
		ic := sc.instCache.(*instanceCache)
		ic.setInstances(instances)

		hostToService := make(map[string]string)
		for svc, instances := range svcInstances {
			hostToService[instances[0].Host()] = svc
		}
		// 先不带命名空间进行查询
		for host, svcId := range hostToService {
			instArgs := &store.InstanceArgs{
				Hosts: []string{host},
			}
			svcArgs := &ServiceArgs{
				EmptyCondition: true,
			}
			amount, services, err := sc.GetServicesByFilter(svcArgs, instArgs, 0, 10)
			if err != nil {
				t.Fatal(err)
			}
			if amount != 1 {
				t.Fatalf("service count is %d, expect 1", amount)
			}
			if len(services) != 1 {
				t.Fatalf("service count is %d, expect 1", len(services))
			}
			if services[0].ID != svcId {
				t.Fatalf("service id not match, actual %s, expect %s", services[0].ID, svcId)
			}
		}
	})
}

func TestServiceCache_NamespaceCount(t *testing.T) {
	ctl, _, sc, ic := newTestServiceCache(t)
	defer ctl.Finish()

	t.Run("先刷新serviceCache，在刷新InstancesCache，计数等待一段时间之后正常", func(t *testing.T) {
		_ = sc.clear()
		_ = ic.clear()

		nsList := []string{"default", "test-1", "test-2", "test-3"}
		services := genModelServiceByNamespaces(100, nsList)
		sc.setServices(services)
		expectNsInsCount := make(map[string]int)
		acutalNsInsCount := make(map[string]int)

		// 这个时候拉取，数据不正常
		for i := range nsList {
			ns := nsList[i]
			acutalNsInsCount[ns] = int(sc.GetNamesapceCntInfo(ns).InstanceCnt.TotalInstanceCount)
		}

		fmt.Printf("expect ns-ins count : %#v\n", expectNsInsCount)
		fmt.Printf("acutal ns-ins count : %#v\n", acutalNsInsCount)
		if reflect.DeepEqual(expectNsInsCount, acutalNsInsCount) {
			t.Fatal("namespace count should be incurrect")
		}

		culTask := func(prefix string, isAdd bool, judge func()) {
			// 生存测试的实例数据列表
			svcInstances, instances := genModelInstancesByServicesWithInsId(services, 2, prefix)

			if !isAdd {
				for i := range instances {
					ins := instances[i]
					ins.Valid = rand.Int31n(int32(10)) < 5
				}
			}

			for svcId, instances := range svcInstances {
				svc := services[svcId]
				if _, ok := expectNsInsCount[svc.Namespace]; !ok {
					expectNsInsCount[svc.Namespace] = 0
				}
				count := 0
				for i := range instances {
					ins := instances[i]
					if ins.Valid {
						count++
					}
				}
				expectNsInsCount[svc.Namespace] += count
			}

			// 更新 instanceCache 缓存
			ic := sc.instCache.(*instanceCache)
			ic.setInstances(instances)

			time.Sleep(time.Duration(2 * time.Second))

			// 这个时候计算，数据应该正确
			acutalNsInsCount = make(map[string]int)
			for i := range nsList {
				ns := nsList[i]
				acutalNsInsCount[ns] = int(sc.GetNamesapceCntInfo(ns).InstanceCnt.TotalInstanceCount)
			}
			fmt.Printf("expect ns-ins count : %#v\n", expectNsInsCount)
			fmt.Printf("acutal ns-ins count : %#v\n", acutalNsInsCount)
			judge()
		}

		culTask("test-1", true, func() {
			if !reflect.DeepEqual(expectNsInsCount, acutalNsInsCount) {
				t.Fatal("namespace count is no currect")
			}
		})

		// 只更新 instance 数据，实例计算应该要正确
		culTask("test-2", true, func() {
			if !reflect.DeepEqual(expectNsInsCount, acutalNsInsCount) {
				t.Fatal("namespace count is no currect")
			}
		})

		// 只更新 instance 数据，实例计算应该要正确
		culTask("test-3", false, func() {
			if !reflect.DeepEqual(expectNsInsCount, acutalNsInsCount) {
				t.Fatal("namespace count is no currect")
			}
		})
	})

	t.Run("先刷新instancesCache，在刷新serviceCache，计数等待一段时间之后正常", func(t *testing.T) {
		_ = sc.clear()
		_ = ic.clear()

		ic := sc.instCache.(*instanceCache)

		nsList := []string{"default", "test-1", "test-2", "test-3"}

		services := genModelServiceByNamespaces(50, nsList)
		svcInstances, instances := genModelInstancesByServices(services, 2)
		expectNsCount := make(map[string]int)

		for svcId, instances := range svcInstances {
			svc := services[svcId]
			if _, ok := expectNsCount[svc.Namespace]; !ok {
				expectNsCount[svc.Namespace] = 0
			}
			expectNsCount[svc.Namespace] += len(instances)
		}

		ic.setInstances(instances)
		time.Sleep(time.Duration(5 * time.Second))
		sc.setServices(services)
		time.Sleep(time.Duration(5 * time.Second))

		acutalNsCount := make(map[string]int)
		for i := range nsList {
			ns := nsList[i]
			acutalNsCount[ns] = int(sc.GetNamesapceCntInfo(ns).InstanceCnt.TotalInstanceCount)
		}

		fmt.Printf("expect ns count : %#v\n", expectNsCount)
		fmt.Printf("acutal ns count : %#v\n", acutalNsCount)

		if !reflect.DeepEqual(expectNsCount, acutalNsCount) {
			t.Fatal("namespace count is no currect")
		}
	})
}
