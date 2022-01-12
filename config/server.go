/*
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
	"errors"
	"github.com/polarismesh/polaris-server/cache"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/config/service"
	"github.com/polarismesh/polaris-server/store"
	"go.uber.org/zap"
	"sync"
	"time"
)

const (
	EventTypePublishConfigFile = "PublishConfigFile"
)

var (
	server      = new(Server)
	once        = sync.Once{}
	initialized = false
)

// StartupConfig 配置中心模块启动参数
type StartupConfig struct {
	Open bool `yaml:"open"`
}

// Server 配置中心核心服务
type Server struct {
	storage     store.Store
	cache       *cache.FileCache
	service     service.API
	watchCenter *watchCenter
}

// InitConfigModule 初始化配置中心模块
func InitConfigModule(open bool) error {
	if !open {
		initialized = true
		return nil
	}

	var err error
	once.Do(func() {
		err = doInit()
	})

	if err != nil {
		return err
	}

	initialized = true
	return nil
}

func doInit() error {
	//1. 初始化存储模块
	s, err := store.GetStore()
	if err != nil {
		log.GetConfigLogger().Errorf("[Config][Server] can not get store, err: %s", err.Error())
		return errors.New("can not get store")
	}
	if s == nil {
		log.GetConfigLogger().Errorf("[Config][Server] store is null")
		return errors.New("store is null")
	}
	server.storage = s

	//2. 初始化缓存模块
	fileCache := cache.NewFileCache(s)
	server.cache = fileCache

	//3. 初始化 service 模块
	serviceImpl := service.NewServiceImpl(s, fileCache)
	server.service = serviceImpl

	//4. 初始化事件中心
	eventCenter := NewEventCenter()
	server.watchCenter = NewWatchCenter(eventCenter)

	//5. 初始化发布事件扫描器
	err = initReleaseMessageScanner(s, fileCache, eventCenter, time.Second)
	if err != nil {
		log.GetConfigLogger().Error("[Config][Server] init release message scanner error. ", zap.Error(err))
		return errors.New("init config module error")
	}

	log.GetConfigLogger().Infof("[Config][Server] startup config module success.")

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
