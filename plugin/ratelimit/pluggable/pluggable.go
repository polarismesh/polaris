package pluggable

import (
	"context"
	"fmt"
	"time"

	pluginsdk "github.com/polaris-contrib/polaris-server-remote-plugin-common"
	"github.com/polaris-contrib/polaris-server-remote-plugin-common/api"

	"github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/pluggable"
	"github.com/polarismesh/polaris/plugin"
)

const (
	// PluginName pluggable rate limit plugin
	PluginName = "pluggable-rate-limit"
)

// 插件注册
func init() {
	pluggable.AddServiceDiscoveryCallback(
		api.Plugin_ServiceDesc.ServiceName,
		func(name string, dialer pluggable.GRPCConnectionDialer) {
			log.Infof("[Pluggable] registering rate limit plugin %s", name)
			plugin.RegisterPlugin(PluginName, newGRPCRateLimiter(dialer))
		},
	)
}

// newGRPCRateLimiter creates a new grpc rate limiter.
func newGRPCRateLimiter(dialer pluggable.GRPCConnectionDialer) *RateLimiter {
	return &RateLimiter{
		GRPCConnector: pluggable.NewGRPCConnectorWithDialer(dialer, api.NewPluginClient),
	}
}

// RateLimiter pluggable rate limit plugin
type RateLimiter struct {
	*pluggable.GRPCConnector
}

// Name returns the name of the plugin.
func (r *RateLimiter) Name() string {
	return PluginName
}

// Initialize initializes the plugin.
func (r *RateLimiter) Initialize(c *plugin.ConfigEntry) error {
	if err := r.Dial(r.Name()); err != nil {
		return err
	}

	if err := r.Ping(); err != nil {
		return fmt.Errorf("[Pluggable] failed to ping plugin server: %w", err)
	}

	log.Infof("[Pluggable] initialized pluggable rate limit plugin")
	return nil
}

// Destroy destroys the plugin.
func (r *RateLimiter) Destroy() error {
	log.Infof("[Pluggable] destroying pluggable rate limit plugin")
	return r.Close()
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
	log.Debugf("[Pluggable] request allow with type: %v and key: %s", typ, key)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req, err := pluginsdk.MarshalRequest(&api.RateLimitPluginRequest{
		Type: rateLimitTypes[typ],
		Key:  key,
	})

	rsp, err := r.Client.Call(ctx, req)
	if err != nil {
		log.Errorf("[Pluggable] fail to request plugin server, get error: %v", err)
		return false
	}

	var reply api.RateLimitPluginResponse
	if err = pluginsdk.UnmarshalResponse(rsp, &reply); err != nil {
		log.Errorf("[Pluggable] unmarshal response go error: %v", err)
		return false
	}

	return reply.GetAllow()
}
