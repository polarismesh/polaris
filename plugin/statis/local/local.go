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

package local

import (
	"time"

	"github.com/polarismesh/polaris-server/plugin"
)

/**
 * @brief 注册统计插件
 */
func init() {
	s := &StatisWorker{}
	plugin.RegisterPlugin(s.Name(), s)
}

/**
 * StatisWorker 本地统计插件
 */
type StatisWorker struct {
	interval time.Duration

	acc chan *APICall
	acs *APICallStatis
}

/**
 * Name 获取统计插件名称
 */
func (s *StatisWorker) Name() string {
	return "local"
}

/**
 * Initialize 初始化统计插件
 */
func (s *StatisWorker) Initialize(conf *plugin.ConfigEntry) error {

	// 设置统计打印周期
	interval := conf.Option["interval"].(int)
	s.interval = time.Duration(interval) * time.Second

	outputPath := conf.Option["outputPath"].(string)

	// 初始化接口调用统计
	s.acc = make(chan *APICall, 1024)
	s.acs = &APICallStatis{
		statis: make(map[string]*APICallStatisItem),
		logger: newLogger(outputPath + "/" + "apicall.log"),
	}

	go s.Run()

	return nil
}

/**
 * Destroy 销毁统计插件
 */
func (s *StatisWorker) Destroy() error {
	return nil
}

/**
 * AddAPICall 上报请求
 */
func (s *StatisWorker) AddAPICall(api, protocol string, code int, duration int64) error {
	s.acc <- &APICall{
		api:      api,
		code:     code,
		protocol: protocol,
		duration: duration,
	}

	return nil
}

/**
 * Run 主流程
 */
func (s *StatisWorker) Run() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.acs.log()
		case ac := <-s.acc:
			// Local APICall observation data printing
			s.acs.add(ac)
		}
	}
}
