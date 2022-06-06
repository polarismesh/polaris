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
	"net"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/connlimit"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/plugin"
)

// InitServer BaseGrpcServer.Run 中回调函数的定义
type InitServer func(*grpc.Server) error

// BaseGrpcServer base utilities and functions for gRPC Connector
type BaseGrpcServer struct {
	listenIP        string
	listenPort      uint32
	connLimitConfig *connlimit.Config
	start           bool
	restart         bool
	exitCh          chan struct{}

	protocol string

	server     *grpc.Server
	statis     plugin.Statis
	ratelimit  plugin.Ratelimit
	OpenMethod map[string]bool

	cache   Cache
	convert MessageToCache
}

// GetPort get the connector listen port value
func (b *BaseGrpcServer) GetPort() uint32 {
	return b.listenPort
}

// Initialize init the gRPC server
func (b *BaseGrpcServer) Initialize(ctx context.Context, conf map[string]interface{}, initOptions ...InitOption) error {

	for i := range initOptions {
		initOptions[i](b)
	}

	b.listenIP = conf["listenIP"].(string)
	b.listenPort = uint32(conf["listenPort"].(int))

	if raw, _ := conf["connLimit"].(map[interface{}]interface{}); raw != nil {
		connConfig, err := connlimit.ParseConnLimitConfig(raw)
		if err != nil {
			return err
		}
		b.connLimitConfig = connConfig
	}
	if ratelimit := plugin.GetRatelimit(); ratelimit != nil {
		log.Infof("[API-Server] %s server open the ratelimit", b.protocol)
		b.ratelimit = ratelimit
	}

	return nil
}

// Stop stopping the gRPC server
func (b *BaseGrpcServer) Stop(protocol string) {
	connlimit.RemoveLimitListener(protocol)
	if b.server != nil {
		b.server.Stop()
	}
}

// Run server main loop
func (b *BaseGrpcServer) Run(errCh chan error, protocol string, initServer InitServer) {
	log.Infof("[API-Server] start %s server", protocol)
	b.exitCh = make(chan struct{})
	b.start = true
	defer func() {
		close(b.exitCh)
		b.start = false
	}()

	address := fmt.Sprintf("%v:%v", b.listenIP, b.listenPort)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Error("[API-Server][GRPC] %v", zap.Error(err))
		errCh <- err
		return
	}
	defer listener.Close()

	// 如果设置最大连接数
	if b.connLimitConfig != nil && b.connLimitConfig.OpenConnLimit {
		log.Infof("[API-Server][GRPC] grpc server use max connection limit: %d, grpc max limit: %d",
			b.connLimitConfig.MaxConnPerHost, b.connLimitConfig.MaxConnLimit)
		listener, err = connlimit.NewListener(listener, protocol, b.connLimitConfig)
		if err != nil {
			log.Error("[API-Server][GRPC] conn limit init", zap.Error(err))
			errCh <- err
			return
		}

	}

	server := grpc.NewServer(
		grpc.UnaryInterceptor(b.unaryInterceptor),
		grpc.StreamInterceptor(b.streamInterceptor),
	)

	if err = initServer(server); err != nil {
		errCh <- err
		return
	}
	b.server = server

	b.statis = plugin.GetStatis()

	if err := server.Serve(listener); err != nil {
		log.Errorf("[API-Server][GRPC] %v", err)
		errCh <- err
		return
	}

	log.Infof("[API-Server] %s server stop", protocol)
}

var notPrintableMethods = map[string]bool{
	"/v1.PolarisGRPC/Heartbeat": true,
}

func (b *BaseGrpcServer) unaryInterceptor(ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (rsp interface{}, err error) {

	stream := newVirtualStream(ctx,
		WithVirtualStreamBaseServer(b),
		WithVirtualStreamMethod(info.FullMethod),
		WithVirtualStreamPreProcessFunc(b.preprocess),
		WithVirtualStreamPostProcessFunc(b.postprocess),
	)

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

func (b *BaseGrpcServer) streamInterceptor(srv interface{}, ss grpc.ServerStream,
	info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {

	stream := newVirtualStream(ss.Context(),
		WithVirtualStreamBaseServer(b),
		WithVirtualStreamServerStream(ss),
		WithVirtualStreamMethod(info.FullMethod),
		WithVirtualStreamPreProcessFunc(b.preprocess),
		WithVirtualStreamPostProcessFunc(b.postprocess),
	)

	err = handler(srv, stream)
	if err != nil { // 存在非EOF读错误或者写错误
		log.Error("[API-Server][GRPC] handler stream",
			zap.String("client-address", stream.ClientAddress),
			zap.String("user-agent", stream.UserAgent),
			zap.String("request-id", stream.RequestID),
			zap.String("method", stream.Method),
			zap.Error(err),
		)

		_ = b.statis.AddAPICall(stream.Method, "gRPC", stream.Code, 0)
	}

	return
}

// PreProcessFunc preprocess function define
type PreProcessFunc func(stream *VirtualStream, isPrint bool) error

func (b *BaseGrpcServer) preprocess(stream *VirtualStream, isPrint bool) error {
	// 设置开始时间
	stream.StartTime = time.Now()

	if isPrint {
		// 打印请求
		log.Info("[API-Server][GRPC] receive request",
			zap.String("client-address", stream.ClientAddress),
			zap.String("user-agent", stream.UserAgent),
			zap.String("request-id", stream.RequestID),
			zap.String("method", stream.Method),
		)
	}

	return nil
}

// PostProcessFunc postprocess function define
type PostProcessFunc func(stream *VirtualStream, m interface{})

func (b *BaseGrpcServer) postprocess(stream *VirtualStream, m interface{}) {
	var code int
	var polarisCode uint32
	if response, ok := m.(api.ResponseMessage); ok {
		polarisCode = response.GetCode().GetValue()
		code = api.CalcCode(response)

		// 打印回复
		if code != http.StatusOK {
			log.Error("[API-Server][GRPC] send response",
				zap.String("client-address", stream.ClientAddress),
				zap.String("user-agent", stream.UserAgent),
				zap.String("request-id", stream.RequestID),
				zap.String("method", stream.Method),
				zap.String("response", response.String()),
			)
		}
		if polarisCode > 0 {
			code = int(polarisCode)
		}
	} else {
		code = stream.Code
		// 打印回复
		if code != int(codes.OK) {
			log.Error("[API-Server][GRPC] send response",
				zap.String("client-address", stream.ClientAddress),
				zap.String("user-agent", stream.UserAgent),
				zap.String("request-id", stream.RequestID),
				zap.String("method", stream.Method),
				zap.String("response", response.String()),
			)
		}
	}

	// 接口调用统计
	now := time.Now()
	diff := now.Sub(stream.StartTime)

	// 打印耗时超过1s的请求
	if diff > time.Second {
		log.Info("[API-Server][GRPC] handling time > 1s",
			zap.String("client-address", stream.ClientAddress),
			zap.String("user-agent", stream.UserAgent),
			zap.String("request-id", stream.RequestID),
			zap.String("method", stream.Method),
			zap.Duration("handling-time", diff),
		)
	}
	_ = b.statis.AddAPICall(stream.Method, "gRPC", code, diff.Nanoseconds())
}

// Restart restart gRPC server
func (b *BaseGrpcServer) Restart(
	initialize func() error, run func(), protocol string, option map[string]interface{}) error {
	log.Infof("[API-Server][GRPC] restart %s server with new config: %+v", protocol, option)

	b.restart = true
	b.Stop(protocol)
	if b.start {
		<-b.exitCh
	}

	log.Infof("[API-Server][GRPC] old %s server has stopped, begin restarting it", protocol)
	if err := initialize(); err != nil {
		log.Errorf("restart %s server err: %s", protocol, err.Error())
		return err
	}

	log.Infof("[API-Server][GRPC] init %s server successfully, restart it", protocol)
	b.restart = false
	go run()

	return nil
}

// EnterRatelimit api ratelimit
func (b *BaseGrpcServer) EnterRatelimit(ip string, method string) uint32 {
	if b.ratelimit == nil {
		return api.ExecuteSuccess
	}

	// ipRatelimit
	if ok := b.ratelimit.Allow(plugin.IPRatelimit, ip); !ok {
		log.Error("[API-Server][GRPC] ip ratelimit is not allow", zap.String("client-ip", ip),
			zap.String("method", method))
		return api.IPRateLimit
	}
	// apiRatelimit
	if ok := b.ratelimit.Allow(plugin.APIRatelimit, method); !ok {
		log.Error("[API-Server][GRPC] api rate limit is not allow", zap.String("client-ip", ip),
			zap.String("method", method))
		return api.APIRateLimit
	}

	return api.ExecuteSuccess
}

// AllowAccess api allow access
func (b *BaseGrpcServer) AllowAccess(method string) bool {
	if len(b.OpenMethod) == 0 {
		return true
	}
	_, ok := b.OpenMethod[method]
	return ok
}

// ConvertContext 将GRPC上下文转换成内部上下文
func ConvertContext(ctx context.Context) context.Context {
	var (
		requestID = ""
		userAgent = ""
	)
	meta, exist := metadata.FromIncomingContext(ctx)
	if exist {
		ids := meta["request-id"]
		if len(ids) > 0 {
			requestID = ids[0]
		}
		agents := meta["user-agent"]
		if len(agents) > 0 {
			userAgent = agents[0]
		}
	}

	var (
		clientIP = ""
		address  = ""
	)
	if pr, ok := peer.FromContext(ctx); ok && pr.Addr != nil {
		address = pr.Addr.String()
		addrSlice := strings.Split(address, ":")
		if len(addrSlice) == 2 {
			clientIP = addrSlice[0]
		}
	}

	ctx = context.Background()
	ctx = context.WithValue(ctx, utils.StringContext("request-id"), requestID)
	ctx = context.WithValue(ctx, utils.StringContext("client-ip"), clientIP)
	ctx = context.WithValue(ctx, utils.ContextClientAddress, address)
	ctx = context.WithValue(ctx, utils.StringContext("user-agent"), userAgent)
	return ctx
}
