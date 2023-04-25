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

package apiserver

import (
	"reflect"
	"testing"
)

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
					RegisterAccess,
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
			name: "case=1",
			args: args{
				include: []string{
					DiscoverAccess,
				},
				protocol: "grpc",
			},
			want: map[string]bool{
				"/v1.PolarisGRPC/Discover":     true,
				"/v1.PolarisGRPC/ReportClient": true,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetClientOpenMethod(tt.args.include, tt.args.protocol)
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
