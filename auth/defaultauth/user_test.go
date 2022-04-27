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

package defaultauth

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/polarismesh/polaris-server/cache"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/plugin"
	storemock "github.com/polarismesh/polaris-server/store/mock"
	"github.com/stretchr/testify/assert"
)

func Test_server_CreateUsers(t *testing.T) {
	reset(false)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	users := createMockUser(10)
	groups := createMockUserGroup(users)

	storage := storemock.NewMockStore(ctrl)

	storage.EXPECT().AddUser(gomock.Any()).AnyTimes().Return(nil)
	storage.EXPECT().GetUserByName(gomock.Eq("create-user-1"), gomock.Any()).AnyTimes().Return(nil, nil)
	storage.EXPECT().GetUserByName(gomock.Eq("create-user-2"), gomock.Any()).AnyTimes().Return(&model.User{
		Name: "create-user-2",
	}, nil)
	storage.EXPECT().GetUsersForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(users, nil)
	storage.EXPECT().GetGroupsForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(groups, nil)

	cfg := &cache.Config{
		Open: true,
		Resources: []cache.ConfigEntry{
			{
				Name: "users",
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	if err := cache.TestCacheInitialize(ctx, cfg, storage); err != nil {
		t.Fatal(err)
	}

	cacheMgn, err := cache.GetCacheManager()
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		cancel()
		cacheMgn.Clear()
		time.Sleep(2 * time.Second)
	}()
	time.Sleep(time.Second)

	checker := &defaultAuthChecker{}
	checker.cacheMgn = cacheMgn
	checker.authPlugin = plugin.GetAuth()

	svr := &serverAuthAbility{
		authMgn: checker,
		target: &server{
			storage:  storage,
			cacheMgn: cacheMgn,
			authMgn:  checker,
		},
	}

	t.Run("主账户创建账户-成功", func(t *testing.T) {
		createUsersReq := []*api.User{
			{
				Id:       &wrappers.StringValue{Value: utils.NewUUID()},
				Name:     &wrappers.StringValue{Value: "create-user-1"},
				Password: &wrappers.StringValue{Value: "create-user-1"},
			},
		}

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[0].Token)
		resp := svr.CreateUsers(reqCtx, createUsersReq)

		t.Logf("CreateUsers resp : %+v", resp)
		assert.Equal(t, api.ExecuteSuccess, resp.Code.GetValue(), "create users must success")
	})

	t.Run("主账户创建账户-无用户名-失败", func(t *testing.T) {
		createUsersReq := []*api.User{
			{
				Id: &wrappers.StringValue{Value: utils.NewUUID()},
			},
		}

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[0].Token)
		resp := svr.CreateUsers(reqCtx, createUsersReq)

		t.Logf("CreateUsers resp : %+v", resp)
		assert.Equal(t, api.ExecuteException, resp.Code.GetValue(), "create users must fail")
		assert.Equal(t, api.InvalidUserName, resp.Responses[0].Code.GetValue(), "create users must fail")
	})

	t.Run("主账户创建账户-密码错误-失败", func(t *testing.T) {
		createUsersReq := []*api.User{
			{
				Id:       &wrappers.StringValue{Value: utils.NewUUID()},
				Name:     &wrappers.StringValue{Value: "create-user-1"},
				Password: &wrappers.StringValue{Value: ""},
			},
		}

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[0].Token)
		resp := svr.CreateUsers(reqCtx, createUsersReq)

		t.Logf("CreateUsers resp : %+v", resp)
		assert.Equal(t, api.ExecuteException, resp.Code.GetValue(), "create users must fail")
		assert.Equal(t, api.InvalidUserPassword, resp.Responses[0].Code.GetValue(), "create users must fail")
	})

	t.Run("主账户创建账户-同名用户-失败", func(t *testing.T) {
		createUsersReq := []*api.User{
			{
				Id:       &wrappers.StringValue{Value: utils.NewUUID()},
				Name:     &wrappers.StringValue{Value: "create-user-2"},
				Password: &wrappers.StringValue{Value: "create-user-2"},
			},
		}

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[0].Token)
		resp := svr.CreateUsers(reqCtx, createUsersReq)

		t.Logf("CreateUsers resp : %+v", resp)
		assert.Equal(t, api.ExecuteException, resp.Code.GetValue(), "create users must fail")
		assert.Equal(t, api.UserExisted, resp.Responses[0].Code.GetValue(), "create users must fail")
	})

	t.Run("主账户创建账户-token为空-失败", func(t *testing.T) {
		createUsersReq := []*api.User{
			{
				Id:       &wrappers.StringValue{Value: utils.NewUUID()},
				Name:     &wrappers.StringValue{Value: "create-user-2"},
				Password: &wrappers.StringValue{Value: "create-user-2"},
			},
		}

		resp := svr.CreateUsers(context.Background(), createUsersReq)
		t.Logf("CreateUsers resp : %+v", resp)
		assert.Equal(t, api.ExecuteException, resp.Code.GetValue(), "create users must fail")
		assert.Equal(t, api.EmptyAutToken, resp.Responses[0].Code.GetValue(), "create users must fail")
	})

	t.Run("主账户创建账户-token非法-失败", func(t *testing.T) {
		createUsersReq := []*api.User{
			{
				Id:       &wrappers.StringValue{Value: utils.NewUUID()},
				Name:     &wrappers.StringValue{Value: "create-user-2"},
				Password: &wrappers.StringValue{Value: "create-user-2"},
			},
		}

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "utils.ContextAuthTokenKey")
		resp := svr.CreateUsers(reqCtx, createUsersReq)
		t.Logf("CreateUsers resp : %+v", resp)
		assert.Equal(t, api.ExecuteException, resp.Code.GetValue(), "create users must fail")
		assert.Equal(t, api.AuthTokenVerifyException, resp.Responses[0].Code.GetValue(), "create users must fail")
	})

	t.Run("主账户创建账户-token被禁用-失败", func(t *testing.T) {
		users[0].TokenEnable = false
		// 让 cache 可以刷新到
		time.Sleep(time.Second)

		createUsersReq := []*api.User{
			{
				Id:       &wrappers.StringValue{Value: utils.NewUUID()},
				Name:     &wrappers.StringValue{Value: "create-user-2"},
				Password: &wrappers.StringValue{Value: "create-user-2"},
			},
		}

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[0].Token)
		resp := svr.CreateUsers(reqCtx, createUsersReq)

		t.Logf("CreateUsers resp : %+v", resp)
		assert.Equal(t, api.ExecuteException, resp.Code.GetValue(), "create users must fail")
		assert.Equal(t, api.TokenDisabled, resp.Responses[0].Code.GetValue(), "create users must fail")

		users[0].TokenEnable = true
		time.Sleep(time.Second)
	})

	t.Run("子主账户创建账户-失败", func(t *testing.T) {
		createUsersReq := []*api.User{
			{
				Id:       &wrappers.StringValue{Value: utils.NewUUID()},
				Name:     &wrappers.StringValue{Value: "create-user-1"},
				Password: &wrappers.StringValue{Value: "create-user-1"},
			},
		}

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)
		resp := svr.CreateUsers(reqCtx, createUsersReq)

		t.Logf("CreateUsers resp : %+v", resp)
		assert.Equal(t, api.ExecuteException, resp.Code.GetValue(), "create users must fail")
		assert.Equal(t, api.OperationRoleException, resp.Responses[0].Code.GetValue(), "create users must fail")
	})

	t.Run("用户组token创建账户-失败", func(t *testing.T) {
		createUsersReq := []*api.User{
			{
				Id:       &wrappers.StringValue{Value: utils.NewUUID()},
				Name:     &wrappers.StringValue{Value: "create-user-1"},
				Password: &wrappers.StringValue{Value: "create-user-1"},
			},
		}

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groups[1].Token)
		resp := svr.CreateUsers(reqCtx, createUsersReq)

		t.Logf("CreateUsers resp : %+v", resp)
		assert.Equal(t, api.ExecuteException, resp.Code.GetValue(), "create users must fail")
		assert.Equal(t, api.OperationRoleException, resp.Responses[0].Code.GetValue(), "create users must fail")
	})
}

func Test_server_UpdateUser(t *testing.T) {
	reset(false)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	users := createMockUser(10)
	groups := createMockUserGroup(users)

	storage := storemock.NewMockStore(ctrl)

	storage.EXPECT().UpdateUser(gomock.Any()).AnyTimes().Return(nil)
	storage.EXPECT().GetUsersForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(users, nil)
	storage.EXPECT().GetGroupsForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(groups, nil)

	cfg := &cache.Config{
		Open: true,
		Resources: []cache.ConfigEntry{
			{
				Name: "users",
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	if err := cache.TestCacheInitialize(ctx, cfg, storage); err != nil {
		t.Fatal(err)
	}

	cacheMgn, err := cache.GetCacheManager()
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		cancel()
		cacheMgn.Clear()
		time.Sleep(2 * time.Second)
	}()
	time.Sleep(time.Second)

	checker := &defaultAuthChecker{}
	checker.cacheMgn = cacheMgn
	checker.authPlugin = plugin.GetAuth()

	svr := &serverAuthAbility{
		authMgn: checker,
		target: &server{
			storage:  storage,
			cacheMgn: cacheMgn,
			authMgn:  checker,
		},
	}

	t.Run("主账户更新账户信息-正常更新自己的信息", func(t *testing.T) {
		req := &api.User{
			Id:      &wrappers.StringValue{Value: users[0].ID},
			Comment: &wrappers.StringValue{Value: "update owner account info"},
		}

		storage.EXPECT().GetUser(gomock.Eq(users[0].ID)).Return(users[0], nil)

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[0].Token)
		resp := svr.UpdateUser(reqCtx, req)

		t.Logf("UpdateUsers resp : %+v", resp)
		assert.Equal(t, api.ExecuteSuccess, resp.Code.GetValue(), "update user must success")
	})

	t.Run("主账户更新账户信息-更新不存在的子账户", func(t *testing.T) {
		uid := utils.NewUUID()
		req := &api.User{
			Id:      &wrappers.StringValue{Value: uid},
			Comment: &wrappers.StringValue{Value: "update owner account info"},
		}

		storage.EXPECT().GetUser(gomock.Any()).Return(nil, nil)

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[0].Token)
		resp := svr.UpdateUser(reqCtx, req)

		t.Logf("UpdateUsers resp : %+v", resp)
		assert.Equal(t, api.NotFoundUser, resp.Code.GetValue(), "update user must fail")
	})

	t.Run("主账户更新账户信息-更新不属于自己的子账户", func(t *testing.T) {
		uid := utils.NewUUID()
		req := &api.User{
			Id:      &wrappers.StringValue{Value: uid},
			Comment: &wrappers.StringValue{Value: "update owner account info"},
		}

		storage.EXPECT().GetUser(gomock.Any()).Return(&model.User{
			ID:    uid,
			Owner: utils.NewUUID(),
		}, nil)

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[0].Token)
		resp := svr.UpdateUser(reqCtx, req)

		t.Logf("UpdateUsers resp : %+v", resp)
		assert.Equal(t, api.NotAllowedAccess, resp.Code.GetValue(), "update user must fail")
	})

	t.Run("子账户更新账户信息-正常更新自己的信息", func(t *testing.T) {
		req := &api.User{
			Id:      &wrappers.StringValue{Value: users[1].ID},
			Comment: &wrappers.StringValue{Value: "update owner account info"},
		}

		storage.EXPECT().GetUser(gomock.Any()).Return(users[1], nil)

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)
		resp := svr.UpdateUser(reqCtx, req)

		t.Logf("UpdateUsers resp : %+v", resp)
		assert.Equal(t, api.ExecuteSuccess, resp.Code.GetValue(), "update user must fail")
	})

	t.Run("子账户更新账户信息-更新别的账户", func(t *testing.T) {
		req := &api.User{
			Id:      &wrappers.StringValue{Value: users[2].ID},
			Comment: &wrappers.StringValue{Value: "update owner account info"},
		}

		storage.EXPECT().GetUser(gomock.Any()).Return(users[2], nil)

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)
		resp := svr.UpdateUser(reqCtx, req)

		t.Logf("UpdateUsers resp : %+v", resp)
		assert.Equal(t, api.NotAllowedAccess, resp.Code.GetValue(), "update user must fail")
	})

	t.Run("用户组Token更新账户信息-更新别的账户", func(t *testing.T) {
		req := &api.User{
			Id:      &wrappers.StringValue{Value: users[2].ID},
			Comment: &wrappers.StringValue{Value: "update owner account info"},
		}

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groups[1].Token)
		resp := svr.UpdateUser(reqCtx, req)

		t.Logf("UpdateUsers resp : %+v", resp)
		assert.Equal(t, api.OperationRoleException, resp.Code.GetValue(), "update user must fail")
	})
}

func Test_server_UpdateUserPassword(t *testing.T) {
	reset(false)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	users := createMockUser(10)
	groups := createMockUserGroup(users)

	storage := storemock.NewMockStore(ctrl)

	storage.EXPECT().UpdateUser(gomock.Any()).AnyTimes().Return(nil)
	storage.EXPECT().GetUsersForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(users, nil)
	storage.EXPECT().GetGroupsForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(groups, nil)

	cfg := &cache.Config{
		Open: true,
		Resources: []cache.ConfigEntry{
			{
				Name: "users",
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	if err := cache.TestCacheInitialize(ctx, cfg, storage); err != nil {
		t.Fatal(err)
	}

	cacheMgn, err := cache.GetCacheManager()
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		cancel()
		cacheMgn.Clear()
		time.Sleep(2 * time.Second)
	}()
	time.Sleep(time.Second)

	checker := &defaultAuthChecker{}
	checker.cacheMgn = cacheMgn
	checker.authPlugin = plugin.GetAuth()

	svr := &serverAuthAbility{
		authMgn: checker,
		target: &server{
			storage:  storage,
			cacheMgn: cacheMgn,
			authMgn:  checker,
		},
	}

	t.Run("主账户正常更新自身账户密码", func(t *testing.T) {
		req := &api.ModifyUserPassword{
			Id:          &wrappers.StringValue{Value: users[0].ID},
			OldPassword: &wrappers.StringValue{Value: "polaris"},
			NewPassword: &wrappers.StringValue{Value: "polaris@2021"},
		}

		storage.EXPECT().GetUser(gomock.Any()).Return(users[0], nil)

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[0].Token)
		resp := svr.UpdateUserPassword(reqCtx, req)
		t.Logf("CreateUsers resp : %+v", resp)
		assert.Equal(t, api.ExecuteSuccess, resp.Code.GetValue(), "update user must success")
	})

	t.Run("主账户正常更新自身账户密码-新密码非法", func(t *testing.T) {
		req := &api.ModifyUserPassword{
			Id:          &wrappers.StringValue{Value: users[0].ID},
			OldPassword: &wrappers.StringValue{Value: "polaris"},
			NewPassword: &wrappers.StringValue{Value: "pola"},
		}

		storage.EXPECT().GetUser(gomock.Any()).Return(users[0], nil)

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[0].Token)
		resp := svr.UpdateUserPassword(reqCtx, req)
		t.Logf("CreateUsers resp : %+v", resp)
		assert.Equal(t, api.ExecuteException, resp.Code.GetValue(), "update user must fail")

		req = &api.ModifyUserPassword{
			Id:          &wrappers.StringValue{Value: users[0].ID},
			OldPassword: &wrappers.StringValue{Value: "polaris"},
			NewPassword: &wrappers.StringValue{Value: ""},
		}

		storage.EXPECT().GetUser(gomock.Any()).Return(users[0], nil)

		reqCtx = context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[0].Token)
		resp = svr.UpdateUserPassword(reqCtx, req)
		t.Logf("CreateUsers resp : %+v", resp)
		assert.Equal(t, api.ExecuteException, resp.Code.GetValue(), "update user must fail")

		req = &api.ModifyUserPassword{
			Id:          &wrappers.StringValue{Value: users[0].ID},
			OldPassword: &wrappers.StringValue{Value: "polaris"},
			NewPassword: &wrappers.StringValue{Value: "polarispolarispolarispolaris"},
		}

		storage.EXPECT().GetUser(gomock.Any()).Return(users[0], nil)

		reqCtx = context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[0].Token)
		resp = svr.UpdateUserPassword(reqCtx, req)
		t.Logf("CreateUsers resp : %+v", resp)
		assert.Equal(t, api.ExecuteException, resp.Code.GetValue(), "update user must fail")
	})

	t.Run("主账户正常更新子账户密码", func(t *testing.T) {
		req := &api.ModifyUserPassword{
			Id:          &wrappers.StringValue{Value: users[1].ID},
			NewPassword: &wrappers.StringValue{Value: "polaris@sub"},
		}

		storage.EXPECT().GetUser(gomock.Any()).Return(users[1], nil)

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[0].Token)
		resp := svr.UpdateUserPassword(reqCtx, req)
		t.Logf("CreateUsers resp : %+v", resp)
		assert.Equal(t, api.ExecuteSuccess, resp.Code.GetValue(), "update user must success")
	})

	t.Run("主账户正常更新子账户密码-子账户非自己", func(t *testing.T) {

		uid := utils.NewUUID()

		req := &api.ModifyUserPassword{
			Id:          &wrappers.StringValue{Value: uid},
			NewPassword: &wrappers.StringValue{Value: "polaris@subaccount"},
		}

		storage.EXPECT().GetUser(gomock.Any()).Return(&model.User{
			ID:    uid,
			Owner: utils.NewUUID(),
		}, nil)

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[0].Token)
		resp := svr.UpdateUserPassword(reqCtx, req)
		t.Logf("CreateUsers resp : %+v", resp)
		assert.Equal(t, api.NotAllowedAccess, resp.Code.GetValue(), "update user must fail")
	})

	t.Run("子账户更新账户密码-自身-携带正确原密码", func(t *testing.T) {
		req := &api.ModifyUserPassword{
			Id:          &wrappers.StringValue{Value: users[2].ID},
			OldPassword: &wrappers.StringValue{Value: "polaris"},
			NewPassword: &wrappers.StringValue{Value: "users[1].Password"},
		}

		storage.EXPECT().GetUser(gomock.Any()).Return(users[2], nil)

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[2].Token)
		resp := svr.UpdateUserPassword(reqCtx, req)
		t.Logf("CreateUsers resp : %+v", resp)
		assert.Equal(t, api.ExecuteSuccess, resp.Code.GetValue(), "update user must fail")
	})

	t.Run("子账户更新账户密码-自身-携带错误原密码", func(t *testing.T) {
		req := &api.ModifyUserPassword{
			Id:          &wrappers.StringValue{Value: users[1].ID},
			OldPassword: &wrappers.StringValue{Value: "users[1].Password"},
			NewPassword: &wrappers.StringValue{Value: "users[1].Password"},
		}

		storage.EXPECT().GetUser(gomock.Any()).Return(users[1], nil)

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)
		resp := svr.UpdateUserPassword(reqCtx, req)
		t.Logf("CreateUsers resp : %+v", resp)
		assert.Equal(t, api.ExecuteException, resp.Code.GetValue(), "update user must fail")
	})

	t.Run("子账户更新账户密码-自身-无携带原密码", func(t *testing.T) {
		req := &api.ModifyUserPassword{
			Id:          &wrappers.StringValue{Value: users[1].ID},
			NewPassword: &wrappers.StringValue{Value: "users[1].Password"},
		}

		storage.EXPECT().GetUser(gomock.Any()).Return(users[1], nil)

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)
		resp := svr.UpdateUserPassword(reqCtx, req)
		t.Logf("CreateUsers resp : %+v", resp)
		assert.Equal(t, api.ExecuteException, resp.Code.GetValue(), "update user must fail")
	})

	t.Run("子账户更新账户密码-不是自己", func(t *testing.T) {
		req := &api.ModifyUserPassword{
			Id:          &wrappers.StringValue{Value: users[2].ID},
			NewPassword: &wrappers.StringValue{Value: "users[2].Password"},
		}

		storage.EXPECT().GetUser(gomock.Any()).Return(users[2], nil)

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)
		resp := svr.UpdateUserPassword(reqCtx, req)
		t.Logf("CreateUsers resp : %+v", resp)
		assert.Equal(t, api.NotAllowedAccess, resp.Code.GetValue(), "update user must fail")
	})
}
