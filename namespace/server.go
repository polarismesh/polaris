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

package namespace

import (
	"context"

	"github.com/polarismesh/polaris-server/cache"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/plugin"
	"github.com/polarismesh/polaris-server/store"
)

type Server struct {
	storage store.Store

	caches *cache.CacheManager

	cfg     Config
	auth    plugin.Auth
	history plugin.History

	hooks []ResourceHook
}

func (s *Server) afterNamespaceResource(ctx context.Context, req *api.Namespace, save *model.Namespace,
	remove bool) error {

	event := &ResourceEvent{
		ReqNamespace: req,
		Namespace:    save,
		IsRemove:     remove,
	}

	for index := range s.hooks {
		hook := s.hooks[index]
		if err := hook.After(ctx, model.RNamespace, event); err != nil {
			return err
		}
	}

	return nil
}

// RecordHistory server对外提供history插件的简单封装
func (svr *Server) RecordHistory(entry *model.RecordEntry) {
	// 如果插件没有初始化，那么不记录history
	if svr.history == nil {
		return
	}
	// 如果数据为空，则不需要打印了
	if entry == nil {
		return
	}

	// 调用插件记录history
	svr.history.Record(entry)
}

// SetResourceHooks 返回Cache
func (s *Server) SetResourceHooks(hooks ...ResourceHook) {
	s.hooks = hooks
}
