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

package test

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/naming"
	. "github.com/smartystreets/goconvey/convey"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// 测试新建实例
func TestCreateInstance(t *testing.T) {
	_, serviceResp := createCommonService(t, 100)
	defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	t.Run("正常创建实例-服务没有提前创建", func(t *testing.T) {
		instanceReq, instanceResp := createCommonInstance(t, &api.Service{
			Name:      wrapperspb.String("test-nocreate-service"),
			Namespace: utils.NewStringValue(naming.DefaultNamespace),
		}, 1000)
		defer cleanInstance(instanceResp.GetId().GetValue())

		if instanceResp.GetId().GetValue() != "" {
			t.Logf("pass: %s", instanceResp.GetId().GetValue())
		} else {
			t.Fatalf("error")
		}

		if instanceResp.GetNamespace().GetValue() == instanceReq.GetNamespace().GetValue() &&
			instanceResp.GetService().GetValue() == instanceReq.GetService().GetValue() {
			t.Logf("pass")
		} else {
			t.Fatalf("error: %+v", instanceResp)
		}
	})

	t.Run("正常创建实例-服务已创建", func(t *testing.T) {
		instanceReq, instanceResp := createCommonInstance(t, serviceResp, 1000)
		defer cleanInstance(instanceResp.GetId().GetValue())

		if instanceResp.GetId().GetValue() != "" {
			t.Logf("pass: %s", instanceResp.GetId().GetValue())
		} else {
			t.Fatalf("error")
		}

		if instanceResp.GetNamespace().GetValue() == instanceReq.GetNamespace().GetValue() &&
			instanceResp.GetService().GetValue() == instanceReq.GetService().GetValue() {
			t.Logf("pass")
		} else {
			t.Fatalf("error: %+v", instanceResp)
		}
	})

	t.Run("重复注册，会返回资源已经存在", func(t *testing.T) {
		req, instanceResp := createCommonInstance(t, serviceResp, 1000)
		defer cleanInstance(instanceResp.GetId().GetValue())

		time.Sleep(time.Second)
		resp := server.CreateInstance(defaultCtx, req)
		if !respSuccess(resp) && resp.GetCode().GetValue() == api.ExistedResource {
			t.Logf("pass: %+v", resp)
		} else {
			t.Fatalf("error: %+v", resp)
		}
		if resp.Instance.GetId().GetValue() == "" {
			t.Fatalf("error: %+v", resp)
		}
	})

	t.Run("instance有metadata个数和字符要求的限制", func(t *testing.T) {
		instanceReq := &api.Instance{
			ServiceToken: utils.NewStringValue(serviceResp.GetToken().GetValue()),
			Service:      utils.NewStringValue(serviceResp.GetName().GetValue()),
			Namespace:    utils.NewStringValue(serviceResp.GetNamespace().GetValue()),
			Host:         utils.NewStringValue("123"),
			Port:         utils.NewUInt32Value(456),
			Metadata:     make(map[string]string),
		}
		for i := 0; i < naming.MaxMetadataLength+1; i++ {
			instanceReq.Metadata[fmt.Sprintf("%d", i)] = fmt.Sprintf("%d", i)
		}
		if resp := server.CreateInstance(defaultCtx, instanceReq); respSuccess(resp) {
			t.Fatalf("error")
		} else {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		}
	})
	t.Run("healthcheck为空测试", func(t *testing.T) {
		instanceReq := &api.Instance{
			ServiceToken: utils.NewStringValue(serviceResp.GetToken().GetValue()),
			Service:      utils.NewStringValue(serviceResp.GetName().GetValue()),
			Namespace:    utils.NewStringValue(serviceResp.GetNamespace().GetValue()),
			Host:         utils.NewStringValue("aaaaaaaaaaaaaa"),
			Port:         utils.NewUInt32Value(456),
			HealthCheck:  &api.HealthCheck{},
		}
		resp := server.CreateInstance(defaultCtx, instanceReq)
		if !respSuccess(resp) {
			t.Fatalf("error: %+v", resp)
		}
		defer cleanInstance(resp.GetInstance().GetId().GetValue())

		time.Sleep(time.Second)
		cleanInstance(resp.GetInstance().GetId().GetValue())
		instanceReq.HealthCheck = &api.HealthCheck{
			Heartbeat: &api.HeartbeatHealthCheck{},
		}
		resp = server.CreateInstance(defaultCtx, instanceReq)
		if !respSuccess(resp) {
			t.Fatalf("error: %+v", resp)
		}
		getResp := server.GetInstances(map[string]string{"host": instanceReq.GetHost().GetValue()})
		t.Logf("%+v", getResp)
		if getResp.GetInstances()[0].HealthCheck.Type != api.HealthCheck_HEARTBEAT {
			t.Fatalf("error")
		}
		if getResp.GetInstances()[0].HealthCheck.Heartbeat.Ttl.Value != naming.DefaultTLL {
			t.Fatalf("error")
		}
	})
}

// 测试异常场景
func TestCreateInstanceWithNoService(t *testing.T) {
	t.Run("无权限注册，可以捕获正常的错误", func(t *testing.T) {
		_, serviceResp := createCommonService(t, 900)
		defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
		var reqs []*api.Instance
		reqs = append(reqs, &api.Instance{
			Service:      serviceResp.Name,
			Namespace:    serviceResp.Namespace,
			ServiceToken: serviceResp.Token,
			Host:         utils.NewStringValue("1111"),
			Port:         utils.NewUInt32Value(0),
		})
		reqs = append(reqs, &api.Instance{
			Service:      serviceResp.Name,
			Namespace:    serviceResp.Namespace,
			ServiceToken: utils.NewStringValue("error token"),
			Host:         utils.NewStringValue("1111"),
			Port:         utils.NewUInt32Value(1),
		})
		resps := server.CreateInstances(defaultCtx, reqs)
		if respSuccess(resps) {
			t.Fatalf("error")
		}
		if resps.Responses[0].GetCode().GetValue() != api.ExecuteSuccess {
			t.Fatalf("error: %d", resps.Responses[0].GetCode().GetValue())
		}
		defer cleanInstance(resps.Responses[0].GetInstance().GetId().GetValue())
		if resps.Responses[1].GetCode().GetValue() != api.Unauthorized {
			t.Fatalf("error: %d", resps.Responses[0].GetCode().GetValue())
		}
	})
}

// 并发注册
func TestCreateInstance2(t *testing.T) {
	t.Run("并发注册，可以正常注册", func(t *testing.T) {
		var serviceResps []*api.Service
		for i := 0; i < 10; i++ {
			_, serviceResp := createCommonService(t, i)
			defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
			serviceResps = append(serviceResps, serviceResp)
		}

		time.Sleep(updateCacheInterval)
		total := 1024
		var wg sync.WaitGroup
		start := time.Now()
		errs := make(chan error)
		for i := 0; i < total; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				var req *api.Instance
				var resp *api.Instance
				req, resp = createCommonInstance(t, serviceResps[index%10], index)
				for c := 0; c < 10; c++ {
					if updateResp := server.UpdateInstance(defaultCtx, req); !respSuccess(updateResp) {
						errs <- fmt.Errorf("error: %+v", updateResp)
						return
					}
				}
				removeCommonInstance(t, serviceResps[index%10], resp.GetId().GetValue())
				cleanInstance(resp.GetId().GetValue())
			}(i)
		}

		go func() {
			wg.Wait()
			close(errs)
		}()

		for err := range errs {
			if err != nil {
				t.Fatal(err)
			}
		}
		t.Logf("consume: %v", time.Now().Sub(start))
	})
}

// 并发更新同一个实例
func TestUpdateInstanceManyTimes(t *testing.T) {
	_, serviceResp := createCommonService(t, 100)
	defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	instanceReq, instanceResp := createCommonInstance(t, serviceResp, 10)
	defer cleanInstance(instanceResp.GetId().GetValue())

	var wg sync.WaitGroup
	errs := make(chan error)
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			for c := 0; c < 16; c++ {
				instanceReq.Weight.Value = uint32(rand.Int() % 32767)
				if updateResp := server.UpdateInstance(defaultCtx, instanceReq); !respSuccess(updateResp) {
					errs <- fmt.Errorf("error: %+v", updateResp)
					return
				}
			}
		}(i)
	}
	go func() {
		wg.Wait()
		close(errs)
	}()

	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
}

// 测试获取实例
func TestGetInstances(t *testing.T) {
	t.Run("可以正常获取到实例信息", func(t *testing.T) {
		_ = server.Cache().Clear() // 为了防止影响，每个函数需要把缓存的内容清空
		_, serviceResp := createCommonService(t, 320)
		defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		time.Sleep(updateCacheInterval)
		instanceReq, instanceResp := createCommonInstance(t, serviceResp, 30)
		defer cleanInstance(instanceResp.GetId().GetValue())

		// 需要等待一会，等本地缓存更新
		time.Sleep(updateCacheInterval)

		req := &api.Service{
			Name:      utils.NewStringValue(instanceResp.GetService().GetValue()),
			Namespace: utils.NewStringValue(instanceResp.GetNamespace().GetValue()),
		}
		resp := server.ServiceInstancesCache(defaultCtx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}

		discoveryCheck(t, req, resp)

		if len(resp.Instances) != 1 {
			t.Fatalf("error : %d", len(resp.Instances))
		}

		instanceCheck(t, instanceReq, resp.GetInstances()[0])
		t.Logf("pass: %+v", resp.GetInstances()[0])
	})
	t.Run("注册实例，查询实例列表，实例反注册，revision会改变", func(t *testing.T) {
		_ = server.Cache().Clear() // 为了防止影响，每个函数需要把缓存的内容清空
		_, serviceResp := createCommonService(t, 100)
		defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		_, instanceResp := createCommonInstance(t, serviceResp, 90)
		defer cleanInstance(instanceResp.GetId().GetValue())

		time.Sleep(updateCacheInterval)
		resp := server.ServiceInstancesCache(defaultCtx, serviceResp)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		revision := resp.GetService().GetRevision()

		// 再注册一个实例，revision会改变
		_, instanceResp = createCommonInstance(t, serviceResp, 100)
		defer cleanInstance(instanceResp.GetId().GetValue())

		time.Sleep(updateCacheInterval)
		resp = server.ServiceInstancesCache(defaultCtx, serviceResp)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}

		if revision == resp.GetService().GetRevision() {
			t.Fatalf("error")
		}
		t.Logf("%s, %s", revision, resp.GetService().GetRevision())
	})
}

// 测试获取多个实例
func TestGetInstances1(t *testing.T) {
	discover := func(t *testing.T, service *api.Service, expectCount int) *api.DiscoverResponse {
		time.Sleep(updateCacheInterval)
		resp := server.ServiceInstancesCache(defaultCtx, service)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		discoveryCheck(t, service, resp)
		if len(resp.Instances) != expectCount {
			t.Fatalf("error : %d", len(resp.Instances))
		}
		return resp
	}
	t.Run("注册并反注册多个实例，可以正常获取", func(t *testing.T) {
		_ = server.Cache().Clear() // 为了防止影响，每个函数需要把缓存的内容清空
		_, serviceResp := createCommonService(t, 320)
		defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		var ids []string
		for i := 0; i < 10; i++ {
			_, instanceResp := createCommonInstance(t, serviceResp, i)
			ids = append(ids, instanceResp.GetId().GetValue())
			defer cleanInstance(instanceResp.GetId().GetValue())
		}
		discover(t, serviceResp, 10)

		// 反注册一部分
		for i := 1; i < 6; i++ {
			removeCommonInstance(t, serviceResp, ids[i])
		}
		discover(t, serviceResp, 5)
	})
	t.Run("传递revision， revision有变化则有数据，否则无数据返回", func(t *testing.T) {
		_ = server.Cache().Clear() // 为了防止影响，每个函数需要把缓存的内容清空
		_, serviceResp := createCommonService(t, 100)
		defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
		for i := 0; i < 5; i++ {
			_, instanceResp := createCommonInstance(t, serviceResp, i)
			defer cleanInstance(instanceResp.GetId().GetValue())
		}
		firstResp := discover(t, serviceResp, 5)

		serviceResp.Revision = firstResp.Service.GetRevision()
		if resp := server.ServiceInstancesCache(defaultCtx, serviceResp); !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		} else {
			if len(resp.Instances) != 0 {
				t.Fatalf("error: %d", len(resp.Instances))
			}
			t.Logf("%+v", resp)
		}

		// 多注册一个实例，revision发生改变
		_, instanceResp := createCommonInstance(t, serviceResp, 20)
		defer cleanInstance(instanceResp.GetId().GetValue())
		discover(t, serviceResp, 6)

	})
}

// 反注册测试
func TestRemoveInstance(t *testing.T) {
	_, serviceResp := createCommonService(t, 15)
	defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
	t.Run("实例创建完马上反注册，可以成功", func(t *testing.T) {
		_, instanceResp := createCommonInstance(t, serviceResp, 88)
		defer cleanInstance(instanceResp.GetId().GetValue())

		removeCommonInstance(t, serviceResp, instanceResp.GetId().GetValue())
		t.Logf("pass")
	})

	t.Run("注册完实例，反注册，再注册，可以成功", func(t *testing.T) {
		_, instanceResp := createCommonInstance(t, serviceResp, 888)
		defer cleanInstance(instanceResp.GetId().GetValue())

		removeCommonInstance(t, serviceResp, instanceResp.GetId().GetValue())

		time.Sleep(time.Second)
		_, instanceResp = createCommonInstance(t, serviceResp, 888)
		defer cleanInstance(instanceResp.GetId().GetValue())
		t.Logf("pass")
	})
	t.Run("重复反注册，返回成功", func(t *testing.T) {
		_, instanceResp := createCommonInstance(t, serviceResp, 999)
		defer cleanInstance(instanceResp.GetId().GetValue())

		removeCommonInstance(t, serviceResp, instanceResp.GetId().GetValue())
		time.Sleep(time.Second)
		removeCommonInstance(t, serviceResp, instanceResp.GetId().GetValue())
	})
}

// 测试从数据库拉取实例信息
func TestListInstances(t *testing.T) {
	t.Run("list实例列表，返回的数据字段都存在", func(t *testing.T) {
		_, serviceResp := createCommonService(t, 1156)
		defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		instanceReq, instanceResp := createCommonInstance(t, serviceResp, 200)
		defer cleanInstance(instanceResp.GetId().GetValue())

		query := map[string]string{"offset": "0", "limit": "100"}
		query["host"] = instanceReq.GetHost().GetValue()
		query["port"] = strconv.FormatUint(uint64(instanceReq.GetPort().GetValue()), 10)
		resp := server.GetInstances(query)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if len(resp.Instances) != 1 {
			t.Fatalf("error: %d", len(resp.Instances))
		}

		instanceCheck(t, instanceReq, resp.Instances[0])
	})
	t.Run("list实例列表，offset和limit能正常工作", func(t *testing.T) {
		_, serviceResp := createCommonService(t, 115)
		defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		time.Sleep(updateCacheInterval)
		total := 50
		for i := 0; i < total; i++ {
			_, instanceResp := createCommonInstance(t, serviceResp, i+1)
			defer cleanInstance(instanceResp.GetId().GetValue())
		}

		query := map[string]string{"offset": "10", "limit": "20", "host": "127.0.0.1"}
		resp := server.GetInstances(query)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}

		if len(resp.Instances) == 20 {
			t.Logf("pass")
		}
	})

	t.Run("list实例列表，可以进行正常字段过滤", func(t *testing.T) {
		// 先任意找几个实例字段过滤
		_, serviceResp := createCommonService(t, 200)
		defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		time.Sleep(updateCacheInterval)
		total := 10
		instance := new(api.Instance)
		for i := 0; i < total; i++ {
			_, instanceResp := createCommonInstance(t, serviceResp, i+1)
			defer cleanInstance(instanceResp.GetId().GetValue())
			instance = instanceResp
		}

		host := instance.GetHost().GetValue()
		port := strconv.FormatUint(uint64(instance.GetPort().GetValue()), 10)
		query := map[string]string{"limit": "20", "host": host, "port": port}
		resp := server.GetInstances(query)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if len(resp.Instances) == 1 {
			t.Logf("pass")
		}
	})
}

// 测试list实例列表
func TestListInstances1(t *testing.T) {
	// 先任意找几个实例字段过滤
	_, serviceResp := createCommonService(t, 800)
	defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	checkAmountAndSize := func(t *testing.T, resp *api.BatchQueryResponse, expect int, size int) {
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if resp.GetAmount().GetValue() != uint32(expect) {
			t.Fatalf("error: %d", resp.GetAmount().GetValue())
		}
		if len(resp.Instances) != size {
			t.Fatalf("error: %d", len(resp.Instances))
		}
	}

	t.Run("list实例，使用service和namespace过滤", func(t *testing.T) {
		total := 102
		for i := 0; i < total; i++ {
			_, instanceResp := createCommonInstance(t, serviceResp, i+2)
			defer cleanInstance(instanceResp.GetId().GetValue())
		}
		query := map[string]string{
			"offset":    "0",
			"limit":     "100",
			"service":   serviceResp.GetName().GetValue(),
			"namespace": serviceResp.GetNamespace().GetValue(),
		}

		resp := server.GetInstances(query)
		checkAmountAndSize(t, resp, total, 100)
	})

	t.Run("list实例，先删除实例，再查询会过滤删除的", func(t *testing.T) {
		total := 50
		for i := 0; i < total; i++ {
			_, instanceResp := createCommonInstance(t, serviceResp, i+2)
			defer cleanInstance(instanceResp.GetId().GetValue())
			if i%2 == 0 {
				removeCommonInstance(t, serviceResp, instanceResp.GetId().GetValue())
			}
		}

		query := map[string]string{
			"service":   serviceResp.GetName().GetValue(),
			"namespace": serviceResp.GetNamespace().GetValue(),
		}
		resp := server.GetInstances(query)
		checkAmountAndSize(t, resp, total/2, total/2)

	})
	t.Run("true和false测试", func(t *testing.T) {
		_, instanceResp := createCommonInstance(t, serviceResp, 10)
		defer cleanInstance(instanceResp.GetId().GetValue())

		query := map[string]string{
			"service":   serviceResp.GetName().GetValue(),
			"namespace": serviceResp.GetNamespace().GetValue(),
			"isolate":   "false",
			"healthy":   "false",
		}
		checkAmountAndSize(t, server.GetInstances(query), 1, 1)

		query["isolate"] = "true"
		checkAmountAndSize(t, server.GetInstances(query), 0, 0)

		query["isolate"] = "false"
		query["healthy"] = "true"
		checkAmountAndSize(t, server.GetInstances(query), 0, 0)

		query["isolate"] = "0"
		query["healthy"] = "0"
		checkAmountAndSize(t, server.GetInstances(query), 1, 1)

		query["health_status"] = "1"
		checkAmountAndSize(t, server.GetInstances(query), 1, 1)

		query["health_status"] = "0"
		delete(query, "healthy")
		checkAmountAndSize(t, server.GetInstances(query), 1, 1)

		query["health_status"] = "1"
		checkAmountAndSize(t, server.GetInstances(query), 0, 0)
	})
	t.Run("metadata条件测试", func(t *testing.T) {
		_, instanceResp1 := createCommonInstance(t, serviceResp, 10)
		defer cleanInstance(instanceResp1.GetId().GetValue())
		_, instanceResp2 := createCommonInstance(t, serviceResp, 20)
		defer cleanInstance(instanceResp2.GetId().GetValue())
		//只返回第一个实例的查询
		query := map[string]string{
			"service":   serviceResp.GetName().GetValue(),
			"namespace": serviceResp.GetNamespace().GetValue(),
			"keys":      "internal-personal-xxx",
			"values":    "internal-personal-xxx_10",
		}
		checkAmountAndSize(t, server.GetInstances(query), 1, 1)
		//使用共同的元数据查询，返回两个实例
		query = map[string]string{
			"service":   serviceResp.GetName().GetValue(),
			"namespace": serviceResp.GetNamespace().GetValue(),
			"keys":      "my-meta-a1",
			"values":    "1111",
		}
		checkAmountAndSize(t, server.GetInstances(query), 2, 2)
		//使用不存在的元数据查询，返回零个实例
		query = map[string]string{
			"service":   serviceResp.GetName().GetValue(),
			"namespace": serviceResp.GetNamespace().GetValue(),
			"keys":      "nokey",
			"values":    "novalue",
		}
		checkAmountAndSize(t, server.GetInstances(query), 0, 0)
	})
	t.Run("metadata只有key或者value，返回错误", func(t *testing.T) {
		query := map[string]string{
			"service":   serviceResp.GetName().GetValue(),
			"namespace": serviceResp.GetNamespace().GetValue(),
			"keys":      "internal-personal-xxx",
		}
		resp := server.GetInstances(query)
		if resp.GetCode().GetValue() != api.InvalidQueryInsParameter {
			t.Fatalf("resp is %v, not InvalidQueryInsParameter", resp)
		}
		query = map[string]string{
			"service":   serviceResp.GetName().GetValue(),
			"namespace": serviceResp.GetNamespace().GetValue(),
			"values":    "internal-personal-xxx",
		}
		resp = server.GetInstances(query)
		if resp.GetCode().GetValue() != api.InvalidQueryInsParameter {
			t.Fatalf("resp is %v, not InvalidQueryInsParameter", resp)
		}
	})
}

// 测试地域获取
func TestInstancesContainLocation(t *testing.T) {
	locationCheck := func(lhs *api.Location, rhs *api.Location) {
		if lhs.GetRegion().GetValue() != rhs.GetRegion().GetValue() {
			t.Fatalf("error: %v, %v", lhs, rhs)
		}
		if lhs.GetZone().GetValue() != rhs.GetZone().GetValue() {
			t.Fatalf("error: %v, %v", lhs, rhs)
		}
		if lhs.GetCampus().GetValue() != rhs.GetCampus().GetValue() {
			t.Fatalf("error: %v, %v", lhs, rhs)
		}
	}

	_, service := createCommonService(t, 123)
	defer cleanServiceName(service.GetName().GetValue(), service.GetNamespace().GetValue())

	instance := &api.Instance{
		Service:      service.GetName(),
		Namespace:    service.GetNamespace(),
		ServiceToken: service.GetToken(),
		Host:         utils.NewStringValue("123456"),
		Port:         utils.NewUInt32Value(9090),
		Location: &api.Location{
			Region: utils.NewStringValue("region1"),
			Zone:   utils.NewStringValue("zone1"),
			Campus: utils.NewStringValue("campus1"),
		},
	}
	resp := server.CreateInstance(defaultCtx, instance)
	if !respSuccess(resp) {
		t.Fatalf("error: %+v", resp)
	}
	defer cleanInstance(resp.GetInstance().GetId().GetValue())

	getResp := server.GetInstances(map[string]string{
		"service": instance.GetService().GetValue(), "namespace": instance.GetNamespace().GetValue(),
	})
	if !respSuccess(getResp) {
		t.Fatalf("error: %+v", getResp)
	}
	getInstances := getResp.GetInstances()
	if len(getInstances) != 1 {
		t.Fatalf("error: %d", len(getInstances))
	}
	t.Logf("%v", getInstances[0])
	locationCheck(instance.GetLocation(), getInstances[0].GetLocation())

	time.Sleep(updateCacheInterval)
	discoverResp := server.ServiceInstancesCache(defaultCtx, service)
	if len(discoverResp.GetInstances()) != 1 {
		t.Fatalf("error: %d", len(discoverResp.GetInstances()))
	}
	t.Logf("%v", discoverResp.GetInstances()[0])
	locationCheck(instance.GetLocation(), discoverResp.GetInstances()[0].GetLocation())
}

// 测试实例更新
func TestUpdateInstance(t *testing.T) {
	_, serviceResp := createCommonService(t, 123)
	defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	instanceReq, instanceResp := createCommonInstance(t, serviceResp, 22)
	defer cleanInstance(instanceResp.GetId().GetValue())
	t.Run("更新实例，所有属性都可以生效", func(t *testing.T) {
		// update
		instanceReq.Protocol = utils.NewStringValue("update-protocol")
		instanceReq.Version = utils.NewStringValue("update-version")
		instanceReq.Priority = utils.NewUInt32Value(30)
		instanceReq.Weight = utils.NewUInt32Value(500)
		instanceReq.Healthy = utils.NewBoolValue(false)
		instanceReq.Isolate = utils.NewBoolValue(true)
		instanceReq.LogicSet = utils.NewStringValue("update-logic-set")
		instanceReq.HealthCheck = &api.HealthCheck{
			Type: api.HealthCheck_HEARTBEAT,
			Heartbeat: &api.HeartbeatHealthCheck{
				Ttl: utils.NewUInt32Value(6),
			},
		}
		instanceReq.Metadata = map[string]string{
			"internal-personal-xxx": fmt.Sprintf("internal-personal-xxx_2412323"),
			"tencent":               "1111",
			"yyyy":                  "2222",
		}
		instanceReq.ServiceToken = serviceResp.Token

		if resp := server.UpdateInstance(defaultCtx, instanceReq); !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}

		// 查询数据
		query := map[string]string{
			"host": instanceReq.GetHost().GetValue(),
			"port": strconv.FormatUint(uint64(instanceReq.GetPort().GetValue()), 10),
		}
		resp := server.GetInstances(query)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if len(resp.GetInstances()) != 1 {
			t.Fatalf("error: %d", len(resp.GetInstances()))
		}

		instanceReq.Service = instanceResp.Service
		instanceReq.Namespace = instanceResp.Namespace
		instanceCheck(t, instanceReq, resp.Instances[0])
	})
	t.Run("实例只更新metadata，revision也会发生改变", func(t *testing.T) {
		instanceReq.Metadata = map[string]string{
			"new-metadata": "new-value",
		}

		serviceName := serviceResp.GetName().GetValue()
		namespaceName := serviceResp.GetNamespace().GetValue()
		firstInstances := getInstancesWithService(t, serviceName, namespaceName, 1)

		if resp := server.UpdateInstance(defaultCtx, instanceReq); !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}

		secondInstances := getInstancesWithService(t, serviceName, namespaceName, 1)
		if firstInstances[0].GetRevision().GetValue() != secondInstances[0].GetRevision().GetValue() {
			t.Logf("pass %s, %s",
				firstInstances[0].GetRevision().GetValue(), secondInstances[0].GetRevision().GetValue())
		} else {
			t.Fatalf("error")
		}

		instanceCheck(t, instanceReq, secondInstances[0])
	})
	t.Run("metadata太长，update会报错", func(t *testing.T) {
		instanceReq.Metadata = make(map[string]string)
		for i := 0; i < naming.MaxMetadataLength+1; i++ {
			instanceReq.Metadata[fmt.Sprintf("%d", i)] = "a"
		}
		if resp := server.UpdateInstance(defaultCtx, instanceReq); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error")
		}
	})
}

/**
 * @brief 根据ip修改隔离状态
 */
func TestUpdateIsolate(t *testing.T) {
	_, serviceResp := createCommonService(t, 111)
	defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
	t.Run("修改超过100个实例的隔离状态", func(t *testing.T) {
		instancesReq := make([]*api.Instance, 0, 210)
		for i := 0; i < 210; i++ {
			instanceReq := &api.Instance{
				ServiceToken: utils.NewStringValue(serviceResp.GetToken().GetValue()),
				Service:      utils.NewStringValue(serviceResp.GetName().GetValue()),
				Namespace:    utils.NewStringValue(serviceResp.GetNamespace().GetValue()),
				Host:         utils.NewStringValue("127.0.0.1"),
				Port:         utils.NewUInt32Value(uint32(i)),
			}
			resp := server.CreateInstance(defaultCtx, instanceReq)
			if !respSuccess(resp) {
				t.Fatalf("error: %s", resp.GetInfo().GetValue())
			}
			instancesReq = append(instancesReq, instanceReq)
			defer cleanInstance(resp.GetInstance().GetId().GetValue())
		}
		req := &api.Instance{
			ServiceToken: utils.NewStringValue(serviceResp.GetToken().GetValue()),
			Service:      utils.NewStringValue(serviceResp.GetName().GetValue()),
			Namespace:    utils.NewStringValue(serviceResp.GetNamespace().GetValue()),
			Host:         utils.NewStringValue("127.0.0.1"),
			Isolate:      utils.NewBoolValue(true),
		}
		if resp := server.UpdateInstanceIsolate(defaultCtx, req); !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("根据ip修改隔离状态", func(t *testing.T) {
		instanceNum := 20
		portNum := 2
		revisions := make(map[string]string, instanceNum)
		instancesReq := make([]*api.Instance, 0, instanceNum)
		for i := 0; i < instanceNum/portNum; i++ {
			for j := 1; j <= portNum; j++ {
				instanceReq := &api.Instance{
					ServiceToken: utils.NewStringValue(serviceResp.GetToken().GetValue()),
					Service:      utils.NewStringValue(serviceResp.GetName().GetValue()),
					Namespace:    utils.NewStringValue(serviceResp.GetNamespace().GetValue()),
					Host:         utils.NewStringValue(fmt.Sprintf("%d.%d.%d.%d", i, i, i, i)),
					Port:         utils.NewUInt32Value(uint32(j)),
					Isolate:      utils.NewBoolValue(false),
					Healthy:      utils.NewBoolValue(true),
					Metadata: map[string]string{
						"internal-personal-xxx": fmt.Sprintf("internal-personal-xxx_%d", i),
					},
				}
				resp := server.CreateInstance(defaultCtx, instanceReq)
				if !respSuccess(resp) {
					t.Fatalf("error: %s", resp.GetInfo().GetValue())
				}
				instanceReq.Isolate = utils.NewBoolValue(true)
				instancesReq = append(instancesReq, instanceReq)
				revisions[resp.GetInstance().GetId().GetValue()] = resp.GetInstance().GetRevision().GetValue()
				defer cleanInstance(resp.GetInstance().GetId().GetValue())
			}
		}

		if resp := server.UpdateInstancesIsolate(defaultCtx, instancesReq); !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}

		// 检查隔离状态和revision是否改变
		for i := 0; i < instanceNum/portNum; i++ {
			filter := map[string]string{
				"service":   serviceResp.GetName().GetValue(),
				"namespace": serviceResp.GetNamespace().GetValue(),
				"host":      fmt.Sprintf("%d.%d.%d.%d", i, i, i, i),
			}

			resp := server.GetInstances(filter)
			if !respSuccess(resp) {
				t.Fatalf("error: %s", resp.GetInfo().GetValue())
			}

			if len(resp.GetInstances()) != portNum {
				t.Fatalf("error: %d", len(resp.GetInstances()))
			}

			actualInstances := resp.GetInstances()
			for _, instance := range actualInstances {
				if !instance.GetIsolate().GetValue() ||
					instance.GetRevision().GetValue() == revisions[instance.GetId().GetValue()] {
					t.Fatalf("error instance is %+v", instance)
				}
			}
		}
		t.Log("pass")
	})

	t.Run("并发更新", func(t *testing.T) {
		instanceReq, instanceResp := createCommonInstance(t, serviceResp, 123)
		defer cleanInstance(instanceResp.GetId().GetValue())

		var wg sync.WaitGroup
		errs := make(chan error)
		for i := 0; i < 64; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				for c := 0; c < 16; c++ {
					instanceReq.Isolate = utils.NewBoolValue(true)
					if resp := server.UpdateInstanceIsolate(defaultCtx, instanceReq); !respSuccess(resp) {
						errs <- fmt.Errorf("error: %+v", resp)
						return
					}
				}
			}(i)
		}
		go func() {
			wg.Wait()
			close(errs)
		}()

		for err := range errs {
			if err != nil {
				t.Fatal(err)
			}
		}
		t.Log("pass")
	})

	t.Run("若隔离状态相同，则不需要更新", func(t *testing.T) {
		instanceReq, instanceResp := createCommonInstance(t, serviceResp, 456)
		defer cleanInstance(instanceResp.GetId().GetValue())

		resp := server.UpdateInstanceIsolate(defaultCtx, instanceReq)
		if resp.GetCode().GetValue() == api.NoNeedUpdate {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})
}

/**
 * @brief 根据ip删除服务实例
 */
func TestDeleteInstanceByHost(t *testing.T) {
	_, serviceResp := createCommonService(t, 222)
	defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	t.Run("根据ip删除服务实例", func(t *testing.T) {
		instanceNum := 20
		portNum := 2
		instancesReq := make([]*api.Instance, 0, instanceNum)
		for i := 0; i < instanceNum/portNum; i++ {
			for j := 1; j <= portNum; j++ {
				instanceReq := &api.Instance{
					ServiceToken: utils.NewStringValue(serviceResp.GetToken().GetValue()),
					Service:      utils.NewStringValue(serviceResp.GetName().GetValue()),
					Namespace:    utils.NewStringValue(serviceResp.GetNamespace().GetValue()),
					Host:         utils.NewStringValue(fmt.Sprintf("%d.%d.%d.%d", i, i, i, i)),
					Port:         utils.NewUInt32Value(uint32(j)),
				}
				resp := server.CreateInstance(defaultCtx, instanceReq)
				if !respSuccess(resp) {
					t.Fatalf("error: %s", resp.GetInfo().GetValue())
				}
				instancesReq = append(instancesReq, instanceReq)
				defer cleanInstance(resp.GetInstance().GetId().GetValue())
			}
		}

		if resp := server.DeleteInstancesByHost(defaultCtx, instancesReq); !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}

		// 检查隔离状态和revision是否改变
		getInstancesWithService(t,
			serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(), 0)
		t.Log("pass")
	})

	t.Run("删除超过100个实例", func(t *testing.T) {
		instancesReq := make([]*api.Instance, 0, 210)
		for i := 0; i < 210; i++ {
			instanceReq := &api.Instance{
				ServiceToken: utils.NewStringValue(serviceResp.GetToken().GetValue()),
				Service:      utils.NewStringValue(serviceResp.GetName().GetValue()),
				Namespace:    utils.NewStringValue(serviceResp.GetNamespace().GetValue()),
				Host:         utils.NewStringValue("127.0.0.2"),
				Port:         utils.NewUInt32Value(uint32(i)),
			}
			resp := server.CreateInstance(defaultCtx, instanceReq)
			if !respSuccess(resp) {
				t.Fatalf("error: %s", resp.GetInfo().GetValue())
			}
			instancesReq = append(instancesReq, instanceReq)
			defer cleanInstance(resp.GetInstance().GetId().GetValue())
		}
		req := &api.Instance{
			ServiceToken: utils.NewStringValue(serviceResp.GetToken().GetValue()),
			Service:      utils.NewStringValue(serviceResp.GetName().GetValue()),
			Namespace:    utils.NewStringValue(serviceResp.GetNamespace().GetValue()),
			Host:         utils.NewStringValue("127.0.0.1"),
			Isolate:      utils.NewBoolValue(true),
		}
		if resp := server.DeleteInstanceByHost(defaultCtx, req); !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})
}

// 测试enable_health_check
func TestUpdateHealthCheck(t *testing.T) {
	getAndCheck := func(t *testing.T, req *api.Instance) {
		query := map[string]string{
			"host": req.GetHost().GetValue(),
			"port": strconv.FormatUint(uint64(req.GetPort().GetValue()), 10),
		}
		resp := server.GetInstances(query)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if len(resp.GetInstances()) != 1 {
			t.Fatalf("error: %d", len(resp.GetInstances()))
		}
		t.Logf("%+v", resp.Instances[0])

		instanceCheck(t, req, resp.Instances[0])
	}
	_, serviceResp := createCommonService(t, 321)
	defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	instanceReq, instanceResp := createCommonInstance(t, serviceResp, 10)
	defer cleanInstance(instanceResp.GetId().GetValue())
	instanceReq.ServiceToken = serviceResp.Token
	t.Run("health_check可以随意关闭", func(t *testing.T) {
		// 打开 -> 打开
		instanceReq.Weight = utils.NewUInt32Value(300)
		if resp := server.UpdateInstance(defaultCtx, instanceReq); !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		getAndCheck(t, instanceReq)

		// 打开-> 关闭
		instanceReq.EnableHealthCheck = utils.NewBoolValue(false)
		if resp := server.UpdateInstance(defaultCtx, instanceReq); !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		instanceReq.HealthCheck = nil
		getAndCheck(t, instanceReq)

		// 关闭 -> 关闭
		instanceReq.Weight = utils.NewUInt32Value(200)
		if resp := server.UpdateInstance(defaultCtx, instanceReq); !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		getAndCheck(t, instanceReq)

		// 关闭 -> 打开
		instanceReq.EnableHealthCheck = utils.NewBoolValue(true)
		instanceReq.HealthCheck = &api.HealthCheck{
			Type: api.HealthCheck_HEARTBEAT,
			Heartbeat: &api.HeartbeatHealthCheck{
				Ttl: utils.NewUInt32Value(8),
			},
		}
		if resp := server.UpdateInstance(defaultCtx, instanceReq); !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		getAndCheck(t, instanceReq)
	})
	t.Run("healthcheck为空的异常测试", func(t *testing.T) {
		instanceReq.HealthCheck = &api.HealthCheck{
			Heartbeat: &api.HeartbeatHealthCheck{
				Ttl: utils.NewUInt32Value(0),
			},
		}
		if resp := server.UpdateInstance(defaultCtx, instanceReq); !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		instanceReq.HealthCheck = &api.HealthCheck{
			Type: api.HealthCheck_HEARTBEAT,
			Heartbeat: &api.HeartbeatHealthCheck{
				Ttl: utils.NewUInt32Value(naming.DefaultTLL),
			},
		}
		getAndCheck(t, instanceReq)
	})
}

// 测试删除实例
func TestDeleteInstance(t *testing.T) {
	_, serviceResp := createCommonService(t, 123)
	defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	getInstance := func(t *testing.T, s *api.Service, expect int) []*api.Instance {
		filters := map[string]string{"service": s.GetName().GetValue(), "namespace": s.GetNamespace().GetValue()}
		getResp := server.GetInstances(filters)
		if !respSuccess(getResp) {
			t.Fatalf("error")
		}
		if len(getResp.GetInstances()) != expect {
			t.Fatalf("error")
		}
		return getResp.GetInstances()
	}

	t.Run("可以通过ID删除实例", func(t *testing.T) {
		_, instanceResp := createCommonInstance(t, serviceResp, 10)
		defer cleanInstance(instanceResp.GetId().GetValue())
		removeCommonInstance(t, serviceResp, instanceResp.GetId().GetValue())

		getInstance(t, serviceResp, 0)
	})
	t.Run("可以通过四元组删除实例", func(t *testing.T) {
		req := &api.Instance{
			ServiceToken: serviceResp.GetToken(),
			Service:      serviceResp.GetName(),
			Namespace:    serviceResp.GetNamespace(),
			Host:         utils.NewStringValue("abc"),
			Port:         utils.NewUInt32Value(8080),
		}
		resp := server.CreateInstance(defaultCtx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		instanceResp := resp.GetInstance()
		t.Logf("%+v", getInstance(t, serviceResp, 1))
		defer cleanInstance(instanceResp.GetId().GetValue())

		removeInstanceWithAttrs(t, serviceResp, instanceResp)
		getInstance(t, serviceResp, 0)
	})
	t.Run("可以通过五元组删除实例", func(t *testing.T) {
		_, instanceResp := createCommonInstance(t, serviceResp, 55)
		defer cleanInstance(instanceResp.GetId().GetValue())

		removeInstanceWithAttrs(t, serviceResp, instanceResp)
		getInstance(t, serviceResp, 0)
	})
}

// 批量创建服务实例
// 步骤：
// 1. n个服务，每个服务m个服务实例
// 2. n个协程同时发请求
func TestBatchCreateInstances(t *testing.T) {
	Convey("批量创建服务", t, func() {
		n := 32
		m := 128
		var services []*api.Service
		for i := 0; i < n; i++ {
			_, service := createCommonService(t, i)
			services = append(services, service)
		}
		defer cleanServices(services)

		var wg sync.WaitGroup
		idCh := make(chan string, n*m)
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				for j := 0; j < m; j++ {
					_, instance := createCommonInstance(t, services[index], j)
					idCh <- instance.GetId().GetValue()
				}
			}(i)
		}

		var deleteCount int32
		for i := 0; i < n; i++ {
			go func() {
				for {
					select {
					case id := <-idCh:
						cleanInstance(id)
						atomic.AddInt32(&deleteCount, 1)
					}
				}
			}()
		}

		wg.Wait()
		for {
			count := atomic.LoadInt32(&deleteCount)
			if count == int32(n*m) {
				return
			}
			t.Logf("%d", count)
			time.Sleep(time.Second * 1)
		}

	})
}

// 测试批量接口返回的顺序
func TestCreateInstancesOrder(t *testing.T) {
	t.Run("测试批量接口返回的顺序与发送的数据一致", func(t *testing.T) {
		_, service := createCommonService(t, 123)
		defer cleanServiceName(service.GetName().GetValue(), service.GetNamespace().GetValue())
		var instances []*api.Instance
		for j := 0; j < 10; j++ {
			instances = append(instances, &api.Instance{
				Service:      service.GetName(),
				Namespace:    service.GetNamespace(),
				ServiceToken: service.GetToken(),
				Host:         utils.NewStringValue("a.b.c.d"),
				Port:         utils.NewUInt32Value(uint32(j)),
			})
		}

		resps := server.CreateInstances(defaultCtx, instances)
		if !respSuccess(resps) {
			t.Fatalf("error: %+v", resps)
		}
		for i, resp := range resps.GetResponses() {
			if resp.GetInstance().GetPort().GetValue() != instances[i].GetPort().GetValue() {
				t.Fatalf("error")
			}
			cleanInstance(resp.GetInstance().GetId().GetValue())
		}
	})
}

// 测试批量删除实例
func TestBatchDeleteInstances(t *testing.T) {
	_, service := createCommonService(t, 234)
	defer cleanServiceName(service.GetName().GetValue(), service.GetNamespace().GetValue())
	createInstances := func(t *testing.T) ([]*api.Instance, *api.BatchWriteResponse) {
		var instances []*api.Instance
		for j := 0; j < 100; j++ {
			instances = append(instances, &api.Instance{
				Service:      service.GetName(),
				Namespace:    service.GetNamespace(),
				ServiceToken: service.GetToken(),
				Host:         utils.NewStringValue("a.b.c.d"),
				Port:         utils.NewUInt32Value(uint32(j)),
			})
		}
		resps := server.CreateInstances(defaultCtx, instances)
		if !respSuccess(resps) {
			t.Fatalf("error: %+v", resps)
		}
		return instances, resps
	}
	t.Run("测试batch删除实例，单个接口", func(t *testing.T) {
		_, resps := createInstances(t)
		var wg sync.WaitGroup
		errs := make(chan error)
		for _, resp := range resps.GetResponses() {
			wg.Add(1)
			go func(instance *api.Instance) {
				defer func() {
					cleanInstance(instance.GetId().GetValue())
					wg.Done()
				}()
				req := &api.Instance{Id: instance.Id, ServiceToken: service.Token}
				if out := server.DeleteInstance(defaultCtx, req); !respSuccess(out) {
					errs <- fmt.Errorf("error: %+v", out)
					return
				}
			}(resp.GetInstance())
		}
		go func() {
			wg.Wait()
			close(errs)
		}()

		for err := range errs {
			if err != nil {
				t.Fatal(err)
			}
		}
	})
	t.Run("测试batch删除实例，批量接口", func(t *testing.T) {
		instances, instancesResp := createInstances(t)
		// 删除body的token，测试header的token是否可行
		for _, instance := range instances {
			instance.ServiceToken = nil
			instance.Id = nil
		}
		ctx := context.WithValue(defaultCtx, utils.StringContext("polaris-token"), service.GetToken().GetValue())
		if out := server.DeleteInstances(ctx, instances); !respSuccess(out) {
			t.Fatalf("error: %+v", out)
		} else {
			t.Logf("%+v", out)
		}
		resps := server.GetInstances(map[string]string{
			"service":   service.GetName().GetValue(),
			"namespace": service.GetNamespace().GetValue(),
		})
		if !respSuccess(resps) {
			t.Fatalf("error: %+v", resps)
		}
		if len(resps.GetInstances()) != 0 {
			t.Fatalf("error : %d", len(resps.GetInstances()))
		}
		for _, entry := range instancesResp.GetResponses() {
			cleanInstance(entry.GetInstance().GetId().GetValue())
		}
	})
}

// 验证成功创建和删除实例的response
func TestInstanceResponse(t *testing.T) {
	_, service := createCommonService(t, 234)
	defer cleanServiceName(service.GetName().GetValue(), service.GetNamespace().GetValue())
	create := func() (*api.Instance, *api.Instance) {
		ins := &api.Instance{
			Service:      service.GetName(),
			Namespace:    service.GetNamespace(),
			ServiceToken: service.GetToken(),
			Host:         utils.NewStringValue("a.b.c.d"),
			Port:         utils.NewUInt32Value(uint32(100)),
		}
		resps := server.CreateInstances(defaultCtx, []*api.Instance{ins})
		if !respSuccess(resps) {
			t.Fatalf("error: %+v", resps)
		}
		return ins, resps.Responses[0].GetInstance()
	}
	t.Run("创建实例，返回的信息不能包括token，包括id", func(t *testing.T) {
		ins, respIns := create()
		defer cleanInstance(respIns.GetId().GetValue())
		t.Logf("%+v", respIns)
		if respIns.GetService().GetValue() != ins.GetService().GetValue() ||
			respIns.GetNamespace().GetValue() != ins.GetNamespace().GetValue() ||
			respIns.GetHost().GetValue() != ins.GetHost().GetValue() ||
			respIns.GetPort().GetValue() != ins.GetPort().GetValue() ||
			respIns.GetId().GetValue() == "" || respIns.GetServiceToken().GetValue() != "" {
			t.Fatalf("error")
		}
	})
	t.Run("删除实例，返回的信息包括req，不增加信息", func(t *testing.T) {
		req, resp := create()
		defer cleanInstance(resp.GetId().GetValue())
		time.Sleep(time.Second)
		resps := server.DeleteInstances(defaultCtx, []*api.Instance{req})
		if !respSuccess(resps) {
			t.Fatalf("error: %+v", resps)
		}
		respIns := resps.GetResponses()[0].GetInstance()
		if respIns.GetId().GetValue() != "" || respIns.GetService() != req.GetService() ||
			respIns.GetNamespace() != req.GetNamespace() || respIns.GetHost() != req.GetHost() ||
			respIns.GetPort() != req.GetPort() || respIns.GetServiceToken() != req.GetServiceToken() {
			t.Fatalf("error")
		}
		t.Logf("pass")
	})
}

// 测试实例创建与删除的异常场景2
func TestCreateInstancesBadCase2(t *testing.T) {
	_, service := createCommonService(t, 123)
	defer cleanServiceName(service.GetName().GetValue(), service.GetNamespace().GetValue())
	t.Run("重复多个一样的实例注册，其中一个成功，其他的失败", func(t *testing.T) {
		time.Sleep(time.Second)
		var instances []*api.Instance
		for j := 0; j < 3; j++ {
			instances = append(instances, &api.Instance{
				Service:      service.GetName(),
				Namespace:    service.GetNamespace(),
				ServiceToken: service.GetToken(),
				Host:         utils.NewStringValue("a.b.c.d"),
				Port:         utils.NewUInt32Value(uint32(100)),
			})
		}

		resps := server.CreateInstances(defaultCtx, instances)
		t.Logf("%+v", resps)
		if respSuccess(resps) {
			t.Fatalf("error: %+v", resps)
		}
		for _, resp := range resps.GetResponses() {
			if resp.GetInstance().GetId().GetValue() != "" {
				cleanInstance(resp.GetInstance().GetId().GetValue())
			}
		}
	})
	t.Run("重复发送同样实例的反注册请求，可以正常返回，一个成功，其他的失败", func(t *testing.T) {
		time.Sleep(time.Second)
		instance := &api.Instance{
			Service:      service.GetName(),
			Namespace:    service.GetNamespace(),
			ServiceToken: service.GetToken(),
			Host:         utils.NewStringValue("a.b.c.d"),
			Port:         utils.NewUInt32Value(uint32(100)),
		}
		resps := server.CreateInstances(defaultCtx, []*api.Instance{instance})
		if !respSuccess(resps) {
			t.Fatalf("error: %+v", resps)
		}
		defer cleanInstance(resps.Responses[0].Instance.GetId().GetValue())

		delReqs := make([]*api.Instance, 0, 10)
		for i := 0; i < 2; i++ {
			delReqs = append(delReqs, &api.Instance{
				Id:           resps.Responses[0].Instance.GetId(),
				ServiceToken: service.GetToken(),
			})
		}
		time.Sleep(time.Second)
		resps = server.DeleteInstances(defaultCtx, delReqs)
		if respSuccess(resps) {
			t.Fatalf("error: %s", resps)
		}
		for _, resp := range resps.GetResponses() {
			if resp.GetCode().GetValue() != api.ExecuteSuccess &&
				resp.GetCode().GetValue() != api.SameInstanceRequest {
				t.Fatalf("error: %+v", resp)
			}
		}
	})
}

// 测试实例创建和删除的流量限制
func TestInstanceRatelimit(t *testing.T) {
	Convey("超过ratelimit，返回错误", t, func() {
		_, serviceResp := createCommonService(t, 100)
		defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		instanceReq, instanceResp := createCommonInstance(t, serviceResp, 110)
		server.CreateInstance(defaultCtx, instanceReq)
		defer cleanInstance(instanceResp.GetId().GetValue())
		for i := 0; i < 10; i++ {
			resp := server.CreateInstance(defaultCtx, instanceReq)
			So(resp.GetCode().GetValue(), ShouldEqual, api.InstanceTooManyRequests)
		}
		time.Sleep(time.Second)
		resp := server.CreateInstance(defaultCtx, instanceReq)
		So(resp.GetCode().GetValue(), ShouldEqual, api.ExistedResource)
	})
}

// 测试instance，no need update
func TestInstanceNoNeedUpdate(t *testing.T) {
	_, serviceResp := createCommonService(t, 222)
	defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
	instanceReq, instanceResp := createCommonInstance(t, serviceResp, 222)
	defer cleanInstance(instanceResp.GetId().GetValue())
	Convey("instance没有变更，不需要更新", t, func() {
		resp := server.UpdateInstance(defaultCtx, instanceReq)
		So(resp.GetCode().GetValue(), ShouldEqual, api.NoNeedUpdate)
	})
	Convey("metadata为空，不需要更新", t, func() {
		oldMeta := instanceReq.GetMetadata()
		instanceReq.Metadata = nil
		defer func() { instanceReq.Metadata = oldMeta }()
		resp := server.UpdateInstance(defaultCtx, instanceReq)
		So(resp.GetCode().GetValue(), ShouldEqual, api.NoNeedUpdate)
	})
	Convey("healthCheck为nil，不需要更新", t, func() {
		oldHealthCheck := instanceReq.GetHealthCheck()
		instanceReq.HealthCheck = nil
		defer func() { instanceReq.HealthCheck = oldHealthCheck }()
		So(server.UpdateInstance(defaultCtx, instanceReq).GetCode().GetValue(), ShouldEqual, api.NoNeedUpdate)
	})
}

// 实例数据更新测试
// 部分数据变更，触发更新
func TestUpdateInstanceFiled(t *testing.T) {
	_, serviceResp := createCommonService(t, 555)
	defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
	instanceReq, instanceResp := createCommonInstance(t, serviceResp, 555)
	defer cleanInstance(instanceResp.GetId().GetValue())
	Convey("metadata变更", t, func() {
		instanceReq.Metadata = map[string]string{}
		So(server.UpdateInstance(defaultCtx, instanceReq).GetCode().GetValue(), ShouldEqual, api.ExecuteSuccess)

		instanceReq.Metadata = map[string]string{"123": "456", "789": "abc", "135": "246"}
		So(server.UpdateInstance(defaultCtx, instanceReq).GetCode().GetValue(), ShouldEqual, api.ExecuteSuccess)

		instanceReq.Metadata["890"] = "678"
		So(server.UpdateInstance(defaultCtx, instanceReq).GetCode().GetValue(), ShouldEqual, api.ExecuteSuccess)

		delete(instanceReq.Metadata, "135")
		So(server.UpdateInstance(defaultCtx, instanceReq).GetCode().GetValue(), ShouldEqual, api.ExecuteSuccess)
	})
	Convey("healthCheck变更", t, func() {
		instanceReq.HealthCheck.Heartbeat.Ttl.Value = 33
		So(server.UpdateInstance(defaultCtx, instanceReq).GetCode().GetValue(), ShouldEqual, api.ExecuteSuccess)

		instanceReq.EnableHealthCheck = utils.NewBoolValue(false)
		So(server.UpdateInstance(defaultCtx, instanceReq).GetCode().GetValue(), ShouldEqual, api.ExecuteSuccess)
		newInstanceResp := server.GetInstances(map[string]string{
			"service":   serviceResp.GetName().GetValue(),
			"namespace": serviceResp.GetNamespace().GetValue(),
		})
		So(newInstanceResp.GetInstances()[0].GetHealthCheck(), ShouldBeNil)
		instanceReq.HealthCheck = nil

		instanceReq.EnableHealthCheck = utils.NewBoolValue(true)
		So(server.UpdateInstance(defaultCtx, instanceReq).GetCode().GetValue(), ShouldEqual, api.NoNeedUpdate)

		instanceReq.HealthCheck = &api.HealthCheck{
			Type:      api.HealthCheck_HEARTBEAT,
			Heartbeat: &api.HeartbeatHealthCheck{Ttl: utils.NewUInt32Value(50)},
		}
		So(server.UpdateInstance(defaultCtx, instanceReq).GetCode().GetValue(), ShouldEqual, api.ExecuteSuccess)
	})
	Convey("其他字段变更", t, func() {
		instanceReq.Protocol.Value = "new-protocol-1"
		So(server.UpdateInstance(defaultCtx, instanceReq).GetCode().GetValue(), ShouldEqual, api.ExecuteSuccess)

		instanceReq.Version.Value = "new-version-1"
		So(server.UpdateInstance(defaultCtx, instanceReq).GetCode().GetValue(), ShouldEqual, api.ExecuteSuccess)

		instanceReq.Priority.Value = 88
		So(server.UpdateInstance(defaultCtx, instanceReq).GetCode().GetValue(), ShouldEqual, api.ExecuteSuccess)

		instanceReq.Weight.Value = 500
		So(server.UpdateInstance(defaultCtx, instanceReq).GetCode().GetValue(), ShouldEqual, api.ExecuteSuccess)

		instanceReq.Healthy.Value = true
		So(server.UpdateInstance(defaultCtx, instanceReq).GetCode().GetValue(), ShouldEqual, api.ExecuteSuccess)

		instanceReq.Isolate.Value = true
		So(server.UpdateInstance(defaultCtx, instanceReq).GetCode().GetValue(), ShouldEqual, api.ExecuteSuccess)

		instanceReq.LogicSet.Value = "new-logic-set-1"
		So(server.UpdateInstance(defaultCtx, instanceReq).GetCode().GetValue(), ShouldEqual, api.ExecuteSuccess)

		newInstanceResp := server.GetInstances(map[string]string{
			"service":   serviceResp.GetName().GetValue(),
			"namespace": serviceResp.GetNamespace().GetValue(),
		})
		instanceCheck(t, newInstanceResp.GetInstances()[0], instanceReq)
	})
}

// 根据服务名获取实例列表并且做基础的判断
func getInstancesWithService(t *testing.T, name string, namespace string, expectCount int) []*api.Instance {
	query := map[string]string{
		"service":   name,
		"namespace": namespace,
	}
	resp := server.GetInstances(query)
	if !respSuccess(resp) {
		t.Fatalf("error: %s", resp.GetInfo().GetValue())
	}

	if len(resp.GetInstances()) != expectCount {
		t.Fatalf("error: %d", len(resp.GetInstances()))
	}

	return resp.GetInstances()
}

// test对instance字段进行校验
func TestCheckInstanceFieldLen(t *testing.T) {
	_, serviceResp := createCommonService(t, 800)
	defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
	ins := &api.Instance{
		ServiceToken: serviceResp.GetToken(),
		Service:      serviceResp.GetName(),
		Namespace:    serviceResp.GetNamespace(),
		Host:         utils.NewStringValue("127.0.0.1"),
		Protocol:     utils.NewStringValue("grpc"),
		Version:      utils.NewStringValue("1.0.1"),
		LogicSet:     utils.NewStringValue("sz"),
		Metadata:     map[string]string{},
	}

	t.Run("服务名超长", func(t *testing.T) {
		str := genSpecialStr(129)
		oldName := ins.Service
		ins.Service = utils.NewStringValue(str)
		resp := server.CreateInstance(defaultCtx, ins)
		ins.Service = oldName
		if resp.Code.Value != api.InvalidServiceName {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("host超长", func(t *testing.T) {
		str := genSpecialStr(129)
		oldHost := ins.Host
		ins.Host = utils.NewStringValue(str)
		resp := server.CreateInstance(defaultCtx, ins)
		ins.Host = oldHost
		if resp.Code.Value != api.InvalidInstanceHost {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("protocol超长", func(t *testing.T) {
		str := genSpecialStr(129)
		oldProtocol := ins.Protocol
		ins.Protocol = utils.NewStringValue(str)
		resp := server.CreateInstance(defaultCtx, ins)
		ins.Protocol = oldProtocol
		if resp.Code.Value != api.InvalidInstanceProtocol {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("version超长", func(t *testing.T) {
		str := genSpecialStr(129)
		oldVersion := ins.Version
		ins.Version = utils.NewStringValue(str)
		resp := server.CreateInstance(defaultCtx, ins)
		ins.Version = oldVersion
		if resp.Code.Value != api.InvalidInstanceVersion {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("logicSet超长", func(t *testing.T) {
		str := genSpecialStr(129)
		oldLogicSet := ins.LogicSet
		ins.LogicSet = utils.NewStringValue(str)
		resp := server.CreateInstance(defaultCtx, ins)
		ins.LogicSet = oldLogicSet
		if resp.Code.Value != api.InvalidInstanceLogicSet {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("metadata超长", func(t *testing.T) {
		str := genSpecialStr(129)
		oldMetadata := ins.Metadata
		oldMetadata[str] = str
		resp := server.CreateInstance(defaultCtx, ins)
		ins.Metadata = make(map[string]string)
		if resp.Code.Value != api.InvalidMetadata {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("port超长", func(t *testing.T) {
		oldPort := ins.Port
		ins.Port = utils.NewUInt32Value(70000)
		resp := server.CreateInstance(defaultCtx, ins)
		ins.Port = oldPort
		if resp.Code.Value != api.InvalidInstancePort {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("weight超长", func(t *testing.T) {
		oldWeight := ins.Weight
		ins.Weight = utils.NewUInt32Value(70000)
		resp := server.CreateInstance(defaultCtx, ins)
		ins.Weight = oldWeight
		if resp.Code.Value != api.InvalidParameter {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("检测字段为空指针", func(t *testing.T) {
		oldName := ins.Service
		ins.Service = nil
		resp := server.CreateInstance(defaultCtx, ins)
		ins.Service = oldName
		if resp.Code.Value != api.InvalidServiceName {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("检测字段为空", func(t *testing.T) {
		oldName := ins.Service
		ins.Service = utils.NewStringValue("")
		resp := server.CreateInstance(defaultCtx, ins)
		ins.Service = oldName
		if resp.Code.Value != api.InvalidServiceName {
			t.Fatalf("%+v", resp)
		}
	})
}

// test对instance入参进行校验
func TestCheckInstanceParam(t *testing.T) {
	// get instances接口限制(service+namespace)或者host必传，其它传参均拒绝服务
	_, serviceResp := createCommonService(t, 1254)
	defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	instanceReq, instanceResp := createCommonInstance(t, serviceResp, 153)
	defer cleanInstance(instanceResp.GetId().GetValue())

	t.Run("只传service", func(t *testing.T) {
		query := map[string]string{}
		query["service"] = "test"
		resp := server.GetInstances(query)
		if resp.Code.Value != api.InvalidQueryInsParameter {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("只传namespace", func(t *testing.T) {
		query := map[string]string{}
		query["namespace"] = "test"
		resp := server.GetInstances(query)
		if resp.Code.Value != api.InvalidQueryInsParameter {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("只传port", func(t *testing.T) {
		query := map[string]string{}
		query["port"] = "123"
		resp := server.GetInstances(query)
		if resp.Code.Value != api.InvalidQueryInsParameter {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("只传version", func(t *testing.T) {
		query := map[string]string{}
		query["version"] = "123"
		resp := server.GetInstances(query)
		if resp.Code.Value != api.InvalidQueryInsParameter {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("只传protocol", func(t *testing.T) {
		query := map[string]string{}
		query["protocol"] = "http"
		resp := server.GetInstances(query)
		if resp.Code.Value != api.InvalidQueryInsParameter {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("传service+port", func(t *testing.T) {
		query := map[string]string{}
		query["service"] = "test"
		query["port"] = "123"
		resp := server.GetInstances(query)
		if resp.Code.Value != api.InvalidQueryInsParameter {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("传namespace+port", func(t *testing.T) {
		query := map[string]string{}
		query["namespace"] = "test"
		query["port"] = "123"
		resp := server.GetInstances(query)
		if resp.Code.Value != api.InvalidQueryInsParameter {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("传service+namespace", func(t *testing.T) {
		query := map[string]string{}
		query["service"] = instanceReq.GetService().Value
		query["namespace"] = instanceReq.GetNamespace().Value
		resp := server.GetInstances(query)
		if resp.Code.Value != api.ExecuteSuccess {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("传service+namespace+host", func(t *testing.T) {
		query := map[string]string{}
		query["service"] = instanceReq.GetService().Value
		query["namespace"] = instanceReq.GetNamespace().Value
		query["host"] = instanceReq.GetHost().Value
		resp := server.GetInstances(query)
		if resp.Code.Value != api.ExecuteSuccess {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("传service+namespace+port", func(t *testing.T) {
		query := map[string]string{}
		query["service"] = instanceReq.GetService().Value
		query["namespace"] = instanceReq.GetNamespace().Value
		query["port"] = strconv.Itoa(int(instanceReq.GetPort().Value))
		resp := server.GetInstances(query)
		if resp.Code.Value != api.ExecuteSuccess {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("传host", func(t *testing.T) {
		query := map[string]string{}
		query["host"] = instanceReq.GetHost().Value
		resp := server.GetInstances(query)
		if resp.Code.Value != api.ExecuteSuccess {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("传host+namespace", func(t *testing.T) {
		query := map[string]string{}
		query["host"] = instanceReq.GetHost().Value
		query["namespace"] = instanceReq.GetNamespace().Value
		resp := server.GetInstances(query)
		if resp.Code.Value != api.ExecuteSuccess {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("传host+port", func(t *testing.T) {
		query := map[string]string{}
		query["host"] = instanceReq.GetHost().Value
		query["port"] = strconv.Itoa(int(instanceReq.GetPort().Value))
		resp := server.GetInstances(query)
		if resp.Code.Value != api.ExecuteSuccess {
			t.Fatalf("%+v", resp)
		}
	})
}
