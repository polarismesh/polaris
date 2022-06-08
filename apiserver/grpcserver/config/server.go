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

package config

import (
	"context"
	"fmt"
	"strings"

	"github.com/polarismesh/polaris-server/apiserver/grpcserver"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/config"

	"google.golang.org/grpc"

	"github.com/polarismesh/polaris-server/apiserver"
	api "github.com/polarismesh/polaris-server/common/api/v1"
)

// ConfigGRPCServer 配置中心 GRPC API 服务器
type ConfigGRPCServer struct {
	grpcserver.BaseGrpcServer
	configServer config.ConfigCenterServer
	openAPI      map[string]apiserver.APIConfig
}

// GetPort 获取端口
func (g *ConfigGRPCServer) GetPort() uint32 {
	return g.BaseGrpcServer.GetPort()
}

// GetProtocol 获取Server的协议
func (g *ConfigGRPCServer) GetProtocol() string {
	return "grpc"
}

// Initialize 初始化GRPC API服务器
func (g *ConfigGRPCServer) Initialize(ctx context.Context, option map[string]interface{},
	api map[string]apiserver.APIConfig) error {
	g.openAPI = api
	return g.BaseGrpcServer.Initialize(ctx, option, grpcserver.WithProtocol(g.GetProtocol()))
}

// Run 启动GRPC API服务器
func (g *ConfigGRPCServer) Run(errCh chan error) {
	g.BaseGrpcServer.Run(errCh, g.GetProtocol(), func(server *grpc.Server) error {
		for name, apiConfig := range g.openAPI {
			switch name {
			case "client":
				if apiConfig.Enable {
					api.RegisterPolarisConfigGRPCServer(server, g)
					openMethod, getErr := getConfigClientOpenMethod(g.GetProtocol())
					if getErr != nil {
						return getErr
					}
					if g.BaseGrpcServer.OpenMethod == nil {
						g.BaseGrpcServer.OpenMethod = openMethod
					} else {
						for method, opened := range openMethod {
							g.BaseGrpcServer.OpenMethod[method] = opened
						}
					}
				}
			default:
				log.Errorf("[Config] api %s does not exist in grpcserver", name)
				return fmt.Errorf("api %s does not exist in grpcserver", name)
			}
		}
		var err error
		if g.configServer, err = config.GetServer(); err != nil {
			log.Errorf("[Config] %v", err)
			return err
		}

		return nil
	})
}

// Stop 关闭GRPC
func (g *ConfigGRPCServer) Stop() {
	g.BaseGrpcServer.Stop(g.GetProtocol())
}

// Restart 重启Server
func (g *ConfigGRPCServer) Restart(option map[string]interface{}, api map[string]apiserver.APIConfig,
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
func (g *ConfigGRPCServer) enterRateLimit(ip string, method string) uint32 {
	return g.BaseGrpcServer.EnterRatelimit(ip, method)
}

// allowAccess 限制访问
func (g *ConfigGRPCServer) allowAccess(method string) bool {
	return g.BaseGrpcServer.AllowAccess(method)
}

func getConfigClientOpenMethod(protocol string) (map[string]bool, error) {
	openMethods := []string{"GetConfigFile", "WatchConfigFiles"}

	openMethod := make(map[string]bool)

	for _, item := range openMethods {
		method := "/v1.PolarisConfig" + strings.ToUpper(protocol) + "/" + item
		openMethod[method] = true
	}

	return openMethod, nil
}
