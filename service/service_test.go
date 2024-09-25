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

package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/wrappers"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/polarismesh/polaris/auth"
	"github.com/polarismesh/polaris/cache"
	cachetypes "github.com/polarismesh/polaris/cache/api"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/namespace"
	"github.com/polarismesh/polaris/service"
	"github.com/polarismesh/polaris/store"
	"github.com/polarismesh/polaris/store/mock"
)

// 测试新增服务
func TestCreateService(t *testing.T) {

	t.Run("正常创建服务", func(t *testing.T) {
		discoverSuit := &DiscoverTestSuit{}
		if err := discoverSuit.Initialize(); err != nil {
			t.Fatal(err)
		}
		serviceReq, serviceResp := discoverSuit.createCommonService(t, 9)

		t.Cleanup(func() {
			discoverSuit.cleanAllService()
			discoverSuit.Destroy()
		})

		if serviceResp.GetName().GetValue() == serviceReq.GetName().GetValue() &&
			serviceResp.GetNamespace().GetValue() == serviceReq.GetNamespace().GetValue() &&
			serviceResp.GetToken().GetValue() != "" {
			t.Logf("pass")
		} else {
			t.Fatalf("error: %+v", serviceResp)
		}
	})

	t.Run("创建重复名字的服务，会返回失败", func(t *testing.T) {
		discoverSuit := &DiscoverTestSuit{}
		if err := discoverSuit.Initialize(); err != nil {
			t.Fatal(err)
		}

		serviceReq, _ := discoverSuit.createCommonService(t, 9)
		t.Cleanup(func() {
			discoverSuit.cleanAllService()
			discoverSuit.Destroy()
		})

		resp := discoverSuit.DiscoverServer().CreateServices(discoverSuit.DefaultCtx, []*apiservice.Service{serviceReq})
		if !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error")
		}
	})

	t.Run("创建服务，删除，再次创建，可以正常创建", func(t *testing.T) {
		discoverSuit := &DiscoverTestSuit{}
		if err := discoverSuit.Initialize(); err != nil {
			t.Fatal(err)
		}

		serviceReq, serviceResp := discoverSuit.createCommonService(t, 100)
		t.Cleanup(func() {
			discoverSuit.cleanAllService()
			discoverSuit.Destroy()
		})

		req := &apiservice.Service{
			Name:      utils.NewStringValue(serviceResp.GetName().GetValue()),
			Namespace: utils.NewStringValue(serviceResp.GetNamespace().GetValue()),
			Token:     utils.NewStringValue(serviceResp.GetToken().GetValue()),
		}
		discoverSuit.removeCommonServices(t, []*apiservice.Service{req})

		if resp := discoverSuit.DiscoverServer().CreateServices(discoverSuit.DefaultCtx, []*apiservice.Service{serviceReq}); !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}

		t.Logf("pass")
	})
	t.Run("并发创建不同服务", func(t *testing.T) {
		discoverSuit := &DiscoverTestSuit{}
		if err := discoverSuit.Initialize(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			discoverSuit.cleanAllService()
			discoverSuit.Destroy()
		})

		var wg sync.WaitGroup
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				serviceReq, _ := discoverSuit.createCommonService(t, index)
				discoverSuit.cleanServiceName(serviceReq.GetName().GetValue(), serviceReq.GetNamespace().GetValue())
			}(i)
		}
		wg.Wait()
	})
	t.Run("并发创建相同服务", func(t *testing.T) {
		discoverSuit := &DiscoverTestSuit{}
		if err := discoverSuit.Initialize(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			discoverSuit.cleanAllService()
			discoverSuit.Destroy()
		})

		var wg sync.WaitGroup
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(_ int) {
				defer wg.Done()
				serviceReq := genMainService(1)
				resp := discoverSuit.DiscoverServer().CreateServices(discoverSuit.DefaultCtx, []*apiservice.Service{serviceReq})

				if resp.GetCode().GetValue() == uint32(apimodel.Code_ExistedResource) {
					assert.True(t, len(resp.GetResponses()[0].GetService().GetId().GetValue()) > 0)
				}
			}(i)
		}
		wg.Wait()
	})
	t.Run("命名空间不存在，可以自动创建服务", func(t *testing.T) {
		discoverSuit := &DiscoverTestSuit{}
		if err := discoverSuit.Initialize(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			discoverSuit.cleanAllService()
			discoverSuit.Destroy()
		})

		service := &apiservice.Service{
			Name:      utils.NewStringValue("abc"),
			Namespace: utils.NewStringValue(utils.NewUUID()),
			Owners:    utils.NewStringValue("my"),
		}
		resp := discoverSuit.DiscoverServer().CreateServices(discoverSuit.DefaultCtx, []*apiservice.Service{service})
		if !respSuccess(resp) {
			t.Fatalf("error")
		}
		t.Logf("pass: %s", resp.GetInfo().GetValue())
	})
	t.Run("创建服务，metadata个数太多，报错", func(t *testing.T) {
		discoverSuit := &DiscoverTestSuit{}
		if err := discoverSuit.Initialize(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			discoverSuit.cleanAllService()
			discoverSuit.Destroy()
		})

		svc := &apiservice.Service{
			Name:      utils.NewStringValue("999"),
			Namespace: utils.NewStringValue("Polaris"),
			Owners:    utils.NewStringValue("my"),
		}
		svc.Metadata = make(map[string]string)
		for i := 0; i < service.MaxMetadataLength+1; i++ {
			svc.Metadata[fmt.Sprintf("aa-%d", i)] = "value"
		}
		if resp := discoverSuit.DiscoverServer().CreateServices(discoverSuit.DefaultCtx, []*apiservice.Service{svc}); !respSuccess(resp) {
			t.Logf("%s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error")
		}
	})
}

// delete services
func TestRemoveServices(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	t.Run("删除单个服务，删除成功", func(t *testing.T) {
		serviceReq, serviceResp := discoverSuit.createCommonService(t, 59)
		defer discoverSuit.cleanServiceName(serviceReq.GetName().GetValue(), serviceReq.GetNamespace().GetValue())

		req := &apiservice.Service{
			Name:      utils.NewStringValue(serviceResp.GetName().GetValue()),
			Namespace: utils.NewStringValue(serviceResp.GetNamespace().GetValue()),
			Token:     utils.NewStringValue(serviceResp.GetToken().GetValue()),
		}

		// wait for data cache
		time.Sleep(time.Second * 2)
		discoverSuit.removeCommonServices(t, []*apiservice.Service{req})
		out := discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, map[string]string{"name": req.GetName().GetValue()})
		if !respSuccess(out) {
			t.Fatalf(out.GetInfo().GetValue())
		}
		if len(out.GetServices()) != 0 {
			t.Fatalf("error: %d", len(out.GetServices()))
		}
	})

	t.Run("删除多个服务，删除成功", func(t *testing.T) {
		var reqs []*apiservice.Service
		for i := 0; i < 100; i++ {
			serviceReq, serviceResp := discoverSuit.createCommonService(t, i)
			defer discoverSuit.cleanServiceName(serviceReq.GetName().GetValue(), serviceReq.GetNamespace().GetValue())
			req := &apiservice.Service{
				Name:      utils.NewStringValue(serviceResp.GetName().GetValue()),
				Namespace: utils.NewStringValue(serviceResp.GetNamespace().GetValue()),
				Token:     utils.NewStringValue(serviceResp.GetToken().GetValue()),
			}
			reqs = append(reqs, req)
		}

		// wait for data cache
		time.Sleep(time.Second * 2)
		discoverSuit.removeCommonServices(t, reqs)
	})

	t.Run("创建一个服务，马上删除，可以正常删除", func(t *testing.T) {
		serviceReq, serviceResp := discoverSuit.createCommonService(t, 19)
		defer discoverSuit.cleanServiceName(serviceReq.GetName().GetValue(), serviceReq.GetNamespace().GetValue())

		req := &apiservice.Service{
			Name:      utils.NewStringValue(serviceResp.GetName().GetValue()),
			Namespace: utils.NewStringValue(serviceResp.GetNamespace().GetValue()),
			Token:     utils.NewStringValue(serviceResp.GetToken().GetValue()),
		}
		discoverSuit.removeCommonServices(t, []*apiservice.Service{req})
	})
	// TODO 需要具体排查为什么在 github-action 无法跑过
	// t.Run("创建服务和实例，删除服务，删除失败", func(t *testing.T) {
	// 	serviceReq, serviceResp := discoverSuit.createCommonService(t, 19)
	// 	defer discoverSuit.cleanServiceName(serviceReq.GetName().GetValue(), serviceReq.GetNamespace().GetValue())

	// 	_, instanceResp := discoverSuit.createCommonInstance(t, serviceResp, 100)
	// 	defer discoverSuit.cleanInstance(instanceResp.GetId().GetValue())

	// 	resp := discoverSuit.DiscoverServer().DeleteServices(discoverSuit.DefaultCtx, []*apiservice.Service{serviceResp})
	// 	if !respSuccess(resp) {
	// 		t.Logf("pass: %s", resp.GetInfo().GetValue())
	// 	} else {
	// 		t.Fatalf("error: %s", resp.GetInfo().GetValue())
	// 	}
	// })

	t.Run("并发删除服务", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 0; i < 20; i++ {
			serviceReq, serviceResp := discoverSuit.createCommonService(t, i)
			defer discoverSuit.cleanServiceName(serviceReq.GetName().GetValue(), serviceReq.GetNamespace().GetValue())
			req := &apiservice.Service{
				Name:      utils.NewStringValue(serviceResp.GetName().GetValue()),
				Namespace: utils.NewStringValue(serviceResp.GetNamespace().GetValue()),
				Token:     utils.NewStringValue(serviceResp.GetToken().GetValue()),
			}

			wg.Add(1)
			go func(reqs []*apiservice.Service) {
				defer wg.Done()
				discoverSuit.removeCommonServices(t, reqs)
			}([]*apiservice.Service{req})
		}
		wg.Wait()
	})
}

// 关联测试
func TestDeleteService2(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	t.Run("重复删除服务，返回成功", func(t *testing.T) {
		serviceReq, serviceResp := discoverSuit.createCommonService(t, 20)
		defer discoverSuit.cleanServiceName(serviceReq.GetName().GetValue(), serviceReq.GetNamespace().GetValue())

		discoverSuit.removeCommonServices(t, []*apiservice.Service{serviceResp})
		discoverSuit.removeCommonServices(t, []*apiservice.Service{serviceResp})
	})
	t.Run("存在别名的情况下，删除服务会失败", func(t *testing.T) {
		serviceReq, serviceResp := discoverSuit.createCommonService(t, 20)
		defer discoverSuit.cleanServiceName(serviceReq.GetName().GetValue(), serviceReq.GetNamespace().GetValue())

		aliasResp1 := discoverSuit.createCommonAlias(serviceResp, "", defaultAliasNs, apiservice.AliasType_CL5SID)
		defer discoverSuit.cleanServiceName(aliasResp1.Alias.Alias.Value, serviceResp.Namespace.Value)
		aliasResp2 := discoverSuit.createCommonAlias(serviceResp, "", defaultAliasNs, apiservice.AliasType_CL5SID)
		defer discoverSuit.cleanServiceName(aliasResp2.Alias.Alias.Value, serviceResp.Namespace.Value)

		// 删除服务
		resp := discoverSuit.DiscoverServer().DeleteServices(discoverSuit.DefaultCtx, []*apiservice.Service{serviceResp})
		if respSuccess(resp) {
			t.Fatalf("error")
		}
		t.Logf("pass: %s", resp.GetInfo().GetValue())
	})
}

// 测试批量获取服务负责人
func TestGetServiceOwner(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	t.Run("服务个数为0，返回错误", func(t *testing.T) {
		var reqs []*apiservice.Service
		if resp := discoverSuit.DiscoverServer().GetServiceOwner(discoverSuit.DefaultCtx, reqs); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})

	t.Run("服务个数超过100，返回错误", func(t *testing.T) {
		reqs := make([]*apiservice.Service, 0, 101)
		for i := 0; i < 101; i++ {
			req := &apiservice.Service{
				Namespace: utils.NewStringValue("Test"),
				Name:      utils.NewStringValue("test"),
			}
			reqs = append(reqs, req)
		}
		if resp := discoverSuit.DiscoverServer().GetServiceOwner(discoverSuit.DefaultCtx, reqs); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})

	t.Run("查询100个超长服务名的服务负责人，数据库不会报错", func(t *testing.T) {
		reqs := make([]*apiservice.Service, 0, 100)
		for i := 0; i < 100; i++ {
			req := &apiservice.Service{
				Namespace: utils.NewStringValue("Development"),
				Name:      utils.NewStringValue(genSpecialStr(128)),
			}
			reqs = append(reqs, req)
		}
		if resp := discoverSuit.DiscoverServer().GetServiceOwner(discoverSuit.DefaultCtx, reqs); !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})
}

// 测试获取服务函数
func TestGetService(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	t.Run("查询服务列表，可以正常返回", func(t *testing.T) {
		resp := discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, map[string]string{})
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.Info.GetValue())
		}
	})
	t.Run("查询服务列表，只有limit和offset，可以正常返回预计个数的服务", func(t *testing.T) {
		total := 20
		reqs := make([]*apiservice.Service, 0, total)
		for i := 0; i < total; i++ {
			serviceReq, _ := discoverSuit.createCommonService(t, i+10)
			reqs = append(reqs, serviceReq)
			defer discoverSuit.cleanServiceName(serviceReq.GetName().GetValue(), serviceReq.GetNamespace().GetValue())
		}

		// 创建完，直接查询
		filters := map[string]string{"offset": "0", "limit": "100"}
		resp := discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, filters)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.Info.GetValue())
		}

		if resp.GetSize().GetValue() >= uint32(total) && resp.GetSize().GetValue() <= 100 {
			t.Logf("pass")
		} else {
			t.Fatalf("error: %d %d", resp.GetSize().GetValue(), total)
		}
	})

	t.Run("查询服务列表，没有filter，只回复默认的service", func(t *testing.T) {
		total := 10
		for i := 0; i < total; i++ {
			serviceReq, _ := discoverSuit.createCommonService(t, i+10)
			defer discoverSuit.cleanServiceName(serviceReq.GetName().GetValue(), serviceReq.GetNamespace().GetValue())
		}

		resp := discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, map[string]string{})
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.Info.GetValue())
		}
		if resp.GetSize().GetValue() >= 10 {
			t.Logf("pass")
		} else {
			t.Fatalf("error: %d", resp.GetSize().GetValue())
		}
	})
	t.Run("查询服务列表，只能查询到源服务，无法查询到别名", func(t *testing.T) {
		total := 10
		for i := 0; i < total; i++ {
			_, serviceResp := discoverSuit.createCommonService(t, i+102)
			defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
			aliasResp := discoverSuit.createCommonAlias(serviceResp, "", defaultAliasNs, apiservice.AliasType_CL5SID)
			defer discoverSuit.cleanServiceName(aliasResp.Alias.Alias.Value, serviceResp.Namespace.Value)
		}
		resp := discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, map[string]string{"business": "business-102"})
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.Info.GetValue())
		}
		if resp.GetSize().GetValue() != 1 {
			t.Fatalf("error: %d", resp.GetSize().GetValue())
		}
	})
}

// 测试获取服务列表，参数校验
func TestGetServices2(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	t.Run("查询服务列表，limit有最大为100的限制", func(t *testing.T) {
		total := 101
		for i := 0; i < total; i++ {
			serviceReq, _ := discoverSuit.createCommonService(t, i+10)
			defer discoverSuit.cleanServiceName(serviceReq.GetName().GetValue(), serviceReq.GetNamespace().GetValue())
		}

		filters := map[string]string{"offset": "0", "limit": "600"}
		resp := discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, filters)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.Info.GetValue())
		}
		if resp.GetSize().GetValue() == service.QueryMaxLimit {
			t.Logf("pass")
		} else {
			t.Fatalf("error: %d", resp.GetSize().GetValue())
		}
	})
	t.Run("查询服务列表，offset参数不为int，返回错误", func(t *testing.T) {
		filters := map[string]string{"offset": "abc", "limit": "200"}
		resp := discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, filters)
		if !respSuccess(resp) {
			t.Logf("pass: %s", resp.Info.GetValue())
		} else {
			t.Fatalf("error")
		}
	})
	t.Run("查询服务列表，limit参数不为int，返回错误", func(t *testing.T) {
		filters := map[string]string{"offset": "0", "limit": "ss"}
		resp := discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, filters)
		if !respSuccess(resp) {
			t.Logf("pass: %s", resp.Info.GetValue())
		} else {
			t.Fatalf("error")
		}
	})
	t.Run("查询服务列表，offset参数为负数，返回错误", func(t *testing.T) {
		filters := map[string]string{"offset": "-100", "limit": "10"}
		resp := discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, filters)
		if !respSuccess(resp) {
			t.Logf("pass: %s", resp.Info.GetValue())
		} else {
			t.Fatalf("error")
		}
	})
	t.Run("查询服务列表，limit参数为负数，返回错误", func(t *testing.T) {
		filters := map[string]string{"offset": "100", "limit": "-10"}
		resp := discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, filters)
		if !respSuccess(resp) {
			t.Logf("pass: %s", resp.Info.GetValue())
		} else {
			t.Fatalf("error")
		}
	})
	t.Run("查询服务列表，单独提供port参数，返回错误", func(t *testing.T) {
		filters := map[string]string{"port": "100"}
		resp := discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, filters)
		if !respSuccess(resp) {
			t.Logf("pass: %s", resp.Info.GetValue())
		} else {
			t.Fatalf("error")
		}
	})
	t.Run("查询服务列表，port参数有误，返回错误", func(t *testing.T) {
		filters := map[string]string{"port": "p100", "host": "127.0.0.1"}
		resp := discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, filters)
		if !respSuccess(resp) {
			t.Logf("pass: %s", resp.Info.GetValue())
		} else {
			t.Fatalf("error")
		}
	})
}

// 有基础的过滤条件的查询服务列表
func TestGetService3(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	t.Run("根据服务名，可以正常过滤", func(t *testing.T) {
		var reqs []*apiservice.Service
		serviceReq, _ := discoverSuit.createCommonService(t, 100)
		reqs = append(reqs, serviceReq)
		defer discoverSuit.cleanServiceName(serviceReq.GetName().GetValue(), serviceReq.GetNamespace().GetValue())

		namespaceReq, _ := discoverSuit.createCommonNamespace(t, 100)
		defer discoverSuit.cleanNamespace(namespaceReq.GetName().GetValue())

		serviceReq.Namespace = utils.NewStringValue(namespaceReq.GetName().GetValue())
		if resp := discoverSuit.DiscoverServer().CreateServices(discoverSuit.DefaultCtx, []*apiservice.Service{serviceReq}); !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		reqs = append(reqs, serviceReq)
		defer discoverSuit.cleanServiceName(serviceReq.GetName().GetValue(), serviceReq.GetNamespace().GetValue())

		name := serviceReq.GetName().GetValue()
		filters := map[string]string{"offset": "0", "limit": "10", "name": name}
		resp := discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, filters)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}

		discoverSuit.CheckGetService(t, reqs, resp.GetServices())
		t.Logf("pass")
	})

	t.Run("多重过滤条件，可以生效", func(t *testing.T) {
		total := 10
		var name, namespace string
		for i := 0; i < total; i++ {
			serviceReq, _ := discoverSuit.createCommonService(t, 100)
			defer discoverSuit.cleanServiceName(serviceReq.GetName().GetValue(), serviceReq.GetNamespace().GetValue())
			if i == 5 {
				name = serviceReq.GetName().GetValue()
				namespace = serviceReq.GetNamespace().GetValue()
			}
		}
		filters := map[string]string{"offset": "0", "limit": "10", "name": name, "namespace": namespace}
		resp := discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, filters)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if len(resp.Services) != 1 {
			t.Fatalf("error: %d", len(resp.Services))
		}
	})

	t.Run("businessr过滤条件会生效", func(t *testing.T) {
		total := 60
		for i := 0; i < total; i++ {
			serviceReq, _ := discoverSuit.createCommonService(t, i+10)
			defer discoverSuit.cleanServiceName(serviceReq.GetName().GetValue(), serviceReq.GetNamespace().GetValue())
		}

		filters := map[string]string{"offset": "0", "limit": "100", "business": "business-60"}
		resp := discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, filters)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if len(resp.Services) != 1 {
			b, _ := json.Marshal(resp.Services)
			t.Logf("[error] services : %s", string(b))
			t.Fatalf("error: %d", len(resp.Services))
		}
	})
}

// 异常场景
func TestGetServices4(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	t.Run("查询服务列表，新建一批服务，删除部分，再查询，可以过滤掉删除的", func(t *testing.T) {
		total := 50
		for i := 0; i < total; i++ {
			serviceReq, serviceResp := discoverSuit.createCommonService(t, i+5)
			defer discoverSuit.cleanServiceName(serviceReq.GetName().GetValue(), serviceReq.GetNamespace().GetValue())
			if i%2 == 0 {
				discoverSuit.removeCommonServices(t, []*apiservice.Service{serviceResp})
			}
		}

		query := map[string]string{
			"offset": "0",
			"limit":  "100",
			"name":   "test-service-*",
		}
		resp := discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, query)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.Info.GetValue())
		}
		if resp.GetSize().GetValue() == uint32(total/2) {
			t.Logf("pass")
		} else {
			t.Fatalf("error: %d", resp.GetSize().GetValue())
		}
	})
	// 新建几个服务，不同metadata
	t.Run("根据metadata可以过滤services", func(t *testing.T) {
		service1 := genMainService(1)
		service1.Metadata = map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		}
		service2 := genMainService(2)
		service2.Metadata = map[string]string{
			"key2": "value2",
			"key3": "value3",
		}
		service3 := genMainService(3)
		service3.Metadata = map[string]string{"key3": "value3"}
		if resp := discoverSuit.DiscoverServer().CreateServices(discoverSuit.DefaultCtx, []*apiservice.Service{service1, service2, service3}); !respSuccess(resp) {
			t.Fatalf("error: %+v", resp)
		}
		defer discoverSuit.cleanServiceName(service1.GetName().GetValue(), service1.GetNamespace().GetValue())
		defer discoverSuit.cleanServiceName(service2.GetName().GetValue(), service2.GetNamespace().GetValue())
		defer discoverSuit.cleanServiceName(service3.GetName().GetValue(), service3.GetNamespace().GetValue())

		resps := discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, map[string]string{"keys": "key3", "values": "value3"})
		if len(resps.GetServices()) != 3 && resps.GetAmount().GetValue() != 3 {
			t.Fatalf("error: %d", len(resps.GetServices()))
		}
		resps = discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, map[string]string{"keys": "key2", "values": "value2"})
		if len(resps.GetServices()) != 2 && resps.GetAmount().GetValue() != 2 {
			t.Fatalf("error: %d", len(resps.GetServices()))
		}
		resps = discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, map[string]string{"keys": "key1", "values": "value1"})
		if len(resps.GetServices()) != 1 && resps.GetAmount().GetValue() != 1 {
			t.Fatalf("error: %d", len(resps.GetServices()))
		}
		resps = discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, map[string]string{"keys": "key1", "values": "value2"})
		if len(resps.GetServices()) != 0 && resps.GetAmount().GetValue() != 0 {
			t.Fatalf("error: %d", len(resps.GetServices()))
		}
	})
}

// 联合查询场景
func TestGetServices5(t *testing.T) {
	t.SkipNow()
	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	getServiceCheck := func(resp *apiservice.BatchQueryResponse, amount, size uint32) {
		t.Logf("gocheck resp: %v", resp)
		convey.So(respSuccess(resp), convey.ShouldEqual, true)
		convey.So(resp.GetAmount().GetValue(), convey.ShouldEqual, amount)
		convey.So(resp.GetSize().GetValue(), convey.ShouldEqual, size)
	}
	convey.Convey("支持host查询到服务", t, func() {
		_, serviceResp := discoverSuit.createCommonService(t, 200)
		defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
		instanceReq, instanceResp := discoverSuit.createCommonInstance(t, serviceResp, 100)
		defer discoverSuit.cleanInstance(instanceResp.GetId().GetValue())
		instanceReq, instanceResp = discoverSuit.createCommonInstance(t, serviceResp, 101)
		defer discoverSuit.cleanInstance(instanceResp.GetId().GetValue())
		query := map[string]string{
			"owner": "service-owner-200",
			"host":  instanceReq.GetHost().GetValue(),
		}
		convey.Convey("check-1", func() { getServiceCheck(discoverSuit.DiscoverServer().GetServices(context.Background(), query), 1, 1) })

		// 同host的实例，对应一个服务，那么返回值也是一个
		instanceReq.Port.Value = 999
		resp := discoverSuit.DiscoverServer().CreateInstances(discoverSuit.DefaultCtx, []*apiservice.Instance{instanceReq})
		convey.So(respSuccess(resp), convey.ShouldEqual, true)
		defer discoverSuit.cleanInstance(resp.Responses[0].Instance.GetId().GetValue())
		convey.Convey("check-2", func() { getServiceCheck(discoverSuit.DiscoverServer().GetServices(context.Background(), query), 1, 1) })
	})
	convey.Convey("支持host和port配合查询服务", t, func() {
		host1 := "127.0.0.1"
		port1 := uint32(8081)
		host2 := "127.0.0.2"
		port2 := uint32(8082)
		_, serviceResp1 := discoverSuit.createCommonService(t, 200)
		defer discoverSuit.cleanServiceName(serviceResp1.GetName().GetValue(), serviceResp1.GetNamespace().GetValue())
		_, instanceResp1 := discoverSuit.addHostPortInstance(t, serviceResp1, host1, port1)
		defer discoverSuit.cleanInstance(instanceResp1.GetId().GetValue())
		_, serviceResp2 := discoverSuit.createCommonService(t, 300)
		defer discoverSuit.cleanServiceName(serviceResp2.GetName().GetValue(), serviceResp2.GetNamespace().GetValue())
		_, instanceResp2 := discoverSuit.addHostPortInstance(t, serviceResp2, host1, port2)
		defer discoverSuit.cleanInstance(instanceResp2.GetId().GetValue())
		_, serviceResp3 := discoverSuit.createCommonService(t, 400)
		defer discoverSuit.cleanServiceName(serviceResp3.GetName().GetValue(), serviceResp3.GetNamespace().GetValue())
		_, instanceResp3 := discoverSuit.addHostPortInstance(t, serviceResp3, host2, port1)
		defer discoverSuit.cleanInstance(instanceResp3.GetId().GetValue())
		_, serviceResp4 := discoverSuit.createCommonService(t, 500)
		defer discoverSuit.cleanServiceName(serviceResp4.GetName().GetValue(), serviceResp4.GetNamespace().GetValue())
		_, instanceResp4 := discoverSuit.addHostPortInstance(t, serviceResp4, host2, port2)
		defer discoverSuit.cleanInstance(instanceResp4.GetId().GetValue())

		query := map[string]string{
			"host": host1,
			"port": strconv.Itoa(int(port1)),
		}
		convey.Convey("check-1-1", func() {
			getServiceCheck(
				discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, query), 1, 1)
		})
		query["host"] = host1 + "," + host2
		convey.Convey("check-2-1", func() {
			getServiceCheck(
				discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, query), 2, 2)
		})
		query["port"] = fmt.Sprintf("%d,%d", port1, port2)
		convey.Convey("check-2-2", func() {
			getServiceCheck(
				discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, query), 4, 4)
		})
	})
	convey.Convey("多个服务，对应同个host，返回多个服务", t, func() {
		count := 10
		var instance *apiservice.Instance
		for i := 0; i < count; i++ {
			_, serviceResp := discoverSuit.createCommonService(t, i)
			defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
			_, instanceResp := discoverSuit.createCommonInstance(t, serviceResp, 100)
			defer discoverSuit.cleanInstance(instanceResp.GetId().GetValue())
			instance = instanceResp
			_, instanceResp = discoverSuit.createCommonInstance(t, serviceResp, 202)
			defer discoverSuit.cleanInstance(instanceResp.GetId().GetValue())
		}
		query := map[string]string{
			"host":  instance.GetHost().GetValue(),
			"limit": "5",
		}
		convey.Convey("check-1", func() {
			getServiceCheck(
				discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, query), uint32(count), 5)
		})
	})
}

// 模糊匹配测试
func TestGetService6(t *testing.T) {
	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()
	t.Run("namespace模糊匹配过滤条件会生效", func(t *testing.T) {
		total := 60
		for i := 0; i < total; i++ {
			_, serviceResp := discoverSuit.createCommonService(t, i+100)
			defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
		}

		filters := map[string]string{"offset": "0",
			"limit":     "100",
			"namespace": "*ef*"}
		resp := discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, filters)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if len(resp.Services) != total {
			t.Fatalf("error: %d", len(resp.Services))
		}

		filters = map[string]string{"offset": "0",
			"limit":     "100",
			"namespace": "def*"}
		resp = discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, filters)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if len(resp.Services) != total {
			t.Fatalf("error: %d", len(resp.Services))
		}
	})

	t.Run("service模糊匹配过滤条件会生效", func(t *testing.T) {
		total := 60
		for i := 0; i < total; i++ {
			_, serviceResp := discoverSuit.createCommonService(t, i+200)
			defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
		}

		filters := map[string]string{"offset": "0",
			"limit": "100",
			"name":  "*est-service-21*"}
		resp := discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, filters)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if len(resp.Services) != 10 {
			t.Fatalf("error: %d", len(resp.Services))
		}
	})

	t.Run("instance_keys和instance_values模糊匹配过滤条件会生效", func(t *testing.T) {
		_, serviceResp := discoverSuit.createCommonService(t, 999)
		defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
		total := 10
		for i := 0; i < total; i++ {
			_, instanceResp := discoverSuit.createCommonInstance(t, serviceResp, i+100)
			defer discoverSuit.cleanInstance(instanceResp.GetId().GetValue())
		}

		filters := map[string]string{
			"offset":          "0",
			"limit":           "100",
			"instance_keys":   "2my-meta,my-meta-a1",
			"instance_values": "my-meta-100,111*",
		}
		resp := discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, filters)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if len(resp.Services) != 1 {
			t.Fatalf("error: %d", len(resp.Services))
		}
		if resp.Services[0].TotalInstanceCount.Value != uint32(total) {
			t.Fatalf("error: %d", resp.Services[0].TotalInstanceCount.Value)
		}

		filters = map[string]string{"offset": "0",
			"limit":           "100",
			"instance_keys":   "2my-meta,my-meta-a1,my-1meta-o3",
			"instance_values": "my-meta-100,1111,not-exists",
		}
		resp = discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, filters)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if len(resp.Services) != 0 {
			t.Fatalf("error: %d", len(resp.Services))
		}
	})

	t.Run("instance_keys和instance_values长度不相等会报错", func(t *testing.T) {
		filters := map[string]string{"offset": "0",
			"limit":           "100",
			"instance_keys":   "2my-meta,my-meta-a1",
			"instance_values": "my-meta-100,1111,oneMore",
		}
		resp := discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, filters)
		if resp.Code.Value != api.InvalidParameter {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})
}

// 测试更新服务
func TestUpdateService(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	_, serviceResp := discoverSuit.createCommonService(t, 200)
	defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
	t.Run("正常更新服务，所有属性都生效", func(t *testing.T) {
		updateReq := &apiservice.Service{
			Name:      serviceResp.Name,
			Namespace: serviceResp.Namespace,
			Metadata: map[string]string{
				"new-key":   "1",
				"new-key-2": "2",
				"new-key-3": "3",
			},
			Ports:      utils.NewStringValue("new-ports"),
			Business:   utils.NewStringValue("new-business"),
			Department: utils.NewStringValue("new-business"),
			CmdbMod1:   utils.NewStringValue("new-cmdb-mod1"),
			CmdbMod2:   utils.NewStringValue("new-cmdb-mo2"),
			CmdbMod3:   utils.NewStringValue("new-cmdb-mod3"),
			Comment:    utils.NewStringValue("new-comment"),
			Owners:     utils.NewStringValue("new-owner"),
			Token:      serviceResp.Token,
		}
		resp := discoverSuit.DiscoverServer().UpdateServices(discoverSuit.DefaultCtx, []*apiservice.Service{updateReq})
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}

		// get service
		query := map[string]string{
			"name":      updateReq.GetName().GetValue(),
			"namespace": updateReq.GetNamespace().GetValue(),
		}
		services := discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, query)
		if !respSuccess(services) {
			t.Fatalf("error: %s", services.GetInfo().GetValue())
		}
		if services.GetSize().GetValue() != 1 {
			t.Fatalf("error: %d", services.GetSize().GetValue())
		}

		serviceCheck(t, updateReq, services.GetServices()[0])
	})
	t.Run("更新服务，metadata数据个数太多，报错", func(t *testing.T) {
		serviceResp.Metadata = make(map[string]string)
		for i := 0; i < service.MaxMetadataLength+1; i++ {
			serviceResp.Metadata[fmt.Sprintf("update-%d", i)] = "abc"
		}
		if resp := discoverSuit.DiscoverServer().UpdateServices(discoverSuit.DefaultCtx, []*apiservice.Service{serviceResp}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error")
		}
	})
	t.Run("更新服务，metadata为空，长度为0，则删除所有metadata", func(t *testing.T) {
		serviceResp.Metadata = make(map[string]string)
		if resp := discoverSuit.DiscoverServer().UpdateServices(discoverSuit.DefaultCtx, []*apiservice.Service{serviceResp}); !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		getResp := discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, map[string]string{"name": serviceResp.Name.Value})
		if !respSuccess(getResp) {
			t.Fatalf("error: %s", getResp.GetInfo().GetValue())
		}
		if len(getResp.Services[0].Metadata) != 0 {
			t.Fatalf("error: %d", len(getResp.Services[0].Metadata))
		}
	})
	t.Run("更新服务，不允许更新别名", func(t *testing.T) {
		aliasResp := discoverSuit.createCommonAlias(serviceResp, "update.service.alias.xxx", defaultAliasNs, apiservice.AliasType_DEFAULT)
		defer discoverSuit.cleanServiceName(aliasResp.Alias.Alias.Value, serviceResp.Namespace.Value)

		aliasService := &apiservice.Service{
			Name:       aliasResp.Alias.Alias,
			Namespace:  serviceResp.Namespace,
			Department: utils.NewStringValue("123"),
			Token:      serviceResp.Token,
		}
		if resp := discoverSuit.DiscoverServer().UpdateServices(discoverSuit.DefaultCtx, []*apiservice.Service{aliasService}); respSuccess(resp) {
			t.Fatalf("error: update alias success")
		} else {
			t.Logf("update alias return: %s", resp.GetInfo().GetValue())
		}
	})
}

// 服务更新，noChange测试
func TestNoNeedUpdateService(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	_, serviceResp := discoverSuit.createCommonService(t, 500)
	defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
	t.Run("数据没有任意变更，返回不需要变更", func(t *testing.T) {
		resp := discoverSuit.DiscoverServer().UpdateServices(discoverSuit.DefaultCtx, []*apiservice.Service{serviceResp})
		if resp.GetCode().GetValue() != api.NoNeedUpdate {
			t.Fatalf("error: %+v", resp)
		}
	})
	req := &apiservice.Service{
		Name:      serviceResp.Name,
		Namespace: serviceResp.Namespace,
		Token:     serviceResp.Token,
	}
	t.Run("metadata为空，不需要变更", func(t *testing.T) {
		req.Metadata = nil
		if resp := discoverSuit.DiscoverServer().UpdateServices(discoverSuit.DefaultCtx, []*apiservice.Service{req}); resp.GetCode().GetValue() != api.NoNeedUpdate {
			t.Fatalf("error: %+v", resp)
		}
		req.Comment = serviceResp.Comment
		if resp := discoverSuit.DiscoverServer().UpdateServices(discoverSuit.DefaultCtx, []*apiservice.Service{req}); resp.GetCode().GetValue() != api.NoNeedUpdate {
			t.Fatalf("error: %+v", resp)
		}
	})
	t.Run("metadata不为空，但是没变更，也不需要更新", func(t *testing.T) {
		req.Metadata = serviceResp.Metadata
		if resp := discoverSuit.DiscoverServer().UpdateServices(discoverSuit.DefaultCtx, []*apiservice.Service{req}); resp.GetCode().GetValue() != api.NoNeedUpdate {
			t.Fatalf("error: %+v", resp)
		}
	})
	t.Run("其他字段更新，metadata没有更新，不需要更新metadata", func(t *testing.T) {
		req.Metadata = serviceResp.Metadata
		req.Comment = utils.NewStringValue("1357986420")
		if resp := discoverSuit.DiscoverServer().UpdateServices(discoverSuit.DefaultCtx, []*apiservice.Service{req}); resp.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
			t.Fatalf("error: %+v", resp)
		}
	})
	t.Run("只有一个字段变更，service就执行变更操作", func(t *testing.T) {
		baseReq := apiservice.Service{
			Name:      serviceResp.Name,
			Namespace: serviceResp.Namespace,
			Token:     serviceResp.Token,
		}

		r := baseReq
		r.Ports = utils.NewStringValue("90909090")
		if resp := discoverSuit.DiscoverServer().UpdateServices(discoverSuit.DefaultCtx, []*apiservice.Service{&r}); resp.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
			t.Fatalf("error: %+v", resp)
		}

		r = baseReq
		r.Business = utils.NewStringValue("new-business")
		if resp := discoverSuit.DiscoverServer().UpdateServices(discoverSuit.DefaultCtx, []*apiservice.Service{&r}); resp.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
			t.Fatalf("error: %+v", resp)
		}

		r = baseReq
		r.Department = utils.NewStringValue("new-department-1")
		if resp := discoverSuit.DiscoverServer().UpdateServices(discoverSuit.DefaultCtx, []*apiservice.Service{&r}); resp.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
			t.Fatalf("error: %+v", resp)
		}

		r = baseReq
		r.CmdbMod1 = utils.NewStringValue("new-CmdbMod1-1")
		if resp := discoverSuit.DiscoverServer().UpdateServices(discoverSuit.DefaultCtx, []*apiservice.Service{&r}); resp.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
			t.Fatalf("error: %+v", resp)
		}

		r = baseReq
		r.CmdbMod2 = utils.NewStringValue("new-CmdbMod2-1")
		if resp := discoverSuit.DiscoverServer().UpdateServices(discoverSuit.DefaultCtx, []*apiservice.Service{&r}); resp.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
			t.Fatalf("error: %+v", resp)
		}

		r = baseReq
		r.CmdbMod3 = utils.NewStringValue("new-CmdbMod3-1")
		if resp := discoverSuit.DiscoverServer().UpdateServices(discoverSuit.DefaultCtx, []*apiservice.Service{&r}); resp.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
			t.Fatalf("error: %+v", resp)
		}

		r = baseReq
		r.Comment = utils.NewStringValue("new-Comment-1")
		if resp := discoverSuit.DiscoverServer().UpdateServices(discoverSuit.DefaultCtx, []*apiservice.Service{&r}); resp.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
			t.Fatalf("error: %+v", resp)
		}

		r = baseReq
		r.Owners = utils.NewStringValue("new-Owners-1")
		if resp := discoverSuit.DiscoverServer().UpdateServices(discoverSuit.DefaultCtx, []*apiservice.Service{&r}); resp.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
			t.Fatalf("error: %+v", resp)
		}
	})
}

// 测试serviceToken相关的操作
func TestServiceToken(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	_, serviceResp := discoverSuit.createCommonService(t, 200)
	defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
	t.Run("可以正常获取serviceToken", func(t *testing.T) {
		req := &apiservice.Service{
			Name:      serviceResp.GetName(),
			Namespace: serviceResp.GetNamespace(),
			Token:     serviceResp.GetToken(),
		}

		resp := discoverSuit.DiscoverServer().GetServiceToken(discoverSuit.DefaultCtx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if resp.GetService().GetToken().GetValue() != serviceResp.GetToken().GetValue() {
			t.Fatalf("error")
		}
	})

	t.Run("获取别名的token，返回源服务的token", func(t *testing.T) {
		aliasResp := discoverSuit.createCommonAlias(serviceResp, fmt.Sprintf("get.token.xxx-%s", utils.NewUUID()[:8]), defaultAliasNs, apiservice.AliasType_DEFAULT)
		defer discoverSuit.cleanServiceName(aliasResp.Alias.Alias.Value, serviceResp.Namespace.Value)
		t.Logf("%+v", aliasResp)

		req := &apiservice.Service{
			Name:      aliasResp.Alias.Alias,
			Namespace: aliasResp.Alias.AliasNamespace,
			Token:     serviceResp.GetToken(),
		}
		t.Logf("%+v", req)
		if resp := discoverSuit.DiscoverServer().GetServiceToken(discoverSuit.DefaultCtx, req); !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		} else if resp.GetService().GetToken().GetValue() != serviceResp.GetToken().GetValue() {
			t.Fatalf("error")
		}
	})

	t.Run("可以正常更新serviceToken", func(t *testing.T) {
		resp := discoverSuit.DiscoverServer().UpdateServiceToken(discoverSuit.DefaultCtx, serviceResp)
		if !respSuccess(resp) {
			t.Fatalf("error :%s", resp.GetInfo().GetValue())
		}
		if resp.GetService().GetToken().GetValue() == serviceResp.GetToken().GetValue() {
			t.Fatalf("error: %s %s", resp.GetService().GetToken().GetValue(),
				serviceResp.GetToken().GetValue())
		}
		serviceResp.Token.Value = resp.Service.Token.Value // set token
	})

	t.Run("alias不允许更新token", func(t *testing.T) {
		aliasResp := discoverSuit.createCommonAlias(serviceResp, "update.token.xxx", defaultAliasNs, apiservice.AliasType_DEFAULT)
		defer discoverSuit.cleanServiceName(aliasResp.Alias.Alias.Value, serviceResp.Namespace.Value)

		req := &apiservice.Service{
			Name:      aliasResp.Alias.Alias,
			Namespace: serviceResp.Namespace,
			Token:     serviceResp.Token,
		}
		if resp := discoverSuit.DiscoverServer().UpdateServiceToken(discoverSuit.DefaultCtx, req); respSuccess(resp) {
			t.Fatalf("error")
		}
	})
}

// 测试response格式化
func TestFormatBatchWriteResponse(t *testing.T) {
	t.Run("同样的错误码，返回一个错误码4XX", func(t *testing.T) {
		responses := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
		for i := 0; i < 10; i++ {
			api.Collect(responses, api.NewResponse(apimodel.Code_NotFoundService))
		}

		responses = api.FormatBatchWriteResponse(responses)
		if responses.GetCode().GetValue() != uint32(apimodel.Code_NotFoundService) {
			t.Fatalf("%+v", responses)
		}
	})
	t.Run("同样的错误码，返回一个错误码5XX", func(t *testing.T) {
		responses := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
		for i := 0; i < 10; i++ {
			api.Collect(responses, api.NewResponse(apimodel.Code_StoreLayerException))
		}

		responses = api.FormatBatchWriteResponse(responses)
		if responses.GetCode().GetValue() != uint32(apimodel.Code_StoreLayerException) {
			t.Fatalf("%+v", responses)
		}
	})
	t.Run("有5XX和2XX，返回5XX", func(t *testing.T) {
		responses := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
		api.Collect(responses, api.NewResponse(apimodel.Code_ExecuteSuccess))
		api.Collect(responses, api.NewResponse(apimodel.Code_NotFoundNamespace))
		api.Collect(responses, api.NewResponse(apimodel.Code_ParseRateLimitException))
		api.Collect(responses, api.NewResponse(apimodel.Code_ParseException))
		responses = api.FormatBatchWriteResponse(responses)
		if responses.GetCode().GetValue() != api.ExecuteException {
			t.Fatalf("%+v", responses)
		}
	})
	t.Run("没有5XX，有4XX，返回4XX", func(t *testing.T) {
		responses := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
		api.Collect(responses, api.NewResponse(apimodel.Code_ExecuteSuccess))
		api.Collect(responses, api.NewResponse(apimodel.Code_NotFoundNamespace))
		api.Collect(responses, api.NewResponse(apimodel.Code_NoNeedUpdate))
		api.Collect(responses, api.NewResponse(apimodel.Code_InvalidInstanceID))
		api.Collect(responses, api.NewResponse(apimodel.Code_ExecuteSuccess))
		responses = api.FormatBatchWriteResponse(responses)
		if responses.GetCode().GetValue() != api.BadRequest {
			t.Fatalf("%+v", responses)
		}
	})
	t.Run("全是2XX", func(t *testing.T) {
		responses := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
		api.Collect(responses, api.NewResponse(apimodel.Code_ExecuteSuccess))
		api.Collect(responses, api.NewResponse(apimodel.Code_NoNeedUpdate))
		api.Collect(responses, api.NewResponse(apimodel.Code_DataNoChange))
		api.Collect(responses, api.NewResponse(apimodel.Code_NoNeedUpdate))
		api.Collect(responses, api.NewResponse(apimodel.Code_ExecuteSuccess))
		responses = api.FormatBatchWriteResponse(responses)
		if responses.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
			t.Fatalf("%+v", responses)
		}
	})
}

// test对service字段进行校验
func TestCheckServiceFieldLen(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	service := genMainService(400)
	t.Run("服务名超长", func(t *testing.T) {
		str := genSpecialStr(129)
		oldName := service.Name
		service.Name = utils.NewStringValue(str)
		resp := discoverSuit.DiscoverServer().CreateServices(discoverSuit.DefaultCtx, []*apiservice.Service{service})
		service.Name = oldName
		if resp.Code.Value != api.InvalidServiceName {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("命名空间超长", func(t *testing.T) {
		str := genSpecialStr(129)
		oldNameSpace := service.Namespace
		service.Namespace = utils.NewStringValue(str)
		resp := discoverSuit.DiscoverServer().CreateServices(discoverSuit.DefaultCtx, []*apiservice.Service{service})
		service.Namespace = oldNameSpace
		if resp.Code.Value != api.InvalidNamespaceName {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("Metadata超长", func(t *testing.T) {
		str := genSpecialStr(129)
		oldMetadata := service.Metadata
		oldMetadata[str] = str
		resp := discoverSuit.DiscoverServer().CreateServices(discoverSuit.DefaultCtx, []*apiservice.Service{service})
		service.Metadata = make(map[string]string)
		if resp.Code.Value != api.InvalidMetadata {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("服务ports超长", func(t *testing.T) {
		str := genSpecialStr(8193)
		oldPort := service.Ports
		service.Ports = utils.NewStringValue(str)
		resp := discoverSuit.DiscoverServer().CreateServices(discoverSuit.DefaultCtx, []*apiservice.Service{service})
		service.Ports = oldPort
		if resp.Code.Value != api.InvalidServicePorts {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("服务Business超长", func(t *testing.T) {
		str := genSpecialStr(129)
		oldBusiness := service.Business
		service.Business = utils.NewStringValue(str)
		resp := discoverSuit.DiscoverServer().CreateServices(discoverSuit.DefaultCtx, []*apiservice.Service{service})
		service.Business = oldBusiness
		if resp.Code.Value != api.InvalidServiceBusiness {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("服务-部门超长", func(t *testing.T) {
		str := genSpecialStr(1025)
		oldDepartment := service.Department
		service.Department = utils.NewStringValue(str)
		resp := discoverSuit.DiscoverServer().CreateServices(discoverSuit.DefaultCtx, []*apiservice.Service{service})
		service.Department = oldDepartment
		if resp.Code.Value != api.InvalidServiceDepartment {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("服务cmdb超长", func(t *testing.T) {
		str := genSpecialStr(1025)
		oldCMDB := service.CmdbMod1
		service.CmdbMod1 = utils.NewStringValue(str)
		resp := discoverSuit.DiscoverServer().CreateServices(discoverSuit.DefaultCtx, []*apiservice.Service{service})
		service.CmdbMod1 = oldCMDB
		if resp.Code.Value != api.InvalidServiceCMDB {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("服务comment超长", func(t *testing.T) {
		str := genSpecialStr(1025)
		oldComment := service.Comment
		service.Comment = utils.NewStringValue(str)
		resp := discoverSuit.DiscoverServer().CreateServices(discoverSuit.DefaultCtx, []*apiservice.Service{service})
		service.Comment = oldComment
		if resp.Code.Value != api.InvalidServiceComment {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("服务token超长", func(t *testing.T) {
		str := genSpecialStr(2049)
		oldToken := service.Token
		service.Token = utils.NewStringValue(str)
		resp := discoverSuit.DiscoverServer().CreateServices(discoverSuit.DefaultCtx, []*apiservice.Service{service})
		service.Token = oldToken
		if resp.Code.Value != api.InvalidServiceToken {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("检测字段为空指针", func(t *testing.T) {
		oldName := service.Name
		service.Name = nil
		resp := discoverSuit.DiscoverServer().CreateServices(discoverSuit.DefaultCtx, []*apiservice.Service{service})
		service.Name = oldName
		if resp.Code.Value != api.InvalidServiceName {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("检测字段为空", func(t *testing.T) {
		oldName := service.Name
		service.Name = utils.NewStringValue("")
		resp := discoverSuit.DiscoverServer().CreateServices(discoverSuit.DefaultCtx, []*apiservice.Service{service})
		service.Name = oldName
		if resp.Code.Value != api.InvalidServiceName {
			t.Fatalf("%+v", resp)
		}
	})
}

func TestConcurrencyCreateSameService(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx, cancel := context.WithCancel(context.Background())

	t.Cleanup(func() {
		cancel()
		ctrl.Finish()
	})

	createMockResource := func() (*service.Server, *mock.MockStore) {
		var (
			err      error
			cacheMgr *cache.CacheManager
			nsSvr    namespace.NamespaceOperateServer
		)

		mockStore := mock.NewMockStore(ctrl)
		mockStore.EXPECT().GetMoreNamespaces(gomock.Any()).Return([]*model.Namespace{
			{
				Name: "mock_ns",
			},
		}, nil).AnyTimes()
		mockStore.EXPECT().GetUnixSecond(gomock.Any()).Return(time.Now().Unix(), nil).AnyTimes()
		cacheMgr, err = cache.TestCacheInitialize(ctx, &cache.Config{}, mockStore)
		assert.NoError(t, err)

		userMgn, strategyMgn, err := auth.TestInitialize(ctx, &auth.Config{}, mockStore, cacheMgr)
		assert.NoError(t, err)

		nsSvr, err = namespace.TestInitialize(ctx, &namespace.Config{
			AutoCreate: true,
		}, mockStore, cacheMgr, userMgn, strategyMgn)
		assert.NoError(t, err)

		cacheMgr.OpenResourceCache([]cachetypes.ConfigEntry{
			{
				Name: "namespace",
			},
		}...)
		svr := service.TestNewServer(mockStore, nsSvr, cacheMgr)
		return svr, mockStore
	}

	var (
		req = &apiservice.Service{
			Namespace: &wrapperspb.StringValue{
				Value: "test_ns",
			},
			Name: &wrapperspb.StringValue{
				Value: "test_svc",
			},
		}
	)

	t.Run("正常创建服务", func(t *testing.T) {
		svr, mockStore := createMockResource()

		mockStore.EXPECT().GetNamespace(gomock.Any()).Return(&model.Namespace{
			Name: "mock_ns",
		}, nil).AnyTimes()
		mockStore.EXPECT().GetService(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
		mockStore.EXPECT().AddService(gomock.Any()).Return(nil).AnyTimes()

		resp := svr.CreateService(context.TODO(), req)
		assert.Equal(t, apimodel.Code_ExecuteSuccess, apimodel.Code(resp.GetCode().GetValue()))
		assert.True(t, len(resp.GetService().GetId().GetValue()) > 0)
	})

	t.Run("正常创建服务-目标服务已存在", func(t *testing.T) {
		svr, mockStore := createMockResource()
		mockStore.EXPECT().GetNamespace(gomock.Any()).Return(&model.Namespace{
			Name: "mock_ns",
		}, nil).AnyTimes()
		mockStore.EXPECT().GetService(gomock.Any(), gomock.Any()).Return(&model.Service{
			ID: "mock_svc_id",
		}, nil).AnyTimes()

		resp := svr.CreateService(context.TODO(), req)
		assert.Equal(t, apimodel.Code_ExistedResource, apimodel.Code(resp.GetCode().GetValue()))
		assert.True(t, len(resp.GetService().GetId().GetValue()) > 0)
	})

	t.Run("正常创建服务-存储层主键冲突", func(t *testing.T) {
		svr, mockStore := createMockResource()
		mockStore.EXPECT().GetNamespace(gomock.Any()).Return(&model.Namespace{
			Name: "mock_ns",
		}, nil).AnyTimes()

		var (
			execTime  int32
			mockSvcId = "mock_svc_id"
		)

		mockStore.EXPECT().GetService(gomock.Any(), gomock.Any()).DoAndReturn(func(_, _ string) (*model.Service, error) {
			execTime++
			if execTime == 1 {
				return nil, nil
			}
			if execTime == 2 {
				return &model.Service{ID: mockSvcId}, nil
			}
			return nil, errors.New("run to many times")
		}).AnyTimes()
		mockStore.EXPECT().AddService(gomock.Any()).
			Return(store.NewStatusError(store.DuplicateEntryErr, "mock duplicate error")).AnyTimes()

		resp := svr.CreateService(context.TODO(), req)
		assert.Equal(t, apimodel.Code_ExistedResource, apimodel.Code(resp.GetCode().GetValue()))
		assert.Equal(t, mockSvcId, resp.GetService().GetId().GetValue())
	})
}

func Test_ServiceVisible(t *testing.T) {
	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}

	service := genMainService(int(time.Now().Unix()))

	t.Cleanup(func() {
		discoverSuit.cleanNamespace(service.GetNamespace().GetValue())
		discoverSuit.cleanAllService()
		discoverSuit.Destroy()
	})

	t.Run("创建服务时指定可见性", func(t *testing.T) {
		service.ExportTo = []*wrapperspb.StringValue{wrapperspb.String("mock_namespace")}
		resp := discoverSuit.DiscoverServer().CreateServices(discoverSuit.DefaultCtx, []*apiservice.Service{service})
		assert.Equal(t, apimodel.Code_ExecuteSuccess, apimodel.Code(resp.GetCode().GetValue()))

		_ = discoverSuit.CacheMgr().TestUpdate()

		rsp := discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, map[string]string{
			"name":      service.GetName().GetValue(),
			"namespace": service.GetNamespace().GetValue(),
		})
		assert.Equal(t, apimodel.Code_ExecuteSuccess, apimodel.Code(resp.GetCode().GetValue()))
		assert.True(t, len(rsp.GetServices()) == 1)
		assert.True(t, len(rsp.GetServices()[0].GetExportTo()) == 1)
		assert.Equal(t, model.ExportToMap([]*wrappers.StringValue{wrapperspb.String("mock_namespace")}),
			model.ExportToMap(rsp.GetServices()[0].GetExportTo()))
	})

	t.Run("修改服务时指定可见性", func(t *testing.T) {
		service.ExportTo = []*wrapperspb.StringValue{wrapperspb.String("mock_ns_1"), wrapperspb.String("mock_ns_2")}
		resp := discoverSuit.DiscoverServer().UpdateServices(discoverSuit.DefaultCtx, []*apiservice.Service{service})
		assert.Equal(t, apimodel.Code_ExecuteSuccess, apimodel.Code(resp.GetCode().GetValue()))

		_ = discoverSuit.CacheMgr().TestUpdate()

		rsp := discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, map[string]string{
			"name":      service.GetName().GetValue(),
			"namespace": service.GetNamespace().GetValue(),
		})
		assert.Equal(t, apimodel.Code_ExecuteSuccess, apimodel.Code(resp.GetCode().GetValue()))
		assert.True(t, len(rsp.GetServices()) == 1)
		assert.True(t, len(rsp.GetServices()[0].GetExportTo()) == 2)
		assert.Equal(t, model.ExportToMap([]*wrapperspb.StringValue{wrapperspb.String("mock_ns_1"), wrapperspb.String("mock_ns_2")}),
			model.ExportToMap(rsp.GetServices()[0].GetExportTo()))
	})

	t.Run("清空服务可见性", func(t *testing.T) {
		service.ExportTo = []*wrappers.StringValue{}
		resp := discoverSuit.DiscoverServer().UpdateServices(discoverSuit.DefaultCtx, []*apiservice.Service{service})
		assert.Equal(t, apimodel.Code_ExecuteSuccess, apimodel.Code(resp.GetCode().GetValue()))

		_ = discoverSuit.CacheMgr().TestUpdate()

		rsp := discoverSuit.DiscoverServer().GetServices(discoverSuit.DefaultCtx, map[string]string{
			"name":      service.GetName().GetValue(),
			"namespace": service.GetNamespace().GetValue(),
		})
		assert.Equal(t, apimodel.Code_ExecuteSuccess, apimodel.Code(resp.GetCode().GetValue()))
		assert.True(t, len(rsp.GetServices()) == 1)
		assert.True(t, len(rsp.GetServices()[0].GetExportTo()) == 0)
	})
}

func Test_NamespaceVisible(t *testing.T) {
	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}

	nsVal := &apimodel.Namespace{
		Name: wrapperspb.String(fmt.Sprintf("mock_ns_%d", time.Now().Unix())),
	}

	t.Cleanup(func() {
		discoverSuit.cleanNamespace(nsVal.GetName().GetValue())
		discoverSuit.Destroy()
	})

	t.Run("创建命名空间时指定可见性", func(t *testing.T) {
		nsVal.ServiceExportTo = []*wrapperspb.StringValue{wrapperspb.String("mock_namespace")}
		resp := discoverSuit.NamespaceServer().CreateNamespace(discoverSuit.DefaultCtx, nsVal)
		assert.Equal(t, apimodel.Code_ExecuteSuccess, apimodel.Code(resp.GetCode().GetValue()))

		_ = discoverSuit.CacheMgr().TestUpdate()

		rsp := discoverSuit.NamespaceServer().GetNamespaces(discoverSuit.DefaultCtx, map[string][]string{
			"name": {nsVal.GetName().GetValue()},
		})
		assert.Equal(t, apimodel.Code_ExecuteSuccess, apimodel.Code(resp.GetCode().GetValue()))
		assert.True(t, len(rsp.GetNamespaces()) == 1)
		assert.True(t, len(rsp.GetNamespaces()[0].GetServiceExportTo()) == 1)
		assert.Equal(t, model.ExportToMap([]*wrappers.StringValue{wrapperspb.String("mock_namespace")}),
			model.ExportToMap(rsp.GetNamespaces()[0].GetServiceExportTo()))
	})

	t.Run("修改命名空间时指定可见性", func(t *testing.T) {
		nsVal.ServiceExportTo = []*wrapperspb.StringValue{wrapperspb.String("mock_ns_1"), wrapperspb.String("mock_ns_2")}
		resp := discoverSuit.NamespaceServer().UpdateNamespaces(discoverSuit.DefaultCtx, []*apimodel.Namespace{nsVal})
		assert.Equal(t, apimodel.Code_ExecuteSuccess, apimodel.Code(resp.GetCode().GetValue()))

		_ = discoverSuit.CacheMgr().TestUpdate()

		rsp := discoverSuit.NamespaceServer().GetNamespaces(discoverSuit.DefaultCtx, map[string][]string{
			"name": {nsVal.GetName().GetValue()},
		})
		assert.Equal(t, apimodel.Code_ExecuteSuccess, apimodel.Code(resp.GetCode().GetValue()))
		assert.True(t, len(rsp.GetNamespaces()) == 1)
		assert.True(t, len(rsp.GetNamespaces()[0].GetServiceExportTo()) == 2)
		assert.Equal(t, model.ExportToMap([]*wrapperspb.StringValue{wrapperspb.String("mock_ns_1"), wrapperspb.String("mock_ns_2")}),
			model.ExportToMap(rsp.GetNamespaces()[0].GetServiceExportTo()))
	})

	t.Run("清空命名空间可见性", func(t *testing.T) {
		nsVal.ServiceExportTo = []*wrappers.StringValue{}
		resp := discoverSuit.NamespaceServer().UpdateNamespaces(discoverSuit.DefaultCtx, []*apimodel.Namespace{nsVal})
		assert.Equal(t, apimodel.Code_ExecuteSuccess, apimodel.Code(resp.GetCode().GetValue()))

		_ = discoverSuit.CacheMgr().TestUpdate()

		rsp := discoverSuit.NamespaceServer().GetNamespaces(discoverSuit.DefaultCtx, map[string][]string{
			"name": []string{nsVal.GetName().GetValue()},
		})
		assert.Equal(t, apimodel.Code_ExecuteSuccess, apimodel.Code(resp.GetCode().GetValue()))
		assert.True(t, len(rsp.GetNamespaces()) == 1)
		assert.True(t, len(rsp.GetNamespaces()[0].GetServiceExportTo()) == 0)
	})
}
