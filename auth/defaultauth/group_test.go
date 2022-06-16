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
	"github.com/polarismesh/polaris-server/cache"
	v1 "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/plugin"
	storemock "github.com/polarismesh/polaris-server/store/mock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func Test_server_CreateGroup(t *testing.T) {
	reset(false)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	users := createMockUser(10)
	groups := createMockUserGroup(users)

	newGroups := createMockUserGroup(createMockUser(10))

	groups = append(groups, newGroups...)

	storage := storemock.NewMockStore(ctrl)

	storage.EXPECT().GetUnixSecond().AnyTimes().Return(time.Now().Unix(), nil)
	storage.EXPECT().AddGroup(gomock.Any()).AnyTimes().Return(nil)
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

	t.Run("正常创建用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[0].Token)

		groups := createMockUserGroup(users[:1])
		groups[0].ID = utils.NewUUID()

		storage.EXPECT().GetGroupByName(gomock.Any(), gomock.Any()).Return(nil, nil)

		resp := svr.CreateGroup(reqCtx, &v1.UserGroup{
			Id:   utils.NewStringValue(groups[0].ID),
			Name: utils.NewStringValue(groups[0].Name),
		})

		assert.True(t, resp.GetCode().Value == v1.ExecuteSuccess, resp.Info.GetValue())
	})

	t.Run("用户组已存在", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[0].Token)

		groups := createMockUserGroup(users[:1])
		groups[0].ID = utils.NewUUID()

		storage.EXPECT().GetGroupByName(gomock.Any(), gomock.Any()).Return(groups[0].UserGroup, nil)

		resp := svr.CreateGroup(reqCtx, &v1.UserGroup{
			Id:   utils.NewStringValue(groups[0].ID),
			Name: utils.NewStringValue(groups[0].Name),
		})

		assert.True(t, resp.GetCode().Value == v1.UserGroupExisted, resp.Info.GetValue())
	})

	t.Run("子用户去创建用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)

		groups := createMockUserGroup(users[:1])
		groups[0].ID = utils.NewUUID()

		resp := svr.CreateGroup(reqCtx, &v1.UserGroup{
			Id:   utils.NewStringValue(groups[0].ID),
			Name: utils.NewStringValue(groups[0].Name),
		})

		assert.True(t, resp.GetCode().Value == v1.OperationRoleException, resp.Info.GetValue())
	})

	t.Run("主账户去查询owner为自己的用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[0].Token)

		storage.EXPECT().GetGroup(gomock.Any()).Return(groups[1], nil)

		resp := svr.GetGroup(reqCtx, &v1.UserGroup{
			Id: utils.NewStringValue(groups[1].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.ExecuteSuccess, resp.Info.GetValue())
	})

	t.Run("主账户去查询owner不是自己的用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[0].Token)

		storage.EXPECT().GetGroup(gomock.Any()).Return(groups[len(groups)-1], nil)

		resp := svr.GetGroup(reqCtx, &v1.UserGroup{
			Id: utils.NewStringValue(groups[len(groups)-1].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.NotAllowedAccess, resp.Info.GetValue())
	})

	t.Run("子账户去查询自己所在的用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)

		storage.EXPECT().GetGroup(gomock.Any()).Return(groups[1], nil)

		resp := svr.GetGroup(reqCtx, &v1.UserGroup{
			Id: utils.NewStringValue(groups[1].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.ExecuteSuccess, resp.Info.GetValue())
	})

	t.Run("子账户去查询自己不在的用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)

		storage.EXPECT().GetGroup(gomock.Any()).Return(groups[2], nil)

		resp := svr.GetGroup(reqCtx, &v1.UserGroup{
			Id: utils.NewStringValue(groups[2].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.NotAllowedAccess, resp.Info.GetValue())
	})
}

func Test_server_GetGroup(t *testing.T) {
	reset(false)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	users := createMockUser(10)
	groups := createMockUserGroup(users)

	newGroups := createMockUserGroup(createMockUser(10))

	groups = append(groups, newGroups...)

	storage := storemock.NewMockStore(ctrl)

	storage.EXPECT().GetUnixSecond().AnyTimes().Return(time.Now().Unix(), nil)
	storage.EXPECT().AddGroup(gomock.Any()).AnyTimes().Return(nil)
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

	t.Run("主账户去查询owner为自己的用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[0].Token)

		storage.EXPECT().GetGroup(gomock.Any()).Return(groups[1], nil)

		resp := svr.GetGroup(reqCtx, &v1.UserGroup{
			Id: utils.NewStringValue(groups[1].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.ExecuteSuccess, resp.Info.GetValue())
	})

	t.Run("主账户去查询owner不是自己的用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[0].Token)

		storage.EXPECT().GetGroup(gomock.Any()).Return(groups[len(groups)-1], nil)

		resp := svr.GetGroup(reqCtx, &v1.UserGroup{
			Id: utils.NewStringValue(groups[len(groups)-1].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.NotAllowedAccess, resp.Info.GetValue())
	})

	t.Run("子账户去查询自己所在的用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)

		storage.EXPECT().GetGroup(gomock.Any()).Return(groups[1], nil)

		resp := svr.GetGroup(reqCtx, &v1.UserGroup{
			Id: utils.NewStringValue(groups[1].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.ExecuteSuccess, resp.Info.GetValue())
	})

	t.Run("子账户去查询自己不在的用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)

		storage.EXPECT().GetGroup(gomock.Any()).Return(groups[2], nil)

		resp := svr.GetGroup(reqCtx, &v1.UserGroup{
			Id: utils.NewStringValue(groups[2].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.NotAllowedAccess, resp.Info.GetValue())
	})
}

func Test_server_UpdateGroup(t *testing.T) {
	reset(false)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	users := createMockUser(10)
	groups := createMockUserGroup(users)

	newGroups := createMockUserGroup(createMockUser(10))

	groups = append(groups, newGroups...)

	storage := storemock.NewMockStore(ctrl)

	storage.EXPECT().GetUnixSecond().AnyTimes().Return(time.Now().Unix(), nil)
	storage.EXPECT().AddGroup(gomock.Any()).AnyTimes().Return(nil)
	storage.EXPECT().UpdateGroup(gomock.Any()).AnyTimes().Return(nil)
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

	t.Run("主账户更新用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[0].Token)

		storage.EXPECT().GetGroup(gomock.Any()).Return(groups[1], nil)

		req := &v1.ModifyUserGroup{
			Id: utils.NewStringValue(groups[1].ID),
			Comment: &wrapperspb.StringValue{
				Value: "new test group",
			},
			AddRelations: &v1.UserGroupRelation{
				GroupId: utils.NewStringValue(groups[1].ID),
				Users: []*v1.User{
					{
						Id: utils.NewStringValue(users[2].ID),
					},
					{
						Id: utils.NewStringValue(users[3].ID),
					},
				},
			},
			RemoveRelations: &v1.UserGroupRelation{
				GroupId: utils.NewStringValue(groups[1].ID),
				Users: []*v1.User{
					{
						Id: utils.NewStringValue(users[5].ID),
					},
				},
			},
		}

		resp := svr.UpdateGroups(reqCtx, []*v1.ModifyUserGroup{req})

		assert.True(t, resp.Responses[0].Code.GetValue() == v1.ExecuteSuccess, resp.Responses[0].Info.GetValue())
	})

	t.Run("主账户更新不是自己负责的用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[0].Token)

		req := &v1.ModifyUserGroup{
			Id: utils.NewStringValue(groups[len(groups)-1].ID),
			Comment: &wrapperspb.StringValue{
				Value: "new test group",
			},
			AddRelations: &v1.UserGroupRelation{
				GroupId: utils.NewStringValue(groups[len(groups)-1].ID),
				Users: []*v1.User{
					{
						Id: utils.NewStringValue(users[2].ID),
					},
					{
						Id: utils.NewStringValue(users[3].ID),
					},
				},
			},
			RemoveRelations: &v1.UserGroupRelation{
				GroupId: utils.NewStringValue(groups[len(groups)-1].ID),
				Users: []*v1.User{
					{
						Id: utils.NewStringValue(users[5].ID),
					},
				},
			},
		}

		resp := svr.UpdateGroups(reqCtx, []*v1.ModifyUserGroup{req})
		assert.True(t, resp.Responses[0].Code.GetValue() == v1.NotAllowedAccess, resp.Responses[0].Info.GetValue())
	})

	t.Run("子账户更新用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)

		req := &v1.ModifyUserGroup{
			Id: utils.NewStringValue(groups[2].ID),
			Comment: &wrapperspb.StringValue{
				Value: "new test group",
			},
			AddRelations: &v1.UserGroupRelation{
				GroupId: utils.NewStringValue(groups[2].ID),
				Users: []*v1.User{
					{
						Id: utils.NewStringValue(users[2].ID),
					},
					{
						Id: utils.NewStringValue(users[3].ID),
					},
				},
			},
			RemoveRelations: &v1.UserGroupRelation{
				GroupId: utils.NewStringValue(groups[2].ID),
				Users: []*v1.User{
					{
						Id: utils.NewStringValue(users[5].ID),
					},
				},
			},
		}

		resp := svr.UpdateGroups(reqCtx, []*v1.ModifyUserGroup{req})
		assert.True(t, resp.Responses[0].GetCode().Value == v1.OperationRoleException, resp.Responses[0].Info.GetValue())
	})

}

func Test_server_GetGroupToken(t *testing.T) {
	reset(false)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	users := createMockUser(10)
	groups := createMockUserGroup(users)

	newGroups := createMockUserGroup(createMockUser(10))

	groups = append(groups, newGroups...)

	storage := storemock.NewMockStore(ctrl)
	storage.EXPECT().GetUnixSecond().AnyTimes().Return(time.Now().Unix(), nil)
	storage.EXPECT().AddGroup(gomock.Any()).AnyTimes().Return(nil)
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

	t.Run("主账户去查询owner为自己的用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[0].Token)

		resp := svr.GetGroupToken(reqCtx, &v1.UserGroup{
			Id: utils.NewStringValue(groups[1].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.ExecuteSuccess, resp.Info.GetValue())
	})

	t.Run("主账户去查询owner不是自己的用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[0].Token)

		resp := svr.GetGroupToken(reqCtx, &v1.UserGroup{
			Id: utils.NewStringValue(groups[len(groups)-1].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.NotAllowedAccess, resp.Info.GetValue())
	})

	t.Run("子账户去查询自己所在的用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)

		resp := svr.GetGroupToken(reqCtx, &v1.UserGroup{
			Id: utils.NewStringValue(groups[1].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.ExecuteSuccess, resp.Info.GetValue())
	})

	t.Run("子账户去查询自己不在的用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)

		resp := svr.GetGroupToken(reqCtx, &v1.UserGroup{
			Id: utils.NewStringValue(groups[2].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.NotAllowedAccess, resp.Info.GetValue())
	})
}
