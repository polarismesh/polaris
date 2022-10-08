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

package discover

import (
	"fmt"
	"reflect"
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/polarismesh/polaris/apiserver/grpcserver"
	api "github.com/polarismesh/polaris/common/api/v1"
)

func Test_discoverCacheConvert(t *testing.T) {
	type args struct {
		m interface{}
	}
	tests := []struct {
		name string
		args args
		want *grpcserver.CacheObject
	}{
		{
			name: "DiscoverResponse_INSTANCE",
			args: args{
				m: &api.DiscoverResponse{
					Code: &wrapperspb.UInt32Value{
						Value: api.ExecuteSuccess,
					},
					Type: api.DiscoverResponse_INSTANCE,
					Service: &api.Service{
						Name: &wrapperspb.StringValue{
							Value: "test",
						},
						Namespace: &wrapperspb.StringValue{
							Value: "test",
						},
						Revision: &wrapperspb.StringValue{
							Value: "test",
						},
					},
					Instances: []*api.Instance{},
				},
			},
			want: &grpcserver.CacheObject{
				OriginVal: &api.DiscoverResponse{
					Code: &wrapperspb.UInt32Value{
						Value: api.ExecuteSuccess,
					},
					Type: api.DiscoverResponse_INSTANCE,
					Service: &api.Service{
						Name: &wrapperspb.StringValue{
							Value: "test",
						},
						Namespace: &wrapperspb.StringValue{
							Value: "test",
						},
						Revision: &wrapperspb.StringValue{
							Value: "test",
						},
					},
					Instances: []*api.Instance{},
				},
				CacheType: api.DiscoverResponse_INSTANCE.String(),
				Key:       fmt.Sprintf("%s-%s-%s", "test", "test", "test"),
			},
		},
		{
			name: "DiscoverResponse_SERVICES",
			args: args{
				m: &api.DiscoverResponse{
					Code: &wrapperspb.UInt32Value{
						Value: api.ExecuteSuccess,
					},
					Type: api.DiscoverResponse_SERVICES,
					Service: &api.Service{
						Name: &wrapperspb.StringValue{
							Value: "",
						},
						Namespace: &wrapperspb.StringValue{
							Value: "test",
						},
						Revision: &wrapperspb.StringValue{
							Value: "",
						},
					},
				},
			},
			want: nil,
		},
		{
			name: "DiscoverResponse_RATE_LIMIT",
			args: args{
				m: &api.DiscoverResponse{
					Code: &wrapperspb.UInt32Value{
						Value: api.ExecuteSuccess,
					},
					Type: api.DiscoverResponse_RATE_LIMIT,
					Service: &api.Service{
						Name: &wrapperspb.StringValue{
							Value: "test",
						},
						Namespace: &wrapperspb.StringValue{
							Value: "test",
						},
						Revision: &wrapperspb.StringValue{
							Value: "test",
						},
					},
				},
			},
			want: &grpcserver.CacheObject{
				OriginVal: &api.DiscoverResponse{
					Code: &wrapperspb.UInt32Value{
						Value: api.ExecuteSuccess,
					},
					Type: api.DiscoverResponse_RATE_LIMIT,
					Service: &api.Service{
						Name: &wrapperspb.StringValue{
							Value: "test",
						},
						Namespace: &wrapperspb.StringValue{
							Value: "test",
						},
						Revision: &wrapperspb.StringValue{
							Value: "test",
						},
					},
				},
				CacheType: api.DiscoverResponse_RATE_LIMIT.String(),
				Key:       fmt.Sprintf("%s-%s-%s", "test", "test", "test"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := discoverCacheConvert(tt.args.m); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("discoverCacheConvert() = %v, want %v", got, tt.want)
			}
		})
	}
}
