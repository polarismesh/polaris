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

package utils

import (
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/polarismesh/polaris/apiserver"
	"github.com/polarismesh/polaris/common/log"
)

// GetConfigClientOpenMethod .
func GetConfigClientOpenMethod(protocol string) (map[string]bool, error) {
	openMethods := []string{
		"GetConfigFile",
		"CreateConfigFile",
		"UpdateConfigFile",
		"PublishConfigFile",
		"WatchConfigFiles",
		"GetConfigFileMetadataList",
		"UpsertAndPublishConfigFile",
		"Discover",
	}

	openMethod := make(map[string]bool)

	for _, item := range openMethods {
		method := "/v1.PolarisConfig" + strings.ToUpper(protocol) + "/" + item
		openMethod[method] = true
	}

	return openMethod, nil
}

// GetDiscoverClientOpenMethod 获取客户端openMethod
func GetDiscoverClientOpenMethod(include []string, protocol string) (map[string]bool, error) {
	clientAccess := make(map[string][]string)
	clientAccess[apiserver.DiscoverAccess] = []string{"Discover", "ReportClient", "ReportServiceContract", "GetServiceContract"}
	clientAccess[apiserver.RegisterAccess] = []string{"RegisterInstance", "DeregisterInstance"}
	clientAccess[apiserver.HealthcheckAccess] = []string{"Heartbeat", "BatchHeartbeat", "BatchGetHeartbeat", "BatchDelHeartbeat"}

	openMethod := make(map[string]bool)
	// 如果为空，开启全部接口
	if len(include) == 0 {
		for key := range clientAccess {
			include = append(include, key)
		}
	}

	for _, item := range include {
		if methods, ok := clientAccess[item]; ok {
			for _, method := range methods {
				recordMethod := "/v1.Polaris" + strings.ToUpper(protocol) + "/" + method
				if item == apiserver.HealthcheckAccess && method != "Heartbeat" {
					recordMethod = "/v1.PolarisHeartbeat" + strings.ToUpper(protocol) + "/" + method
				}
				if method == "ReportServiceContract" || method == "GetServiceContract" {
					recordMethod = "/v1.PolarisServiceContract" + strings.ToUpper(protocol) + "/" + method
				}
				openMethod[recordMethod] = true
			}
		} else {
			log.Errorf("method %s does not exist in %sserver client access", item, protocol)
			return nil, fmt.Errorf("method %s does not exist in %sserver client access", item, protocol)
		}
	}
	log.Info("[APIServer] client open method info", zap.Any("openMethod", openMethod))
	return openMethod, nil
}
