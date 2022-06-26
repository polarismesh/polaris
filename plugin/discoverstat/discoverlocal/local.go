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
	"fmt"
	"time"

	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/plugin"
)

// init 注册服务发现统计插件
func init() {
	d := &DiscoverStatisWorker{}
	plugin.RegisterPlugin(d.Name(), d)
}

// DiscoverStatisWorker 服务发现统计插件
type DiscoverStatisWorker struct {
	interval time.Duration

	dcc chan *DiscoverCall
	dcs *DiscoverCallStatis
}

// Name 获取插件名称
func (d *DiscoverStatisWorker) Name() string {
	return "discoverLocal"
}

// Initialize 初始化服务发现统计插件
func (d *DiscoverStatisWorker) Initialize(conf *plugin.ConfigEntry) error {
	// 设置打印周期
	interval, _ := conf.Option["interval"].(int)
	d.interval = time.Duration(interval) * time.Second

	outputPath, _ := conf.Option["outputPath"].(string)

	// 初始化
	d.dcc = make(chan *DiscoverCall, 1024)
	d.dcs = &DiscoverCallStatis{
		statis: make(map[Service]time.Time),
		logger: newLogger(outputPath + "/" + "discovercall.log"),
	}

	go d.Run()

	return nil
}

// Destroy 销毁服务发现统计插件
func (d *DiscoverStatisWorker) Destroy() error {
	return nil
}

// AddDiscoverCall 上报请求
func (d *DiscoverStatisWorker) AddDiscoverCall(service, namespace string, time time.Time) error {
	select {
	case d.dcc <- &DiscoverCall{
		service:   service,
		namespace: namespace,
		time:      time,
	}:
	default:
		log.Errorf("[DiscoverStatis] service: %s, namespace: %s is not captured", service, namespace)
		return fmt.Errorf("[DiscoverStatis] service: %s, namespace: %s is not captured", service, namespace)
	}
	return nil
}

// Run 运行服务发现统计插件
func (d *DiscoverStatisWorker) Run() {
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.dcs.log()
		case dc := <-d.dcc:
			d.dcs.add(dc)
		}
	}
}
