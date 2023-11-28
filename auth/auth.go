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

package auth

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/polarismesh/polaris/cache"
	cachetypes "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/store"
)

const (
	// DefaultUserMgnPluginName default user server name
	DefaultUserMgnPluginName = "defaultUser"
	// DefaultStrategyMgnPluginName default strategy server name
	DefaultStrategyMgnPluginName = "defaultStrategy"
)

// Config 鉴权能力的相关配置参数
type Config struct {
	// Name 原AuthServer名称，已废弃
	Name string
	// Option 原AuthServer的option，已废弃
	// Deprecated
	Option map[string]interface{}
	// User UserOperator的相关配置
	User *UserConfig `yaml:"user"`
	// Strategy StrategyOperator的相关配置
	Strategy *StrategyConfig `yaml:"strategy"`
}

func (c *Config) SetDefault() {
	if c.User == nil {
		c.User = &UserConfig{
			Name:   DefaultUserMgnPluginName,
			Option: map[string]interface{}{},
		}
	}
	if c.Strategy == nil {
		c.Strategy = &StrategyConfig{
			Name:   DefaultStrategyMgnPluginName,
			Option: map[string]interface{}{},
		}
	}
}

// UserConfig UserOperator的相关配置
type UserConfig struct {
	// Name UserOperator的名称
	Name string `yaml:"name"`
	// Option UserOperator的option
	Option map[string]interface{} `yaml:"option"`
}

// StrategyConfig StrategyOperator的相关配置
type StrategyConfig struct {
	// Name StrategyOperator的名称
	Name string `yaml:"name"`
	// Option StrategyOperator的option
	Option map[string]interface{} `yaml:"option"`
}

var (
	// userMgnSlots 保存用户管理manager slot
	userMgnSlots = map[string]UserServer{}
	// strategyMgnSlots 保存策略管理manager slot
	strategyMgnSlots = map[string]StrategyServer{}
	once             sync.Once
	userMgn          UserServer
	strategyMgn      StrategyServer
	finishInit       bool
)

// RegisterUserServer 注册一个新的 UserServer
func RegisterUserServer(s UserServer) error {
	name := s.Name()
	if _, ok := userMgnSlots[name]; ok {
		return fmt.Errorf("UserServer=[%s] exist", name)
	}

	userMgnSlots[name] = s
	return nil
}

// GetUserServer 获取一个 UserServer
func GetUserServer() (UserServer, error) {
	if !finishInit {
		return nil, errors.New("UserServer has not done Initialize")
	}
	return userMgn, nil
}

// RegisterStrategyServer 注册一个新的 StrategyServer
func RegisterStrategyServer(s StrategyServer) error {
	name := s.Name()
	if _, ok := strategyMgnSlots[name]; ok {
		return fmt.Errorf("StrategyServer=[%s] exist", name)
	}

	strategyMgnSlots[name] = s
	return nil
}

// GetStrategyServer 获取一个 StrategyServer
func GetStrategyServer() (StrategyServer, error) {
	if !finishInit {
		return nil, errors.New("StrategyServer has not done Initialize")
	}
	return strategyMgn, nil
}

// Initialize 初始化
func Initialize(ctx context.Context, authOpt *Config, storage store.Store, cacheMgn *cache.CacheManager) error {
	var err error
	once.Do(func() {
		userMgn, strategyMgn, err = initialize(ctx, authOpt, storage, cacheMgn)
	})

	if err != nil {
		return err
	}
	return nil
}

// initialize 包裹了初始化函数，在 Initialize 的时候会在自动调用，全局初始化一次
func initialize(_ context.Context, authOpt *Config, storage store.Store,
	cacheMgr cachetypes.CacheManager) (UserServer, StrategyServer, error) {
	authOpt.SetDefault()
	name := authOpt.User.Name
	if name == "" {
		return nil, nil, errors.New("UserServer Name is empty")
	}

	namedUserMgn, ok := userMgnSlots[name]
	if !ok {
		return nil, nil, fmt.Errorf("no such UserServer plugin. name(%s)", name)
	}
	if err := namedUserMgn.Initialize(authOpt, storage, cacheMgr); err != nil {
		log.Printf("UserServer do initialize err: %s", err.Error())
		return nil, nil, err
	}

	name = authOpt.Strategy.Name
	if name == "" {
		return nil, nil, errors.New("StrategyServer Name is empty")
	}

	namedStrategyMgn, ok := strategyMgnSlots[name]
	if !ok {
		return nil, nil, fmt.Errorf("no such StrategyServer plugin. name(%s)", name)
	}
	if err := namedStrategyMgn.Initialize(authOpt, storage, cacheMgr); err != nil {
		log.Printf("StrategyServer do initialize err: %s", err.Error())
		return nil, nil, err
	}
	finishInit = true
	return namedUserMgn, namedStrategyMgn, nil
}
