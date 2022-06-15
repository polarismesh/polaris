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
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/polarismesh/polaris-server/cache"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
	commontime "github.com/polarismesh/polaris-server/common/time"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/plugin"
	"github.com/polarismesh/polaris-server/service/batch"
	"github.com/polarismesh/polaris-server/store"
)

var (
	server     = new(Server)
	once       = sync.Once{}
	finishInit = false
)

// Server health checks the main server
type Server struct {
	storage        store.Store
	checkers       map[int32]plugin.HealthChecker
	cacheProvider  *CacheProvider
	timeAdjuster   *TimeAdjuster
	dispatcher     *Dispatcher
	checkScheduler *CheckScheduler
	history        plugin.History
	discoverEvent  plugin.DiscoverChannel
	localHost      string
	discoverCh     chan eventWrapper
	bc             *batch.Controller
	serviceCache   cache.ServiceCache
}

// Initialize 初始化
func Initialize(ctx context.Context, hcOpt *Config, cacheOpen bool, bc *batch.Controller) error {
	var err error
	once.Do(func() {
		err = initialize(ctx, hcOpt, cacheOpen, bc)
	})

	if err != nil {
		return err
	}

	finishInit = true
	return nil
}

func initialize(ctx context.Context, hcOpt *Config, cacheOpen bool, bc *batch.Controller) error {
	if !hcOpt.Open {
		return nil
	}
	if !cacheOpen {
		return fmt.Errorf("[healthcheck]cache not open")
	}
	hcOpt.SetDefault()
	if len(hcOpt.Checkers) > 0 {
		server.checkers = make(map[int32]plugin.HealthChecker, len(hcOpt.Checkers))
		for _, entry := range hcOpt.Checkers {
			checker := plugin.GetHealthChecker(entry.Name, &entry)
			if checker == nil {
				return fmt.Errorf("[healthcheck]unknown healthchecker %s", entry.Name)
			}
			// The same health type check plugin can only exist in one
			_, exist := server.checkers[int32(checker.Type())]
			if exist {
				return fmt.Errorf("[healthcheck]duplicate healthchecker %s, checkType %d", entry.Name, checker.Type())
			}

			server.checkers[int32(checker.Type())] = checker
		}
	}
	var err error
	if server.storage, err = store.GetStore(); err != nil {
		return err
	}

	server.bc = bc

	server.localHost = hcOpt.LocalHost
	server.history = plugin.GetHistory()
	server.discoverEvent = plugin.GetDiscoverEvent()

	server.cacheProvider = newCacheProvider(hcOpt.Service)
	server.timeAdjuster = newTimeAdjuster(ctx)
	server.checkScheduler = newCheckScheduler(ctx, hcOpt.SlotNum, hcOpt.MinCheckInterval, hcOpt.MaxCheckInterval)
	server.dispatcher = newDispatcher(ctx)

	server.discoverCh = make(chan eventWrapper, 32)
	go server.receiveEventAndPush()

	return nil
}

// Report heartbeat request
func (s *Server) Report(ctx context.Context, req *api.Instance) *api.Response {
	return s.doReport(ctx, req)
}

// Report report heartbeat request by client
func (s *Server) ReportByClient(ctx context.Context, req *api.Client) *api.Response {
	return s.doReportByClient(ctx, req)
}

// GetServer 获取已经初始化好的Server
func GetServer() (*Server, error) {
	if !finishInit {
		return nil, errors.New("server has not done InitializeServer")
	}

	return server, nil
}

// SetServiceCache 设置服务缓存
func (s *Server) SetServiceCache(serviceCache cache.ServiceCache) {
	s.serviceCache = serviceCache
}

// CacheProvider get cache provider
func (s *Server) CacheProvider() (*CacheProvider, error) {
	if !finishInit {
		return nil, errors.New("cache provider has not done InitializeServer")
	}
	return s.cacheProvider, nil
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

// PublishDiscoverEvent 发布服务事件
func (s *Server) PublishDiscoverEvent(serviceID string, event model.DiscoverEvent) {
	if s.discoverEvent == nil {
		return
	}
	s.discoverCh <- eventWrapper{
		ServiceID: serviceID,
		Event:     event,
	}
}

func (s *Server) receiveEventAndPush() {
	if s.discoverEvent == nil {
		return
	}

	for wrapper := range s.discoverCh {
		var (
			svcID   = wrapper.ServiceID
			event   = wrapper.Event
			service *model.Service
		)
		for {
			service = s.serviceCache.GetServiceByID(svcID)
			if service == nil {
				time.Sleep(500 * time.Millisecond)
				continue
			}
			break
		}
		event.Namespace = service.Namespace
		event.Service = service.Name

		s.discoverEvent.PublishEvent(event)
	}
}

// GetLastHeartbeat 获取上一次心跳的时间
func (s *Server) GetLastHeartbeat(req *api.Instance) *api.Response {
	if len(s.checkers) == 0 {
		return api.NewResponse(api.HealthCheckNotOpen)
	}
	id, errRsp := checkHeartbeatInstance(req)
	if errRsp != nil {
		return errRsp
	}
	req.Id = utils.NewStringValue(id)
	insCache := s.cacheProvider.GetInstance(id)
	if insCache == nil {
		return api.NewInstanceResponse(api.NotFoundResource, req)
	}
	checker, ok := s.checkers[int32(insCache.HealthCheck().GetType())]
	if !ok {
		return api.NewInstanceResponse(api.HeartbeatTypeNotFound, req)
	}
	queryResp, err := checker.Query(&plugin.QueryRequest{
		InstanceId: insCache.ID(),
		Host:       insCache.Host(),
		Port:       insCache.Port(),
	})
	if err != nil {
		return api.NewInstanceRespWithError(api.ExecuteException, err, req)
	}
	req.Service = insCache.Proto.GetService()
	req.Namespace = insCache.Proto.GetNamespace()
	req.Host = insCache.Proto.GetHost()
	req.Port = insCache.Proto.Port
	req.VpcId = insCache.Proto.GetVpcId()
	req.HealthCheck = insCache.Proto.GetHealthCheck()
	req.Metadata["last-heartbeat-timestamp"] = strconv.Itoa(int(queryResp.LastHeartbeatSec))
	req.Metadata["last-heartbeat-time"] = commontime.Time2String(time.Unix(queryResp.LastHeartbeatSec, 0))
	req.Metadata["system-time"] = commontime.Time2String(time.Unix(currentTimeSec(), 0))
	return api.NewInstanceResponse(api.ExecuteSuccess, req)
}

func currentTimeSec() int64 {
	return time.Now().Unix() - server.timeAdjuster.GetDiff()
}

type eventWrapper struct {
	ServiceID string
	Event     model.DiscoverEvent
}
