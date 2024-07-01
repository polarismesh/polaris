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

package batch

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseBatchConfig(t *testing.T) {
	type args struct {
		opt map[string]interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    *Config
		wantErr bool
	}{
		{
			name: "opt_nil",
			args: args{
				opt: nil,
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "register",
			args: args{
				opt: map[string]interface{}{
					"register": map[string]interface{}{
						"queueSize":     0,
						"maxBatchCount": 0,
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "deregister",
			args: args{
				opt: map[string]interface{}{
					"deregister": map[string]interface{}{
						"queueSize":     0,
						"maxBatchCount": 0,
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "clientRegister",
			args: args{
				opt: map[string]interface{}{
					"clientRegister": map[string]interface{}{
						"queueSize":     0,
						"maxBatchCount": 0,
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "clientDeregister",
			args: args{
				opt: map[string]interface{}{
					"clientDeregister": map[string]interface{}{
						"queueSize":     0,
						"maxBatchCount": 0,
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "heartbeat",
			args: args{
				opt: map[string]interface{}{
					"heartbeat": map[string]interface{}{
						"queueSize":     0,
						"maxBatchCount": 0,
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseBatchConfig(tt.args.opt)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseBatchConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseBatchConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckCtrlConfig(t *testing.T) {
	// 测试有效配置
	validCtrl := &CtrlConfig{
		QueueSize:     10,
		MaxBatchCount: 100,
		Concurrency:   5,
	}
	assert.True(t, checkCtrlConfig(validCtrl))

	// 测试无效配置
	invalidCtrl := &CtrlConfig{
		QueueSize:     0,
		MaxBatchCount: 0,
		Concurrency:   0,
	}
	assert.False(t, checkCtrlConfig(invalidCtrl))

	// 测试nil配置
	assert.True(t, checkCtrlConfig(nil))
}
