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
	"net"
	"sync"
	"time"

	"github.com/polarismesh/polaris-server/common/log"
)

// Conn 包装net.Conn
// 目的：拦截Close操作，用于listener计数的Release以及activeConns的删除
type Conn struct {
	net.Conn
	releaseOnce sync.Once
	closed      bool
	host        string
	address     string
	lastAccess  time.Time
	listener    *Listener
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
		c.listener.release(c)
	})
	return err
}

// Read 封装net.Conn Read方法，处理readTimeout的场景
func (c *Conn) Read(b []byte) (int, error) {
	if c.listener.readTimeout <= 0 {
		return c.Conn.Read(b)
	}

	c.lastAccess = time.Now()
	if err := c.Conn.SetReadDeadline(time.Now().Add(c.listener.readTimeout)); err != nil {
		log.Errorf("[connLimit][%s] connection(%s) set read deadline err: %s",
			c.listener.protocol, c.address, err.Error())
	}
	n, err := c.Conn.Read(b)
	if err == nil {
		return n, nil
	}

	if e, ok := err.(net.Error); ok && e.Timeout() {
		if time.Since(c.lastAccess) >= c.listener.readTimeout {
			log.Errorf("[connLimit][%s] read timeout(%v): %s, connection(%s) will be closed by server",
				c.listener.protocol, c.listener.readTimeout, err.Error(), c.address)
		}
	}
	return n, err
}

// 判断conn是否还有效
func (c *Conn) isValid() bool {
	return !c.closed
}
