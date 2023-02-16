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

package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func registerSysMetrics() {
	registry.MustRegister([]prometheus.Collector{
		instanceAsyncRegisCost,
		instanceRegisTaskExpire,
		redisReadFailure,
		redisWriteFailure,
		redisAliveStatus,
	}...)

	go func() {
		lastRedisReadFailureReport.Store(time.Now())
		lastRedisWriteFailureReport.Store(time.Now())
		ticker := time.NewTicker(time.Minute)
		for range ticker.C {
			tn := time.Now()
			if tn.Sub(lastRedisReadFailureReport.Load().(time.Time)) > time.Minute {
				redisReadFailure.Set(0)
			}
			if tn.Sub(lastRedisWriteFailureReport.Load().(time.Time)) > time.Minute {
				redisWriteFailure.Set(0)
			}
		}
	}()
}

// ReportInstanceRegisCost Total time to report the short-term registered task of the reporting instance
func ReportInstanceRegisCost(cost time.Duration) {
	instanceAsyncRegisCost.Observe(float64(cost.Milliseconds()))
}

// ReportDropInstanceRegisTask Record the number of registered tasks discarded
func ReportDropInstanceRegisTask() {
	instanceRegisTaskExpire.Inc()
}

// ReportRedisReadFailure report redis exec read operatio failure
func ReportRedisReadFailure() {
	lastRedisReadFailureReport.Store(time.Now())
	redisReadFailure.Inc()
}

// ReportRedisWriteFailure report redis exec write operatio failure
func ReportRedisWriteFailure() {
	lastRedisWriteFailureReport.Store(time.Now())
	redisWriteFailure.Inc()
}

// ReportRedisIsDead report redis alive status is dead
func ReportRedisIsDead() {
	redisAliveStatus.Set(0)
}

// ReportRedisIsAlive report redis alive status is health
func ReportRedisIsAlive() {
	redisAliveStatus.Set(1)
}
