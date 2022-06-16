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
	cacheMgn *cache.CacheManager
	authMgn  *defaultAuthChecker
}

// initialize
func (svr *server) initialize() error {
	// 获取History插件，注意：插件的配置在bootstrap已经设置好
	svr.history = plugin.GetHistory()
	if svr.history == nil {
		log.AuthScope().Warnf("Not Found History Log Plugin")
	}

	return nil
}

// Login 登陆动作
func (svr *server) Login(req *api.LoginRequest) *api.Response {
	username := req.GetName().GetValue()
	ownerName := req.GetOwner().GetValue()
	if ownerName == "" {
		ownerName = username
	}
	user := svr.cacheMgn.User().GetUserByName(username, ownerName)
	if user == nil {
		return api.NewResponse(api.NotFoundUser)
	}

	// TODO AES 解密操作，在进行密码比对计算
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.GetPassword().GetValue()))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return api.NewResponseWithMsg(api.NotAllowedAccess, model.ErrorWrongUsernameOrPassword.Error())
		}
		return api.NewResponseWithMsg(api.ExecuteException, model.ErrorWrongUsernameOrPassword.Error())
	}

	return api.NewLoginResponse(api.ExecuteSuccess, &api.LoginResponse{
		UserId:  utils.NewStringValue(user.ID),
		OwnerId: utils.NewStringValue(user.Owner),
		Token:   utils.NewStringValue(user.Token),
		Name:    utils.NewStringValue(user.Name),
		Role:    utils.NewStringValue(model.UserRoleNames[user.Type]),
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
func (svr *server) AfterResourceOperation(afterCtx *model.AcquireContext) error {
	if !svr.authMgn.IsOpenAuth() || afterCtx.GetOperation() == model.Read {
		return nil
	}

	// 如果客户端鉴权没有开启，且请求来自客户端，忽略
	if afterCtx.IsFromClient() && !svr.authMgn.IsOpenClientAuth() {
		return nil
	}
	// 如果控制台鉴权没有开启，且请求来自控制台，忽略
	if afterCtx.IsFromConsole() && !svr.authMgn.IsOpenConsoleAuth() {
		return nil
	}

	// 如果 token 信息为空，则代表当前创建的资源，任何人都可以进行操作，不做资源的后置逻辑处理
	if IsEmptyOperator(afterCtx.GetAttachment(model.TokenDetailInfoKey).(OperatorInfo)) {
		return nil
	}

	addUserIds := afterCtx.GetAttachment(model.LinkUsersKey).([]string)
	addGroupIds := afterCtx.GetAttachment(model.LinkGroupsKey).([]string)
	removeUserIds := afterCtx.GetAttachment(model.RemoveLinkUsersKey).([]string)
	removeGroupIds := afterCtx.GetAttachment(model.RemoveLinkGroupsKey).([]string)

	// 只有在创建一个资源的时候，才需要把当前的创建者一并加到里面去
	if afterCtx.GetOperation() == model.Create {
		tokenInfo := afterCtx.GetAttachment(model.TokenDetailInfoKey).(OperatorInfo)
		if tokenInfo.IsUserToken {
			addUserIds = append(addUserIds, tokenInfo.OperatorID)
		} else {
			addGroupIds = append(addGroupIds, tokenInfo.OperatorID)
		}
	}

	log.AuthScope().Info("[Auth][Server] add resource to principal default strategy",
		zap.Any("resource", afterCtx.GetAttachment(model.ResourceAttachmentKey)),
		zap.Any("add_user", addUserIds), zap.Any("add_group", addGroupIds), zap.Any("remove_user", removeUserIds),
		zap.Any("remove_group", removeGroupIds),
	)

	// 添加某些用户、用户组与资源的默认授权关系
	if err := svr.handleUserStrategy(addUserIds, afterCtx, false); err != nil {
		log.AuthScope().Error("[Auth][Server] add user link resource", zap.Error(err))
		return err
	}
	if err := svr.handleGroupStrategy(addGroupIds, afterCtx, false); err != nil {
		log.AuthScope().Error("[Auth][Server] add group link resource", zap.Error(err))
		return err
	}

	// 清理某些用户、用户组与资源的默认授权关系
	if err := svr.handleUserStrategy(removeUserIds, afterCtx, true); err != nil {
		log.AuthScope().Error("[Auth][Server] remove user link resource", zap.Error(err))
		return err
	}
	if err := svr.handleGroupStrategy(removeGroupIds, afterCtx, true); err != nil {
		log.AuthScope().Error("[Auth][Server] remove group link resource", zap.Error(err))
		return err
	}

	return nil
}

// handleUserStrategy
func (svr *server) handleUserStrategy(userIds []string, afterCtx *model.AcquireContext, isRemove bool) error {
	for index := range utils.StringSliceDeDuplication(userIds) {
		userId := userIds[index]
		user := svr.cacheMgn.User().GetUserByID(userId)
		if user == nil {
			return errors.New("not found target user")
		}

		ownerId := user.Owner
		if ownerId == "" {
			ownerId = user.ID
		}
		if err := svr.handlerModifyDefaultStrategy(userId, ownerId, model.PrincipalUser,
			afterCtx, isRemove); err != nil {
			return err
		}
	}

	return nil
}

// handleGroupStrategy
func (svr *server) handleGroupStrategy(groupIds []string, afterCtx *model.AcquireContext, isRemove bool) error {
	for index := range utils.StringSliceDeDuplication(groupIds) {
		groupId := groupIds[index]
		group := svr.cacheMgn.User().GetGroup(groupId)
		if group == nil {
			return errors.New("not found target group")
		}

		ownerId := group.Owner
		if err := svr.handlerModifyDefaultStrategy(groupId, ownerId, model.PrincipalGroup,
			afterCtx, isRemove); err != nil {
			return err
		}
	}

	return nil
}

// handlerModifyDefaultStrategy 处理默认策略的修改
// case 1. 如果默认策略是全部放通
func (svr *server) handlerModifyDefaultStrategy(id, ownerId string, uType model.PrincipalType,
	afterCtx *model.AcquireContext, cleanRealtion bool) error {
	// Get the default policy rules
	strategy, err := svr.storage.GetDefaultStrategyDetailByPrincipal(id, uType)
	if err != nil {
		log.AuthScope().Error("[Auth][Server] get default strategy",
			zap.String("owner", ownerId), zap.String("id", id), zap.Error(err))
		return err
	}

	if strategy == nil {
		return errors.New("not found default strategy rule")
	}

	strategyResource := make([]model.StrategyResource, 0)
	resources := afterCtx.GetAttachment(model.ResourceAttachmentKey).(map[api.ResourceType][]model.ResourceEntry)

	strategyId := strategy.ID

	// 资源删除时，清理该资源与所有策略的关联关系
	if afterCtx.GetOperation() == model.Delete {
		strategyId = ""
	}

	for rType, rIds := range resources {
		for i := range rIds {
			id := rIds[i]
			strategyResource = append(strategyResource, model.StrategyResource{
				StrategyID: strategyId,
				ResType:    int32(rType),
				ResID:      id.ID,
			})
		}
	}

	if afterCtx.GetOperation() == model.Delete || cleanRealtion {
		err = svr.storage.RemoveStrategyResources(strategyResource)
		if err != nil {
			log.AuthScope().Error("[Auth][Server] remove default strategy resource",
				zap.String("owner", ownerId), zap.String("id", id),
				zap.String("type", model.PrincipalNames[uType]), zap.Error(err))
			return err
		}

		return nil
	} else {
		// 如果是写操作，那么采用松添加操作进行新增资源的添加操作(仅忽略主键冲突的错误)
		err = svr.storage.LooseAddStrategyResources(strategyResource)
		if err != nil {
			log.AuthScope().Error("[Auth][Server] update default strategy resource",
				zap.String("owner", ownerId), zap.String("id", id), zap.String("id", id),
				zap.String("type", model.PrincipalNames[uType]), zap.Error(err))
			return err
		}
	}

	return nil
}

func checkHasPassAll(rule *model.StrategyDetail) bool {
	for i := range rule.Resources {
		if rule.Resources[i].ResID == "*" {
			return true
		}
	}

	return false
}
