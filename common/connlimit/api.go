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
	"sync"

	"github.com/polarismesh/polaris/common/log"
)

// limitListener limit obj for Listener
type limitListener struct {
	listenerMap map[string]*Listener // 对象索引
	mu          sync.RWMutex         // 对象锁
}

var (
	limitEntry = limitListener{
		listenerMap: make(map[string]*Listener),
	}
)

// GetLimitListener 获取当前的listener
func GetLimitListener(protocol string) *Listener {
	limitEntry.mu.RLock()
	defer limitEntry.mu.RUnlock()
	obj, ok := limitEntry.listenerMap[protocol]
	if !ok {
		return nil
	}

	return obj
}

// SetLimitListener 设置当前的listener
// 注意：Listener.protocol不能重复
func SetLimitListener(lis *Listener) error {
	limitEntry.mu.Lock()
	defer limitEntry.mu.Unlock()

	if _, ok := limitEntry.listenerMap[lis.protocol]; ok {
		log.Errorf("[ConnLimit] protocol(%s) is existed", lis.protocol)
		return errors.New("protocol is existed")
	}

	limitEntry.listenerMap[lis.protocol] = lis
	return nil
}

// RemoveLimitListener 清理对应协议的链接计数
func RemoveLimitListener(protocol string) {
	limitEntry.mu.Lock()
	defer limitEntry.mu.Unlock()

	delete(limitEntry.listenerMap, protocol)
}
