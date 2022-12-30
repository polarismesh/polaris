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
)

// Hook Hook
type Hook interface {
	// OnAccept call when net.Conn accept
	OnAccept(conn net.Conn)
	// OnRelease call when net.Conn release
	OnRelease(conn net.Conn)
	// OnClose call when net.Listener close
	OnClose()
}

func NewHookListener(l net.Listener, hooks ...Hook) net.Listener {
	return &HookListener{
		hooks:  hooks,
		target: l,
	}
}

// HookListener net.Listener can add hook
type HookListener struct {
	hooks  []Hook
	target net.Listener
}

// Accept waits for and returns the next connection to the listener.
func (l *HookListener) Accept() (net.Conn, error) {
	conn, err := l.target.Accept()
	if err != nil {
		return nil, err
	}

	for i := range l.hooks {
		l.hooks[i].OnAccept(conn)
	}

	return &Conn{Conn: conn, listener: l}, nil
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (l *HookListener) Close() error {
	for i := range l.hooks {
		l.hooks[i].OnClose()
	}
	return l.target.Close()
}

// Addr returns the listener's network address.
func (l *HookListener) Addr() net.Addr {
	return l.target.Addr()
}
