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

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/service/healthcheck"
)

type SelfHeathChecker struct {
	instances []*api.Instance
	interval  int
	cancel    context.CancelFunc
	wg        *sync.WaitGroup
	hcServer  *healthcheck.Server
}

func NewSelfHeathChecker(instances []*api.Instance, interval int) (*SelfHeathChecker, error) {
	hcServer, err := healthcheck.GetServer()
	if nil != err {
		return nil, err
	}
	for _, instance := range instances {
		log.Infof("scheduled check for instance %s:%d",
			instance.GetHost().GetValue(), instance.GetPort().GetValue())
	}
	return &SelfHeathChecker{
		instances: instances,
		interval:  interval,
		hcServer:  hcServer,
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
				if rsp.GetCode().GetValue() != api.ExecuteSuccess {
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
