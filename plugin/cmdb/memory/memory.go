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

package memory

import (
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/plugin"
)

const (
	// PluginName plugin name
	PluginName = "memory"
)

// init 自注册到插件列表
func init() {
	plugin.RegisterPlugin(PluginName, &Memory{})
}

// Memory 定义MemoryCMDB类
type Memory struct {
	key string
}

// Name 返回插件名
func (m *Memory) Name() string {
	return PluginName
}

// Initialize 初始化函数
func (m *Memory) Initialize(c *plugin.ConfigEntry) error {
	option := c.Option
	m.key = option["key"].(string)

	return nil
}

// Destroy 销毁函数
func (m *Memory) Destroy() error {
	return nil
}

// GetLocation 实现CMDB插件接口
func (m *Memory) GetLocation(host string) (*model.Location, error) {
	return nil, nil
}

// Range 实现CMDB插件接口
func (m *Memory) Range(handler func(host string, location *model.Location) (bool, error)) error {
	_, err := handler("", nil)
	if err != nil {
		return err
	}

	return nil
}

// Size 实现CMDB插件接口
func (m *Memory) Size() int32 {
	return 0
}
