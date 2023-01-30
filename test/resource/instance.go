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

package resource

import (
	"fmt"

	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	"github.com/polarismesh/polaris/common/utils"
)

const (
	instanceHost = "%v.%v.%v.%v"
)

/**
 * @brief 创建测试服务实例
 */
func CreateInstances(service *apiservice.Service) []*apiservice.Instance {
	var instances []*apiservice.Instance
	for index := 0; index < 2; index++ {
		host := fmt.Sprintf(instanceHost, index, index, index, index)

		instance := &apiservice.Instance{
			Service:   service.GetName(),
			Namespace: service.GetNamespace(),
			Host:      utils.NewStringValue(host),
			Port:      utils.NewUInt32Value(8),
			Protocol:  utils.NewStringValue("test"),
			Version:   utils.NewStringValue("8.8.8"),
			Priority:  utils.NewUInt32Value(8),
			Weight:    utils.NewUInt32Value(8),
			HealthCheck: &apiservice.HealthCheck{
				Type: apiservice.HealthCheck_HEARTBEAT,
				Heartbeat: &apiservice.HeartbeatHealthCheck{
					Ttl: utils.NewUInt32Value(8),
				},
			},
			Healthy:      utils.NewBoolValue(false),
			Isolate:      utils.NewBoolValue(false),
			Metadata:     map[string]string{"test": "test"},
			LogicSet:     utils.NewStringValue("test"),
			ServiceToken: service.GetToken(),
		}
		instances = append(instances, instance)
	}

	return instances
}

/**
 * @brief 更新测试服务实例
 */
func UpdateInstances(instances []*apiservice.Instance) {
	for _, instance := range instances {
		instance.Protocol = utils.NewStringValue("update")
		instance.Version = utils.NewStringValue("4.4.4")
		instance.Priority = utils.NewUInt32Value(4)
		instance.Weight = utils.NewUInt32Value(4)
		instance.Metadata = map[string]string{"update": "update"}
		instance.LogicSet = utils.NewStringValue("update")
	}
}
