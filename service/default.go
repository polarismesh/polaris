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

package service

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"golang.org/x/sync/singleflight"

	"github.com/polarismesh/polaris/common/eventhub"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/plugin"
)

const (
	// MaxBatchSize max batch size
	MaxBatchSize = 100
	// MaxQuerySize max query size
	MaxQuerySize = 100
)

const (
	// SystemNamespace polaris system namespace
	SystemNamespace = "Polaris"
	// DefaultNamespace default namespace
	DefaultNamespace = "default"
	// ProductionNamespace default namespace
	ProductionNamespace = "Production"
	// DefaultTLL default ttl
	DefaultTLL = 5
)

type ServerProxyFactory func(pre DiscoverServer) (DiscoverServer, error)

var (
	server       DiscoverServer
	namingServer *Server = new(Server)
	once                 = sync.Once{}
	finishInit           = false
	// serverProxyFactories Service Server API 代理工厂
	serverProxyFactories = map[string]ServerProxyFactory{}
)

func RegisterServerProxy(name string, factor ServerProxyFactory) error {
	if _, ok := serverProxyFactories[name]; ok {
		return fmt.Errorf("duplicate ServerProxyFactory, name(%s)", name)
	}
	serverProxyFactories[name] = factor
	return nil
}

// Config 核心逻辑层配置
type Config struct {
	L5Open       *bool                  `yaml:"l5Open"`
	AutoCreate   *bool                  `yaml:"autoCreate"`
	Batch        map[string]interface{} `yaml:"batch"`
	Interceptors []string               `yaml:"-"`
}

// Initialize 初始化
func Initialize(ctx context.Context, namingOpt *Config, opts ...InitOption) error {
	var err error
	once.Do(func() {
		namingServer, server, err = InitServer(ctx, namingOpt, opts...)
	})

	if err != nil {
		return err
	}

	finishInit = true
	return nil
}

// GetServer 获取已经初始化好的Server
func GetServer() (DiscoverServer, error) {
	if !finishInit {
		return nil, errors.New("server has not done InitializeServer")
	}

	return server, nil
}

// GetOriginServer 获取已经初始化好的Server
func GetOriginServer() (*Server, error) {
	if !finishInit {
		return nil, errors.New("server has not done InitializeServer")
	}

	return namingServer, nil
}

// 内部初始化函数
func InitServer(ctx context.Context, namingOpt *Config, opts ...InitOption) (*Server, DiscoverServer, error) {
	actualSvr := new(Server)
	// l5service
	actualSvr.config = *namingOpt
	actualSvr.l5service = &l5service{}
	actualSvr.instanceChains = make([]InstanceChain, 0, 4)
	actualSvr.createServiceSingle = &singleflight.Group{}
	actualSvr.subCtxs = make([]*eventhub.SubscribtionContext, 0, 4)

	for i := range opts {
		opts[i](actualSvr)
	}

	// 插件初始化
	actualSvr.pluginInitialize()

	var proxySvr DiscoverServer
	proxySvr = actualSvr
	// 需要返回包装代理的 DiscoverServer
	order := namingOpt.Interceptors
	for i := range order {
		factory, exist := serverProxyFactories[order[i]]
		if !exist {
			return nil, nil, fmt.Errorf("name(%s) not exist in serverProxyFactories", order[i])
		}

		afterSvr, err := factory(proxySvr)
		if err != nil {
			return nil, nil, err
		}
		proxySvr = afterSvr
	}
	return actualSvr, proxySvr, nil
}

type PluginInstanceEventHandler struct {
	*BaseInstanceEventHandler
	subscriber plugin.DiscoverChannel
}

func (p *PluginInstanceEventHandler) OnEvent(ctx context.Context, any2 any) error {
	e := any2.(model.InstanceEvent)
	p.subscriber.PublishEvent(e)
	return nil
}

// 插件初始化
func (svr *Server) pluginInitialize() {
	// 获取CMDB插件
	svr.cmdb = plugin.GetCMDB()
	if svr.cmdb == nil {
		log.Warnf("Not Found CMDB Plugin")
	}

	// 获取History插件，注意：插件的配置在bootstrap已经设置好
	svr.history = plugin.GetHistory()
	if svr.history == nil {
		log.Warnf("Not Found History Log Plugin")
	}

	subscriber := plugin.GetDiscoverEvent()
	if subscriber == nil {
		log.Warnf("Not found DiscoverEvent Plugin")
		return
	}

	eventHandler := &PluginInstanceEventHandler{
		BaseInstanceEventHandler: NewBaseInstanceEventHandler(svr),
		subscriber:               subscriber,
	}
	subCtx, err := eventhub.Subscribe(eventhub.InstanceEventTopic, eventHandler)
	if err != nil {
		log.Warnf("register DiscoverEvent into eventhub:%s %v", subscriber.Name(), err)
	}
	svr.subCtxs = append(svr.subCtxs, subCtx)
}

func GetChainOrder() []string {
	return []string{
		"auth",
		"paramcheck",
	}
}
