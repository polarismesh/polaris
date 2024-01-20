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

package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"

	testsuit "github.com/polarismesh/polaris/test/suit"
)

func TestClientCheck(t *testing.T) {
	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(func(cfg *testsuit.TestConfig) {
		cfg.HealthChecks.ClientCheckTtl = time.Second
		cfg.HealthChecks.ClientCheckInterval = 5 * time.Second
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		discoverSuit.cleanReportClient()
		discoverSuit.Destroy()
	})

	clientId1 := "111"
	clientId2 := "222"

	discoverSuit.addInstance(t, &apiservice.Instance{
		Service:   wrapperspb.String("polaris.checker"),
		Namespace: wrapperspb.String("Polaris"),
		Host:      wrapperspb.String("127.0.0.1"),
		Port:      wrapperspb.UInt32(8091),
		Protocol:  wrapperspb.String("grpc"),
		Metadata:  map[string]string{"polaris_service": "polaris.checker"},
	})
	time.Sleep(20 * time.Second)
	clientIds := map[string]bool{clientId1: true, clientId2: true}
	for i := 0; i < 10; i++ {
		for clientId := range clientIds {
			fmt.Printf("%d report client for %s, round 1\n", i, clientId)
			discoverSuit.DiscoverServer().ReportClient(context.Background(),
				&apiservice.Client{
					Id: &wrapperspb.StringValue{Value: clientId}, Host: &wrapperspb.StringValue{Value: "127.0.0.1"}})
		}
		time.Sleep(1 * time.Second)
	}

	client1 := discoverSuit.DiscoverServer().Cache().Client().GetClient(clientId1)
	assert.NotNil(t, client1)
	client2 := discoverSuit.DiscoverServer().Cache().Client().GetClient(clientId2)
	assert.NotNil(t, client2)

	delete(clientIds, clientId2)
	for i := 0; i < 50; i++ {
		for clientId := range clientIds {
			fmt.Printf("%d report client for %s, round 2\n", i, clientId)
			discoverSuit.DiscoverServer().ReportClient(context.Background(),
				&apiservice.Client{Id: &wrapperspb.StringValue{Value: clientId},
					Host: &wrapperspb.StringValue{Value: "127.0.0.1"}, Type: apiservice.Client_SDK})
		}
		time.Sleep(1 * time.Second)
	}
	client1 = discoverSuit.DiscoverServer().Cache().Client().GetClient(clientId1)
	assert.NotNil(t, client1)
	client2 = discoverSuit.DiscoverServer().Cache().Client().GetClient(clientId2)
	assert.Nil(t, client2)
}
