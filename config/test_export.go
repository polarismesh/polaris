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

package config

import (
	"context"

	"github.com/polarismesh/polaris/auth"
	"github.com/polarismesh/polaris/cache"
	"github.com/polarismesh/polaris/namespace"
	"github.com/polarismesh/polaris/store"
	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
)

// Initialize 初始化配置中心模块
func TestInitialize(ctx context.Context, config Config, s store.Store, cacheMgn *cache.CacheManager,
	namespaceOperator namespace.NamespaceOperateServer, userMgn auth.UserServer,
	strategyMgn auth.StrategyServer) (ConfigCenterServer, ConfigCenterServer, error) {

	err := originServer.initialize(ctx, config, s, namespaceOperator, cacheMgn)
	if err != nil {
		return nil, nil, err
	}

	server = newServerAuthAbility(originServer, userMgn, strategyMgn)
	return newServerAuthAbility(originServer, userMgn, strategyMgn), originServer, err
}

func TestCompareByVersion(clientConfigFile *apiconfig.ClientConfigFileInfo, cacheEntry *cache.Entry) bool {
	return compareByVersion(clientConfigFile, cacheEntry)
}

func TestCompareByMD5(clientConfigFile *apiconfig.ClientConfigFileInfo, cacheEntry *cache.Entry) bool {
	return compareByMD5(clientConfigFile, cacheEntry)
}

// TestDecryptConfigFile 解密配置文件
func (s *Server) TestDecryptConfigFile(ctx context.Context, configFile *apiconfig.ConfigFile) (err error) {
	return s.decryptConfigFile(ctx, configFile)
}

// TestEncryptConfigFile 解密配置文件
func (s *Server) TestEncryptConfigFile(ctx context.Context,
	configFile *apiconfig.ConfigFile, algorithm string, dataKey string) error {
	return s.encryptConfigFile(ctx, configFile, algorithm, dataKey)
}

// TestMockStore
func (s *Server) TestMockStore(ms store.Store) {
	s.storage = ms
}
