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

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/utils"
)

// create
func (d *DiscoverTestSuit) createCommonNamespace(t *testing.T, id int) (*api.Namespace, *api.Namespace) {
	req := &api.Namespace{
		Name:    utils.NewStringValue(fmt.Sprintf("namespace-%d", id)),
		Comment: utils.NewStringValue(fmt.Sprintf("comment-%d", id)),
		Owners:  utils.NewStringValue(fmt.Sprintf("owner-%d", id)),
	}
	d.cleanNamespace(req.GetName().GetValue())

	resp := d.namespaceSvr.CreateNamespace(d.defaultCtx, req)
	if !respSuccess(resp) {
		t.Fatalf("error: %s", resp.GetInfo().GetValue())
	}

	if resp.GetNamespace().GetToken().GetValue() == "" ||
		req.GetName().GetValue() != resp.GetNamespace().GetName().GetValue() {
		t.Fatalf("errors: %+v", resp)
	}

	return req, resp.GetNamespace()
}

// remove
func (d *DiscoverTestSuit) removeCommonNamespaces(t *testing.T, req []*api.Namespace) {
	resp := d.namespaceSvr.DeleteNamespaces(d.defaultCtx, req)
	if !respSuccess(resp) {
		t.Fatalf("error: %s", resp.GetInfo().GetValue())
	}
}

// update
func (d *DiscoverTestSuit) updateCommonNamespaces(t *testing.T, req []*api.Namespace) {
	resp := d.namespaceSvr.UpdateNamespaces(d.defaultCtx, req)
	if !respSuccess(resp) {
		t.Fatalf("error: %s", resp.GetInfo().GetValue())
	}
}

// 测试新建命名空间
func TestCreateNamespace(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	t.Run("正常创建命名空间", func(t *testing.T) {
		_, resp := discoverSuit.createCommonNamespace(t, 100)
		defer discoverSuit.cleanNamespace(resp.GetName().GetValue())
		t.Logf("pass")
	})

	t.Run("新建命名空间，删除，再创建一个同样的，可以成功", func(t *testing.T) {
		req, resp := discoverSuit.createCommonNamespace(t, 10)
		defer discoverSuit.cleanNamespace(req.GetName().GetValue())

		// remove
		discoverSuit.removeCommonNamespaces(t, []*api.Namespace{resp})
		apiResp := discoverSuit.namespaceSvr.CreateNamespace(discoverSuit.defaultCtx, req)
		if !respSuccess(apiResp) {
			t.Fatalf("error: %s", apiResp.GetInfo().GetValue())
		}

		t.Logf("pass")
	})

	t.Run("新建命名空间和服务，删除命名空间和服务，再创建命名空间", func(t *testing.T) {
		_, namespaceResp := discoverSuit.createCommonNamespace(t, 10)
		defer discoverSuit.cleanNamespace(namespaceResp.GetName().GetValue())

		_, serviceResp := discoverSuit.createCommonService(t, 100)
		defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		discoverSuit.removeCommonServices(t, []*api.Service{serviceResp})
		discoverSuit.removeCommonNamespaces(t, []*api.Namespace{namespaceResp})

		_, namespaceResp = discoverSuit.createCommonNamespace(t, 10)
		defer discoverSuit.cleanNamespace(namespaceResp.GetName().GetValue())
	})
}

// 删除命名空间
func TestRemoveNamespace(t *testing.T) {
	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	t.Run("可以删除命名空间", func(t *testing.T) {
		_, resp := discoverSuit.createCommonNamespace(t, 99)
		defer discoverSuit.cleanNamespace(resp.GetName().GetValue())

		discoverSuit.removeCommonNamespaces(t, []*api.Namespace{resp})
		out := discoverSuit.namespaceSvr.GetNamespaces(discoverSuit.defaultCtx, map[string][]string{"name": {resp.GetName().GetValue()}})
		if !respSuccess(out) {
			t.Fatalf("error: %s", out.GetInfo().GetValue())
		}
		if len(out.GetNamespaces()) != 0 {
			t.Fatalf("error: %d", len(out.GetNamespaces()))
		}
	})

	t.Run("批量删除命名空间", func(t *testing.T) {
		var reqs []*api.Namespace
		for i := 0; i < 20; i++ {
			_, resp := discoverSuit.createCommonNamespace(t, i)
			defer discoverSuit.cleanNamespace(resp.GetName().GetValue())
			reqs = append(reqs, resp)
		}

		time.Sleep(discoverSuit.updateCacheInterval)
		discoverSuit.removeCommonNamespaces(t, reqs)
		t.Logf("pass")
	})

	t.Run("新建命名空间和服务，直接删除名空间，因为有服务，删除会失败", func(t *testing.T) {
		_, namespaceResp := discoverSuit.createCommonNamespace(t, 100)
		defer discoverSuit.cleanNamespace(namespaceResp.GetName().GetValue())

		serviceReq := &api.Service{
			Name:      utils.NewStringValue("abc"),
			Namespace: namespaceResp.GetName(),
			Owners:    utils.NewStringValue("123"),
		}
		if resp := discoverSuit.server.CreateServices(discoverSuit.defaultCtx, []*api.Service{serviceReq}); !respSuccess(resp) {
			t.Fatalf("errror: %s", resp.GetInfo().GetValue())
		}
		defer discoverSuit.cleanServiceName(serviceReq.GetName().GetValue(), serviceReq.GetNamespace().GetValue())

		resp := discoverSuit.namespaceSvr.DeleteNamespace(discoverSuit.defaultCtx, namespaceResp)
		if resp.GetCode().GetValue() != api.NamespaceExistedServices {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Logf("%s", resp.GetInfo().GetValue())
	})
}

// 更新命名空间
func TestUpdateNamespace(t *testing.T) {
	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	t.Run("正常更新命名空间", func(t *testing.T) {
		req, resp := discoverSuit.createCommonNamespace(t, 200)
		defer discoverSuit.cleanNamespace(resp.GetName().GetValue())

		time.Sleep(discoverSuit.updateCacheInterval)

		req.Token = resp.Token
		req.Comment = utils.NewStringValue("new-comment")

		discoverSuit.updateCommonNamespaces(t, []*api.Namespace{req})
		t.Logf("pass")
	})
}

// 获取命名空间列表
func TestGetNamespaces(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	t.Run("正常获取命名空间，可以正常获取", func(t *testing.T) {
		total := 50
		for i := 0; i < total; i++ {
			req, _ := discoverSuit.createCommonNamespace(t, i+200)
			defer discoverSuit.cleanNamespace(req.GetName().GetValue())
		}

		resp := discoverSuit.namespaceSvr.GetNamespaces(discoverSuit.defaultCtx, map[string][]string{})
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if resp.GetSize().GetValue() == uint32(total) {
			t.Fatalf("error: %d", resp.GetSize().GetValue())
		}
	})

	t.Run("分页参数可以正常过滤", func(t *testing.T) {
		total := 20
		for i := 0; i < total; i++ {
			req, _ := discoverSuit.createCommonNamespace(t, i+200)
			defer discoverSuit.cleanNamespace(req.GetName().GetValue())
		}

		query := map[string][]string{
			"offset": {"10"},
			"limit":  {"10"},
		}
		resp := discoverSuit.namespaceSvr.GetNamespaces(discoverSuit.defaultCtx, query)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if resp.GetSize().GetValue() != 10 {
			t.Fatalf("error: %d", resp.GetSize().GetValue())
		}
	})
}

// 测试命名空间的token
func TestNamespaceToken(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	t.Run("可以正常获取到namespaceToken", func(t *testing.T) {
		_, namespaceResp := discoverSuit.createCommonNamespace(t, 1)
		defer discoverSuit.cleanNamespace(namespaceResp.GetName().GetValue())

		resp := discoverSuit.namespaceSvr.GetNamespaceToken(discoverSuit.defaultCtx, namespaceResp)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if resp.GetNamespace().GetToken().GetValue() != namespaceResp.GetToken().GetValue() {
			t.Fatalf("error")
		}
	})
	t.Run("可以正常更新namespace的token", func(t *testing.T) {
		_, namespaceResp := discoverSuit.createCommonNamespace(t, 2)
		defer discoverSuit.cleanNamespace(namespaceResp.GetName().GetValue())

		resp := discoverSuit.namespaceSvr.UpdateNamespaceToken(discoverSuit.defaultCtx, namespaceResp)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if resp.GetNamespace().GetToken().GetValue() == namespaceResp.GetToken().GetValue() {
			t.Fatalf("error")
		}
		t.Logf("%s %s", resp.GetNamespace().GetToken().GetValue(),
			namespaceResp.GetToken().GetValue())
	})
}
