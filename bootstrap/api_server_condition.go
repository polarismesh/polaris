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

package bootstrap

import (
	"errors"
	"sync"
	"time"

	"github.com/polarismesh/polaris/common/log"
)

// ApiServerCondition 启动条件
type ApiServerCondition struct {
	cond *sync.Cond
	cnt  int // 已经成功listen的ApiServer的数量
}

// NewApiServerCond 初始化一个 ApiServerCondition指针
func NewApiServerCond() *ApiServerCondition {
	log.Infof("NewApiServerCond \n")
	var condLock sync.Mutex
	return &ApiServerCondition{cond: sync.NewCond(&condLock)}
}

// Incr 增加Condition的Cnt
func (cc *ApiServerCondition) Incr(serverName string) {
	log.Infof("ApiServerCondition1 -> server:%s Cnt:%d \n", serverName, cc.cnt)
	cc.cond.L.Lock()
	cc.cnt++
	cc.cond.L.Unlock()
	log.Infof("ApiServerCondition2 -> server:%s Cnt:%d \n", serverName, cc.cnt)
	cc.cond.Broadcast()
}

func (cc *ApiServerCondition) Set(i int) {
	log.Infof("ApiServerCondition1 -> Set:%d Cnt:%d \n", i, cc.cnt)
	cc.cond.L.Lock()
	cc.cnt = i
	cc.cond.L.Unlock()
	log.Infof("ApiServerCondition2 -> Set:%d Cnt:%d \n", i, cc.cnt)
	cc.cond.Broadcast()
}

func (cc *ApiServerCondition) GetCnt() int {
	log.Infof("ApiServerCondition->GetCnt cnt:%d \n", cc.cnt)
	return cc.cnt

}

func (cc *ApiServerCondition) Wait(cnt int) error {
	log.Infof("ApiServerCondition->wait cnt:%d \n", cnt)
	cc.cond.L.Lock()

	// 超时
	time.AfterFunc(3*time.Second, func() {
		if cc.cnt < cnt {
			log.Warnf("ApiServerCondition->wait timeout, current cnt:%d \n", cc.cnt)
			cc.Set(cnt)
		}
	})

	for cc.cnt < cnt {
		log.Infof("BeforeApiServerCondition->wait cnt:%d \n", cc.cnt)
		cc.cond.Wait()
	}
	cc.cond.L.Unlock()

	if cc.cnt > cnt {
		log.Infof("[ERROR] ApiServerCondition timeout \n")
		return errors.New("[ERROR] ApiServerCondition timeout")
	}
	return nil
}
