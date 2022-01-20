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

	"go.uber.org/zap"

	"github.com/polarismesh/polaris-server/cache"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/config/service"
	"github.com/polarismesh/polaris-server/store"
)

const (
	eventTypePublishConfigFile  = "PublishConfigFile"
	defaultExpireTimeAfterWrite = 60 * 60 // expire after 1 hour
)

var (
	server      = new(Server)
	once        = sync.Once{}
	initialized = false
)

// StartupConfig 配置中心模块启动参数
type StartupConfig struct {
	Open  bool                   `yaml:"open"`
	Cache map[string]interface{} `yaml:"cache"`
}

// Server 配置中心核心服务
type Server struct {
	storage     store.Store
	cache       *cache.FileCache
	service     service.API
	watchCenter *watchCenter
}

// InitConfigModule 初始化配置中心模块
func InitConfigModule(ctx context.Context, config StartupConfig) error {
	if !config.Open {
		initialized = true
		return nil
	}

	var err error
	once.Do(func() {
		err = doInit(ctx, config)
	})

	if err != nil {
		return err
	}

	initialized = true
	return nil
}

func doInit(ctx context.Context, config StartupConfig) error {
	//1. 初始化存储模块
	storage, err := store.GetStore()
	if err != nil {
		log.ConfigScope().Errorf("[Config][Server] can not get store, err: %s", err.Error())
		return errors.New("can not get store")
	}
	if storage == nil {
		log.ConfigScope().Errorf("[Config][Server] store is null")
		return errors.New("store is null")
	}
	server.storage = storage

	//2. 初始化缓存模块
	expireTimeAfterWrite, ok := config.Cache["expireTimeAfterWrite"]
	if !ok {
		expireTimeAfterWrite = defaultExpireTimeAfterWrite
	}

	cacheParam := cache.FileCacheParam{
		ExpireTimeAfterWrite: expireTimeAfterWrite.(int),
	}
	fileCache := cache.NewFileCache(ctx, storage, cacheParam)
	server.cache = fileCache

	//3. 初始化 service 模块
	serviceImpl := service.NewServiceImpl(storage, fileCache)
	server.service = serviceImpl

	//4. 初始化事件中心
	eventCenter := NewEventCenter()
	server.watchCenter = NewWatchCenter(eventCenter)

	//5. 初始化发布事件扫描器
	err = initReleaseMessageScanner(ctx, storage, fileCache, eventCenter, time.Second)
	if err != nil {
		log.ConfigScope().Error("[Config][Server] init release message scanner error. ", zap.Error(err))
		return errors.New("init config module error")
	}

	log.ConfigScope().Infof("[Config][Server] startup config module success.")

	return nil
}

// GetConfigServer 获取已经初始化好的ConfigServer
func GetConfigServer() (*Server, error) {
	if !initialized {
		return nil, errors.New("config server has not done initialize")
	}

	return server, nil
}

// WatchCenter 获取监听事件中心
func (cs *Server) WatchCenter() *watchCenter {
	return cs.watchCenter
}

// Service 获取配置中心核心服务模块
func (cs *Server) Service() service.API {
	return cs.service
}

// Cache 获取配置中心缓存模块
func (cs *Server) Cache() *cache.FileCache {
	return cs.cache
}
