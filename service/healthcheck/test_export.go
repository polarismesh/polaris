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

	"github.com/polarismesh/polaris/common/eventhub"
	"github.com/polarismesh/polaris/plugin"
	"github.com/polarismesh/polaris/service/batch"
	"github.com/polarismesh/polaris/store"
)

func TestInitialize(ctx context.Context, hcOpt *Config, cacheOpen bool, bc *batch.Controller,
	storage store.Store) (*Server, error) {

	testServer := new(Server)
	testServer.hcOpt = hcOpt

	if !cacheOpen {
		return nil, fmt.Errorf("[healthcheck]cache not open")
	}
	hcOpt.SetDefault()
	if hcOpt.Open {
		if len(hcOpt.Checkers) > 0 {
			testServer.checkers = make(map[int32]plugin.HealthChecker, len(hcOpt.Checkers))
			for _, entry := range hcOpt.Checkers {
				checker := plugin.GetHealthChecker(entry.Name, &entry)
				if checker == nil {
					return nil, fmt.Errorf("[healthcheck]unknown healthchecker %s", entry.Name)
				}
				// The same health type check plugin can only exist in one
				_, exist := testServer.checkers[int32(checker.Type())]
				if exist {
					return nil, fmt.Errorf("[healthcheck]duplicate healthchecker %s, checkType %d",
						entry.Name, checker.Type())
				}
				testServer.checkers[int32(checker.Type())] = checker
				if nil == testServer.defaultChecker {
					testServer.defaultChecker = checker
				}
			}
		} else {
			return nil, fmt.Errorf("[healthcheck]no checker config")
		}
	}
	var err error
	if testServer.storage, err = store.GetStore(); err != nil {
		return nil, err
	}

	testServer.bc = bc

	testServer.localHost = hcOpt.LocalHost
	testServer.history = plugin.GetHistory()
	testServer.discoverEvent = plugin.GetDiscoverEvent()

	testServer.cacheProvider = newCacheProvider(hcOpt.Service, testServer)
	testServer.timeAdjuster = newTimeAdjuster(ctx, testServer.storage)
	testServer.checkScheduler = newCheckScheduler(ctx, hcOpt.SlotNum, hcOpt.MinCheckInterval,
		hcOpt.MaxCheckInterval, hcOpt.ClientCheckInterval, hcOpt.ClientCheckTtl)
	testServer.dispatcher = newDispatcher(ctx, testServer)

	return testServer, testServer.run(ctx)
}

func TestDestroy() {
	eventhub.Unsubscribe(eventhub.LeaderChangeEventTopic, "selfServiceChecker")
	eventhub.Unsubscribe(eventhub.InstanceEventTopic, "instanceHealthChecker")
}
