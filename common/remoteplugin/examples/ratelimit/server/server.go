package main

import (
	"context"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/polarismesh/polaris/common/api/plugin"
	"github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/remoteplugin"
)

type filter struct{}

func (s *filter) Call(ctx context.Context, request *plugin.Request) (*plugin.Response, error) {
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
	remoteplugin.Serve(context.Background(), &filter{})
}
