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

package auth

import (
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"

	"github.com/polarismesh/polaris/common/utils"
)

// PrincipalResourceContainer principal 资源容器
type PrincipalResourceContainer struct {
	denyResources  *utils.SyncMap[apisecurity.ResourceType, *utils.RefSyncSet[string, string]]
	allowResources *utils.SyncMap[apisecurity.ResourceType, *utils.RefSyncSet[string, string]]
}

// NewPrincipalResourceContainer 创建 PrincipalResourceContainer 对象
func NewPrincipalResourceContainer() *PrincipalResourceContainer {
	return &PrincipalResourceContainer{
		allowResources: utils.NewSyncMap[apisecurity.ResourceType, *utils.RefSyncSet[string, string]](),
		denyResources:  utils.NewSyncMap[apisecurity.ResourceType, *utils.RefSyncSet[string, string]](),
	}
}

// Hint 返回该资源命中的策略类型, 优先匹配 deny, 其次匹配 allow, 否则返回 deny
func (p *PrincipalResourceContainer) Hint(rt apisecurity.ResourceType, resId string) (apisecurity.AuthAction, bool) {
	ids, ok := p.denyResources.Load(rt)
	if ok {
		if ids.Contains(utils.MatchAll) {
			return apisecurity.AuthAction_DENY, true
		}
		if ids.Contains(resId) {
			return apisecurity.AuthAction_DENY, true
		}
	}
	ids, ok = p.allowResources.Load(rt)
	if ok {
		if ids.Contains(utils.MatchAll) {
			return apisecurity.AuthAction_ALLOW, true
		}
		if ids.Contains(resId) {
			return apisecurity.AuthAction_ALLOW, true
		}
	}
	return 0, false
}

// SaveResource 保存资源
func (p *PrincipalResourceContainer) SaveResource(a apisecurity.AuthAction, r StrategyResource) {
	if a == apisecurity.AuthAction_ALLOW {
		p.saveResource(p.allowResources, r)
	} else {
		p.saveResource(p.denyResources, r)
	}
}

// DelResource 删除资源
func (p *PrincipalResourceContainer) DelResource(a apisecurity.AuthAction, r StrategyResource) {
	if a == apisecurity.AuthAction_ALLOW {
		p.delResource(p.allowResources, r)
	} else {
		p.delResource(p.denyResources, r)
	}
}

func (p *PrincipalResourceContainer) saveResource(
	container *utils.SyncMap[apisecurity.ResourceType, *utils.RefSyncSet[string, string]], res StrategyResource) {

	resType := apisecurity.ResourceType(res.ResType)
	container.ComputeIfAbsent(resType, func(k apisecurity.ResourceType) *utils.RefSyncSet[string, string] {
		return utils.NewRefSyncSet[string, string]()
	})

	ids, _ := container.Load(resType)
	ids.Add(utils.Reference[string, string]{
		Key:        res.ResID,
		Referencer: res.StrategyID,
	})
}

func (p *PrincipalResourceContainer) delResource(
	container *utils.SyncMap[apisecurity.ResourceType, *utils.RefSyncSet[string, string]], res StrategyResource) {

	resType := apisecurity.ResourceType(res.ResType)
	container.ComputeIfAbsent(resType, func(k apisecurity.ResourceType) *utils.RefSyncSet[string, string] {
		return utils.NewRefSyncSet[string, string]()
	})

	ids, _ := container.Load(resType)
	ids.Remove(utils.Reference[string, string]{
		Key:        res.ResID,
		Referencer: res.StrategyID,
	})
}
