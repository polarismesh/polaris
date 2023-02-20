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

package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestInstanceCheck(t *testing.T) {
	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	instanceId1 := "inst_111"
	instanceId2 := "inst_222"

	discoverSuit.addInstance(t, &apiservice.Instance{
		Service:   wrapperspb.String("polaris.checker"),
		Namespace: wrapperspb.String("Polaris"),
		Host:      wrapperspb.String("127.0.0.1"),
		Port:      wrapperspb.UInt32(8091),
		Protocol:  wrapperspb.String("grpc"),
		Metadata:  map[string]string{"polaris_service": "polaris.checker"},
	})
	instanceIds := map[string]bool{instanceId1: true, instanceId2: true}
	for id := range instanceIds {
		resp := discoverSuit.server.RegisterInstance(context.Background(), &apiservice.Instance{Id: wrapperspb.String(id),
			Service: wrapperspb.String("testSvc"), Namespace: wrapperspb.String("default"),
			Host: wrapperspb.String("127.0.0.1"), Port: wrapperspb.UInt32(8888), Weight: wrapperspb.UInt32(100),
			HealthCheck: &apiservice.HealthCheck{Type: apiservice.HealthCheck_HEARTBEAT, Heartbeat: &apiservice.HeartbeatHealthCheck{
				Ttl: wrapperspb.UInt32(2),
			}},
		})
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), resp.GetCode().GetValue())
	}
	time.Sleep(20 * time.Second)
	for i := 0; i < 50; i++ {
		for instanceId := range instanceIds {
			fmt.Printf("%d report instance for %s, round 1\n", i, instanceId)
			discoverSuit.healthCheckServer.Report(
				context.Background(), &apiservice.Instance{Id: &wrapperspb.StringValue{Value: instanceId}})
		}
		time.Sleep(1 * time.Second)
	}

	instance1 := discoverSuit.server.Cache().Instance().GetInstance(instanceId1)
	assert.NotNil(t, instance1)
	assert.Equal(t, true, instance1.Proto.GetHealthy().GetValue())
	instance2 := discoverSuit.server.Cache().Instance().GetInstance(instanceId2)
	assert.NotNil(t, instance2)
	assert.Equal(t, true, instance2.Proto.GetHealthy().GetValue())

	delete(instanceIds, instanceId2)
	for i := 0; i < 50; i++ {
		for instanceId := range instanceIds {
			fmt.Printf("%d report instance for %s, round 2\n", i, instanceId)
			discoverSuit.healthCheckServer.Report(
				context.Background(), &apiservice.Instance{Id: &wrapperspb.StringValue{Value: instanceId}})
		}
		time.Sleep(1 * time.Second)
	}
	instance1 = discoverSuit.server.Cache().Instance().GetInstance(instanceId1)
	assert.NotNil(t, instance1)
	assert.Equal(t, true, instance1.Proto.GetHealthy().GetValue())
	instance2 = discoverSuit.server.Cache().Instance().GetInstance(instanceId2)
	assert.NotNil(t, instance2)
	assert.Equal(t, false, instance2.Proto.GetHealthy().GetValue())
}
