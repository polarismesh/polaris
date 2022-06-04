package redispool

import (
	"crypto/tls"
	"time"

	commontime "github.com/polarismesh/polaris-server/common/time"
)

// Option functional options for Config
type Option func(c *Config)

// WithConfig set new config for NewPool,keep old code compatibility
func WithConfig(newConfig *Config) Option {
	return func(c *Config) {
		c = newConfig
	}
}

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
func WithMinIdleConns(minIdle int) Option {
	return func(c *Config) {
		c.MaxIdle = minIdle
	}
}

// WithIdleTimeout set idleTimeout
func WithIdleTimeout(idleTimeout time.Duration) Option {
	return func(c *Config) {
		c.IdleTimeout = commontime.Duration(idleTimeout)
	}
}

// WithConnectTimeout set connection timeout
func WithConnectTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.ConnectTimeout = commontime.Duration(timeout)
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
		c.WaitTime = commontime.Duration(t)
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
		c.ReadTimeout = commontime.Duration(timeout)
	}
}

// WithWriteTimeout set writeTimeout
func WithWriteTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.WriteTimeout = commontime.Duration(timeout)
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
		c.PoolTimeout = commontime.Duration(poolTimeout)
	}
}

// WithMaxConnAge set MaxConnAge
func WithMaxConnAge(maxConnAge time.Duration) Option {
	return func(c *Config) {
		c.MaxConnAge = commontime.Duration(maxConnAge)
	}
}

// WithUsername set username
func WithUsername(username string) Option {
	return func(c *Config) {
		c.Username = username
	}
}

// WithTLSConfig set TLSConfig
func WithTLSConfig(tlsConfig *tls.Config) Option {
	return func(c *Config) {
		c.tlsConfig = tlsConfig
	}
}

// WithEnableWithTLS set WithTLS
func WithEnableWithTLS() Option {
	return func(c *Config) {
		c.WithTLS = true
	}
}
