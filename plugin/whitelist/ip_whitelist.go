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

package whitelist

import (
	"errors"

	"github.com/polarismesh/polaris/plugin"
)

const PluginName = "whitelist"

func init() {
	plugin.RegisterPlugin(PluginName, &ipWhitelist{})
}

type ipWhitelist struct {
	ips map[string]bool
}

// Name 插件名称
func (i *ipWhitelist) Name() string {
	return PluginName
}

// Initialize 初始化IP白名单插件
func (i *ipWhitelist) Initialize(conf *plugin.ConfigEntry) error {
	i.ips = make(map[string]bool)
	ips, ok := conf.Option["ip"].([]interface{})
	if !ok {
		return errors.New("whitelist plugin initialize error")
	}
	for _, ip := range ips {
		i.ips[ip.(string)] = true
	}
	return nil
}

// Destroy 销毁插件
func (i *ipWhitelist) Destroy() error {
	return nil
}

// Contain 白名单是否包含IP
func (i *ipWhitelist) Contain(entry interface{}) bool {
	ip, _ := entry.(string)
	return i.ips[ip]
}
