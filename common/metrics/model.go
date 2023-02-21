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

import "time"

type CallMetricType string

const (
	ServerCallMetric        CallMetricType = "server"
	RedisCallMetric         CallMetricType = "redis"
	StoreCallMetric         CallMetricType = "db"
	ProtobufCacheCallMetric CallMetricType = "protobuf"
)

type CallMetric struct {
	Type     CallMetricType
	API      string
	Protocol string
	Code     int
	Times    int
	Success  bool
	Duration time.Duration
	Labels   map[string]string
}

type DiscoveryMetricType string

const (
	ClientMetrics   DiscoveryMetricType = "client"
	ServiceMetrics  DiscoveryMetricType = "service"
	InstanceMetrics DiscoveryMetricType = "instance"
)

type DiscoveryMetric struct {
	Type     DiscoveryMetricType
	Total    int64
	Abnormal int64
	Offline  int64
	Online   int64
	Isolate  int64
	Labels   map[string]string
}

type ConfigMetricType string

const (
	ConfigGroupMetric ConfigMetricType = "config_group"
	FileMetric        ConfigMetricType = "file"
	ReleaseFileMetric ConfigMetricType = "release_file"
)

type ConfigMetrics struct {
	Type    ConfigMetricType
	Total   int64
	Release int64
	Labels  map[string]string
}
