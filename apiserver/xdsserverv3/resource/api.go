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

package resource

import (
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/service"
)

// XDSBuilder .
type XDSBuilder interface {
	// Init
	Init(service.DiscoverServer)
	// Generate
	Generate(option *BuildOption) (interface{}, error)
}

type BuildOption struct {
	RunType        RunType
	Namespace      string
	TLSMode        TLSMode
	Services       map[model.ServiceKey]*ServiceInfo
	OpenOnDemand   bool
	OnDemandServer string
	SelfService    model.ServiceKey
	// 不是比带，只有在 EDS 生成，并且是处理 INBOUND 的时候才会设置
	Client           *XDSClient
	TrafficDirection corev3.TrafficDirection
}

func (opt *BuildOption) Clone() *BuildOption {
	return &BuildOption{
		Namespace: opt.Namespace,
		TLSMode:   opt.TLSMode,
		Services:  opt.Services,
	}
}
