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

package whitelist

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/polarismesh/polaris/plugin"
)

func Test_ipWhitelist_Name(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "get name",
			want: PluginName,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &ipWhitelist{}
			got := i.Name()
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_ipWhitelist_Initialize(t *testing.T) {
	type args struct {
		conf *plugin.ConfigEntry
	}
	tests := []struct {
		name    string
		args    args
		wantIPs map[string]bool
		wantErr error
	}{
		{
			name: "initialize success",
			args: args{
				conf: &plugin.ConfigEntry{
					Name: "whitelist",
					Option: map[string]interface{}{
						"ip": []interface{}{
							"127.0.0.1",
							"192.168.0.1",
						},
					},
				},
			},
			wantIPs: map[string]bool{
				"127.0.0.1":   true,
				"192.168.0.1": true,
			},
			wantErr: nil,
		},
		{
			name: "initialize fail",
			args: args{
				conf: &plugin.ConfigEntry{
					Name: "whitelist",
					Option: map[string]interface{}{
						"ip": "127.0.0.1",
					},
				},
			},
			wantIPs: map[string]bool{},
			wantErr: errors.New("whitelist plugin initialize error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &ipWhitelist{}
			err := i.Initialize(tt.args.conf)
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.wantIPs, i.ips)
		})
	}
}

func Test_ipWhitelist_Destroy(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "destroy",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &ipWhitelist{}
			err := i.Destroy()
			assert.Nil(t, err)
		})
	}
}

func Test_ipWhitelist_Contain(t *testing.T) {
	type args struct {
		elem interface{}
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "contain true",
			args: args{
				elem: "127.0.0.1",
			},
			want: true,
		},
		{
			name: "contain false",
			args: args{
				elem: "0.0.0.0",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &ipWhitelist{
				ips: map[string]bool{
					"127.0.0.1":   true,
					"192.168.0.1": true,
				},
			}
			got := i.Contain(tt.args.elem)
			assert.Equal(t, tt.want, got)
		})
	}
}
