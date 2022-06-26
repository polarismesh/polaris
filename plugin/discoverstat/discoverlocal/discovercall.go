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

package discoverlocal

import (
	"bytes"
	"time"

	"go.uber.org/zap"

	commontime "github.com/polarismesh/polaris-server/common/time"
)

// DiscoverCall 服务发现统计
type DiscoverCall struct {
	service   string
	namespace string
	time      time.Time
}

// Service 服务
type Service struct {
	name      string
	namespace string
}

// DiscoverCallStatis 服务发现统计条目
type DiscoverCallStatis struct {
	statis map[Service]time.Time

	logger *zap.Logger
}

// add 添加服务发现统计数据
func (d *DiscoverCallStatis) add(dc *DiscoverCall) {
	service := Service{
		name:      dc.service,
		namespace: dc.namespace,
	}

	d.statis[service] = dc.time
}

// log 打印服务发现统计
func (d *DiscoverCallStatis) log() {
	if len(d.statis) == 0 {
		return
	}

	var buffer bytes.Buffer
	for service, t := range d.statis {
		buffer.WriteString("service=")
		buffer.WriteString(service.name)
		buffer.WriteString(";")
		buffer.WriteString("namespace=")
		buffer.WriteString(service.namespace)
		buffer.WriteString(";")
		buffer.WriteString("visitTime=")
		buffer.WriteString(commontime.Time2String(t))
		buffer.WriteString("\n")
	}

	d.logger.Info(buffer.String())
	// Reset for every tick
	d.statis = make(map[Service]time.Time)
}
