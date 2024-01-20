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

package memory

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	commonlog "github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/plugin"
)

const (
	// PluginName plugin name
	PluginName = "memory"
)

var (
	log = commonlog.RegisterScope("cmdb", "", 0)
)

// init 自注册到插件列表
func init() {
	plugin.RegisterPlugin(PluginName, &Memory{})
}

// Memory 定义MemoryCMDB类
type Memory struct {
	fetcher Fetcher
	cancel  context.CancelFunc
	IPs     atomic.Value
}

// Name 返回插件名
func (m *Memory) Name() string {
	return PluginName
}

// Initialize 初始化函数
func (m *Memory) Initialize(c *plugin.ConfigEntry) error {
	url, _ := c.Option["url"].(string)
	token, _ := c.Option["token"].(string)
	interval, _ := c.Option["interval"].(string)
	if len(url) == 0 {
		return nil
	}

	tick, err := time.ParseDuration(interval)
	if err != nil {
		tick = 5 * time.Minute
	}

	m.fetcher = &fetcher{
		url:   url,
		token: token,
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	go m.doFetch(ctx, tick)
	return nil
}

// GetLocation 实现CMDB插件接口
func (m *Memory) GetLocation(host string) (*model.Location, error) {
	val := m.IPs.Load()
	if val == nil {
		return nil, nil
	}
	ips := val.(*IPs)

	var (
		target IP
		find   bool
	)

	entry, ok := ips.Hosts[host]
	if ok {
		find = true
		target = entry
	} else {
		for i := range ips.Mask {
			if ips.Mask[i].Match(host) {
				target = ips.Mask[i]
				find = true
				break
			}
		}
		if !find && ips.Backoff != nil {
			find = true
			target = *ips.Backoff
		}
	}

	if !find {
		return nil, nil
	}

	return target.loc, nil
}

func (m *Memory) doFetch(ctx context.Context, interval time.Duration) {
	work := func() {
		ret, ips, err := m.fetcher.GetIPs()
		if err != nil {
			log.Error("[CMDB][Memory] fetch data from remote fail", zap.Error(err))
			return
		}

		data, err := json.Marshal(ret)
		if err != nil {
			log.Error("[CMDB][Memory] marshal receive cmdb data", zap.Error(err))
		} else {
			log.Infof("[CMDB][Memory] receive cmdb data \n%s\n", string(data))
		}

		m.IPs.Store(&ips)
	}

	work()

	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			work()
		case <-ctx.Done():
			return
		}
	}
}

// Destroy 销毁函数
func (m *Memory) Destroy() error {
	if m.cancel != nil {
		m.cancel()
	}
	return nil
}

// Range 实现CMDB插件接口
func (m *Memory) Range(handler func(host string, location *model.Location) (bool, error)) error {
	val := m.IPs.Load()
	if val == nil {
		return nil
	}
	ips := val.(*IPs)

	for i := range ips.Hosts {
		if _, err := handler(ips.Hosts[i].IP, ips.Hosts[i].loc); err != nil {
			return err
		}
	}

	return nil
}

// Size 实现CMDB插件接口
func (m *Memory) Size() int32 {
	val := m.IPs.Load()
	if val == nil {
		return 0
	}
	ips := val.(*IPs)
	return int32(len(ips.Hosts))
}
