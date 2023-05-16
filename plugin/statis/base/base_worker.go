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

package base

import (
	"context"
	"time"

	"github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/metrics"
	"github.com/polarismesh/polaris/store"
)

type BaseWorker struct {
	apiStatis   *ComponentStatics
	redisStatis *ComponentStatics
	innerStatis *ComponentStatics
	cacheStatis *CacheCallStatis
}

func NewBaseWorker(ctx context.Context, handler MetricsHandler) (*BaseWorker, error) {
	cacheStatis, err := NewCacheCallStatis(ctx)
	if err != nil {
		return nil, err
	}
	return &BaseWorker{
		apiStatis:   NewComponentStatics(ctx, metrics.ServerCallMetric, handler),
		redisStatis: NewComponentStatics(ctx, metrics.RedisCallMetric, handler),
		innerStatis: NewComponentStatics(ctx, metrics.SystemCallMetric, handler),
		cacheStatis: cacheStatis,
	}, nil
}

// ReportCallMetrics report call metrics info
func (s *BaseWorker) ReportCallMetrics(metric metrics.CallMetric) {
	switch metric.Type {
	case metrics.ServerCallMetric:
		item := &APICall{
			Api:       metric.API,
			Protocol:  metric.Protocol,
			Code:      metric.Code,
			Duration:  int64(metric.Duration.Nanoseconds()),
			Component: metric.Type,
		}
		s.apiStatis.Add(item)
	case metrics.SystemCallMetric:
		item := &APICall{
			Api:       metric.API,
			Protocol:  metric.Protocol,
			Code:      metric.Code,
			Duration:  int64(metric.Duration.Nanoseconds()),
			Component: metric.Type,
		}
		s.innerStatis.Add(item)
	case metrics.RedisCallMetric:
		item := &APICall{
			Api:       metric.API,
			Protocol:  metric.Protocol,
			Code:      metric.Code,
			Duration:  int64(metric.Duration.Nanoseconds()),
			Component: metric.Type,
		}
		s.redisStatis.Add(item)
	case metrics.ProtobufCacheCallMetric:
		s.cacheStatis.Add(metric)
	}
}

// Run 主流程
func (s *BaseWorker) Run(ctx context.Context, interval time.Duration) {
	getStore, err := store.GetStore()
	if err != nil {
		log.Errorf("[APICall] get store error, %v", err)
		return
	}

	nowSeconds, err := getStore.GetUnixSecond(time.Second)
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

	log.Infof("[APICall] base stats need sleep %ds", diff)
	time.Sleep(time.Duration(diff) * time.Second)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.apiStatis.deal()
			s.redisStatis.deal()
			s.innerStatis.deal()
			s.cacheStatis.deal()
		}
	}
}
