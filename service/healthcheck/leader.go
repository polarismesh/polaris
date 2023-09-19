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
	"time"

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/plugin"
	"github.com/polarismesh/polaris/store"
)

// LeaderChangeEventHandler process the event when server act as leader
type LeaderChangeEventHandler struct {
	svr              *Server
	cacheProvider    *CacheProvider
	ctx              context.Context
	cancel           context.CancelFunc
	minCheckInterval time.Duration
}

// newLeaderChangeEventHandler
func newLeaderChangeEventHandler(svr *Server) *LeaderChangeEventHandler {
	return &LeaderChangeEventHandler{
		svr:              svr,
		cacheProvider:    svr.cacheProvider,
		minCheckInterval: svr.hcOpt.MinCheckInterval,
	}
}

// PreProcess do preprocess logic for event
func (handler *LeaderChangeEventHandler) PreProcess(ctx context.Context, value any) any {
	return value
}

// OnEvent event trigger
func (handler *LeaderChangeEventHandler) OnEvent(ctx context.Context, i interface{}) error {
	e := i.(store.LeaderChangeEvent)
	if e.Key != store.ElectionKeySelfServiceChecker {
		return nil
	}

	if e.Leader {
		handler.startCheckSelfServiceInstances()
	} else {
		handler.stopCheckSelfServiceInstances()
	}
	return nil
}

// startCheckSelfServiceInstances
func (handler *LeaderChangeEventHandler) startCheckSelfServiceInstances() {
	if handler.ctx != nil {
		log.Warn("[healthcheck] receive unexpected leader state event")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	handler.ctx = ctx
	handler.cancel = cancel
	go func() {
		log.Info("[healthcheck] i am leader, start check health of selfService instances")
		ticker := time.NewTicker(handler.minCheckInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				cacheProvider := handler.cacheProvider
				cacheProvider.selfServiceInstances.Range(func(instanceId string, value ItemWithChecker) {
					handler.doCheckSelfServiceInstance(value.GetInstance())
				})
			case <-ctx.Done():
				log.Info("[healthcheck] stop check health of selfService instances")
				return
			}
		}
	}()
}

// startCheckSelfServiceInstances
func (handler *LeaderChangeEventHandler) stopCheckSelfServiceInstances() {
	if handler.ctx == nil {
		return
	}
	handler.cancel()
	handler.ctx = nil
	handler.cancel = nil
}

// startCheckSelfServiceInstances
func (handler *LeaderChangeEventHandler) doCheckSelfServiceInstance(cachedInstance *model.Instance) {
	hcEnable, checker := handler.cacheProvider.isHealthCheckEnable(cachedInstance.Proto)
	if !hcEnable {
		log.Warnf("[Health Check][Check] selfService instance %s:%d not enable healthcheck",
			cachedInstance.Host(), cachedInstance.Port())
		return
	}

	request := &plugin.CheckRequest{
		QueryRequest: plugin.QueryRequest{
			InstanceId: cachedInstance.ID(),
			Host:       cachedInstance.Host(),
			Port:       cachedInstance.Port(),
			Healthy:    cachedInstance.Healthy(),
		},
		CurTimeSec:        handler.svr.currentTimeSec,
		ExpireDurationSec: getExpireDurationSec(cachedInstance.Proto),
	}
	checkResp, err := checker.Check(request)
	if err != nil {
		log.Errorf("[Health Check][Check]fail to check selfService instance %s:%d, id is %s, err is %v",
			cachedInstance.Host(), cachedInstance.Port(), cachedInstance.ID(), err)
		return
	}
	if !checkResp.StayUnchanged {
		code := setInsDbStatus(handler.svr, cachedInstance, checkResp.Healthy, checkResp.LastHeartbeatTimeSec)
		if checkResp.Healthy {
			// from unhealthy to healthy
			log.Infof(
				"[Health Check][Check]selfService instance change from unhealthy to healthy, id is %s, address is %s:%d",
				cachedInstance.ID(), cachedInstance.Host(), cachedInstance.Port())
		} else {
			// from healthy to unhealthy
			log.Infof(
				"[Health Check][Check]selfService instance change from healthy to unhealthy, id is %s, address is %s:%d",
				cachedInstance.ID(), cachedInstance.Host(), cachedInstance.Port())
		}
		if code != apimodel.Code_ExecuteSuccess {
			log.Errorf(
				"[Health Check][Check]fail to update selfService instance, id is %s, address is %s:%d, code is %d",
				cachedInstance.ID(), cachedInstance.Host(), cachedInstance.Port(), code)
		}
	}
}
