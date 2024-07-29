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

package policy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/polarismesh/polaris/auth"
	cachetypes "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/common/model"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
	"github.com/polarismesh/polaris/store"
)

// AuthConfig 鉴权配置
type AuthConfig struct {
	// ConsoleOpen 控制台是否开启鉴权
	ConsoleOpen bool `json:"consoleOpen" xml:"consoleOpen"`
	// ClientOpen 是否开启客户端接口鉴权
	ClientOpen bool `json:"clientOpen" xml:"clientOpen"`
	// Strict 是否启用鉴权的严格模式，即对于没有任何鉴权策略的资源，也必须带上正确的token才能操作, 默认关闭
	// Deprecated
	Strict bool `json:"strict"`
	// ConsoleStrict 是否启用鉴权的严格模式，即对于没有任何鉴权策略的资源，也必须带上正确的token才能操作, 默认关闭
	ConsoleStrict bool `json:"consoleStrict"`
	// ClientStrict 是否启用鉴权的严格模式，即对于没有任何鉴权策略的资源，也必须带上正确的token才能操作, 默认关闭
	ClientStrict bool `json:"clientStrict"`
	// CredibleHeaders 可信请求 Header
	CredibleHeaders map[string]string
	// OpenPrincipalDefaultPolicy 是否开启 principal 默认策略
	OpenPrincipalDefaultPolicy bool `json:"openPrincipalDefaultPolicy"`
}

// DefaultAuthConfig 返回一个默认的鉴权配置
func DefaultAuthConfig() *AuthConfig {
	return &AuthConfig{
		// 针对控制台接口，默认开启鉴权操作
		ConsoleOpen: true,
		// 这里默认开启 OpenAPI 的强 Token 检查模式
		ConsoleStrict: false,
		// 针对客户端接口，默认不开启鉴权操作
		ClientOpen: false,
		// 客户端接口默认不开启 token 强检查模式
		ClientStrict: false,
	}
}

type Server struct {
	options  *AuthConfig
	storage  store.Store
	cacheMgr cachetypes.CacheManager
	checker  auth.AuthChecker
	userSvr  auth.UserServer
}

// PolicyHelper implements auth.StrategyServer.
func (svr *Server) PolicyHelper() auth.PolicyHelper {
	return &DefaultPolicyHelper{
		options:  svr.options,
		storage:  svr.storage,
		cacheMgr: svr.cacheMgr,
		checker:  svr.checker,
	}
}

// initialize
func (svr *Server) Initialize(options *auth.Config, storage store.Store, cacheMgr cachetypes.CacheManager, userSvr auth.UserServer) error {
	svr.cacheMgr = cacheMgr
	svr.userSvr = userSvr
	svr.storage = storage
	if err := svr.ParseOptions(options); err != nil {
		return err
	}

	_ = cacheMgr.OpenResourceCache(cachetypes.ConfigEntry{
		Name: cachetypes.StrategyRuleName,
	})

	checker := &DefaultAuthChecker{
		policyMgr: svr,
	}
	checker.Initialize(svr.options, svr.storage, cacheMgr, userSvr)
	svr.checker = checker
	return nil
}

func (svr *Server) GetOptions() *AuthConfig {
	return svr.options
}

func (svr *Server) ParseOptions(options *auth.Config) error {
	// 新版本鉴权策略配置均从auth.Option中迁移至auth.user.option及auth.strategy.option中
	var (
		strategyContentBytes []byte
		authContentBytes     []byte
		err                  error
	)

	cfg := DefaultAuthConfig()

	// 设置了 auth.strategy.option，将不会继续读取 auth.option
	if len(options.Strategy.Option) > 0 {
		// 判断auth.option是否还有值，有则不兼容
		if len(options.Option) > 0 {
			log.Warn("auth.user.option or auth.strategy.option has set, auth.option will ignore")
		}
		strategyContentBytes, err = json.Marshal(options.Strategy.Option)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(strategyContentBytes, cfg); err != nil {
			return err
		}
	} else {
		log.Warn("[Auth][Checker] auth.option has deprecated, use auth.user.option and auth.strategy.option instead.")
		authContentBytes, err = json.Marshal(options.Option)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(authContentBytes, cfg); err != nil {
			return err
		}
	}
	// 兼容原本老的配置逻辑
	if cfg.Strict {
		cfg.ConsoleOpen = cfg.Strict
	}
	svr.options = cfg
	return nil
}

func (svr *Server) Name() string {
	return auth.DefaultPolicyPluginName
}

func (svr *Server) GetAuthChecker() auth.AuthChecker {
	return svr.checker
}

// RecordHistory Server对外提供history插件的简单封装
func (svr *Server) RecordHistory(entry *model.RecordEntry) {
	plugin.GetHistory().Record(entry)
}

func (svr *Server) isOpenAuth() bool {
	return svr.checker.IsOpenClientAuth() || svr.checker.IsOpenConsoleAuth()
}

// AfterResourceOperation 对于资源的添加删除操作，需要执行后置逻辑
// 所有子用户或者用户分组，都默认获得对所创建的资源的写权限
func (svr *Server) AfterResourceOperation(afterCtx *authcommon.AcquireContext) error {
	if !svr.isOpenAuth() || afterCtx.GetOperation() == authcommon.Read {
		return nil
	}

	// 如果客户端鉴权没有开启，且请求来自客户端，忽略
	if afterCtx.IsFromClient() && !svr.checker.IsOpenClientAuth() {
		return nil
	}
	// 如果控制台鉴权没有开启，且请求来自控制台，忽略
	if afterCtx.IsFromConsole() && !svr.checker.IsOpenConsoleAuth() {
		return nil
	}

	attachVal, ok := afterCtx.GetAttachment(authcommon.TokenDetailInfoKey)
	if !ok {
		return nil
	}
	tokenInfo, ok := attachVal.(auth.OperatorInfo)
	if !ok {
		return nil
	}

	// 如果 token 信息为空，则代表当前创建的资源，任何人都可以进行操作，不做资源的后置逻辑处理
	if auth.IsEmptyOperator(tokenInfo) {
		return nil
	}

	addUserIds := afterCtx.GetAttachments()[authcommon.LinkUsersKey].([]string)
	addGroupIds := afterCtx.GetAttachments()[authcommon.LinkGroupsKey].([]string)
	removeUserIds := afterCtx.GetAttachments()[authcommon.RemoveLinkUsersKey].([]string)
	removeGroupIds := afterCtx.GetAttachments()[authcommon.RemoveLinkGroupsKey].([]string)

	// 只有在创建一个资源的时候，才需要把当前的创建者一并加到里面去
	if afterCtx.GetOperation() == authcommon.Create {
		if tokenInfo.IsUserToken {
			addUserIds = append(addUserIds, tokenInfo.OperatorID)
		} else {
			addGroupIds = append(addGroupIds, tokenInfo.OperatorID)
		}
	}

	log.Info("[Auth][Server] add resource to principal default strategy",
		zap.Any("resource", afterCtx.GetAttachments()[authcommon.ResourceAttachmentKey]),
		zap.Any("add_user", addUserIds),
		zap.Any("add_group", addGroupIds), zap.Any("remove_user", removeUserIds),
		zap.Any("remove_group", removeGroupIds),
	)

	// 添加某些用户、用户组与资源的默认授权关系
	if err := svr.handleChangeUserPolicy(addUserIds, afterCtx, false); err != nil {
		log.Error("[Auth][Server] add user link resource", zap.Error(err))
		return err
	}
	if err := svr.handleChangeUserGroupPolicy(addGroupIds, afterCtx, false); err != nil {
		log.Error("[Auth][Server] add group link resource", zap.Error(err))
		return err
	}

	// 清理某些用户、用户组与资源的默认授权关系
	if err := svr.handleChangeUserPolicy(removeUserIds, afterCtx, true); err != nil {
		log.Error("[Auth][Server] remove user link resource", zap.Error(err))
		return err
	}
	if err := svr.handleChangeUserGroupPolicy(removeGroupIds, afterCtx, true); err != nil {
		log.Error("[Auth][Server] remove group link resource", zap.Error(err))
		return err
	}

	return nil
}

// handleUserStrategy
func (svr *Server) handleChangeUserPolicy(userIds []string, afterCtx *authcommon.AcquireContext, isRemove bool) error {
	for index := range utils.StringSliceDeDuplication(userIds) {
		userId := userIds[index]
		user := svr.userSvr.GetUserHelper().GetUser(context.TODO(), &apisecurity.User{
			Id: wrapperspb.String(userId),
		})
		if user == nil {
			return errors.New("not found target user")
		}

		ownerId := user.GetOwner().GetValue()
		if ownerId == "" {
			ownerId = user.GetId().GetValue()
		}
		if err := svr.changePrincipalPolicies(userId, ownerId, authcommon.PrincipalUser,
			afterCtx, isRemove); err != nil {
			return err
		}
	}
	return nil
}

// handleGroupStrategy
func (svr *Server) handleChangeUserGroupPolicy(groupIds []string, afterCtx *authcommon.AcquireContext, isRemove bool) error {
	for index := range utils.StringSliceDeDuplication(groupIds) {
		groupId := groupIds[index]
		group := svr.userSvr.GetUserHelper().GetGroup(context.TODO(), &apisecurity.UserGroup{
			Id: wrapperspb.String(groupId),
		})
		if group == nil {
			return errors.New("not found target group")
		}
		ownerId := group.GetOwner().GetValue()
		if err := svr.changePrincipalPolicies(groupId, ownerId, authcommon.PrincipalGroup,
			afterCtx, isRemove); err != nil {
			return err
		}
	}

	return nil
}

// changePrincipalPolicies 处理默认策略的修改
// case 1. 如果默认策略是全部放通
func (svr *Server) changePrincipalPolicies(id, ownerId string, uType authcommon.PrincipalType,
	afterCtx *authcommon.AcquireContext, cleanRealtion bool) error {
	// Get the default policy rules
	strategy, err := svr.storage.GetDefaultStrategyDetailByPrincipal(id, uType)
	if err != nil {
		log.Error("[Auth][Server] get default strategy",
			zap.String("owner", ownerId), zap.String("id", id), zap.Error(err))
		return err
	}
	if strategy == nil {
		return errors.New("not found default strategy rule")
	}

	var (
		strategyResource = make([]authcommon.StrategyResource, 0)
		strategyId       = strategy.ID
	)
	attachVal, ok := afterCtx.GetAttachment(authcommon.ResourceAttachmentKey)
	if !ok {
		return nil
	}
	resources, ok := attachVal.(map[apisecurity.ResourceType][]authcommon.ResourceEntry)
	if !ok {
		return nil
	}
	// 资源删除时，清理该资源与所有策略的关联关系
	if afterCtx.GetOperation() == authcommon.Delete {
		strategyId = ""
	}

	for rType, rIds := range resources {
		for i := range rIds {
			id := rIds[i]
			strategyResource = append(strategyResource, authcommon.StrategyResource{
				StrategyID: strategyId,
				ResType:    int32(rType),
				ResID:      id.ID,
			})
		}
	}

	entry := &model.RecordEntry{
		ResourceType: model.RAuthStrategy,
		ResourceName: fmt.Sprintf("%s(%s)", strategy.Name, strategy.ID),
		Operator:     utils.ParseOperator(afterCtx.GetRequestContext()),
		Detail:       utils.MustJson(strategyResource),
		HappenTime:   time.Now(),
	}

	if afterCtx.GetOperation() == authcommon.Delete || cleanRealtion {
		if err = svr.storage.RemoveStrategyResources(strategyResource); err != nil {
			log.Error("[Auth][Server] remove default strategy resource",
				zap.String("owner", ownerId), zap.String("id", id),
				zap.String("type", authcommon.PrincipalNames[uType]), zap.Error(err))
			return err
		}
		entry.OperationType = model.ODelete
		plugin.GetHistory().Record(entry)
		return nil
	}
	// 如果是写操作，那么采用松添加操作进行新增资源的添加操作(仅忽略主键冲突的错误)
	if err = svr.storage.LooseAddStrategyResources(strategyResource); err != nil {
		log.Error("[Auth][Server] update default strategy resource",
			zap.String("owner", ownerId), zap.String("id", id), zap.String("id", id),
			zap.String("type", authcommon.PrincipalNames[uType]), zap.Error(err))
		return err
	}
	entry.OperationType = model.OUpdate
	plugin.GetHistory().Record(entry)
	return nil
}
