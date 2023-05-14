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
	"log"
	"sync"

	"github.com/polarismesh/polaris/cache"
	"github.com/polarismesh/polaris/store"
)

// Config 鉴权能力的相关配置参数
type Config struct {
	// Name 原AuthServer名称，已废弃
	Name string
	// Option 原AuthServer的option，已废弃
	Option map[string]interface{}

	// User UserOperator的相关配置
	User UserConfig `yaml:"user"`
	// Strategy StrategyOperator的相关配置
	Strategy StrategyConfig `yaml:"strategy"`
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
	// UserMgnSlots 保存用户管理manager slot
	UserMgnSlots = map[string]UserOperator{}
	// StrategyMgnSlots 保存策略管理manager slot
	StrategyMgnSlots = map[string]StrategyOperator{}
	once             sync.Once
	userMgn          UserOperator
	strategyMgn      StrategyOperator
	finishInit       bool
)

// RegisterUserOperator 注册一个新的 UserManager
func RegisterUserOperator(s UserOperator) error {
	name := s.Name()
	if _, ok := UserMgnSlots[name]; ok {
		return errors.New("user manager name is exist")
	}

	UserMgnSlots[name] = s
	return nil
}

// GetUserOperator 获取一个 UserManager
func GetUserOperator() (UserOperator, error) {
	if !finishInit {
		return nil, errors.New("AuthServer has not done Initialize")
	}
	return userMgn, nil
}

// RegisterStrategyOperator 注册一个新的 StrategyManager
func RegisterStrategyOperator(s StrategyOperator) error {
	name := s.Name()
	if _, ok := StrategyMgnSlots[name]; ok {
		return errors.New("user manager name is exist")
	}

	StrategyMgnSlots[name] = s
	return nil
}

// GetStrategyOperator 获取一个 StrategyManager
func GetStrategyOperator() (StrategyOperator, error) {
	if !finishInit {
		return nil, errors.New("AuthServer has not done Initialize")
	}
	return strategyMgn, nil
}

// Initialize 初始化
func Initialize(ctx context.Context, authOpt *Config, storage store.Store, cacheMgn *cache.CacheManager) error {
	var err error
	once.Do(func() {
		err = initialize(ctx, authOpt, storage, cacheMgn)
	})

	if err != nil {
		return err
	}

	finishInit = true
	return nil
}

// initialize 包裹了初始化函数，在 Initialize 的时候会在自动调用，全局初始化一次
func initialize(_ context.Context, authOpt *Config, storage store.Store, cacheMgn *cache.CacheManager) error {
	name := authOpt.User.Name
	if name == "" {
		return errors.New("user manager Name is empty")
	}

	namedUserMgn, ok := UserMgnSlots[name]
	if !ok {
		return errors.New("no such name UserManager")
	}

	userMgn = namedUserMgn

	if err := userMgn.Initialize(authOpt, storage, cacheMgn); err != nil {
		log.Printf("auth manager do initialize err: %s", err.Error())
		return err
	}

	name = authOpt.Strategy.Name
	if name == "" {
		return errors.New("strategy manager Name is empty")
	}

	namedStrategyMgn, ok := StrategyMgnSlots[name]
	if !ok {
		return errors.New("no such name StrategyManager")
	}

	strategyMgn = namedStrategyMgn

	if err := strategyMgn.Initialize(authOpt, storage, cacheMgn); err != nil {
		log.Printf("auth manager do initialize err: %s", err.Error())
		return err
	}
	return nil
}
