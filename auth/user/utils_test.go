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

package defaultuser_test

import (
	"testing"

	"github.com/golang/protobuf/ptypes/wrappers"

	defaultuser "github.com/polarismesh/polaris/auth/user"
	"github.com/polarismesh/polaris/common/utils"
)

func Test_checkPassword(t *testing.T) {
	type args struct {
		password *wrappers.StringValue
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			args: args{
				password: utils.NewStringValue("1234"),
			},
			wantErr: true,
		},
		{
			args: args{
				password: utils.NewStringValue("123456"),
			},
			wantErr: false,
		},
		{
			args: args{
				password: utils.NewStringValue("abc45"),
			},
			wantErr: true,
		},
		{
			args: args{
				password: utils.NewStringValue("Aabc456"),
			},
			wantErr: false,
		},
		{
			args: args{
				password: utils.NewStringValue("Aa@@bc456"),
			},
			wantErr: false,
		},
		{
			args: args{
				password: utils.NewStringValue("Aa@@bc45~6"),
			},
			wantErr: false,
		},
		{
			args: args{
				password: utils.NewStringValue("abc456abc4565612"),
			},
			wantErr: false,
		},
		{
			args: args{
				password: utils.NewStringValue("abc456abc456abc456"),
			},
			wantErr: true,
		},
		{
			args: args{
				password: utils.NewStringValue("123456123456123456"),
			},
			wantErr: true,
		},
		{
			args: args{
				password: utils.NewStringValue("abc456abc456abc45612"),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := defaultuser.CheckPassword(tt.args.password); (err != nil) != tt.wantErr {
				t.Errorf("checkPassword() error = %v, wantErr %v, args = %#v", err, tt.wantErr, tt.args.password.GetValue())
			}
		})
	}
}

func Test_checkName(t *testing.T) {
	type args struct {
		name *wrappers.StringValue
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			args: args{
				name: utils.NewStringValue("123123"),
			},
			wantErr: false,
		},
		{
			args: args{
				name: utils.NewStringValue("测试鉴权策略-1"),
			},
			wantErr: false,
		},
		{
			args: args{
				name: utils.NewStringValue("测试鉴-权策略_1."),
			},
			wantErr: false,
		},
		{
			args: args{
				name: utils.NewStringValue("qweqwe-权qweqw策略_1"),
			},
			wantErr: false,
		},
		{
			args: args{
				name: utils.NewStringValue("测试鉴权策略&1"),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := defaultuser.CheckName(tt.args.name); (err != nil) != tt.wantErr {
				t.Errorf("checkName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
