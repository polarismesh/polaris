package main

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/polarismesh/polaris/common/api/plugin"
	"github.com/polarismesh/polaris/common/log"
)

type rateLimiter struct {
}

func (r rateLimiter) Call(ctx context.Context, request *plugin.Request) (*plugin.Response, error) {
	var rateLimitRequest plugin.RateLimitRequest
	err := anypb.UnmarshalTo(request.GetPayload(), &rateLimitRequest, proto.UnmarshalOptions{})
	if err != nil {
		log.Fatalf("fail to unmarshal rate limit request: %+v", err)
	}

	reply := &plugin.RateLimitResponse{Allow: true}
	replyAny, _ := anypb.New(reply)
	data, err := proto.Marshal(reply)
	if err != nil {
		log.Fatal("fail to marshal response data")
	}
	response := &plugin.Response{Reply: &anypb.Any{
		TypeUrl: replyAny.GetTypeUrl(),
		Value:   data,
	}}
	return response, nil
}

func main() {
	// 监听本地的8972端口
	lis, err := net.Listen("tcp", ":8972")
	if err != nil {
		fmt.Printf("failed to listen: %v", err)
		return
	}
	s := grpc.NewServer()                          // 创建gRPC服务器
	plugin.RegisterPluginServer(s, &rateLimiter{}) // 在gRPC服务端注册服务
	// 启动服务
	err = s.Serve(lis)
	if err != nil {
		fmt.Printf("failed to serve: %v", err)
		return
	}
}
