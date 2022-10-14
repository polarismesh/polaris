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

package discover

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/polarismesh/polaris/apiserver"
	"github.com/polarismesh/polaris/apiserver/grpcserver"
	v1 "github.com/polarismesh/polaris/apiserver/grpcserver/discover/v1"
	v2 "github.com/polarismesh/polaris/apiserver/grpcserver/discover/v2"
	apiv1 "github.com/polarismesh/polaris/common/api/v1"
	apiv2 "github.com/polarismesh/polaris/common/api/v2"
	commonlog "github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/service"
	"github.com/polarismesh/polaris/service/healthcheck"
)

var (
	namingLog = commonlog.GetScopeOrDefaultByName(commonlog.NamingLoggerName)

	cacheTypes = map[string]struct{}{
		apiv1.DiscoverResponse_INSTANCE.String():        {},
		apiv1.DiscoverResponse_ROUTING.String():         {},
		apiv1.DiscoverResponse_RATE_LIMIT.String():      {},
		apiv1.DiscoverResponse_CIRCUIT_BREAKER.String(): {},
	}
)

// GRPCServer GRPC API服务器
type GRPCServer struct {
	grpcserver.BaseGrpcServer
	namingServer      service.DiscoverServer
	healthCheckServer *healthcheck.Server
	openAPI           map[string]apiserver.APIConfig

	v1server *v1.DiscoverServer
	v2server *v2.DiscoverServer
}

// GetPort 获取端口
func (g *GRPCServer) GetPort() uint32 {
	return g.BaseGrpcServer.GetPort()
}

// GetProtocol 获取Server的协议
func (g *GRPCServer) GetProtocol() string {
	return "grpc"
}

// Initialize 初始化GRPC API服务器
func (g *GRPCServer) Initialize(ctx context.Context, option map[string]interface{},
	apiConf map[string]apiserver.APIConfig) error {

	g.openAPI = apiConf

	if err := g.BaseGrpcServer.Initialize(ctx, option, g.buildInitOptions(option)...); err != nil {
		return err
	}

	// 引入功能模块和插件
	var err error
	if g.namingServer, err = service.GetServer(); err != nil {
		namingLog.Errorf("%v", err)
		return err
	}

	if g.healthCheckServer, err = healthcheck.GetServer(); err != nil {
		namingLog.Errorf("%v", err)
		return err
	}

	g.v1server = v1.NewDiscoverServer(
		v1.WithAllowAccess(g.allowAccess),
		v1.WithEnterRateLimit(g.enterRateLimit),
		v1.WithHealthCheckerServer(g.healthCheckServer),
		v1.WithNamingServer(g.namingServer),
	)
	g.v2server = v2.NewDiscoverServer(
		v2.WithAllowAccess(g.allowAccess),
		v2.WithEnterRateLimit(g.enterRateLimit),
		v2.WithHealthCheckerServer(g.healthCheckServer),
		v2.WithNamingServer(g.namingServer),
	)

	return nil
}

// Run 启动GRPC API服务器
func (g *GRPCServer) Run(errCh chan error) {

	g.BaseGrpcServer.Run(errCh, g.GetProtocol(), func(server *grpc.Server) error {
		for name, config := range g.openAPI {
			switch name {
			case "client":
				if config.Enable {
					// 注册 v1 版本的 spec discover server
					apiv1.RegisterPolarisGRPCServer(server, g.v1server)
					// 注册 v2 版本的 spec discover server
					apiv2.RegisterPolarisGRPCServer(server, g.v2server)
					openMethod, getErr := apiserver.GetClientOpenMethod(config.Include, g.GetProtocol())
					if getErr != nil {
						return getErr
					}
					g.BaseGrpcServer.OpenMethod = openMethod
				}
			default:
				namingLog.Errorf("[Grpc][Discover] api %s does not exist in grpcserver", name)
				return fmt.Errorf("api %s does not exist in grpcserver", name)
			}
		}
		return nil
	})
}

// Stop 关闭GRPC
func (g *GRPCServer) Stop() {
	g.BaseGrpcServer.Stop(g.GetProtocol())
}

// Restart 重启Server
func (g *GRPCServer) Restart(option map[string]interface{}, api map[string]apiserver.APIConfig,
	errCh chan error) error {
	initFunc := func() error {
		return g.Initialize(context.Background(), option, api)
	}
	runFunc := func() {
		g.Run(errCh)
	}
	return g.BaseGrpcServer.Restart(initFunc, runFunc, g.GetProtocol(), option)
}

// enterRateLimit 限流
func (g *GRPCServer) enterRateLimit(ip string, method string) uint32 {
	return g.BaseGrpcServer.EnterRatelimit(ip, method)
}

// allowAccess 限制访问
func (g *GRPCServer) allowAccess(method string) bool {
	return g.BaseGrpcServer.AllowAccess(method)
}

func (g *GRPCServer) buildInitOptions(option map[string]interface{}) []grpcserver.InitOption {
	initOptions := []grpcserver.InitOption{
		grpcserver.WithProtocol(g.GetProtocol()),
		grpcserver.WithLogger(namingLog),
		grpcserver.WithMessageToCacheObject(discoverCacheConvert),
	}

	types := make([]string, 0, len(cacheTypes))
	for k := range cacheTypes {
		types = append(types, k)
	}

	cache, err := grpcserver.NewCache(option, types)
	if err != nil {
		namingLog.Warn("[Grpc][Discover] new protobuf cache", zap.Error(err))
	}

	if cache != nil {
		initOptions = append(initOptions, grpcserver.WithProtobufCache(cache))
	}

	return initOptions
}

// discoverCacheConvert 将 DiscoverResponse 转换为 grpcserver.CacheObject
// 当前支持进行 pb cache 缓存的 Type
// 1. DiscoverResponse_INSTANCE
// 2. DiscoverResponse_ROUTING
// 3. DiscoverResponse_RATE_LIMIT
// 4. DiscoverResponse_CIRCUIT_BREAKER
func discoverCacheConvert(m interface{}) *grpcserver.CacheObject {
	resp, ok := m.(*apiv1.DiscoverResponse)

	if !ok {
		return nil
	}

	if resp.Code.GetValue() != apiv1.ExecuteSuccess {
		return nil
	}

	if _, ok := cacheTypes[resp.GetType().String()]; !ok {
		return nil
	}

	keyProto := fmt.Sprintf("%s-%s-%s", resp.Service.Namespace.GetValue(),
		resp.Service.Name.GetValue(), resp.Service.Revision.GetValue())

	return &grpcserver.CacheObject{
		OriginVal: resp,
		CacheType: resp.Type.String(),
		Key:       keyProto,
	}
}
