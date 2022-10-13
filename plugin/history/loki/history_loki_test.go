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

	"github.com/stretchr/testify/assert"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/plugin"
)

func TestHistoryLoki_Name(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "get name",
			want: "HistoryLoki",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &HistoryLoki{}
			got := h.Name()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHistoryLoki_Initialize(t *testing.T) {
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
					Name: "HistoryLoki",
					Option: map[string]interface{}{
						"queueSize": 1024,
						"pushURL":   "http://localhost:3100/loki/api/v1/push",
						"tenantID":  "test",
						"lables": map[string]string{
							"env": "test",
							"app": "polaris",
						},
					},
				},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &HistoryLoki{}
			err := h.Initialize(tt.args.conf)
			assert.Equal(t, tt.wantErr, err)
			err = h.Destroy()
			assert.Nil(t, err)
		})
	}
}

func TestHistoryLoki_Destroy(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "plugin destroy",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &HistoryLoki{
				stopCh: make(chan struct{}, 1),
			}
			err := d.Destroy()
			assert.Nil(t, err)
		})
	}
}

func TestHistoryLoki_Record(t *testing.T) {
	type args struct {
		entry *model.RecordEntry
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "record log",
			args: args{
				entry: &model.RecordEntry{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			el := &HistoryLoki{
				entryCh: make(chan *model.RecordEntry, 10),
			}
			el.Record(tt.args.entry)
			assert.Equal(t, 1, len(el.entryCh))
		})
	}
}
