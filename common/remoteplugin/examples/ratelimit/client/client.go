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
	var (
		err     error
		client1 *remoteplugin.Client
		client2 *remoteplugin.Client
	)
	client1, err = remoteplugin.Register(&remoteplugin.Config{Name: "rate-limit-server-v1", Mode: remoteplugin.PluginRumModelLocal})
	if err != nil {
		fmt.Println(err)
		return
	}

	client2, err = remoteplugin.Register(&remoteplugin.Config{Name: "rate-limit-server-v2", Mode: remoteplugin.PluginRumModelLocal})
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

	response, err := client1.Call(
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

	fmt.Printf("response body from plugin-server-1: %s\n", response.String())

	response, err = client2.Call(
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
	fmt.Printf("response body from plugin-server-2: %s\n", response.String())
}
