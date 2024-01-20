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
	"fmt"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	LabelServerNode       = "polaris_server_instance"
	LabelNamespace        = "namespace"
	LabelService          = "service"
	LabelGroup            = "group"
	LabelVersion          = "version"
	LabelApi              = "api"
	LabelApiType          = "api_type"
	LabelProtocol         = "protocol"
	LabelErrCode          = "err_code"
	labelCacheType        = "cache_type"
	labelCacheUpdateCount = "cache_update_count"
	labelBatchJobLabel    = "batch_label"
)

// CallMetricType .
type CallMetricType string

const (
	// SystemCallMetric Time consuming statistics of some asynchronous tasks inside
	SystemCallMetric CallMetricType = "system"
	// ServerCallMetric Apiserver-layer interface call consumption statistics
	ServerCallMetric CallMetricType = "api"
	// RedisCallMetric Redis call time consumption statistics
	RedisCallMetric CallMetricType = "redis"
	// StoreCallMetric Store call time consumption statistics
	StoreCallMetric CallMetricType = "store"
	// ProtobufCacheCallMetric PB encode cache call/hit statistics
	ProtobufCacheCallMetric CallMetricType = "pbCacheCall"
	// XDSResourceBuildCallMetric
	XDSResourceBuildCallMetric CallMetricType = "xds"
)

type TrafficDirection string

const (
	// TrafficDirectionInBound .
	TrafficDirectionInBound TrafficDirection = "INBOUND"
	// TrafficDirectionOutBound .
	TrafficDirectionOutBound TrafficDirection = "OUTBOUND"
)

type CallMetric struct {
	Type             CallMetricType
	API              string
	Protocol         string
	Code             int
	Times            int
	Success          bool
	Duration         time.Duration
	Labels           map[string]string
	TrafficDirection TrafficDirection
}

func (m CallMetric) GetLabels() map[string]string {
	if len(m.Labels) == 0 {
		m.Labels = map[string]string{}
	}
	m.Labels[LabelApi] = m.API
	m.Labels[LabelProtocol] = m.Protocol
	m.Labels[LabelErrCode] = strconv.FormatInt(int64(m.Code), 10)
	return m.Labels
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

func ResourceOfConfigFileList(group string) string {
	return "CONFIG_FILE_LIST:" + group
}

func ResourceOfConfigFile(group, name string) string {
	return "CONFIG_FILE:" + group + "/" + name
}

const (
	ActionGetConfigFile           = "GET_CONFIG_FILE"
	ActionListConfigFiles         = "LIST_CONFIG_FILES"
	ActionListConfigGroups        = "LIST_CONFIG_GROUPS"
	ActionPublishConfigFile       = "PUBLISH_CONFIG_FILE"
	ActionDiscoverInstance        = "DISCOVER_INSTANCE"
	ActionDiscoverServices        = "DISCOVER_SERVICES"
	ActionDiscoverRouterRule      = "DISCOVER_ROUTER_RULE"
	ActionDiscoverRateLimit       = "DISCOVER_RATE_LIMIT"
	ActionDiscoverCircuitBreaker  = "DISCOVER_CIRCUIT_BREAKER"
	ActionDiscoverFaultDetect     = "DISCOVER_FAULT_DETECT"
	ActionDiscoverServiceContract = "DISCOVER_SERVICE_CONTRACT"
)

type ClientDiscoverMetric struct {
	ClientIP  string
	Action    string
	Namespace string
	Resource  string
	Revision  string
	Timestamp int64
	CostTime  int64
	Success   bool
}

func (c ClientDiscoverMetric) String() string {
	revision := c.Revision
	if revision == "" {
		revision = "-"
	}
	return fmt.Sprintf("%s|%s|%s|%s|%s|%s|%dms|%+v", c.ClientIP, c.Action, c.Namespace, c.Resource,
		revision, time.Unix(c.Timestamp/1000, 0).Format(time.DateTime), c.CostTime, c.Success)
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

var (
	clientInstanceTotal   prometheus.Gauge
	serviceCount          *prometheus.GaugeVec
	serviceOnlineCount    *prometheus.GaugeVec
	serviceAbnormalCount  *prometheus.GaugeVec
	serviceOfflineCount   *prometheus.GaugeVec
	instanceCount         *prometheus.GaugeVec
	instanceOnlineCount   *prometheus.GaugeVec
	instanceAbnormalCount *prometheus.GaugeVec
	instanceIsolateCount  *prometheus.GaugeVec
)

var (
	configGroupTotal       *prometheus.GaugeVec
	configFileTotal        *prometheus.GaugeVec
	releaseConfigFileTotal *prometheus.GaugeVec
)

// instance astbc registry metrics
var (
	// instanceAsyncRegisCost 实例异步注册任务耗费时间
	instanceAsyncRegisCost prometheus.Histogram
	// instanceRegisTaskExpire 实例异步注册任务超时无效事件
	instanceRegisTaskExpire prometheus.Counter
	redisReadFailure        prometheus.Gauge
	redisWriteFailure       prometheus.Gauge
	redisAliveStatus        prometheus.Gauge
	// discoveryConnTotal 服务发现客户端链接数量
	discoveryConnTotal prometheus.Gauge
	// configurationConnTotal 配置中心客户端链接数量
	configurationConnTotal prometheus.Gauge
	// sdkClientTotal 客户端链接数量
	sdkClientTotal  prometheus.Gauge
	cacheUpdateCost *prometheus.HistogramVec
	// batchJobUnFinishJobs .
	batchJobUnFinishJobs *prometheus.GaugeVec
)
