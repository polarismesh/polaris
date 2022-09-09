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

package v2

import (
	"time"

	apiv2 "github.com/polarismesh/polaris-server/common/api/v2"
)

// RoutingConfig 路由规则
type RoutingConfig struct {
	// ID 规则唯一标识
	ID string `json:"id"`
	// name 规则名称
	Name string `json:"name"`
	// policy 规则类型
	Policy string `json:"policy"`
	// config 具体的路由规则内容
	Config string `json:"config"`
	// enable 路由规则是否启用
	Enable bool `json:"enable"`
	// revision 路由规则的版本信息
	Revision string `json:"revision"`
	// valid 路由规则是否有效，没有被逻辑删除
	Valid bool `json:"flag"`
	// createtime 规则创建时间
	CreateTime time.Time `json:"ctime"`
	// modifytime 规则修改时间
	ModifyTime time.Time `json:"mtime"`
	// enabletime 规则最近一次启用时间
	EnableTime time.Time `json:"etime"`
}

func (r *RoutingConfig) GetRoutingPolicy() apiv2.RoutingPolicy {
	v, ok := apiv2.RoutingPolicy_value[r.Policy]

	if !ok {
		return apiv2.RoutingPolicy(-1)
	}

	return apiv2.RoutingPolicy(v)
}
