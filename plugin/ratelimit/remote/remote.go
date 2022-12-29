package remote

import (
	"context"
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/polaris-contrib/polaris-server-remote-plugin-common"
	"github.com/polaris-contrib/polaris-server-remote-plugin-common/api"
	"github.com/polaris-contrib/polaris-server-remote-plugin-common/client"

	"github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/plugin"
)

const (
	// PluginName remote rate limit plugin
	PluginName = "remote-rate-limit"
)

// 插件注册
func init() {
	plugin.RegisterPlugin(PluginName, &RateLimiter{})
}

// 接口实现断言
var _ plugin.Ratelimit = (*RateLimiter)(nil)

// RateLimiter 远程限流插件
type RateLimiter struct {
	cfg    *client.Config
	client client.Client
}

// Name 返回插件名
func (r *RateLimiter) Name() string {
	return PluginName
}

// Initialize 初始化函数
func (r *RateLimiter) Initialize(c *plugin.ConfigEntry) error {
	var cfg = &client.Config{}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.StringToTimeDurationHookFunc(),
		ZeroFields: false,
		Result:     cfg,
		TagName:    "yaml",
	})
	if err != nil {
		return err
	}
	if err = decoder.Decode(c.Option); err != nil {
		return err
	}

	r.cfg = cfg
	r.client, err = client.Register(cfg)
	if err != nil {
		return fmt.Errorf("failed to setup rate-limit plugin: %w", err)
	}
	return nil
}

// Destroy 销毁函数
func (r *RateLimiter) Destroy() error {
	if r.client == nil {
		return nil
	}
	return r.client.Close()
}

// rateLimitTypes converts ...
var rateLimitTypes = map[plugin.RatelimitType]api.RatelimitType{
	plugin.IPRatelimit:       api.RatelimitType_IPRatelimit,
	plugin.APIRatelimit:      api.RatelimitType_APIRatelimit,
	plugin.ServiceRatelimit:  api.RatelimitType_ServiceRatelimit,
	plugin.InstanceRatelimit: api.RatelimitType_InstanceRatelimit,
}

// Allow 实现是否放行判断逻辑
func (r *RateLimiter) Allow(typ plugin.RatelimitType, key string) bool {
	req, err := pluginsdk.MarshalRequest(&api.RateLimitPluginRequest{
		Type: rateLimitTypes[typ],
		Key:  key,
	})
	if err != nil {
		log.Errorf("[RateLimit]fail to convert plugin req, get error: %+v", err)
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := r.client.Call(ctx, req)
	if err != nil {
		log.Errorf("[RateLimit]fail to request plugin server, get error: %+v", err)
		return false
	}

	var reply api.RateLimitPluginResponse
	if err = pluginsdk.UnmarshalResponse(response, &reply); err != nil {
		log.Errorf("[RateLimit] %w", err)
		return false
	}

	return reply.GetAllow()
}
