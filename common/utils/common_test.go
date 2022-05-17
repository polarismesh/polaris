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

package utils

import (
	"testing"

	"github.com/golang/protobuf/ptypes/wrappers"
)

// TestCheckResourceName tests the checkResourceName function
func TestCheckResourceName(t *testing.T) {
	type args struct {
		name *wrappers.StringValue
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "nil test", args: args{
			name: nil,
		}, wantErr: true},
		{name: "empty test", args: args{
			name: &wrappers.StringValue{Value: ""},
		}, wantErr: true},
		{name: "illegal treatment", args: args{
			name: &wrappers.StringValue{Value: "a-b-c-d-#"},
		}, wantErr: true},
		{name: "normal treatment", args: args{
			name: &wrappers.StringValue{Value: "a-b-c-d"},
		}, wantErr: false},
		{name: "normal treatment-backslash", args: args{
			name: &wrappers.StringValue{Value: "/a/b/c/d"},
		}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckResourceName(tt.args.name); (err != nil) != tt.wantErr {
				t.Errorf("CheckResourceName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
