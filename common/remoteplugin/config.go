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

package remoteplugin

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
)

const (
	// PluginRumModelLocal 本地模式
	PluginRumModelLocal = "local"
	// PluginRumModelRemote 远端模式
	PluginRumModelRemote = "remote"
)

// Config remote plugin config
type Config struct {
	// Name is the plugin unique and exclusive name
	Name string
	// Path is the plugin absolute file path to load.
	Path string
	// Mode is the plugin serverImp running mode, support local and remote.
	Mode string
	// MaxProcs the max proc number, current plugin can use.
	MaxProcs int
	// Args plugin args
	Args []string
}

// repairConfig repairs config.
func (c *Config) repairConfig() {
	if c.MaxProcs == 0 {
		c.MaxProcs = 1
	}
}

// pluginLoadPath 插件加载路径
func (c *Config) pluginLoadPath() (string, error) {
	fullPath := c.Path
	if fullPath == "" {
		// Use plugin name and using relative path to load plugin.
		dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			return "", fmt.Errorf("fail to find worksapce: %w", err)
		}
		fullPath = path.Join(dir, c.Name)
	}
	if _, err := os.Stat(fullPath); err != nil {
		return "", fmt.Errorf("check plugin file stat error: %w", err)
	}
	return fullPath, nil
}
