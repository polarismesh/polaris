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

package l5pbserver

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/polarismesh/polaris-server/apiserver"
	"github.com/polarismesh/polaris-server/common/api/l5"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/naming"
	"github.com/polarismesh/polaris-server/plugin"
	"go.uber.org/zap"
)

const (
	success            int32  = 100
	sohSize            int    = 2
	headSize           int    = 6
	maxSize            int    = 1024 * 1024 * 2
	defaultClusterName string = "cl5.discover"
)

// 每个链接，封装为一个请求
type cl5Request struct {
	conn       net.Conn
	start      time.Time
	clientAddr string
	cmd        int32
	code       l5Code
}

/**
 * @brief CL5 API服务器
 */
type L5pbserver struct {
	listenIP    string
	listenPort  uint32
	clusterName string // 集群名

	listener     net.Listener
	namingServer naming.DiscoverServer
	statis       plugin.Statis
}

/**
 * @brief 获取端口
 */
func (l *L5pbserver) GetPort() uint32 {
	return l.listenPort
}

/**
 * @brief 获取Server的协议
 */
func (l *L5pbserver) GetProtocol() string {
	return "l5pb"
}

/**
 * @brief 初始化CL5 API服务器
 */
func (l *L5pbserver) Initialize(_ context.Context, option map[string]interface{},
	_ map[string]apiserver.APIConfig) error {
	l.listenIP = option["listenIP"].(string)
	l.listenPort = uint32(option["listenPort"].(int))
	// 获取当前集群
	l.clusterName = defaultClusterName
	if clusterName, _ := option["clusterName"].(string); clusterName != "" {
		l.clusterName = clusterName
	}

	return nil
}

/**
 * @brief 启动CL5 API服务器
 */
func (l *L5pbserver) Run(errCh chan error) {
	log.Infof("start l5pbserver")

	var err error
	// 引入功能模块和插件
	l.namingServer, err = naming.GetServer()
	if err != nil {
		log.Errorf("%v", err)
		errCh <- err
		return
	}
	l.statis = plugin.GetStatis()

	// 初始化 l5pb server
	address := fmt.Sprintf("%v:%v", l.listenIP, l.listenPort)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Errorf("listen error: %v", err)
		errCh <- err
		return
	}
	l.listener = listener

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Errorf("accept error: %v", err)
			errCh <- err
			return
		}
		// log.Infof("new connect: %v", conn.RemoteAddr())
		go l.handleConnection(conn)
	}
}

// stop server
func (l *L5pbserver) Stop() {
	if l.listener != nil {
		_ = l.listener.Close()
	}
}

// restart server
func (l *L5pbserver) Restart(_ map[string]interface{}, _ map[string]apiserver.APIConfig,
	_ chan error) error {
	return nil
}

/**
 * @brief 请求预处理：限频/鉴权
 */
func (l *L5pbserver) PreProcess(req *cl5Request) bool {
	log.Info("[Cl5] handle request", zap.String("ClientAddr", req.clientAddr), zap.Int32("Cmd", req.cmd))
	var result = true
	// 访问频率限制

	// 访问权限控制

	return result
}

/**
 * @brief 请求后处理：统计/告警
 */
func (l *L5pbserver) PostProcess(req *cl5Request) {
	now := time.Now()
	// 统计
	cmdStr, ok := l5.CL5_CMD_name[req.cmd]
	if !ok {
		cmdStr = "Unrecognizable_Cmd"
	}

	diff := now.Sub(req.start)
	// 打印耗时超过1s的请求
	if diff > time.Second {
		log.Info("handling time > 1s",
			zap.String("client-addr", req.clientAddr),
			zap.String("cmd", cmdStr),
			zap.Duration("handling-time", diff),
		)
	}
	_ = l.statis.AddAPICall(cmdStr, "HTTP", int(req.code), diff.Nanoseconds())
	// 告警
}
