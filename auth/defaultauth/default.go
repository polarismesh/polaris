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

package defaultauth

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/auth"
	"github.com/polarismesh/polaris-server/auth/defaultauth/cache"
	"github.com/polarismesh/polaris-server/plugin"
	"github.com/polarismesh/polaris-server/store"
)

var (
	AuthOption *AuthConfig
)

const (
	authName = "defaultAuth"
)

func init() {
	s := &defaultAuthManager{}
	_ = auth.RegisterAuthManager(s)
}

type AuthConfig struct {
	Open bool   `json:"open" xml:"open"`
	Salt string `json:"salt" xml:"salt"`
}

func DefaultAuthConfig() *AuthConfig {
	return &AuthConfig{
		Open: false,
		Salt: "polarismesh@2021",
	}
}

// defaultAuthManager
type defaultAuthManager struct {
	userSvr     *userServer
	strategySvr *authStrategyServer
	cache       *cache.AuthCache
	authPlugin  plugin.Auth
}

// Initialize 执行初始化动作
func (authMgn *defaultAuthManager) Initialize(options *auth.Config) error {
	contentBytes, err := json.Marshal(options.Option)
	if err != nil {
		return err
	}

	cfg := DefaultAuthConfig()
	if err := json.Unmarshal(contentBytes, cfg); err != nil {
		return err
	}

	AuthOption = cfg

	// 获取存储层对象
	s, err := store.GetStore()
	if err != nil {
		log.Errorf("[Auth][Server] can not get store, err: %s", err.Error())
		return errors.New("can not get store")
	}
	if s == nil {
		log.Errorf("[Auth][Server] store is null")
		return errors.New("store is null")
	}

	authCache, err := cache.NewAuthCache(s)
	if err != nil {
		return err
	}

	userSvr, err := newUserServer(s, authCache.UserCache())
	if err != nil {
		return err
	}

	strategySvr, err := newAthStrategyServer(s)
	if err != nil {
		return err
	}

	if err := authCache.Start(context.Background()); err != nil {
		return err
	}

	authPlugin := plugin.GetAuth()
	if authPlugin == nil {
		return errors.New("AuthManager needs to configure plugin.Auth plugin for permission calculation")
	}

	authMgn.userSvr = userSvr
	authMgn.strategySvr = strategySvr
	authMgn.cache = authCache

	return nil
}

// Name
func (authMgn *defaultAuthManager) Name() string {
	return authName
}

// GetUserServer
func (authMgn *defaultAuthManager) GetUserServer() auth.UserServer {
	return newUserServerWithAuth(authMgn, authMgn.userSvr)
}

// GetAuthStrategyServer
func (authMgn *defaultAuthManager) GetAuthStrategyServer() auth.AuthStrategyServer {
	return newStrategyServerWithAuth(authMgn, authMgn.strategySvr)
}
