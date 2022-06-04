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
	"os"
	"sync"

	"github.com/polarismesh/polaris-server/common/log"
	"go.uber.org/zap"
)

var (
	// 插件初始化原子变量
	authOnce = &sync.Once{}
)

// Auth AUTH插件接口
type Auth interface {
	Plugin

	Allow(platformID, platformToken string) bool

	CheckPermission(reqCtx interface{}, authRule interface{}) (bool, error)

	IsWhiteList(ip string) bool
}

// GetAuth 获取Auth插件
func GetAuth() Auth {
	c := &config.Auth

	plugin, exist := pluginSet[c.Name]
	if !exist {
		log.Error("[Plugin][Auth] not found", zap.String("name", c.Name))
		return nil
	}

	authOnce.Do(func() {
		if err := plugin.Initialize(c); err != nil {
			log.Errorf("plugin init err: %s", err.Error())
			os.Exit(-1)
		}
	})

	return plugin.(Auth)
}
