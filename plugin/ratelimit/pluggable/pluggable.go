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

package pluggable

import (
	"context"
	"fmt"
	"time"

	"github.com/polaris-contrib/polaris-server-remote-plugin-common/api"
	"google.golang.org/grpc"

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
	pluggable.AddOnFinished(
		api.RateLimiter_ServiceDesc.ServiceName,
		func(name string, dialer pluggable.GRPCConnectionDialer) {
			log.Infof("[Pluggable] registering rate limit plugin %s", name)
			plugin.RegisterPlugin(PluginName, newGRPCRateLimiter(dialer))
		},
	)
}

// newGRPCRateLimiter creates a new grpc rate limiter.
func newGRPCRateLimiter(dialer pluggable.GRPCConnectionDialer) *RateLimiter {
	return &RateLimiter{
		GRPCConnector: pluggable.NewGRPCConnectorWithDialer(
			dialer,
			func(connInterface grpc.ClientConnInterface) pluggable.GRPCPluginClient {
				client := api.NewRateLimiterClient(connInterface)
				return client.(pluggable.GRPCPluginClient)
			},
		),
	}
}

// RateLimiter pluggable rate limit plugin
type RateLimiter struct {
	*pluggable.GRPCConnector
	client api.RateLimiterClient
}

// Name returns the name of the plugin.
func (r *RateLimiter) Name() string {
	return PluginName
}

// Initialize initializes the plugin.
func (r *RateLimiter) Initialize(c *plugin.ConfigEntry) error {
	if err := r.Dial(context.Background(), r.Name()); err != nil {
		return err
	}

	if err := r.Ping(context.Background()); err != nil {
		return fmt.Errorf("[Pluggable] failed to ping plugin server: %w", err)
	}

	client, ok := r.Client.(api.RateLimiterClient)
	if !ok {
		return fmt.Errorf("[Pluggable] failed to convert client to rate limiter client")
	}
	r.client = client

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

	req := &api.RateLimitRequest{
		Type: rateLimitTypes[typ],
		Key:  key,
	}
	rsp, err := r.client.Allow(ctx, req)
	if err != nil {
		log.Errorf("[Pluggable] fail to request plugin server, get error: %v", err)
		return false
	}
	return rsp.GetAllow()
}
