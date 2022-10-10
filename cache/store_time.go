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

package cache

import (
	"context"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// watchStoreTime The timestamp change of the storage layer, whether the clock is dialed in the detection
func (nc *CacheManager) watchStoreTime(ctx context.Context) {

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	preStoreTime, err := nc.storage.GetUnixSecond()
	if err != nil {
		log.Error("[Store][Time] watch store time", zap.Error(err))
	}

	for {
		select {
		case <-ticker.C:

			storeSec, err := nc.storage.GetUnixSecond()
			if err != nil {
				log.Error("[Store][Time] watch store time", zap.Error(err))
				continue
			}
			// 防止时间回退
			if preStoreTime != 0 && preStoreTime > storeSec {
				atomic.StoreInt64(&nc.storeTimeDiffSec, preStoreTime-storeSec)
			} else {
				preStoreTime = storeSec
				atomic.StoreInt64(&nc.storeTimeDiffSec, 0)
			}

		case <-ctx.Done():
			return
		}
	}

}
