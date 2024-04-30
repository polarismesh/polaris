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

package nacosserver

import (
	"context"
	"sync"

	"github.com/polarismesh/polaris/apiserver"
	"github.com/polarismesh/polaris/apiserver/nacosserver/core"
	"github.com/polarismesh/polaris/apiserver/nacosserver/model"
	nacosv1 "github.com/polarismesh/polaris/apiserver/nacosserver/v1"
	nacosv2 "github.com/polarismesh/polaris/apiserver/nacosserver/v2"
	"github.com/polarismesh/polaris/auth"
	connlimit "github.com/polarismesh/polaris/common/conn/limit"
	"github.com/polarismesh/polaris/common/secure"
	"github.com/polarismesh/polaris/config"
	"github.com/polarismesh/polaris/namespace"
	"github.com/polarismesh/polaris/service"
	"github.com/polarismesh/polaris/service/healthcheck"
)

const (
	ProtooclName = "service-nacos"
)

type NacosServer struct {
	connLimitConfig *connlimit.Config
	tlsInfo         *secure.TLSInfo
	option          map[string]interface{}
	apiConf         map[string]apiserver.APIConfig

	httpPort uint32
	grpcPort uint32

	pushCenter core.PushCenter
	store      *core.NacosDataStorage

	userSvr           auth.UserServer
	strategySvr       auth.StrategyServer
	namespaceSvr      namespace.NamespaceOperateServer
	discoverSvr       service.DiscoverServer
	originDiscoverSvr service.DiscoverServer
	configSvr         config.ConfigCenterServer
	originConfigSvr   config.ConfigCenterServer
	healthSvr         *healthcheck.Server

	v1Svr *nacosv1.NacosV1Server
	v2Svr *nacosv2.NacosV2Server
}

// GetProtocol API协议名
func (n *NacosServer) GetProtocol() string {
	return ProtooclName
}

// GetPort API的监听端口
func (n *NacosServer) GetPort() uint32 {
	return n.httpPort
}

// Initialize API初始化逻辑
func (n *NacosServer) Initialize(ctx context.Context, option map[string]interface{},
	apiConf map[string]apiserver.APIConfig) error {
	n.option = option
	n.apiConf = apiConf

	cfg, err := loadNacosConfig(option)
	if err != nil {
		return err
	}

	n.httpPort = uint32(cfg.ListenPort)
	n.grpcPort = uint32(cfg.ListenPort + 1000)

	// 连接数限制的配置
	n.connLimitConfig = cfg.ConnLimit

	// tls 配置信息
	if cfg.TLS != nil {
		n.tlsInfo = &secure.TLSInfo{
			CertFile:      cfg.TLS.CertFile,
			KeyFile:       cfg.TLS.KeyFile,
			TrustedCAFile: cfg.TLS.TrustedCAFile,
		}
	}
	model.ConvertPolarisNamespaceVal = cfg.DefaultNamespace
	return nil
}

// Run API服务的主逻辑循环
func (n *NacosServer) Run(errCh chan error) {
	if err := n.prepareRun(); err != nil {
		errCh <- err
		return
	}

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		option := copyOption(n.option)
		option["listenPort"] = n.httpPort
		if err := n.v1Svr.Initialize(context.Background(), option, n.httpPort, n.apiConf); err != nil {
			errCh <- err
			return
		}
		n.v1Svr.Run(errCh)
	}()

	go func() {
		defer wg.Done()
		option := copyOption(n.option)
		option["listenPort"] = n.grpcPort
		if err := n.v2Svr.Initialize(context.Background(), n.option, n.grpcPort, n.apiConf); err != nil {
			errCh <- err
			return
		}
		n.v2Svr.Run(errCh)
	}()

	wg.Wait()
}

func copyOption(m map[string]interface{}) map[string]interface{} {
	ret := map[string]interface{}{}
	for k, v := range m {
		ret[k] = v
	}
	return ret
}

func (n *NacosServer) initPolarisResource() error {
	var err error
	n.namespaceSvr, err = namespace.GetServer()
	if err != nil {
		return err
	}

	n.discoverSvr, err = service.GetServer()
	if err != nil {
		return err
	}
	n.originDiscoverSvr, err = service.GetOriginServer()
	if err != nil {
		return err
	}
	n.healthSvr, err = healthcheck.GetServer()
	if err != nil {
		return err
	}

	n.configSvr, err = config.GetServer()
	if err != nil {
		return err
	}
	n.originConfigSvr, err = config.GetOriginServer()
	if err != nil {
		return err
	}

	n.userSvr, err = auth.GetUserServer()
	if err != nil {
		return err
	}
	n.strategySvr, err = auth.GetStrategyServer()
	if err != nil {
		return err
	}
	return nil
}

func (n *NacosServer) prepareRun() error {
	err := n.initPolarisResource()
	if err != nil {
		return err
	}

	n.store = core.NewNacosDataStorage(n.discoverSvr.Cache())
	n.v1Svr, err = nacosv1.NewNacosV1Server(n.store,
		nacosv1.WithConnLimitConfig(n.connLimitConfig),
		nacosv1.WithTLS(n.tlsInfo),
		nacosv1.WithNamespaceSvr(n.namespaceSvr),
		nacosv1.WithDiscoverSvr(n.discoverSvr, n.originDiscoverSvr, n.healthSvr),
		nacosv1.WithConfigSvr(n.configSvr, n.originConfigSvr),
		nacosv1.WithAuthSvr(n.userSvr),
	)
	if err != nil {
		return err
	}

	n.v2Svr, err = nacosv2.NewNacosV2Server(n.v1Svr, n.store,
		nacosv2.WithConnLimitConfig(n.connLimitConfig),
		nacosv2.WithTLS(n.tlsInfo),
		nacosv2.WithNamespaceSvr(n.namespaceSvr),
		nacosv2.WithDiscoverSvr(n.discoverSvr, n.originDiscoverSvr, n.healthSvr),
		nacosv2.WithConfigSvr(n.configSvr, n.originConfigSvr),
		nacosv2.WithAuthSvr(n.userSvr),
	)
	if err != nil {
		return err
	}

	n.v2Svr.RegistryDebugRoute()
	return nil
}

// Stop 停止API端口监听
func (n *NacosServer) Stop() {
	if n.v1Svr != nil {
		n.v1Svr.Stop()
	}
	if n.v2Svr != nil {
		n.v2Svr.Stop()
	}
}

// Restart 重启API
func (n *NacosServer) Restart(option map[string]interface{}, api map[string]apiserver.APIConfig,
	errCh chan error) error {
	return nil
}
