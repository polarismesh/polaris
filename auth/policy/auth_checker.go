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
	"github.com/pkg/errors"
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/auth"
	cachetypes "github.com/polarismesh/polaris/cache/api"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

var (
	// ErrorNotAllowedAccess 鉴权失败
	ErrorNotAllowedAccess error = errors.New(api.Code2Info(api.NotAllowedAccess))
	// ErrorInvalidParameter 不合法的参数
	ErrorInvalidParameter error = errors.New(api.Code2Info(api.InvalidParameter))
	// ErrorNotPermission .
	ErrorNotPermission = errors.New("no permission")
)

// DefaultAuthChecker 北极星自带的默认鉴权中心
type DefaultAuthChecker struct {
	conf     *AuthConfig
	cacheMgr cachetypes.CacheManager
	userSvr  auth.UserServer
}

func (d *DefaultAuthChecker) SetCacheMgr(mgr cachetypes.CacheManager) {
	d.cacheMgr = mgr
}

// Initialize 执行初始化动作
func (d *DefaultAuthChecker) Initialize(conf *AuthConfig, s store.Store,
	cacheMgr cachetypes.CacheManager, userSvr auth.UserServer) error {
	d.conf = conf
	d.cacheMgr = cacheMgr
	d.userSvr = userSvr
	return nil
}

// Cache 获取缓存统一管理
func (d *DefaultAuthChecker) Cache() cachetypes.CacheManager {
	return d.cacheMgr
}

// IsOpenConsoleAuth 针对控制台是否开启了操作鉴权
func (d *DefaultAuthChecker) IsOpenConsoleAuth() bool {
	return d.conf.ConsoleOpen
}

// IsOpenClientAuth 针对客户端是否开启了操作鉴权
func (d *DefaultAuthChecker) IsOpenClientAuth() bool {
	return d.conf.ClientOpen
}

// IsOpenAuth 返回对于控制台/客户端任意其中的一个是否开启了操作鉴权
func (d *DefaultAuthChecker) IsOpenAuth() bool {
	return d.IsOpenConsoleAuth() || d.IsOpenClientAuth()
}

// AllowResourceOperate 是否允许资源的操作
func (d *DefaultAuthChecker) AllowResourceOperate(ctx *model.AcquireContext, opInfo *model.ResourceOpInfo) bool {
	// 如果鉴权能力没有开启，那就默认都可以进行编辑
	if !d.IsOpenAuth() {
		return true
	}
	attachVal, ok := ctx.GetAttachment(model.TokenDetailInfoKey)
	if !ok {
		// TODO need log
		return false
	}
	tokenInfo, ok := attachVal.(auth.OperatorInfo)

	principal := model.Principal{
		PrincipalID: tokenInfo.OperatorID,
		PrincipalRole: func() model.PrincipalType {
			if tokenInfo.IsUserToken {
				return model.PrincipalUser
			}
			return model.PrincipalGroup
		}(),
	}

	editable := d.cacheMgr.AuthStrategy().IsResourceEditable(principal, opInfo.ResourceType, opInfo.ResourceID)
	return editable
}

// CheckClientPermission 执行检查客户端动作判断是否有权限，并且对 RequestContext 注入操作者数据
func (d *DefaultAuthChecker) CheckClientPermission(preCtx *model.AcquireContext) (bool, error) {
	preCtx.SetFromClient()
	if !d.IsOpenClientAuth() {
		return true, nil
	}
	if d.IsOpenClientAuth() && !d.conf.ClientStrict {
		preCtx.SetAllowAnonymous(true)
	}
	return d.CheckPermission(preCtx)
}

// CheckConsolePermission 执行检查控制台动作判断是否有权限，并且对 RequestContext 注入操作者数据
func (d *DefaultAuthChecker) CheckConsolePermission(preCtx *model.AcquireContext) (bool, error) {
	preCtx.SetFromConsole()
	if !d.IsOpenConsoleAuth() {
		return true, nil
	}
	if d.IsOpenConsoleAuth() && !d.conf.ConsoleStrict {
		preCtx.SetAllowAnonymous(true)
	}
	if preCtx.GetModule() == model.MaintainModule {
		return d.checkMaintainPermission(preCtx)
	}
	return d.CheckPermission(preCtx)
}

// CheckMaintainPermission 执行检查运维动作判断是否有权限
func (d *DefaultAuthChecker) checkMaintainPermission(preCtx *model.AcquireContext) (bool, error) {
	if preCtx.GetOperation() == model.Read {
		return true, nil
	}

	attachVal, ok := preCtx.GetAttachment(model.TokenDetailInfoKey)
	if !ok {
		return false, model.ErrorTokenNotExist
	}
	tokenInfo, ok := attachVal.(auth.OperatorInfo)
	if !ok {
		return false, model.ErrorTokenNotExist
	}

	if tokenInfo.Disable {
		return false, model.ErrorTokenDisabled
	}
	if !tokenInfo.IsUserToken {
		return false, errors.New("only user role can access maintain API")
	}
	if tokenInfo.Role != model.OwnerUserRole {
		return false, errors.New("only owner account can access maintain API")
	}
	return true, nil
}

// CheckPermission 执行检查动作判断是否有权限
//
//	step 1. 判断是否开启了鉴权
//	step 2. 对token进行检查判断
//		case 1. 如果 token 被禁用
//				a. 读操作，直接放通
//				b. 写操作，快速失败
//	step 3. 拉取token对应的操作者相关信息，注入到请求上下文中
//	step 4. 进行权限检查
func (d *DefaultAuthChecker) CheckPermission(authCtx *model.AcquireContext) (bool, error) {
	if err := d.userSvr.CheckCredential(authCtx); err != nil {
		return false, err
	}

	attachVal, ok := authCtx.GetAttachment(model.TokenDetailInfoKey)
	if !ok {
		return false, model.ErrorTokenNotExist
	}
	operatorInfo, ok := attachVal.(auth.OperatorInfo)
	if !ok {
		return false, model.ErrorTokenNotExist
	}
	// 这里需要检查当 token 被禁止的情况，如果 token 被禁止，无论是否可以操作目标资源，都无法进行写操作
	if operatorInfo.Disable {
		return false, model.ErrorTokenDisabled
	}

	log.Debug("[Auth][Checker] check permission args", utils.RequestID(authCtx.GetRequestContext()),
		zap.String("method", authCtx.GetMethod()), zap.Any("resources", authCtx.GetAccessResources()))

	if pass, _ := d.doCheckPermission(authCtx); pass {
		return ok, nil
	}

	// 强制同步一次db中strategy数据到cache
	if err := d.cacheMgr.AuthStrategy().ForceSync(); err != nil {
		log.Error("[Auth][Checker] force sync strategy to cache failed",
			utils.RequestID(authCtx.GetRequestContext()), zap.Error(err))
		return false, err
	}
	return d.doCheckPermission(authCtx)
}

// doCheckPermission 执行权限检查
func (d *DefaultAuthChecker) doCheckPermission(authCtx *model.AcquireContext) (bool, error) {

	var checkNamespace, checkSvc, checkCfgGroup bool

	reqRes := authCtx.GetAccessResources()
	nsResEntries := reqRes[apisecurity.ResourceType_Namespaces]
	svcResEntries := reqRes[apisecurity.ResourceType_Services]
	cfgResEntries := reqRes[apisecurity.ResourceType_ConfigGroups]

	principleID, _ := authCtx.GetAttachments()[model.OperatorIDKey].(string)
	principleType, _ := authCtx.GetAttachments()[model.OperatorPrincipalType].(model.PrincipalType)
	p := model.Principal{
		PrincipalID:   principleID,
		PrincipalRole: principleType,
	}
	checkNamespace = d.checkAction(p, apisecurity.ResourceType_Namespaces, nsResEntries, authCtx)
	checkSvc = d.checkAction(p, apisecurity.ResourceType_Services, svcResEntries, authCtx)
	checkCfgGroup = d.checkAction(p, apisecurity.ResourceType_ConfigGroups, cfgResEntries, authCtx)

	checkAllResEntries := checkNamespace && checkSvc && checkCfgGroup

	var err error
	if !checkAllResEntries {
		err = ErrorNotPermission
	}
	return checkAllResEntries, err
}

// checkAction 检查操作是否和策略匹配
func (d *DefaultAuthChecker) checkAction(principal model.Principal,
	resType apisecurity.ResourceType, resources []model.ResourceEntry, ctx *model.AcquireContext) bool {
	// TODO 后续可针对读写操作进行鉴权, 并且可以针对具体的方法调用进行鉴权控制

	switch ctx.GetOperation() {
	case model.Read:
		return true
	default:
		for _, entry := range resources {
			if !d.cacheMgr.AuthStrategy().IsResourceEditable(principal, resType, entry.ID) {
				return false
			}
		}
	}
	return true
}
