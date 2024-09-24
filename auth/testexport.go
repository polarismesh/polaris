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

	cachetypes "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/store"
)

// TestInitialize 包裹了初始化函数，在 Initialize 的时候会在自动调用，全局初始化一次
func TestInitialize(ctx context.Context, authOpt *Config, storage store.Store,
	cacheMgn cachetypes.CacheManager) (UserServer, StrategyServer, error) {
	userSvr, strategySvr, err := initialize(ctx, authOpt, storage, cacheMgn)
	if err != nil {
		return nil, nil, err
	}
	userMgn = userSvr
	strategyMgn = strategySvr
	return userSvr, strategySvr, nil
}

func TestClean() {
	userMgn = nil
	strategyMgn = nil
}
