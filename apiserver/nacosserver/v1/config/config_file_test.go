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

package config

import (
	"testing"

	"github.com/polarismesh/polaris/apiserver/nacosserver/model"
)

func Test_md5ResultString(t *testing.T) {
	type args struct {
		items []*model.ConfigListenItem
	}
	tests := []struct {
		name string
		args args
		want string
	}{{
		name: "case-0",
		args: args{
			items: []*model.ConfigListenItem{
				{
					Tenant: "test2",
					Group:  "test1",
					DataId: "test0",
				},
			},
		},
		want: "test0%02test1%02test2%01",
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := md5ResultString(tt.args.items); got != tt.want {
				t.Errorf("md5ResultString() = %v, want %v", got, tt.want)
			}
		})
	}
}
