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
	"strings"

	"github.com/pkg/errors"
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/auth"
	cachetypes "github.com/polarismesh/polaris/cache/api"
	api "github.com/polarismesh/polaris/common/api/v1"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
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
	conf      *AuthConfig
	cacheMgr  cachetypes.CacheManager
	userSvr   auth.UserServer
	policyMgr *Server
}

// Initialize 执行初始化动作
func (d *DefaultAuthChecker) Initialize(conf *AuthConfig, s store.Store,
	cacheMgr cachetypes.CacheManager, userSvr auth.UserServer) error {
	d.conf = conf
	d.cacheMgr = cacheMgr
	d.userSvr = userSvr
	return nil
}

func (d *DefaultAuthChecker) SetCacheMgr(mgr cachetypes.CacheManager) {
	d.cacheMgr = mgr
}

func (d *DefaultAuthChecker) GetConfig() *AuthConfig {
	return d.conf
}

func (d *DefaultAuthChecker) SetConfig(conf *AuthConfig) {
	d.conf = conf
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
func (d *DefaultAuthChecker) ResourcePredicate(ctx *authcommon.AcquireContext, res *authcommon.ResourceEntry) bool {
	// 如果是客户端请求，并且鉴权能力没有开启，那就默认都可以进行操作
	if ctx.IsFromClient() && !d.IsOpenClientAuth() {
		return true
	}
	// 如果是控制台请求，并且鉴权能力没有开启，那就默认都可以进行操作
	if ctx.IsFromConsole() && !d.IsOpenConsoleAuth() {
		return true
	}

	p, ok := ctx.GetAttachment(authcommon.PrincipalKey)
	if !ok {
		return false
	}
	policyCache := d.cacheMgr.AuthStrategy()

	principals := d.listAllPrincipals(p.(authcommon.Principal))
	for i := range principals {
		ret := policyCache.Hint(ctx.GetRequestContext(), principals[i], res)
		if ret != apisecurity.AuthAction_DENY {
			return true
		}
	}
	return false
}

// CheckClientPermission 执行检查客户端动作判断是否有权限，并且对 RequestContext 注入操作者数据
func (d *DefaultAuthChecker) CheckClientPermission(preCtx *authcommon.AcquireContext) (bool, error) {
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
func (d *DefaultAuthChecker) CheckConsolePermission(preCtx *authcommon.AcquireContext) (bool, error) {
	preCtx.SetFromConsole()
	if !d.IsOpenConsoleAuth() {
		return true, nil
	}
	if d.IsOpenConsoleAuth() && !d.conf.ConsoleStrict {
		preCtx.SetAllowAnonymous(true)
	}
	// 如果是初始化主用户的请求，直接放行
	if authcommon.IsInitMainUser(preCtx.GetRequestContext()) {
		return true, nil
	}
	return d.CheckPermission(preCtx)
}

// CheckPermission 执行检查动作判断是否有权限
func (d *DefaultAuthChecker) CheckPermission(authCtx *authcommon.AcquireContext) (bool, error) {
	if err := d.userSvr.CheckCredential(authCtx); err != nil {
		return false, err
	}
	log.Info("[Auth][Checker] check permission args", utils.RequestID(authCtx.GetRequestContext()),
		zap.Any("method", authCtx.GetMethods()), zap.Any("resources", authCtx.GetAccessResources()))

	if pass, _ := d.doCheckPermission(authCtx); pass {
		return true, nil
	}

	// 触发缓存的同步，避免鉴权策略和角色信息不一致导致的权限检查失败
	if err := d.resyncData(authCtx); err != nil {
		return false, err
	}
	return d.doCheckPermission(authCtx)
}

func (d *DefaultAuthChecker) resyncData(authCtx *authcommon.AcquireContext) error {
	if err := d.cacheMgr.AuthStrategy().Update(); err != nil {
		log.Error("[Auth][Checker] force sync policy failed", utils.RequestID(authCtx.GetRequestContext()), zap.Error(err))
		return err
	}
	if err := d.cacheMgr.Role().Update(); err != nil {
		log.Error("[Auth][Checker] force sync role failed", utils.RequestID(authCtx.GetRequestContext()), zap.Error(err))
		return err
	}
	return nil
}

// doCheckPermission 执行权限检查
func (d *DefaultAuthChecker) doCheckPermission(authCtx *authcommon.AcquireContext) (bool, error) {
	if d.IsCredible(authCtx) {
		return true, nil
	}
	cur := authCtx.GetAttachments()[authcommon.PrincipalKey].(authcommon.Principal)

	principals := d.listAllPrincipals(cur)

	// 遍历所有的 principal，检查是否有一个符合要求
	for i := range principals {
		principal := principals[i]
		allowPolicies := d.cacheMgr.AuthStrategy().GetPrincipalPolicies("allow", principal)
		denyPolicies := d.cacheMgr.AuthStrategy().GetPrincipalPolicies("deny", principal)

		resources := authCtx.GetAccessResources()

		// 先执行 deny 策略
		for i := range denyPolicies {
			item := denyPolicies[i]
			if d.MatchPolicy(authCtx, item, principal, resources) {
				return false, ErrorNotPermission
			}
		}

		// 处理 allow 策略，只要有一个放开，就可以认为通过
		for i := range allowPolicies {
			item := allowPolicies[i]
			if d.MatchPolicy(authCtx, item, principal, resources) {
				return true, nil
			}
		}
	}
	return false, ErrorNotPermission
}

func (d *DefaultAuthChecker) listAllPrincipals(p authcommon.Principal) []authcommon.Principal {
	principals := make([]authcommon.Principal, 0, 4)
	principals = append(principals, p)
	// 获取角色列表
	roles := d.cacheMgr.Role().GetPrincipalRoles(p)
	for i := range roles {
		principals = append(principals, authcommon.Principal{
			PrincipalID:   roles[i].ID,
			PrincipalType: authcommon.PrincipalRole,
		})
	}

	// 如果是用户，获取所在的用户组列表
	if p.PrincipalType == authcommon.PrincipalUser {
		groups := d.cacheMgr.User().GetUserLinkGroupIds(p.PrincipalID)
		for i := range groups {
			principals = append(principals, authcommon.Principal{
				PrincipalID:   groups[i],
				PrincipalType: authcommon.PrincipalGroup,
			})
		}
	}
	return principals
}

// IsCredible 检查是否是可信的请求
func (d *DefaultAuthChecker) IsCredible(authCtx *authcommon.AcquireContext) bool {
	reqHeaders, ok := authCtx.GetRequestContext().Value(utils.ContextRequestHeaders).(map[string][]string)
	if !ok || len(d.conf.CredibleHeaders) == 0 {
		return false
	}
	matched := true
	for k, v := range d.conf.CredibleHeaders {
		val, exist := reqHeaders[strings.ToLower(k)]
		if !exist {
			matched = false
			break
		}
		if len(val) == 0 {
			matched = false
		}
		matched = v == val[0]
		if !matched {
			break
		}
	}
	return matched
}

// MatchPolicy 检查策略是否匹配
func (d *DefaultAuthChecker) MatchPolicy(authCtx *authcommon.AcquireContext, policy *authcommon.StrategyDetail,
	principal authcommon.Principal, resources map[apisecurity.ResourceType][]authcommon.ResourceEntry) bool {
	if !d.MatchCalleeFunctions(authCtx, principal, policy) {
		log.Error("server function match policy fail", utils.RequestID(authCtx.GetRequestContext()),
			zap.String("principal", principal.String()), zap.String("policy-id", policy.ID))
		return false
	}
	if !d.MatchResourceOperateable(authCtx, principal, policy) {
		log.Error("access resource match policy fail", utils.RequestID(authCtx.GetRequestContext()),
			zap.String("principal", principal.String()), zap.String("policy-id", policy.ID))
		return false
	}
	return true
}

// MatchCalleeFunctions 检查操作方法是否和策略匹配
func (d *DefaultAuthChecker) MatchCalleeFunctions(authCtx *authcommon.AcquireContext,
	principal authcommon.Principal, policy *authcommon.StrategyDetail) bool {

	// 如果开启了兼容模式，并且策略没有对可调用方法的拦截，那么就认为匹配成功
	if d.conf.Compatible && len(policy.CalleeMethods) == 0 {
		return true
	}

	functions := policy.CalleeMethods

	allMatch := 0
	for _, method := range authCtx.GetMethods() {
		curMatch := false
		for i := range functions {
			if utils.IsMatchAll(functions[i]) {
				return true
			}
			if functions[i] == string(method) {
				curMatch = true
				break
			}
			if utils.IsWildMatch(string(method), functions[i]) {
				curMatch = true
				break
			}
		}
		if curMatch {
			allMatch++
		}
	}
	return allMatch == len(authCtx.GetMethods())
}

type (
	compatibleChecker func(ctx context.Context, cacheSvr cachetypes.CacheManager, resource *authcommon.ResourceEntry) bool
)

var (
	compatibleResource = map[apisecurity.ResourceType]compatibleChecker{
		apisecurity.ResourceType_UserGroups: func(ctx context.Context, cacheSvr cachetypes.CacheManager,
			resource *authcommon.ResourceEntry) bool {
			saveVal := cacheSvr.User().GetGroup(resource.ID)
			if saveVal == nil {
				return false
			}
			operator := utils.ParseUserID(ctx)
			_, exist := saveVal.UserIds[operator]
			return exist
		},
		apisecurity.ResourceType_PolicyRules: func(ctx context.Context, cacheSvr cachetypes.CacheManager,
			resource *authcommon.ResourceEntry) bool {
			saveVal := cacheSvr.AuthStrategy().GetPolicyRule(resource.ID)
			if saveVal == nil {
				return false
			}
			operator := utils.ParseUserID(ctx)
			for i := range saveVal.Principals {
				if saveVal.Principals[i].PrincipalID == operator {
					return true
				}
			}
			return false
		},
	}
)

// checkAction 检查操作资源是否和策略匹配
func (d *DefaultAuthChecker) MatchResourceOperateable(authCtx *authcommon.AcquireContext,
	principal authcommon.Principal, policy *authcommon.StrategyDetail) bool {

	// 检查下 principal 有没有 condition 信息
	principalCondition := make([]authcommon.Condition, 0, 4)
	// 这里主要兼容一些内部特殊场景，可能在 role/user/group 关联某个策略时，会有一些额外的关系属性，这里在 extend 统一查找
	_ = json.Unmarshal([]byte(principal.Extend["condition"]), &principalCondition)

	ctx := context.Background()
	if len(principalCondition) != 0 {
		ctx = context.WithValue(context.Background(), authcommon.ContextKeyConditions{}, principalCondition)
	}

	matchCheck := func(resType apisecurity.ResourceType, resources []authcommon.ResourceEntry) bool {
		for i := range resources {
			actionResult := d.cacheMgr.AuthStrategy().Hint(ctx, principal, &resources[i])
			if policy.IsMatchAction(actionResult.String()) {
				return true
			}
			// 兼容模式下，对于用户组和策略规则，走一遍兜底的检查逻辑
			if _, ok := compatibleResource[resType]; ok && d.conf.Compatible {
				return true
			}
		}
		return false
	}

	reqRes := authCtx.GetAccessResources()
	isMatch := true
	for k, v := range reqRes {
		subMatch := matchCheck(k, v)
		isMatch = isMatch && subMatch
	}
	return isMatch
}
