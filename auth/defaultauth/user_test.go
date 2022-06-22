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
	"github.com/polarismesh/polaris-server/auth"
	"github.com/polarismesh/polaris-server/cache"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/plugin"
	storemock "github.com/polarismesh/polaris-server/store/mock"
	"github.com/stretchr/testify/assert"
)

type UserTest struct {
	admin    *model.User
	ownerOne *model.User
	ownerTwo *model.User

	users     []*model.User
	newUsers  []*model.User
	groups    []*model.UserGroupDetail
	newGroups []*model.UserGroupDetail
	allGroups []*model.UserGroupDetail

	storage  *storemock.MockStore
	cacheMgn *cache.CacheManager
	checker  auth.AuthChecker

	svr *serverAuthAbility

	cancel context.CancelFunc
	ctrl   *gomock.Controller
}

func newUserTest(t *testing.T) *UserTest {
	reset(false)
	ctrl := gomock.NewController(t)

	log.AuthScope().SetOutputLevel(log.DebugLevel)
	log.CacheScope().SetOutputLevel(log.DebugLevel)

	users := createMockUser(10, "one")
	newUsers := createMockUser(10, "two")
	admin := createMockUser(1, "admin")[0]
	admin.Type = model.AdminUserRole
	admin.Owner = ""
	groups := createMockUserGroup(users)

	storage := storemock.NewMockStore(ctrl)
	storage.EXPECT().GetUnixSecond().AnyTimes().Return(time.Now().Unix(), nil)
	storage.EXPECT().AddUser(gomock.Any()).AnyTimes().Return(nil)
	storage.EXPECT().GetUserByName(gomock.Eq("create-user-1"), gomock.Any()).AnyTimes().Return(nil, nil)
	storage.EXPECT().GetUserByName(gomock.Eq("create-user-2"), gomock.Any()).AnyTimes().Return(&model.User{
		Name: "create-user-2",
	}, nil)

	allUsers := append(append(users, newUsers...), admin)

	storage.EXPECT().GetUsersForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(allUsers, nil)
	storage.EXPECT().GetGroupsForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(groups, nil)
	storage.EXPECT().UpdateUser(gomock.Any()).AnyTimes().Return(nil)
	storage.EXPECT().DeleteUser(gomock.Any()).AnyTimes().Return(nil)

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

	time.Sleep(5 * time.Second)

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

	return &UserTest{
		admin:    admin,
		ownerOne: users[0],
		ownerTwo: newUsers[0],

		users:    users,
		newUsers: newUsers,
		groups:   groups,

		storage:  storage,
		cacheMgn: cacheMgn,
		checker:  checker,
		svr:      svr,

		cancel: cancel,
		ctrl:   ctrl,
	}
}

func (g *UserTest) Clean() {
	g.ctrl.Finish()
	g.cancel()
	_ = g.cacheMgn.Clear()
	time.Sleep(2 * time.Second)
}

func Test_server_CreateUsers(t *testing.T) {

	userTest := newUserTest(t)

	defer userTest.Clean()

	t.Run("主账户创建账户-成功", func(t *testing.T) {
		createUsersReq := []*api.User{
			{
				Id:       &wrappers.StringValue{Value: utils.NewUUID()},
				Name:     &wrappers.StringValue{Value: "create-user-1"},
				Password: &wrappers.StringValue{Value: "create-user-1"},
			},
		}

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.ownerOne.Token)
		resp := userTest.svr.CreateUsers(reqCtx, createUsersReq)

		t.Logf("CreateUsers resp : %+v", resp)
		assert.Equal(t, api.ExecuteSuccess, resp.Code.GetValue(), "create users must success")
	})

	t.Run("主账户创建账户-无用户名-失败", func(t *testing.T) {
		createUsersReq := []*api.User{
			{
				Id: &wrappers.StringValue{Value: utils.NewUUID()},
			},
		}

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.users[0].Token)
		resp := userTest.svr.CreateUsers(reqCtx, createUsersReq)

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

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.users[0].Token)
		resp := userTest.svr.CreateUsers(reqCtx, createUsersReq)

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

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.users[0].Token)
		resp := userTest.svr.CreateUsers(reqCtx, createUsersReq)

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

		resp := userTest.svr.CreateUsers(context.Background(), createUsersReq)
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
		resp := userTest.svr.CreateUsers(reqCtx, createUsersReq)
		t.Logf("CreateUsers resp : %+v", resp)
		assert.Equal(t, api.ExecuteException, resp.Code.GetValue(), "create users must fail")
		assert.Equal(t, api.AuthTokenVerifyException, resp.Responses[0].Code.GetValue(), "create users must fail")
	})

	t.Run("主账户创建账户-token被禁用-失败", func(t *testing.T) {
		userTest.users[0].TokenEnable = false
		// 让 cache 可以刷新到
		time.Sleep(time.Second)

		createUsersReq := []*api.User{
			{
				Id:       &wrappers.StringValue{Value: utils.NewUUID()},
				Name:     &wrappers.StringValue{Value: "create-user-2"},
				Password: &wrappers.StringValue{Value: "create-user-2"},
			},
		}

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.users[0].Token)
		resp := userTest.svr.CreateUsers(reqCtx, createUsersReq)

		t.Logf("CreateUsers resp : %+v", resp)
		assert.Equal(t, api.ExecuteException, resp.Code.GetValue(), "create users must fail")
		assert.Equal(t, api.TokenDisabled, resp.Responses[0].Code.GetValue(), "create users must fail")

		userTest.users[0].TokenEnable = true
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

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.users[1].Token)
		resp := userTest.svr.CreateUsers(reqCtx, createUsersReq)

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

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.groups[1].Token)
		resp := userTest.svr.CreateUsers(reqCtx, createUsersReq)

		t.Logf("CreateUsers resp : %+v", resp)
		assert.Equal(t, api.ExecuteException, resp.Code.GetValue(), "create users must fail")
		assert.Equal(t, api.OperationRoleException, resp.Responses[0].Code.GetValue(), "create users must fail")
	})
}

func Test_server_UpdateUser(t *testing.T) {

	userTest := newUserTest(t)
	defer userTest.Clean()

	t.Run("主账户更新账户信息-正常更新自己的信息", func(t *testing.T) {
		req := &api.User{
			Id:      &wrappers.StringValue{Value: userTest.users[0].ID},
			Comment: &wrappers.StringValue{Value: "update owner account info"},
		}

		userTest.storage.EXPECT().GetUser(gomock.Any()).Return(userTest.users[0], nil)

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.users[0].Token)
		resp := userTest.svr.UpdateUser(reqCtx, req)

		t.Logf("UpdateUsers resp : %+v", resp)
		assert.Equal(t, api.ExecuteSuccess, resp.Code.GetValue(), "update user must success")
	})

	t.Run("主账户更新账户信息-更新不存在的子账户", func(t *testing.T) {
		uid := utils.NewUUID()
		req := &api.User{
			Id:      &wrappers.StringValue{Value: uid},
			Comment: &wrappers.StringValue{Value: "update owner account info"},
		}

		userTest.storage.EXPECT().GetUser(gomock.Any()).Return(nil, nil)

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.users[0].Token)
		resp := userTest.svr.UpdateUser(reqCtx, req)

		t.Logf("UpdateUsers resp : %+v", resp)
		assert.Equal(t, api.NotFoundUser, resp.Code.GetValue(), "update user must fail")
	})

	t.Run("主账户更新账户信息-更新不属于自己的子账户", func(t *testing.T) {
		uid := utils.NewUUID()
		req := &api.User{
			Id:      &wrappers.StringValue{Value: uid},
			Comment: &wrappers.StringValue{Value: "update owner account info"},
		}

		userTest.storage.EXPECT().GetUser(gomock.Any()).Return(&model.User{
			ID:    uid,
			Owner: utils.NewUUID(),
		}, nil)

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.users[0].Token)
		resp := userTest.svr.UpdateUser(reqCtx, req)

		t.Logf("UpdateUsers resp : %+v", resp)
		assert.Equal(t, api.NotAllowedAccess, resp.Code.GetValue(), "update user must fail")
	})

	t.Run("子账户更新账户信息-正常更新自己的信息", func(t *testing.T) {
		req := &api.User{
			Id:      &wrappers.StringValue{Value: userTest.users[1].ID},
			Comment: &wrappers.StringValue{Value: "update owner account info"},
		}

		userTest.storage.EXPECT().GetUser(gomock.Any()).Return(userTest.users[1], nil)

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.users[1].Token)
		resp := userTest.svr.UpdateUser(reqCtx, req)

		t.Logf("UpdateUsers resp : %+v", resp)
		assert.Equal(t, api.ExecuteSuccess, resp.Code.GetValue(), "update user must fail")
	})

	t.Run("子账户更新账户信息-更新别的账户", func(t *testing.T) {
		req := &api.User{
			Id:      &wrappers.StringValue{Value: userTest.users[2].ID},
			Comment: &wrappers.StringValue{Value: "update owner account info"},
		}

		userTest.storage.EXPECT().GetUser(gomock.Any()).Return(userTest.users[2], nil)

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.users[1].Token)
		resp := userTest.svr.UpdateUser(reqCtx, req)

		t.Logf("UpdateUsers resp : %+v", resp)
		assert.Equal(t, api.NotAllowedAccess, resp.Code.GetValue(), "update user must fail")
	})

	t.Run("用户组Token更新账户信息-更新别的账户", func(t *testing.T) {
		req := &api.User{
			Id:      &wrappers.StringValue{Value: userTest.users[2].ID},
			Comment: &wrappers.StringValue{Value: "update owner account info"},
		}

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.groups[1].Token)
		resp := userTest.svr.UpdateUser(reqCtx, req)

		t.Logf("UpdateUsers resp : %+v", resp)
		assert.Equal(t, api.OperationRoleException, resp.Code.GetValue(), "update user must fail")
	})
}

func Test_server_UpdateUserPassword(t *testing.T) {

	userTest := newUserTest(t)
	defer userTest.Clean()

	t.Run("主账户正常更新自身账户密码", func(t *testing.T) {
		req := &api.ModifyUserPassword{
			Id:          &wrappers.StringValue{Value: userTest.users[0].ID},
			OldPassword: &wrappers.StringValue{Value: "polaris"},
			NewPassword: &wrappers.StringValue{Value: "polaris@2021"},
		}

		userTest.storage.EXPECT().GetUser(gomock.Any()).Return(userTest.users[0], nil)

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.users[0].Token)
		resp := userTest.svr.UpdateUserPassword(reqCtx, req)
		t.Logf("CreateUsers resp : %+v", resp)
		assert.Equal(t, api.ExecuteSuccess, resp.Code.GetValue(), "update user must success")
	})

	t.Run("主账户正常更新自身账户密码-新密码非法", func(t *testing.T) {
		req := &api.ModifyUserPassword{
			Id:          &wrappers.StringValue{Value: userTest.users[0].ID},
			OldPassword: &wrappers.StringValue{Value: "polaris"},
			NewPassword: &wrappers.StringValue{Value: "pola"},
		}

		userTest.storage.EXPECT().GetUser(gomock.Any()).Return(userTest.users[0], nil)

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.users[0].Token)
		resp := userTest.svr.UpdateUserPassword(reqCtx, req)
		t.Logf("CreateUsers resp : %+v", resp)
		assert.Equal(t, api.ExecuteException, resp.Code.GetValue(), "update user must fail")

		req = &api.ModifyUserPassword{
			Id:          &wrappers.StringValue{Value: userTest.users[0].ID},
			OldPassword: &wrappers.StringValue{Value: "polaris"},
			NewPassword: &wrappers.StringValue{Value: ""},
		}

		userTest.storage.EXPECT().GetUser(gomock.Any()).Return(userTest.users[0], nil)

		reqCtx = context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.users[0].Token)
		resp = userTest.svr.UpdateUserPassword(reqCtx, req)
		t.Logf("CreateUsers resp : %+v", resp)
		assert.Equal(t, api.ExecuteException, resp.Code.GetValue(), "update user must fail")

		req = &api.ModifyUserPassword{
			Id:          &wrappers.StringValue{Value: userTest.users[0].ID},
			OldPassword: &wrappers.StringValue{Value: "polaris"},
			NewPassword: &wrappers.StringValue{Value: "polarispolarispolarispolaris"},
		}

		userTest.storage.EXPECT().GetUser(gomock.Any()).Return(userTest.users[0], nil)

		reqCtx = context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.users[0].Token)
		resp = userTest.svr.UpdateUserPassword(reqCtx, req)
		t.Logf("CreateUsers resp : %+v", resp)
		assert.Equal(t, api.ExecuteException, resp.Code.GetValue(), "update user must fail")
	})

	t.Run("主账户正常更新子账户密码", func(t *testing.T) {
		req := &api.ModifyUserPassword{
			Id:          &wrappers.StringValue{Value: userTest.users[1].ID},
			NewPassword: &wrappers.StringValue{Value: "polaris@sub"},
		}

		userTest.storage.EXPECT().GetUser(gomock.Any()).Return(userTest.users[1], nil)

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.users[0].Token)
		resp := userTest.svr.UpdateUserPassword(reqCtx, req)
		t.Logf("CreateUsers resp : %+v", resp)
		assert.Equal(t, api.ExecuteSuccess, resp.Code.GetValue(), "update user must success")
	})

	t.Run("主账户正常更新子账户密码-子账户非自己", func(t *testing.T) {

		uid := utils.NewUUID()

		req := &api.ModifyUserPassword{
			Id:          &wrappers.StringValue{Value: uid},
			NewPassword: &wrappers.StringValue{Value: "polaris@subaccount"},
		}

		userTest.storage.EXPECT().GetUser(gomock.Any()).Return(&model.User{
			ID:    uid,
			Owner: utils.NewUUID(),
		}, nil)

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.users[0].Token)
		resp := userTest.svr.UpdateUserPassword(reqCtx, req)
		t.Logf("CreateUsers resp : %+v", resp)
		assert.Equal(t, api.NotAllowedAccess, resp.Code.GetValue(), "update user must fail")
	})

	t.Run("子账户更新账户密码-自身-携带正确原密码", func(t *testing.T) {
		req := &api.ModifyUserPassword{
			Id:          &wrappers.StringValue{Value: userTest.users[2].ID},
			OldPassword: &wrappers.StringValue{Value: "polaris"},
			NewPassword: &wrappers.StringValue{Value: "users[1].Password"},
		}

		userTest.storage.EXPECT().GetUser(gomock.Any()).Return(userTest.users[2], nil)

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.users[2].Token)
		resp := userTest.svr.UpdateUserPassword(reqCtx, req)
		t.Logf("CreateUsers resp : %+v", resp)
		assert.Equal(t, api.ExecuteSuccess, resp.Code.GetValue(), "update user must fail")
	})

	t.Run("子账户更新账户密码-自身-携带错误原密码", func(t *testing.T) {
		req := &api.ModifyUserPassword{
			Id:          &wrappers.StringValue{Value: userTest.users[1].ID},
			OldPassword: &wrappers.StringValue{Value: "users[1].Password"},
			NewPassword: &wrappers.StringValue{Value: "users[1].Password"},
		}

		userTest.storage.EXPECT().GetUser(gomock.Any()).Return(userTest.users[1], nil)

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.users[1].Token)
		resp := userTest.svr.UpdateUserPassword(reqCtx, req)
		t.Logf("CreateUsers resp : %+v", resp)
		assert.Equal(t, api.ExecuteException, resp.Code.GetValue(), "update user must fail")
	})

	t.Run("子账户更新账户密码-自身-无携带原密码", func(t *testing.T) {
		req := &api.ModifyUserPassword{
			Id:          &wrappers.StringValue{Value: userTest.users[1].ID},
			NewPassword: &wrappers.StringValue{Value: "users[1].Password"},
		}

		userTest.storage.EXPECT().GetUser(gomock.Any()).Return(userTest.users[1], nil)

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.users[1].Token)
		resp := userTest.svr.UpdateUserPassword(reqCtx, req)
		t.Logf("CreateUsers resp : %+v", resp)
		assert.Equal(t, api.ExecuteException, resp.Code.GetValue(), "update user must fail")
	})

	t.Run("子账户更新账户密码-不是自己", func(t *testing.T) {
		req := &api.ModifyUserPassword{
			Id:          &wrappers.StringValue{Value: userTest.users[2].ID},
			NewPassword: &wrappers.StringValue{Value: "users[2].Password"},
		}

		userTest.storage.EXPECT().GetUser(gomock.Any()).Return(userTest.users[2], nil)

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.users[1].Token)
		resp := userTest.svr.UpdateUserPassword(reqCtx, req)
		t.Logf("CreateUsers resp : %+v", resp)
		assert.Equal(t, api.NotAllowedAccess, resp.Code.GetValue(), "update user must fail")
	})
}

func Test_server_DeleteUser(t *testing.T) {
	userTest := newUserTest(t)
	defer userTest.Clean()

	t.Run("主账户删除自己", func(t *testing.T) {
		userTest.storage.EXPECT().GetUser(gomock.Any()).Return(userTest.users[0], nil)

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.users[0].Token)
		resp := userTest.svr.DeleteUser(reqCtx, &api.User{
			Id: utils.NewStringValue(userTest.users[0].ID),
		})

		assert.True(t, resp.GetCode().Value == api.NotAllowedAccess, resp.Info.GetValue())
	})

	t.Run("主账户删除另外一个主账户", func(t *testing.T) {
		uid := utils.NewUUID()
		userTest.storage.EXPECT().GetUser(gomock.Any()).Return(&model.User{
			ID:    uid,
			Type:  model.OwnerUserRole,
			Owner: "",
		}, nil)

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.users[0].Token)
		resp := userTest.svr.DeleteUser(reqCtx, &api.User{
			Id: utils.NewStringValue(uid),
		})

		assert.True(t, resp.GetCode().Value == api.NotAllowedAccess, resp.Info.GetValue())
	})

	t.Run("主账户删除自己的子账户", func(t *testing.T) {
		userTest.storage.EXPECT().GetUser(gomock.Eq(userTest.users[1].ID)).Return(userTest.users[1], nil)

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.users[0].Token)
		resp := userTest.svr.DeleteUser(reqCtx, &api.User{
			Id: utils.NewStringValue(userTest.users[1].ID),
		})

		assert.True(t, resp.GetCode().Value == api.ExecuteSuccess, resp.Info.GetValue())
	})

	t.Run("主账户删除不是自己的子账户", func(t *testing.T) {
		uid := utils.NewUUID()
		oid := utils.NewUUID()
		userTest.storage.EXPECT().GetUser(gomock.Any()).Return(&model.User{
			ID:    uid,
			Type:  model.OwnerUserRole,
			Owner: oid,
		}, nil)

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.users[0].Token)
		resp := userTest.svr.DeleteUser(reqCtx, &api.User{
			Id: utils.NewStringValue(uid),
		})

		assert.True(t, resp.GetCode().Value == api.NotAllowedAccess, resp.Info.GetValue())
	})

	t.Run("管理员删除主账户-主账户下没有子账户", func(t *testing.T) {
		userTest.storage.EXPECT().GetUser(gomock.Any()).Return(userTest.users[0], nil)
		userTest.storage.EXPECT().GetSubCount(gomock.Any()).Return(uint32(0), nil)

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.admin.Token)
		resp := userTest.svr.DeleteUser(reqCtx, &api.User{
			Id: utils.NewStringValue(userTest.users[0].ID),
		})

		assert.True(t, resp.GetCode().Value == api.ExecuteSuccess, resp.Info.GetValue())
	})

	t.Run("管理员删除主账户-主账户下还有子账户", func(t *testing.T) {
		userTest.storage.EXPECT().GetUser(gomock.Any()).Return(userTest.users[0], nil)
		userTest.storage.EXPECT().GetSubCount(gomock.Any()).Return(uint32(1), nil)

		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.admin.Token)
		resp := userTest.svr.DeleteUser(reqCtx, &api.User{
			Id: utils.NewStringValue(userTest.users[0].ID),
		})

		assert.True(t, resp.GetCode().Value == api.SubAccountExisted, resp.Info.GetValue())
	})

	t.Run("子账户删除用户", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.users[1].Token)
		resp := userTest.svr.DeleteUser(reqCtx, &api.User{
			Id: utils.NewStringValue(userTest.users[0].ID),
		})

		assert.True(t, resp.GetCode().Value == api.OperationRoleException, resp.Info.GetValue())
	})
}

func Test_server_GetUserToken(t *testing.T) {

	userTest := newUserTest(t)
	defer userTest.Clean()

	t.Run("主账户查询自己的Token", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.ownerOne.Token)

		resp := userTest.svr.GetUserToken(reqCtx, &api.User{
			Id: utils.NewStringValue(userTest.users[0].ID),
		})

		assert.True(t, resp.GetCode().Value == api.ExecuteSuccess, resp.Info.GetValue())
	})

	t.Run("子账户查询自己的Token", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.users[1].Token)

		resp := userTest.svr.GetUserToken(reqCtx, &api.User{
			Id: utils.NewStringValue(userTest.users[1].ID),
		})

		assert.True(t, resp.GetCode().Value == api.ExecuteSuccess, resp.Info.GetValue())
	})

	t.Run("主账户查询子账户的Token", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.ownerOne.Token)

		resp := userTest.svr.GetUserToken(reqCtx, &api.User{
			Id: utils.NewStringValue(userTest.users[1].ID),
		})

		assert.True(t, resp.GetCode().Value == api.ExecuteSuccess, resp.Info.GetValue())
	})

	t.Run("主账户查询别的主账户的Token", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.ownerOne.Token)

		resp := userTest.svr.GetUserToken(reqCtx, &api.User{
			Id: utils.NewStringValue(userTest.ownerTwo.ID),
		})

		assert.True(t, resp.GetCode().Value == api.NotAllowedAccess, resp.Info.GetValue())
	})

	t.Run("主账户查询不属于自己子账户的Token", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.ownerOne.Token)

		resp := userTest.svr.GetUserToken(reqCtx, &api.User{
			Id: utils.NewStringValue(userTest.newUsers[1].ID),
		})

		assert.True(t, resp.GetCode().Value == api.NotAllowedAccess, resp.Info.GetValue())
	})
}

func Test_server_RefreshUserToken(t *testing.T) {

	userTest := newUserTest(t)
	defer userTest.Clean()

	t.Run("主账户刷新自己的Token", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.ownerOne.Token)

		userTest.storage.EXPECT().GetUser(gomock.Any()).Return(userTest.users[0], nil)

		resp := userTest.svr.ResetUserToken(reqCtx, &api.User{
			Id: utils.NewStringValue(userTest.users[0].ID),
		})

		assert.True(t, resp.GetCode().Value == api.ExecuteSuccess, resp.Info.GetValue())
	})

	t.Run("子账户刷新自己的Token", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.users[1].Token)
		userTest.storage.EXPECT().GetUser(gomock.Any()).Return(userTest.users[1], nil)
		resp := userTest.svr.ResetUserToken(reqCtx, &api.User{
			Id: utils.NewStringValue(userTest.users[1].ID),
		})

		assert.True(t, resp.GetCode().Value == api.ExecuteSuccess, resp.Info.GetValue())
	})

	t.Run("主账户刷新子账户的Token", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.ownerOne.Token)
		userTest.storage.EXPECT().GetUser(gomock.Any()).Return(userTest.users[1], nil)
		resp := userTest.svr.ResetUserToken(reqCtx, &api.User{
			Id: utils.NewStringValue(userTest.users[1].ID),
		})

		assert.True(t, resp.GetCode().Value == api.ExecuteSuccess, resp.Info.GetValue())
	})

	t.Run("主账户刷新别的主账户的Token", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.ownerOne.Token)
		userTest.storage.EXPECT().GetUser(gomock.Any()).Return(userTest.ownerTwo, nil)
		resp := userTest.svr.ResetUserToken(reqCtx, &api.User{
			Id: utils.NewStringValue(userTest.ownerTwo.ID),
		})

		assert.True(t, resp.GetCode().Value == api.NotAllowedAccess, resp.Info.GetValue())
	})

	t.Run("主账户刷新不属于自己子账户的Token", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.ownerOne.Token)
		userTest.storage.EXPECT().GetUser(gomock.Any()).Return(userTest.newUsers[1], nil)
		resp := userTest.svr.ResetUserToken(reqCtx, &api.User{
			Id: utils.NewStringValue(userTest.newUsers[1].ID),
		})

		assert.True(t, resp.GetCode().Value == api.NotAllowedAccess, resp.Info.GetValue())
	})
}

func Test_server_UpdateUserToken(t *testing.T) {

	userTest := newUserTest(t)
	defer userTest.Clean()

	t.Run("主账户刷新自己的Token状态", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.ownerOne.Token)

		userTest.storage.EXPECT().GetUser(gomock.Eq(userTest.users[0].ID)).Return(userTest.users[0], nil)

		resp := userTest.svr.UpdateUserToken(reqCtx, &api.User{
			Id: utils.NewStringValue(userTest.users[0].ID),
		})

		assert.True(t, resp.GetCode().Value == api.NotAllowedAccess, resp.Info.GetValue())
	})

	t.Run("子账户刷新自己的Token状态", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.users[4].Token)
		resp := userTest.svr.UpdateUserToken(reqCtx, &api.User{
			Id: utils.NewStringValue(userTest.users[4].ID),
		})

		assert.True(t, resp.GetCode().Value == api.OperationRoleException, resp.Info.GetValue())
	})

	t.Run("主账户刷新子账户的Token状态", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.ownerOne.Token)
		userTest.storage.EXPECT().GetUser(gomock.Eq(userTest.users[3].ID)).Return(userTest.users[3], nil)
		resp := userTest.svr.UpdateUserToken(reqCtx, &api.User{
			Id: utils.NewStringValue(userTest.users[3].ID),
		})

		assert.True(t, resp.GetCode().Value == api.ExecuteSuccess, resp.Info.GetValue())
	})

	t.Run("主账户刷新别的主账户的Token状态", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.ownerOne.Token)

		t.Logf("operator-id : %s, user-two-owner : %s", userTest.ownerOne.ID, userTest.ownerTwo.Owner)

		userTest.storage.EXPECT().GetUser(gomock.Eq(userTest.ownerTwo.ID)).Return(userTest.ownerTwo, nil)
		resp := userTest.svr.UpdateUserToken(reqCtx, &api.User{
			Id: utils.NewStringValue(userTest.ownerTwo.ID),
		})

		assert.Truef(t, resp.GetCode().Value == api.NotAllowedAccess, "code=%d, msg=%s", resp.Code.GetValue(), resp.Info.GetValue())
	})

	t.Run("主账户刷新不属于自己子账户的Token状态", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, userTest.ownerOne.Token)
		userTest.storage.EXPECT().GetUser(gomock.Eq(userTest.newUsers[3].ID)).Return(userTest.newUsers[3], nil)
		resp := userTest.svr.UpdateUserToken(reqCtx, &api.User{
			Id: utils.NewStringValue(userTest.newUsers[3].ID),
		})

		assert.True(t, resp.GetCode().Value == api.NotAllowedAccess, resp.Info.GetValue())
	})
}
