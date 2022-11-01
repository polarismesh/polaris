package main

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	pluginPB "github.com/polarismesh/polaris/common/api/plugin"
	"github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/remoteplugin"
)

func main() {
	client, err := remoteplugin.Register("rate-limit-server", nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	req := &pluginPB.RateLimitRequest{}
	ruleAny, _ := anypb.New(req)
	data, err := proto.Marshal(ruleAny)
	if err != nil {
		log.Fatal("unable to marshal request")
	}
	response, err := client.Call(
		context.Background(),
		&pluginPB.Request{Payload: &anypb.Any{
			TypeUrl: ruleAny.GetTypeUrl(),
			Value:   data,
		}},
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("response body: %s\n", response.String())
}
