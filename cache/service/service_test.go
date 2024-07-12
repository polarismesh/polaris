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
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"github.com/stretchr/testify/assert"

	types "github.com/polarismesh/polaris/cache/api"
	cachemock "github.com/polarismesh/polaris/cache/mock"
	"github.com/polarismesh/polaris/common/eventhub"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
	"github.com/polarismesh/polaris/store/mock"
)

// 生成一个测试的serviceCache和对应的mock对象
func newTestServiceCache(t *testing.T) (*gomock.Controller, *mock.MockStore, *serviceCache, *instanceCache) {
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
	_ = mockInstCache.Initialize(opt)
	_ = mockSvcCache.Initialize(opt)

	return ctl, storage, mockSvcCache.(*serviceCache), mockInstCache.(*instanceCache)
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
			Revision:   utils.NewUUID(),
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
			ID:         fmt.Sprintf("%s-ID-%d", namespace, i),
			Namespace:  namespace,
			Name:       fmt.Sprintf("Name-%d", i),
			Valid:      true,
			Revision:   utils.NewUUID(),
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
				Proto: &apiservice.Instance{
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

	t.Run("所有数据为空, 可以正常获取数据", func(t *testing.T) {
		gomock.InOrder(
			storage.EXPECT().
				GetMoreServices(gomock.Any(), sc.IsFirstUpdate(), sc.disableBusiness, sc.needMeta).
				Return(nil, nil).Times(1),
			storage.EXPECT().GetServicesCount().AnyTimes().Return(uint32(0), nil),
		)

		if err := sc.Update(); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		if sum := getServiceCacheCount(sc); sum != 0 {
			t.Fatalf("error: %d", sum)
		}
	})
	t.Run("有数据更新, 数据正常", func(t *testing.T) {
		_ = sc.Clear()
		services := genModelService(100)
		gomock.InOrder(
			storage.EXPECT().GetMoreServices(gomock.Any(), sc.IsFirstUpdate(), sc.disableBusiness, sc.needMeta).
				Return(services, nil),
			storage.EXPECT().GetServicesCount().AnyTimes().Return(uint32(len(services)), nil),
		)

		if err := sc.Update(); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		if sum := getServiceCacheCount(sc); sum != 100 {
			t.Fatalf("error: %d", sum)
		}
	})
	t.Run("有数据更新, 重复更新, 数据更新正常", func(t *testing.T) {
		_ = sc.Clear()
		services1 := genModelService(100)
		services2 := genModelService(300)
		gomock.InOrder(
			storage.EXPECT().GetMoreServices(gomock.Any(), sc.IsFirstUpdate(), sc.disableBusiness, sc.needMeta).
				Return(services1, nil),
			storage.EXPECT().GetServicesCount().AnyTimes().Return(uint32(len(services1)), nil),
		)

		if err := sc.Update(); err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		gomock.InOrder(
			storage.EXPECT().GetMoreServices(gomock.Any(), sc.IsFirstUpdate(), sc.disableBusiness, sc.needMeta).
				Return(services2, nil),
			storage.EXPECT().GetServicesCount().AnyTimes().Return(uint32(len(services2)), nil),
		)
		_ = sc.Update()
		if sum := getServiceCacheCount(sc); sum != 300 {
			t.Fatalf("error: %d", sum)
		}
	})
}

// TestServiceUpdate1 测试缓存更新函数1
func TestServiceUpdate1(t *testing.T) {
	ctl, storage, sc, _ := newTestServiceCache(t)
	defer ctl.Finish()

	t.Run("服务全部被删除, 会被清除掉", func(t *testing.T) {
		_ = sc.Clear()
		services := genModelService(100)
		gomock.InOrder(storage.EXPECT().
			GetMoreServices(gomock.Any(), sc.IsFirstUpdate(), sc.disableBusiness, sc.needMeta).Return(services, nil),
			storage.EXPECT().GetServicesCount().AnyTimes().Return(uint32(100), nil),
		)
		_ = sc.Update()

		// 把所有的都置为false
		for _, service := range services {
			service.Valid = false
		}

		gomock.InOrder(storage.EXPECT().
			GetMoreServices(gomock.Any(), sc.IsFirstUpdate(), sc.disableBusiness, sc.needMeta).Return(services, nil),
			storage.EXPECT().GetServicesCount().AnyTimes().Return(uint32(0), nil),
		)
		_ = sc.Update()

		if sum := getServiceCacheCount(sc); sum != 0 {
			t.Fatalf("error: %d", sum)
		}
	})

	t.Run("服务部分被删除, 缓存内容正常", func(t *testing.T) {
		_ = sc.Clear()
		services := genModelService(100)
		gomock.InOrder(storage.EXPECT().
			GetMoreServices(gomock.Any(), sc.IsFirstUpdate(), sc.disableBusiness, sc.needMeta).Return(services, nil),
			storage.EXPECT().GetServicesCount().AnyTimes().Return(uint32(len(services)), nil),
		)
		_ = sc.Update()

		// 把所有的都置为false
		count := len(services)
		idx := 0
		for _, service := range services {
			if idx%2 == 0 {
				service.Valid = false
				count--
			}
			idx++
		}

		gomock.InOrder(storage.EXPECT().
			GetMoreServices(gomock.Any(), sc.IsFirstUpdate(), sc.disableBusiness, sc.needMeta).Return(services, nil),
			storage.EXPECT().GetServicesCount().AnyTimes().Return(uint32(count), nil),
		)
		_ = sc.Update()

		if sum := getServiceCacheCount(sc); sum != 50 { // remain half
			t.Fatalf("error: %d", sum)
		}
	})
}

// TestServiceUpdate2 测试缓存更新
func TestServiceUpdate2(t *testing.T) {
	ctl, storage, sc, _ := newTestServiceCache(t)
	defer ctl.Finish()

	t.Run("store返回失败, update会返回失败", func(t *testing.T) {
		_ = sc.Clear()
		gomock.InOrder(
			storage.EXPECT().GetMoreServices(gomock.Any(), sc.IsFirstUpdate(), sc.disableBusiness, sc.needMeta).
				Return(nil, fmt.Errorf("store error")),
			storage.EXPECT().GetServicesCount().AnyTimes().Return(uint32(0), nil),
		)

		if err := sc.Update(); err != nil {
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
	t.Run("可以根据服务名和命名空间, 正常获取缓存服务信息", func(t *testing.T) {
		_ = sc.Clear()
		services := genModelService(20)
		sc.setServices(services)

		for _, entry := range services {
			service := sc.GetServiceByName(entry.Name, entry.Namespace)
			if service == nil {
				t.Fatalf("error")
			}
		}
	})
	t.Run("服务不存在, 返回为空", func(t *testing.T) {
		_ = sc.Clear()
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

	t.Run("可以根据服务ID, 正常获取缓存的服务信息", func(t *testing.T) {
		_ = sc.Clear()
		services := genModelService(30)
		sc.setServices(services)

		for _, entry := range services {
			service := sc.GetServiceByID(entry.ID)
			if service == nil {
				t.Fatalf("error")
			}
		}
	})

	t.Run("缓存内容为空, 根据ID获取数据, 会返回为空", func(t *testing.T) {
		_ = sc.Clear()
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
				Proto: &apiservice.Instance{
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
	ctl, mockStore, sc, _ := newTestServiceCache(t)
	defer ctl.Finish()

	t.Run("可以根据服务host-正常获取缓存的服务信息", func(t *testing.T) {
		_ = sc.Clear()
		services := genModelServiceByNamespace(100, "default")
		sc.setServices(services)

		svcInstances, instances := genModelInstancesByServices(services, 2)
		ic := sc.instCache.(*instanceCache)

		mockStore.EXPECT().GetServicesCount().Return(uint32(len(services)), nil).AnyTimes()
		mockStore.EXPECT().GetInstancesCountTx(gomock.Any()).Return(uint32(len(instances)), nil).AnyTimes()
		mockStore.EXPECT().GetMoreServices(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(services, nil).AnyTimes()
		mockStore.EXPECT().GetMoreInstances(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(instances, nil).AnyTimes()
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
			svcArgs := &types.ServiceArgs{
				EmptyCondition: true,
			}
			amount, services, err := sc.GetServicesByFilter(context.Background(), svcArgs, instArgs, 0, 10)
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

	t.Run("先刷新instancesCache, 在刷新serviceCache, 计数等待一段时间之后正常", func(t *testing.T) {
		_ = sc.Clear()
		_ = ic.Clear()

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
			acutalNsCount[ns] = int(sc.GetNamespaceCntInfo(ns).InstanceCnt.TotalInstanceCount)
		}

		fmt.Printf("expect ns count : %#v\n", expectNsCount)
		fmt.Printf("acutal ns count : %#v\n", acutalNsCount)

		if !reflect.DeepEqual(expectNsCount, acutalNsCount) {
			t.Fatal("namespace count is no currect")
		}
	})
}

// TestRevisionWorker 测试revision的管道是否正常
func TestRevisionWorker(t *testing.T) {
	ctl := gomock.NewController(t)
	storage := mock.NewMockStore(ctl)
	mockCacheMgr := cachemock.NewMockCacheManager(ctl)

	mockCacheMgr.EXPECT().GetReportInterval().Return(time.Second).AnyTimes()
	mockCacheMgr.EXPECT().GetUpdateCacheInterval().Return(time.Second).AnyTimes()
	storage.EXPECT().GetUnixSecond(gomock.Any()).AnyTimes().Return(time.Now().Unix(), nil)
	defer ctl.Finish()

	t.Run("revision计算, chan可以正常收发", func(t *testing.T) {
		svcCache := NewServiceCache(storage, mockCacheMgr)
		mockInstCache := NewInstanceCache(storage, mockCacheMgr)
		mockCacheMgr.EXPECT().GetCacher(types.CacheInstance).Return(mockInstCache).AnyTimes()
		mockCacheMgr.EXPECT().GetCacher(types.CacheService).Return(svcCache).AnyTimes()
		_ = svcCache.Initialize(map[string]interface{}{})
		_ = mockInstCache.Initialize(map[string]interface{}{})

		t.Cleanup(func() {
			_ = mockInstCache.Clear()
			_ = svcCache.Clear()
		})

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
		_ = svcCache.Update()
		time.Sleep(time.Second * 10)
		assert.Equal(t, maxTotal, svcCache.GetRevisionWorker().GetServiceRevisionCount())

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
		_ = svcCache.Update()
		time.Sleep(time.Second * 20)
		// 检查是否有正常计算
		assert.Equal(t, maxTotal/2, svcCache.GetRevisionWorker().GetServiceRevisionCount())
	})
}

// TestComputeRevision 测试计算revision的函数
func TestComputeRevision(t *testing.T) {
	t.Run("instances为空, 可以正常计算", func(t *testing.T) {
		out, err := ComputeRevision("123", nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, out)
	})

	t.Run("instances内容一样, 不同顺序, 计算出的revision一样", func(t *testing.T) {
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

		// 交换一下数据, 数据内容不变, revision应该保证不变
		tmp := instances[0]
		instances[0] = instances[1]
		instances[1] = instances[3]
		instances[3] = tmp

		rhs, err := ComputeRevision("123", nil)
		assert.NoError(t, err)
		assert.Equal(t, lhs, rhs)
	})

	t.Run("serviceRevision发生改变, 返回改变", func(t *testing.T) {
		lhs, err := ComputeRevision("123", nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, lhs)

		rhs, err := ComputeRevision("456", nil)
		assert.NoError(t, err)
		assert.NotEqual(t, lhs, rhs)
	})

	t.Run("instances内容改变, 返回改变", func(t *testing.T) {
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

func Test_serviceCache_GetVisibleServicesInOtherNamespace(t *testing.T) {
	ctl := gomock.NewController(t)
	storage := mock.NewMockStore(ctl)
	mockCacheMgr := cachemock.NewMockCacheManager(ctl)
	mockCacheMgr.EXPECT().GetReportInterval().Return(time.Second).AnyTimes()
	mockCacheMgr.EXPECT().GetUpdateCacheInterval().Return(time.Second).AnyTimes()
	defer ctl.Finish()

	t.Run("服务可见性查询判断", func(t *testing.T) {
		serviceList := map[string]*model.Service{
			"service-1": {
				ID:        "service-1",
				Name:      "service-1",
				Namespace: "ns-1",
				ExportTo: map[string]struct{}{
					"ns-2": {},
				},
				Valid: true,
			},
			"service-2": {
				ID:        "service-2",
				Name:      "service-2",
				Namespace: "ns-2",
				ExportTo:  map[string]struct{}{},
				Valid:     true,
			},
			"service-3": {
				ID:        "service-3",
				Name:      "service-3",
				Namespace: "ns-3",
				ExportTo: map[string]struct{}{
					"ns-2": {},
				},
				Valid: true,
			},
		}

		svcCache := NewServiceCache(storage, mockCacheMgr).(*serviceCache)
		mockInstCache := NewInstanceCache(storage, mockCacheMgr)
		mockCacheMgr.EXPECT().GetCacher(types.CacheInstance).Return(mockInstCache).AnyTimes()
		mockCacheMgr.EXPECT().GetCacher(types.CacheService).Return(svcCache).AnyTimes()
		_ = svcCache.Initialize(map[string]interface{}{})
		_ = mockInstCache.Initialize(map[string]interface{}{})
		t.Cleanup(func() {
			_ = svcCache.Close()
			_ = mockInstCache.Close()
		})

		_, _, _ = svcCache.setServices(serviceList)
		visibles := svcCache.GetVisibleServicesInOtherNamespace("service-1", "ns-2")
		assert.Equal(t, 1, len(visibles))
		assert.Equal(t, "ns-1", visibles[0].Namespace)
	})

	t.Run("服务可见性查询判断", func(t *testing.T) {
		serviceList := map[string]*model.Service{
			"service-1": {
				ID:        "service-1",
				Name:      "service-1",
				Namespace: "ns-1",
				Valid:     true,
			},
			"service-2": {
				ID:        "service-2",
				Name:      "service-2",
				Namespace: "ns-2",
				Valid:     true,
			},
			"service-3": {
				ID:        "service-3",
				Name:      "service-3",
				Namespace: "ns-3",
				Valid:     true,
			},
			"service-4": {
				ID:        "service-4",
				Name:      "service-4",
				Namespace: "ns-4",
				Valid:     true,
			},
		}

		svcCache := NewServiceCache(storage, mockCacheMgr).(*serviceCache)
		mockInstCache := NewInstanceCache(storage, mockCacheMgr)
		mockCacheMgr.EXPECT().GetCacher(types.CacheInstance).Return(mockInstCache).AnyTimes()
		mockCacheMgr.EXPECT().GetCacher(types.CacheService).Return(svcCache).AnyTimes()
		_ = svcCache.Initialize(map[string]interface{}{})
		_ = mockInstCache.Initialize(map[string]interface{}{})
		t.Cleanup(func() {
			_ = svcCache.Close()
			_ = mockInstCache.Close()
		})

		_, _, _ = svcCache.setServices(serviceList)

		svcCache.handleNamespaceChange(context.Background(), &eventhub.CacheNamespaceEvent{
			EventType: eventhub.EventCreated,
			Item: &model.Namespace{
				Name: "ns-1",
				ServiceExportTo: map[string]struct{}{
					"ns-2": {},
					"ns-3": {},
				},
			},
		})

		visibles := svcCache.GetVisibleServicesInOtherNamespace("service-1", "ns-2")
		assert.Equal(t, 1, len(visibles))
		assert.Equal(t, "ns-1", visibles[0].Namespace)

		visibles = svcCache.GetVisibleServicesInOtherNamespace("service-1", "ns-3")
		assert.Equal(t, 1, len(visibles))
		assert.Equal(t, "ns-1", visibles[0].Namespace)

		visibles = svcCache.GetVisibleServicesInOtherNamespace("service-1", "ns-4")
		assert.Equal(t, 0, len(visibles))
	})

}
