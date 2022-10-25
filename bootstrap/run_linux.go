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
	"os"
	"os/signal"
	"syscall"

	"github.com/polarismesh/polaris/apiserver"
	"github.com/polarismesh/polaris/common/log"
)

var (
	linuxSignals = []os.Signal{
		syscall.SIGINT, syscall.SIGTERM,
		syscall.SIGSEGV, syscall.SIGUSR1,
	}
	ch = make(chan os.Signal, 1)
)

// WaitSignal 等待信号量或err chan 从而执行restart或平滑退出
func WaitSignal(servers []apiserver.Apiserver, errCh chan error) {
	defer StopServers(servers)

	// 监听信号量
	signal.Notify(ch, darwinSignals...)

label:
	for {
		select {
		case s := <-ch:
			if s2, ok := s.(syscall.Signal); ok && s2 == syscall.SIGUSR1 { // 重启信号量
				if err := RestartServers(errCh); err != nil { // 重启失败，直接退出
					log.Errorf("restart servers err: %s", err.Error())
					return
				}

				log.Infof("restart servers success: %s", s.String())
				// 重启成功，就需要监听信号量然后执行相应的操作
				signal.Notify(ch, darwinSignals...)
				break label
			}

			log.Infof("catch signal(%s), stop servers", s.String())
			return
		case err := <-errCh:
			log.Errorf("catch api server err: %s", err.Error())
			return
		default:
			// server is running...
		}
	}
}
