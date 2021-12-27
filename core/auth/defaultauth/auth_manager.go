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
	"strings"

	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/core/auth"

	api "github.com/polarismesh/polaris-server/common/api/v1"
)

var (
	emptyVal = struct{}{}
	passAll  = true
)

// Login 登陆动作
func (authMgn *polarisAuthManager) Login(name, password string) (string, error) {
	return "", nil
}

// HasPermission 执行检查动作判断是否有权限
// 如果当前的接口访问凭据token，操作者类型需要分两种情况进行考虑
// Case 1:
// 	如果当前token凭据是以用户的身份角色，则需要按找如下规则进行拉取涉及的权限
// 		a: 查询该用户所在的所有用户组信息（子用户可以获得所在的用户分组的所有资源写权限）
// 		b: 根据用户ID，查询所有该用户可能涉及的所有资源策略
// Case 2:
// 	如果当前token凭据是以用户组的省份角色，则需要按照如下规则进行策略拉取
// 		a: 根据用户组ID查询所有涉及该用户组的策略列表
func (authMgn *polarisAuthManager) HasPermission(ctx *auth.AcquireContext) (bool, error) {
	if !authMgn.IsOpenAuth() {
		return true, nil
	}
	tokenInfo, err := ParseToken(authMgn.opt.Salt, ctx.Token)
	if err != nil {
		return false, err
	}
	ut, id, err := authMgn.checkToken(tokenInfo[2])
	if err != nil {
		return false, err
	}

	var (
		strategys []*model.StrategyDetail
		ownerId   string
	)

	if ut == "uid" {
		strategys = authMgn.findStrategiesByUserID(id)
		// 获取主账户的 id 信息
		user := authMgn.cache.UserCache().GetUser(id)
		if user == nil {
			return false, errors.New("user not found")
		}
		ownerId = user.Owner
	} else {
		strategys = authMgn.findStrategiesByGroupID(id)
		// 获取主账户的 id 信息
		group := authMgn.cache.UserCache().GetUserGroup(id)
		if group == nil {
			return false, errors.New("usergroup not found")
		}
		ownerId = group.Owner
	}

	ctx.Attachment[auth.OperatoRoleKey] = ut
	ctx.Attachment[auth.OperatorIDKey] = id
	ctx.Attachment[auth.OperatorOwnerKey] = ownerId

	return authMgn.hasPermission(ctx, strategys)
}

// ChangeOpenStatus 修改权限功能的开关状态
func (authMgn *polarisAuthManager) ChangeOpenStatus(status auth.AuthStatus) bool {
	authMgn.opt.Open = (status == auth.OpenAuthService)
	return false
}

// IsOpenAuth 返回是否开启了操作鉴权
func (authMgn *polarisAuthManager) IsOpenAuth() bool {
	return authMgn.opt.Open
}

// AfterResourceOperation 对于资源的添加删除操作，需要执行后置逻辑
// 1、每个资源都属于所创建的主用户，每个主账号只能看到自己的资源，跨主账号资源不可见；
// 2、同一主用户下的所有子用户，对主账号的资源都默认具备读权限；
// 3、所有子用户或者用户分组，都默认获得对所创建的资源的写权限
// 4、子用户可以获得所在的用户分组的所有资源写权限
func (authMgn *polarisAuthManager) AfterResourceOperation(afterCtx *auth.AcquireContext) {
	//TODO 需要仔细考虑，这个地方很重要！！！
}

func (authMgn *polarisAuthManager) hasPermission(ctx *auth.AcquireContext, strategys []*model.StrategyDetail) (bool, error) {
	reqRes := ctx.Resources
	var (
		checkNamespace   bool = false
		checkService     bool = true
		checkConfigGroup bool = true
	)

	for sPos := range strategys {
		rule := strategys[sPos]
		if !authMgn.checkAction(rule.Action, ctx.Operation) {
			continue
		}
		searchMaps := buildSearchMap(rule.Resources)

		// 检查 namespace
		checkNamespace = checkAnyElementExist(reqRes[api.ResourceType_Namespaces], searchMaps[0])
		// 检查 service
		if ctx.Module == auth.DiscoverModule {
			checkService = checkAnyElementExist(reqRes[api.ResourceType_Services], searchMaps[1])
		}
		// 检查 config_group
		if ctx.Module == auth.ConfigModule {
			checkConfigGroup = checkAnyElementExist(reqRes[api.ResourceType_ConfigGroups], searchMaps[2])
		}
	}

	if checkNamespace && checkService && checkConfigGroup {
		return true, nil
	}

	return false, errors.New("permission check failed, operation is forbidden")
}

// findStrategiesByUserID
func (authMgn *polarisAuthManager) findStrategiesByUserID(id string) []*model.StrategyDetail {
	// 第一步，先拉去这个用户自己涉及的所有策略信息
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
func (authMgn *polarisAuthManager) findStrategiesByGroupID(id string) []*model.StrategyDetail {
	return authMgn.cache.StrategyCache().GetStrategyDetailsByGroupID(id)
}

// checkAction 检查操作是否和策略匹配
func (authMgn *polarisAuthManager) checkAction(expect string, actual auth.ResourceOperation) bool {
	return true
}

// checkToken 对 token 进行检查
func (authMgn *polarisAuthManager) checkToken(tokenUserInfo string) (string, string, error) {
	detail := strings.Split(tokenUserInfo, "/")
	if len(detail) != 2 {
		return "", "", errors.New("illegal token")
	}

	infoType := detail[0]
	id := detail[1]
	if infoType == "uid" {
		user := authMgn.cache.UserCache().GetUser(id)
		if user == nil {
			return "", "", errors.New("invalid token, no such user")
		}
		return detail[0], detail[1], nil
	}

	if infoType == "groupid" {
		group := authMgn.cache.UserCache().GetUserGroup(id)
		if group == nil {
			return "", "", errors.New("invalid token, no such group")
		}
		return detail[0], detail[1], nil
	}

	return "", "", errors.New("invalid token, unknown operator role type")
}

// checkAnyElementExist
func checkAnyElementExist(waitSearch []string, searchMaps *SearchMap) bool {
	if searchMaps.passAll {
		return true
	}

	for i := range waitSearch {
		ns := waitSearch[i]
		if _, ok := searchMaps.items[ns]; ok {
			return true
		}
	}

	return false
}

// buildSearchMap
func buildSearchMap(ss []model.StrategyResource) []*SearchMap {
	nsSearchMaps := &SearchMap{
		items:   make(map[string]interface{}),
		passAll: false,
	}
	svcSearchMaps := &SearchMap{
		items:   make(map[string]interface{}),
		passAll: false,
	}
	cfgSearchMaps := &SearchMap{
		items:   make(map[string]interface{}),
		passAll: false,
	}

	for i := range ss {
		val := ss[i]
		if val.ResType == int32(api.ResourceType_Namespaces) {
			nsSearchMaps.items[val.ResID] = emptyVal
			nsSearchMaps.passAll = (val.ResID == "*")
			continue
		}
		if val.ResType == int32(api.ResourceType_Services) {
			svcSearchMaps.items[val.ResID] = emptyVal
			svcSearchMaps.passAll = (val.ResID == "*")
			continue
		}
		if val.ResType == int32(api.ResourceType_ConfigGroups) {
			cfgSearchMaps.items[val.ResID] = emptyVal
			cfgSearchMaps.passAll = (val.ResID == "*")
			continue
		}
	}

	return []*SearchMap{nsSearchMaps, svcSearchMaps, cfgSearchMaps}
}

type SearchMap struct {
	items   map[string]interface{}
	passAll bool
}
