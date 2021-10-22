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

package grpc

import (
	"context"
	"fmt"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/connlimit"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/plugin"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// BaseGrpcServer base utilities and functions for GRPC Connector
type BaseGrpcServer struct {
	listenIP        string
	listenPort      uint32
	connLimitConfig *connlimit.Config
	start           bool
	restart         bool
	exitCh          chan struct{}

	server     *grpc.Server
	statis     plugin.Statis
	ratelimit  plugin.Ratelimit
	OpenMethod map[string]bool
}

/**
 * @brief 获取Server的协议
 */
func (b *BaseGrpcServer) GetPort() uint32 {
	return b.listenPort
}

/**
 * @brief 初始化GRPC API服务器
 */
func (b *BaseGrpcServer) Initialize(ctx context.Context, option map[string]interface{}, protocol string) error {
	b.listenIP = option["listenIP"].(string)
	b.listenPort = uint32(option["listenPort"].(int))

	if raw, _ := option["connLimit"].(map[interface{}]interface{}); raw != nil {
		connConfig, err := connlimit.ParseConnLimitConfig(raw)
		if err != nil {
			return err
		}
		b.connLimitConfig = connConfig
	}
	if ratelimit := plugin.GetRatelimit(); ratelimit != nil {
		log.Infof("%s server open the ratelimit", protocol)
		b.ratelimit = ratelimit
	}

	return nil
}

// 关闭GRPC
func (b *BaseGrpcServer) Stop(protocol string) {
	connlimit.RemoveLimitListener(protocol)
	if b.server != nil {
		b.server.Stop()
	}
}

/**
 * @brief 启动GRPC API服务器
 */
func (b *BaseGrpcServer) Run(errCh chan error, protocol string, initServer func(*grpc.Server) error) {
	log.Infof("start %s server", protocol)
	b.exitCh = make(chan struct{})
	b.start = true
	defer func() {
		close(b.exitCh)
		b.start = false
	}()

	address := fmt.Sprintf("%v:%v", b.listenIP, b.listenPort)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Errorf("%v", err)
		errCh <- err
		return
	}
	defer listener.Close()

	// 如果设置最大连接数
	if b.connLimitConfig != nil && b.connLimitConfig.OpenConnLimit {
		log.Infof("grpc server use max connection limit: %d, grpc max limit: %d",
			b.connLimitConfig.MaxConnPerHost, b.connLimitConfig.MaxConnLimit)
		listener, err = connlimit.NewListener(listener, protocol, b.connLimitConfig)
		if err != nil {
			log.Errorf("conn limit init err: %s", err.Error())
			errCh <- err
			return
		}

	}

	server := grpc.NewServer(
		grpc.UnaryInterceptor(b.unaryInterceptor),
		grpc.StreamInterceptor(b.streamInterceptor),
	)

	if err = initServer(server); nil != err {
		errCh <- err
		return
	}
	b.server = server

	b.statis = plugin.GetStatis()

	if err := server.Serve(listener); err != nil {
		log.Errorf("%v", err)
		errCh <- err
		return
	}

	log.Infof("%s server stop", protocol)
}

/**
 * @brief 虚拟Stream
 * @note 继承ServerStream
 */
type VirtualStream struct {
	Method        string
	ClientAddress string
	ClientIP      string
	UserAgent     string
	RequestID     string

	grpc.ServerStream
	Code int

	preprocess  PreProcessFunc
	postprocess PostProcessFunc

	StartTime time.Time
}

/**
 * @brief VirtualStream接收消息函数
 * @note 拦截ServerSteam接收消息函数
 */
func (v *VirtualStream) RecvMsg(m interface{}) error {
	err := v.ServerStream.RecvMsg(m)
	if err == io.EOF {
		return err
	}

	if err == nil {
		err = v.preprocess(v, false)
	} else {
		v.Code = -1
	}

	return err
}

/**
 * @brief VirtualStream发送消息函数
 * @note 拦截ServerSteam发送消息函数
 */
func (v *VirtualStream) SendMsg(m interface{}) error {
	v.postprocess(v, m)

	err := v.ServerStream.SendMsg(m)
	if err != nil {
		v.Code = -2
	}

	return err
}

/**
 * @brief 创建VirtualStream
 */
func newVirtualStream(ctx context.Context, method string, stream grpc.ServerStream,
	preprocess PreProcessFunc, postprocess PostProcessFunc) *VirtualStream {
	var clientAddress string
	var clientIP string
	var userAgent string
	var requestID string

	peer, exist := peer.FromContext(ctx)
	if exist {
		clientAddress = peer.Addr.String()
		// 解析获取clientIP
		items := strings.Split(clientAddress, ":")
		if len(items) == 2 {
			clientIP = items[0]
		}
	}

	meta, exist := metadata.FromIncomingContext(ctx)
	if exist {
		agents := meta["user-agent"]
		if len(agents) > 0 {
			userAgent = agents[0]
		}

		ids := meta["request-id"]
		if len(ids) > 0 {
			requestID = ids[0]
		}
	}

	return &VirtualStream{
		Method:        method,
		ClientAddress: clientAddress,
		ClientIP:      clientIP,
		UserAgent:     userAgent,
		RequestID:     requestID,
		ServerStream:  stream,
		Code:          0,
		preprocess:    preprocess,
		postprocess:   postprocess,
	}
}

var notPrintableMethods = map[string]bool{
	"/v1.PolarisGRPC/Heartbeat": true,
}

/**
 * @brief 在接收和回复请求时统一处理
 */
func (b *BaseGrpcServer) unaryInterceptor(ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (rsp interface{}, err error) {

	stream := newVirtualStream(ctx, info.FullMethod, nil, b.preprocess, b.postprocess)

	func() {
		_, ok := notPrintableMethods[info.FullMethod]
		var printable = !ok
		if err := b.preprocess(stream, printable); err != nil {
			return
		}

		// 判断是否允许访问
		if ok := b.AllowAccess(stream.Method); !ok {
			rsp = api.NewResponse(api.ClientAPINotOpen)
			return
		}

		// handler执行前，限流
		if code := b.EnterRatelimit(stream.ClientIP, stream.Method); code != api.ExecuteSuccess {
			rsp = api.NewResponse(code)
			return
		}
		rsp, err = handler(ctx, req)
	}()

	b.postprocess(stream, rsp)

	return
}

/**
 * @brief 在接收和回复请求时统一处理
 */
func (b *BaseGrpcServer) streamInterceptor(srv interface{}, ss grpc.ServerStream,
	info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {

	stream := newVirtualStream(ss.Context(), info.FullMethod, ss, b.preprocess, b.postprocess)

	err = handler(srv, stream)
	if err != nil { // 存在非EOF读错误或者写错误
		log.Error(err.Error(),
			zap.String("client-address", stream.ClientAddress),
			zap.String("user-agent", stream.UserAgent),
			zap.String("request-id", stream.RequestID),
			zap.String("method", stream.Method),
		)

		b.statis.AddAPICall(stream.Method, stream.Code, 0)
	}

	return
}

// 请求预处理函数定义
type PreProcessFunc func(stream *VirtualStream, isPrint bool) error

/**
 * @brief 请求预处理
 */
func (b *BaseGrpcServer) preprocess(stream *VirtualStream, isPrint bool) error {
	// 设置开始时间
	stream.StartTime = time.Now()

	if isPrint {
		// 打印请求
		log.Info("receive request",
			zap.String("client-address", stream.ClientAddress),
			zap.String("user-agent", stream.UserAgent),
			zap.String("request-id", stream.RequestID),
			zap.String("method", stream.Method),
		)
	}

	return nil
}

// 请求后处理函数定义
type PostProcessFunc func(stream *VirtualStream, m interface{})

/**
 * @brief 请求后处理
 */
func (b *BaseGrpcServer) postprocess(stream *VirtualStream, m interface{}) {
	var code int
	if response, ok := m.(api.ResponseMessage); ok {
		code = api.CalcCode(response)

		// 打印回复
		if code != http.StatusOK {
			log.Error(response.String(),
				zap.String("client-address", stream.ClientAddress),
				zap.String("user-agent", stream.UserAgent),
				zap.String("request-id", stream.RequestID),
				zap.String("method", stream.Method),
			)
		}
	} else {
		code = stream.Code
		// 打印回复
		if code != int(codes.OK) {
			log.Error(response.String(),
				zap.String("client-address", stream.ClientAddress),
				zap.String("user-agent", stream.UserAgent),
				zap.String("request-id", stream.RequestID),
				zap.String("method", stream.Method),
			)
		}
	}

	// 接口调用统计
	now := time.Now()
	diff := now.Sub(stream.StartTime)

	// 打印耗时超过1s的请求
	if diff > time.Second {
		log.Info("handling time > 1s",
			zap.String("client-address", stream.ClientAddress),
			zap.String("user-agent", stream.UserAgent),
			zap.String("request-id", stream.RequestID),
			zap.String("method", stream.Method),
			zap.Duration("handling-time", diff),
		)
	}

	_ = b.statis.AddAPICall(stream.Method, code, diff.Nanoseconds())
}

// restart
func (b *BaseGrpcServer) Restart(
	initialize func() error, run func(), protocol string, option map[string]interface{}) error {
	log.Infof("restart %s server with new config: %+v", protocol, option)

	b.restart = true
	b.Stop(protocol)
	if b.start {
		<-b.exitCh
	}

	log.Infof("old %s server has stopped, begin restarting it", protocol)
	if err := initialize(); err != nil {
		log.Errorf("restart %s server err: %s", protocol, err.Error())
		return err
	}

	log.Infof("init %s server successfully, restart it", protocol)
	b.restart = false
	go run()

	return nil
}

// 限流
func (b *BaseGrpcServer) EnterRatelimit(ip string, method string) uint32 {
	if b.ratelimit == nil {
		return api.ExecuteSuccess
	}

	//ipRatelimit
	if ok := b.ratelimit.Allow(plugin.IPRatelimit, ip); !ok {
		log.Error("[grpc] ip ratelimit is not allow", zap.String("client-ip", ip),
			zap.String("method", method))
		return api.IPRateLimit
	}
	// apiRatelimit
	if ok := b.ratelimit.Allow(plugin.APIRatelimit, method); !ok {
		log.Error("[grpc] api rate limit is not allow", zap.String("client-ip", ip),
			zap.String("method", method))
		return api.APIRateLimit
	}

	return api.ExecuteSuccess
}

// 限制访问
func (b *BaseGrpcServer) AllowAccess(method string) bool {
	if len(b.OpenMethod) == 0 {
		return true
	}
	_, ok := b.OpenMethod[method]
	return ok
}
