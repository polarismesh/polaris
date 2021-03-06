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
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/pkg/errors"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/utils"
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
						"user": {
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
						"callerName": {
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

/**
 * @brief ????????????????????????
 */
func TestCreateCircuitBreaker(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	t.Run("???????????????????????????????????????", func(t *testing.T) {
		circuitBreakerReq, circuitBreakerResp := discoverSuit.createCommonCircuitBreaker(t, 0)
		defer discoverSuit.cleanCircuitBreaker(circuitBreakerResp.GetId().GetValue(), circuitBreakerResp.GetVersion().GetValue())
		checkCircuitBreaker(t, circuitBreakerReq, circuitBreakerReq, circuitBreakerResp)
	})

	t.Run("???????????????????????????????????????", func(t *testing.T) {
		_, circuitBreakerResp := discoverSuit.createCommonCircuitBreaker(t, 0)
		defer discoverSuit.cleanCircuitBreaker(circuitBreakerResp.GetId().GetValue(), circuitBreakerResp.GetVersion().GetValue())

		if resp := discoverSuit.server.CreateCircuitBreakers(discoverSuit.defaultCtx, []*api.CircuitBreaker{circuitBreakerResp}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("??????????????????????????????????????????????????????", func(t *testing.T) {
		_, circuitBreakerResp := discoverSuit.createCommonCircuitBreaker(t, 0)
		defer discoverSuit.cleanCircuitBreaker(circuitBreakerResp.GetId().GetValue(), circuitBreakerResp.GetVersion().GetValue())
		discoverSuit.deleteCircuitBreaker(t, circuitBreakerResp)

		newCircuitBreakerReq, newCircuitBreakerResp := discoverSuit.createCommonCircuitBreaker(t, 0)
		checkCircuitBreaker(t, newCircuitBreakerReq, newCircuitBreakerReq, newCircuitBreakerResp)
		discoverSuit.cleanCircuitBreaker(newCircuitBreakerResp.GetId().GetValue(), newCircuitBreakerResp.GetVersion().GetValue())
	})

	t.Run("????????????????????????????????????????????????????????????", func(t *testing.T) {
		circuitBreaker := &api.CircuitBreaker{}
		if resp := discoverSuit.server.CreateCircuitBreakers(discoverSuit.defaultCtx, []*api.CircuitBreaker{circuitBreaker}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("????????????????????????????????????????????????????????????", func(t *testing.T) {
		circuitBreaker := &api.CircuitBreaker{
			Namespace: utils.NewStringValue(DefaultNamespace),
			Owners:    utils.NewStringValue("test"),
		}
		if resp := discoverSuit.server.CreateCircuitBreakers(discoverSuit.defaultCtx, []*api.CircuitBreaker{circuitBreaker}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("???????????????????????????????????????????????????????????????", func(t *testing.T) {
		circuitBreaker := &api.CircuitBreaker{
			Name:   utils.NewStringValue("name-test-1"),
			Owners: utils.NewStringValue("test"),
		}
		if resp := discoverSuit.server.CreateCircuitBreakers(discoverSuit.defaultCtx, []*api.CircuitBreaker{circuitBreaker}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("???????????????????????????????????????", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				_, circuitBreakerResp := discoverSuit.createCommonCircuitBreaker(t, index)
				discoverSuit.cleanCircuitBreaker(circuitBreakerResp.GetId().GetValue(), circuitBreakerResp.GetVersion().GetValue())
			}(i)
		}
		wg.Wait()
	})
}

/**
 * @brief ??????????????????????????????
 */
func TestCreateCircuitBreakerVersion(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	_, cbResp := discoverSuit.createCommonCircuitBreaker(t, 0)
	defer discoverSuit.cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

	t.Run("??????????????????????????????", func(t *testing.T) {
		cbVersionReq, cbVersionResp := discoverSuit.createCommonCircuitBreakerVersion(t, cbResp, 0)
		defer discoverSuit.cleanCircuitBreaker(cbResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())
		checkCircuitBreaker(t, cbVersionReq, cbVersionReq, cbVersionResp)
	})

	t.Run("??????id?????????????????????????????????", func(t *testing.T) {
		cbVersionReq := &api.CircuitBreaker{
			Id:      cbResp.GetId(),
			Version: utils.NewStringValue("test"),
			Token:   cbResp.GetToken(),
		}

		resp := discoverSuit.server.CreateCircuitBreakerVersions(discoverSuit.defaultCtx, []*api.CircuitBreaker{cbVersionReq})
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		cbVersionResp := resp.Responses[0].GetCircuitBreaker()

		defer discoverSuit.cleanCircuitBreaker(cbResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		checkCircuitBreaker(t, cbVersionReq, cbVersionReq, cbVersionResp)
	})

	t.Run("??????name???namespace?????????????????????????????????", func(t *testing.T) {
		cbVersionReq := &api.CircuitBreaker{
			Version:   utils.NewStringValue("test"),
			Name:      cbResp.GetName(),
			Namespace: cbResp.GetNamespace(),
			Token:     cbResp.GetToken(),
		}

		resp := discoverSuit.server.CreateCircuitBreakerVersions(discoverSuit.defaultCtx, []*api.CircuitBreaker{cbVersionReq})
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		cbVersionResp := resp.Responses[0].GetCircuitBreaker()

		defer discoverSuit.cleanCircuitBreaker(cbResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		checkCircuitBreaker(t, cbVersionReq, cbVersionReq, cbVersionResp)
	})

	t.Run("????????????????????????????????????????????????????????????", func(t *testing.T) {
		cbVersionReq, cbVersionResp := discoverSuit.createCommonCircuitBreakerVersion(t, cbResp, 0)
		defer discoverSuit.cleanCircuitBreaker(cbResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		discoverSuit.deleteCircuitBreaker(t, cbVersionResp)
		cbVersionReq, cbVersionResp = discoverSuit.createCommonCircuitBreakerVersion(t, cbResp, 0)
		checkCircuitBreaker(t, cbVersionReq, cbVersionReq, cbVersionResp)
	})

	t.Run("??????????????????????????????????????????????????????", func(t *testing.T) {
		_, cbResp := discoverSuit.createCommonCircuitBreaker(t, 1)
		discoverSuit.cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

		version := &api.CircuitBreaker{
			Id:      cbResp.GetId(),
			Version: utils.NewStringValue("test"),
			Token:   cbResp.GetToken(),
			Owners:  cbResp.GetOwners(),
		}

		if resp := discoverSuit.server.CreateCircuitBreakerVersions(discoverSuit.defaultCtx, []*api.CircuitBreaker{version}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("??????master????????????????????????????????????", func(t *testing.T) {
		if resp := discoverSuit.server.CreateCircuitBreakerVersions(discoverSuit.defaultCtx, []*api.CircuitBreaker{cbResp}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("??????????????????????????????????????????version???????????????", func(t *testing.T) {
		version := &api.CircuitBreaker{
			Id:     cbResp.GetId(),
			Token:  cbResp.GetToken(),
			Owners: cbResp.GetOwners(),
		}
		if resp := discoverSuit.server.CreateCircuitBreakerVersions(discoverSuit.defaultCtx, []*api.CircuitBreaker{version}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("??????????????????????????????????????????token???????????????", func(t *testing.T) {
		version := &api.CircuitBreaker{
			Id:      cbResp.GetId(),
			Version: cbResp.GetVersion(),
			Owners:  cbResp.GetOwners(),
		}
		if resp := discoverSuit.server.CreateCircuitBreakerVersions(discoverSuit.defaultCtx, []*api.CircuitBreaker{version}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("??????????????????????????????????????????name???????????????", func(t *testing.T) {
		version := &api.CircuitBreaker{
			Version:   cbResp.GetVersion(),
			Token:     cbResp.GetToken(),
			Namespace: cbResp.GetNamespace(),
		}
		if resp := discoverSuit.server.CreateCircuitBreakerVersions(discoverSuit.defaultCtx, []*api.CircuitBreaker{version}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("??????????????????????????????????????????namespace???????????????", func(t *testing.T) {
		version := &api.CircuitBreaker{
			Version: cbResp.GetVersion(),
			Token:   cbResp.GetToken(),
			Name:    cbResp.GetName(),
		}
		if resp := discoverSuit.server.CreateCircuitBreakerVersions(discoverSuit.defaultCtx, []*api.CircuitBreaker{version}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("?????????????????????????????????????????????????????????", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 0; i <= 50; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				cbVersionReq, cbVersionResp := discoverSuit.createCommonCircuitBreakerVersion(t, cbResp, index)
				checkCircuitBreaker(t, cbVersionReq, cbVersionReq, cbVersionResp)
				defer discoverSuit.cleanCircuitBreaker(cbResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())
			}(i)
		}
		wg.Wait()
		t.Log("pass")
	})
}

/**
 * @brief ??????????????????
 */
func Test_DeleteCircuitBreaker(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	getCircuitBreakerVersions := func(t *testing.T, id string, expectNum uint32) {
		filters := map[string]string{
			"id": id,
		}
		resp := discoverSuit.server.GetCircuitBreakerVersions(context.Background(), filters)
		if !respSuccess(resp) {
			t.Fatal("error")
		}
		if resp.GetAmount().GetValue() != expectNum {
			t.Fatalf("error, actual num is %d, expect num is %d", resp.GetAmount().GetValue(), expectNum)
		} else {
			t.Log("pass")
		}
	}

	t.Run("??????name???namespace??????master?????????????????????", func(t *testing.T) {
		// ??????????????????
		_, cbResp := discoverSuit.createCommonCircuitBreaker(t, 0)
		defer discoverSuit.cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

		// ????????????????????????
		for i := 1; i <= 10; i++ {
			_, cbVersionResp := discoverSuit.createCommonCircuitBreakerVersion(t, cbResp, i)
			defer discoverSuit.cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())
		}

		rule := &api.CircuitBreaker{
			Version:   cbResp.GetVersion(),
			Name:      cbResp.GetName(),
			Namespace: cbResp.GetNamespace(),
			Token:     cbResp.GetToken(),
		}

		discoverSuit.deleteCircuitBreaker(t, rule)
		getCircuitBreakerVersions(t, cbResp.GetId().GetValue(), 10)
	})

	t.Run("??????master?????????????????????", func(t *testing.T) {
		// ??????????????????
		_, cbResp := discoverSuit.createCommonCircuitBreaker(t, 0)
		defer discoverSuit.cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

		// ????????????????????????
		for i := 1; i <= 10; i++ {
			_, cbVersionResp := discoverSuit.createCommonCircuitBreakerVersion(t, cbResp, i)
			defer discoverSuit.cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())
		}

		discoverSuit.deleteCircuitBreaker(t, cbResp)
		getCircuitBreakerVersions(t, cbResp.GetId().GetValue(), 10)
	})

	t.Run("?????????master?????????????????????", func(t *testing.T) {
		// ??????????????????
		_, cbResp := discoverSuit.createCommonCircuitBreaker(t, 0)
		defer discoverSuit.cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

		// ????????????????????????
		_, cbVersionResp := discoverSuit.createCommonCircuitBreakerVersion(t, cbResp, 0)
		defer discoverSuit.cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		// ????????????????????????
		for i := 1; i <= 10; i++ {
			_, cbVersionResp := discoverSuit.createCommonCircuitBreakerVersion(t, cbResp, i)
			defer discoverSuit.cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())
		}

		// ?????????????????????????????????
		discoverSuit.deleteCircuitBreaker(t, cbVersionResp)

		getCircuitBreakerVersions(t, cbResp.GetId().GetValue(), 1+10)
	})

	t.Run("??????name???namespace?????????master?????????????????????", func(t *testing.T) {
		// ??????????????????
		_, cbResp := discoverSuit.createCommonCircuitBreaker(t, 0)
		defer discoverSuit.cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

		// ????????????????????????
		_, cbVersionResp := discoverSuit.createCommonCircuitBreakerVersion(t, cbResp, 0)
		defer discoverSuit.cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		// ????????????????????????
		for i := 1; i <= 10; i++ {
			_, cbVersionResp := discoverSuit.createCommonCircuitBreakerVersion(t, cbResp, i)
			defer discoverSuit.cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())
		}

		// ?????????????????????????????????
		rule := &api.CircuitBreaker{
			Version:   cbVersionResp.GetVersion(),
			Name:      cbVersionResp.GetName(),
			Namespace: cbVersionResp.GetNamespace(),
			Token:     cbVersionResp.GetToken(),
		}
		discoverSuit.deleteCircuitBreaker(t, rule)

		getCircuitBreakerVersions(t, cbResp.GetId().GetValue(), 1+10)
	})

	t.Run("?????????????????????????????????????????????", func(t *testing.T) {
		// ??????????????????
		_, cbResp := discoverSuit.createCommonCircuitBreaker(t, 0)
		discoverSuit.cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

		discoverSuit.deleteCircuitBreaker(t, cbResp)
		getCircuitBreakerVersions(t, cbResp.GetId().GetValue(), 0)
	})

	t.Run("????????????????????????????????????token???????????????", func(t *testing.T) {
		// ??????????????????
		_, cbResp := discoverSuit.createCommonCircuitBreaker(t, 0)
		defer discoverSuit.cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

		rule := &api.CircuitBreaker{
			Id:      cbResp.GetId(),
			Version: cbResp.GetVersion(),
		}

		if resp := discoverSuit.server.DeleteCircuitBreakers(discoverSuit.defaultCtx, []*api.CircuitBreaker{rule}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("????????????????????????????????????name???id???????????????", func(t *testing.T) {
		// ??????????????????
		_, cbResp := discoverSuit.createCommonCircuitBreaker(t, 0)
		defer discoverSuit.cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

		rule := &api.CircuitBreaker{
			Version:   cbResp.GetVersion(),
			Namespace: cbResp.GetNamespace(),
			Token:     cbResp.GetToken(),
		}

		if resp := discoverSuit.server.DeleteCircuitBreakers(discoverSuit.defaultCtx, []*api.CircuitBreaker{rule}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("???????????????????????????????????????", func(t *testing.T) {
		// ????????????
		_, serviceResp := discoverSuit.createCommonService(t, 0)
		defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		// ??????????????????
		_, cbResp := discoverSuit.createCommonCircuitBreaker(t, 0)
		defer discoverSuit.cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

		// ????????????????????????
		_, cbVersionResp := discoverSuit.createCommonCircuitBreakerVersion(t, cbResp, 0)
		defer discoverSuit.cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		// ??????????????????
		discoverSuit.releaseCircuitBreaker(t, cbVersionResp, serviceResp)
		defer discoverSuit.cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		// // ??????master??????
		// if resp := discoverSuit.server.DeleteCircuitBreakers(discoverSuit.defaultCtx, []*api.CircuitBreaker{cbResp}); !respSuccess(resp) {
		// 	t.Logf("pass: %s", resp.GetInfo().GetValue())
		// } else {
		// 	t.Fatalf("error : %s", resp.GetInfo().GetValue())
		// }

		// ??????????????????
		if resp := discoverSuit.server.DeleteCircuitBreakers(discoverSuit.defaultCtx, []*api.CircuitBreaker{cbVersionResp}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("??????????????????????????????????????????????????????????????????????????????????????????????????????", func(t *testing.T) {
		// ????????????
		_, serviceResp := discoverSuit.createCommonService(t, 0)
		defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		// ??????????????????
		_, cbResp := discoverSuit.createCommonCircuitBreaker(t, 0)
		defer discoverSuit.cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

		// ????????????????????????
		_, cbVersionResp := discoverSuit.createCommonCircuitBreakerVersion(t, cbResp, 0)
		defer discoverSuit.cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		// ????????????????????????
		_, newCbVersionResp := discoverSuit.createCommonCircuitBreakerVersion(t, cbResp, 1)
		defer discoverSuit.cleanCircuitBreaker(newCbVersionResp.GetId().GetValue(), newCbVersionResp.GetVersion().GetValue())

		// ??????????????????
		discoverSuit.releaseCircuitBreaker(t, cbVersionResp, serviceResp)
		defer discoverSuit.cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		discoverSuit.deleteCircuitBreaker(t, newCbVersionResp)
		getCircuitBreakerVersions(t, cbResp.GetId().GetValue(), 1+1)
	})

	t.Run("?????????????????????????????????????????????", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 1; i <= 50; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				_, cbResp := discoverSuit.createCommonCircuitBreaker(t, index)
				defer discoverSuit.cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())
				discoverSuit.deleteCircuitBreaker(t, cbResp)
			}(i)
		}
		wg.Wait()
		t.Log("pass")
	})
}

/**
 * @brief ????????????????????????
 */
func TestUpdateCircuitBreaker(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	// ??????????????????
	_, cbResp := discoverSuit.createCommonCircuitBreaker(t, 0)
	defer discoverSuit.cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

	t.Run("??????master????????????????????????????????????", func(t *testing.T) {
		cbResp.Inbounds = []*api.CbRule{}
		discoverSuit.updateCircuitBreaker(t, cbResp)

		filters := map[string]string{
			"id":      cbResp.GetId().GetValue(),
			"version": cbResp.GetVersion().GetValue(),
		}

		resp := discoverSuit.server.GetCircuitBreaker(context.Background(), filters)
		if !respSuccess(resp) {
			t.Fatal("error")
		}
		checkCircuitBreaker(t, cbResp, cbResp, resp.GetConfigWithServices()[0].GetCircuitBreaker())
	})

	t.Run("????????????????????????????????????????????????", func(t *testing.T) {
		if resp := discoverSuit.server.UpdateCircuitBreakers(discoverSuit.defaultCtx, []*api.CircuitBreaker{cbResp}); respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("????????????????????????????????????????????????????????????", func(t *testing.T) {
		rule := &api.CircuitBreaker{
			Id:      cbResp.GetId(),
			Version: cbResp.GetVersion(),
			Token:   cbResp.GetToken(),
		}
		if resp := discoverSuit.server.UpdateCircuitBreakers(discoverSuit.defaultCtx, []*api.CircuitBreaker{rule}); respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("??????????????????????????????", func(t *testing.T) {
		rule := &api.CircuitBreaker{
			Id:      cbResp.GetId(),
			Version: cbResp.GetVersion(),
			Token:   cbResp.GetToken(),
			Owners:  utils.NewStringValue(""),
		}
		if resp := discoverSuit.server.UpdateCircuitBreakers(discoverSuit.defaultCtx, []*api.CircuitBreaker{rule}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("????????????????????????????????????????????????", func(t *testing.T) {
		// ????????????????????????
		_, cbVersionResp := discoverSuit.createCommonCircuitBreakerVersion(t, cbResp, 0)
		defer discoverSuit.cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		if resp := discoverSuit.server.UpdateCircuitBreakers(discoverSuit.defaultCtx, []*api.CircuitBreaker{cbVersionResp}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("?????????????????????????????????????????????", func(t *testing.T) {
		discoverSuit.cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())
		if resp := discoverSuit.server.UpdateCircuitBreakers(discoverSuit.defaultCtx, []*api.CircuitBreaker{cbResp}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("????????????????????????????????????token???????????????", func(t *testing.T) {
		rule := &api.CircuitBreaker{
			Id:      cbResp.GetId(),
			Version: cbResp.GetVersion(),
		}
		if resp := discoverSuit.server.UpdateCircuitBreakers(discoverSuit.defaultCtx, []*api.CircuitBreaker{rule}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("???????????????????????????,??????????????????", func(t *testing.T) {
		var wg sync.WaitGroup
		errs := make(chan error)
		for i := 1; i <= 50; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				// ??????????????????
				_, cbResp := discoverSuit.createCommonCircuitBreaker(t, index)
				defer discoverSuit.cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

				cbResp.Owners = utils.NewStringValue(fmt.Sprintf("test-owner-%d", index))

				discoverSuit.updateCircuitBreaker(t, cbResp)

				filters := map[string]string{
					"id":      cbResp.GetId().GetValue(),
					"version": cbResp.GetVersion().GetValue(),
				}
				resp := discoverSuit.server.GetCircuitBreaker(context.Background(), filters)
				if !respSuccess(resp) {
					errs <- fmt.Errorf("error : %v", resp)
					return
				}

				if len(resp.GetConfigWithServices()) != 1 {
					panic(errors.WithStack(fmt.Errorf("%#v", resp)))
				}

				checkCircuitBreaker(t, cbResp, cbResp, resp.GetConfigWithServices()[0].GetCircuitBreaker())
			}(i)
		}
		wg.Wait()

		select {
		case err := <-errs:
			if err != nil {
				t.Fatal(err)
			}
		default:
			return
		}
	})
}

/**
 * @brief ????????????????????????
 */
func TestReleaseCircuitBreaker(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	// ????????????
	_, serviceResp := discoverSuit.createCommonService(t, 0)
	defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	// ??????????????????
	_, cbResp := discoverSuit.createCommonCircuitBreaker(t, 0)
	defer discoverSuit.cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

	// ???????????????????????????
	_, cbVersionResp := discoverSuit.createCommonCircuitBreakerVersion(t, cbResp, 0)
	defer discoverSuit.cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

	t.Run("????????????????????????", func(t *testing.T) {
		_ = discoverSuit.server.Cache().Clear()

		time.Sleep(5 * time.Second)

		discoverSuit.releaseCircuitBreaker(t, cbVersionResp, serviceResp)
		defer discoverSuit.cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		// ??????????????????
		time.Sleep(discoverSuit.updateCacheInterval)

		resp := discoverSuit.server.GetCircuitBreakerWithCache(discoverSuit.defaultCtx, serviceResp)
		checkCircuitBreaker(t, cbVersionResp, cbResp, resp.GetCircuitBreaker())
	})

	t.Run("??????name???namespace??????????????????", func(t *testing.T) {
		_ = discoverSuit.server.Cache().Clear()

		time.Sleep(5 * time.Second)

		rule := &api.CircuitBreaker{
			Version:   cbVersionResp.GetVersion(),
			Name:      cbVersionResp.GetName(),
			Namespace: cbVersionResp.GetNamespace(),
		}
		discoverSuit.releaseCircuitBreaker(t, rule, serviceResp)
		defer discoverSuit.cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		// ??????????????????
		time.Sleep(discoverSuit.updateCacheInterval)

		resp := discoverSuit.server.GetCircuitBreakerWithCache(discoverSuit.defaultCtx, serviceResp)
		checkCircuitBreaker(t, cbVersionResp, cbResp, resp.GetCircuitBreaker())
	})

	t.Run("????????????????????????????????????????????????", func(t *testing.T) {
		_ = discoverSuit.server.Cache().Clear()

		time.Sleep(5 * time.Second)

		discoverSuit.releaseCircuitBreaker(t, cbVersionResp, serviceResp)
		defer discoverSuit.cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		// ???????????????????????????
		_, cbVersionResp := discoverSuit.createCommonCircuitBreakerVersion(t, cbResp, 1)
		defer discoverSuit.cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		// ????????????????????????
		discoverSuit.releaseCircuitBreaker(t, cbVersionResp, serviceResp)
		defer discoverSuit.cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		// ??????????????????
		time.Sleep(discoverSuit.updateCacheInterval)

		resp := discoverSuit.server.GetCircuitBreakerWithCache(discoverSuit.defaultCtx, serviceResp)
		checkCircuitBreaker(t, cbVersionResp, cbResp, resp.GetCircuitBreaker())
	})

	t.Run("??????????????????????????????????????????????????????", func(t *testing.T) {
		_ = discoverSuit.server.Cache().Clear()

		time.Sleep(5 * time.Second)

		discoverSuit.releaseCircuitBreaker(t, cbVersionResp, serviceResp)
		defer discoverSuit.cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		// ????????????
		_, serviceResp2 := discoverSuit.createCommonService(t, 1)
		defer discoverSuit.cleanServiceName(serviceResp2.GetName().GetValue(), serviceResp2.GetNamespace().GetValue())

		discoverSuit.releaseCircuitBreaker(t, cbVersionResp, serviceResp2)
		defer discoverSuit.cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		// ??????????????????
		time.Sleep(2 * discoverSuit.updateCacheInterval)

		ret, err := discoverSuit.storage.GetCircuitBreakerForCache(time.Time{}, true)
		if err != nil {
			t.Fatal(err)
		}

		s, _ := json.Marshal(ret)
		t.Logf("cb cache : %#v", string(s))

		resp := discoverSuit.server.GetCircuitBreakerWithCache(discoverSuit.defaultCtx, serviceResp)
		t.Logf("%s service-1 resp : %#v", time.Now().String(), resp.GetCircuitBreaker())
		checkCircuitBreaker(t, cbVersionResp, cbResp, resp.GetCircuitBreaker())

		resp = discoverSuit.server.GetCircuitBreakerWithCache(discoverSuit.defaultCtx, serviceResp2)
		t.Logf("service-2 resp : %#v", resp.GetCircuitBreaker())
		checkCircuitBreaker(t, cbVersionResp, cbResp, resp.GetCircuitBreaker())
	})

	t.Run("???????????????????????????????????????????????????????????????", func(t *testing.T) {
		release := &api.ConfigRelease{
			Service: &api.Service{
				Name:      serviceResp.GetName(),
				Namespace: utils.NewStringValue("Test"),
				Token:     serviceResp.GetToken(),
			},
			CircuitBreaker: cbVersionResp,
		}

		if resp := discoverSuit.server.ReleaseCircuitBreakers(discoverSuit.defaultCtx, []*api.ConfigRelease{release}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("???????????????????????????????????????????????????????????????", func(t *testing.T) {
		discoverSuit.releaseCircuitBreaker(t, cbVersionResp, serviceResp)
		defer discoverSuit.cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		release := &api.ConfigRelease{
			Service:        serviceResp,
			CircuitBreaker: cbVersionResp,
		}

		if resp := discoverSuit.server.ReleaseCircuitBreakers(discoverSuit.defaultCtx, []*api.ConfigRelease{release}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("????????????????????????????????????token???????????????", func(t *testing.T) {

		oldCtx := discoverSuit.defaultCtx
		discoverSuit.defaultCtx = context.Background()

		defer func() {
			discoverSuit.defaultCtx = oldCtx
		}()

		release := &api.ConfigRelease{
			Service: &api.Service{
				Name:      serviceResp.GetName(),
				Namespace: serviceResp.GetNamespace(),
			},
			CircuitBreaker: cbVersionResp,
		}
		if resp := discoverSuit.server.ReleaseCircuitBreakers(discoverSuit.defaultCtx, []*api.ConfigRelease{release}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("???????????????????????????????????????????????????", func(t *testing.T) {
		_, serviceResp := discoverSuit.createCommonService(t, 1)
		discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		release := &api.ConfigRelease{
			Service:        serviceResp,
			CircuitBreaker: cbVersionResp,
		}
		if resp := discoverSuit.server.ReleaseCircuitBreakers(discoverSuit.defaultCtx, []*api.ConfigRelease{release}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("????????????????????????master?????????????????????", func(t *testing.T) {
		release := &api.ConfigRelease{
			Service:        serviceResp,
			CircuitBreaker: cbResp,
		}
		if resp := discoverSuit.server.ReleaseCircuitBreakers(discoverSuit.defaultCtx, []*api.ConfigRelease{release}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("?????????????????????????????????????????????", func(t *testing.T) {
		_, cbVersionResp := discoverSuit.createCommonCircuitBreakerVersion(t, cbResp, 1)
		discoverSuit.cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		release := &api.ConfigRelease{
			Service:        serviceResp,
			CircuitBreaker: cbVersionResp,
		}
		if resp := discoverSuit.server.ReleaseCircuitBreakers(discoverSuit.defaultCtx, []*api.ConfigRelease{release}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("??????????????????????????????????????????", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 1; i <= 50; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				_, cbVersionResp := discoverSuit.createCommonCircuitBreakerVersion(t, cbResp, index)
				defer discoverSuit.cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

				discoverSuit.releaseCircuitBreaker(t, cbVersionResp, serviceResp)
				defer discoverSuit.cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
					cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())
			}(i)
		}
		wg.Wait()
		t.Log("pass")
	})
}

/**
 * @brief ????????????????????????
 */
func TestUnBindCircuitBreaker(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	// ????????????
	_, serviceResp := discoverSuit.createCommonService(t, 0)
	defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	// ??????????????????
	_, cbResp := discoverSuit.createCommonCircuitBreaker(t, 0)
	defer discoverSuit.cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

	// ???????????????????????????
	_, cbVersionResp := discoverSuit.createCommonCircuitBreakerVersion(t, cbResp, 0)
	defer discoverSuit.cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

	t.Run("????????????????????????", func(t *testing.T) {
		_ = discoverSuit.server.Cache().Clear()

		time.Sleep(5 * time.Second)

		// ??????????????????
		discoverSuit.releaseCircuitBreaker(t, cbVersionResp, serviceResp)
		defer discoverSuit.cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		discoverSuit.unBindCircuitBreaker(t, cbVersionResp, serviceResp)

		// ??????????????????
		time.Sleep(discoverSuit.updateCacheInterval)

		resp := discoverSuit.server.GetCircuitBreakerWithCache(discoverSuit.defaultCtx, serviceResp)
		if resp != nil && resp.GetCircuitBreaker() == nil {
			t.Log("pass")
		} else {
			t.Fatalf("err is %+v", resp)
		}
	})

	t.Run("????????????????????????????????????, ????????????", func(t *testing.T) {
		_ = discoverSuit.server.Cache().Clear()

		time.Sleep(5 * time.Second)

		// ??????????????????
		discoverSuit.releaseCircuitBreaker(t, cbVersionResp, serviceResp)
		defer discoverSuit.cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		// ???????????????????????????
		_, newCbVersionResp := discoverSuit.createCommonCircuitBreakerVersion(t, cbResp, 1)
		defer discoverSuit.cleanCircuitBreaker(newCbVersionResp.GetId().GetValue(), newCbVersionResp.GetVersion().GetValue())

		discoverSuit.unBindCircuitBreaker(t, newCbVersionResp, serviceResp)

		// ??????????????????
		time.Sleep(discoverSuit.updateCacheInterval)

		resp := discoverSuit.server.GetCircuitBreakerWithCache(discoverSuit.defaultCtx, serviceResp)
		checkCircuitBreaker(t, cbVersionResp, cbResp, resp.GetCircuitBreaker())
	})

	t.Run("???????????????????????????token???????????????", func(t *testing.T) {
		oldCtx := discoverSuit.defaultCtx
		discoverSuit.defaultCtx = context.Background()

		defer func() {
			discoverSuit.defaultCtx = oldCtx
		}()

		unbind := &api.ConfigRelease{
			Service: &api.Service{
				Name:      serviceResp.GetName(),
				Namespace: serviceResp.GetNamespace(),
			},
			CircuitBreaker: cbVersionResp,
		}

		if resp := discoverSuit.server.UnBindCircuitBreakers(discoverSuit.defaultCtx, []*api.ConfigRelease{unbind}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("???????????????????????????????????????????????????", func(t *testing.T) {
		_, serviceResp := discoverSuit.createCommonService(t, 1)
		discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		unbind := &api.ConfigRelease{
			Service:        serviceResp,
			CircuitBreaker: cbVersionResp,
		}

		if resp := discoverSuit.server.UnBindCircuitBreakers(discoverSuit.defaultCtx, []*api.ConfigRelease{unbind}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("???????????????????????????????????????????????????", func(t *testing.T) {
		// ???????????????????????????
		_, cbVersionResp := discoverSuit.createCommonCircuitBreakerVersion(t, cbResp, 1)
		discoverSuit.cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		unbind := &api.ConfigRelease{
			Service:        serviceResp,
			CircuitBreaker: cbVersionResp,
		}

		if resp := discoverSuit.server.UnBindCircuitBreakers(discoverSuit.defaultCtx, []*api.ConfigRelease{unbind}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("??????master????????????????????????????????????", func(t *testing.T) {
		unbind := &api.ConfigRelease{
			Service:        serviceResp,
			CircuitBreaker: cbResp,
		}

		if resp := discoverSuit.server.UnBindCircuitBreakers(discoverSuit.defaultCtx, []*api.ConfigRelease{unbind}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("?????????????????????????????????name???????????????", func(t *testing.T) {
		unbind := &api.ConfigRelease{
			Service: serviceResp,
			CircuitBreaker: &api.CircuitBreaker{
				Version:   cbVersionResp.GetVersion(),
				Namespace: cbVersionResp.GetNamespace(),
			},
		}

		if resp := discoverSuit.server.UnBindCircuitBreakers(discoverSuit.defaultCtx, []*api.ConfigRelease{unbind}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("????????????????????????", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 1; i <= 50; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				// ????????????
				_, serviceResp := discoverSuit.createCommonService(t, index)
				defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

				// ??????????????????
				discoverSuit.releaseCircuitBreaker(t, cbVersionResp, serviceResp)
				defer discoverSuit.cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
					cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

				discoverSuit.unBindCircuitBreaker(t, cbVersionResp, serviceResp)
			}(i)
		}
		wg.Wait()
		t.Log("pass")
	})
}

/**
 * @brief ????????????????????????
 */
func TestGetCircuitBreaker(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	versionNum := 10
	serviceNum := 2
	releaseVersion := &api.CircuitBreaker{}
	deleteVersion := &api.CircuitBreaker{}
	svc := &api.Service{}

	// ??????????????????
	_, cbResp := discoverSuit.createCommonCircuitBreaker(t, 0)
	defer discoverSuit.cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

	// ????????????????????????
	for i := 1; i <= versionNum; i++ {
		// ???????????????????????????
		_, cbVersionResp := discoverSuit.createCommonCircuitBreakerVersion(t, cbResp, i)
		defer discoverSuit.cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		if i == 5 {
			releaseVersion = cbVersionResp
		}

		if i == versionNum {
			deleteVersion = cbVersionResp
		}
	}

	// ?????????????????????????????????
	discoverSuit.deleteCircuitBreaker(t, deleteVersion)

	// ??????????????????
	for i := 1; i <= serviceNum; i++ {
		_, serviceResp := discoverSuit.createCommonService(t, i)
		if i == 1 {
			svc = serviceResp
		}
		defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		discoverSuit.releaseCircuitBreaker(t, releaseVersion, serviceResp)
		defer discoverSuit.cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			releaseVersion.GetId().GetValue(), releaseVersion.GetVersion().GetValue())
	}

	t.Run("???????????????????????????????????????", func(t *testing.T) {
		filters := map[string]string{
			"id": cbResp.GetId().GetValue(),
		}

		resp := discoverSuit.server.GetCircuitBreakerVersions(context.Background(), filters)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if resp.GetAmount().GetValue() != resp.GetSize().GetValue() ||
			resp.GetSize().GetValue() != uint32(versionNum) || len(resp.GetConfigWithServices()) != versionNum {
			t.Fatalf("amount is %d, size is %d, num is %d, expect num is %d", resp.GetAmount().GetValue(),
				resp.GetSize().GetValue(), len(resp.GetConfigWithServices()), versionNum)
		}
		t.Logf("pass: num is %d", resp.GetSize().GetValue())
	})

	t.Run("??????????????????????????????????????????", func(t *testing.T) {
		filters := map[string]string{
			"id": cbResp.GetId().GetValue(),
		}

		resp := discoverSuit.server.GetReleaseCircuitBreakers(context.Background(), filters)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if resp.GetAmount().GetValue() != resp.GetSize().GetValue() ||
			resp.GetSize().GetValue() != uint32(serviceNum) {
			t.Fatalf("amount is %d, size is %d, expect num is %d", resp.GetAmount().GetValue(),
				resp.GetSize().GetValue(), versionNum)
		}
		t.Logf("pass: num is %d", resp.GetSize().GetValue())
	})

	t.Run("???????????????????????????????????????", func(t *testing.T) {
		filters := map[string]string{
			"id":      releaseVersion.GetId().GetValue(),
			"version": releaseVersion.GetVersion().GetValue(),
		}

		resp := discoverSuit.server.GetCircuitBreaker(context.Background(), filters)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		checkCircuitBreaker(t, releaseVersion, cbResp, resp.GetConfigWithServices()[0].GetCircuitBreaker())
	})

	t.Run("???????????????????????????????????????", func(t *testing.T) {
		filters := map[string]string{
			"service":   svc.GetName().GetValue(),
			"namespace": svc.GetNamespace().GetValue(),
		}

		resp := discoverSuit.server.GetCircuitBreakerByService(context.Background(), filters)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		checkCircuitBreaker(t, releaseVersion, cbResp, resp.GetConfigWithServices()[0].GetCircuitBreaker())
	})
}

/**
 * @brief ????????????????????????
 */
func TestGetCircuitBreaker2(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	// ????????????
	_, serviceResp := discoverSuit.createCommonService(t, 0)
	defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	// ??????????????????
	_, cbResp := discoverSuit.createCommonCircuitBreaker(t, 0)
	discoverSuit.cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

	t.Run("????????????????????????????????????????????????", func(t *testing.T) {
		filters := map[string]string{
			"id": cbResp.GetId().GetValue(),
		}

		resp := discoverSuit.server.GetCircuitBreakerVersions(context.Background(), filters)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if resp.GetAmount().GetValue() != 0 || resp.GetSize().GetValue() != 0 ||
			len(resp.GetConfigWithServices()) != 0 {
			t.Fatalf("amount is %d, size is %d, num is %d", resp.GetAmount().GetValue(),
				resp.GetSize().GetValue(), len(resp.GetConfigWithServices()))
		}
		t.Logf("pass: resp is %+v, configServices is %+v", resp, resp.GetConfigWithServices())
	})

	t.Run("????????????????????????????????????????????????????????????", func(t *testing.T) {
		filters := map[string]string{
			"id": cbResp.GetId().GetValue(),
		}

		resp := discoverSuit.server.GetReleaseCircuitBreakers(context.Background(), filters)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if resp.GetAmount().GetValue() != 0 || resp.GetSize().GetValue() != 0 ||
			len(resp.GetConfigWithServices()) != 0 {
			t.Fatalf("amount is %d, size is %d, num is %d", resp.GetAmount().GetValue(),
				resp.GetSize().GetValue(), len(resp.GetConfigWithServices()))
		}
		t.Logf("pass: resp is %+v, configServices is %+v", resp, resp.GetConfigWithServices())
	})

	t.Run("???????????????????????????????????????????????????????????????", func(t *testing.T) {
		filters := map[string]string{
			"id":      cbResp.GetId().GetValue(),
			"version": cbResp.GetVersion().GetValue(),
		}

		resp := discoverSuit.server.GetCircuitBreaker(context.Background(), filters)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if resp.GetAmount().GetValue() != 0 || resp.GetSize().GetValue() != 0 ||
			len(resp.GetConfigWithServices()) != 0 {
			t.Fatalf("amount is %d, size is %d, num is %d", resp.GetAmount().GetValue(),
				resp.GetSize().GetValue(), len(resp.GetConfigWithServices()))
		}
		t.Logf("pass: resp is %+v, configServices is %+v", resp, resp.GetConfigWithServices())
	})

	t.Run("????????????????????????????????????????????????", func(t *testing.T) {
		filters := map[string]string{
			"service":   serviceResp.GetName().GetValue(),
			"namespace": serviceResp.GetNamespace().GetValue(),
		}

		resp := discoverSuit.server.GetCircuitBreakerByService(context.Background(), filters)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if resp.GetAmount().GetValue() != 0 || resp.GetSize().GetValue() != 0 ||
			len(resp.GetConfigWithServices()) != 0 {
			t.Fatalf("amount is %d, size is %d, num is %d", resp.GetAmount().GetValue(),
				resp.GetSize().GetValue(), len(resp.GetConfigWithServices()))
		}
		t.Logf("pass: resp is %+v, configServices is %+v", resp, resp.GetConfigWithServices())
	})
}

// test???CircuitBreaker??????????????????
func TestCheckCircuitBreakerFieldLen(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	circuitBreaker := &api.CircuitBreaker{
		Name:       utils.NewStringValue("name-test-123"),
		Namespace:  utils.NewStringValue(DefaultNamespace),
		Owners:     utils.NewStringValue("owner-test"),
		Comment:    utils.NewStringValue("comment-test"),
		Department: utils.NewStringValue("department-test"),
		Business:   utils.NewStringValue("business-test"),
	}
	t.Run("???????????????", func(t *testing.T) {
		str := genSpecialStr(500)
		oldName := circuitBreaker.Name
		circuitBreaker.Name = utils.NewStringValue(str)
		resp := discoverSuit.server.CreateCircuitBreakers(discoverSuit.defaultCtx, []*api.CircuitBreaker{circuitBreaker})
		circuitBreaker.Name = oldName
		if resp.Code.Value != api.InvalidCircuitBreakerName {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("????????????????????????", func(t *testing.T) {
		str := genSpecialStr(65)
		oldNamespace := circuitBreaker.Namespace
		circuitBreaker.Namespace = utils.NewStringValue(str)
		resp := discoverSuit.server.CreateCircuitBreakers(discoverSuit.defaultCtx, []*api.CircuitBreaker{circuitBreaker})
		circuitBreaker.Namespace = oldNamespace
		if resp.Code.Value != api.InvalidCircuitBreakerNamespace {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("??????business??????", func(t *testing.T) {
		str := genSpecialStr(65)
		oldBusiness := circuitBreaker.Business
		circuitBreaker.Business = utils.NewStringValue(str)
		resp := discoverSuit.server.CreateCircuitBreakers(discoverSuit.defaultCtx, []*api.CircuitBreaker{circuitBreaker})
		circuitBreaker.Business = oldBusiness
		if resp.Code.Value != api.InvalidCircuitBreakerBusiness {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("??????????????????", func(t *testing.T) {
		str := genSpecialStr(1025)
		oldDepartment := circuitBreaker.Department
		circuitBreaker.Department = utils.NewStringValue(str)
		resp := discoverSuit.server.CreateCircuitBreakers(discoverSuit.defaultCtx, []*api.CircuitBreaker{circuitBreaker})
		circuitBreaker.Department = oldDepartment
		if resp.Code.Value != api.InvalidCircuitBreakerDepartment {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("??????comment??????", func(t *testing.T) {
		str := genSpecialStr(1025)
		oldComment := circuitBreaker.Comment
		circuitBreaker.Comment = utils.NewStringValue(str)
		resp := discoverSuit.server.CreateCircuitBreakers(discoverSuit.defaultCtx, []*api.CircuitBreaker{circuitBreaker})
		circuitBreaker.Comment = oldComment
		if resp.Code.Value != api.InvalidCircuitBreakerComment {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("??????owner??????", func(t *testing.T) {
		str := genSpecialStr(1025)
		oldOwners := circuitBreaker.Owners
		circuitBreaker.Owners = utils.NewStringValue(str)
		resp := discoverSuit.server.CreateCircuitBreakers(discoverSuit.defaultCtx, []*api.CircuitBreaker{circuitBreaker})
		circuitBreaker.Owners = oldOwners
		if resp.Code.Value != api.InvalidCircuitBreakerOwners {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("????????????????????????", func(t *testing.T) {
		release := &api.ConfigRelease{
			Service: &api.Service{
				Name:      utils.NewStringValue("test"),
				Namespace: utils.NewStringValue("default"),
				Token:     utils.NewStringValue("test"),
			},
			CircuitBreaker: &api.CircuitBreaker{
				Name:      utils.NewStringValue("test"),
				Namespace: utils.NewStringValue("default"),
				Version:   utils.NewStringValue("1.0"),
			},
		}
		t.Run("?????????????????????????????????", func(t *testing.T) {
			str := genSpecialStr(1025)
			oldName := release.Service.Name
			release.Service.Name = utils.NewStringValue(str)
			resp := discoverSuit.server.ReleaseCircuitBreakers(discoverSuit.defaultCtx, []*api.ConfigRelease{release})
			release.Service.Name = oldName
			if resp.Code.Value != api.InvalidServiceName {
				t.Fatalf("%+v", resp)
			}
		})
		t.Run("??????????????????????????????????????????", func(t *testing.T) {
			str := genSpecialStr(1025)
			oldNamespace := release.Service.Namespace
			release.Service.Namespace = utils.NewStringValue(str)
			resp := discoverSuit.server.ReleaseCircuitBreakers(discoverSuit.defaultCtx, []*api.ConfigRelease{release})
			release.Service.Namespace = oldNamespace
			if resp.Code.Value != api.InvalidNamespaceName {
				t.Fatalf("%+v", resp)
			}
		})
		t.Run("????????????????????????token??????", func(t *testing.T) {
			str := genSpecialStr(2049)
			oldToken := release.Service.Token
			release.Service.Token = utils.NewStringValue(str)
			resp := discoverSuit.server.ReleaseCircuitBreakers(discoverSuit.defaultCtx, []*api.ConfigRelease{release})
			release.Service.Token = oldToken
			if resp.Code.Value != api.InvalidServiceToken {
				t.Fatalf("%+v", resp)
			}
		})
		t.Run("?????????????????????????????????", func(t *testing.T) {
			str := genSpecialStr(1025)
			oldName := release.CircuitBreaker.Name
			release.CircuitBreaker.Name = utils.NewStringValue(str)
			resp := discoverSuit.server.ReleaseCircuitBreakers(discoverSuit.defaultCtx, []*api.ConfigRelease{release})
			release.CircuitBreaker.Name = oldName
			if resp.Code.Value != api.InvalidCircuitBreakerName {
				t.Fatalf("%+v", resp)
			}
		})
		t.Run("??????????????????????????????????????????", func(t *testing.T) {
			str := genSpecialStr(1025)
			oldNamespace := release.CircuitBreaker.Namespace
			release.CircuitBreaker.Namespace = utils.NewStringValue(str)
			resp := discoverSuit.server.ReleaseCircuitBreakers(discoverSuit.defaultCtx, []*api.ConfigRelease{release})
			release.CircuitBreaker.Namespace = oldNamespace
			if resp.Code.Value != api.InvalidCircuitBreakerNamespace {
				t.Fatalf("%+v", resp)
			}
		})
		t.Run("????????????????????????version??????", func(t *testing.T) {
			str := genSpecialStr(1025)
			oldVersion := release.CircuitBreaker.Version
			release.CircuitBreaker.Version = utils.NewStringValue(str)
			resp := discoverSuit.server.ReleaseCircuitBreakers(discoverSuit.defaultCtx, []*api.ConfigRelease{release})
			release.CircuitBreaker.Version = oldVersion
			if resp.Code.Value != api.InvalidCircuitBreakerVersion {
				t.Fatalf("%+v", resp)
			}
		})
	})

}
