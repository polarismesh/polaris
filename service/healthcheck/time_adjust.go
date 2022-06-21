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

package healthcheck

import (
	"context"
	"sync/atomic"
	"time"
)

const adjustInterval = 60 * time.Second

// TimeAdjuster adjust the seconds from databases
type TimeAdjuster struct {
	diff int64
}

func newTimeAdjuster(ctx context.Context) *TimeAdjuster {
	adjuster := &TimeAdjuster{}
	go adjuster.doTimeAdjust(ctx)
	return adjuster
}

func (t *TimeAdjuster) doTimeAdjust(ctx context.Context) {
	t.calcDiff()
	ticker := time.NewTicker(adjustInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Infof("[Health Check] time adjuster has been stopped")
			return
		case <-ticker.C:
			t.calcDiff()
		}
	}
}

func (t *TimeAdjuster) calcDiff() {
	curTimeSecond, err := server.storage.GetUnixSecond()
	if err != nil {
		log.Errorf("[Health Check] fail to get now from store, err is %s", err.Error())
		return
	}
	if curTimeSecond == 0 {
		return
	}
	sysNow := time.Now().Unix()
	diff := sysNow - curTimeSecond
	if diff != 0 {
		log.Infof("[Health Check] time diff from now is %d", diff)
	}
	atomic.StoreInt64(&t.diff, diff)
}

// GetDiff get diff time between store and current PC
func (t *TimeAdjuster) GetDiff() int64 {
	return atomic.LoadInt64(&t.diff)
}
