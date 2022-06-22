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

package discover

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/utils"
)

/**
 * @brief 创建平台
 */
func createCommonPlatform(t *testing.T, id int) (*api.Platform, *api.Platform) {
	req := &api.Platform{
		Id:         utils.NewStringValue(fmt.Sprintf("id-%d", id)),
		Name:       utils.NewStringValue("name"),
		Domain:     utils.NewStringValue(fmt.Sprintf("domain-%d", id)),
		Qps:        utils.NewUInt32Value(1),
		Owner:      utils.NewStringValue(fmt.Sprintf("owner-%d", id)),
		Department: utils.NewStringValue(fmt.Sprintf("department-%d", id)),
		Comment:    utils.NewStringValue(fmt.Sprintf("comment-%d", id)),
	}
	cleanPlatform(req.GetId().GetValue())

	resp := server.CreatePlatform(defaultCtx, req)
	if !respSuccess(resp) {
		t.Fatalf("error: %s", resp.GetInfo().GetValue())
	}
	if resp.GetPlatform().GetToken().GetValue() == "" {
		t.Fatalf("error: %+v", resp)
	}
	if _, err := comparePlatform(req, resp.GetPlatform()); err != nil {
		t.Fatalf("error: %s", err.Error())
	}

	return req, resp.GetPlatform()
}

/**
 * @brief 删除平台
 */
func removeCommonPlatform(t *testing.T, req *api.Platform) {
	resp := server.DeletePlatform(defaultCtx, req)
	if !respSuccess(resp) {
		t.Fatalf("error: %s", resp.GetInfo().GetValue())
	}
}

/**
 * @brief 修改平台
 */
func updateCommonPlatform(t *testing.T, req *api.Platform) {
	resp := server.UpdatePlatform(defaultCtx, req)
	if !respSuccess(resp) {
		t.Fatalf("error: %s", resp.GetInfo().GetValue())
	}
}

/**
 * @brief 修改平台内容
 */
func updatePlatformContent(req *api.Platform) {
	req.Name = utils.NewStringValue("update-name")
	req.Domain = utils.NewStringValue("update-domain")
	req.Qps = utils.NewUInt32Value(req.GetQps().GetValue() + 1)
	req.Owner = utils.NewStringValue("update-owner")
	req.Department = utils.NewStringValue("update-department")
	req.Comment = utils.NewStringValue("update-comment")
}

/**
 * @brief 从数据库彻底删除平台
 */
func cleanPlatform(id string) {
	if id == "" {
		panic("id is empty")
	}

	log.Infof("clean platform: %s", id)
	str := `delete from platform where id = ?`
	if _, err := db.Exec(str, id); err != nil {
		panic(err)
	}
}

func comparePlatform(correctItem *api.Platform, item *api.Platform) (bool, error) {
	switch {
	case correctItem.GetId().GetValue() != item.GetId().GetValue():
		return false, errors.New("error id")
	case correctItem.GetName().GetValue() != item.GetName().GetValue():
		return false, errors.New("error name")
	case correctItem.GetDomain().GetValue() != item.GetDomain().GetValue():
		return false, errors.New("error domain")
	case correctItem.GetQps().GetValue() != item.GetQps().GetValue():
		return false, errors.New("error qps")
	case correctItem.GetOwner().GetValue() != item.GetOwner().GetValue():
		return false, errors.New("error owner")
	case correctItem.GetDepartment().GetValue() != item.GetDepartment().GetValue():
		return false, errors.New("error department")
	case correctItem.GetComment().GetValue() != item.GetComment().GetValue():
		return false, errors.New("error comment")
	}
	return true, nil
}

/**
 * @brief 测试新建平台
 */
func TestCreatePlatform(t *testing.T) {
	t.Run("正常创建平台，返回成功", func(t *testing.T) {
		_, resp := createCommonPlatform(t, 1)
		defer cleanPlatform(resp.GetId().GetValue())
		t.Log("pass")
	})

	t.Run("创建平台，删除后，再创建同名平台，返回成功", func(t *testing.T) {
		req, resp := createCommonPlatform(t, 1)
		defer cleanPlatform(resp.GetId().GetValue())

		// 删除平台
		removeCommonPlatform(t, resp)
		apiResp := server.CreatePlatform(defaultCtx, req)
		if !respSuccess(apiResp) {
			t.Fatalf("error: %s", apiResp.GetInfo().GetValue())
		} else {
			t.Log("pass")
		}
	})

	t.Run("重复创建平台，返回失败", func(t *testing.T) {
		req, _ := createCommonPlatform(t, 1)
		defer cleanPlatform(req.GetId().GetValue())

		resp := server.CreatePlatform(defaultCtx, req)
		if !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("创建平台时没有传递id，返回失败", func(t *testing.T) {
		req := &api.Platform{}

		resp := server.CreatePlatform(defaultCtx, req)
		if !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("创建平台时没有传递负责人, 返回失败", func(t *testing.T) {
		req := &api.Platform{
			Id: utils.NewStringValue(fmt.Sprintf("id-%d", 1)),
		}

		resp := server.CreatePlatform(defaultCtx, req)
		if !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("并发创建平台，返回成功", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 1; i <= 500; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				_, resp := createCommonPlatform(t, index)
				defer cleanPlatform(resp.GetId().GetValue())
			}(i)
		}
		wg.Wait()
		t.Log("pass")
	})
}

/**
 * @brief 测试删除平台
 */
func TestDeletePlatform(t *testing.T) {
	getPlatforms := func(t *testing.T, id string, expectNum uint32) {
		filter := map[string]string{
			"id": id,
		}
		resp := server.GetPlatforms(context.Background(), filter)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if resp.GetAmount().GetValue() != expectNum {
			t.Fatalf("error, actual num is %d, expect num is %d", resp.GetAmount().GetValue(), expectNum)
		}
	}

	t.Run("删除存在的平台，返回成功", func(t *testing.T) {
		_, resp := createCommonPlatform(t, 1)
		defer cleanPlatform(resp.GetId().GetValue())
		removeCommonPlatform(t, resp)
		getPlatforms(t, resp.GetId().GetValue(), 0)
		t.Log("pass")
	})

	t.Run("删除不存在的平台，返回成功", func(t *testing.T) {
		_, resp := createCommonPlatform(t, 1)
		defer cleanPlatform(resp.GetId().GetValue())

		removeCommonPlatform(t, resp)

		apiResp := server.DeletePlatform(defaultCtx, resp)
		if respSuccess(apiResp) {
			t.Log("pass")
		} else {
			t.Fatalf("error: %s", apiResp.GetInfo().GetValue())
		}

		getPlatforms(t, resp.GetId().GetValue(), 0)
	})

	t.Run("使用系统token删除，返回成功", func(t *testing.T) {
		req, _ := createCommonPlatform(t, 1)
		defer cleanPlatform(req.GetId().GetValue())

		ctx := context.WithValue(defaultCtx, utils.StringContext("polaris-token"), "polaris@12345678")

		apiReq := &api.Platform{
			Id: req.GetId(),
		}

		resp := server.DeletePlatform(ctx, apiReq)
		if respSuccess(resp) {
			t.Log("pass")
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}

		getPlatforms(t, req.GetId().GetValue(), 0)
	})

	t.Run("删除平台时没有传递平台id，返回失败", func(t *testing.T) {
		req := &api.Platform{}

		resp := server.DeletePlatform(defaultCtx, req)
		if !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error")
		}
	})

	t.Run("删除平台时没有传递平台Token，返回失败", func(t *testing.T) {
		req, _ := createCommonPlatform(t, 1)
		defer cleanPlatform(req.GetId().GetValue())

		apiReq := &api.Platform{
			Id: req.GetId(),
		}

		resp := server.DeletePlatform(defaultCtx, apiReq)
		if !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("并发删除平台，返回成功", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 1; i <= 500; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				_, resp := createCommonPlatform(t, index)
				defer cleanPlatform(resp.GetId().GetValue())
				removeCommonPlatform(t, resp)
			}(i)
		}
		wg.Wait()
		t.Log("pass")
	})
}

/**
 * @brief 测试更新平台
 */
func TestUpdatePlatform(t *testing.T) {
	t.Run("更新平台，返回成功", func(t *testing.T) {
		req, _ := createCommonPlatform(t, 1)
		defer cleanPlatform(req.GetId().GetValue())
		updatePlatformContent(req)
		updateCommonPlatform(t, req)
		filter := map[string]string{
			"id": req.GetId().GetValue(),
		}
		resp := server.GetPlatforms(context.Background(), filter)
		if !respSuccess(resp) {
			t.Fatal("error")
		}
		comparePlatform(req, resp.GetPlatforms()[0])
	})

	t.Run("更新不存在的平台，返回失败", func(t *testing.T) {
		req, _ := createCommonPlatform(t, 1)
		cleanPlatform(req.GetId().GetValue())
		resp := server.UpdatePlatform(defaultCtx, req)
		if !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("更新平台时没有传递token，返回错误", func(t *testing.T) {
		req, _ := createCommonPlatform(t, 1)
		defer cleanPlatform(req.GetId().GetValue())
		req.Token = utils.NewStringValue("")

		resp := server.UpdatePlatform(defaultCtx, req)
		if !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("并发更新平台，返回成功", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 1; i <= 500; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				req, resp := createCommonPlatform(t, index)
				defer cleanPlatform(resp.GetId().GetValue())
				updatePlatformContent(req)
				updateCommonPlatform(t, req)
			}(i)
		}
		wg.Wait()
		t.Log("pass")
	})
}

/**
 * @brief 测试查询平台
 */
func TestGetPlatform(t *testing.T) {
	platformNum := 10
	platformName := "name"
	for i := 1; i <= platformNum; i++ {
		req, _ := createCommonPlatform(t, i)
		defer cleanPlatform(req.GetId().GetValue())
	}

	t.Run("查询平台，过滤条件为id", func(t *testing.T) {
		filter := map[string]string{
			"id": fmt.Sprintf("id-%d", platformNum),
		}
		resp := server.GetPlatforms(context.Background(), filter)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if resp.GetSize().GetValue() != 1 {
			t.Fatalf("expect num is 1, actual num is %d", resp.GetSize().GetValue())
		}
		t.Log("pass")
	})

	t.Run("查询平台，过滤条件为name", func(t *testing.T) {
		filter := map[string]string{
			"name": platformName,
		}
		resp := server.GetPlatforms(context.Background(), filter)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if resp.GetSize().GetValue() != uint32(platformNum) {
			t.Fatalf("expect num is %d, actual num is %d", platformNum, resp.GetSize().GetValue())
		}
		t.Log("pass")
	})

	t.Run("查询平台，过滤条件为name和owner", func(t *testing.T) {
		filter := map[string]string{
			"name":  platformName,
			"owner": fmt.Sprintf("owner-%d", platformNum),
		}
		resp := server.GetPlatforms(context.Background(), filter)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if resp.GetSize().GetValue() != 1 {
			t.Fatalf("expect num is 1, actual num is %d", resp.GetSize().GetValue())
		}
		t.Log("pass")
	})

	t.Run("查询平台，过滤条件为不存在的name", func(t *testing.T) {
		filter := map[string]string{
			"name": "not exist",
		}
		resp := server.GetPlatforms(context.Background(), filter)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if resp.GetSize().GetValue() != 0 {
			t.Fatalf("expect num is 0, actual num is %d", resp.GetSize().GetValue())
		}
		t.Log("pass")
	})

	t.Run("查询平台，过滤条件为domain，返回失败", func(t *testing.T) {
		filter := map[string]string{
			"domain": "test",
		}
		resp := server.GetPlatforms(context.Background(), filter)
		if !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("查询平台，offset为负数，返回失败", func(t *testing.T) {
		filter := map[string]string{
			"offset": "-3",
		}
		resp := server.GetPlatforms(context.Background(), filter)
		if !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error")
		}
	})
}
