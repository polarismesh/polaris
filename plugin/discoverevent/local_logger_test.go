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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocalLoggerConfig_Validate(t *testing.T) {
	type fields struct {
		OutputPath         string
		RotationMaxSize    int
		RotationMaxAge     int
		RotationMaxBackups int
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr error
	}{
		{
			name: "OutputPath is empty",
			fields: fields{
				OutputPath: "",
			},
			wantErr: errors.New("OutputPath is empty"),
		},
		{
			name: "RotationMaxSize is <= 0",
			fields: fields{
				OutputPath:      "./discover-event",
				RotationMaxSize: 0,
			},
			wantErr: errors.New("RotationMaxSize is <= 0"),
		},
		{
			name: "RotationMaxAge is <= 0",
			fields: fields{
				OutputPath:      "./discover-event",
				RotationMaxSize: 50,
				RotationMaxAge:  0,
			},
			wantErr: errors.New("RotationMaxAge is <= 0"),
		},
		{
			name: "RotationMaxBackups is <= 0",
			fields: fields{
				OutputPath:         "./discover-event",
				RotationMaxSize:    50,
				RotationMaxAge:     7,
				RotationMaxBackups: 0,
			},
			wantErr: errors.New("RotationMaxBackups is <= 0"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &LocalLoggerConfig{
				OutputPath:         tt.fields.OutputPath,
				RotationMaxSize:    tt.fields.RotationMaxSize,
				RotationMaxAge:     tt.fields.RotationMaxAge,
				RotationMaxBackups: tt.fields.RotationMaxBackups,
			}
			err := c.Validate()
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func Test_defaultLocalLoggerConfig(t *testing.T) {
	tests := []struct {
		name string
		want *LocalLoggerConfig
	}{
		{
			name: "default local logger config",
			want: &LocalLoggerConfig{
				OutputPath:         "./discover-event",
				RotationMaxSize:    50,
				RotationMaxAge:     7,
				RotationMaxBackups: 100,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := defaultLocalLoggerConfig()
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_newLogger(t *testing.T) {
	type args struct {
		file       string
		maxSizeMB  int
		maxBackups int
		maxAge     int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "new zap logger",
			args: args{
				file:       "./discover-event/discoverevent.log",
				maxSizeMB:  50,
				maxAge:     7,
				maxBackups: 100,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newLogger(tt.args.file, tt.args.maxSizeMB, tt.args.maxBackups, tt.args.maxAge)
			assert.NotNil(t, got)
		})
	}
}

func Test_newLocalLogger(t *testing.T) {
	type args struct {
		opt map[string]interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    Logger
		wantErr bool
	}{
		{
			name: "new local logger",
			args: args{
				opt: map[string]interface{}{
					"outputPath":         "./discover-event",
					"rotationMaxAge":     8,
					"rotationMaxBackups": 100,
					"rotationMaxSize":    500,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newLocalLogger(tt.args.opt)
			assert.NotNil(t, got)
			assert.Nil(t, err)
		})
	}
}
