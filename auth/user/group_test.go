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

package defaultuser_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/polarismesh/polaris/auth"
	defaultauth "github.com/polarismesh/polaris/auth/user"
	"github.com/polarismesh/polaris/cache"
	cachetypes "github.com/polarismesh/polaris/cache/api"
	v1 "github.com/polarismesh/polaris/common/api/v1"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
	storemock "github.com/polarismesh/polaris/store/mock"
)

type GroupTest struct {
	ctrl *gomock.Controller

	ownerOne *authcommon.User
	ownerTwo *authcommon.User

	users     []*authcommon.User
	groups    []*authcommon.UserGroupDetail
	newGroups []*authcommon.UserGroupDetail
	allGroups []*authcommon.UserGroupDetail

	storage  *storemock.MockStore
	cacheMgn *cache.CacheManager
	checker  auth.AuthChecker
	cancel   context.CancelFunc

	svr auth.UserServer
}

func newGroupTest(t *testing.T) *GroupTest {
	reset(false)
	ctrl := gomock.NewController(t)

	users := createMockUser(10)
	groups := createMockUserGroup(users)

	newUsers := createMockUser(10)
	newGroups := createMockUserGroup(newUsers)

	allGroups := append(groups, newGroups...)

	storage := storemock.NewMockStore(ctrl)

	storage.EXPECT().GetServicesCount().AnyTimes().Return(uint32(1), nil)
	storage.EXPECT().GetUnixSecond(gomock.Any()).AnyTimes().Return(time.Now().Unix(), nil)
	storage.EXPECT().AddGroup(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
	storage.EXPECT().UpdateUser(gomock.Any()).AnyTimes().Return(nil)
	storage.EXPECT().GetUsersForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(append(users, newUsers...), nil)
	storage.EXPECT().GetGroupsForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(allGroups, nil)

	cfg := &cache.Config{}

	ctx, cancel := context.WithCancel(context.Background())
	cacheMgn, err := cache.TestCacheInitialize(ctx, cfg, storage)
	if err != nil {
		t.Error(err)
	}
	_ = cacheMgn.OpenResourceCache([]cachetypes.ConfigEntry{
		{
			Name: cachetypes.UsersName,
		},
	}...)
	t.Cleanup(func() {
		_ = cacheMgn.Close()
	})

	_, proxySvr, err := defaultauth.BuildServer()
	if err != nil {
		t.Fatal(err)
	}
	proxySvr.Initialize(&auth.Config{
		User: &auth.UserConfig{
			Name: auth.DefaultUserMgnPluginName,
			Option: map[string]interface{}{
				"salt": "polarismesh@2021",
			},
		},
	}, storage, nil, cacheMgn)
	_ = cacheMgn.TestUpdate()
	return &GroupTest{
		ctrl:      ctrl,
		ownerOne:  users[0],
		ownerTwo:  newUsers[0],
		users:     users,
		groups:    groups,
		newGroups: newGroups,
		allGroups: allGroups,
		storage:   storage,
		cacheMgn:  cacheMgn,
		cancel:    cancel,
		svr:       proxySvr,
	}
}

func (g *GroupTest) Clean() {
	g.cancel()
	g.cacheMgn.Close()
	g.ctrl.Finish()
}

func Test_server_CreateGroup(t *testing.T) {
	groupTest := newGroupTest(t)

	defer groupTest.Clean()

	t.Run("正常创建用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.users[0].Token)

		groups := createMockUserGroup(groupTest.users[:1])
		groups[0].ID = utils.NewUUID()

		groupTest.storage.EXPECT().GetGroupByName(gomock.Any(), gomock.Any()).Return(nil, nil)

		resp := groupTest.svr.CreateGroup(reqCtx, &apisecurity.UserGroup{
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

		resp := groupTest.svr.CreateGroup(reqCtx, &apisecurity.UserGroup{
			Id:   utils.NewStringValue(groups[0].ID),
			Name: utils.NewStringValue(groups[0].Name),
		})

		assert.True(t, resp.GetCode().Value == v1.UserGroupExisted, resp.Info.GetValue())
	})

	t.Run("子用户去创建用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.users[1].Token)

		groups := createMockUserGroup(groupTest.users[:1])
		groups[0].ID = utils.NewUUID()

		resp := groupTest.svr.CreateGroup(reqCtx, &apisecurity.UserGroup{
			Id:   utils.NewStringValue(groups[0].ID),
			Name: utils.NewStringValue(groups[0].Name),
		})

		assert.True(t, resp.GetCode().Value == v1.OperationRoleException, resp.Info.GetValue())
	})

	t.Run("主账户去查询owner为自己的用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.users[0].Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).Return(groupTest.groups[1], nil)

		resp := groupTest.svr.GetGroup(reqCtx, &apisecurity.UserGroup{
			Id: utils.NewStringValue(groupTest.groups[1].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.ExecuteSuccess, resp.Info.GetValue())
	})

	t.Run("主账户去查询owner不是自己的用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.ownerTwo.Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).Return(groupTest.groups[3], nil)

		resp := groupTest.svr.GetGroup(reqCtx, &apisecurity.UserGroup{
			Id: utils.NewStringValue(groupTest.groups[3].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.NotAllowedAccess, resp.Info.GetValue())
	})

	t.Run("子账户去查询自己所在的用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.users[1].Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).Return(groupTest.groups[1], nil).AnyTimes()

		resp := groupTest.svr.GetGroup(reqCtx, &apisecurity.UserGroup{
			Id: utils.NewStringValue(groupTest.groups[1].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.ExecuteSuccess, resp.Info.GetValue())
	})

	t.Run("子账户去查询自己不在的用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.users[1].Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).Return(groupTest.groups[2], nil).AnyTimes()

		resp := groupTest.svr.GetGroup(reqCtx, &apisecurity.UserGroup{
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

		resp := groupTest.svr.GetGroup(reqCtx, &apisecurity.UserGroup{
			Id: utils.NewStringValue(groupTest.groups[1].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.ExecuteSuccess, resp.Info.GetValue())
	})

	t.Run("主账户去查询owner不是自己的用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.ownerOne.Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).Return(groupTest.newGroups[0], nil)

		resp := groupTest.svr.GetGroup(reqCtx, &apisecurity.UserGroup{
			Id: utils.NewStringValue(groupTest.newGroups[0].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.NotAllowedAccess, resp.Info.GetValue())
	})

	t.Run("子账户去查询自己所在的用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.users[1].Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).Return(groupTest.groups[1], nil).AnyTimes()

		resp := groupTest.svr.GetGroup(reqCtx, &apisecurity.UserGroup{
			Id: utils.NewStringValue(groupTest.groups[1].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.ExecuteSuccess, resp.Info.GetValue())
	})

	t.Run("子账户去查询自己不在的用户组", func(t *testing.T) {
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.users[1].Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).Return(groupTest.groups[2], nil).AnyTimes()

		resp := groupTest.svr.GetGroup(reqCtx, &apisecurity.UserGroup{
			Id: utils.NewStringValue(groupTest.groups[2].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.NotAllowedAccess, resp.Info.GetValue())
	})
}

func Test_server_UpdateGroup(t *testing.T) {
	t.Run("主账户更新用户组", func(t *testing.T) {
		groupTest := newGroupTest(t)

		t.Cleanup(func() {
			groupTest.Clean()
		})
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.ownerOne.Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).Return(groupTest.groups[1], nil)
		groupTest.storage.EXPECT().UpdateGroup(gomock.Any()).Return(nil)

		req := &apisecurity.ModifyUserGroup{
			Id: utils.NewStringValue(groupTest.groups[1].ID),
			Comment: &wrapperspb.StringValue{
				Value: "new test group",
			},
			AddRelations: &apisecurity.UserGroupRelation{
				GroupId: utils.NewStringValue(groupTest.groups[1].ID),
				Users: []*apisecurity.User{
					{
						Id: utils.NewStringValue(groupTest.users[2].ID),
					},
					{
						Id: utils.NewStringValue(groupTest.users[3].ID),
					},
				},
			},
			RemoveRelations: &apisecurity.UserGroupRelation{
				GroupId: utils.NewStringValue(groupTest.groups[1].ID),
				Users: []*apisecurity.User{
					{
						Id: utils.NewStringValue(groupTest.users[5].ID),
					},
				},
			},
		}

		resp := groupTest.svr.UpdateGroups(reqCtx, []*apisecurity.ModifyUserGroup{req})

		assert.True(t, resp.Responses[0].Code.GetValue() == v1.ExecuteSuccess, resp.Responses[0].Info.GetValue())
	})

	t.Run("主账户更新不是自己负责的用户组", func(t *testing.T) {
		groupTest := newGroupTest(t)

		t.Cleanup(func() {
			groupTest.Clean()
		})
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.ownerOne.Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).AnyTimes().Return(groupTest.newGroups[1], nil)

		req := &apisecurity.ModifyUserGroup{
			Id: utils.NewStringValue(groupTest.newGroups[0].ID),
			Comment: &wrapperspb.StringValue{
				Value: "new test group",
			},
			AddRelations: &apisecurity.UserGroupRelation{
				GroupId: utils.NewStringValue(groupTest.groups[0].ID),
				Users: []*apisecurity.User{
					{
						Id: utils.NewStringValue(groupTest.users[2].ID),
					},
					{
						Id: utils.NewStringValue(groupTest.users[3].ID),
					},
				},
			},
			RemoveRelations: &apisecurity.UserGroupRelation{
				GroupId: utils.NewStringValue(groupTest.groups[0].ID),
				Users: []*apisecurity.User{
					{
						Id: utils.NewStringValue(groupTest.users[5].ID),
					},
				},
			},
		}

		resp := groupTest.svr.UpdateGroups(reqCtx, []*apisecurity.ModifyUserGroup{req})
		assert.True(t, resp.Responses[0].Code.GetValue() == v1.NotAllowedAccess, resp.Responses[0].Info.GetValue())
	})

	t.Run("子账户更新用户组", func(t *testing.T) {
		groupTest := newGroupTest(t)

		t.Cleanup(func() {
			groupTest.Clean()
		})
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.users[1].Token)

		req := &apisecurity.ModifyUserGroup{
			Id: utils.NewStringValue(groupTest.groups[2].ID),
			Comment: &wrapperspb.StringValue{
				Value: "new test group",
			},
			AddRelations: &apisecurity.UserGroupRelation{
				GroupId: utils.NewStringValue(groupTest.groups[2].ID),
				Users: []*apisecurity.User{
					{
						Id: utils.NewStringValue(groupTest.users[2].ID),
					},
					{
						Id: utils.NewStringValue(groupTest.users[3].ID),
					},
				},
			},
			RemoveRelations: &apisecurity.UserGroupRelation{
				GroupId: utils.NewStringValue(groupTest.groups[2].ID),
				Users: []*apisecurity.User{
					{
						Id: utils.NewStringValue(groupTest.users[5].ID),
					},
				},
			},
		}

		resp := groupTest.svr.UpdateGroups(reqCtx, []*apisecurity.ModifyUserGroup{req})
		assert.True(t, resp.Responses[0].GetCode().Value == v1.OperationRoleException, resp.Responses[0].Info.GetValue())
	})

	t.Run("更新用户组-啥都没用动过", func(t *testing.T) {
		groupTest := newGroupTest(t)

		t.Cleanup(func() {
			groupTest.Clean()
		})
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.users[0].Token)
		groupTest.storage.EXPECT().GetGroup(gomock.Any()).Return(groupTest.groups[2], nil)

		req := &apisecurity.ModifyUserGroup{
			Id:              utils.NewStringValue(groupTest.groups[2].ID),
			Comment:         &wrapperspb.StringValue{Value: groupTest.groups[2].Comment},
			AddRelations:    &apisecurity.UserGroupRelation{},
			RemoveRelations: &apisecurity.UserGroupRelation{},
		}

		resp := groupTest.svr.UpdateGroups(reqCtx, []*apisecurity.ModifyUserGroup{req})
		assert.True(t, resp.Responses[0].GetCode().Value == v1.NoNeedUpdate, resp.Responses[0].Info.GetValue())
	})

}

func Test_server_GetGroupToken(t *testing.T) {
	t.Run("主账户去查询owner为自己的用户组", func(t *testing.T) {
		groupTest := newGroupTest(t)

		t.Cleanup(func() {
			groupTest.Clean()
		})
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.users[0].Token)

		resp := groupTest.svr.GetGroupToken(reqCtx, &apisecurity.UserGroup{
			Id: utils.NewStringValue(groupTest.groups[1].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.ExecuteSuccess, resp.Info.GetValue())
	})

	t.Run("主账户去查询owner不是自己的用户组", func(t *testing.T) {
		groupTest := newGroupTest(t)

		t.Cleanup(func() {
			groupTest.Clean()
		})
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.ownerTwo.Token)

		resp := groupTest.svr.GetGroupToken(reqCtx, &apisecurity.UserGroup{
			Id: utils.NewStringValue(groupTest.groups[1].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.NotAllowedAccess, resp.Info.GetValue())
	})

	t.Run("子账户去查询自己所在的用户组", func(t *testing.T) {
		groupTest := newGroupTest(t)

		t.Cleanup(func() {
			groupTest.Clean()
		})
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.users[1].Token)

		resp := groupTest.svr.GetGroupToken(reqCtx, &apisecurity.UserGroup{
			Id: utils.NewStringValue(groupTest.groups[1].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.ExecuteSuccess, resp.Info.GetValue())
	})

	t.Run("子账户去查询自己不在的用户组", func(t *testing.T) {
		groupTest := newGroupTest(t)

		t.Cleanup(func() {
			groupTest.Clean()
		})
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.users[1].Token)

		resp := groupTest.svr.GetGroupToken(reqCtx, &apisecurity.UserGroup{
			Id: utils.NewStringValue(groupTest.groups[2].ID),
		})

		assert.True(t, resp.GetCode().Value == v1.NotAllowedAccess, resp.Info.GetValue())
	})
}

func Test_server_DeleteGroup(t *testing.T) {

	t.Run("正常删除用户组", func(t *testing.T) {
		groupTest := newGroupTest(t)

		t.Cleanup(func() {
			groupTest.Clean()
		})
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.ownerOne.Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).AnyTimes().Return(groupTest.groups[0], nil)
		groupTest.storage.EXPECT().DeleteGroup(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

		batchResp := groupTest.svr.DeleteGroups(reqCtx, []*apisecurity.UserGroup{
			{
				Id: utils.NewStringValue(groupTest.groups[0].ID),
			},
		})

		assert.True(t, batchResp.GetCode().Value == v1.ExecuteSuccess, batchResp.Info.GetValue())
	})

	t.Run("删除用户组-用户组不存在", func(t *testing.T) {
		groupTest := newGroupTest(t)

		t.Cleanup(func() {
			groupTest.Clean()
		})
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.ownerOne.Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).AnyTimes().Return(nil, nil)
		groupTest.storage.EXPECT().DeleteGroup(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

		batchResp := groupTest.svr.DeleteGroups(reqCtx, []*apisecurity.UserGroup{
			{
				Id: utils.NewStringValue(groupTest.groups[0].ID),
			},
		})

		assert.True(t, batchResp.GetCode().Value == v1.ExecuteSuccess, batchResp.Info.GetValue())
	})

	t.Run("删除用户组-不是用户组的owner", func(t *testing.T) {
		groupTest := newGroupTest(t)

		t.Cleanup(func() {
			groupTest.Clean()
		})
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.ownerTwo.Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).AnyTimes().Return(groupTest.groups[0], nil)
		groupTest.storage.EXPECT().DeleteGroup(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

		batchResp := groupTest.svr.DeleteGroups(reqCtx, []*apisecurity.UserGroup{
			{
				Id: utils.NewStringValue(groupTest.groups[0].ID),
			},
		})

		assert.True(t, batchResp.Responses[0].GetCode().Value == v1.NotAllowedAccess, batchResp.Responses[0].Info.GetValue())
	})

	t.Run("删除用户组-非owner角色", func(t *testing.T) {
		groupTest := newGroupTest(t)

		t.Cleanup(func() {
			groupTest.Clean()
		})
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.users[1].Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).AnyTimes().Return(groupTest.groups[0], nil)
		groupTest.storage.EXPECT().DeleteGroup(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

		batchResp := groupTest.svr.DeleteGroups(reqCtx, []*apisecurity.UserGroup{
			{
				Id: utils.NewStringValue(groupTest.groups[0].ID),
			},
		})

		assert.True(t, batchResp.Responses[0].GetCode().Value == v1.OperationRoleException, batchResp.Responses[0].Info.GetValue())
	})

}

func Test_server_UpdateGroupToken(t *testing.T) {
	t.Run("正常更新用户组Token的Enable状态", func(t *testing.T) {
		groupTest := newGroupTest(t)
		t.Cleanup(func() {
			groupTest.Clean()
		})
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.ownerOne.Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).AnyTimes().Return(groupTest.groups[0], nil)
		groupTest.storage.EXPECT().UpdateGroup(gomock.Any()).AnyTimes().Return(nil)

		batchResp := groupTest.svr.EnableGroupToken(reqCtx, &apisecurity.UserGroup{
			Id: utils.NewStringValue(groupTest.groups[2].ID),
		})

		assert.True(t, batchResp.GetCode().Value == v1.ExecuteSuccess, batchResp.Info.GetValue())
	})

	t.Run("非Owner角色更新用户组Token的Enable状态", func(t *testing.T) {
		groupTest := newGroupTest(t)
		t.Cleanup(func() {
			groupTest.Clean()
		})
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.users[2].Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).AnyTimes().Return(groupTest.groups[0], nil)

		batchResp := groupTest.svr.EnableGroupToken(reqCtx, &apisecurity.UserGroup{
			Id: utils.NewStringValue(groupTest.groups[2].ID),
		})

		assert.True(t, batchResp.Code.Value == v1.OperationRoleException, batchResp.Info.GetValue())
	})

	t.Run("更新用户组Token的Enable状态-非group的owner", func(t *testing.T) {
		groupTest := newGroupTest(t)
		t.Cleanup(func() {
			groupTest.Clean()
		})
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.ownerTwo.Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).AnyTimes().Return(groupTest.groups[0], nil)

		batchResp := groupTest.svr.EnableGroupToken(reqCtx, &apisecurity.UserGroup{
			Id: utils.NewStringValue(groupTest.groups[2].ID),
		})

		assert.True(t, batchResp.Code.Value == v1.NotAllowedAccess, batchResp.Info.GetValue())
	})
}

func Test_server_RefreshGroupToken(t *testing.T) {
	t.Run("正常更新用户组Token的Enable状态", func(t *testing.T) {
		groupTest := newGroupTest(t)
		t.Cleanup(func() {
			groupTest.Clean()
		})
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.ownerOne.Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).Return(groupTest.groups[0], nil)
		groupTest.storage.EXPECT().UpdateGroup(gomock.Any()).Return(nil)

		batchResp := groupTest.svr.ResetGroupToken(reqCtx, &apisecurity.UserGroup{
			Id: utils.NewStringValue(groupTest.groups[2].ID),
		})

		assert.True(t, batchResp.GetCode().Value == v1.ExecuteSuccess, batchResp.Info.GetValue())
	})

	t.Run("非Owner角色更新用户组Token的Enable状态", func(t *testing.T) {
		groupTest := newGroupTest(t)
		t.Cleanup(func() {
			groupTest.Clean()
		})
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.users[2].Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).AnyTimes().Return(groupTest.groups[0], nil)

		batchResp := groupTest.svr.ResetGroupToken(reqCtx, &apisecurity.UserGroup{
			Id: utils.NewStringValue(groupTest.groups[2].ID),
		})

		assert.True(t, batchResp.Code.Value == v1.OperationRoleException, batchResp.Info.GetValue())
	})

	t.Run("更新用户组Token的Enable状态-非group的owner", func(t *testing.T) {
		groupTest := newGroupTest(t)
		t.Cleanup(func() {
			groupTest.Clean()
		})
		reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groupTest.ownerTwo.Token)

		groupTest.storage.EXPECT().GetGroup(gomock.Any()).Return(groupTest.groups[0], nil).AnyTimes()

		batchResp := groupTest.svr.ResetGroupToken(reqCtx, &apisecurity.UserGroup{
			Id: utils.NewStringValue(groupTest.groups[2].ID),
		})

		assert.True(t, batchResp.Code.Value == v1.NotAllowedAccess, batchResp.Info.GetValue())
	})
}

func Test_AuthServer_NormalOperateUserGroup(t *testing.T) {
	suit := &AuthTestSuit{}
	if err := suit.Initialize(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		suit.cleanAllAuthStrategy()
		suit.cleanAllUser()
		suit.cleanAllUserGroup()
		suit.Destroy()
	})

	users := createApiMockUser(10, "test")
	for i := range users {
		users[i].Id = utils.NewStringValue(utils.NewUUID())
	}

	groups := createMockApiUserGroup([]*apisecurity.User{users[0]})

	t.Run("正常创建用户组", func(t *testing.T) {
		bresp := suit.UserServer().CreateUsers(suit.DefaultCtx, users)
		if !respSuccess(bresp) {
			t.Fatal(bresp.GetInfo().GetValue())
		}

		_ = suit.CacheMgr().TestUpdate()

		resp := suit.UserServer().CreateGroup(suit.DefaultCtx, groups[0])

		if !respSuccess(resp) {
			t.Fatal(resp.GetInfo().GetValue())
		}

		groups[0].Id = utils.NewStringValue(resp.GetUserGroup().Id.Value)
	})

	t.Run("正常更新用户组", func(t *testing.T) {

		time.Sleep(time.Second)

		req := []*apisecurity.ModifyUserGroup{
			{
				Id:   utils.NewStringValue(groups[0].GetId().GetValue()),
				Name: utils.NewStringValue(groups[0].GetName().GetValue()),
				Comment: &wrapperspb.StringValue{
					Value: "update user group",
				},
				AddRelations: &apisecurity.UserGroupRelation{
					Users: users[3:],
				},
			},
		}

		resp := suit.UserServer().UpdateGroups(suit.DefaultCtx, req)
		if !respSuccess(resp) {
			t.Fatal(resp.GetInfo().GetValue())
		}

		_ = suit.CacheMgr().TestUpdate()

		qresp := suit.UserServer().GetGroup(suit.DefaultCtx, groups[0])

		if !respSuccess(resp) {
			t.Fatal(resp.GetInfo().GetValue())
		}

		assert.Equal(t, req[0].GetComment().GetValue(), qresp.GetUserGroup().GetComment().GetValue())
		assert.Equal(t, len(users[3:])+1, len(qresp.GetUserGroup().GetRelation().GetUsers()))
	})

	t.Run("正常更新用户组Token", func(t *testing.T) {
		resp := suit.UserServer().ResetGroupToken(suit.DefaultCtx, groups[0])

		if !respSuccess(resp) {
			t.Fatal(resp.GetInfo().GetValue())
		}

		_ = suit.CacheMgr().TestUpdate()

		qresp := suit.UserServer().GetGroupToken(suit.DefaultCtx, groups[0])
		if !respSuccess(qresp) {
			t.Fatal(resp.GetInfo().GetValue())
		}
		assert.Equal(t, resp.GetUserGroup().GetAuthToken().GetValue(), qresp.GetUserGroup().GetAuthToken().GetValue())
	})

	t.Run("正常查询某个用户组下的用户列表", func(t *testing.T) {
		qresp := suit.UserServer().GetUsers(suit.DefaultCtx, map[string]string{
			"group_id": groups[0].GetId().GetValue(),
		})

		if !respSuccess(qresp) {
			t.Fatal(qresp.GetInfo().GetValue())
		}

		assert.Equal(t, 8, len(qresp.GetUsers()))

		expectUsers := []string{users[0].Id.Value}
		for _, u := range users[3:] {
			expectUsers = append(expectUsers, u.Id.Value)
		}

		retUsers := []string{}
		for i := range qresp.GetUsers() {
			retUsers = append(retUsers, qresp.GetUsers()[i].Id.Value)
		}
		assert.ElementsMatch(t, expectUsers, retUsers)
	})

	t.Run("正常查询用户组列表", func(t *testing.T) {
		qresp := suit.UserServer().GetGroups(suit.DefaultCtx, map[string]string{})

		if !respSuccess(qresp) {
			t.Fatal(qresp.GetInfo().GetValue())
		}

		assert.True(t, len(qresp.GetUserGroups()) == 1)
		assert.Equal(t, groups[0].GetId().GetValue(), qresp.GetUserGroups()[0].Id.GetValue())
	})

	t.Run("查询某个用户所在的所有分组", func(t *testing.T) {
		qresp := suit.UserServer().GetGroups(suit.DefaultCtx, map[string]string{
			"user_id": users[0].GetId().GetValue(),
		})

		if !respSuccess(qresp) {
			t.Fatal(qresp.GetInfo().GetValue())
		}

		assert.True(t, len(qresp.GetUserGroups()) == 1)
		assert.Equal(t, groups[0].GetId().GetValue(), qresp.GetUserGroups()[0].Id.GetValue())
	})

	t.Run("正常删除用户组", func(t *testing.T) {
		resp := suit.UserServer().DeleteGroups(suit.DefaultCtx, groups)

		if !respSuccess(resp) {
			t.Fatal(resp.GetInfo().GetValue())
		}

		qresp := suit.UserServer().GetGroup(suit.DefaultCtx, groups[0])

		if respSuccess(qresp) {
			t.Fatal(qresp.GetInfo().GetValue())
		}

		assert.Equal(t, v1.NotFoundUserGroup, qresp.GetCode().GetValue())
	})
}
