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
	"fmt"
	"strings"

	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/core/auth"
	"golang.org/x/crypto/bcrypt"
)

var (
	emptyVal = struct{}{}
	passAll  = true

	ErrorNoUser                  error = errors.New("no such user")
	ErrorNoUserGroup             error = errors.New("no such group")
	ErrorWrongUsernameOrPassword error = errors.New("name or password is wrong")
	ErrorInvalidToken            error = errors.New("invalid token, token not exist")
	ErrorTokenDisabled           error = errors.New("token already disabled")
)

// Login 登陆动作
func (authMgn *defaultAuthManager) Login(name, password string) (string, error) {
	user := authMgn.cache.UserCache().GetUserByName(name)

	if user == nil {
		return "", ErrorNoUser
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return "", ErrorWrongUsernameOrPassword
		}
		return "", err
	}

	return user.Token, nil
}

// HasPermission 执行检查动作判断是否有权限
// 如果当前的接口访问凭据token，操作者类型需要分两种情况进行考虑
// Case 1:
// 	如果当前token凭据是以用户的身份角色，则需要按如下规则进行拉取涉及的权限
// 		a: 查询该用户所在的所有用户组信息（子用户可以获得所在的用户分组的所有资源写权限）
// 		b: 根据用户ID，查询该用户可能涉及的所有资源策略
// Case 2:
// 	如果当前token凭据是以用户组的身份角色，则需要按照如下规则进行策略拉取
// 		a: 根据用户组ID查询所有涉及该用户组的策略列表
func (authMgn *defaultAuthManager) HasPermission(ctx *model.AcquireContext) (bool, error) {
	if !authMgn.IsOpenAuth() {
		return true, nil
	}

	// 随机字符串::[uid/xxx | groupid/xxx]
	tokenInfo, err := authMgn.ParseToken(ctx.Token)
	if err != nil {
		return false, err
	}
	if err := authMgn.checkToken(tokenInfo); err != nil {
		return false, err
	}

	if tokenInfo.IsOwner {
		return true, nil
	}

	var (
		strategys []*model.StrategyDetail
		ownerId   string
	)

	if tokenInfo.Role == auth.RoleForUser {
		strategys = authMgn.findStrategiesByUserID(tokenInfo.ID)
		// 获取主账户的 id 信息
		user := authMgn.cache.UserCache().GetUser(tokenInfo.ID)
		if user == nil {
			return false, errors.New("user not found")
		}
		ownerId = user.Owner
	} else {
		strategys = authMgn.findStrategiesByGroupID(tokenInfo.ID)
		// 获取主账户的 id 信息
		group := authMgn.cache.UserCache().GetUserGroup(tokenInfo.ID)
		if group == nil {
			return false, errors.New("usergroup not found")
		}
		ownerId = group.Owner
	}

	ctx.Attachment[auth.OperatoRoleKey] = tokenInfo.Role
	ctx.Attachment[auth.OperatorIDKey] = tokenInfo.ID
	ctx.Attachment[auth.OperatorOwnerKey] = ownerId

	return authMgn.authPlugin.CheckPermission(ctx, strategys)
}

// ChangeOpenStatus 修改权限功能的开关状态
func (authMgn *defaultAuthManager) ChangeOpenStatus(status auth.AuthStatus) bool {
	AuthOption.Open = (status == auth.OpenAuthService)
	return false
}

// IsOpenAuth 返回是否开启了操作鉴权
func (authMgn *defaultAuthManager) IsOpenAuth() bool {
	return AuthOption.Open
}

// AfterResourceOperation 对于资源的添加删除操作，需要执行后置逻辑
// 所有子用户或者用户分组，都默认获得对所创建的资源的写权限
func (authMgn *defaultAuthManager) AfterResourceOperation(afterCtx *model.AcquireContext) {
	roleId := afterCtx.Attachment[auth.OperatorIDKey].(string)
	name := fmt.Sprintf("%s%s", model.DefaultStrategyPrefix, roleId)

	// 获取默认的策略规则
	strategy, err := authMgn.strategySvr.storage.GetStrategyDetailByName(name)
	if err != nil {
		log.Errorf("[Auth][Server] get default strategy by name(%s)", name)
		return
	}

	strategyResource := make([]*model.StrategyResource, 0)

	resources := afterCtx.Resources
	for rType, rIds := range resources {
		for i := range rIds {
			id := rIds[i]
			strategyResource = append(strategyResource, &model.StrategyResource{
				StrategyID: strategy.ID,
				ResType:    int32(rType),
				ResID:      id,
			})
		}
	}

	if afterCtx.Operation == model.Create {
		// 如果是写操作，那么采用松添加操作进行新增资源的添加操作
		err = authMgn.strategySvr.storage.LooseAddStrategyResources(strategyResource)
		if err != nil {
			log.Errorf("[Auth][Server] update default strategy by name(%s)", name)
			return
		}
	} else {
		err = authMgn.strategySvr.storage.DeleteStrategyResources(strategyResource)
		if err != nil {
			log.Errorf("[Auth][Server] remove default strategy by name(%s)", name)
			return
		}
	}

}

// findStrategiesByUserID
func (authMgn *defaultAuthManager) findStrategiesByUserID(id string) []*model.StrategyDetail {
	// 第一步，先拉取这个用户自己涉及的所有策略信息
	rules := authMgn.cache.StrategyCache().GetStrategyDetailsByUID(id)

	// 第二步，拉取这个用户所属的 group 信息
	groupIds := authMgn.cache.UserCache().ListUserBelongGroupIDS(id)
	for i := range groupIds {
		rules := authMgn.findStrategiesByGroupID(groupIds[i])
		rules = append(rules, rules...)
	}

	// 对拉取下来的策略进行去重
	temp := make(map[string]*model.StrategyDetail)

	for i := range rules {
		rule := rules[i]
		temp[rule.ID] = rule
	}

	ret := make([]*model.StrategyDetail, 0, len(temp))
	for _, val := range temp {
		ret = append(ret, val)
	}

	return ret
}

// findStrategiesByGroupID
func (authMgn *defaultAuthManager) findStrategiesByGroupID(id string) []*model.StrategyDetail {

	return authMgn.cache.StrategyCache().GetStrategyDetailsByGroupID(id)
}

// checkToken 对 token 进行检查
func (authMgn *defaultAuthManager) checkToken(tokenUserInfo TokenInfo) error {

	infoType := tokenUserInfo.Role
	id := tokenUserInfo.ID
	if infoType == auth.RoleForUser {
		user := authMgn.cache.UserCache().GetUser(id)
		if user == nil {
			return ErrorNoUser
		}
		if tokenUserInfo.Origin != user.Token {
			return ErrorInvalidToken
		}

		if !user.TokenEnable {
			return ErrorTokenDisabled
		}

		return nil
	}

	if infoType == auth.RoleForUserGroup {
		group := authMgn.cache.UserCache().GetUserGroup(id)
		if group == nil {
			return ErrorNoUserGroup
		}
		if tokenUserInfo.Origin != group.Token {
			return ErrorInvalidToken
		}

		if !group.TokenEnable {
			return ErrorTokenDisabled
		}
		return nil
	}

	return errors.New("invalid token, unknown operator role type")
}

// ParseToken
func (authMgn *defaultAuthManager) ParseToken(t string) (TokenInfo, error) {
	ret, err := decryptMessage([]byte(AuthOption.Salt), t)
	if err != nil {
		return TokenInfo{}, err
	}
	tokenDetails := strings.Split(ret, TokenSplit)
	if len(tokenDetails) != 2 {
		return TokenInfo{}, errors.New("illegal token")
	}

	detail := strings.Split(tokenDetails[1], "/")
	if len(detail) != 2 {
		return TokenInfo{}, errors.New("illegal token")
	}

	tokenInfo := TokenInfo{
		Origin:  t,
		RandStr: tokenDetails[0],
		Role:    detail[0],
		ID:      detail[1],
		IsOwner: authMgn.cache.UserCache().IsOwner(detail[1]),
	}

	return tokenInfo, nil
}
