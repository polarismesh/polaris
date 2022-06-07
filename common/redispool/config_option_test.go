package redispool

import (
	"context"
	"testing"
	"time"
)

func TestNewRedisClient(t *testing.T) {
	config := DefaultConfig()

	// mock config read
	config.KvAddr = "127.0.0.1:6379"
	config.MaxConnAge = 1000
	config.MinIdleConns = 30

	client := NewRedisClient(WithConfig(config))
	err := client.Set(context.Background(), "polaris", 1, 60*time.Second).Err()
	if err != nil {
		t.Fatalf("test redis client error:%v", err)
	}

	t.Log("test success")
}
