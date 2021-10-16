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
	"github.com/polarismesh/polaris-server/common/log"
	"sync/atomic"
	"time"
)

const adjustInterval = 10 * time.Second

// TimeAdjuster adjust the seconds from database
type TimeAdjuster struct {
	ctx  context.Context
	diff int64
}

func newTimeAdjuster(ctx context.Context) *TimeAdjuster {
	adjuster := &TimeAdjuster{ctx: ctx}
	go adjuster.doTimeAdjust()
	return adjuster
}

func (t *TimeAdjuster) doTimeAdjust() {
	t.calcDiff()
	ticker := time.NewTicker(adjustInterval)
	defer ticker.Stop()
	for {
		select {
		case <-t.ctx.Done():
			log.Infof("[healthcheck]time adjuster has been stopped")
			return
		case <-ticker.C:
			t.calcDiff()
		}
	}
}

func (t *TimeAdjuster) calcDiff() {
	curTimeSecond, err := server.storage.GetNow()
	if nil != err {
		log.Errorf("[healthcheck]fail to get now from store, err is %s", err.Error())
		return
	}
	sysNow := time.Now().Unix()
	diff := sysNow - curTimeSecond
	atomic.StoreInt64(&t.diff, diff)
}

// GetDiff get diff time between store and current PC
func (t *TimeAdjuster) GetDiff() int64 {
	return atomic.LoadInt64(&t.diff)
}
