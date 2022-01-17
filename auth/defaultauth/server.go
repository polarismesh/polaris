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
	"errors"

	"github.com/polarismesh/polaris-server/cache"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/plugin"
	"github.com/polarismesh/polaris-server/store"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type server struct {
	storage  store.Store
	history  plugin.History
	cacheMgn *cache.NamingCache
}

// Login 登陆动作
func (svr *server) Login(req *api.LoginRequest) *api.Response {
	username := req.GetName().GetValue()
	owner := req.GetOwner().GetValue()
	if owner == "" {
		owner = username
	}
	user := svr.cacheMgn.User().GetUserByName(username, owner)

	if user == nil {
		return api.NewResponse(api.NotFoundUser)
	}

	// TODO AES 解密操作，在进行密码比对计算

	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.GetPassword().GetValue()))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return api.NewResponseWithMsg(api.NotAllowedAccess, ErrorWrongUsernameOrPassword.Error())
		}
		return api.NewResponseWithMsg(api.ExecuteException, ErrorWrongUsernameOrPassword.Error())
	}

	return api.NewLoginResponse(api.ExecuteSuccess, &api.LoginResponse{
		Token: utils.NewStringValue(user.Token),
		Name:  utils.NewStringValue(user.Name),
		Role:  utils.NewStringValue(model.UserRoleNames[user.Type]),
	})
}

// RecordHistory server对外提供history插件的简单封装
func (svr *server) RecordHistory(entry *model.RecordEntry) {
	// 如果插件没有初始化，那么不记录history
	if svr.history == nil {
		return
	}
	// 如果数据为空，则不需要打印了
	if entry == nil {
		return
	}

	// 调用插件记录history
	svr.history.Record(entry)
}

// AfterResourceOperation 对于资源的添加删除操作，需要执行后置逻辑
// 所有子用户或者用户分组，都默认获得对所创建的资源的写权限
func (svr *server) AfterResourceOperation(afterCtx *model.AcquireContext) {
	operatorId := afterCtx.GetAttachment()[model.OperatorIDKey].(string)
	principalType := afterCtx.GetAttachment()[model.OperatorPrincipalType].(model.PrincipalType)

	// 获取该用户的默认策略信息
	name := model.BuildDefaultStrategyName(operatorId, principalType)
	ownerId := afterCtx.GetAttachment()[model.OperatorOwnerKey].(string)
	// Get the default policy rules
	strategy, err := svr.storage.GetStrategyDetailByName(ownerId, name)
	if err != nil {
		log.GetAuthLogger().Error("[Auth][Server] get default strategy",
			zap.String("owner", ownerId), zap.String("name", name), zap.Error(err))
		return
	}

	strategyResource := make([]model.StrategyResource, 0)
	resources := afterCtx.GetAttachment()[model.ResourceAttachmentKey].(map[api.ResourceType][]model.ResourceEntry)

	for rType, rIds := range resources {
		for i := range rIds {
			id := rIds[i]
			strategyResource = append(strategyResource, model.StrategyResource{
				StrategyID: strategy.ID,
				ResType:    int32(rType),
				ResID:      id.ID,
			})
		}
	}

	if afterCtx.GetOperation() == model.Create {
		// 如果是写操作，那么采用松添加操作进行新增资源的添加操作(仅忽略主键冲突的错误)
		err = svr.storage.LooseAddStrategyResources(strategyResource)
		if err != nil {
			log.GetAuthLogger().Error("[Auth][Server] update default strategy resource",
				zap.String("owner", ownerId), zap.String("name", name), zap.Error(err))
			return
		}
	}
	if afterCtx.GetOperation() == model.Delete {
		err = svr.storage.RemoveStrategyResources(strategyResource)
		if err != nil {
			log.GetAuthLogger().Error("[Auth][Server] remove default strategy resource",
				zap.String("owner", ownerId), zap.String("name", name), zap.Error(err))
			return
		}
	}
}
