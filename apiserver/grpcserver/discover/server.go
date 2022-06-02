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

	"github.com/polarismesh/polaris-server/apiserver"
	"github.com/polarismesh/polaris-server/apiserver/grpcserver"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/service"
	"github.com/polarismesh/polaris-server/service/healthcheck"
)

// GRPCServer GRPC API服务器
type GRPCServer struct {
	grpcserver.BaseGrpcServer
	namingServer      service.DiscoverServer
	healthCheckServer *healthcheck.Server
	openAPI           map[string]apiserver.APIConfig
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

	return g.BaseGrpcServer.Initialize(ctx, option, g.buildInitOptions(option)...)
}

// Run 启动GRPC API服务器
func (g *GRPCServer) Run(errCh chan error) {
	g.BaseGrpcServer.Run(errCh, g.GetProtocol(), func(server *grpc.Server) error {
		for name, config := range g.openAPI {
			switch name {
			case "client":
				if config.Enable {
					api.RegisterPolarisGRPCServer(server, g)
					openMethod, getErr := apiserver.GetClientOpenMethod(config.Include, g.GetProtocol())
					if getErr != nil {
						return getErr
					}
					g.BaseGrpcServer.OpenMethod = openMethod
				}
			default:
				log.Errorf("[Grpc][Discover] api %s does not exist in grpcserver", name)
				return fmt.Errorf("api %s does not exist in grpcserver", name)
			}
		}
		// 引入功能模块和插件
		var err error
		if g.namingServer, err = service.GetServer(); err != nil {
			log.Errorf("%v", err)
			return err
		}

		if g.healthCheckServer, err = healthcheck.GetServer(); err != nil {
			log.Errorf("%v", err)
			return err
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
	cacheTypes := []string{
		api.DiscoverResponse_INSTANCE.String(),
		api.DiscoverResponse_CLUSTER.String(),
		api.DiscoverResponse_ROUTING.String(),
		api.DiscoverResponse_RATE_LIMIT.String(),
		api.DiscoverResponse_CIRCUIT_BREAKER.String(),
		api.DiscoverResponse_SERVICES.String(),
	}

	initOptions := []grpcserver.InitOption{
		grpcserver.WithProtocol(g.GetProtocol()),
		grpcserver.WithMessageToCacheObject(discoverCacheConvert),
	}

	cache, err := grpcserver.NewCache(option, cacheTypes)
	if err != nil {
		log.Warn("[Grpc][Discover] new protobuf cache", zap.Error(err))
	}

	if cache != nil {
		initOptions = append(initOptions, grpcserver.WithProtobufCache(cache))
	}

	return initOptions
}

func discoverCacheConvert(m interface{}) *grpcserver.CacheObject {
	resp, ok := m.(*api.DiscoverResponse)

	if !ok {
		return nil
	}

	if resp.Code.GetValue() != api.ExecuteSuccess {
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
