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
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/mitchellh/mapstructure"
)

const (
	// redisStandalone 单机模式
	redisStandalone = "standalone"
	// redisSentinel 哨兵模式
	redisSentinel = "sentinel"
	// redisCluster 集群模式
	redisCluster = "cluster"
)

// Config redis pool configuration
type Config struct {
	// DeployMode is the run mode of the redis pool, support `standalone`、`cluster`、`sentinel`、or `ckv`
	DeployMode string `json:"deployMode"`
	// StandaloneConfig standalone-deploy-mode config
	StandaloneConfig
	// StandaloneConfig sentinel-deploy-mode config
	SentinelConfig
	// ClusterConfig cluster-deploy-mode config
	ClusterConfig
}

// UnmarshalJSON unmarshal config from json
func (c *Config) UnmarshalJSON(data []byte) error {
	var configmap map[string]interface{}
	if err := json.Unmarshal(data, &configmap); err != nil {
		return err
	}
	// 需要以用户的配置为优先
	raw := DefaultConfig().StandaloneConfig
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.StringToTimeDurationHookFunc(),
		ZeroFields: false,
		Result:     &raw,
		TagName:    "json",
	})
	if err != nil {
		return err
	}
	if err = decoder.Decode(configmap); err != nil {
		return err
	}
	c.DeployMode, _ = configmap["deployMode"].(string)
	c.StandaloneConfig = raw
	switch c.DeployMode {
	case redisCluster:
		var clusterConfig ClusterConfig
		if err = json.Unmarshal(data, &clusterConfig); err != nil {
			return fmt.Errorf("unmarshal redis cluster config error: %w", err)
		}
		c.ClusterConfig = clusterConfig
	case redisSentinel:
		var sentinelConfig SentinelConfig
		if err = json.Unmarshal(data, &sentinelConfig); err != nil {
			return fmt.Errorf("unmarshal redis sentinel config error: %w", err)
		}
		c.SentinelConfig = sentinelConfig
	case redisStandalone:
	default:
	}
	return nil
}

// StandaloneOptions standalone-mode options
func (c *Config) StandaloneOptions() *redis.Options {
	redisOption := &redis.Options{
		Addr:         c.KvAddr,
		Username:     c.KvUser,
		Password:     c.KvPasswd,
		MaxRetries:   c.MaxRetries,
		DialTimeout:  c.ConnectTimeout,
		PoolSize:     c.PoolSize,
		MinIdleConns: c.MinIdleConns,
		IdleTimeout:  c.IdleTimeout,
		DB:           c.DB,
		ReadTimeout:  c.ReadTimeout,
		WriteTimeout: c.WriteTimeout,
		PoolTimeout:  c.PoolTimeout,
		MaxConnAge:   c.MaxConnAge,
	}

	if redisOption.ReadTimeout == 0 {
		redisOption.ReadTimeout = c.MsgTimeout
	}

	if redisOption.WriteTimeout == 0 {
		redisOption.WriteTimeout = c.MsgTimeout
	}

	if c.MaxConnAge == 0 {
		redisOption.MaxConnAge = 1800 * time.Second
	}

	if c.WithTLS {
		redisOption.TLSConfig = c.tlsConfig
		if redisOption.TLSConfig == nil {
			redisOption.TLSConfig = &tls.Config{
				MinVersion: tls.VersionTLS12,
			}
		}
	}
	return redisOption
}

// ClusterOptions cluster-mode client options
func (c *Config) ClusterOptions() *redis.ClusterOptions {
	standalone := c.StandaloneOptions()
	return &redis.ClusterOptions{
		Addrs: c.ClusterConfig.Addrs,

		RouteByLatency: c.ClusterConfig.RouteByLatency,
		RouteRandomly:  c.ClusterConfig.RouteRandomly,

		Username:     standalone.Username,
		Password:     standalone.Password,
		MaxRetries:   standalone.MaxRetries,
		DialTimeout:  standalone.DialTimeout,
		PoolSize:     standalone.PoolSize,
		MinIdleConns: standalone.MinIdleConns,
		IdleTimeout:  standalone.IdleTimeout,
		ReadTimeout:  standalone.ReadTimeout,
		WriteTimeout: standalone.WriteTimeout,
		PoolTimeout:  standalone.PoolTimeout,
		MaxConnAge:   standalone.MaxConnAge,
		TLSConfig:    standalone.TLSConfig,
	}
}

// FailOverOptions sentinel-model options
func (c *Config) FailOverOptions() *redis.FailoverOptions {
	standalone := c.StandaloneOptions()
	return &redis.FailoverOptions{
		SentinelAddrs: c.SentinelConfig.Addrs,
		MasterName:    c.SentinelConfig.MasterName,

		SentinelUsername: c.SentinelConfig.SentinelUsername,
		SentinelPassword: c.SentinelConfig.SentinelPassword,

		Username:     standalone.Username,
		Password:     standalone.Password,
		MaxRetries:   standalone.MaxRetries,
		DialTimeout:  standalone.DialTimeout,
		PoolSize:     standalone.PoolSize,
		MinIdleConns: standalone.MinIdleConns,
		IdleTimeout:  standalone.IdleTimeout,
		ReadTimeout:  standalone.ReadTimeout,
		WriteTimeout: standalone.WriteTimeout,
		PoolTimeout:  standalone.PoolTimeout,
		MaxConnAge:   standalone.MaxConnAge,
		TLSConfig:    standalone.TLSConfig,
	}
}

// StandaloneConfig redis pool basic-configuration, also used as sentinel/cluster common config.
type StandaloneConfig struct {
	// KvAddr is the address of the redis server
	KvAddr string `json:"kvAddr"`

	// Use the specified Username to authenticate the current connection
	// with one of the connections defined in the ACL list when connecting
	// to a Redis 6.0 instance, or greater, that is using the Redis ACL system.
	KvUser string `json:"kvUser"`

	// KvPasswd for go-redis password or username (redis 6.0 version)
	// Optional password. Must match the password specified in the
	// requirepass server configuration option (if connecting to a Redis 5.0 instance, or lower),
	// or the User Password when connecting to a Redis 6.0 instance, or greater,
	// that is using the Redis ACL system.
	KvPasswd string `json:"kvPasswd"`

	// Minimum number of idle connections which is useful when establishing
	// new connection is slow.
	MinIdleConns int `json:"minIdleConns"`

	// Amount of time after which client closes idle connections.
	// Should be less than server's timeout.
	// Default is 5 minutes. -1 disables idle timeout check.
	IdleTimeout time.Duration `json:"idleTimeout"`

	// ConnectTimeout for go-redis is Dial timeout for establishing new connections.
	// Default is 5 seconds.
	ConnectTimeout time.Duration `json:"connectTimeout"`

	MsgTimeout    time.Duration `json:"msgTimeout"`
	Concurrency   int           `json:"concurrency"`
	Compatible    bool          `json:"compatible"`
	MaxRetry      int           `json:"maxRetry"`
	MinBatchCount int           `json:"minBatchCount"`
	WaitTime      time.Duration `json:"waitTime"`

	// MaxRetries is Maximum number of retries before giving up.
	// Default is 3 retries; -1 (not 0) disables retries.
	MaxRetries int `json:"maxRetries"`

	// DB is Database to be selected after connecting to the server.
	DB int `json:"DB"`

	// ReadTimeout for socket reads. If reached, commands will fail
	// with a timeout instead of blocking. Use value -1 for no timeout and 0 for default.
	// Default is 3 seconds.
	ReadTimeout time.Duration `json:"readTimeout"`

	// WriteTimeout for socket writes. If reached, commands will fail
	// with a timeout instead of blocking.
	// Default is ReadTimeout.
	WriteTimeout time.Duration `json:"writeTimeout"`

	// Maximum number of socket connections.
	// Default is 10 connections per every available CPU as reported by runtime.GOMAXPROCS.
	PoolSize int `json:"poolSize" mapstructure:"poolSize"`

	// Amount of time client waits for connection if all connections
	// are busy before returning an error.
	// Default is ReadTimeout + 1 second.
	PoolTimeout time.Duration `json:"poolTimeout"`

	// Connection age at which client retires (closes) the connection.
	// Default is to not close aged connections.
	MaxConnAge time.Duration `json:"maxConnAge"`

	// WithTLS whether open TLSConfig
	// if WithTLS is true, you should call WithEnableWithTLS,and then TLSConfig is not should be nil
	// In this case you should call WithTLSConfig func to set tlsConfig
	WithTLS bool `json:"withTLS"`

	// TLS Config to use. When set TLS will be negotiated.
	tlsConfig *tls.Config
}

// SentinelConfig sentinel pool configuration.
// See github.com/go-redis/redis/v8/redis.FailoverOptions
type SentinelConfig struct {
	// MasterName is the name of the master instance
	MasterName string `json:"masterName"`

	// A seed list of host:port addresses of sentinel servers.
	// Use shor name, to keep in line with ClusterConfig.Addrs
	Addrs []string `json:"addrs"`

	// Username ACL User and Password
	SentinelUsername string `json:"sentinelUsername"`
	// Password ACL User and Password
	SentinelPassword string `json:"sentinelPassword"`

	// Route all commands to slave read-only nodes.
	SlaveOnly bool

	// Use slaves disconnected with master when cannot get connected slaves
	// Now, this option only works in RandomSlaveAddr function.
	UseDisconnectedSlaves bool
}

// ClusterConfig redis cluster pool configuration
// See github.com/go-redis/redis/v8/redis.ClusterOptions
type ClusterConfig struct {
	// A seed list of host:port addresses of cluster nodes.
	Addrs []string

	// Enables read-only commands on slave nodes.
	ReadOnly bool

	// Allows routing read-only commands to the closest master or slave node.
	// It automatically enables ReadOnly.
	RouteByLatency bool

	// Allows routing read-only commands to the random master or slave node.
	// It automatically enables ReadOnly.
	RouteRandomly bool
}

// DefaultConfig redis pool configuration with default values
func DefaultConfig() *Config {
	return &Config{
		StandaloneConfig: StandaloneConfig{
			PoolSize:       200,
			MinIdleConns:   30,
			IdleTimeout:    120 * time.Second,
			ConnectTimeout: 300 * time.Millisecond,
			MsgTimeout:     300 * time.Millisecond,
			Concurrency:    200,
			Compatible:     false,
			MaxRetry:       2,
			MinBatchCount:  10,
			WaitTime:       50 * time.Millisecond,
			DB:             0,
			PoolTimeout:    3 * time.Second,
			MaxConnAge:     1800 * time.Second,
		},
	}
}

// Validate validate config params
func (c *Config) Validate() error {
	if len(c.KvAddr) == 0 {
		return errors.New("kvAddr is empty")
	}
	// password is required only when ACL's user is given
	if len(c.KvUser) > 0 && len(c.KvPasswd) == 0 {
		return errors.New("kvPasswd is empty")
	}
	if c.MinIdleConns <= 0 {
		return errors.New("minIdleConns is empty")
	}
	if c.PoolSize <= 0 {
		return errors.New("poolSize is empty")
	}
	if c.IdleTimeout == 0 {
		return errors.New("idleTimeout is empty")
	}
	if c.ConnectTimeout == 0 {
		return errors.New("connectTimeout is empty")
	}
	if c.MsgTimeout == 0 {
		return errors.New("msgTimeout is empty")
	}
	if c.Concurrency <= 0 {
		return errors.New("concurrency is empty")
	}
	if c.MaxRetry < 0 {
		return errors.New("maxRetry is empty")
	}

	if c.DeployMode == redisSentinel {
		if len(c.SentinelConfig.Addrs) == 0 {
			return errors.New("sentinel address list is empty")
		}
		if c.SentinelConfig.SentinelUsername != "" && c.SentinelConfig.SentinelPassword == "" {
			return errors.New("sentinel acl username or password is empty")
		}
	}

	if c.DeployMode == redisCluster && len(c.ClusterConfig.Addrs) == 0 {
		return errors.New("cluster address list is empty")
	}
	return nil
}
