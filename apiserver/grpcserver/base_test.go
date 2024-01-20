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

package grpcserver

import (
	"context"
	"reflect"
	"testing"

	"google.golang.org/grpc/metadata"

	"github.com/polarismesh/polaris/apiserver"
	grpchelp "github.com/polarismesh/polaris/apiserver/grpcserver/utils"
	"github.com/polarismesh/polaris/common/utils"
)

func mockGrpcContext(testVal map[string]string) context.Context {

	md := make(metadata.MD)

	for k := range testVal {
		md[k] = []string{testVal[k]}
	}

	ctx := metadata.NewIncomingContext(context.Background(), md)

	return ctx
}

func TestConvertContext(t *testing.T) {
	type args struct {
		ctx             context.Context
		externalHeaders []string
	}
	tests := []struct {
		name string
		args args
		want context.Context
	}{
		{
			name: "",
			args: args{
				ctx: mockGrpcContext(map[string]string{
					"internal-key-1": "internal-value-1",
				}),
			},
			want: func() context.Context {
				ctx := context.Background()

				md := make(metadata.MD)

				testVal := map[string]string{
					"internal-key-1": "internal-value-1",
				}

				for k := range testVal {
					md[k] = []string{testVal[k]}
				}

				ctx = context.WithValue(ctx, utils.ContextGrpcHeader, md)
				ctx = context.WithValue(ctx, utils.StringContext("request-id"), "")
				ctx = context.WithValue(ctx, utils.StringContext("client-ip"), "")
				ctx = context.WithValue(ctx, utils.ContextClientAddress, "")
				ctx = context.WithValue(ctx, utils.StringContext("user-agent"), "")

				return ctx
			}(),
		},
		{
			name: "",
			args: args{
				ctx: mockGrpcContext(map[string]string{
					"internal-key-1": "internal-value-1",
					"request-id":     "request-id",
					"user-agent":     "user-agent",
				}),
			},
			want: func() context.Context {

				md := make(metadata.MD)

				testVal := map[string]string{
					"internal-key-1": "internal-value-1",
					"request-id":     "request-id",
					"user-agent":     "user-agent",
				}
				for k := range testVal {
					md[k] = []string{testVal[k]}
				}

				ctx := context.Background()
				ctx = context.WithValue(ctx, utils.ContextGrpcHeader, md)
				ctx = context.WithValue(ctx, utils.StringContext("request-id"), "request-id")
				ctx = context.WithValue(ctx, utils.StringContext("client-ip"), "")
				ctx = context.WithValue(ctx, utils.ContextClientAddress, "")
				ctx = context.WithValue(ctx, utils.StringContext("user-agent"), "user-agent")

				return ctx
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.ConvertGRPCContext(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertContext() = %v, \n want %v", got, tt.want)
			}
		})
	}
}

func TestGetClientOpenMethod(t *testing.T) {
	type args struct {
		include  []string
		protocol string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]bool
		wantErr bool
	}{
		{
			name: "case=1",
			args: args{
				include: []string{
					apiserver.RegisterAccess,
				},
				protocol: "grpc",
			},
			want: map[string]bool{
				"/v1.PolarisGRPC/RegisterInstance":   true,
				"/v1.PolarisGRPC/DeregisterInstance": true,
			},
			wantErr: false,
		},
		{
			name: "case=2",
			args: args{
				include: []string{
					apiserver.DiscoverAccess,
				},
				protocol: "grpc",
			},
			want: map[string]bool{
				"/v1.PolarisGRPC/Discover":                             true,
				"/v1.PolarisGRPC/ReportClient":                         true,
				"/v1.PolarisServiceContractGRPC/ReportServiceContract": true,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := grpchelp.GetClientOpenMethod(tt.args.include, tt.args.protocol)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetClientOpenMethod() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetClientOpenMethod() = %v, want %v", got, tt.want)
			}
		})
	}
}
