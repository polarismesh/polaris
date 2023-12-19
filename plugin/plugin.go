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

package plugin

import (
	"fmt"
	"sync"
)

var (
	pluginSet = make(map[string]Plugin)
	config    = &Config{}
	once      sync.Once
)

// RegisterPlugin 注册插件
func RegisterPlugin(name string, plugin Plugin) {
	if _, exist := pluginSet[name]; exist {
		panic(fmt.Sprintf("existed plugin: name=%v", name))
	}

	pluginSet[name] = plugin
}

// SetPluginConfig 设置插件配置
func SetPluginConfig(c *Config) {
	config = c
}

// Plugin 通用插件接口
type Plugin interface {
	Name() string
	Initialize(c *ConfigEntry) error
	Destroy() error
}

// ConfigEntry 单个插件配置
type ConfigEntry struct {
	Name   string                 `yaml:"name"`
	Option map[string]interface{} `yaml:"option"`
}

// Config 插件配置
type Config struct {
	CMDB                 ConfigEntry      `yaml:"cmdb"`
	RateLimit            ConfigEntry      `yaml:"ratelimit"`
	History              PluginChanConfig `yaml:"history"`
	Statis               PluginChanConfig `yaml:"statis"`
	DiscoverStatis       ConfigEntry      `yaml:"discoverStatis"`
	ParsePassword        ConfigEntry      `yaml:"parsePassword"`
	Whitelist            ConfigEntry      `yaml:"whitelist"`
	MeshResourceValidate ConfigEntry      `yaml:"meshResourceValidate"`
	DiscoverEvent        PluginChanConfig `yaml:"discoverEvent"`
	Crypto               PluginChanConfig `yaml:"crypto"`
}

// PluginChanConfig 插件执行链配置
type PluginChanConfig struct {
	Name    string                 `yaml:"name"`
	Option  map[string]interface{} `yaml:"option"`
	Entries []ConfigEntry          `yaml:"entries"`
}
