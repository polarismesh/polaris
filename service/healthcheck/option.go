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

package healthcheck

import (
	"context"
	"fmt"

	"github.com/polarismesh/polaris/cache"
	"github.com/polarismesh/polaris/common/eventhub"
	"github.com/polarismesh/polaris/plugin"
	"github.com/polarismesh/polaris/service/batch"
	"github.com/polarismesh/polaris/store"
)

// Server health checks the main server
// type Server struct {
// 	hcOpt                *Config
// 	localHost            string
// 	bc                   *batch.Controller
// }

type serverOption func(svr *Server) error

// WithStore .
func WithStore(s store.Store) serverOption {
	return func(svr *Server) error {
		svr.storage = s
		return nil
	}
}

// WithBatchController .
func WithBatchController(ba *batch.Controller) serverOption {
	return func(svr *Server) error {
		svr.bc = ba
		return nil
	}
}

// withChecker .
func withChecker() serverOption {
	return func(svr *Server) error {
		hcOpt := svr.hcOpt
		if hcOpt.IsOpen() && len(hcOpt.Checkers) == 0 {
			return fmt.Errorf("[healthcheck]no checker config")
		}

		svr.checkers = make(map[int32]plugin.HealthChecker, len(hcOpt.Checkers))
		for _, entry := range hcOpt.Checkers {
			checker := plugin.GetHealthChecker(entry.Name, &entry)
			if checker == nil {
				return fmt.Errorf("[healthcheck]unknown healthchecker %s", entry.Name)
			}
			// The same health type check plugin can only exist in one
			_, exist := svr.checkers[int32(checker.Type())]
			if exist {
				return fmt.Errorf("[healthcheck]duplicate healthchecker %s, checkType %d", entry.Name, checker.Type())
			}
			svr.checkers[int32(checker.Type())] = checker
			if nil == svr.defaultChecker {
				svr.defaultChecker = checker
			}
		}
		return nil
	}
}

// withCacheProvider .
func withCacheProvider() serverOption {
	return func(svr *Server) error {
		svr.cacheProvider = newCacheProvider(svr.hcOpt.Service, svr)
		return nil
	}
}

// withCheckScheduler .
func withCheckScheduler(cs *CheckScheduler) serverOption {
	return func(svr *Server) error {
		svr.checkScheduler = cs
		cs.svr = svr
		return nil
	}
}

// withDispatcher .
func withDispatcher(ctx context.Context) serverOption {
	return func(svr *Server) error {
		svr.dispatcher = newDispatcher(ctx, svr)
		return nil
	}
}

// WithTimeAdjuster .
func WithTimeAdjuster(adjuster *TimeAdjuster) serverOption {
	return func(svr *Server) error {
		svr.timeAdjuster = adjuster
		return nil
	}
}

// WithCache .
func WithCache(cacheMgr cache.CacheManager) serverOption {
	return func(svr *Server) error {
		svr.serviceCache = cacheMgr.Service()
		svr.instanceCache = cacheMgr.Instance()
		return nil
	}
}

// withSubscriber .
func withSubscriber(ctx context.Context) serverOption {
	return func(svr *Server) error {
		svr.subCtxs = make([]*eventhub.SubscribtionContext, 0, 4)

		subCtx, err := eventhub.SubscribeWithFunc(eventhub.CacheInstanceEventTopic,
			svr.cacheProvider.handleInstanceCacheEvent)
		if err != nil {
			return err
		}
		svr.subCtxs = append(svr.subCtxs, subCtx)
		subCtx, err = eventhub.SubscribeWithFunc(eventhub.CacheClientEventTopic,
			svr.cacheProvider.handleClientCacheEvent)
		if err != nil {
			return err
		}
		svr.subCtxs = append(svr.subCtxs, subCtx)

		leaderChangeEventHandler := newLeaderChangeEventHandler(svr)
		subCtx, err = eventhub.Subscribe(eventhub.LeaderChangeEventTopic, leaderChangeEventHandler)
		if err != nil {
			return err
		}
		svr.subCtxs = append(svr.subCtxs, subCtx)

		resourceEventHandler := newResourceHealthCheckHandler(ctx, svr)
		// 监听服务实例的删除事件，然后清理心跳 key 数据
		subCtx, err = eventhub.Subscribe(eventhub.InstanceEventTopic, resourceEventHandler)
		if err != nil {
			return err
		}
		svr.subCtxs = append(svr.subCtxs, subCtx)

		// 监听客户端实例的删除事件，然后清理心跳 key 数据
		subCtx, err = eventhub.Subscribe(eventhub.ClientEventTopic, resourceEventHandler)
		if err != nil {
			return err
		}
		svr.subCtxs = append(svr.subCtxs, subCtx)

		if err := svr.storage.StartLeaderElection(store.ElectionKeySelfServiceChecker); err != nil {
			return err
		}
		return nil
	}
}
