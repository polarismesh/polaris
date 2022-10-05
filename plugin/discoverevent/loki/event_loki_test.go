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

package loki

import (
	"testing"

	"github.com/polarismesh/polaris-server/plugin"
	"github.com/stretchr/testify/assert"
)

func Test_discoverEventLoki_Name(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "get name",
			want: "discoverEventLoki",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &discoverEventLoki{}
			got := d.Name()
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_discoverEventLoki_Initialize(t *testing.T) {
	type args struct {
		conf *plugin.ConfigEntry
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "loki logger config",
			args: args{
				conf: &plugin.ConfigEntry{
					Name: "discoverEvent",
					Option: map[string]interface{}{
						"queueSize": 1024,
						"pushURL":   "http://localhost:3100/loki/api/v1/push",
					},
				},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &discoverEventLoki{}
			err := d.Initialize(tt.args.conf)
			assert.Equal(t, tt.wantErr, err)
			err = d.Destroy()
			assert.Nil(t, err)
		})
	}
}

func Test_discoverEventLoki_Destroy(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "plugin destroy",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &discoverEventLoki{
				stopCh: make(chan struct{}),
			}
			err := d.Destroy()
			assert.Nil(t, err)
		})
	}
}
