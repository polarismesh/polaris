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

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/plugin"
)

type defaultAuth struct {
	whiteList string
}

func (da *defaultAuth) Name() string {
	return PluginName
}

/**
 * Initialize 初始化鉴权插件
 */
func (da *defaultAuth) Initialize(conf *plugin.ConfigEntry) error {
	whiteList, _ := conf.Option["white-list"].(string)
	da.whiteList = whiteList

	return nil
}

// Destroy 销毁函数
func (da *defaultAuth) Destroy() error {
	return nil
}

func (da *defaultAuth) Allow(platformID, platformToken string) bool {
	return true
}

// CheckPermission 
//  @receiver da 
//  @param reqCtx 
//  @param authRule 
//  @return bool 
//  @return error 
func (da *defaultAuth) CheckPermission(reqCtx interface{}, authRule interface{}) (bool, error) {

	ctx, ok := reqCtx.(*model.AcquireContext)
	if !ok {
		return false, errors.New("invalid parameter")
	}

	ownerId := ctx.GetAttachment()[model.OperatorIDKey].(string)
	strategys, ok := authRule.([]*model.StrategyDetail)

	reqRes := ctx.GetAccessResources()
	var (
		checkNamespace   bool = false
		checkService     bool = true
		checkConfigGroup bool = true
	)

	for index := range strategys {
		rule := strategys[index]

		if !da.checkAction(rule.Action, ctx.GetOperation()) {
			continue
		}
		searchMaps := buildSearchMap(rule.Resources)

		// 检查 namespace
		checkNamespace = checkAnyElementExist(ownerId, reqRes[api.ResourceType_Namespaces], searchMaps[0])
		// 检查 service
		if ctx.GetModule() == model.DiscoverModule {
			checkService = checkAnyElementExist(ownerId, reqRes[api.ResourceType_Services], searchMaps[1])
		}
		// 检查 config_group
		if ctx.GetModule() == model.ConfigModule {
			checkConfigGroup = checkAnyElementExist(ownerId, reqRes[api.ResourceType_ConfigGroups], searchMaps[2])
		}

		if checkNamespace && (checkService || checkConfigGroup) {
			return true, nil
		}
	}

	return false, errors.New("permission check failed, operation is forbidden")
}

func (da *defaultAuth) IsWhiteList(ip string) bool {
	if ip == "" || da.whiteList == "" {
		return false
	}

	return strings.Contains(da.whiteList, ip)
}

// checkAction 检查操作是否和策略匹配
func (da *defaultAuth) checkAction(expect string, actual model.ResourceOperation) bool {
	return true
}

// checkAnyElementExist 检查待操作的资源是否符合鉴权资源列表的配置
//  @param userId 当前的用户信息
//  @param waitSearch 访问的资源
//  @param searchMaps 鉴权策略中某一类型的资源列表信息
//  @return bool 是否可以操作本次被访问的所有资源
func checkAnyElementExist(userId string, waitSearch []model.ResourceEntry, searchMaps *SearchMap) bool {
	if len(waitSearch) == 0 {
		return true
	}

	if searchMaps.passAll {
		return true
	}

	for i := range waitSearch {
		entry := waitSearch[i]
		if entry.Owner == userId {
			continue
		}
		if _, ok := searchMaps.items[entry.ID]; !ok {
			return false
		}
	}

	return true
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
			nsSearchMaps.passAll = (val.ResID == "*") || nsSearchMaps.passAll
			continue
		}
		if val.ResType == int32(api.ResourceType_Services) {
			svcSearchMaps.items[val.ResID] = emptyVal
			svcSearchMaps.passAll = (val.ResID == "*") || nsSearchMaps.passAll
			continue
		}
		if val.ResType == int32(api.ResourceType_ConfigGroups) {
			cfgSearchMaps.items[val.ResID] = emptyVal
			cfgSearchMaps.passAll = (val.ResID == "*") || nsSearchMaps.passAll
			continue
		}
	}

	return []*SearchMap{nsSearchMaps, svcSearchMaps, cfgSearchMaps}
}

// SearchMap 权限搜索map
type SearchMap struct {
	// 某个资源策略的去重map
	items map[string]interface{}
	// 该资源策略是否允许全部操作
	passAll bool
}
