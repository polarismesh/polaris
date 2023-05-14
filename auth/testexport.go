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

	"github.com/polarismesh/polaris/cache"
	"github.com/polarismesh/polaris/store"
)

// TestInitialize 包裹了初始化函数，在 Initialize 的时候会在自动调用，全局初始化一次
func TestInitialize(_ context.Context, authOpt *Config, storage store.Store,
	cacheMgn *cache.CacheManager) (UserOperator, StrategyOperator, error) {

	name := authOpt.User.Name
	if name == "" {
		return nil, nil, errors.New("user manager Name is empty")
	}
	namedUserMgn, ok := UserMgnSlots[name]
	if !ok {
		return nil, nil, errors.New("no such name UserManager")
	}
	userMgn = namedUserMgn

	name = authOpt.Strategy.Name
	if name == "" {
		return nil, nil, errors.New("strategy manager Name is empty")
	}
	namedStrategyMgn, ok := StrategyMgnSlots[name]
	if !ok {
		return nil, nil, errors.New("no such name StrategyManager")
	}
	strategyMgn = namedStrategyMgn

	finishInit = true

	if err := userMgn.Initialize(authOpt, storage, cacheMgn); err != nil {
		log.Printf("user manager do initialize err: %s", err.Error())
		return nil, nil, err
	}
	if err := strategyMgn.Initialize(authOpt, storage, cacheMgn); err != nil {
		log.Printf("user manager do initialize err: %s", err.Error())
		return nil, nil, err
	}
	return userMgn, strategyMgn, nil
}
