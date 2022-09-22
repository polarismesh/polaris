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
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/anypb"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	apiv2 "github.com/polarismesh/polaris-server/common/api/v2"
	"github.com/polarismesh/polaris-server/common/utils"
)

// checkSameRoutingConfigV2 检查routingConfig前后是否一致
func checkSameRoutingConfigV2V2(t *testing.T, lhs []*apiv2.Routing, rhs []*apiv2.Routing) {
	if len(lhs) != len(rhs) {
		t.Fatal("error: len(lhs) != len(rhs)")
	}
}

// TestCreateRoutingConfigV2 测试创建路由配置
func TestCreateRoutingConfigV2(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	t.Run("正常创建路由配置配置请求", func(t *testing.T) {
		req := discoverSuit.createCommonRoutingConfigV2(t, 3)
		defer discoverSuit.cleanCommonRoutingConfigV2(req)

		// 对写进去的数据进行查询
		time.Sleep(discoverSuit.updateCacheInterval * 5)
		out := discoverSuit.server.GetRoutingConfigsV2(discoverSuit.defaultCtx, map[string]string{
			"limit":  "100",
			"offset": "0",
		})
		if !respSuccessV2(out) {
			t.Fatalf("error: %+v", out)
		}

		// assert.Equal(t, int(3), int(out.Amount), "query routing size")
	})
}

// TestDeleteRoutingConfigV2 测试删除路由配置
func TestDeleteRoutingConfigV2(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	t.Run("可以正常删除路由配置", func(t *testing.T) {
		resp := discoverSuit.createCommonRoutingConfigV2(t, 1)
		discoverSuit.deleteCommonRoutingConfigV2(t, resp[0])
		defer discoverSuit.cleanCommonRoutingConfigV2(resp)

		serviceName := fmt.Sprintf("in-source-service-%d", 0)
		namespaceName := fmt.Sprintf("in-source-service-%d", 0)

		// 删除之后，数据不见
		time.Sleep(discoverSuit.updateCacheInterval)
		out := discoverSuit.server.GetRoutingConfigWithCache(discoverSuit.defaultCtx, &api.Service{
			Name:      utils.NewStringValue(serviceName),
			Namespace: utils.NewStringValue(namespaceName),
		})
		assert.Nil(t, out.GetRouting())
	})
}

// TestUpdateRoutingConfigV2 测试更新路由配置
func TestUpdateRoutingConfigV2(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	t.Run("可以正常更新路由配置", func(t *testing.T) {
		req := discoverSuit.createCommonRoutingConfigV2(t, 1)
		defer discoverSuit.cleanCommonRoutingConfigV2(req)
		// 对写进去的数据进行查询
		time.Sleep(discoverSuit.updateCacheInterval)
		out := discoverSuit.server.GetRoutingConfigsV2(discoverSuit.defaultCtx, map[string]string{
			"limit":  "100",
			"offset": "0",
		})
		if !respSuccessV2(out) {
			t.Fatalf("error: %+v", out)
		}

		assert.Equal(t, uint32(1), out.Size, "query routing size")

		ret, err := unmarshalRoutingV2toAnySlice(out.GetData())
		assert.NoError(t, err)
		routing := ret[0]

		updateName := "update routing second"
		routing.Name = updateName

		discoverSuit.server.UpdateRoutingConfigsV2(discoverSuit.defaultCtx, []*apiv2.Routing{routing})
		time.Sleep(discoverSuit.updateCacheInterval)
		out = discoverSuit.server.GetRoutingConfigsV2(discoverSuit.defaultCtx, map[string]string{
			"limit":  "100",
			"offset": "0",
			"id":     routing.Id,
		})

		if !respSuccessV2(out) {
			t.Fatalf("error: %+v", out)
		}

		assert.Equal(t, uint32(1), out.Size, "query routing size")
		ret, err = unmarshalRoutingV2toAnySlice(out.GetData())
		assert.NoError(t, err)
		assert.Equal(t, updateName, ret[0].Name)
	})
}

// test对routing字段进行校验
func TestCreateCheckRoutingFieldLenV2(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	any, _ := ptypes.MarshalAny(&apiv2.RuleRoutingConfig{})

	req := &apiv2.Routing{
		Id:            "",
		Name:          "test-routing",
		Namespace:     "",
		Enable:        false,
		RoutingPolicy: apiv2.RoutingPolicy_RulePolicy,
		RoutingConfig: any,
		Revision:      "",
		Ctime:         "",
		Mtime:         "",
		Etime:         "",
		Priority:      0,
		Description:   "",
		ExtendInfo: map[string]string{
			"": "",
		},
	}

	t.Run("创建路由规则，规则名称超长", func(t *testing.T) {
		str := genSpecialStr(129)
		oldName := req.Name
		req.Name = str
		resp := discoverSuit.server.CreateRoutingConfigsV2(discoverSuit.defaultCtx, []*apiv2.Routing{req})
		req.Name = oldName
		if resp.Code != api.InvalidRoutingName {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("创建路由规则，路由规则类型不正确", func(t *testing.T) {
		oldPolicy := req.RoutingPolicy
		req.RoutingPolicy = apiv2.RoutingPolicy(123)
		resp := discoverSuit.server.CreateRoutingConfigsV2(discoverSuit.defaultCtx, []*apiv2.Routing{req})
		req.RoutingPolicy = oldPolicy
		if resp.Code != api.InvalidRoutingPolicy {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("创建路由规则，路由规则类型不正确", func(t *testing.T) {
		oldPolicy := req.RoutingPolicy
		req.RoutingPolicy = apiv2.RoutingPolicy_MetadataPolicy
		resp := discoverSuit.server.CreateRoutingConfigsV2(discoverSuit.defaultCtx, []*apiv2.Routing{req})
		req.RoutingPolicy = oldPolicy
		if resp.Code != api.InvalidRoutingPolicy {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("创建路由规则，路由优先级不正确", func(t *testing.T) {
		oldPriority := req.Priority
		req.Priority = 11
		resp := discoverSuit.server.CreateRoutingConfigsV2(discoverSuit.defaultCtx, []*apiv2.Routing{req})
		req.Priority = oldPriority
		if resp.Code != api.InvalidRoutingPriority {
			t.Fatalf("%+v", resp)
		}
	})
}

// TestUpdateCheckRoutingFieldLenV2 test对routing字段进行校验
func TestUpdateCheckRoutingFieldLenV2(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	any, _ := ptypes.MarshalAny(&apiv2.RuleRoutingConfig{})

	req := &apiv2.Routing{
		Id:            "12312312312312313",
		Name:          "test-routing",
		Namespace:     "",
		Enable:        false,
		RoutingPolicy: apiv2.RoutingPolicy_RulePolicy,
		RoutingConfig: any,
		Revision:      "",
		Ctime:         "",
		Mtime:         "",
		Etime:         "",
		Priority:      0,
		Description:   "",
		ExtendInfo: map[string]string{
			"": "",
		},
	}

	t.Run("更新路由规则，规则ID为空", func(t *testing.T) {
		oldId := req.Id
		req.Id = ""
		resp := discoverSuit.server.UpdateRoutingConfigsV2(discoverSuit.defaultCtx, []*apiv2.Routing{req})
		req.Id = oldId
		if resp.Code != api.InvalidRoutingID {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("更新路由规则，规则名称超长", func(t *testing.T) {
		str := genSpecialStr(129)
		oldName := req.Name
		req.Name = str
		resp := discoverSuit.server.UpdateRoutingConfigsV2(discoverSuit.defaultCtx, []*apiv2.Routing{req})
		req.Name = oldName
		if resp.Code != api.InvalidRoutingName {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("更新路由规则，路由规则类型不正确", func(t *testing.T) {
		oldPolicy := req.RoutingPolicy
		req.RoutingPolicy = apiv2.RoutingPolicy(123)
		resp := discoverSuit.server.UpdateRoutingConfigsV2(discoverSuit.defaultCtx, []*apiv2.Routing{req})
		req.RoutingPolicy = oldPolicy
		if resp.Code != api.InvalidRoutingPolicy {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("更新路由规则，路由规则类型不正确", func(t *testing.T) {
		oldPolicy := req.RoutingPolicy
		req.RoutingPolicy = apiv2.RoutingPolicy_MetadataPolicy
		resp := discoverSuit.server.UpdateRoutingConfigsV2(discoverSuit.defaultCtx, []*apiv2.Routing{req})
		req.RoutingPolicy = oldPolicy
		if resp.Code != api.InvalidRoutingPolicy {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("更新路由规则，路由优先级不正确", func(t *testing.T) {
		oldPriority := req.Priority
		req.Priority = 11
		resp := discoverSuit.server.UpdateRoutingConfigsV2(discoverSuit.defaultCtx, []*apiv2.Routing{req})
		req.Priority = oldPriority
		if resp.Code != api.InvalidRoutingPriority {
			t.Fatalf("%+v", resp)
		}
	})
}

// marshalRoutingV2toAnySlice 转换为 []*apiv2.Routing 数组
func unmarshalRoutingV2toAnySlice(routings []*anypb.Any) ([]*apiv2.Routing, error) {
	ret := make([]*apiv2.Routing, 0, len(routings))

	for i := range routings {
		entry := routings[i]

		msg := &apiv2.Routing{}
		if err := ptypes.UnmarshalAny(entry, msg); err != nil {
			return nil, err
		}

		ret = append(ret, msg)
	}

	return ret, nil
}
