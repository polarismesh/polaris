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

package v2

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/polarismesh/polaris/apiserver"
	"github.com/polarismesh/polaris/apiserver/nacosserver/core"
	v1 "github.com/polarismesh/polaris/apiserver/nacosserver/v1"
	"github.com/polarismesh/polaris/apiserver/nacosserver/v2/config"
	"github.com/polarismesh/polaris/apiserver/nacosserver/v2/discover"
	nacospb "github.com/polarismesh/polaris/apiserver/nacosserver/v2/pb"
	"github.com/polarismesh/polaris/apiserver/nacosserver/v2/remote"
	api "github.com/polarismesh/polaris/common/api/v1"
	connhook "github.com/polarismesh/polaris/common/conn/hook"
	connlimit "github.com/polarismesh/polaris/common/conn/limit"
	"github.com/polarismesh/polaris/common/metrics"
	"github.com/polarismesh/polaris/common/secure"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
)

func NewNacosV2Server(v1Svr *v1.NacosV1Server, store *core.NacosDataStorage, options ...option) (*NacosV2Server, error) {
	connMgr := remote.NewConnectionManager()
	svr := &NacosV2Server{
		connectionManager: connMgr,
		discoverOpt: &discover.ServerOption{
			Store:             store,
			ConnectionManager: connMgr,
		},
		discoverSvr: &discover.DiscoverServer{},
		configOpt: &config.ServerOption{
			Store:             store,
			ConnectionManager: connMgr,
		},
		configSvr: &config.ConfigServer{},
	}

	for i := range options {
		options[i](svr)
	}

	if err := svr.discoverSvr.Initialize(svr.discoverOpt); err != nil {
		return nil, err
	}
	if err := svr.configSvr.Initialize(svr.configOpt); err != nil {
		return nil, err
	}

	return svr, nil
}

// NacosV2Server gRPC API服务器
type NacosV2Server struct {
	listenIP        string
	listenPort      uint32
	connLimitConfig *connlimit.Config
	tlsInfo         *secure.TLSInfo
	option          map[string]interface{}
	openAPI         map[string]apiserver.APIConfig
	start           bool
	protocol        string

	server     *grpc.Server
	ratelimit  plugin.Ratelimit
	OpenMethod map[string]bool
	rateLimit  plugin.Ratelimit
	whitelist  plugin.Whitelist

	discoverOpt *discover.ServerOption
	discoverSvr *discover.DiscoverServer

	configOpt *config.ServerOption
	configSvr *config.ConfigServer

	v1Svr             *v1.NacosV1Server
	connectionManager *remote.ConnectionManager
	handleRegistry    map[string]*remote.RequestHandlerWarrper
}

// GetProtocol 获取Server的协议
func (h *NacosV2Server) GetProtocol() string {
	return "nacos-grpc"
}

// Initialize 初始化HTTP API服务器
func (h *NacosV2Server) Initialize(ctx context.Context, option map[string]interface{}, port uint32,
	apiConf map[string]apiserver.APIConfig) error {
	h.option = option
	h.openAPI = apiConf
	h.listenIP = option["listenIP"].(string)
	h.listenPort = port
	if raw, _ := option["connLimit"].(map[interface{}]interface{}); raw != nil {
		connConfig, err := connlimit.ParseConnLimitConfig(raw)
		if err != nil {
			return err
		}
		h.connLimitConfig = connConfig
	}
	if raw, _ := option["tls"].(map[interface{}]interface{}); raw != nil {
		tlsConfig, err := secure.ParseTLSConfig(raw)
		if err != nil {
			return err
		}
		h.tlsInfo = &secure.TLSInfo{
			CertFile:      tlsConfig.CertFile,
			KeyFile:       tlsConfig.KeyFile,
			TrustedCAFile: tlsConfig.TrustedCAFile,
		}
	}

	if ratelimit := plugin.GetRatelimit(); ratelimit != nil {
		nacoslog.Infof("[API-Server] %s server open the ratelimit", h.protocol)
		h.ratelimit = ratelimit
	}
	h.initHandlers()
	return nil
}

// initHandlers .
func (h *NacosV2Server) initHandlers() {
	h.handleRegistry = map[string]*remote.RequestHandlerWarrper{
		nacospb.TypeServerCheckRequest: {
			Handler: h.handleServerCheckRequest,
			PayloadBuilder: func() nacospb.CustomerPayload {
				return nacospb.NewServerCheckRequest()
			},
		},
		nacospb.TypeHealthCheckRequest: {
			Handler: h.handleHealthCheckRequest,
			PayloadBuilder: func() nacospb.CustomerPayload {
				return nacospb.NewHealthCheckRequest()
			},
		},
	}

	for k, v := range h.discoverSvr.ListGRPCHandlers() {
		h.handleRegistry[k] = v
	}
	for k, v := range h.configSvr.ListGRPCHandlers() {
		h.handleRegistry[k] = v
	}

	for k := range h.handleRegistry {
		nacoslog.Info("[API-Server][NACOS-V2] handle registry", zap.String("type", k))
	}
}

// Run 启动GRPC API服务器
func (h *NacosV2Server) Run(errCh chan error) {
	h.start = true
	defer func() {
		h.start = false
	}()

	address := fmt.Sprintf("%v:%v", h.listenIP, h.listenPort)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		nacoslog.Error("[API-Server][NACOS-V2] %v", zap.Error(err))
		errCh <- err
		return
	}
	defer listener.Close()

	listener = connhook.NewHookListener(listener, h.connectionManager)

	// 如果设置最大连接数
	if h.connLimitConfig != nil && h.connLimitConfig.OpenConnLimit {
		nacoslog.Infof("[API-Server][NACOS-V2] grpc server use max connection limit: %d, grpc max limit: %d",
			h.connLimitConfig.MaxConnPerHost, h.connLimitConfig.MaxConnLimit)
		listener, err = connlimit.NewListener(listener, h.protocol, h.connLimitConfig)
		if err != nil {
			nacoslog.Error("[API-Server][NACOS-V2] conn limit init", zap.Error(err))
			errCh <- err
			return
		}
	}

	nacoslog.Infof("[API-Server][NACOS-V2] open connection counter net.Listener")

	// 指定使用服务端证书创建一个 TLS credentials
	var creds credentials.TransportCredentials
	if !h.tlsInfo.IsEmpty() {
		creds, err = credentials.NewServerTLSFromFile(h.tlsInfo.CertFile, h.tlsInfo.KeyFile)
		if err != nil {
			nacoslog.Error("failed to create credentials: %v", zap.Error(err))
			errCh <- err
			return
		}
	}

	// 设置 grpc server options
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(h.unaryInterceptor),
		grpc.StreamInterceptor(h.streamInterceptor),
		grpc.StatsHandler(h.connectionManager),
	}
	if creds != nil {
		// 指定使用 TLS credentials
		opts = append(opts, grpc.Creds(creds))
	}
	h.server = grpc.NewServer(opts...)
	nacospb.RegisterRequestServer(h.server, h)
	nacospb.RegisterBiRequestStreamServer(h.server, h)

	if err := h.server.Serve(listener); err != nil {
		nacoslog.Errorf("[API-Server][NACOS-V2] %v", err)
		errCh <- err
		return
	}

	nacoslog.Infof("[API-Server][NACOS-V2] %s server stop", h.protocol)
}

var notPrintableMethods map[string]struct{}

func (b *NacosV2Server) unaryInterceptor(ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (rsp interface{}, err error) {
	stream := newVirtualStream(ctx,
		WithVirtualStreamBaseServer(b),
		WithVirtualStreamLogger(nacoslog),
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
			return
		}

		// handler执行前，限流
		if code := b.EnterRatelimit(stream.ClientIP, stream.Method); code != uint32(api.ExecuteSuccess) {
			return
		}
		rsp, err = handler(ctx, req)
	}()

	b.postprocess(stream, rsp)
	return
}

func (b *NacosV2Server) streamInterceptor(srv interface{}, ss grpc.ServerStream,
	info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
	stream := newVirtualStream(ss.Context(),
		WithVirtualStreamBaseServer(b),
		WithVirtualStreamServerStream(ss),
		WithVirtualStreamMethod(info.FullMethod),
		WithVirtualStreamPreProcessFunc(b.preprocess),
		WithVirtualStreamPostProcessFunc(b.postprocess),
	)

	err = handler(srv, stream)
	if err != nil {
		fromError, ok := status.FromError(err)
		if ok && fromError.Code() == codes.Canceled {
			// 存在非EOF读错误或者写错误
			nacoslog.Info("[API-Server][NACOS-V2] handler stream is canceled by client",
				zap.String("client-address", stream.ClientAddress),
				zap.String("user-agent", stream.UserAgent),
				utils.ZapRequestID(stream.RequestID),
				zap.String("method", stream.Method),
				zap.Error(err),
			)
		} else {
			// 存在非EOF读错误或者写错误
			nacoslog.Error("[API-Server][NACOS-V2] handler stream",
				zap.String("client-address", stream.ClientAddress),
				zap.String("user-agent", stream.UserAgent),
				utils.ZapRequestID(stream.RequestID),
				zap.String("method", stream.Method),
				zap.Error(err),
			)
		}

		plugin.GetStatis().ReportCallMetrics(metrics.CallMetric{
			Type:     metrics.ServerCallMetric,
			API:      stream.Method,
			Protocol: "NACOS-V2",
			Code:     int(stream.Code),
			Duration: 0,
		})
	}
	return
}

// PreProcessFunc preprocess function define
type PreProcessFunc func(stream *VirtualStream, isPrint bool) error

func (b *NacosV2Server) preprocess(stream *VirtualStream, _ bool) error {
	// 设置开始时间
	stream.StartTime = time.Now()
	return nil
}

// PostProcessFunc postprocess function define
type PostProcessFunc func(stream *VirtualStream, m interface{})

func (b *NacosV2Server) postprocess(stream *VirtualStream, m interface{}) {
	// 接口调用统计
	diff := time.Since(stream.StartTime)

	plugin.GetStatis().ReportCallMetrics(metrics.CallMetric{
		Type:     metrics.ServerCallMetric,
		API:      stream.Method,
		Protocol: "NACOS-V2",
		Code:     int(stream.Code),
		Duration: diff,
	})
}

// GetPort get the connector listen port value
func (h *NacosV2Server) GetPort() uint32 {
	return h.listenPort
}

// Stop stopping the gRPC server
func (h *NacosV2Server) Stop() {
	connlimit.RemoveLimitListener(h.GetProtocol())
	if h.server != nil {
		h.server.Stop()
	}
}

// EnterRatelimit api ratelimit
func (h *NacosV2Server) EnterRatelimit(ip string, method string) uint32 {
	if h.ratelimit == nil {
		return api.ExecuteSuccess
	}

	// ipRatelimit
	if ok := h.ratelimit.Allow(plugin.IPRatelimit, ip); !ok {
		nacoslog.Error("[API-Server][NACOS-V2] ip ratelimit is not allow", zap.String("client-ip", ip),
			zap.String("method", method))
		return api.IPRateLimit
	}
	// apiRatelimit
	if ok := h.ratelimit.Allow(plugin.APIRatelimit, method); !ok {
		nacoslog.Error("[API-Server][NACOS-V2] api rate limit is not allow", zap.String("client-ip", ip),
			zap.String("method", method))
		return api.APIRateLimit
	}

	return api.ExecuteSuccess
}

// AllowAccess api allow access
func (h *NacosV2Server) AllowAccess(method string) bool {
	if len(h.OpenMethod) == 0 {
		return true
	}
	_, ok := h.OpenMethod[method]
	return ok
}

// ConvertContext 将GRPC上下文转换成内部上下文
func (h *NacosV2Server) ConvertContext(ctx context.Context) context.Context {
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
	} else {
		meta = metadata.MD{}
	}

	var (
		clientIP = ""
		address  = ""
		connID   = ""
		connMeta remote.ConnectionMeta
	)
	if pr, ok := peer.FromContext(ctx); ok && pr.Addr != nil {
		address = pr.Addr.String()
		addrSlice := strings.Split(address, ":")
		if len(addrSlice) == 2 {
			clientIP = addrSlice[0]
		}

		client, find := h.connectionManager.GetClientByAddr(pr.Addr.String())
		if find {
			connID = client.ID
			connMeta = client.ConnMeta
		}
	}

	ctx = context.Background()
	ctx = context.WithValue(ctx, utils.ContextGrpcHeader, meta)
	ctx = context.WithValue(ctx, utils.StringContext("request-id"), requestID)
	ctx = context.WithValue(ctx, utils.ContextClientAddress, address)
	ctx = context.WithValue(ctx, utils.StringContext("user-agent"), userAgent)
	ctx = context.WithValue(ctx, remote.ClientIPKey{}, clientIP)
	ctx = context.WithValue(ctx, remote.ConnIDKey{}, connID)
	ctx = context.WithValue(ctx, remote.ConnectionInfoKey{}, connMeta)
	return ctx
}
