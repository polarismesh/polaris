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

package discoverevent

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/polarismesh/polaris-server/common/model"
	"github.com/stretchr/testify/assert"
)

func TestLokiLoggerConfig_UnmarshalJSON(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "unmarshal json",
			args: args{
				data: []byte(`{
					"pushURL": "http://127.0.0.1:3100/loki/api/v1/push",
					"tenantID": "test",
					"labels": {
					"env": "test"
					},
					"timeout": "5s"
					}`),
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &LokiLoggerConfig{}
			err := c.UnmarshalJSON(tt.args.data)
			assert.Nil(t, err)
			assert.Equal(t, "http://127.0.0.1:3100/loki/api/v1/push", c.PushURL)
			assert.Equal(t, "test", c.TenantID)
			assert.Equal(t, map[string]string{"env": "test"}, c.Labels)
			assert.Equal(t, 5*time.Second, c.Timeout)
		})
	}
}

func TestLokiLoggerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		pushURL string
		wantErr error
	}{
		{
			name:    "validate push url error",
			pushURL: "",
			wantErr: errors.New("PushURL is empty"),
		},
		{
			name:    "validate push url success",
			pushURL: "http://127.0.0.1:3100/loki/api/v1/push",
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &LokiLoggerConfig{
				PushURL: tt.pushURL,
			}
			err := c.Validate()
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func Test_genLabels(t *testing.T) {
	type args struct {
		labels map[string]string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "empty labels",
			args: args{
				labels: map[string]string{},
			},
			want: "{}",
		},
		{
			name: "one labels",
			args: args{
				labels: map[string]string{
					"source": "discoverEvent",
				},
			},
			want: "{source=\"discoverEvent\"}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := genLabels(tt.args.labels); got != tt.want {
				t.Errorf("genLabels() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newLokiLogger(t *testing.T) {
	type args struct {
		opt map[string]interface{}
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "new loki logger",
			args: args{
				opt: map[string]interface{}{
					"labels":   map[string]string{"app": "polaris", "env": "test"},
					"pushURL":  "http://localhost:3100/loki/api/v1/push",
					"tenantID": "test",
					"timeout":  "5s",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newLokiLogger(tt.args.opt)
			assert.NotNil(t, got)
			assert.Nil(t, err)
		})
	}
}

func TestLokiLogger_Log(t *testing.T) {
	type fields struct {
		pushURL  string
		tenantID string
		labels   map[string]string
		client   http.Client
		timeout  time.Duration
	}
	type args struct {
		events []model.DiscoverEvent
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &LokiLogger{
				pushURL:  tt.fields.pushURL,
				tenantID: tt.fields.tenantID,
				labels:   tt.fields.labels,
				client:   tt.fields.client,
				timeout:  tt.fields.timeout,
			}
			l.Log(tt.args.events)
		})
	}
}

func TestLokiLogger_send(t *testing.T) {
	type fields struct {
		pushURL  string
		tenantID string
		labels   map[string]string
		client   http.Client
		timeout  time.Duration
	}
	type args struct {
		ctx     context.Context
		reqBody []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *http.Response
		want1   []byte
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &LokiLogger{
				pushURL:  tt.fields.pushURL,
				tenantID: tt.fields.tenantID,
				labels:   tt.fields.labels,
				client:   tt.fields.client,
				timeout:  tt.fields.timeout,
			}
			got, got1, err := l.send(tt.args.ctx, tt.args.reqBody)
			if (err != nil) != tt.wantErr {
				t.Errorf("LokiLogger.send() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LokiLogger.send() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("LokiLogger.send() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
