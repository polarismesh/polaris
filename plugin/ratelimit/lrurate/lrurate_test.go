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

package lrurate

import (
	"testing"

	"github.com/polarismesh/polaris/plugin"
)

// getEntry 获取初始化的entry
func getEntry() *plugin.ConfigEntry {
	entry := plugin.ConfigEntry{
		Option: make(map[string]interface{}),
	}
	entry.Option["rateLimitIPLruSize"] = 10
	entry.Option["rateLimitIPRate"] = 10
	entry.Option["rateLimitIPBurst"] = 10
	entry.Option["rateLimitServiceLruSize"] = 10
	entry.Option["rateLimitServiceRate"] = 10
	entry.Option["rateLimitServiceBurst"] = 10

	return &entry
}

// getUninitializedEntry 获取未初始化的entry
func getUninitializedEntry() *plugin.ConfigEntry {
	entry := plugin.ConfigEntry{
		Option: make(map[string]interface{}),
	}

	return &entry
}

// TestInvalidConfig 测试错误配置
func TestInvalidConfig(t *testing.T) {
	entry := getUninitializedEntry()
	s := &LRURate{}

	t.Run("InvalidIPLruSize", func(t *testing.T) {
		if err := s.Initialize(entry); err == nil {
			t.Errorf("failed, shouldn't Initialize")
		}

		entry.Option["rateLimitIPLruSize"] = 0
		if err := s.Initialize(entry); err == nil {
			t.Errorf("failed, shouldn't Initialize")
		}
	})

	t.Run("InvalidIPLruRate", func(t *testing.T) {
		entry.Option["rateLimitIPLruSize"] = 10
		if err := s.Initialize(entry); err == nil {
			t.Errorf("failed, shouldn't Initialize")
		}

		entry.Option["rateLimitIPRate"] = 0
		if err := s.Initialize(entry); err == nil {
			t.Errorf("failed, shouldn't Initialize")
		}
	})

	t.Run("InvalidIPLruBurst", func(t *testing.T) {
		entry.Option["rateLimitIPRate"] = 10
		if err := s.Initialize(entry); err == nil {
			t.Errorf("failed, shouldn't Initialize")
		}

		entry.Option["rateLimitIPBurst"] = 0
		if err := s.Initialize(entry); err == nil {
			t.Errorf("failed, shouldn't Initialize")
		}
	})

	t.Run("InvalidServiceLruSize", func(t *testing.T) {
		entry.Option["rateLimitIPBurst"] = 10
		if err := s.Initialize(entry); err == nil {
			t.Errorf("failed, shouldn't Initialize")
		}

		entry.Option["rateLimitServiceLruSize"] = 0
		if err := s.Initialize(entry); err == nil {
			t.Errorf("failed, shouldn't Initialize")
		}
	})

	t.Run("InvalidServiceLruRate", func(t *testing.T) {
		entry.Option["rateLimitServiceLruSize"] = 10
		if err := s.Initialize(entry); err == nil {
			t.Errorf("failed, shouldn't Initialize")
		}

		entry.Option["rateLimitServiceRate"] = 0
		if err := s.Initialize(entry); err == nil {
			t.Errorf("failed, shouldn't Initialize")
		}
	})

	t.Run("InvalidServiceLruBurst", func(t *testing.T) {
		entry.Option["rateLimitServiceRate"] = 10
		if err := s.Initialize(entry); err == nil {
			t.Errorf("failed, shouldn't Initialize")
		}

		entry.Option["rateLimitServiceBurst"] = 0
		if err := s.Initialize(entry); err == nil {
			t.Errorf("failed, shouldn't Initialize")
		}
	})
}

// TestValidConfig 测试正确配置
func TestValidConfig(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		entry := getEntry()
		s := &LRURate{}
		if err := s.Initialize(entry); err != nil {
			t.Errorf("failed: %s", err)
		} else {
			t.Logf("pass")
		}
	})
}

// TestCommon 测试一般函数
func TestCommon(t *testing.T) {
	entry := getEntry()
	s := &LRURate{}
	if err := s.Initialize(entry); err != nil {
		t.Fatalf("failed: %s", err)
	} else {
		t.Logf("pass")
	}

	t.Run("Name", func(t *testing.T) {
		if s.Name() != "lrurate" {
			t.Errorf("failed, invalid plgin name: %s", s.Name())
		} else {
			t.Logf("pass")
		}
	})

	t.Run("Destroy", func(t *testing.T) {
		if s.Destroy() != nil {
			t.Errorf("failed, bad Destroy")
		} else {
			t.Logf("pass")
		}
	})
}

// TestRateLimit 测试限流功能
func TestRateLimit(t *testing.T) {
	ipLruSize := 10
	ipRate := 100
	ipBurst := 200

	serviceLruSize := 10
	serviceRate := 50
	serviceBurst := 100

	entry := plugin.ConfigEntry{
		Option: make(map[string]interface{}),
	}
	entry.Option["rateLimitIPLruSize"] = ipLruSize
	entry.Option["rateLimitIPRate"] = ipRate
	entry.Option["rateLimitIPBurst"] = ipBurst
	entry.Option["rateLimitServiceLruSize"] = serviceLruSize
	entry.Option["rateLimitServiceRate"] = serviceRate
	entry.Option["rateLimitServiceBurst"] = serviceBurst

	s := LRURate{}
	if err := s.Initialize(&entry); err != nil {
		t.Errorf("failed: %s", err)
	} else {
		t.Logf("pass")
	}

	t.Run("RateLimit_UNKNOWN", func(t *testing.T) {
		count := 0
		total := 2 * ipBurst
		for i := 0; i < total; i++ {
			if s.Allow(10, "19216811") {
				count++
			}
		}

		if count != total {
			t.Errorf("failed, count: %d not %d", count, total)
		} else {
			t.Logf("pass")
		}
	})

	t.Run("RateLimit_IP", func(t *testing.T) {
		count := 0
		total := ipBurst + 10
		for i := 0; i < total; i++ {
			if s.Allow(plugin.IPRatelimit, "19216811") {
				count++
			}
		}

		if count != ipBurst {
			t.Errorf("failed, count: %d not %d", count, ipBurst)
		} else {
			t.Logf("pass")
		}
	})

	t.Run("RateLimit_SERVICE_SERVICE", func(t *testing.T) {
		count := 0
		total := serviceBurst + 10
		for i := 0; i < total; i++ {
			if s.Allow(plugin.ServiceRatelimit, "hello_world") {
				count++
			}
		}

		if count != serviceBurst {
			t.Errorf("failed, count: %d not %d", count, serviceBurst)
		} else {
			t.Logf("pass")
		}
	})

	t.Run("RateLimit_SERVICE_SERVICEID", func(t *testing.T) {
		count := 0
		for i := 0; i < serviceBurst+10; i++ {
			if s.Allow(plugin.ServiceRatelimit, "helloworld") {
				count++
			}
		}

		if count != serviceBurst {
			t.Errorf("failed, count: %d not %d", count, serviceBurst)
		} else {
			t.Logf("pass")
		}
	})
}
