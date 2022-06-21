/**
 * Tencent is pleased to support the open source community by making CL5 available.
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

package cache

import (
	"context"

	"github.com/polarismesh/polaris-server/store"
)

// TestCacheInitialize 由于某一些模块依赖Cache，但是由于Cache模块初始化采用sync.Once，会导致单元测试之间Cache存在脏数据问题，因此为了确保单
// 元测试能够正常执行，这里将 cache.initialize 方法导出并命名为 TestCacheInitialize，仅用于单元测试初始化一个完整的 CacheManager
var (
	TestCacheInitialize = func(ctx context.Context, cacheOpt *Config, storage store.Store) error {
		if err := initialize(ctx, cacheOpt, storage); err != nil {
			finishInit = false
			return err
		}
		finishInit = true
		return Run(ctx)
	}
)
