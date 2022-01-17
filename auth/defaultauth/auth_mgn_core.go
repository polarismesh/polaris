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

	"github.com/polarismesh/polaris-server/auth"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
	"go.uber.org/zap"
)

var (
	ErrorNoUser                  error = errors.New("no such user")
	ErrorNoUserGroup             error = errors.New("no such user group")
	ErrorNoNamespace             error = errors.New("no such namespace")
	ErrorNoService               error = errors.New("no such service")
	ErrorWrongUsernameOrPassword error = errors.New("name or password is wrong")
	ErrorTokenNotExist           error = errors.New("token not exist")
	ErrorTokenInvalid            error = errors.New("invalid token")
	ErrorTokenDisabled           error = errors.New("token already disabled")
)

// CheckPermission 执行检查动作判断是否有权限
// 	step 1. 判断是否开启了鉴权
// 	step 2. 对token进行检查判断
// 	step 3. 拉取token对应的操作者相关信息，注入到请求上下文中
// 	step 4. 进行权限检查
func (authMgn *defaultAuthManager) CheckPermission(authCtx *model.AcquireContext) (bool, error) {
	if !authMgn.IsOpenAuth() {
		return true, nil
	}

	ctx, tokenInfo, err := authMgn.verifyToken(authCtx.GetRequestContext(), authCtx.GetToken())
	if err != nil {
		return false, err
	}

	// 重新设置 RequestContext
	authCtx.SetRequestContext(ctx)

	// TODO 如果访问的资源，其 owner 找不到对应的用户，则认为是可以被随意操作的资源
	authMgn.removeNoOwnerResources(authCtx)

	strategys, ownerId, err := authMgn.findStrategies(tokenInfo)
	if err != nil {
		return false, err
	}

	authCtx.GetAttachment()[model.OperatorRoleKey] = tokenInfo.Role
	authCtx.GetAttachment()[model.OperatorPrincipalType] = func() model.PrincipalType {
		if tokenInfo.IsUserToken {
			return model.PrincipalUser
		}
		return model.PrincipalUserGroup
	}()
	authCtx.GetAttachment()[model.OperatorIDKey] = tokenInfo.OperatorID
	authCtx.GetAttachment()[model.OperatorOwnerKey] = ownerId
	authCtx.GetAttachment()[model.OperatorLinkStrategy] = strategys

	return authMgn.authPlugin.CheckPermission(authCtx, strategys)
}

// findStrategies Inquire about TOKEN information, the actual all-associated authentication strategy
func (authMgn *defaultAuthManager) findStrategies(tokenInfo TokenInfo) ([]*model.StrategyDetail, string, error) {
	var (
		strategys []*model.StrategyDetail
		ownerId   string
	)

	if tokenInfo.IsUserToken {
		strategys = authMgn.findStrategiesByUserID(tokenInfo.OperatorID)
		user := authMgn.cacheMgn.User().GetUserByID(tokenInfo.OperatorID)
		if user == nil {
			return nil, "", ErrorNoUser
		}
		ownerId = user.Owner
	} else {
		strategys = authMgn.findStrategiesByGroupID(tokenInfo.OperatorID)
		group := authMgn.cacheMgn.User().GetUserGroup(tokenInfo.OperatorID)
		if group == nil {
			return nil, "", ErrorNoUserGroup
		}
		ownerId = group.Owner
	}

	return strategys, ownerId, nil
}

// ChangeOpenStatus 修改权限功能的开关状态
func (authMgn *defaultAuthManager) ChangeOpenStatus(status auth.AuthStatus) bool {
	AuthOption.Open = (status == auth.OpenAuthService)
	return true
}

// IsOpenAuth 返回是否开启了操作鉴权
func (authMgn *defaultAuthManager) IsOpenAuth() bool {
	return AuthOption.Open
}

// findStrategiesByUserID 根据 user-id 查找相关联的鉴权策略（用户自己的 + 用户所在用户组的）
func (authMgn *defaultAuthManager) findStrategiesByUserID(id string) []*model.StrategyDetail {
	// The first step, first pull all the strategy information involved in this user.
	rules := authMgn.cacheMgn.AuthStrategy().GetStrategyDetailsByUID(id)

	// Step 2, pull the Group information to which this user belongs
	groupIds := authMgn.cacheMgn.User().ListUserBelongGroupIDS(id)
	for i := range groupIds {
		rules := authMgn.findStrategiesByGroupID(groupIds[i])
		rules = append(rules, rules...)
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

// findStrategiesByGroupID 根据 group-id 查找相关联的鉴权策略
func (authMgn *defaultAuthManager) findStrategiesByGroupID(id string) []*model.StrategyDetail {

	return authMgn.cacheMgn.AuthStrategy().GetStrategyDetailsByGroupID(id)
}

// checkToken 对 token 进行检查
func (authMgn *defaultAuthManager) checkToken(tokenUserInfo TokenInfo) (string, error) {

	id := tokenUserInfo.OperatorID
	if tokenUserInfo.IsUserToken {
		user := authMgn.Cache().User().GetUserByID(id)
		if user == nil {
			return "", ErrorNoUser
		}
		if tokenUserInfo.Origin != user.Token {
			return "", ErrorTokenNotExist
		}
		if !user.TokenEnable {
			return "", ErrorTokenDisabled
		}
		if user.Owner == "" {
			return user.ID, nil
		}
		return user.Owner, nil
	} else {
		group := authMgn.Cache().User().GetUserGroup(id)
		if group == nil {
			return "", ErrorNoUserGroup
		}
		if tokenUserInfo.Origin != group.Token {
			return "", ErrorTokenNotExist
		}
		if !group.TokenEnable {
			return "", ErrorTokenDisabled
		}
		return group.Owner, nil
	}
}

// verifyAuth token
func (authMgn *defaultAuthManager) verifyToken(ctx context.Context, token string) (context.Context, TokenInfo, error) {
	tokenInfo, err := authMgn.DecodeToken(token)
	if err != nil {
		return nil, TokenInfo{}, err
	}

	owner, err := authMgn.checkToken(tokenInfo)
	if err != nil {
		return nil, TokenInfo{}, err
	}

	ctx = context.WithValue(ctx, utils.ContextIsOwnerKey, tokenInfo.Role != model.SubAccountUserRole)
	ctx = context.WithValue(ctx, utils.ContextUserIDKey, tokenInfo.OperatorID)
	ctx = context.WithValue(ctx, utils.ContextOwnerIDKey, owner)

	if tokenInfo.IsUserToken {
		user := authMgn.Cache().User().GetUserByID(tokenInfo.OperatorID)
		tokenInfo.Role = user.Type
		ctx = context.WithValue(ctx, utils.ContextUserRoleIDKey, user.Type)
	}

	return ctx, tokenInfo, nil
}

// DecodeToken
func (authMgn *defaultAuthManager) DecodeToken(t string) (TokenInfo, error) {
	ret, err := decryptMessage([]byte(AuthOption.Salt), t)
	if err != nil {
		return TokenInfo{}, err
	}
	tokenDetails := strings.Split(ret, TokenSplit)
	if len(tokenDetails) != 2 {
		return TokenInfo{}, ErrorTokenInvalid
	}

	detail := strings.Split(tokenDetails[1], "/")
	if len(detail) != 2 {
		return TokenInfo{}, ErrorTokenInvalid
	}

	tokenInfo := TokenInfo{
		Origin:      t,
		RandStr:     tokenDetails[0],
		IsUserToken: detail[0] == model.TokenForUser,
		OperatorID:  detail[1],
	}

	return tokenInfo, nil
}

func (authMgn *defaultAuthManager) removeNoOwnerResources(authCtx *model.AcquireContext) {
	resources := authCtx.GetAccessResources()

	cacheMgn := authMgn.Cache()

	newAccessRes := make(map[api.ResourceType][]model.ResourceEntry, 0)

	// 检查命名空间
	nsRes := resources[api.ResourceType_Namespaces]
	newNsRes := make([]model.ResourceEntry, 0)
	for index := range nsRes {
		if val := cacheMgn.User().GetUserByID(nsRes[index].Owner); val != nil {
			newNsRes = append(newNsRes, nsRes[index])
		}
	}
	newAccessRes[api.ResourceType_Namespaces] = newNsRes

	if authCtx.GetModule() == model.DiscoverModule {
		// 检查命名空间
		svcRes := resources[api.ResourceType_Services]
		newSvcRes := make([]model.ResourceEntry, 0)
		for index := range svcRes {
			if val := cacheMgn.User().GetUserByID(svcRes[index].Owner); val != nil {
				newSvcRes = append(newSvcRes, svcRes[index])
			}
		}
		newAccessRes[api.ResourceType_Services] = newSvcRes
	}

	if authCtx.GetModule() == model.ConfigModule {
		// 检查命名空间
		cfgRes := resources[api.ResourceType_ConfigGroups]
		newCfgRes := make([]model.ResourceEntry, 0)
		for index := range cfgRes {
			if val := cacheMgn.User().GetUserByID(cfgRes[index].Owner); val != nil {
				newCfgRes = append(newCfgRes, cfgRes[index])
			}
		}
		newAccessRes[api.ResourceType_ConfigGroups] = newCfgRes
	}

	log.GetAuthLogger().Info("remove no owner resource", zap.Any("access resource", newAccessRes))

	authCtx.SetAccessResources(newAccessRes)

}
