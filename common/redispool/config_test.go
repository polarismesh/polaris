package redispool

import (
	"encoding/json"
	"testing"

	"github.com/polarismesh/polaris-server/plugin"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func Test_UnmarshalClusterConfig(t *testing.T) {
	raw := `
name: heartbeatRedis
option:
  deployMode: cluster
  addrs:
    - "127.0.0.1:7001"
    - "127.0.0.1:7002"
    - "127.0.0.1:7003"
  kvPasswd: "polaris"
  poolSize: 233
  minIdleConns: 30
  idleTimeout: 120s
  connectTimeout: 200ms
  msgTimeout: 200ms
  concurrency: 200
  withTLS: false
`
	var entry plugin.ConfigEntry
	if err := yaml.Unmarshal([]byte(raw), &entry); err != nil {
		t.Fatalf("unmarshal yaml error: %v", err)
	}

	data, err := json.Marshal(entry.Option)
	if err != nil {
		t.Fatalf("marshal config entry got error: %v", err)
	}

	var config Config
	if err = json.Unmarshal(data, &config); err != nil {
		t.Fatalf("unmarshal to json got error:%v", err)
	}

	assert.Equal(t, config.DeployMode, "cluster")
	assert.Equal(t, config.KvAddr, "")
	assert.Equal(t, config.KvPasswd, "polaris")
	assert.Equal(t, config.PoolSize, 233)
	assert.Equal(t, config.ClusterConfig.Addrs, []string{
		"127.0.0.1:7001",
		"127.0.0.1:7002",
		"127.0.0.1:7003",
	})
}

func Test_SentinelConfig(t *testing.T) {
	raw := `
name: heartbeatRedis
option:
  deployMode: sentinel
  addrs:
    - "127.0.0.1:26379"
    - "127.0.0.2:26379"
    - "127.0.0.3:26379"
  masterName: "my-sentinel-master-name"
  sentinelUsername: "sentinel-polaris" # sentinel 客户端的用户名
  sentinelPassword: "sentinel-polaris-password" # sentinel 客户端的密码
  kvPasswd: "polaris" # redis 客户端的密码
  poolSize: 233
  minIdleConns: 30
  idleTimeout: 120s
  connectTimeout: 200ms
  msgTimeout: 200ms
  concurrency: 200
  withTLS: false
`

	var entry plugin.ConfigEntry
	if err := yaml.Unmarshal([]byte(raw), &entry); err != nil {
		t.Fatalf("unmarshal yaml error: %v", err)
	}

	data, err := json.Marshal(entry.Option)
	if err != nil {
		t.Fatalf("marshal config entry got error: %v", err)
	}

	var config Config
	if err = json.Unmarshal(data, &config); err != nil {
		t.Fatalf("unmarshal to json got error:%v", err)
	}

	assert.Equal(t, config.DeployMode, "sentinel")
	assert.Equal(t, config.KvAddr, "")
	assert.Equal(t, config.KvPasswd, "polaris")
	assert.Equal(t, config.PoolSize, 233)

	assert.Equal(t, config.SentinelConfig.MasterName, "my-sentinel-master-name")
	assert.Equal(t, config.SentinelConfig.SentinelUsername, "sentinel-polaris")
	assert.Equal(t, config.SentinelConfig.SentinelPassword, "sentinel-polaris-password")
	assert.Equal(t, config.SentinelConfig.Addrs, []string{
		"127.0.0.1:26379",
		"127.0.0.2:26379",
		"127.0.0.3:26379",
	})
}
