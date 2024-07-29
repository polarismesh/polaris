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
	// 如果鉴权能力没有开启，那就默认都可以进行操作
	if !d.IsOpenAuth() {
		return true
	}

	p, ok := ctx.GetAttachment(authcommon.PrincipalKey)
	if !ok {
		return false
	}
	return d.cacheMgr.AuthStrategy().Hint(p.(authcommon.Principal), res) != apisecurity.AuthAction_DENY
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
	return d.CheckPermission(preCtx)
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
func (d *DefaultAuthChecker) CheckPermission(authCtx *authcommon.AcquireContext) (bool, error) {
	if err := d.userSvr.CheckCredential(authCtx); err != nil {
		return false, err
	}
	if log.DebugEnabled() {
		log.Debug("[Auth][Checker] check permission args", utils.RequestID(authCtx.GetRequestContext()),
			zap.String("method", string(authCtx.GetMethod())), zap.Any("resources", authCtx.GetAccessResources()))
	}

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
		log.Error("[Auth][Checker] force sync policy rule to cache failed", utils.RequestID(authCtx.GetRequestContext()), zap.Error(err))
		return err
	}
	if err := d.cacheMgr.Role().Update(); err != nil {
		log.Error("[Auth][Checker] force sync role to cache failed", utils.RequestID(authCtx.GetRequestContext()), zap.Error(err))
		return err
	}
	return nil
}

// doCheckPermission 执行权限检查
func (d *DefaultAuthChecker) doCheckPermission(authCtx *authcommon.AcquireContext) (bool, error) {
	p, _ := authCtx.GetAttachments()[authcommon.PrincipalKey].(authcommon.Principal)
	if d.IsCredible(authCtx) {
		return true, nil
	}

	allowPolicies := d.cacheMgr.AuthStrategy().GetPrincipalPolicies("allow", p)
	denyPolicies := d.cacheMgr.AuthStrategy().GetPrincipalPolicies("deny", p)

	resources := authCtx.GetAccessResources()

	// 先执行 deny 策略
	for i := range denyPolicies {
		item := denyPolicies[i]
		if d.MatchPolicy(authCtx, item, p, resources) {
			return false, ErrorNotPermission
		}
	}

	// 处理 allow 策略，只要有一个放开，就可以认为通过
	for i := range allowPolicies {
		item := allowPolicies[i]
		if d.MatchPolicy(authCtx, item, p, resources) {
			return true, nil
		}
	}
	return false, ErrorNotPermission
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
		return false
	}
	if !d.MatchResourceOperateable(authCtx, principal, policy) {
		return false
	}
	if !d.MatchResourceConditions(authCtx, principal, policy) {
		return false
	}
	return true
}

// MatchCalleeFunctions 检查操作方法是否和策略匹配
func (d *DefaultAuthChecker) MatchCalleeFunctions(authCtx *authcommon.AcquireContext,
	principal authcommon.Principal, policy *authcommon.StrategyDetail) bool {
	functions := policy.CalleeMethods
	for i := range functions {
		if functions[i] == string(authCtx.GetMethod()) {
			return true
		}
		if utils.IsWildMatch(string(authCtx.GetMethod()), functions[i]) {
			return true
		}
	}
	return false
}

// checkAction 检查操作资源是否和策略匹配
func (d *DefaultAuthChecker) MatchResourceOperateable(authCtx *authcommon.AcquireContext,
	principal authcommon.Principal, policy *authcommon.StrategyDetail) bool {
	matchCheck := func(resType apisecurity.ResourceType, resources []authcommon.ResourceEntry) bool {
		for i := range resources {
			actionResult := d.cacheMgr.AuthStrategy().Hint(principal, &resources[i])
			if actionResult.String() == policy.Action {
				return true
			}
		}
		return false
	}

	reqRes := authCtx.GetAccessResources()
	isMatch := false
	for k, v := range reqRes {
		if isMatch = matchCheck(k, v); isMatch {
			break
		}
	}
	return isMatch
}

// MatchResourceConditions 检查操作资源所拥有的标签是否和策略匹配
func (d *DefaultAuthChecker) MatchResourceConditions(authCtx *authcommon.AcquireContext,
	principal authcommon.Principal, policy *authcommon.StrategyDetail) bool {
	matchCheck := func(resType apisecurity.ResourceType, resources []authcommon.ResourceEntry) bool {
		conditions := policy.Conditions

		for i := range resources {
			allMatch := true
			for j := range conditions {
				condition := conditions[j]
				resVal, ok := resources[i].Metadata[condition.Key]
				if !ok {
					allMatch = false
					break
				}
				compareFunc, ok := conditionCompareDict[condition.CompareFunc]
				if !ok {
					allMatch = false
					break
				}
				if allMatch = compareFunc(resVal, condition.Value); !allMatch {
					break
				}
			}
			if allMatch {
				return true
			}
		}
		return false
	}

	reqRes := authCtx.GetAccessResources()
	isMatch := false
	for k, v := range reqRes {
		if isMatch = matchCheck(k, v); isMatch {
			break
		}
	}
	return isMatch
}

var (
	conditionCompareDict = map[string]func(string, string) bool{
		"for_any_value:string_equal": func(s1, s2 string) bool {
			return s1 == s2
		},
	}
)
