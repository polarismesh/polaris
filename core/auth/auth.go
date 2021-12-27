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
	"errors"
	"fmt"
	"os"
	"sync"
)

type Config struct {
	Name   string
	Option map[string]interface{}
}

var (
	// AuthManagerSlots store slots
	AuthManagerSlots = make(map[string]AuthManager)
	once             = &sync.Once{}
	config           = &Config{}
)

/**
 * RegisterAuthManager 注册一个新的Store
 */
func RegisterAuthManager(s AuthManager) error {
	name := s.Name()
	if _, ok := AuthManagerSlots[name]; ok {
		return errors.New("auth manager name is exist")
	}

	AuthManagerSlots[name] = s
	return nil
}

/**
 * GetStore 获取Store
 */
func GetAuthManager() (AuthManager, error) {
	name := config.Name
	if name == "" {
		return nil, errors.New("auth manager Name is empty")
	}

	authMgn, ok := AuthManagerSlots[name]
	if !ok {
		return nil, errors.New("no such name AuthManager")
	}

	initialize(authMgn)
	return authMgn, nil
}

/**
 * SetStoreConfig 设置store的conf
 */
func SetStoreConfig(conf *Config) {
	config = conf
}

/**
 * @brief 包裹了初始化函数，在GetStore的时候会在自动调用，全局初始化一次
 */
func initialize(authMgn AuthManager) {
	once.Do(func() {
		if err := authMgn.Initialize(config); err != nil {
			fmt.Printf("auth manager do initialize err: %s", err.Error())
			os.Exit(-1)
		}
	})
}
