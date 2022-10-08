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

package xdsserverv3

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_extensions_common_ratelimit_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/common/ratelimit/v3"
	lrl "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/local_ratelimit/v3"
	envoy_type_v3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/duration"
	_struct "github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/wrappers"
	"go.uber.org/atomic"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
)

func generateRateLimitString(ruleType api.Rule_Type) (string, string, map[string]*anypb.Any) {
	rule := &api.Rule{
		Namespace: &wrappers.StringValue{Value: "Test"},
		Service:   &wrappers.StringValue{Value: "TestService1"},
		Resource:  api.Rule_QPS,
		Type:      ruleType,
		Method: &api.MatchString{
			Type:  0,
			Value: &wrappers.StringValue{Value: "/info"},
		},
		Labels: map[string]*api.MatchString{
			"uin": {
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
	// 期待的结果
	expectRes := make(map[string]*anypb.Any)
	expectStruct := lrl.LocalRateLimit{
		StatPrefix: "http_local_rate_limiter",
		FilterEnabled: &core.RuntimeFractionalPercent{
			RuntimeKey: "local_rate_limit_enabled",
			DefaultValue: &envoy_type_v3.FractionalPercent{
				Numerator:   uint32(100),
				Denominator: envoy_type_v3.FractionalPercent_HUNDRED,
			},
		},
		FilterEnforced: &core.RuntimeFractionalPercent{
			RuntimeKey: "local_rate_limit_enforced",
			DefaultValue: &envoy_type_v3.FractionalPercent{
				Numerator:   uint32(100),
				Denominator: envoy_type_v3.FractionalPercent_HUNDRED,
			},
		},
	}
	if rule.AmountMode == api.Rule_GLOBAL_TOTAL {
		expectStruct.LocalRateLimitPerDownstreamConnection = true
	}
	expectStruct.Descriptors = []*envoy_extensions_common_ratelimit_v3.LocalRateLimitDescriptor{
		{
			Entries: []*envoy_extensions_common_ratelimit_v3.RateLimitDescriptor_Entry{
				{
					Key:   "uin",
					Value: "109870111",
				},
			},
			TokenBucket: &envoy_type_v3.TokenBucket{
				MaxTokens:    1000,
				FillInterval: &duration.Duration{Seconds: 1},
			},
		},
	}
	pbst, err := ptypes.MarshalAny(&expectStruct)
	if err != nil {
		panic(err)
	}
	expectRes["envoy.filters.http.local_ratelimit"] = pbst

	// 测试用限流字符串
	labelStr, _ := json.Marshal(rule.Labels)
	rule.Labels = nil
	ruleStr, _ := json.Marshal(rule)
	if ruleType == api.Rule_GLOBAL {
		expectRes = nil
	}
	return string(ruleStr), string(labelStr), expectRes
}

func generateGlobalRateLimitRule() ([]*model.RateLimit, map[string]*anypb.Any) {
	ruleStr, labelStr, expectRes := generateRateLimitString(api.Rule_GLOBAL)
	var rateLimits []*model.RateLimit
	rateLimits = append(rateLimits, &model.RateLimit{
		ID:         "ratelimit-1",
		ServiceID:  "service-1",
		Labels:     labelStr,
		Rule:       ruleStr,
		Revision:   "revision-1",
		Valid:      false,
		CreateTime: time.Now(),
		ModifyTime: time.Now(),
	})
	return rateLimits, expectRes
}

func generateLocalRateLimitRule() ([]*model.RateLimit, map[string]*anypb.Any) {
	ruleStr, labelStr, expectRes := generateRateLimitString(api.Rule_LOCAL)
	var rateLimits []*model.RateLimit
	rateLimits = append(rateLimits, &model.RateLimit{
		ID:         "ratelimit-2",
		ServiceID:  "service-2",
		Labels:     labelStr,
		Rule:       ruleStr,
		Revision:   "revision-2",
		Valid:      false,
		CreateTime: time.Now(),
		ModifyTime: time.Now(),
	})
	return rateLimits, expectRes
}

func Test_makeLocalRateLimit(t *testing.T) {
	localRateLimitStr, want1 := generateLocalRateLimitRule()
	globalRateLimitStr, want2 := generateGlobalRateLimitRule()
	type args struct {
		conf []*model.RateLimit
	}
	tests := []struct {
		name string
		args args
		want map[string]*anypb.Any
	}{
		{
			"make local rate limit for local rate limit config",
			args{
				localRateLimitStr,
			},
			want1,
		},
		{
			"make local rate limit for global rate limit config",
			args{
				globalRateLimitStr,
			},
			want2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := makeLocalRateLimit(tt.args.conf); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("makeLocalRateLimit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseNodeID(t *testing.T) {
	testTable := []struct {
		NodeID string

		Namespace string
		UUID      string
		HostIP    string
	}{
		{
			NodeID:    "default/9b9f5630-81a1-47cd-a558-036eb616dc71~172.17.1.1",
			Namespace: "default",
			UUID:      "9b9f5630-81a1-47cd-a558-036eb616dc71",
			HostIP:    "172.17.1.1",
		},
		{
			NodeID:    "namespace/9b9f5630-81a1-47cd-a558-036eb616dc71~1.1.1.1",
			Namespace: "namespace",
			UUID:      "9b9f5630-81a1-47cd-a558-036eb616dc71",
			HostIP:    "1.1.1.1",
		},
		{
			NodeID:    "default/67c745dd-35b3-40fe-8a9d-64ea6ec19fb2~10.244.0.229",
			Namespace: "default",
			HostIP:    "10.244.0.229",
			UUID:      "67c745dd-35b3-40fe-8a9d-64ea6ec19fb2",
		},
		// bad case
		{
			NodeID:    "namespace",
			Namespace: "",
			UUID:      "",
			HostIP:    "",
		},
	}
	for _, item := range testTable {
		ns, id, hostip := parseNodeID(item.NodeID)
		if ns != item.Namespace || id != item.UUID || hostip != item.HostIP {
			t.Fatalf("parse node id [%s] expected ['%s' '%s' '%s'] got : ['%s' '%s' '%s']",
				item.NodeID,
				item.Namespace, item.UUID, item.HostIP,
				ns, id, hostip,
			)
		}
	}
}

func TestNodeHashID(t *testing.T) {
	testTable := []struct {
		Node     *core.Node
		TargetID string
	}{
		{
			Node: &core.Node{
				Id: "default/9b9f5630-81a1-47cd-a558-036eb616dc71~172.17.1.1",
				Metadata: &_struct.Struct{
					Fields: map[string]*structpb.Value{
						TLSModeTag: &_struct.Value{
							Kind: &_struct.Value_StringValue{
								StringValue: TLSModeStrict,
							},
						},
					},
				},
			},
			TargetID: "default/" + TLSModeStrict,
		},
		{
			Node: &core.Node{
				Id: "polaris/9b9f5630-81a1-47cd-a558-036eb616dc71~172.17.1.1",
				Metadata: &_struct.Struct{
					Fields: map[string]*structpb.Value{
						TLSModeTag: &_struct.Value{
							Kind: &_struct.Value_StringValue{
								StringValue: TLSModePermissive,
							},
						},
					},
				},
			},
			TargetID: "polaris/" + TLSModePermissive,
		},
		{
			Node: &core.Node{
				Id: "default/9b9f5630-81a1-47cd-a558-036eb616dc71~172.17.1.1",
				Metadata: &_struct.Struct{
					Fields: map[string]*structpb.Value{
						TLSModeTag: &_struct.Value{
							Kind: &_struct.Value_StringValue{
								StringValue: TLSModeNone,
							},
						},
					},
				},
			},
			TargetID: "default",
		},
		// bad case: wrong tls mode
		{
			Node: &core.Node{
				Id: "default/9b9f5630-81a1-47cd-a558-036eb616dc71~172.17.1.1",
				Metadata: &_struct.Struct{
					Fields: map[string]*structpb.Value{
						TLSModeTag: &_struct.Value{
							Kind: &_struct.Value_StringValue{
								StringValue: "abc",
							},
						},
					},
				},
			},
			TargetID: "default",
		},
		// no node metadata
		{
			Node: &core.Node{
				Id: "default/9b9f5630-81a1-47cd-a558-036eb616dc71~172.17.1.1",
			},
			TargetID: "default",
		},
		// metadata does not contain tls mode kv
		{
			Node: &core.Node{
				Id: "default/9b9f5630-81a1-47cd-a558-036eb616dc71~172.17.1.1",
				Metadata: &_struct.Struct{
					Fields: map[string]*structpb.Value{
						"hello": &_struct.Value{
							Kind: &_struct.Value_StringValue{
								StringValue: "abc",
							},
						},
					},
				},
			},
			TargetID: "default",
		},
	}
	for i, item := range testTable {
		id := PolarisNodeHash{}.ID(item.Node)
		if id != item.TargetID {
			t.Fatalf("test case [%d] failed: expect ID %s, got ID %s",
				i, item.TargetID, id)
		}
	}
}

//go:embed testdata/data.json
var testServicesData []byte

//go:embed testdata/dump.yaml
var noInboundDump []byte

//go:embed testdata/permissive.dump.yaml
var permissiveDump []byte

//go:embed testdata/strict.dump.yaml
var strictDump []byte

func TestSnapshot(t *testing.T) {
	sis := map[string][]*ServiceInfo{}
	json.Unmarshal(testServicesData, &sis)

	x := XDSServer{
		CircuitBreakerConfigGetter: func(id string) *model.ServiceWithCircuitBreaker {
			return nil
		},
		RatelimitConfigGetter: func(serviceID string) []*model.RateLimit { return nil },
		versionNum:            atomic.NewUint64(1),
		cache:                 cache.NewSnapshotCache(true, cache.IDHash{}, nil),
	}
	x.pushRegistryInfoToXDSCache(sis)

	snapshot, _ := x.cache.GetSnapshot("default")
	dumpYaml := dumpSnapShot(snapshot)
	if !bytes.Equal(noInboundDump, dumpYaml) {
		t.Fatal(string(dumpYaml))
	}

	snapshot, _ = x.cache.GetSnapshot("default/permissive")
	dumpYaml = dumpSnapShot(snapshot)
	if !bytes.Equal(permissiveDump, dumpYaml) {
		t.Fatal(string(dumpYaml))
	}

	snapshot, _ = x.cache.GetSnapshot("default/strict")
	dumpYaml = dumpSnapShot(snapshot)
	if !bytes.Equal(strictDump, dumpYaml) {
		t.Fatal(string(dumpYaml))
	}
}
