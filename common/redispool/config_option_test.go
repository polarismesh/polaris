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

package redispool

import (
	"crypto/tls"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewRedisClient(t *testing.T) {
	config := DefaultConfig()

	t.Log("before config: ", config)

	_ = NewRedisClient(config,
		WithAddr("127.0.0.1:6379"),
		WithMaxConnAge(1000*time.Second),
		WithMinIdleConns(30),
	)
	assert.Equal(t, config.KvAddr, "127.0.0.1:6379")
	assert.Equal(t, config.MaxConnAge, 1000*time.Second)
	assert.Equal(t, config.MinIdleConns, 30)

	t.Log("after config: ", config)

	// client := NewRedisClient(WithConfig(config))
	// err := client.Set(context.Background(), "polaris", 1, 60*time.Second).Err()
	// if err != nil {
	// 	t.Fatalf("test redis client error:%v", err)
	// }

	t.Log("test success")
}

// testOptions optional functions for test
var testOptions = []Option{
	WithAddr("127.0.0.1:6379"),
	WithUsername(""),
	WithPwd("polaris"),
	WithMinIdleConns(1234),
	WithIdleTimeout(time.Minute),
	WithConnectTimeout(time.Millisecond * 10),
	WithConcurrency(20),
	WithCompatible(false),
	WithMaxRetry(5),
	WithMinBatchCount(1),
	WithWaitTime(time.Millisecond * 50),
	WithMaxRetries(5),
	WithDB(16),
	WithReadTimeout(time.Millisecond * 200),
	WithWriteTimeout(time.Millisecond * 200),
	WithPoolSize(2000),
	WithPoolTimeout(time.Second * 3),
	WithMaxConnAge(time.Minute * 30),
	WithTLSConfig(&tls.Config{
		InsecureSkipVerify: true,
	}),
}

func Test_WithStandalone(t *testing.T) {
	config := DefaultConfig()
	for _, option := range testOptions {
		option(config)
	}
	assert.Equal(t, &Config{
		DeployMode: "",
		StandaloneConfig: StandaloneConfig{
			KvAddr:         "127.0.0.1:6379",
			KvUser:         "",
			KvPasswd:       "polaris",
			MinIdleConns:   1234,
			IdleTimeout:    time.Minute,
			ConnectTimeout: time.Millisecond * 10,
			Concurrency:    20,
			MaxRetry:       5,
			MinBatchCount:  1,
			WaitTime:       time.Millisecond * 50,
			MaxRetries:     5,
			DB:             16,
			ReadTimeout:    time.Millisecond * 200,
			WriteTimeout:   time.Millisecond * 200,
			MsgTimeout:     time.Millisecond * 300,
			PoolSize:       2000,
			PoolTimeout:    time.Second * 3,
			MaxConnAge:     time.Minute * 30,
			WithTLS:        true,
			tlsConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}, config)
}

func Test_WithCluster(t *testing.T) {
	config := DefaultConfig()
	options := append(testOptions, WithCluster(ClusterConfig{
		Addrs: []string{
			"192.168.0.1:7000",
			"192.168.0.1:17000",
		},
		ReadOnly: true, // 开启从库读
	}))

	for _, option := range options {
		option(config)
	}

	assert.Equal(t, &Config{
		DeployMode: redisCluster,
		StandaloneConfig: StandaloneConfig{
			KvPasswd:       "polaris",
			MinIdleConns:   1234,
			IdleTimeout:    time.Minute,
			ConnectTimeout: time.Millisecond * 10,
			Concurrency:    20,
			MaxRetry:       5,
			MinBatchCount:  1,
			WaitTime:       time.Millisecond * 50,
			MaxRetries:     5,
			DB:             16,
			ReadTimeout:    time.Millisecond * 200,
			WriteTimeout:   time.Millisecond * 200,
			MsgTimeout:     time.Millisecond * 300,
			PoolSize:       2000,
			PoolTimeout:    time.Second * 3,
			MaxConnAge:     time.Minute * 30,
			WithTLS:        true,
			tlsConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		ClusterConfig: ClusterConfig{
			Addrs: []string{
				"192.168.0.1:7000",
				"192.168.0.1:17000",
			},
			ReadOnly: true,
		},
	}, config)
}

func TestWithSentinel(t *testing.T) {
	config := DefaultConfig()
	options := append(testOptions, WithSentinel(SentinelConfig{
		Addrs: []string{
			"192.168.0.1:26379",
			"192.168.0.2:26379",
		},
		MasterName: "sentinel_master_name",
	}))

	for _, option := range options {
		option(config)
	}

	assert.Equal(t, &Config{
		DeployMode: redisSentinel,
		StandaloneConfig: StandaloneConfig{
			KvPasswd:       "polaris",
			MinIdleConns:   1234,
			IdleTimeout:    time.Minute,
			ConnectTimeout: time.Millisecond * 10,
			Concurrency:    20,
			MaxRetry:       5,
			MinBatchCount:  1,
			WaitTime:       time.Millisecond * 50,
			MaxRetries:     5,
			DB:             16,
			ReadTimeout:    time.Millisecond * 200,
			WriteTimeout:   time.Millisecond * 200,
			MsgTimeout:     time.Millisecond * 300,
			PoolSize:       2000,
			PoolTimeout:    time.Second * 3,
			MaxConnAge:     time.Minute * 30,
			WithTLS:        true,
			tlsConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		SentinelConfig: SentinelConfig{
			Addrs: []string{
				"192.168.0.1:26379",
				"192.168.0.2:26379",
			},
			MasterName: "sentinel_master_name",
		},
	}, config)
}
