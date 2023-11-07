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

package model

import "testing"

func TestReplaceNacosService(t *testing.T) {
	type args struct {
		service string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "01",
			args: args{
				service: "DEFAULT_GROUP@@service-a",
			},
			want: "service-a",
		},
		{
			name: "02",
			args: args{
				service: "DEFAULT_GROUP11@@service-a",
			},
			want: "DEFAULT_GROUP11__service-a",
		},
		{
			name: "03",
			args: args{
				service: "service-a",
			},
			want: "service-a",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ReplaceNacosService(tt.args.service); got != tt.want {
				t.Errorf("ReplaceNacosService() = %v, want %v", got, tt.want)
			}
		})
	}
}
