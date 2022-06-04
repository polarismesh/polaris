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
	"errors"
	"sync"
	"time"

	"github.com/polarismesh/polaris-server/auth"

	"go.uber.org/zap"

	"github.com/polarismesh/polaris-server/cache"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/store"
)

var _ ConfigCenterServer = (*Server)(nil)

const (
	eventTypePublishConfigFile  = "PublishConfigFile"
	defaultExpireTimeAfterWrite = 60 * 60 // expire after 1 hour
)

var (
	server       ConfigCenterServer
	originServer = new(Server)
	once         = sync.Once{}
	initialized  = false
)

// StartupConfig 配置中心模块启动参数
type Config struct {
	Open  bool                   `yaml:"open"`
	Cache map[string]interface{} `yaml:"cache"`
}

// Server 配置中心核心服务
type Server struct {
	storage     store.Store
	cache       *cache.FileCache
	watchCenter *watchCenter
	connManager *connManager
}

// Initialize 初始化配置中心模块
func Initialize(ctx context.Context, config Config, s store.Store, cacheMgn *cache.NamingCache,
	authSvr auth.AuthServer) error {

	if !config.Open {
		initialized = true
		return nil
	}

	var err error
	once.Do(func() {
		err = initialize(ctx, config, s, cacheMgn, authSvr)
	})

	if err != nil {
		return err
	}

	initialized = true
	return nil
}

func initialize(ctx context.Context, config Config, s store.Store, cacheMgn *cache.NamingCache,
	authSvr auth.AuthServer) error {

	originServer.storage = s

	// 2. 初始化缓存模块
	expireTimeAfterWrite, ok := config.Cache["expireTimeAfterWrite"]
	if !ok {
		expireTimeAfterWrite = defaultExpireTimeAfterWrite
	}

	cacheParam := cache.FileCacheParam{
		ExpireTimeAfterWrite: expireTimeAfterWrite.(int),
	}
	fileCache := cache.NewFileCache(ctx, s, cacheParam)
	originServer.cache = fileCache

	// 3. 初始化 service 模块

	// 4. 初始化事件中心
	eventCenter := NewEventCenter()
	originServer.watchCenter = NewWatchCenter(eventCenter)

	// 5. 初始化连接管理器
	connMng := NewConfigConnManager(ctx, originServer.watchCenter)
	originServer.connManager = connMng

	// 6. 初始化发布事件扫描器
	if err := initReleaseMessageScanner(ctx, s, fileCache, eventCenter, time.Second); err != nil {
		log.ConfigScope().Error("[Config][Server] init release message scanner error. ", zap.Error(err))
		return errors.New("init config module error")
	}

	log.ConfigScope().Infof("[Config][Server] startup config module success.")

	server = newServerAuthAbility(originServer, authSvr)

	return nil
}

// GetServer 获取已经初始化好的ConfigServer
func GetServer() (ConfigCenterServer, error) {
	if !initialized {
		return nil, errors.New("config server has not done initialize")
	}

	return server, nil
}

func GetOriginServer() (*Server, error) {
	if !initialized {
		return nil, errors.New("config server has not done initialize")
	}

	return originServer, nil
}

// WatchCenter 获取监听事件中心
func (cs *Server) WatchCenter() *watchCenter {
	return cs.watchCenter
}

// Cache 获取配置中心缓存模块
func (cs *Server) Cache() *cache.FileCache {
	return cs.cache
}

// ConnManager 获取配置中心连接管理器
func (cs *Server) ConnManager() *connManager {
	return cs.connManager
}
