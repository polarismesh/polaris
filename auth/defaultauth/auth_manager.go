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
	"golang.org/x/crypto/bcrypt"
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

// Login 登陆动作
//  @receiver authMgn
//  @param req
//  @return string
//  @return error
func (authMgn *defaultAuthManager) Login(req *api.LoginRequest) *api.Response {
	user := authMgn.cacheMgn.User().GetUserByName(req.GetName().GetValue(), req.GetOwner().GetValue())

	if user == nil {
		return api.NewResponse(api.NotFoundUser)
	}

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

// CheckPermission 执行检查动作判断是否有权限
func (authMgn *defaultAuthManager) CheckPermission(authCtx *model.AcquireContext) (bool, error) {
	if !authMgn.IsOpenAuth() {
		return true, nil
	}

	ctx, tokenInfo, err := authMgn.verifyToken(authCtx.GetRequestContext(), authCtx.GetToken())
	if err != nil {
		return false, err
	}
	authCtx.SetRequestContext(ctx)

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

// AfterResourceOperation 对于资源的添加删除操作，需要执行后置逻辑
// 所有子用户或者用户分组，都默认获得对所创建的资源的写权限
func (authMgn *defaultAuthManager) AfterResourceOperation(afterCtx *model.AcquireContext) {
	operatorId := afterCtx.GetAttachment()[model.OperatorIDKey].(string)
	principalType := afterCtx.GetAttachment()[model.OperatorPrincipalType].(model.PrincipalType)

	// 获取该用户的默认策略信息
	name := model.BuildDefaultStrategyName(operatorId, principalType)
	ownerId := afterCtx.GetAttachment()[model.OperatorOwnerKey].(string)
	// Get the default policy rules
	strategy, err := authMgn.strategySvr.storage.GetStrategyDetailByName(ownerId, name)
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
		err = authMgn.strategySvr.storage.LooseAddStrategyResources(strategyResource)
		if err != nil {
			log.GetAuthLogger().Error("[Auth][Server] update default strategy resource",
				zap.String("owner", ownerId), zap.String("name", name), zap.Error(err))
			return
		}
	}
	if afterCtx.GetOperation() == model.Delete {
		err = authMgn.strategySvr.storage.RemoveStrategyResources(strategyResource)
		if err != nil {
			log.GetAuthLogger().Error("[Auth][Server] remove default strategy resource",
				zap.String("owner", ownerId), zap.String("name", name), zap.Error(err))
			return
		}
	}

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
	ctx = context.WithValue(ctx, utils.ContextUserRoleIDKey, tokenInfo.Role)
	ctx = context.WithValue(ctx, utils.ContextOwnerIDKey, owner)

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
