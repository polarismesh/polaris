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

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/polarismesh/polaris/cache"
)

func TestInstanceCheck(t *testing.T) {
	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
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
		resp := discoverSuit.DiscoverServer().RegisterInstance(context.Background(), &apiservice.Instance{Id: wrapperspb.String(id),
			Service: wrapperspb.String("testSvc"), Namespace: wrapperspb.String("default"),
			Host: wrapperspb.String("127.0.0.1"), Port: wrapperspb.UInt32(8888), Weight: wrapperspb.UInt32(100),
			HealthCheck: &apiservice.HealthCheck{Type: apiservice.HealthCheck_HEARTBEAT, Heartbeat: &apiservice.HeartbeatHealthCheck{
				Ttl: wrapperspb.UInt32(2),
			}},
		})
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), resp.GetCode().GetValue())
	}
	//time.Sleep(20 * time.Second)
	for i := 0; i < 50; i++ {
		for instanceId := range instanceIds {
			discoverSuit.HealthCheckServer().Report(
				context.Background(), &apiservice.Instance{Id: &wrapperspb.StringValue{Value: instanceId}})
		}
		time.Sleep(1 * time.Second)
	}

	_ = discoverSuit.DiscoverServer().Cache().(*cache.CacheManager).TestUpdate()
	instance1 := discoverSuit.DiscoverServer().Cache().Instance().GetInstance(instanceId1)
	assert.NotNil(t, instance1)
	assert.Equal(t, true, instance1.Proto.GetHealthy().GetValue())
	instance2 := discoverSuit.DiscoverServer().Cache().Instance().GetInstance(instanceId2)
	assert.NotNil(t, instance2)
	assert.Equal(t, true, instance2.Proto.GetHealthy().GetValue())

	delete(instanceIds, instanceId2)
	for i := 0; i < 50; i++ {
		for instanceId := range instanceIds {
			fmt.Printf("%d report instance for %s, round 2\n", i, instanceId)
			discoverSuit.HealthCheckServer().Report(
				context.Background(), &apiservice.Instance{Id: &wrapperspb.StringValue{Value: instanceId}})
		}
		time.Sleep(1 * time.Second)
	}
	instance1 = discoverSuit.DiscoverServer().Cache().Instance().GetInstance(instanceId1)
	assert.NotNil(t, instance1)
	assert.Equal(t, true, instance1.Proto.GetHealthy().GetValue())
	instance2 = discoverSuit.DiscoverServer().Cache().Instance().GetInstance(instanceId2)
	assert.NotNil(t, instance2)
	assert.Equal(t, false, instance2.Proto.GetHealthy().GetValue())
}

func TestInstanceImmediatelyCheck(t *testing.T) {
	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	instanceId1 := "inst_333"
	instanceId2 := "inst_444"

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
		resp := discoverSuit.DiscoverServer().RegisterInstance(context.Background(), &apiservice.Instance{Id: wrapperspb.String(id),
			Service: wrapperspb.String("testSvc"), Namespace: wrapperspb.String("default"),
			Host: wrapperspb.String("127.0.0.1"), Port: wrapperspb.UInt32(8888), Weight: wrapperspb.UInt32(100),
			HealthCheck: &apiservice.HealthCheck{Type: apiservice.HealthCheck_HEARTBEAT, Heartbeat: &apiservice.HeartbeatHealthCheck{
				Ttl: wrapperspb.UInt32(2),
			}},
		})
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), resp.GetCode().GetValue())
	}
	for i := 0; i < 10; i++ {
		for instanceId := range instanceIds {
			fmt.Printf("%d report instance for %s, round 2\n", i, instanceId)
			hbResp := discoverSuit.HealthCheckServer().Report(
				context.Background(), &apiservice.Instance{Id: &wrapperspb.StringValue{Value: instanceId}})
			assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), hbResp.GetCode().GetValue())
		}
		time.Sleep(1 * time.Second)
	}
}

func TestInstanceCheckSuspended(t *testing.T) {
	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	instanceId1 := "inst_555"
	instanceId2 := "inst_666"

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
		resp := discoverSuit.DiscoverServer().RegisterInstance(context.Background(), &apiservice.Instance{Id: wrapperspb.String(id),
			Service: wrapperspb.String("testSvc"), Namespace: wrapperspb.String("default"),
			Host: wrapperspb.String("127.0.0.1"), Port: wrapperspb.UInt32(8888), Weight: wrapperspb.UInt32(100),
			HealthCheck: &apiservice.HealthCheck{Type: apiservice.HealthCheck_HEARTBEAT, Heartbeat: &apiservice.HeartbeatHealthCheck{
				Ttl: wrapperspb.UInt32(2),
			}},
		})
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), resp.GetCode().GetValue())
	}
	time.Sleep(5 * time.Second)
	discoverSuit.addInstance(t, &apiservice.Instance{
		Service:   wrapperspb.String("polaris.checker"),
		Namespace: wrapperspb.String("Polaris"),
		Host:      wrapperspb.String("127.0.0.1"),
		Port:      wrapperspb.UInt32(8091),
		Protocol:  wrapperspb.String("grpc"),
		Isolate:   wrapperspb.Bool(true),
		Metadata:  map[string]string{"polaris_service": "polaris.checker"},
	})
	time.Sleep(5 * time.Second)
	discoverSuit.addInstance(t, &apiservice.Instance{
		Service:   wrapperspb.String("polaris.checker"),
		Namespace: wrapperspb.String("Polaris"),
		Host:      wrapperspb.String("127.0.0.1"),
		Port:      wrapperspb.UInt32(8091),
		Protocol:  wrapperspb.String("grpc"),
		Isolate:   wrapperspb.Bool(false),
		Metadata:  map[string]string{"polaris_service": "polaris.checker"},
	})
	time.Sleep(5 * time.Second)
	checkers := discoverSuit.HealthCheckServer().Checkers()
	for _, checker := range checkers {
		suspendTimeSec := checker.SuspendTimeSec()
		assert.True(t, suspendTimeSec > 0)
	}

}

func TestSelfInstanceCheck(t *testing.T) {
	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	instanceId1 := "inst_self_1"
	instanceId2 := "inst_self_2"

	instanceIds := map[string]bool{instanceId1: true, instanceId2: true}
	hbInterval := 4

	for instanceId := range instanceIds {
		discoverSuit.addInstance(t, &apiservice.Instance{
			Id:                wrapperspb.String(instanceId),
			Service:           wrapperspb.String("polaris.checker"),
			Namespace:         wrapperspb.String("Polaris"),
			Host:              wrapperspb.String(instanceId),
			Port:              wrapperspb.UInt32(8091),
			Protocol:          wrapperspb.String("grpc"),
			EnableHealthCheck: wrapperspb.Bool(true),
			HealthCheck: &apiservice.HealthCheck{
				Type: apiservice.HealthCheck_HEARTBEAT,
				Heartbeat: &apiservice.HeartbeatHealthCheck{
					Ttl: &wrapperspb.UInt32Value{Value: uint32(hbInterval)},
				},
			},
			Metadata: map[string]string{"polaris_service": "polaris.checker"},
		})
	}

	for i := 0; i < 5; i++ {
		for instanceId := range instanceIds {
			fmt.Printf("%d report self instance for %s, round 1\n", i, instanceId)
			discoverSuit.HealthCheckServer().Report(
				context.Background(), &apiservice.Instance{
					Id: wrapperspb.String(instanceId),
				})
		}
		time.Sleep(2 * time.Second)
	}

	cacheProvider, _ := discoverSuit.HealthCheckServer().CacheProvider()
	instance1 := cacheProvider.GetSelfServiceInstance(instanceId1)
	assert.NotNil(t, instance1)
	assert.Equal(t, true, instance1.Proto.GetHealthy().GetValue())
	instance2 := cacheProvider.GetSelfServiceInstance(instanceId2)
	assert.NotNil(t, instance2)
	assert.Equal(t, true, instance2.Proto.GetHealthy().GetValue())

	delete(instanceIds, instanceId2)
	for i := 0; i < 10; i++ {
		for instanceId := range instanceIds {
			fmt.Printf("%d report instance for %s, round 2\n", i, instanceId)
			discoverSuit.HealthCheckServer().Report(
				context.Background(), &apiservice.Instance{
					Id: &wrapperspb.StringValue{Value: instanceId},
				})
		}
		time.Sleep(2 * time.Second)
	}
	instance1 = cacheProvider.GetSelfServiceInstance(instanceId1)
	assert.NotNil(t, instance1)
	assert.Equal(t, true, instance1.Proto.GetHealthy().GetValue())
	instance2 = cacheProvider.GetSelfServiceInstance(instanceId2)
	assert.NotNil(t, instance2)
	assert.Equal(t, false, instance2.Proto.GetHealthy().GetValue())
}

func TestSelfInstanceImmediatelyCheck(t *testing.T) {
	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	instanceId1 := "inst_self_1"
	instanceId2 := "inst_self_2"

	hbInterval := 1
	discoverSuit.addInstance(t, &apiservice.Instance{
		Id:                wrapperspb.String(instanceId1),
		Service:           wrapperspb.String("polaris.checker"),
		Namespace:         wrapperspb.String("Polaris"),
		Host:              wrapperspb.String(instanceId1),
		Port:              wrapperspb.UInt32(8091),
		Protocol:          wrapperspb.String("grpc"),
		EnableHealthCheck: wrapperspb.Bool(true),
		HealthCheck: &apiservice.HealthCheck{
			Type: apiservice.HealthCheck_HEARTBEAT,
			Heartbeat: &apiservice.HeartbeatHealthCheck{
				Ttl: &wrapperspb.UInt32Value{Value: uint32(hbInterval)},
			},
		},
		Metadata: map[string]string{"polaris_service": "polaris.checker"},
	})

	discoverSuit.addInstance(t, &apiservice.Instance{
		Id:                wrapperspb.String(instanceId2),
		Service:           wrapperspb.String("polaris.checker"),
		Namespace:         wrapperspb.String("Polaris"),
		Host:              wrapperspb.String(instanceId2),
		Port:              wrapperspb.UInt32(8091),
		Protocol:          wrapperspb.String("grpc"),
		EnableHealthCheck: wrapperspb.Bool(false),
		Metadata:          map[string]string{"polaris_service": "polaris.checker"},
	})

	time.Sleep(10 * time.Second)
	_ = discoverSuit.DiscoverServer().Cache().Instance().Update()

	cacheProvider, _ := discoverSuit.HealthCheckServer().CacheProvider()
	instance1 := cacheProvider.GetSelfServiceInstance(instanceId1)
	assert.NotNil(t, instance1)
	assert.Equal(t, false, instance1.Proto.GetHealthy().GetValue())
	instance2 := cacheProvider.GetSelfServiceInstance(instanceId2)
	assert.NotNil(t, instance2)
	assert.Equal(t, true, instance2.Proto.GetHealthy().GetValue())

}
