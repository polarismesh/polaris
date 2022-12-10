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
	"github.com/prometheus/client_golang/prometheus"

	"github.com/polarismesh/polaris/common/utils"
)

var (
	instanceAsyncRegisCost = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "instance_regis_cost_time",
		Help: "instance regis cost time",
		ConstLabels: map[string]string{
			"polaris_server_instance": utils.LocalHost,
		},
	})

	instanceRegisTaskExpire = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "instance_regis_task_expire",
		Help: "instance regis task expire that server drop it",
		ConstLabels: map[string]string{
			"polaris_server_instance": utils.LocalHost,
		},
	})
)

// redis metrics
var (
	redisReadFailure = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "redis_read_failure",
		Help: "polaris exec redis read operation success",
		ConstLabels: map[string]string{
			"polaris_server_instance": utils.LocalHost,
		},
	})

	redisWriteFailure = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "redis_write_failure",
		Help: "polaris exec redis write operation success",
		ConstLabels: map[string]string{
			"polaris_server_instance": utils.LocalHost,
		},
	})

	redisAliveStatus = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "redis_alive_status",
		Help: "polaris redis alive status",
		ConstLabels: map[string]string{
			"polaris_server_instance": utils.LocalHost,
		},
	})
)
