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

package password

import "github.com/polarismesh/polaris/plugin"

const (
	PluginName = "localParse"
)

// init 初始化注册函数
func init() {
	plugin.RegisterPlugin(PluginName, &Password{})
}

// Password 密码插件
type Password struct{}

// Name 返回插件名字
func (p *Password) Name() string {
	return PluginName
}

// Destroy 销毁插件
func (p *Password) Destroy() error {
	return nil
}

// Initialize 插件初始化
func (p *Password) Initialize(c *plugin.ConfigEntry) error {
	return nil
}

// ParsePassword 解析密码
func (p *Password) ParsePassword(cipher string) (string, error) {
	return "", nil
}
