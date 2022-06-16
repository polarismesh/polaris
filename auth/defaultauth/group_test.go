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
	"github.com/polarismesh/polaris-server/auth"
	"github.com/polarismesh/polaris-server/cache"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	v1 "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/plugin"
	storemock "github.com/polarismesh/polaris-server/store/mock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type GroupTest struct {
	ownerOne *model.User
	ownerTwo *model.User

	users     []*model.User
	groups    []*model.UserGroupDetail
	newGroups []*model.UserGroupDetail
	allGroups []*model.UserGroupDetail

	storage  *storemock.MockStore
	cacheMgn *cache.CacheManager
	checker  auth.AuthChecker

	svr *serverAuthAbility

	cancel context.CancelFunc
}

func newGroupTest(t *testing.T) *GroupTest {
	reset(false)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	users := createMockUser(10)
	groups := createMockUserGroup(users)

	newUsers := createMockUser(10)
	newGroups := createMockUserGroup(newUsers)

	allGroups := append(groups, newGroups...)

	storage := storemock.NewMockStore(ctrl)

	storage.EXPECT().GetUnixSecond().AnyTimes().Return(time.Now().Unix(), nil)
	storage.EXPECT().AddGroup(gomock.Any()).AnyTimes().Return(nil)
	storage.EXPECT().UpdateUser(gomock.Any()).AnyTimes().Return(nil)
	storage.EXPECT().GetUsersForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(append(users, newUsers...), nil)
	storage.EXPECT().GetGroupsForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(allGroups, nil)

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

	return &GroupTest{
		ownerOne: users[0],
		ownerTwo: newUsers[0],

		users:     users,
		groups:    groups,
		newGroups: newGroups,
		allGroups: allGroups,

		storage:  storage,
		cacheMgn: cacheMgn,
		checker:  checker,
		svr:      svr,

		cancel: cancel,
	}
}

func (g *GroupTest) Clean() {
	g.cancel()
	_ = g.cacheMgn.Clear()
	time.Sleep(2 * time.Second)
}

func Test_server_CreateGroup(t *testing.T) {
	groupTest := newGroupTest(t)

	defer groupTest.Clean()

	t.Run("正常创建用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.users[0].Token)

		groups := createMockUserGroup(groupTest.users[:1])
		groups[0].ID = utils.NewUUID()

		groupTest.storage.EXPECT().GetGroupByName(gomock.Any(), gomock.Any()).Return(nil, nil)

		resp := groupTest.svr.CreateGroup(reqCtx, &v1.UserGroup{
			Id:   utils.NewStringValue(groups[0].ID),
			Name: utils.NewStringValue(groups[0].Name),
		})

		assert.True(t, resp.GetCode().Value == v1.ExecuteSuccess, resp.Info.GetValue())
	})

	t.Run("用户组已存在", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.users[0].Token)

		groups := createMockUserGroup(groupTest.users[:1])
		groups[0].ID = utils.NewUUID()

		groupTest.storage.EXPECT().GetGroupByName(gomock.Any(), gomock.Any()).Return(groups[0].UserGroup, nil)

		resp := groupTest.svr.CreateGroup(reqCtx, &v1.UserGroup{
			Id:   utils.NewStringValue(groups[0].ID),
			Name: utils.NewStringValue(groups[0].Name),
		})

		assert.True(t, resp.GetCode().Value == v1.UserGroupExisted, resp.Info.GetValue())
	})

	t.Run("子用户去创建用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.users[1].Token)

		groups := createMockUserGroup(groupTest.users[:1])
		groups[0].ID = utils.NewUUID()

		resp := groupTest.svr.CreateGroup(reqCtx, &v1.UserGroup{
			Id:   utils.NewStringValue(groups[0].ID),
			Name: utils.NewStringValue(groups[0].Name),
		})

		assert.True(t, resp.GetCode().Value == v1.OperationRoleException, resp.Info.GetValue())
	})

	t.Run("主账户去查询owner为自己的用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.users[0].Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).Return(groupTest.groups[1], nil)

		resp := groupTest.svr.GetGroup(reqCtx, &v1.UserGroup{
			Id: utils.NewStringValue(groupTest.groups[1].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.ExecuteSuccess, resp.Info.GetValue())
	})

	t.Run("主账户去查询owner不是自己的用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.ownerTwo.Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).Return(groupTest.groups[3], nil)

		resp := groupTest.svr.GetGroup(reqCtx, &v1.UserGroup{
			Id: utils.NewStringValue(groupTest.groups[3].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.NotAllowedAccess, resp.Info.GetValue())
	})

	t.Run("子账户去查询自己所在的用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.users[1].Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).Return(groupTest.groups[1], nil)

		resp := groupTest.svr.GetGroup(reqCtx, &v1.UserGroup{
			Id: utils.NewStringValue(groupTest.groups[1].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.ExecuteSuccess, resp.Info.GetValue())
	})

	t.Run("子账户去查询自己不在的用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.users[1].Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).Return(groupTest.groups[2], nil)

		resp := groupTest.svr.GetGroup(reqCtx, &v1.UserGroup{
			Id: utils.NewStringValue(groupTest.groups[2].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.NotAllowedAccess, resp.Info.GetValue())
	})
}

func Test_server_GetGroup(t *testing.T) {
	groupTest := newGroupTest(t)

	defer groupTest.Clean()
	t.Run("主账户去查询owner为自己的用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.users[0].Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).Return(groupTest.groups[1], nil)

		resp := groupTest.svr.GetGroup(reqCtx, &v1.UserGroup{
			Id: utils.NewStringValue(groupTest.groups[1].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.ExecuteSuccess, resp.Info.GetValue())
	})

	t.Run("主账户去查询owner不是自己的用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.ownerOne.Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).Return(groupTest.newGroups[0], nil)

		resp := groupTest.svr.GetGroup(reqCtx, &v1.UserGroup{
			Id: utils.NewStringValue(groupTest.newGroups[0].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.NotAllowedAccess, resp.Info.GetValue())
	})

	t.Run("子账户去查询自己所在的用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.users[1].Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).Return(groupTest.groups[1], nil)

		resp := groupTest.svr.GetGroup(reqCtx, &v1.UserGroup{
			Id: utils.NewStringValue(groupTest.groups[1].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.ExecuteSuccess, resp.Info.GetValue())
	})

	t.Run("子账户去查询自己不在的用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.users[1].Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).Return(groupTest.groups[2], nil)

		resp := groupTest.svr.GetGroup(reqCtx, &v1.UserGroup{
			Id: utils.NewStringValue(groupTest.groups[2].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.NotAllowedAccess, resp.Info.GetValue())
	})
}

func Test_server_UpdateGroup(t *testing.T) {
	groupTest := newGroupTest(t)

	defer groupTest.Clean()

	t.Run("主账户更新用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.ownerOne.Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).Return(groupTest.groups[1], nil)
		groupTest.storage.EXPECT().UpdateGroup(gomock.Any()).Return(nil)

		req := &v1.ModifyUserGroup{
			Id: utils.NewStringValue(groupTest.groups[1].ID),
			Comment: &wrapperspb.StringValue{
				Value: "new test group",
			},
			AddRelations: &v1.UserGroupRelation{
				GroupId: utils.NewStringValue(groupTest.groups[1].ID),
				Users: []*v1.User{
					{
						Id: utils.NewStringValue(groupTest.users[2].ID),
					},
					{
						Id: utils.NewStringValue(groupTest.users[3].ID),
					},
				},
			},
			RemoveRelations: &v1.UserGroupRelation{
				GroupId: utils.NewStringValue(groupTest.groups[1].ID),
				Users: []*v1.User{
					{
						Id: utils.NewStringValue(groupTest.users[5].ID),
					},
				},
			},
		}

		resp := groupTest.svr.UpdateGroups(reqCtx, []*v1.ModifyUserGroup{req})

		assert.True(t, resp.Responses[0].Code.GetValue() == v1.ExecuteSuccess, resp.Responses[0].Info.GetValue())
	})

	t.Run("主账户更新不是自己负责的用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.ownerOne.Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).Return(groupTest.newGroups[1], nil)

		req := &v1.ModifyUserGroup{
			Id: utils.NewStringValue(groupTest.newGroups[0].ID),
			Comment: &wrapperspb.StringValue{
				Value: "new test group",
			},
			AddRelations: &v1.UserGroupRelation{
				GroupId: utils.NewStringValue(groupTest.groups[0].ID),
				Users: []*v1.User{
					{
						Id: utils.NewStringValue(groupTest.users[2].ID),
					},
					{
						Id: utils.NewStringValue(groupTest.users[3].ID),
					},
				},
			},
			RemoveRelations: &v1.UserGroupRelation{
				GroupId: utils.NewStringValue(groupTest.groups[0].ID),
				Users: []*v1.User{
					{
						Id: utils.NewStringValue(groupTest.users[5].ID),
					},
				},
			},
		}

		resp := groupTest.svr.UpdateGroups(reqCtx, []*v1.ModifyUserGroup{req})
		assert.True(t, resp.Responses[0].Code.GetValue() == v1.NotAllowedAccess, resp.Responses[0].Info.GetValue())
	})

	t.Run("子账户更新用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.users[1].Token)

		req := &v1.ModifyUserGroup{
			Id: utils.NewStringValue(groupTest.groups[2].ID),
			Comment: &wrapperspb.StringValue{
				Value: "new test group",
			},
			AddRelations: &v1.UserGroupRelation{
				GroupId: utils.NewStringValue(groupTest.groups[2].ID),
				Users: []*v1.User{
					{
						Id: utils.NewStringValue(groupTest.users[2].ID),
					},
					{
						Id: utils.NewStringValue(groupTest.users[3].ID),
					},
				},
			},
			RemoveRelations: &v1.UserGroupRelation{
				GroupId: utils.NewStringValue(groupTest.groups[2].ID),
				Users: []*v1.User{
					{
						Id: utils.NewStringValue(groupTest.users[5].ID),
					},
				},
			},
		}

		resp := groupTest.svr.UpdateGroups(reqCtx, []*v1.ModifyUserGroup{req})
		assert.True(t, resp.Responses[0].GetCode().Value == v1.OperationRoleException, resp.Responses[0].Info.GetValue())
	})

	t.Run("更新用户组-啥都没用动过", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.users[0].Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).Return(groupTest.groups[2], nil)

		req := &v1.ModifyUserGroup{
			Id:              utils.NewStringValue(groupTest.groups[2].ID),
			Comment:         &wrapperspb.StringValue{Value: groupTest.groups[2].Comment},
			AddRelations:    &v1.UserGroupRelation{},
			RemoveRelations: &v1.UserGroupRelation{},
		}

		resp := groupTest.svr.UpdateGroups(reqCtx, []*v1.ModifyUserGroup{req})
		assert.True(t, resp.Responses[0].GetCode().Value == v1.NoNeedUpdate, resp.Responses[0].Info.GetValue())
	})

}

func Test_server_GetGroupToken(t *testing.T) {
	groupTest := newGroupTest(t)

	defer groupTest.Clean()

	t.Run("主账户去查询owner为自己的用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.users[0].Token)

		resp := groupTest.svr.GetGroupToken(reqCtx, &v1.UserGroup{
			Id: utils.NewStringValue(groupTest.groups[1].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.ExecuteSuccess, resp.Info.GetValue())
	})

	t.Run("主账户去查询owner不是自己的用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.ownerTwo.Token)

		resp := groupTest.svr.GetGroupToken(reqCtx, &v1.UserGroup{
			Id: utils.NewStringValue(groupTest.groups[1].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.NotAllowedAccess, resp.Info.GetValue())
	})

	t.Run("子账户去查询自己所在的用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.users[1].Token)

		resp := groupTest.svr.GetGroupToken(reqCtx, &v1.UserGroup{
			Id: utils.NewStringValue(groupTest.groups[1].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.ExecuteSuccess, resp.Info.GetValue())
	})

	t.Run("子账户去查询自己不在的用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.users[1].Token)

		resp := groupTest.svr.GetGroupToken(reqCtx, &v1.UserGroup{
			Id: utils.NewStringValue(groupTest.groups[2].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.NotAllowedAccess, resp.Info.GetValue())
	})
}

func Test_server_DeleteGroup(t *testing.T) {

	groupTest := newGroupTest(t)

	defer groupTest.Clean()

	t.Run("正常删除用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.ownerOne.Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).Return(groupTest.groups[0], nil)
		groupTest.storage.EXPECT().DeleteGroup(gomock.Any()).Return(nil)

		batchResp := groupTest.svr.DeleteGroups(reqCtx, []*api.UserGroup{
			{
				Id: utils.NewStringValue(groupTest.groups[0].ID),
			},
		})

		assert.True(t, batchResp.GetCode().Value == v1.ExecuteSuccess, batchResp.Info.GetValue())
	})

	t.Run("删除用户组-用户组不存在", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.ownerOne.Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).Return(nil, nil)
		groupTest.storage.EXPECT().DeleteGroup(gomock.Any()).Return(nil)

		batchResp := groupTest.svr.DeleteGroups(reqCtx, []*api.UserGroup{
			{
				Id: utils.NewStringValue(groupTest.groups[0].ID),
			},
		})

		assert.True(t, batchResp.GetCode().Value == v1.ExecuteSuccess, batchResp.Info.GetValue())
	})

	t.Run("删除用户组-不是用户组的owner", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.ownerTwo.Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).Return(groupTest.groups[0], nil)
		groupTest.storage.EXPECT().DeleteGroup(gomock.Any()).Return(nil)

		batchResp := groupTest.svr.DeleteGroups(reqCtx, []*api.UserGroup{
			{
				Id: utils.NewStringValue(groupTest.groups[0].ID),
			},
		})

		assert.True(t, batchResp.Responses[0].GetCode().Value == v1.NotAllowedAccess, batchResp.Responses[0].Info.GetValue())
	})

	t.Run("删除用户组-非owner角色", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.users[1].Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).Return(groupTest.groups[0], nil)
		groupTest.storage.EXPECT().DeleteGroup(gomock.Any()).Return(nil)

		batchResp := groupTest.svr.DeleteGroups(reqCtx, []*api.UserGroup{
			{
				Id: utils.NewStringValue(groupTest.groups[0].ID),
			},
		})

		assert.True(t, batchResp.Responses[0].GetCode().Value == v1.OperationRoleException, batchResp.Responses[0].Info.GetValue())
	})

}



func Test_server_UpdateGroupToken(t *testing.T) {

	groupTest := newGroupTest(t)

	defer groupTest.Clean()

	t.Run("正常更新用户组Token的Enable状态", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.ownerOne.Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).Return(groupTest.groups[0], nil)
		groupTest.storage.EXPECT().UpdateGroup(gomock.Any()).Return(nil)

		batchResp := groupTest.svr.UpdateGroupToken(reqCtx, &api.UserGroup{
			Id: utils.NewStringValue(groupTest.groups[2].ID),
		})

		assert.True(t, batchResp.GetCode().Value == v1.ExecuteSuccess, batchResp.Info.GetValue())
	})

	t.Run("非Owner角色更新用户组Token的Enable状态", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.users[2].Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).Return(groupTest.groups[0], nil)


		batchResp := groupTest.svr.UpdateGroupToken(reqCtx, &api.UserGroup{
			Id: utils.NewStringValue(groupTest.groups[2].ID),
		})

		assert.True(t, batchResp.Code.Value == v1.OperationRoleException, batchResp.Info.GetValue())
	})


	t.Run("更新用户组Token的Enable状态-非group的owner", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.ownerTwo.Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).Return(groupTest.groups[0], nil)

		batchResp := groupTest.svr.UpdateGroupToken(reqCtx, &api.UserGroup{
			Id: utils.NewStringValue(groupTest.groups[2].ID),
		})

		assert.True(t, batchResp.Code.Value == v1.NotAllowedAccess, batchResp.Info.GetValue())
	})
}



func Test_server_RefreshGroupToken(t *testing.T) {

	groupTest := newGroupTest(t)

	defer groupTest.Clean()

	t.Run("正常更新用户组Token的Enable状态", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.ownerOne.Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).Return(groupTest.groups[0], nil)
		groupTest.storage.EXPECT().UpdateGroup(gomock.Any()).Return(nil)

		batchResp := groupTest.svr.ResetGroupToken(reqCtx, &api.UserGroup{
			Id: utils.NewStringValue(groupTest.groups[2].ID),
		})

		assert.True(t, batchResp.GetCode().Value == v1.ExecuteSuccess, batchResp.Info.GetValue())
	})

	t.Run("非Owner角色更新用户组Token的Enable状态", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.users[2].Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).Return(groupTest.groups[0], nil)

		batchResp := groupTest.svr.ResetGroupToken(reqCtx, &api.UserGroup{
			Id: utils.NewStringValue(groupTest.groups[2].ID),
		})

		assert.True(t, batchResp.Code.Value == v1.OperationRoleException, batchResp.Info.GetValue())
	})


	t.Run("更新用户组Token的Enable状态-非group的owner", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.ownerTwo.Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).Return(groupTest.groups[0], nil)

		batchResp := groupTest.svr.ResetGroupToken(reqCtx, &api.UserGroup{
			Id: utils.NewStringValue(groupTest.groups[2].ID),
		})

		assert.True(t, batchResp.Code.Value == v1.NotAllowedAccess, batchResp.Info.GetValue())
	})
}

