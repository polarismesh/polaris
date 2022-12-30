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

package connhook

import (
	"net"
	"sync"
)

// Conn 包装net.Conn
// 目的：拦截Close操作，用于listener计数的Release以及activeConns的删除
type Conn struct {
	net.Conn
	releaseOnce sync.Once
	closed      bool
	listener    *HookListener
}

// Close 包装net.Conn.Close, 用于连接计数
func (c *Conn) Close() error {
	if c.closed {
		return nil
	}

	err := c.Conn.Close()
	c.releaseOnce.Do(func() {
		// 调用监听的listener，释放计数以及activeConns
		// 保证只执行一次
		c.closed = true

		for i := range c.listener.hooks {
			c.listener.hooks[i].OnRelease(c)
		}

	})
	return err
}
