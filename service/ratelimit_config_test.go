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

func TestServer_CreateRateLimitJson(t *testing.T) {
	rule := &api.Rule{
		Namespace: &wrappers.StringValue{Value: "Test"},
		Service:   &wrappers.StringValue{Value: "TestService1"},
		Resource:  api.Rule_QPS,
		Type:      api.Rule_LOCAL,
		Method: &api.MatchString{
			Type:  0,
			Value: &wrappers.StringValue{Value: "/info"},
		},
		Labels: map[string]*api.MatchString{
			"uin": &api.MatchString{
				Type:  0,
				Value: &wrappers.StringValue{Value: "109870111"},
			},
		},
		AmountMode: api.Rule_GLOBAL_TOTAL,
		Amounts: []*api.Amount{
			{
				MaxAmount: &wrappers.UInt32Value{Value: 1000},
				ValidDuration: &duration.Duration{
					Seconds: 1,
				},
			},
		},
		Action:   &wrappers.StringValue{Value: "reject"},
		Failover: api.Rule_FAILOVER_LOCAL,
		Disable:  &wrappers.BoolValue{Value: false},
	}
	marshaler := &jsonpb.Marshaler{}
	ruleStr, err := marshaler.MarshalToString(rule)
	if nil != err {
		t.Fatal(err)
	}
	fmt.Printf(ruleStr)
}
