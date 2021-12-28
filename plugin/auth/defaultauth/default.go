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

const (
	PluginName string = "defaultAuth"
)

var (
	emptyVal = struct{}{}
	passAll  = true
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

func (da *defaultAuth) Allow(platformID, platformToken string) bool {
	return true
}

func (da *defaultAuth) CheckPermission(reqCtx interface{}, authRule interface{}) (bool, error) {

	ctx, ok := reqCtx.(*model.AcquireContext)
	if !ok {
		return false, errors.New("invalid parameter")
	}

	strategys, ok := authRule.([]*model.StrategyDetail)

	reqRes := ctx.Resources
	var (
		checkNamespace   bool = false
		checkService     bool = true
		checkConfigGroup bool = true
	)

	for sPos := range strategys {
		rule := strategys[sPos]
		if !da.checkAction(rule.Action, ctx.Operation) {
			continue
		}
		searchMaps := buildSearchMap(rule.Resources)

		// 检查 namespace
		checkNamespace = checkAnyElementExist(reqRes[api.ResourceType_Namespaces], searchMaps[0])
		// 检查 service
		if ctx.Module == model.DiscoverModule {
			checkService = checkAnyElementExist(reqRes[api.ResourceType_Services], searchMaps[1])
		}
		// 检查 config_group
		if ctx.Module == model.ConfigModule {
			checkConfigGroup = checkAnyElementExist(reqRes[api.ResourceType_ConfigGroups], searchMaps[2])
		}
	}

	if checkNamespace && checkService && checkConfigGroup {
		return true, nil
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
