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
	"github.com/polarismesh/polaris/apiserver/nacosserver/core"
	"github.com/polarismesh/polaris/apiserver/nacosserver/v2/remote"
	"github.com/polarismesh/polaris/auth"
	cachetypes "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/config"
	"github.com/polarismesh/polaris/namespace"
)

type ServerOption struct {
	// nacos
	ConnectionManager *remote.ConnectionManager
	Store             *core.NacosDataStorage

	// polaris
	UserSvr         auth.UserServer
	CheckerSvr      auth.StrategyServer
	NamespaceSvr    namespace.NamespaceOperateServer
	ConfigSvr       config.ConfigCenterServer
	OriginConfigSvr config.ConfigCenterServer
}

type ConfigServer struct {
	userSvr         auth.UserServer
	checkerSvr      auth.StrategyServer
	namespaceSvr    namespace.NamespaceOperateServer
	configSvr       config.ConfigCenterServer
	originConfigSvr config.ConfigCenterServer
	cacheSvr        cachetypes.CacheManager
}

func (h *ConfigServer) Initialize(opt *ServerOption) error {
	h.userSvr = opt.UserSvr
	h.checkerSvr = opt.CheckerSvr
	h.namespaceSvr = opt.NamespaceSvr
	h.configSvr = opt.ConfigSvr
	h.originConfigSvr = opt.OriginConfigSvr
	h.cacheSvr = opt.Store.Cache()
	return nil
}

func (h *ConfigServer) ListGRPCHandlers() map[string]*remote.RequestHandlerWarrper {
	return nil
}
