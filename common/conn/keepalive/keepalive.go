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

package keepalive

import (
	"net"
	"time"
)

var DefaultAlivePeriodTime = 3 * time.Minute

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
// 来自net/http
type TcpKeepAliveListener struct {
	periodTime time.Duration
	*net.TCPListener
}

func NewTcpKeepAliveListener(periodTime time.Duration, ln *net.TCPListener) net.Listener {
	return &TcpKeepAliveListener{
		periodTime:  periodTime,
		TCPListener: ln,
	}
}

// Accept 来自于net/http
func (ln TcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	err = tc.SetKeepAlive(true)
	if err != nil {
		return nil, err
	}

	err = tc.SetKeepAlivePeriod(ln.periodTime)
	if err != nil {
		return nil, err
	}

	return tc, nil
}
