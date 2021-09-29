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
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/naming"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/polarismesh/polaris-server/common/utils"

	"github.com/polarismesh/polaris-server/naming/cache"

	api "github.com/polarismesh/polaris-server/common/api/v1"

	// 使用mysql库

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang/protobuf/ptypes/wrappers"
	_ "github.com/polarismesh/polaris-server/store/sqldb"
	"github.com/stretchr/testify/assert"
)

// 测试样例结构体
type Case struct {
	field  string
	req    *api.Instance
	expect uint32
}

const (
	timeoutTimes = 2
)

// 测试健康检查功能未开启的情况
func TestHealthCheckNotOpen(t *testing.T) {
	_, serviceResp := createCommonService(t, 131)
	defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
	_, req := createInstanceNotOpenHealthCheck(t, serviceResp, 131)
	req.ServiceToken = &wrappers.StringValue{Value: serviceResp.GetToken().GetValue()}
	defer cleanInstance(req.GetId().GetValue())

	var heartbeatOnDisabledIns uint32 = 400141
	wait4Cache()
	rsp := server.Heartbeat(context.Background(), req)
	fmt.Printf("actiual:%v", rsp.GetCode().GetValue())
	assert.EqualValues(t, heartbeatOnDisabledIns, rsp.GetCode().GetValue())
}

// 测试错误输入的情况
func TestWrongInput(t *testing.T) {
	// 输入样例
	cases := []Case{
		{"req", nil, api.EmptyRequest},
		{"service_token", &api.Instance{}, api.InvalidServiceToken},
		{"service", &api.Instance{
			Namespace: &wrappers.StringValue{Value: "n"},
			Host:      &wrappers.StringValue{Value: "h"},
			Port:      &wrappers.UInt32Value{Value: 1},
			HealthCheck: &api.HealthCheck{
				Heartbeat: &api.HeartbeatHealthCheck{Ttl: &wrappers.UInt32Value{Value: 1}},
			},
			ServiceToken: &wrappers.StringValue{Value: "t"},
		}, api.InvalidServiceName},
		{"namespace", &api.Instance{
			Service: &wrappers.StringValue{Value: "s"},
			Host:    &wrappers.StringValue{Value: "h"},
			Port:    &wrappers.UInt32Value{Value: 1},
			HealthCheck: &api.HealthCheck{
				Heartbeat: &api.HeartbeatHealthCheck{Ttl: &wrappers.UInt32Value{Value: 1}},
			},
			ServiceToken: &wrappers.StringValue{Value: "t"},
		}, api.InvalidNamespaceName},
		{"host", &api.Instance{
			Service:   &wrappers.StringValue{Value: "s"},
			Namespace: &wrappers.StringValue{Value: "n"},
			Port:      &wrappers.UInt32Value{Value: 1},
			HealthCheck: &api.HealthCheck{
				Heartbeat: &api.HeartbeatHealthCheck{Ttl: &wrappers.UInt32Value{Value: 1}},
			},
			ServiceToken: &wrappers.StringValue{Value: "t"},
		}, api.InvalidInstanceHost},
		{"port", &api.Instance{
			Service:   &wrappers.StringValue{Value: "s"},
			Namespace: &wrappers.StringValue{Value: "n"},
			Host:      &wrappers.StringValue{Value: "h"},
			HealthCheck: &api.HealthCheck{
				Heartbeat: &api.HeartbeatHealthCheck{Ttl: &wrappers.UInt32Value{Value: 1}},
			},
			ServiceToken: &wrappers.StringValue{Value: "t"},
		}, api.InvalidInstancePort},
	}

	for _, c := range cases {
		func(c Case) {
			t.Run(fmt.Sprintf("测试输入缺少%v的情况", c.field), func(t *testing.T) {
				t.Parallel()
				rsp := server.Heartbeat(context.Background(), c.req)
				assert.EqualValues(t, c.expect, rsp.GetCode().GetValue())
			})
		}(c)
	}

	t.Run("测试传入非法token的情况", func(t *testing.T) {
		t.Parallel()
		var (
			req   *api.Instance
			rsp   *api.Response
			index = 1006
		)
		_, serviceResp := createCommonService(t, index)
		defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
		_, req = createCommonInstance(t, serviceResp, index)
		defer cleanInstance(req.GetId().GetValue())
		req.ServiceToken = &wrappers.StringValue{Value: serviceResp.GetToken().GetValue()}
		wait4Cache()
		req.ServiceToken = &wrappers.StringValue{Value: "err token"}
		rsp = server.Heartbeat(context.Background(), req)
		assert.EqualValues(t, api.Unauthorized, rsp.GetCode().GetValue())
	})
}

// 测试输入正确的情况
func TestHealthCheckInputRight(t *testing.T) {
	t.Run("测试输入正确，使用id为key的情况", func(t *testing.T) {
		var (
			req   *api.Instance
			rsp   *api.Response
			index = 15
		)
		_, serviceResp := createCommonService(t, index)
		defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
		_, req = createCommonInstance(t, serviceResp, index)
		defer cleanInstance(req.GetId().GetValue())
		wait4Cache()
		req.Service = nil
		req.ServiceToken = &wrappers.StringValue{Value: serviceResp.GetToken().GetValue()}
		rsp = server.Heartbeat(context.Background(), req)
		assert.EqualValues(t, api.ExecuteSuccess, rsp.GetCode().GetValue())
		assert.True(t, getHealthStatus(req.GetHost().GetValue(), int(req.GetPort().GetValue())))
	})

	t.Run("测试输入正确，使用四元组为key的情况", func(t *testing.T) {
		var (
			req   *api.Instance
			rsp   *api.Response
			index = 16
		)
		_, serviceResp := createCommonService(t, index)
		defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
		_, req = createCommonInstance(t, serviceResp, index)
		defer cleanInstance(req.GetId().GetValue())
		wait4Cache()
		req.Id = nil
		req.ServiceToken = &wrappers.StringValue{Value: serviceResp.GetToken().GetValue()}
		rsp = server.Heartbeat(context.Background(), req)
		assert.EqualValues(t, api.ExecuteSuccess, rsp.GetCode().GetValue())
		assert.True(t, getHealthStatus(req.GetHost().GetValue(), int(req.GetPort().GetValue())))
	})
}

// 测试健康状态的变化
func TestTurnUnhealthy(t *testing.T) {
	t.Run("测试从健康变为不健康", turnUnhealthy)
}

// 测试ttl增加
func TestTtlIncrease(t *testing.T) {
	t.Run("测试ttl增加", ttlIncrease)
}

// 测试ttl减少
func TestTtlDecrease(t *testing.T) {
	t.Run("测试ttl减少", ttlDecrease)
}

// 从健康转变为不健康
func turnUnhealthy(t *testing.T) {
	//t.Parallel()
	var (
		req   *api.Instance
		rsp   *api.Response
		ttl   uint32 = 1
		index        = 6001
	)
	_, serviceResp := createCommonService(t, index)
	defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
	_, req = createCommonInstance(t, serviceResp, index)
	req.ServiceToken = &wrappers.StringValue{Value: serviceResp.GetToken().GetValue()}
	defer cleanInstance(req.GetId().GetValue())
	updateTTL(t, req, ttl)
	wait4Cache()
	rsp = server.Heartbeat(context.Background(), req)
	assert.EqualValues(t, api.ExecuteSuccess, rsp.GetCode().GetValue())
	time.Sleep(time.Duration(ttl) * time.Second)
	assert.True(t, getHealthStatus(req.GetHost().GetValue(), int(req.GetPort().GetValue())))

	rsp = server.Heartbeat(context.Background(), req)
	assert.EqualValues(t, api.ExecuteSuccess, rsp.GetCode().GetValue())
	time.Sleep(time.Duration(ttl) * time.Second)
	assert.True(t, getHealthStatus(req.GetHost().GetValue(), int(req.GetPort().GetValue())))

	// 停止发送心跳，等到超时时间到了后，该实例变为不健康
	time.Sleep(time.Duration((timeoutTimes+1)*ttl+3) * time.Second)
	assert.False(t, getHealthStatus(req.GetHost().GetValue(), int(req.GetPort().GetValue())))
}

// 多次心跳，ttl增加
func ttlIncrease(t *testing.T) {
	//t.Parallel()
	var (
		req        *api.Instance
		rsp        *api.Response
		ttl        uint32 = 1
		anotherTTL uint32 = 3
		index             = 50002
	)

	_, serviceResp := createCommonService(t, index)
	defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
	_, req = createCommonInstance(t, serviceResp, index)
	req.ServiceToken = &wrappers.StringValue{Value: serviceResp.GetToken().GetValue()}
	defer cleanInstance(req.GetId().GetValue())
	updateTTL(t, req, ttl)
	wait4Cache()

	rsp = server.Heartbeat(context.Background(), req)
	assert.EqualValues(t, api.ExecuteSuccess, rsp.GetCode().GetValue())
	// 更新ttl
	updateTTL(t, req, anotherTTL)

	time.Sleep(time.Second)
	rsp = server.Heartbeat(context.Background(), req)
	if rsp.GetCode().GetValue() != api.ExecuteSuccess {
		t.Errorf("heartBeat err:%s", rsp.GetInfo().GetValue())
	}
	assert.EqualValues(t, api.ExecuteSuccess, rsp.GetCode().GetValue())

	timeoutSec := timeoutTimes*anotherTTL + 1
	oldTimeoutSec := timeoutTimes*ttl + 1
	// 确保旧超时时间后，按照新的ttl来计算还未超时
	assert.Greater(t, timeoutSec, oldTimeoutSec)

	// 等待旧超时时间过去，此时实例应该还未超时
	time.Sleep(time.Duration(oldTimeoutSec) * time.Second)
	assert.True(t, getHealthStatus(req.GetHost().GetValue(), int(req.GetPort().GetValue())))

	// 再等待达到新超时时间，此时实例应该超时
	time.Sleep(time.Duration(timeoutSec+5) * time.Second)
	assert.False(t, getHealthStatus(req.GetHost().GetValue(), int(req.GetPort().GetValue())))
}

// 多次心跳，ttl减少
func ttlDecrease(t *testing.T) {
	//t.Parallel()
	var (
		req *api.Instance
		// instance   *api.Instance
		rsp        *api.Response
		ttl        uint32 = 3
		anotherTTL uint32 = 1
		index             = 50003
	)
	_, serviceResp := createCommonService(t, index)
	defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
	_, req = createCommonInstance(t, serviceResp, index)
	defer cleanInstance(req.GetId().GetValue())
	req.ServiceToken = &wrappers.StringValue{Value: serviceResp.GetToken().GetValue()}
	updateTTL(t, req, ttl)
	wait4Cache()

	rsp = server.Heartbeat(context.Background(), req)
	assert.EqualValues(t, api.ExecuteSuccess, rsp.GetCode().GetValue())

	// 更新ttl
	updateTTL(t, req, anotherTTL)
	wait4Cache()

	rsp = server.Heartbeat(context.Background(), req)
	assert.EqualValues(t, api.ExecuteSuccess, rsp.GetCode().GetValue())

	timeoutSec := timeoutTimes*anotherTTL + 2
	oldTimeoutSec := timeoutTimes * ttl
	// 确保超时时间后，按照旧的ttl来计算还未超时
	assert.Less(t, timeoutSec, oldTimeoutSec)

	// 等待旧超时时间过去，此时实例应该就已经超时
	time.Sleep(time.Duration(oldTimeoutSec) * time.Second)
	assert.False(t, getHealthStatus(req.GetHost().GetValue(), int(req.GetPort().GetValue())))
}

// 获取实例健康状态
func getHealthStatus(host string, port int) bool {
	query := map[string]string{"limit": "20", "host": host, "port": strconv.Itoa(port)}
	rsp := server.GetInstances(query)
	if !respSuccess(rsp) {
		panic("寻找实例失败")
	}
	instances := rsp.GetInstances()
	if len(instances) != 1 {
		panic(fmt.Sprintf("找到的实例不唯一，已找到%v个实例", len(instances)))
	}
	return instances[0].GetHealthy().GetValue()
}

// 获取实例健康状态
func getHealthStatusByID(id string) bool {
	ins := server.Cache().Instance().GetInstance(id)
	if ins == nil {
		panic("寻找实例失败")
	}

	return ins.Proto.Healthy.Value
}

// 等待cache加载数据，在创建实例后需要使用
func wait4Cache() {
	time.Sleep(2 * cache.UpdateCacheInterval)
}

// 更新实例的ttl
func updateTTL(t *testing.T, instance *api.Instance, ttl uint32) {
	instance.HealthCheck = &api.HealthCheck{Heartbeat: &api.HeartbeatHealthCheck{Ttl: utils.NewUInt32Value(ttl)}}
	if resp := server.UpdateInstance(defaultCtx, instance); !respSuccess(resp) {
		t.Fatalf("error: %s", resp.GetInfo().GetValue())
	}
}

// 获取实例的ttl
func getTTL(t *testing.T, id string) uint32 {
	insCache := server.Cache().Instance().GetInstance(id)
	if insCache == nil {
		return 0
	}
	return insCache.HealthCheck().Heartbeat.Ttl.Value
}

// 测试存在不合法的实例的情况
func TestInvalidHealthInstance(t *testing.T) {
	t.Run("测试不存在实例的情况", func(t *testing.T) {
		t.Parallel()
		var (
			req   *api.Instance
			rsp   *api.Response
			index = 1004
		)
		_, serviceResp := createCommonService(t, index)
		defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
		_, req = createCommonInstance(t, serviceResp, index)
		// 创建一个实例，然后将其删除
		cleanInstance(req.GetId().GetValue())
		wait4Cache()
		req.ServiceToken = &wrappers.StringValue{Value: serviceResp.GetToken().GetValue()}
		rsp = server.Heartbeat(context.Background(), req)
		assert.EqualValues(t, api.NotFoundResource, rsp.GetCode().GetValue())
	})

	t.Run("测试不存在service的情况", func(t *testing.T) {
		t.Parallel()
		var (
			req   *api.Instance
			rsp   *api.Response
			index = 1007
		)
		_, serviceResp := createCommonService(t, index)
		_, req = createCommonInstance(t, serviceResp, index)
		defer cleanInstance(req.GetId().GetValue())
		// 删除服务
		cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
		wait4Cache()
		req.ServiceToken = &wrappers.StringValue{Value: serviceResp.GetToken().GetValue()}
		rsp = server.Heartbeat(context.Background(), req)
		assert.EqualValues(t, api.NotFoundResource, rsp.GetCode().GetValue())
	})

	t.Run("测试存在ttl非法的实例的情况", func(t *testing.T) {
		t.Parallel()
		var (
			req   *api.Instance
			rsp   *api.Response
			index = 1005
		)
		_, serviceResp := createCommonService(t, index)
		defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
		_, req = createCommonInstance(t, serviceResp, index)
		defer cleanInstance(req.GetId().GetValue())
		req.ServiceToken = &wrappers.StringValue{Value: serviceResp.GetToken().GetValue()}
		updateTTL(t, req, 123123)
		wait4Cache()

		ttl := getTTL(t, req.GetId().GetValue())
		assert.EqualValues(t, ttl, 5)

		rsp = server.Heartbeat(context.Background(), req)
		assert.EqualValues(t, api.ExecuteSuccess, rsp.GetCode().GetValue())
	})
}

// 测试大量heartbeat用时
func TestHeartBeatUseTime(t *testing.T) {
	t.Run("测试大量heartbeat用时", heartBeatUseTime)
}

// 测试大量heartbeat用时
func heartBeatUseTime(t *testing.T) {
	heartBeatBatch(t, 10, 80)
}

// 模拟正常多实例心跳上报情境
func heartBeatBatch(t *testing.T, serviceNum, insNum int) {
	var (
		req      *api.Instance
		index    = 10000
		insArray = make([]*api.Instance, 0)
		wg       sync.WaitGroup
		mu       sync.Mutex
	)

	start := time.Now()
	// 创建服务和实例
	for i := 0; i < serviceNum; i++ {
		var wgt sync.WaitGroup
		_, serviceResp := createCommonService(t, index+i)
		defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
		for j := 0; j < insNum; j++ {
			go func(index int, serviceResp *api.Service) {
				wgt.Add(1)
				defer wgt.Done()
				_, req = createCommonInstance(t, serviceResp, index)
				req.ServiceToken = &wrappers.StringValue{Value: serviceResp.GetToken().GetValue()}

				mu.Lock()
				insArray = append(insArray, req)
				mu.Unlock()
			}(index+i+j, serviceResp)
			//睡眠0.1毫秒，削峰
			time.Sleep(time.Microsecond * 100)
		}
		wgt.Wait()
		time.Sleep(100 * time.Millisecond)
	}
	t.Logf("create use time:%+v", time.Now().Sub(start))
	time.Sleep(time.Second * 2)

	wg.Add(len(insArray))
	exceedNum := 0
	now := time.Now()
	for _, ins := range insArray {
		go func(ins *api.Instance) {
			resp := server.Heartbeat(context.Background(), ins)
			if resp.GetCode().GetValue() != api.ExecuteSuccess {
				exceedNum++
			}
			wg.Done()
		}(ins)
		time.Sleep(time.Microsecond)
	}
	wg.Wait()
	t.Logf("first, use time:%v, exceedNum:%d", time.Now().Sub(now), exceedNum)

	time.Sleep(time.Second * 3)
	wg.Add(len(insArray))
	exceedNum = 0
	now = time.Now()
	for _, ins := range insArray {
		go func(ins *api.Instance) {
			resp := server.Heartbeat(context.Background(), ins)
			if resp.GetCode().GetValue() != api.ExecuteSuccess {
				exceedNum++
			}
			wg.Done()
		}(ins)
		time.Sleep(time.Microsecond)
	}
	wg.Wait()
	t.Logf("third, use time:%v, exceedNum:%d", time.Now().Sub(now), exceedNum)
	t.Logf("len:%d", len(insArray))

	for _, ins := range insArray {
		assert.True(t, getHealthStatusByID(ins.GetId().GetValue()))
	}
	ttl := 5
	time.Sleep(time.Duration((timeoutTimes+2)*ttl+4) * time.Second)
	wait4Cache()
	time.Sleep(20 * time.Second)
	for _, ins := range insArray {
		assert.False(t, getHealthStatusByID(ins.GetId().GetValue()))
	}
	for _, ins := range insArray {
		cleanInstance(ins.GetId().GetValue())
	}
}

// 测试ckv+节点变更
func TestCkvNodeChange(t *testing.T) {
	t.Logf("第一次测试心跳")
	heartBeatBatch(t, 8, 20)

	name := cfg.Naming.HealthCheck.KvServiceName
	namespace := cfg.Naming.HealthCheck.KvNamespace
	service := server.Cache().Service().GetServiceByName(name, namespace)
	if service == nil {
		t.Fatalf("cannot get service, name:%s, namespace:%s", name, namespace)
	}
	instances := server.Cache().Instance().GetInstancesByServiceID(service.ID)
	if len(instances) == 0 {
		t.Fatalf("cannot get instance, name:%s, namespace:%s", name, namespace)
	}
	t.Logf("len:%d, instaces:%+v", len(instances), instances[0])

	_ = server.Cache().Clear() // 为了防止影响，每个函数需要把缓存的内容清空
	creq := &api.Instance{
		ServiceToken: utils.NewStringValue(service.Token),
		Id:           utils.NewStringValue(instances[0].ID()),
	}
	// 节点增加
	t.Logf("ckv节点增加")
	addInstance(t, ins2Api(instances[0], service.Token, service.Name, service.Namespace))
	time.Sleep(30 * time.Second)
	t.Logf("再次测试心跳")
	heartBeatBatch(t, 10, 20)

	// 节点减少
	t.Logf("ckv节点减少")
	resp := server.DeleteInstance(defaultCtx, creq)
	if !respSuccess(resp) {
		t.Fatalf("error: %s", resp.GetInfo().GetValue())
	}
	time.Sleep(30 * time.Second)
	t.Logf("再次测试心跳")
	heartBeatBatch(t, 10, 20)

	t.Logf("ok")
}

func ins2Api(ins *model.Instance, token, name, namespace string) *api.Instance {
	rand.Seed(time.Now().UnixNano())
	return &api.Instance{
		ServiceToken: utils.NewStringValue(token),
		Service:      utils.NewStringValue(name),
		Namespace:    utils.NewStringValue(namespace),
		VpcId:        utils.NewStringValue(strconv.Itoa(rand.Intn(10000))),
		Host:         utils.NewStringValue(ins.Host()),
		Port:         utils.NewUInt32Value(ins.Port()),
		Protocol:     utils.NewStringValue(ins.Protocol()),
		Version:      utils.NewStringValue(ins.Version()),
		Priority:     utils.NewUInt32Value(ins.Priority()),
		Weight:       utils.NewUInt32Value(ins.Weight()),
		HealthCheck:  ins.HealthCheck(),
		Healthy:      utils.NewBoolValue(ins.Healthy()),
		Isolate:      utils.NewBoolValue(ins.Isolate()),
		LogicSet:     utils.NewStringValue(ins.LogicSet()),
		Metadata:     ins.Metadata(),
	}
}

// 新增一个不开启健康检查的实例
func createInstanceNotOpenHealthCheck(t *testing.T, service *api.Service, id int) (
	*api.Instance, *api.Instance) {
	instanceReq := &api.Instance{
		ServiceToken: utils.NewStringValue(service.GetToken().GetValue()),
		Service:      utils.NewStringValue(service.GetName().GetValue()),
		Namespace:    utils.NewStringValue(service.GetNamespace().GetValue()),
		VpcId:        utils.NewStringValue(fmt.Sprintf("vpcid-%d", id)),
		Host:         utils.NewStringValue(fmt.Sprintf("10.10.10.%d", id)),
		Port:         utils.NewUInt32Value(8000 + uint32(id)),
		Protocol:     utils.NewStringValue(fmt.Sprintf("protocol-%d", id)),
		Version:      utils.NewStringValue(fmt.Sprintf("version-%d", id)),
		Priority:     utils.NewUInt32Value(1 + uint32(id)%10),
		Weight:       utils.NewUInt32Value(1 + uint32(id)%1000),
		HealthCheck:  nil,
		Healthy:      utils.NewBoolValue(false), // 默认是非健康，因为打开了healthCheck
		Isolate:      utils.NewBoolValue(false),
		LogicSet:     utils.NewStringValue(fmt.Sprintf("logic-set-%d", id)),
		Metadata: map[string]string{
			"internal-personal-xxx":        fmt.Sprintf("internal-personal-xxx_%d", id),
			"2my-meta":                     fmt.Sprintf("my-meta-%d", id),
			"my-meta-a1":                   "1111",
			"smy-xmeta-h2":                 "2222",
			"my-1meta-o3":                  "2222",
			"my-2meta-4c":                  "2222",
			"my-3meta-d5":                  "2222",
			"dmy-meta-6p":                  "2222",
			"1my-pmeta-d7":                 "2222",
			"my-dmeta-8c":                  "2222",
			"my-xmeta-9p":                  "2222",
			"other-meta-x":                 "xxx",
			"other-meta-1":                 "xx11",
			"amy-instance":                 "my-instance",
			"very-long-key-data-xxxxxxxxx": "Y",
			"very-long-key-data-uuuuuuuuu": "P",
		},
	}

	resp := server.CreateInstance(defaultCtx, instanceReq)
	if respSuccess(resp) {
		return instanceReq, resp.GetInstance()
	}

	if resp.GetCode().GetValue() != api.ExistedResource {
		t.Fatalf("error: %s", resp.GetInfo().GetValue())
	}

	// repeated
	InstanceID, _ := naming.CalculateInstanceID(instanceReq.GetNamespace().GetValue(), instanceReq.GetService().GetValue(),
		instanceReq.GetVpcId().GetValue(), instanceReq.GetHost().GetValue(), instanceReq.GetPort().GetValue())
	cleanInstance(InstanceID)
	t.Logf("repeatd create instance(%s)", InstanceID)
	resp = server.CreateInstance(defaultCtx, instanceReq)
	if !respSuccess(resp) {
		t.Fatalf("error: %s", resp.GetInfo().GetValue())
	}

	return instanceReq, resp.GetInstance()
}
