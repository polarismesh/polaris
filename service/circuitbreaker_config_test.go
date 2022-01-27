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
	"fmt"
	"testing"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/wrappers"
	api "github.com/polarismesh/polaris-server/common/api/v1"
)

func TestServer_CreateCircuitBreakerJson(t *testing.T) {
	rule := &api.CircuitBreaker{}
	rule.Id = &wrappers.StringValue{Value: "12345678"}
	rule.Version = &wrappers.StringValue{Value: "1.0.0"}
	rule.Name = &wrappers.StringValue{Value: "testCbRule"}
	rule.Namespace = &wrappers.StringValue{Value: "Test"}
	rule.Service = &wrappers.StringValue{Value: "TestService1"}
	rule.ServiceNamespace = &wrappers.StringValue{Value: "Test"}
	rule.Inbounds = []*api.CbRule{
		{
			Sources: []*api.SourceMatcher{
				{
					Service:   &wrappers.StringValue{Value: "*"},
					Namespace: &wrappers.StringValue{Value: "*"},
					Labels: map[string]*api.MatchString{
						"user": &api.MatchString{
							Type:  0,
							Value: &wrappers.StringValue{Value: "vip"},
						},
					},
				},
			},
			Destinations: []*api.DestinationSet{
				{
					Method: &api.MatchString{
						Type:  0,
						Value: &wrappers.StringValue{Value: "/info"},
					},
					Resource: api.DestinationSet_INSTANCE,
					Type:     api.DestinationSet_LOCAL,
					Scope:    api.DestinationSet_CURRENT,
					Policy: &api.CbPolicy{
						ErrorRate: &api.CbPolicy_ErrRateConfig{
							Enable:                 &wrappers.BoolValue{Value: true},
							RequestVolumeThreshold: &wrappers.UInt32Value{Value: 10},
							ErrorRateToOpen:        &wrappers.UInt32Value{Value: 50},
						},
						Consecutive: &api.CbPolicy_ConsecutiveErrConfig{
							Enable:                 &wrappers.BoolValue{Value: true},
							ConsecutiveErrorToOpen: &wrappers.UInt32Value{Value: 10},
						},
						SlowRate: &api.CbPolicy_SlowRateConfig{
							Enable:         &wrappers.BoolValue{Value: true},
							MaxRt:          &duration.Duration{Seconds: 1},
							SlowRateToOpen: &wrappers.UInt32Value{Value: 80},
						},
					},
					Recover: &api.RecoverConfig{
						SleepWindow: &duration.Duration{
							Seconds: 1,
						},
						OutlierDetectWhen: api.RecoverConfig_ON_RECOVER,
					},
				},
			},
		},
	}
	rule.Outbounds = []*api.CbRule{
		{
			Sources: []*api.SourceMatcher{
				{
					Labels: map[string]*api.MatchString{
						"callerName": &api.MatchString{
							Type:  0,
							Value: &wrappers.StringValue{Value: "xyz"},
						},
					},
				},
			},
			Destinations: []*api.DestinationSet{
				{
					Namespace: &wrappers.StringValue{Value: "Test"},
					Service:   &wrappers.StringValue{Value: "TestService1"},
					Method: &api.MatchString{
						Type:  0,
						Value: &wrappers.StringValue{Value: "/info"},
					},
					Resource: api.DestinationSet_INSTANCE,
					Type:     api.DestinationSet_LOCAL,
					Scope:    api.DestinationSet_CURRENT,
					Policy: &api.CbPolicy{
						ErrorRate: &api.CbPolicy_ErrRateConfig{
							Enable:                 &wrappers.BoolValue{Value: true},
							RequestVolumeThreshold: &wrappers.UInt32Value{Value: 10},
							ErrorRateToOpen:        &wrappers.UInt32Value{Value: 50},
						},
						Consecutive: &api.CbPolicy_ConsecutiveErrConfig{
							Enable:                 &wrappers.BoolValue{Value: true},
							ConsecutiveErrorToOpen: &wrappers.UInt32Value{Value: 10},
						},
						SlowRate: &api.CbPolicy_SlowRateConfig{
							Enable:         &wrappers.BoolValue{Value: true},
							MaxRt:          &duration.Duration{Seconds: 1},
							SlowRateToOpen: &wrappers.UInt32Value{Value: 80},
						},
					},
					Recover: &api.RecoverConfig{
						SleepWindow: &duration.Duration{
							Seconds: 1,
						},
						OutlierDetectWhen: api.RecoverConfig_ON_RECOVER,
					},
				},
			},
		},
	}
	rule.Business = &wrappers.StringValue{Value: "polaris"}
	rule.Owners = &wrappers.StringValue{Value: "polaris"}

	marshaler := &jsonpb.Marshaler{}
	ruleStr, err := marshaler.MarshalToString(rule)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf(ruleStr)
}
