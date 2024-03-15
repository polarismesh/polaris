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

package apiserver

import (
	"context"
	"fmt"

	"github.com/polarismesh/polaris/common/model"
)

const (
	DiscoverAccess    string = "discover"
	RegisterAccess    string = "register"
	HealthcheckAccess string = "healthcheck"
	CreateFileAccess  string = "createfile"
)

// Config API服务器配置
type Config struct {
	Name   string
	Option map[string]interface{}
	API    map[string]APIConfig
}

// APIConfig API配置
type APIConfig struct {
	Enable  bool
	Include []string
}

// Apiserver API服务器接口
type Apiserver interface {
	// GetProtocol API协议名
	GetProtocol() string
	// GetPort API的监听端口
	GetPort() uint32
	// Initialize API初始化逻辑
	Initialize(ctx context.Context, option map[string]interface{}, api map[string]APIConfig) error
	// Run API服务的主逻辑循环
	Run(errCh chan error)
	// Stop 停止API端口监听
	Stop()
	// Restart 重启API
	Restart(option map[string]interface{}, api map[string]APIConfig, errCh chan error) error
}

type EnrichApiserver interface {
	Apiserver
	DebugHandlers() []model.DebugHandler
}

var (
	Slots = make(map[string]Apiserver)
)

// Register 注册API服务器
func Register(name string, server Apiserver) error {
	if _, exist := Slots[name]; exist {
		return fmt.Errorf("apiserver name:%s exist", name)
	}

	Slots[name] = server

	return nil
}
