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

var winSignals = []os.Signal{
	syscall.SIGINT, syscall.SIGTERM,
	syscall.SIGSEGV,
}

// RunMainLoop server主循环
func RunMainLoop(servers []apiserver.Apiserver, errCh chan error) {
	defer StopServers(servers)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, winSignals...)
	for {
		select {
		case s := <-ch:
			log.Infof("catch signal(%+v), stop servers", s)
			return
		case err := <-errCh:
			log.Errorf("catch api server err: %s", err.Error())
			return
		}
	}
}
