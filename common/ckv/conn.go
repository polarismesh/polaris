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

package ckv

import (
	"github.com/gomodule/redigo/redis"
)

/**
 * Conn ckv连接结构体
 */
type Conn struct {
	index int // conn在连接池的序号
	addr  string
	conn  redis.Conn
}

/**
 * newConn 新建ckv连接
 */
func newConn(index int, addr, passwd string) (*Conn, error) {
	c, err := redis.Dial("tcp", addr, redis.DialPassword(passwd))
	if err != nil {
		return nil, err
	}

	return &Conn{index, addr, c}, nil
}

/**
 * Addr 返回ckv地址
 */
func (c *Conn) Addr() string {
	return c.addr
}

/**
 *Get Get请求
 */
func (c *Conn) Get(key string) (string, error) {
	return redis.String(c.conn.Do("GET", key))
}

/**
 * Set Set请求
 */
func (c *Conn) Set(key, value string) (err error) {
	_, err = c.conn.Do("SET", key, value)
	return
}

/**
 * Del Del请求
 */
func (c *Conn) Del(key string) (err error) {
	_, err = c.conn.Do("DEL", key)
	return
}
