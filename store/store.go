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

package store

import (
	"errors"
	"fmt"
	"os"
	"sync"
)

/**
 * @brief Store的通用配置
 */
type Config struct {
	Name   string
	Option map[string]interface{}
}

var (
	StoreSlots = make(map[string]Store)
	once       = &sync.Once{}
	config     = &Config{}
)

/**
 * @brief 注册一个新的Store
 */
func RegisterStore(s Store) error {
	name := s.Name()
	if _, ok := StoreSlots[name]; ok {
		return errors.New("Store name is exist")
	}

	StoreSlots[name] = s
	return nil
}

/**
 * @brief 获取Store
 */
func GetStore() (Store, error) {
	name := config.Name
	if name == "" {
		return nil, errors.New("Store Name is empty")
	}

	store, ok := StoreSlots[name]
	if !ok {
		return nil, errors.New("No such name Store")
	}

	initialize(store)
	return store, nil
}

/**
 * @brief 设置store的conf
 */
func SetStoreConfig(conf *Config) {
	config = conf
}

/**
 * @brief 包裹了初始化函数，在GetStore的时候会在自动调用，全局初始化一次
 */
func initialize(s Store) {
	once.Do(func() {
		if err := s.Initialize(config); err != nil {
			fmt.Printf("err: %s", err.Error())
			os.Exit(-1)
		}
	})
}
