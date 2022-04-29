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
	"errors"
	"strings"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
	"go.uber.org/zap"
)

// IsOpenConsoleAuth 针对控制台是否开启了操作鉴权
func (checker *defaultAuthChecker) IsOpenConsoleAuth() bool {
	return AuthOption.ConsoleOpen
}

// IsOpenClientAuth 针对客户端是否开启了操作鉴权
func (checker *defaultAuthChecker) IsOpenClientAuth() bool {
	return AuthOption.ClientOpen
}

func (checker *defaultAuthChecker) IsOpenMaintainAuth() bool {
	return AuthOption.MaintainOpen
}

// IsOpenAuth 返回对于控制台/客户端任意其中的一个是否开启了操作鉴权
func (checker *defaultAuthChecker) IsOpenAuth() bool {
	return checker.IsOpenConsoleAuth() || checker.IsOpenClientAuth()
}

// CheckClientPermission 执行检查客户端动作判断是否有权限，并且对 RequestContext 注入操作者数据
func (checker *defaultAuthChecker) CheckClientPermission(preCtx *model.AcquireContext) (bool, error) {
	if !checker.IsOpenClientAuth() {
		return true, nil
	}

	return checker.CheckPermission(preCtx)
}

// CheckConsolePermission 执行检查控制台动作判断是否有权限，并且对 RequestContext 注入操作者数据
func (checker *defaultAuthChecker) CheckConsolePermission(preCtx *model.AcquireContext) (bool, error) {
	if !checker.IsOpenConsoleAuth() {
		return true, nil
	}

	return checker.CheckPermission(preCtx)
}

// CheckClientPermission 执行检查运维动作判断是否有权限
func (checker *defaultAuthChecker) CheckMaintainPermission(preCtx *model.AcquireContext) (bool, error) {
	if !checker.IsOpenMaintainAuth() {
		return true, nil
	}

	if err := checker.VerifyCredential(preCtx); err != nil {
		return false, err
	}

	tokenInfo := preCtx.GetAttachment(model.TokenDetailInfoKey).(OperatorInfo)

	if tokenInfo.Disable {
		return false, model.ErrorTokenDisabled
	}

	if tokenInfo.Role != model.OwnerUserRole {
		return false, errors.New("only owner account can access maintain API")
	}

	return true, nil
}

// CheckPermission 执行检查动作判断是否有权限
// 	step 1. 判断是否开启了鉴权
// 	step 2. 对token进行检查判断
// 		case 1. 如果 token 被禁用
// 				a. 读操作，直接放通
// 				b. 写操作，快速失败
// 	step 3. 拉取token对应的操作者相关信息，注入到请求上下文中
// 	step 4. 进行权限检查
func (checker *defaultAuthChecker) CheckPermission(authCtx *model.AcquireContext) (bool, error) {
	reqId := utils.ParseRequestID(authCtx.GetRequestContext())
	if err := checker.VerifyCredential(authCtx); err != nil {
		return false, err
	}

	if authCtx.GetOperation() == model.Read {
		return true, nil
	}

	operatorInfo := authCtx.GetAttachment(model.TokenDetailInfoKey).(OperatorInfo)
	// 这里需要检查当 token 被禁止的情况，如果 token 被禁止，无论是否可以操作目标资源，都无法进行写操作
	if operatorInfo.Disable {
		return false, model.ErrorTokenDisabled
	}

	strategies, err := checker.findStrategies(operatorInfo)
	if err != nil {
		log.AuthScope().Error("[Auth][Checker] find strategies when check permission", utils.ZapRequestID(reqId),
			zap.Error(err), zap.Any("token", operatorInfo.String()))
		return false, err
	}

	authCtx.SetAttachment(model.OperatorLinkStrategy, strategies)

	noResourceNeedCheck := checker.removeNoStrategyResources(authCtx)
	if !noResourceNeedCheck && len(strategies) == 0 {
		log.AuthScope().Error("[Auth][Checker]", utils.ZapRequestID(reqId),
			zap.String("msg", "need check resource is not empty, but strategies is empty"))
		return false, errors.New("no permission")
	}

	log.AuthScope().Info("[Auth][Checker] check permission args", zap.Any("resources", authCtx.GetAccessResources()),
		zap.Any("strategies", strategies))

	ok, err := checker.authPlugin.CheckPermission(authCtx, strategies)
	if err != nil {
		log.AuthScope().Error("[Auth][Checker] check permission args", utils.ZapRequestID(reqId),
			zap.String("method", authCtx.GetMethod()), zap.Any("resources", authCtx.GetAccessResources()),
			zap.Any("strategies", strategies))
		log.AuthScope().Error("[Auth][Checker] check permission when request arrive", utils.ZapRequestID(reqId),
			zap.Error(err))
	}

	return ok, err
}

func canDowngradeAnonymous(authCtx *model.AcquireContext, err error) bool {
	if authCtx.GetModule() == model.AuthModule {
		return false
	}

	if AuthOption.Strict {
		return false
	}

	if errors.Is(err, model.ErrorTokenInvalid) {
		return true
	}
	if errors.Is(err, model.ErrorTokenNotExist) {
		return true
	}

	return false
}

// VerifyCredential 对 token 进行检查验证，并将 verify 过程中解析出的数据注入到 model.AcquireContext 中
// step 1. 首先对 token 进行解析，获取相关的数据信息，注入到整个的 AcquireContext 中
// step 2. 最后对 token 进行一些验证步骤的执行
// step 3. 兜底措施：如果开启了鉴权的非严格模式，则根据错误的类型，判断是否转为匿名用户进行访问
// 		- 如果不是访问权限控制相关模块（用户、用户组、权限策略），不得转为匿名用户
func (checker *defaultAuthChecker) VerifyCredential(authCtx *model.AcquireContext) error {
	reqId := utils.ParseRequestID(authCtx.GetRequestContext())

	checkErr := func() error {
		operator, err := checker.decodeToken(authCtx.GetToken())
		if err != nil {
			log.AuthScope().Error("[Auth][Checker] decode token", zap.Error(err))
			return model.ErrorTokenInvalid
		}

		ownerId, isOwner, err := checker.checkToken(&operator)
		if err != nil {
			log.AuthScope().Error("[Auth][Checker] check token", zap.Error(err))
			return err
		}

		operator.OwnerID = ownerId
		ctx := authCtx.GetRequestContext()
		ctx = context.WithValue(ctx, utils.ContextIsOwnerKey, isOwner)
		ctx = context.WithValue(ctx, utils.ContextUserIDKey, operator.OperatorID)
		ctx = context.WithValue(ctx, utils.ContextOwnerIDKey, ownerId)
		authCtx.SetRequestContext(ctx)
		checker.parseOperatorInfo(operator, authCtx)
		if operator.Disable {
			log.AuthScope().Warn("[Auth][Checker] token already disabled", utils.ZapRequestID(reqId),
				zap.Any("token", operator.String()))
		}
		return nil
	}()

	if checkErr != nil {
		if !canDowngradeAnonymous(authCtx, checkErr) {
			return checkErr
		}
		log.AuthScope().Warn("[Auth][Checker] parse operator info, downgrade to anonymous", utils.ZapRequestID(reqId),
			zap.Error(checkErr))
		// 操作者信息解析失败，降级为匿名用户
		authCtx.SetAttachment(model.TokenDetailInfoKey, newAnonymous())
	}

	return nil
}

func (checker *defaultAuthChecker) parseOperatorInfo(operator OperatorInfo, authCtx *model.AcquireContext) {

	ctx := authCtx.GetRequestContext()

	if operator.IsUserToken {
		user := checker.Cache().User().GetUserByID(operator.OperatorID)
		operator.Role = user.Type
		ctx = context.WithValue(ctx, utils.ContextUserRoleIDKey, user.Type)
	}

	authCtx.SetAttachment(model.OperatorRoleKey, operator.Role)
	authCtx.SetAttachment(model.OperatorPrincipalType, func() model.PrincipalType {
		if operator.IsUserToken {
			return model.PrincipalUser
		}
		return model.PrincipalGroup
	}())
	authCtx.SetAttachment(model.OperatorIDKey, operator.OperatorID)
	authCtx.SetAttachment(model.OperatorOwnerKey, operator)
	authCtx.SetAttachment(model.TokenDetailInfoKey, operator)

	authCtx.SetRequestContext(ctx)
}

// decodeToken 解析 token 信息，如果 t == ""，直接返回一个空对象
func (checker *defaultAuthChecker) decodeToken(t string) (OperatorInfo, error) {
	if t == "" {
		return OperatorInfo{}, model.ErrorTokenInvalid
	}

	ret, err := decryptMessage([]byte(AuthOption.Salt), t)
	if err != nil {
		return OperatorInfo{}, err
	}
	tokenDetails := strings.Split(ret, TokenSplit)
	if len(tokenDetails) != 2 {
		return OperatorInfo{}, model.ErrorTokenInvalid
	}

	detail := strings.Split(tokenDetails[1], "/")
	if len(detail) != 2 {
		return OperatorInfo{}, model.ErrorTokenInvalid
	}

	tokenInfo := OperatorInfo{
		Origin:      t,
		IsUserToken: detail[0] == model.TokenForUser,
		OperatorID:  detail[1],
		Role:        model.UnknownUserRole,
	}
	return tokenInfo, nil
}

// checkToken 对 token 进行检查，如果 token 是一个空，直接返回默认值，但是不返回错误
// return {owner-id} {is-owner} {error}
func (checker *defaultAuthChecker) checkToken(tokenInfo *OperatorInfo) (string, bool, error) {
	if IsEmptyOperator(*tokenInfo) {
		return "", false, nil
	}

	id := tokenInfo.OperatorID
	if tokenInfo.IsUserToken {
		user := checker.Cache().User().GetUserByID(id)
		if user == nil {
			return "", false, model.ErrorNoUser
		}
		if tokenInfo.Origin != user.Token {
			return "", false, model.ErrorTokenNotExist
		}
		tokenInfo.Disable = !user.TokenEnable
		if user.Owner == "" {
			return user.ID, true, nil
		}
		return user.Owner, false, nil
	} else {
		group := checker.Cache().User().GetGroup(id)
		if group == nil {
			return "", false, model.ErrorNoUserGroup
		}
		if tokenInfo.Origin != group.Token {
			return "", false, model.ErrorTokenNotExist
		}
		tokenInfo.Disable = !group.TokenEnable
		return group.Owner, false, nil
	}
}

// findStrategiesByUserID 根据 user-id 查找相关联的鉴权策略（用户自己的 + 用户所在用户组的）
func (checker *defaultAuthChecker) findStrategiesByUserID(userId string) []*model.StrategyDetail {
	// Step 1, first pull all the strategy information involved in this user.
	rules := checker.cacheMgn.AuthStrategy().GetStrategyDetailsByUID(userId)

	// Step 2, pull the Group information to which this user belongs
	groupIds := checker.cacheMgn.User().GetUserLinkGroupIds(userId)
	for i := range groupIds {
		ret := checker.findStrategiesByGroupID(groupIds[i])
		rules = append(rules, ret...)
	}

	// Take the strategy that pulls down
	temp := make(map[string]*model.StrategyDetail)

	for i := range rules {
		rule := rules[i]
		temp[rule.ID] = rule
	}

	ret := make([]*model.StrategyDetail, 0, len(temp))
	for _, val := range temp {
		ret = append(ret, val)
	}

	return rules
}

// findStrategies Inquire about TOKEN information, the actual all-associated authentication strategy
func (checker *defaultAuthChecker) findStrategies(tokenInfo OperatorInfo) ([]*model.StrategyDetail, error) {
	var (
		strategies []*model.StrategyDetail
	)

	if IsEmptyOperator(tokenInfo) {
		return make([]*model.StrategyDetail, 0), nil
	}

	if tokenInfo.IsUserToken {
		user := checker.cacheMgn.User().GetUserByID(tokenInfo.OperatorID)
		if user == nil {
			return nil, model.ErrorNoUser
		}
		strategies = checker.findStrategiesByUserID(tokenInfo.OperatorID)
	} else {
		group := checker.cacheMgn.User().GetGroup(tokenInfo.OperatorID)
		if group == nil {
			return nil, model.ErrorNoUserGroup
		}
		strategies = checker.findStrategiesByGroupID(tokenInfo.OperatorID)
	}

	return strategies, nil
}

// findStrategiesByGroupID 根据 group-id 查找相关联的鉴权策略
func (checker *defaultAuthChecker) findStrategiesByGroupID(id string) []*model.StrategyDetail {

	return checker.cacheMgn.AuthStrategy().GetStrategyDetailsByGroupID(id)
}

// removeNoStrategyResources 移除没有关联任何鉴权策略的资源
func (checker *defaultAuthChecker) removeNoStrategyResources(authCtx *model.AcquireContext) bool {
	reqId := utils.ParseRequestID(authCtx.GetRequestContext())

	resources := authCtx.GetAccessResources()
	cacheMgn := checker.Cache()
	newAccessRes := make(map[api.ResourceType][]model.ResourceEntry, 0)

	checkIsFree := func(resType api.ResourceType, entry model.ResourceEntry) bool {
		// if entry.Owner == "" ||
		// 	strings.Compare(strings.ToLower(entry.Owner), strings.ToLower("polaris")) == 0 {
		// 	return true
		// }
		return !cacheMgn.AuthStrategy().IsResourceLinkStrategy(resType, entry.ID)
	}

	// 检查命名空间
	nsRes := resources[api.ResourceType_Namespaces]
	newNsRes := make([]model.ResourceEntry, 0)
	for index := range nsRes {
		if checkIsFree(api.ResourceType_Namespaces, nsRes[index]) {
			continue
		}
		newNsRes = append(newNsRes, nsRes[index])
	}
	newAccessRes[api.ResourceType_Namespaces] = newNsRes

	if authCtx.GetModule() == model.DiscoverModule {
		// 检查服务
		svcRes := resources[api.ResourceType_Services]
		newSvcRes := make([]model.ResourceEntry, 0)
		for index := range svcRes {
			if checkIsFree(api.ResourceType_Services, svcRes[index]) {
				continue
			}
			newSvcRes = append(newSvcRes, svcRes[index])
		}
		newAccessRes[api.ResourceType_Services] = newSvcRes
	}

	if authCtx.GetModule() == model.ConfigModule {
		// 检查配置
		cfgRes := resources[api.ResourceType_ConfigGroups]
		newCfgRes := make([]model.ResourceEntry, 0)
		for index := range cfgRes {
			if checkIsFree(api.ResourceType_ConfigGroups, cfgRes[index]) {
				continue
			}
			newCfgRes = append(newCfgRes, cfgRes[index])
		}
		newAccessRes[api.ResourceType_ConfigGroups] = newCfgRes
	}

	log.AuthScope().Info("[Auth][Checker] remove no link strategy final result", utils.ZapRequestID(reqId),
		zap.Any("resource", newAccessRes))

	authCtx.SetAccessResources(newAccessRes)
	noResourceNeedCheck := authCtx.IsAccessResourceEmpty()
	if noResourceNeedCheck {
		log.AuthScope().Debug("[Auth][Checker]", utils.ZapRequestID(reqId),
			zap.String("msg", "need check permission resource is empty"))
	}

	return noResourceNeedCheck
}
