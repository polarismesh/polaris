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
		client3 *remoteplugin.Client
	)
	if client1, err = remoteplugin.Register(
		&remoteplugin.Config{Name: "rate-limit-server-v1", Mode: remoteplugin.PluginRumModelLocal},
	); err != nil {
		log.Fatalf("server-v1 register failed: %+v", err)
		return
	}

	if client2, err = remoteplugin.Register(
		&remoteplugin.Config{Name: "rate-limit-server-v2", Mode: remoteplugin.PluginRumModelLocal},
	); err != nil {
		log.Fatalf("server-v2 register failed: %+v", err)
	}

	if client3, err = remoteplugin.Register(&remoteplugin.Config{
		Name: "rate-limit-server-v3", Mode: remoteplugin.PluginRumModelRemote,
		Remote: remoteplugin.RemoteConfig{Address: "0.0.0.0:8972"},
	}); err != nil {
		log.Fatalf("server-v3 register failed: %+v", err)
	}

	for i := 0; i < 10000; i++ {
		clientInvoke(client1, "1")
		clientInvoke(client2, "2")
		clientInvoke(client3, "3")
	}
}

func clientInvoke(client *remoteplugin.Client, name string) {
	req := &pluginPB.RateLimitRequest{}
	ruleAny, _ := anypb.New(req)

	data, err := proto.Marshal(ruleAny)
	if err != nil {
		log.Fatal("unable to marshal request")
	}

	response, err := client.Call(context.Background(), &pluginPB.Request{
		Payload: &anypb.Any{
			TypeUrl: ruleAny.GetTypeUrl(),
			Value:   data,
		}},
	)

	if err != nil {
		log.Errorf("client-%s fail to invoke: %+v", name, err)
		return
	}

	fmt.Printf("response body from client-%s: %s\n", name, response.String())
}
