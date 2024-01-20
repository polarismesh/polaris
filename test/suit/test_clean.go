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

package testsuit

import (
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
)

// TestDataClean 测试套件数据清理
type TestDataClean interface {
	InjectSuit(*DiscoverTestSuit)
	// CleanNamespace
	CleanNamespace(name string)
	// CleanReportClient
	CleanReportClient()
	// CleanAllService
	CleanAllService()
	// CleanService
	CleanService(name, namespace string)
	// CleanServices
	CleanServices(services []*apiservice.Service)
	// CleanInstance
	CleanInstance(instanceID string)
	// CleanCommonRoutingConfig
	CleanCommonRoutingConfig(service string, namespace string)
	// TruncateCommonRoutingConfigV2
	TruncateCommonRoutingConfigV2()
	// CleanCommonRoutingConfigV2
	CleanCommonRoutingConfigV2(rules []*apitraffic.RouteRule)
	// CleanRateLimit
	CleanRateLimit(id string)
	// CleanCircuitBreaker
	CleanCircuitBreaker(id, version string)
	// CleanCircuitBreakerRelation
	CleanCircuitBreakerRelation(name, namespace, ruleID, ruleVersion string)
	// ClearTestDataWhenUseRDS
	ClearTestDataWhenUseRDS() error
	// CleanServiceContract
	CleanServiceContract() error
}
