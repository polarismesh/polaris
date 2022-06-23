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
	"time"
)

// Option functional options for Config
type Option func(c *Config)

// WithAddr set redis addr
func WithAddr(addr string) Option {
	return func(c *Config) {
		c.KvAddr = addr
	}
}

// WithPwd set pwd
func WithPwd(pwd string) Option {
	return func(c *Config) {
		c.KvPasswd = pwd
	}
}

// WithMinIdleConns set minIdleConns
func WithMinIdleConns(minIdleConns int) Option {
	return func(c *Config) {
		c.MinIdleConns = minIdleConns
	}
}

// WithIdleTimeout set idleTimeout
func WithIdleTimeout(idleTimeout time.Duration) Option {
	return func(c *Config) {
		c.IdleTimeout = idleTimeout
	}
}

// WithConnectTimeout set connection timeout
func WithConnectTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.ConnectTimeout = timeout
	}
}

// WithConcurrency set concurrency size
func WithConcurrency(size int) Option {
	return func(c *Config) {
		c.Concurrency = size
	}
}

// WithCompatible set Compatible
func WithCompatible(b bool) Option {
	return func(c *Config) {
		c.Compatible = b
	}
}

// WithMaxRetry set pool MaxRetry
func WithMaxRetry(maxRetry int) Option {
	return func(c *Config) {
		c.MaxRetry = maxRetry
	}
}

// WithMinBatchCount set MinBatchCount
func WithMinBatchCount(n int) Option {
	return func(c *Config) {
		c.MinBatchCount = n
	}
}

// WithWaitTime set wait timeout
func WithWaitTime(t time.Duration) Option {
	return func(c *Config) {
		c.WaitTime = t
	}
}

// WithMaxRetries set maxRetries
func WithMaxRetries(maxRetries int) Option {
	return func(c *Config) {
		c.MaxRetries = maxRetries
	}
}

// WithDB set redis db
func WithDB(num int) Option {
	return func(c *Config) {
		c.DB = num
	}
}

// WithReadTimeout set readTimeout
func WithReadTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.ReadTimeout = timeout
	}
}

// WithWriteTimeout set writeTimeout
func WithWriteTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.WriteTimeout = timeout
	}
}

// WithPoolSize set pool size
func WithPoolSize(poolSize int) Option {
	return func(c *Config) {
		c.PoolSize = poolSize
	}
}

// WithPoolTimeout set pool timeout
func WithPoolTimeout(poolTimeout time.Duration) Option {
	return func(c *Config) {
		c.PoolTimeout = poolTimeout
	}
}

// WithMaxConnAge set MaxConnAge
func WithMaxConnAge(maxConnAge time.Duration) Option {
	return func(c *Config) {
		c.MaxConnAge = maxConnAge
	}
}

// WithUsername set username
func WithUsername(username string) Option {
	return func(c *Config) {
		c.KvUser = username
	}
}

// WithTLSConfig set TLSConfig
func WithTLSConfig(tlsConfig *tls.Config) Option {
	return func(c *Config) {
		c.WithTLS = true
		c.tlsConfig = tlsConfig
	}
}

// WithCluster use redis cluster
func WithCluster(cc ClusterConfig) Option {
	return func(c *Config) {
		c.DeployMode = redisCluster
		c.StandaloneConfig.KvAddr = ""
		c.ClusterConfig = cc
	}
}

// WithSentinel use redis sentinel
func WithSentinel(sc SentinelConfig) Option {
	return func(c *Config) {
		c.DeployMode = redisSentinel
		c.StandaloneConfig.KvAddr = ""
		c.SentinelConfig = sc
	}
}
