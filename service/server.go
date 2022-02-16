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
	"time"

	"github.com/modern-go/reflect2"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	"github.com/polarismesh/polaris-server/auth"
	"github.com/polarismesh/polaris-server/cache"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/plugin"
	"github.com/polarismesh/polaris-server/service/batch"
	"github.com/polarismesh/polaris-server/store"
)

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

	creareServiceSingle *singleflight.Group

	hooks []ResourceHook
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

func (s *Server) SetResourceHook(hooks []ResourceHook) {
	s.hooks = hooks
}

func (s *Server) afterNamespaceResource(ctx context.Context, req *api.Namespace, save *model.Namespace, remove bool) {
	event := &ResourceEvent{
		ReqNamespace: req,
		Namespace:    save,
		IsRemove:     remove,
	}

	for index := range s.hooks {
		hook := s.hooks[index]
		hook.After(ctx, model.RNamespace, event)
	}
}

func (s *Server) afterServiceResource(ctx context.Context, req *api.Service, save *model.Service, remove bool) {
	event := &ResourceEvent{
		ReqService: req,
		Service:    save,
		IsRemove:   remove,
	}

	for index := range s.hooks {
		hook := s.hooks[index]
		hook.After(ctx, model.RService, event)
	}
}
