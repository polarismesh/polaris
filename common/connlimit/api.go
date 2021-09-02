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

package connlimit

import (
	"errors"
	"github.com/polarismesh/polaris-server/common/log"
	"sync"
)

var (
	// 全局对象索引
	connLimitObj = make(map[string]*Listener)
	// 全局对象锁
	connLimitMu = new(sync.Mutex)
)

// 获取当前的listener
func GetLimitListener(protocol string) *Listener {
	connLimitMu.Lock()
	defer connLimitMu.Unlock()
	obj, ok := connLimitObj[protocol]
	if !ok {
		return nil
	}

	return obj
}

// 设置当前的listener
// 注意：Listener.protocol不能重复
func SetLimitListener(lis *Listener) error {
	connLimitMu.Lock()
	defer connLimitMu.Unlock()

	if _, ok := connLimitObj[lis.protocol]; ok {
		log.Errorf("[ConnLimit] protocol(%s) is existed", lis.protocol)
		return errors.New("protocol is existed")
	}

	connLimitObj[lis.protocol] = lis
	return nil
}

// 清理对应协议的链接计数
func RemoteLimitListener(protocol string) {
	connLimitMu.Lock()
	defer connLimitMu.Unlock()
	delete(connLimitObj, protocol)
}
