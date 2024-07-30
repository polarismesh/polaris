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
	"runtime"
	"time"

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	api "github.com/polarismesh/polaris/common/api/v1"
	connhook "github.com/polarismesh/polaris/common/conn/hook"
	connlimit "github.com/polarismesh/polaris/common/conn/limit"
	commonlog "github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/metrics"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/secure"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
)

// InitServer BaseGrpcServer.Run 中回调函数的定义
type InitServer func(*grpc.Server) error

// BaseGrpcServer base utilities and functions for gRPC Connector
type BaseGrpcServer struct {
	listenIP        string
	listenPort      uint32
	connLimitConfig *connlimit.Config
	tlsInfo         *secure.TLSInfo
	start           bool
	restart         bool
	exitCh          chan struct{}

	protocol string

	bz authcommon.BzModule

	server     *grpc.Server
	statis     plugin.Statis
	ratelimit  plugin.Ratelimit
	OpenMethod map[string]bool

	cache   Cache
	convert MessageToCache

	log *commonlog.Scope
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

	if raw, _ := conf["tls"].(map[interface{}]interface{}); raw != nil {
		tlsConfig, err := secure.ParseTLSConfig(raw)
		if err != nil {
			return err
		}
		b.tlsInfo = &secure.TLSInfo{
			CertFile:      tlsConfig.CertFile,
			KeyFile:       tlsConfig.KeyFile,
			TrustedCAFile: tlsConfig.TrustedCAFile,
		}
	}

	if ratelimit := plugin.GetRatelimit(); ratelimit != nil {
		b.log.Infof("[API-Server] %s server open the ratelimit", b.protocol)
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
	b.log.Infof("[API-Server] start %s server", protocol)
	b.exitCh = make(chan struct{})
	b.start = true
	defer func() {
		close(b.exitCh)
		b.start = false
	}()

	address := fmt.Sprintf("%v:%v", b.listenIP, b.listenPort)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		b.log.Error("[API-Server][GRPC] %v", zap.Error(err))
		errCh <- err
		return
	}
	defer listener.Close()

	// 如果设置最大连接数
	if b.connLimitConfig != nil && b.connLimitConfig.OpenConnLimit {
		b.log.Infof("[API-Server][GRPC] grpc server use max connection limit: %d, grpc max limit: %d",
			b.connLimitConfig.MaxConnPerHost, b.connLimitConfig.MaxConnLimit)
		listener, err = connlimit.NewListener(listener, protocol, b.connLimitConfig)
		if err != nil {
			b.log.Error("[API-Server][GRPC] conn limit init", zap.Error(err))
			errCh <- err
			return
		}
	}

	b.log.Infof("[API-Server][GRPC] open connection counter net.Listener")
	listener = connhook.NewHookListener(listener, &connCounterHook{
		bz: b.bz,
	})

	// 指定使用服务端证书创建一个 TLS credentials
	var creds credentials.TransportCredentials
	if !b.tlsInfo.IsEmpty() {
		creds, err = credentials.NewServerTLSFromFile(b.tlsInfo.CertFile, b.tlsInfo.KeyFile)
		if err != nil {
			b.log.Error("failed to create credentials: %v", zap.Error(err))
			errCh <- err
			return
		}
	}

	// 设置 grpc server options
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(b.unaryInterceptor),
		grpc.StreamInterceptor(b.streamInterceptor),
	}
	if creds != nil {
		// 指定使用 TLS credentials
		opts = append(opts, grpc.Creds(creds))
	}
	server := grpc.NewServer(opts...)

	if err = initServer(server); err != nil {
		errCh <- err
		return
	}
	b.server = server

	b.statis = plugin.GetStatis()

	if err := server.Serve(listener); err != nil {
		b.log.Errorf("[API-Server][GRPC] %v", err)
		errCh <- err
		return
	}

	b.log.Infof("[API-Server] %s server stop", protocol)
}

var notPrintableMethods = map[string]bool{
	"/v1.PolarisGRPC/Heartbeat": true,
}

func (b *BaseGrpcServer) unaryInterceptor(ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (rsp interface{}, err error) {
	stream := newVirtualStream(ctx,
		WithVirtualStreamBaseServer(b),
		WithVirtualStreamLogger(b.log),
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
			rsp = api.NewResponse(apimodel.Code_ClientAPINotOpen)
			return
		}

		// handler执行前，限流
		if code := b.EnterRatelimit(stream.ClientIP, stream.Method); code != uint32(api.ExecuteSuccess) {
			rsp = api.NewResponse(apimodel.Code(code))
			return
		}
		defer func() {
			if panicInfo := recover(); panicInfo != nil {
				var buf [4086]byte
				n := runtime.Stack(buf[:], false)
				b.log.Errorf("panic %+v", string(buf[:n]))
			}
		}()

		rsp, err = handler(ctx, req)
	}()

	b.postprocess(stream, rsp)

	return
}

func (b *BaseGrpcServer) recoverFunc(i interface{}, w http.ResponseWriter) {

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

	defer func() {
		if panicInfo := recover(); panicInfo != nil {
			var buf [4086]byte
			n := runtime.Stack(buf[:], false)
			b.log.Errorf("panic recovered: %v, STACK: %s", panicInfo, buf[0:n])
		}
	}()

	err = handler(srv, stream)
	if err != nil {
		fromError, ok := status.FromError(err)
		if ok && fromError.Code() == codes.Canceled {
			// 存在非EOF读错误或者写错误
			b.log.Info("[API-Server][GRPC] handler stream is canceled by client",
				zap.String("client-address", stream.ClientAddress),
				zap.String("user-agent", stream.UserAgent),
				utils.ZapRequestID(stream.RequestID),
				zap.String("method", stream.Method),
				zap.Error(err),
			)
		} else {
			// 存在非EOF读错误或者写错误
			b.log.Error("[API-Server][GRPC] handler stream",
				zap.String("client-address", stream.ClientAddress),
				zap.String("user-agent", stream.UserAgent),
				utils.ZapRequestID(stream.RequestID),
				zap.String("method", stream.Method),
				zap.Error(err),
			)
		}

		b.statis.ReportCallMetrics(metrics.CallMetric{
			Type:     metrics.ServerCallMetric,
			API:      stream.Method,
			Protocol: "gRPC",
			Code:     int(stream.Code),
			Duration: 0,
		})
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
		b.log.Info("[API-Server][GRPC] receive request",
			zap.String("client-address", stream.ClientAddress),
			zap.String("user-agent", stream.UserAgent),
			utils.ZapRequestID(stream.RequestID),
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
			b.log.Error("[API-Server][GRPC] send response",
				zap.String("client-address", stream.ClientAddress),
				zap.String("user-agent", stream.UserAgent),
				utils.ZapRequestID(stream.RequestID),
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
			b.log.Error("[API-Server][GRPC] send response",
				zap.String("client-address", stream.ClientAddress),
				zap.String("user-agent", stream.UserAgent),
				utils.ZapRequestID(stream.RequestID),
				zap.String("method", stream.Method),
				zap.String("response", response.String()),
			)
		}
	}

	// 接口调用统计
	diff := time.Since(stream.StartTime)

	// 打印耗时超过1s的请求
	if diff > time.Second {
		b.log.Info("[API-Server][GRPC] handling time > 1s",
			zap.String("client-address", stream.ClientAddress),
			zap.String("user-agent", stream.UserAgent),
			utils.ZapRequestID(stream.RequestID),
			zap.String("method", stream.Method),
			zap.Duration("handling-time", diff),
		)
	}

	b.statis.ReportCallMetrics(metrics.CallMetric{
		Type:     metrics.ServerCallMetric,
		API:      stream.Method,
		Protocol: "gRPC",
		Code:     int(stream.Code),
		Duration: diff,
	})
}

// Restart restart gRPC server
func (b *BaseGrpcServer) Restart(
	initialize func() error, run func(), protocol string, option map[string]interface{}) error {
	b.log.Infof("[API-Server][GRPC] restart %s server with new config: %+v", protocol, option)

	b.restart = true
	b.Stop(protocol)
	if b.start {
		<-b.exitCh
	}

	b.log.Infof("[API-Server][GRPC] old %s server has stopped, begin restarting it", protocol)
	if err := initialize(); err != nil {
		b.log.Errorf("restart %s server err: %s", protocol, err.Error())
		return err
	}

	b.log.Infof("[API-Server][GRPC] init %s server successfully, restart it", protocol)
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
		b.log.Error("[API-Server][GRPC] ip ratelimit is not allow", zap.String("client-ip", ip),
			zap.String("method", method))
		return api.IPRateLimit
	}
	// apiRatelimit
	if ok := b.ratelimit.Allow(plugin.APIRatelimit, method); !ok {
		b.log.Error("[API-Server][GRPC] api rate limit is not allow", zap.String("client-ip", ip),
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

type connCounterHook struct {
	bz authcommon.BzModule
}

func (h *connCounterHook) OnAccept(conn net.Conn) {
	if h.bz == authcommon.DiscoverModule {
		metrics.AddDiscoveryClientConn()
	}
	if h.bz == authcommon.ConfigModule {
		metrics.AddConfigurationClientConn()
	}
	metrics.AddSDKClientConn()
}

func (h *connCounterHook) OnRelease(conn net.Conn) {
	if h.bz == authcommon.DiscoverModule {
		metrics.RemoveDiscoveryClientConn()
	}
	if h.bz == authcommon.ConfigModule {
		metrics.RemoveConfigurationClientConn()
	}
	metrics.RemoveSDKClientConn()
}

func (h *connCounterHook) OnClose() {
	metrics.ResetDiscoveryClientConn()
	metrics.ResetConfigurationClientConn()
	metrics.ResetSDKClientConn()
}
