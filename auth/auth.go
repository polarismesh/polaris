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

	"github.com/polarismesh/polaris-server/cache"
	"github.com/polarismesh/polaris-server/store"
)

// Config 鉴权能力的相关配置参数
type Config struct {
	Name   string
	Option map[string]interface{}
}

var (
	// Slots store slots
	Slots      = map[string]AuthServer{}
	once       sync.Once
	authSvr    AuthServer
	finishInit bool
)

// RegisterAuthServer 注册一个新的 AuthManager
func RegisterAuthServer(s AuthServer) error {
	name := s.Name()
	if _, ok := Slots[name]; ok {
		return errors.New("auth manager name is exist")
	}

	Slots[name] = s
	return nil
}

// GetAuthServer 获取一个 AuthManager
func GetAuthServer() (AuthServer, error) {
	if !finishInit {
		return nil, errors.New("AuthServer has not done Initialize")
	}
	return authSvr, nil
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
	name := authOpt.Name
	if name == "" {
		return errors.New("auth manager Name is empty")
	}

	mgn, ok := Slots[name]
	if !ok {
		return errors.New("no such name AuthManager")
	}

	authSvr = mgn

	if err := authSvr.Initialize(authOpt, storage, cacheMgn); err != nil {
		log.Printf("auth manager do initialize err: %s", err.Error())
		return err
	}
	return nil
}
