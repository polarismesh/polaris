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
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/naming/batch"
	"github.com/polarismesh/polaris-server/plugin"
	"github.com/polarismesh/polaris-server/store"
	"strconv"
	"sync"
	"time"
)

var (
	server     = new(Server)
	once       = sync.Once{}
	finishInit = false
)

// Server health check main server
type Server struct {
	storage        store.Store
	checkers       map[int32]plugin.HealthChecker
	cacheProvider  *CacheProvider
	timeAdjuster   *TimeAdjuster
	dispatcher     *Dispatcher
	checkScheduler *CheckScheduler
	history        plugin.History
	localHost      string
	bc             *batch.Controller
}

// Initialize 初始化
func Initialize(ctx context.Context, hcOpt *Config, cacheOpen bool) error {
	var err error
	once.Do(func() {
		err = initialize(ctx, hcOpt, cacheOpen)
	})

	if err != nil {
		return err
	}

	finishInit = true
	return nil
}

func initialize(ctx context.Context, hcOpt *Config, cacheOpen bool) error {
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
			if nil == checker {
				return fmt.Errorf("[healthcheck]unknown healthchecker %s", entry.Name)
			}
			server.checkers[int32(checker.Type())] = checker
		}
	}
	var err error
	if server.storage, err = store.GetStore(); nil != err {
		return err
	}
	// 批量控制器
	batchConfig, err := batch.ParseBatchConfig(hcOpt.Batch)
	if err != nil {
		return err
	}
	bc, err := batch.NewBatchCtrlWithConfig(server.storage, nil, plugin.GetAuth(), batchConfig)
	if err != nil {
		log.Errorf("new batch ctrl with config err: %s", err.Error())
		return err
	}
	server.bc = bc
	if server.bc != nil {
		server.bc.Start(ctx)
	}
	server.localHost = hcOpt.LocalHost
	server.history = plugin.GetHistory()
	server.cacheProvider = newCacheProvider(hcOpt.Service)
	server.timeAdjuster = newTimeAdjuster(ctx)
	server.checkScheduler = newCheckScheduler(ctx, hcOpt.SlotNum, hcOpt.MinCheckInterval, hcOpt.MaxCheckInterval)
	server.dispatcher = newDispatcher(ctx)
	return nil
}

// Report report heartbeat request
func (s *Server) Report(ctx context.Context, req *api.Instance) *api.Response {
	return s.doReport(ctx, req)
}

// GetServer 获取已经初始化好的Server
func GetServer() (*Server, error) {
	if !finishInit {
		return nil, errors.New("server has not done InitializeServer")
	}

	return server, nil
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
	checker, ok := s.checkers[int32(insCache.GetHealthCheck().GetType())]
	if !ok {
		return api.NewInstanceResponse(api.HeartbeatTypeNotFound, req)
	}
	queryResp, err := checker.Query(&plugin.QueryRequest{
		InstanceId: insCache.GetId().GetValue(),
		Host:       insCache.GetHost().GetValue(),
		Port:       insCache.GetPort().GetValue(),
	})
	if err != nil {
		return api.NewInstanceRespWithError(api.ExecuteException, err, req)
	}
	req.Service = insCache.GetService()
	req.Namespace = insCache.GetNamespace()
	req.Host = insCache.GetHost()
	req.Port = insCache.GetPort()
	req.VpcId = insCache.GetVpcId()
	req.HealthCheck = insCache.GetHealthCheck()
	req.Metadata["last-heartbeat-timestamp"] = strconv.Itoa(int(queryResp.LastHeartbeatSec))
	req.Metadata["last-heartbeat-time"] = time2String(time.Unix(queryResp.LastHeartbeatSec, 0))
	req.Metadata["system-time"] = time2String(time.Unix(currentTimeSec(), 0))
	return api.NewInstanceResponse(api.ExecuteSuccess, req)
}

// time2String time.Time转为字符串时间
func time2String(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func currentTimeSec() int64 {
	return time.Now().Unix() - server.timeAdjuster.GetDiff()
}
