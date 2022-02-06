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

package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/modern-go/reflect2"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	"github.com/polarismesh/polaris-server/auth"
	"github.com/polarismesh/polaris-server/cache"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/plugin"
	"github.com/polarismesh/polaris-server/service/batch"
	"github.com/polarismesh/polaris-server/store"
)

const (
	// MaxBatchSize max batch size
	MaxBatchSize = 100
	// MaxQuerySize max query size
	MaxQuerySize = 100
)

const (
	// SystemNamespace polaris system namespace
	SystemNamespace = "Polaris"
	// DefaultNamespace default namespace
	DefaultNamespace = "default"
	// ProductionNamespace default namespace
	ProductionNamespace = "Production"
	// DefaultTLL default ttl
	DefaultTLL = 5
)

var (
	server     = new(Server)
	once       = sync.Once{}
	finishInit = false
)

// Config 核心逻辑层配置
type Config struct {
	Auth  map[string]interface{} `yaml:"auth"`
	Batch map[string]interface{} `yaml:"batch"`
}

// Server 对接API层的server层，用以处理业务逻辑
type Server struct {
	storage store.Store

	caches    *cache.NamingCache
	authority auth.Authority
	bc        *batch.Controller

	cmdb           plugin.CMDB
	history        plugin.History
	ratelimit      plugin.Ratelimit
	discoverStatis plugin.DiscoverStatis
	discoverEvent  plugin.DiscoverChannel
	auth           plugin.Auth

	l5service *l5service

	creatServiceSingle *singleflight.Group
}

// Initialize 初始化
func Initialize(ctx context.Context, namingOpt *Config, cacheOpt *cache.Config, listener cache.Listener) error {
	var err error
	once.Do(func() {
		err = initialize(ctx, namingOpt, cacheOpt, listener)
	})

	if err != nil {
		return err
	}

	finishInit = true
	return nil
}

// GetServer 获取已经初始化好的Server
func GetServer() (*Server, error) {
	if !finishInit {
		return nil, errors.New("server has not done InitializeServer")
	}

	return server, nil
}

// Authority 返回鉴权对象，获取鉴权信息
func (s *Server) Authority() auth.Authority {
	return s.authority
}

// Cache 返回Cache
func (s *Server) Cache() *cache.NamingCache {
	return s.caches
}

// RecordHistory server对外提供history插件的简单封装
func (s *Server) RecordHistory(entry *model.RecordEntry) {
	// 如果插件没有初始化，那么不记录history
	if s.history == nil {
		return
	}
	// 如果数据为空，则不需要打印了
	if entry == nil {
		return
	}

	// 调用插件记录history
	s.history.Record(entry)
}

// RecordDiscoverStatis 打印服务发现统计
func (s *Server) RecordDiscoverStatis(service, namespace string) {
	if s.discoverStatis == nil {
		return
	}

	_ = s.discoverStatis.AddDiscoverCall(service, namespace, time.Now())
}

// PublishDiscoverEvent 发布服务事件
func (s *Server) PublishDiscoverEvent(event model.DiscoverEvent) {
	if s.discoverEvent == nil {
		return
	}

	s.discoverEvent.PublishEvent(event)
}

// GetServiceInstanceRevision 获取服务实例的revision
func (s *Server) GetServiceInstanceRevision(serviceID string, instances []*model.Instance) (string, error) {
	revision := s.caches.GetServiceInstanceRevision(serviceID)
	if revision != "" {
		return revision, nil
	}

	data, err := cache.ComputeRevision(serviceID, instances)
	if err != nil {
		return "", err
	}

	return data, nil
}

// 封装一下cmdb的GetLocation
func (s *Server) getLocation(host string) *model.Location {
	if s.cmdb == nil {
		return nil
	}

	location, err := s.cmdb.GetLocation(host)
	if err != nil {
		log.Errorf("[Server] get location(%s) err: %s", host, err.Error())
		return nil
	}
	return location
}

// 实例访问限流
func (s *Server) allowInstanceAccess(instanceID string) bool {
	if s.ratelimit == nil {
		return true
	}

	if ok := s.ratelimit.Allow(plugin.InstanceRatelimit, instanceID); !ok {
		log.Error("[Server][ratelimit] instance is not allow access", zap.String("instance", instanceID))
		return false
	}

	return true

}

// 内部初始化函数
func initialize(ctx context.Context, namingOpt *Config, cacheOpt *cache.Config, listener cache.Listener) error {
	// 获取存储层对象
	s, err := store.GetStore()
	if err != nil {
		log.Errorf("[Naming][Server] can not get store, err: %s", err.Error())
		return errors.New("can not get store")
	}
	if s == nil {
		log.Errorf("[Naming][Server] store is null")
		return errors.New("store is null")
	}
	server.storage = s

	// 初始化鉴权模块
	authority, err := auth.NewAuthority(namingOpt.Auth)
	if err != nil {
		log.Errorf("[Naming][Server] new auth err: %s", err.Error())
		return err
	}
	server.authority = authority

	// cache模块，可以不开启
	// 对于控制台集群，只访问控制台接口的，可以不开启cache
	if cacheOpt.Open {
		cache.SetCacheConfig(cacheOpt)
		log.Infof("cache is open, can access the client api function")
		var listeners []cache.Listener
		if !reflect2.IsNil(listener) {
			listeners = append(listeners, listener)
		}
		caches, cacheErr := cache.NewNamingCache(s, listeners)
		if cacheErr != nil {
			log.Errorf("[Naming][Server] new naming cache err: %s", cacheErr.Error())
			return cacheErr
		}
		server.caches = caches
		if startErr := server.caches.Start(ctx); startErr != nil {
			log.Errorf("[Naming][Server] start naming cache err: %s", startErr.Error())
			return startErr
		}
	}

	// 批量控制器
	batchConfig, err := batch.ParseBatchConfig(namingOpt.Batch)
	if err != nil {
		return err
	}
	bc, err := batch.NewBatchCtrlWithConfig(server.storage, server.authority, plugin.GetAuth(), batchConfig)
	if err != nil {
		log.Errorf("new batch ctrl with config err: %s", err.Error())
		return err
	}
	server.bc = bc
	if server.bc != nil {
		server.bc.Start(ctx)
	}

	// l5service
	server.l5service = &l5service{}

	server.creatServiceSingle = &singleflight.Group{}

	// 插件初始化
	pluginInitialize()

	return nil
}

// 插件初始化
func pluginInitialize() {
	// 获取CMDB插件
	server.cmdb = plugin.GetCMDB()
	if server.cmdb == nil {
		log.Warnf("Not Found CMDB Plugin")
	}

	// 获取History插件，注意：插件的配置在bootstrap已经设置好
	server.history = plugin.GetHistory()
	if server.history == nil {
		log.Warnf("Not Found History Log Plugin")
	}

	// 获取限流插件
	server.ratelimit = plugin.GetRatelimit()
	if server.ratelimit == nil {
		log.Warnf("Not found Ratelimit Plugin")
	}

	// 获取DiscoverStatis插件
	server.discoverStatis = plugin.GetDiscoverStatis()
	if server.discoverStatis == nil {
		log.Warnf("Not Found Discover Statis Plugin")
	}

	// 获取服务事件插件
	server.discoverEvent = plugin.GetDiscoverEvent()
	if server.discoverEvent == nil {
		log.Warnf("Not found DiscoverEvent Plugin")
	}

	// 获取鉴权插件
	server.auth = plugin.GetAuth()
	if server.auth == nil {
		log.Warnf("Not found Auth Plugin")
	}
}
