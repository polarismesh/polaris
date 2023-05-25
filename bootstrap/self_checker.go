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

package bootstrap

import (
	"context"
	"sync"
	"time"

	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/service"
	"github.com/polarismesh/polaris/service/healthcheck"
)

type SelfHeathChecker struct {
	instances   []*apiservice.Instance
	interval    int
	cancel      context.CancelFunc
	wg          *sync.WaitGroup
	discoverSvr service.DiscoverServer
	hcServer    *healthcheck.Server
}

func NewSelfHeathChecker(instances []*apiservice.Instance, interval int) (*SelfHeathChecker, error) {
	hcServer, err := healthcheck.GetServer()
	if nil != err {
		return nil, err
	}
	discoverSvr, err := service.GetOriginServer()
	if nil != err {
		return nil, err
	}
	for _, instance := range instances {
		log.Infof("scheduled check for instance %s:%d",
			instance.GetHost().GetValue(), instance.GetPort().GetValue())
	}
	return &SelfHeathChecker{
		instances:   instances,
		interval:    interval,
		discoverSvr: discoverSvr,
		hcServer:    hcServer,
	}, nil
}

func (s *SelfHeathChecker) Start() {
	s.wg = &sync.WaitGroup{}
	s.wg.Add(1)
	var ctx context.Context
	ctx, s.cancel = context.WithCancel(context.Background())
	ticker := time.NewTicker(time.Duration(s.interval) * time.Second)
	for {
		select {
		case <-ctx.Done():
			log.Info("[Bootstrap] server health check has been terminated")
			s.wg.Done()
			ticker.Stop()
			return
		case <-ticker.C:
			for _, instance := range s.instances {
				rsp := s.hcServer.Report(context.Background(), instance)

				switch rsp.GetCode().GetValue() {
				case api.ExecuteSuccess:
					continue
				case api.NotFoundResource:
					// 这里可能实例被错误摘除了，这里重新触发一次重注册流程，确保核心流程不受影响
					log.Infof("[Bootstrap] heartbeat not founf instance for %s:%d, code is %d, try re-register",
						instance.GetHost().GetValue(), instance.GetPort().GetValue(), rsp.GetCode().GetValue())
					resp := s.discoverSvr.CreateInstances(genContext(), []*apiservice.Instance{instance})
					if resp.GetCode().GetValue() != api.ExecuteSuccess {
						log.Errorf("[Bootstrap] re-register fail for %s:%d, code is %d, info %s",
							instance.GetHost().GetValue(), instance.GetPort().GetValue(),
							resp.GetCode().GetValue(), resp.GetInfo().GetValue())
					}
				default:
					log.Errorf("[Bootstrap] heartbeat fail for %s:%d, code is %d, info %s",
						instance.GetHost().GetValue(), instance.GetPort().GetValue(),
						rsp.GetCode().GetValue(), rsp.GetInfo().GetValue())
				}
			}
		}
	}
}

func (s *SelfHeathChecker) Stop() {
	s.cancel()
	s.wg.Wait()
}
