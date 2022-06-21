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
	"net/http"
	"time"

	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/plugin"
	"github.com/polarismesh/polaris-server/store"
)

// init 注册统计插件
func init() {
	s := &StatisWorker{}
	plugin.RegisterPlugin(s.Name(), s)
}

// StatisWorker 本地统计插件
type StatisWorker struct {
	interval time.Duration

	acc chan *APICall
	acs *APICallStatis

	cacheCall   chan *CacheCall
	cacheStatis *CacheCallStatis
}

// Name 获取统计插件名称
func (s *StatisWorker) Name() string {
	return "local"
}

// Initialize 初始化统计插件
func (s *StatisWorker) Initialize(conf *plugin.ConfigEntry) error {
	// 设置统计打印周期
	var err error
	interval := conf.Option["interval"].(int)
	s.interval = time.Duration(interval) * time.Second

	outputPath := conf.Option["outputPath"].(string)

	// 初始化 prometheus 输出
	prometheusStatis, err := NewPrometheusStatis()
	if err != nil {
		return err
	}

	// 初始化接口调用统计
	s.acc = make(chan *APICall, 1024)
	s.acs, err = newAPICallStatis(outputPath, prometheusStatis)
	if err != nil {
		return err
	}
	go s.Run()

	s.cacheCall = make(chan *CacheCall, 1024)
	s.cacheStatis, err = newCacheCallStatis(outputPath, prometheusStatis)

	return nil
}

// Destroy 销毁统计插件
func (s *StatisWorker) Destroy() error {
	return nil
}

const maxAddDuration = 800 * time.Millisecond

// AddAPICall 上报请求
func (s *StatisWorker) AddAPICall(api string, protocol string, code int, duration int64) error {
	startTime := time.Now()
	s.acc <- &APICall{
		api:       api,
		protocol:  protocol,
		code:      code,
		duration:  duration,
		component: plugin.ComponentServer,
	}
	passDuration := time.Since(startTime)
	if passDuration >= maxAddDuration {
		log.Warnf("[APICall]add api call cost %s, exceed max %s", passDuration, maxAddDuration)
	}
	return nil
}

// AddRedisCall 上报redis请求
func (s *StatisWorker) AddRedisCall(api string, code int, duration int64) error {
	s.acc <- &APICall{
		api:       api,
		code:      code,
		duration:  duration,
		component: plugin.ComponentRedis,
	}
	return nil
}

// AddCacheCall 上报 Cache 指标信息
func (s *StatisWorker) AddCacheCall(component string, cacheType string, miss bool, call int) error {
	s.cacheCall <- &CacheCall{
		cacheType: cacheType,
		miss:      miss,
		component: component,
		count:     int32(call),
	}
	return nil
}

// GetPrometheusHandler 获取 prometheus http handler
func (s *StatisWorker) GetPrometheusHandler() http.Handler {
	return s.acs.prometheusStatis.GetHttpHandler()
}

// Run 主流程
func (s *StatisWorker) Run() {

	store, err := store.GetStore()
	if err != nil {
		log.Errorf("[APICall] get store error, %v", err)
		return
	}

	nowSeconds, err := store.GetUnixSecond()
	if err != nil {
		log.Errorf("[APICall] get now second from store error, %v", err)
		return
	}
	if nowSeconds == 0 {
		nowSeconds = time.Now().Unix()
	}
	dest := nowSeconds
	dest += 60
	dest = dest - (dest % 60)
	diff := dest - nowSeconds

	log.Infof("[APICall] prometheus stats need sleep %ds", diff)

	time.Sleep(time.Duration(diff) * time.Second)

	ticker := time.NewTicker(s.interval)
	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		case <-ticker.C:
			s.acs.log()
			s.cacheStatis.log()
		case ac := <-s.acc:
			s.acs.add(ac)
		case ac := <-s.cacheCall:
			s.cacheStatis.add(ac)
		}
	}

}
