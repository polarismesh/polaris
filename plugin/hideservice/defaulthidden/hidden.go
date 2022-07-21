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

package defaulthidden

import (
	"strings"

	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/plugin"
)

const (
	// PluginName plugin name
	PluginName = "defaultHidden"
)

func init() {
	plugin.RegisterPlugin(PluginName, &defaultServiceHidden{})
}

type defaultServiceHidden struct {
	hiddenList []*model.ServiceKey
}

// Name 返回当前插件的名称
func (ds *defaultServiceHidden) Name() string {
	return PluginName
}

// Initialize 初始化插件
func (ds *defaultServiceHidden) Initialize(conf *plugin.ConfigEntry) error {
	services, _ := conf.Option["services"].([]interface{})
	for _, serviceKey := range services {
		namespaceAndName := strings.Split(serviceKey.(string), ":")
		if len(namespaceAndName) != 2 {
			continue
		}
		ds.hiddenList = append(ds.hiddenList, &model.ServiceKey{
			Namespace: strings.TrimSpace(namespaceAndName[0]),
			Name:      strings.TrimSpace(namespaceAndName[1]),
		})
	}
	return nil
}

// Destroy 销毁函数
func (ds *defaultServiceHidden) Destroy() error {
	return nil
}

// GetHiddenList 获取隐藏服务列表
func (ds *defaultServiceHidden) GetHiddenList() []*model.ServiceKey {
	return ds.hiddenList
}
