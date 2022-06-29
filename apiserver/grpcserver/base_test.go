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

	"github.com/polarismesh/polaris-server/common/utils"
	"google.golang.org/grpc/metadata"
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
				externalHeaders: []string{"internal-key-1"},
			},
			want: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, utils.StringContext("request-id"), "")
				ctx = context.WithValue(ctx, utils.StringContext("client-ip"), "")
				ctx = context.WithValue(ctx, utils.ContextClientAddress, "")
				ctx = context.WithValue(ctx, utils.StringContext("user-agent"), "")
				ctx = context.WithValue(ctx, utils.StringContext("internal-key-1"), "internal-value-1")

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
				externalHeaders: []string{"internal-key-1"},
			},
			want: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, utils.StringContext("request-id"), "request-id")
				ctx = context.WithValue(ctx, utils.StringContext("client-ip"), "")
				ctx = context.WithValue(ctx, utils.ContextClientAddress, "")
				ctx = context.WithValue(ctx, utils.StringContext("user-agent"), "user-agent")
				ctx = context.WithValue(ctx, utils.StringContext("internal-key-1"), "internal-value-1")

				return ctx
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConvertContext(tt.args.ctx, tt.args.externalHeaders...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertContext() = %v, \n want %v", got, tt.want)
			}
		})
	}
}
