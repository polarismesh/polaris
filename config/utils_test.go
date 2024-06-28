/*
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

	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/assert"
)

func TestCheckFileName(t *testing.T) {
	w := &wrappers.StringValue{Value: "123abc.test.log"}
	err := CheckFileName(w)
	assert.Equal(t, err, nil)
}

func TestCheckContentLength(t *testing.T) {
	type args struct {
		content string
		max     int
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "01",
			args: args{
				content: "123",
				max:     10,
			},
			wantErr: false,
		},
		{
			name: "02",
			args: args{
				content: "134234123412312323",
				max:     10,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckContentLength(tt.args.content, tt.args.max); (err != nil) != tt.wantErr {
				t.Errorf("CheckContentLength() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
