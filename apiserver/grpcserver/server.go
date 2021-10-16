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

package grpcserver

import (
	"context"
	"fmt"
	"github.com/polarismesh/polaris-server/healthcheck"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/polarismesh/polaris-server/apiserver"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/connlimit"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/naming"
	"github.com/polarismesh/polaris-server/plugin"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

// GRPCServer GRPC API服务器
type GRPCServer struct {
	listenIP        string
	listenPort      uint32
	connLimitConfig *connlimit.Config
	start           bool
	restart         bool
	exitCh          chan struct{}

	server            *grpc.Server
	namingServer      *naming.Server
	healthCheckServer *healthcheck.Server
	statis            plugin.Statis
	ratelimit         plugin.Ratelimit

	openAPI    map[string]apiserver.APIConfig
	openMethod map[string]bool
}

// GetPort 获取端口
func (g *GRPCServer) GetPort() uint32 {
	return g.listenPort
}

// GetProtocol 获取Server的协议
func (g *GRPCServer) GetProtocol() string {
	return "grpc"
}

// Initialize 初始化GRPC API服务器
func (g *GRPCServer) Initialize(_ context.Context, option map[string]interface{},
	api map[string]apiserver.APIConfig) error {
	g.listenIP = option["listenIP"].(string)
	g.listenPort = uint32(option["listenPort"].(int))
	g.openAPI = api

	if raw, _ := option["connLimit"].(map[interface{}]interface{}); raw != nil {
		connConfig, err := connlimit.ParseConnLimitConfig(raw)
		if err != nil {
			return err
		}
		g.connLimitConfig = connConfig
	}
	if rateLimit := plugin.GetRatelimit(); rateLimit != nil {
		log.Infof("grpc server open the ratelimit")
		g.ratelimit = rateLimit
	}

	return nil
}

// Run 启动GRPC API服务器
func (g *GRPCServer) Run(errCh chan error) {
	log.Infof("start grpcserver")
	g.exitCh = make(chan struct{})
	g.start = true
	defer func() {
		close(g.exitCh)
		g.start = false
	}()

	address := fmt.Sprintf("%v:%v", g.listenIP, g.listenPort)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Errorf("%v", err)
		errCh <- err
		return
	}
	defer listener.Close()

	// 如果设置最大连接数
	if g.connLimitConfig != nil && g.connLimitConfig.OpenConnLimit {
		log.Infof("grpc server use max connection limit: %d, grpc max limit: %d",
			g.connLimitConfig.MaxConnPerHost, g.connLimitConfig.MaxConnLimit)
		listener, err = connlimit.NewListener(listener, g.GetProtocol(), g.connLimitConfig)
		if err != nil {
			log.Errorf("conn limit init err: %s", err.Error())
			errCh <- err
			return
		}

	}

	server := grpc.NewServer(
		grpc.UnaryInterceptor(g.unaryInterceptor),
		grpc.StreamInterceptor(g.streamInterceptor),
	)

	for name, config := range g.openAPI {
		switch name {
		case "client":
			if config.Enable {
				api.RegisterPolarisGRPCServer(server, g)
				openMethod, getErr := apiserver.GetClientOpenMethod(config.Include, g.GetProtocol())
				if getErr != nil {
					errCh <- getErr
					return
				}
				g.openMethod = openMethod
			}
		default:
			log.Errorf("api %s does not exist in grpcserver", name)
			errCh <- fmt.Errorf("api %s does not exist in grpcserver", name)
			return
		}
	}
	g.server = server

	// 引入功能模块和插件
	g.namingServer, err = naming.GetServer()
	if err != nil {
		log.Errorf("%v", err)
		errCh <- err
		return
	}
	g.healthCheckServer, err = healthcheck.GetServer()
	if err != nil {
		log.Errorf("%v", err)
		errCh <- err
		return
	}
	g.namingServer.Cache().Instance().AddListener(g.healthCheckServer.CacheProvider())

	g.statis = plugin.GetStatis()

	if err := server.Serve(listener); err != nil {
		log.Errorf("%v", err)
		errCh <- err
		return
	}

	log.Infof("grpcserver stop")
}

// Stop 关闭GRPC
func (g *GRPCServer) Stop() {
	connlimit.RemoveLimitListener(g.GetProtocol())
	if g.server != nil {
		g.server.Stop()
	}
}

// Restart 重启Server
func (g *GRPCServer) Restart(option map[string]interface{}, api map[string]apiserver.APIConfig,
	errCh chan error) error {
	log.Infof("restart grpc server with new config: %+v", option)

	g.restart = true
	g.Stop()
	if g.start {
		<-g.exitCh
	}

	log.Infof("old grpc server has stopped, begin restarting it")
	if err := g.Initialize(context.Background(), option, api); err != nil {
		log.Errorf("restart grpc server err: %s", err.Error())
		return err
	}

	log.Infof("init grpc server successfully, restart it")
	g.restart = false
	go g.Run(errCh)

	return nil
}

// VirtualStream 虚拟Stream
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

// RecvMsg VirtualStream接收消息函数
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

// SendMsg VirtualStream发送消息函数
func (v *VirtualStream) SendMsg(m interface{}) error {
	v.postprocess(v, m)

	err := v.ServerStream.SendMsg(m)
	if err != nil {
		v.Code = -2
	}

	return err
}

func newVirtualStream(ctx context.Context, method string, stream grpc.ServerStream,
	preprocess PreProcessFunc, postprocess PostProcessFunc) *VirtualStream {
	var clientAddress string
	var clientIP string
	var userAgent string
	var requestID string

	p, exist := peer.FromContext(ctx)
	if exist {
		clientAddress = p.Addr.String()
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

// unaryInterceptor 在接收和回复请求时统一处理
func (g *GRPCServer) unaryInterceptor(ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (rsp interface{}, err error) {

	stream := newVirtualStream(ctx, info.FullMethod, nil, g.preprocess, g.postprocess)

	func() {
		var isPrint = !strings.Contains(info.FullMethod, "Heartbeat")
		if err := g.preprocess(stream, isPrint); err != nil {
			return
		}

		// 判断是否允许访问
		if ok := g.allowAccess(stream.Method); !ok {
			rsp = api.NewResponse(api.ClientAPINotOpen)
			return
		}

		// handler执行前，限流
		if code := g.enterRateLimit(stream.ClientIP, stream.Method); code != api.ExecuteSuccess {
			rsp = api.NewResponse(code)
			return
		}
		rsp, err = handler(ctx, req)
	}()

	g.postprocess(stream, rsp)

	return
}

// streamInterceptor 在接收和回复请求时统一处理
func (g *GRPCServer) streamInterceptor(srv interface{}, ss grpc.ServerStream,
	info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {

	stream := newVirtualStream(ss.Context(), info.FullMethod, ss, g.preprocess, g.postprocess)

	err = handler(srv, stream)
	if err != nil { // 存在非EOF读错误或者写错误
		log.Error(err.Error(),
			zap.String("client-address", stream.ClientAddress),
			zap.String("user-agent", stream.UserAgent),
			zap.String("request-id", stream.RequestID),
			zap.String("method", stream.Method),
		)

		g.statis.AddAPICall(stream.Method, stream.Code, 0)
	}

	return
}

// PreProcessFunc 请求预处理函数定义
type PreProcessFunc func(stream *VirtualStream, isPrint bool) error

// preprocess 请求预处理
func (g *GRPCServer) preprocess(stream *VirtualStream, isPrint bool) error {
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

// PostProcessFunc 请求后处理函数定义
type PostProcessFunc func(stream *VirtualStream, m interface{})

// postprocess 请求后处理
func (g *GRPCServer) postprocess(stream *VirtualStream, m interface{}) {
	response := m.(api.ResponseMessage)
	code := api.CalcCode(response)

	// 打印回复
	if code != http.StatusOK {
		log.Error(response.String(),
			zap.String("client-address", stream.ClientAddress),
			zap.String("user-agent", stream.UserAgent),
			zap.String("request-id", stream.RequestID),
			zap.String("method", stream.Method),
		)
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

	_ = g.statis.AddAPICall(stream.Method, int(response.GetCode().GetValue()), diff.Nanoseconds())
}

// enterRateLimit 限流
func (g *GRPCServer) enterRateLimit(ip string, method string) uint32 {
	if g.ratelimit == nil {
		return api.ExecuteSuccess
	}

	// ip ratelimit
	if ok := g.ratelimit.Allow(plugin.IPRatelimit, ip); !ok {
		log.Error("[grpc] ip ratelimit is not allow", zap.String("client-ip", ip),
			zap.String("method", method))
		return api.IPRateLimit
	}
	// api ratelimit
	if ok := g.ratelimit.Allow(plugin.APIRatelimit, method); !ok {
		log.Error("[grpc] api rate limit is not allow", zap.String("client-ip", ip),
			zap.String("method", method))
		return api.APIRateLimit
	}

	return api.ExecuteSuccess
}

// allowAccess 限制访问
func (g *GRPCServer) allowAccess(method string) bool {
	_, ok := g.openMethod[method]
	return ok
}
