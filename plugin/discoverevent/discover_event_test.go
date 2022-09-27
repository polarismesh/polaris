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
	"sync"
	"testing"
	"time"

	json "github.com/json-iterator/go"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/plugin"
	"github.com/stretchr/testify/assert"
)

func Test_discoverEvent_Initialize(t *testing.T) {
	type args struct {
		conf *plugin.ConfigEntry
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "init success",
			args: args{
				conf: &plugin.ConfigEntry{
					Name: "discoverEvent",
					Option: map[string]interface{}{
						"queueSize": 1024,
						"logger": []interface{}{
							map[interface{}]interface{}{
								"name": "local",
								"option": map[interface{}]interface{}{
									"outputPath":         "./discover-event",
									"rotationMaxAge":     8,
									"rotationMaxBackups": 100,
									"rotationMaxSize":    500,
								},
							},
							map[interface{}]interface{}{
								"name": "loki",
								"option": map[interface{}]interface{}{
									"labels": map[interface{}]interface{}{"app": "polaris", "env": "test"}, "pushURL": "http://localhost:3100/loki/api/v1/push",
									"tenantID": "test",
									"timeout":  "5s",
								},
							},
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "local logger config",
			args: args{
				conf: &plugin.ConfigEntry{
					Name: "discoverEvent",
					Option: map[string]interface{}{
						"queueSize": 1024,
						"logger": []interface{}{
							map[interface{}]interface{}{
								"name": "local",
								"option": map[interface{}]interface{}{
									"outputPath":         "./discover-event",
									"rotationMaxAge":     8,
									"rotationMaxBackups": 100,
									"rotationMaxSize":    500,
								},
							},
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "loki logger config",
			args: args{
				conf: &plugin.ConfigEntry{
					Name: "discoverEvent",
					Option: map[string]interface{}{
						"queueSize": 1024,
						"logger": []interface{}{
							map[interface{}]interface{}{
								"name": "loki",
								"option": map[interface{}]interface{}{
									"pushURL": "http://localhost:3100/loki/api/v1/push",
								},
							},
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "no logger config",
			args: args{
				conf: &plugin.ConfigEntry{
					Name: "discoverEvent",
					Option: map[string]interface{}{
						"queueSize": 1024,
					},
				},
			},
			wantErr: errors.New("LoggerConfig is empty"),
		},
		{
			name: "loki push url error",
			args: args{
				conf: &plugin.ConfigEntry{
					Name: "discoverEvent",
					Option: map[string]interface{}{
						"queueSize": 1024,
						"logger": []interface{}{
							map[interface{}]interface{}{
								"name": "loki",
								"option": map[interface{}]interface{}{
									"pushURL": "",
								},
							},
						},
					},
				},
			},
			wantErr: errors.New("PushURL is empty"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &discoverEvent{}
			err := d.Initialize(tt.args.conf)
			assert.Equal(t, err, tt.wantErr)
		})
	}
}

func Test_discoverEvent_Name(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "get name",
			want: "discoverEvent",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &discoverEvent{}
			got := d.Name()
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_discoverEvent_Destroy(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "plugin destroy",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &discoverEvent{}
			err := d.Destroy()
			assert.Nil(t, err)
		})
	}
}

func Test_discoverEvent_PublishEvent(t *testing.T) {
	type args struct {
		event model.DiscoverEvent
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "publish event",
			args: args{
				event: model.DiscoverEvent{
					Namespace:  "Polaris",
					Service:    "polaris.discover",
					Host:       "127.0.0.1",
					Port:       8091,
					EType:      model.EventInstanceOnline,
					CreateTime: time.Now(),
				},
			},
		},
	}
	d := &discoverEvent{}
	conf := &plugin.ConfigEntry{
		Name: "discoverEvent",
		Option: map[string]interface{}{
			"queueSize": 1024,
			"logger": []interface{}{
				map[interface{}]interface{}{
					"name": "local",
					"option": map[interface{}]interface{}{
						"outputPath":         "./discover-event",
						"rotationMaxAge":     8,
						"rotationMaxBackups": 100,
						"rotationMaxSize":    500,
					},
				},
				map[interface{}]interface{}{
					"name": "loki",
					"option": map[interface{}]interface{}{
						"labels": map[interface{}]interface{}{"app": "polaris", "env": "test"}, "pushURL": "http://localhost:3100/loki/api/v1/push",
						"tenantID": "test",
						"timeout":  "5s",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := d.mockInitialize(conf)
			assert.Nil(t, err)
			d.PublishEvent(tt.args.event)
		})
	}
}

// func Test_discoverEvent_Run(t *testing.T) {
// 	tests := []struct {
// 		name string
// 	}{
// 		{
// 			name: "plugin run",
// 		},
// 	}
// 	d := &discoverEvent{}
// 	conf := &plugin.ConfigEntry{
// 		Name: "discoverEvent",
// 		Option: map[string]interface{}{
// 			"queueSize": 1024,
// 			"logger": []interface{}{
// 				map[interface{}]interface{}{
// 					"name": "local",
// 					"option": map[interface{}]interface{}{
// 						"outputPath":         "./discover-event",
// 						"rotationMaxAge":     8,
// 						"rotationMaxBackups": 100,
// 						"rotationMaxSize":    500,
// 					},
// 				},
// 				map[interface{}]interface{}{
// 					"name": "loki",
// 					"option": map[interface{}]interface{}{
// 						"labels": map[interface{}]interface{}{"app": "polaris", "env": "test"}, "pushURL": "http://localhost:3100/loki/api/v1/push",
// 						"tenantID": "test",
// 						"timeout":  "5s",
// 					},
// 				},
// 			},
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			err := d.mockInitialize(conf)
// 			assert.Nil(t, err)
// 			d.Run()
// 		})
// 	}
// }

func Test_discoverEvent_switchEventBuffer(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "switch event buffer",
		},
	}
	d := &discoverEvent{}
	conf := &plugin.ConfigEntry{
		Name: "discoverEvent",
		Option: map[string]interface{}{
			"queueSize": 1024,
			"logger": []interface{}{
				map[interface{}]interface{}{
					"name": "local",
					"option": map[interface{}]interface{}{
						"outputPath":         "./discover-event",
						"rotationMaxAge":     8,
						"rotationMaxBackups": 100,
						"rotationMaxSize":    500,
					},
				},
				map[interface{}]interface{}{
					"name": "loki",
					"option": map[interface{}]interface{}{
						"labels": map[interface{}]interface{}{"app": "polaris", "env": "test"}, "pushURL": "http://localhost:3100/loki/api/v1/push",
						"tenantID": "test",
						"timeout":  "5s",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := d.mockInitialize(conf)
			assert.Nil(t, err)
			d.switchEventBuffer()
		})
	}
}

func Test_discoverEvent_newLogger(t *testing.T) {
	type args struct {
		cfg model.DiscoverEventLoggerConfig
	}
	tests := []struct {
		name    string
		args    args
		want    Logger
		wantErr error
	}{
		{
			name: "new local logger",
			args: args{
				cfg: model.DiscoverEventLoggerConfig{
					Name: "local",
					Option: map[string]interface{}{
						"outputPath":         "./discover-event",
						"rotationMaxAge":     8,
						"rotationMaxBackups": 100,
						"rotationMaxSize":    500,
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "new loki logger",
			args: args{
				cfg: model.DiscoverEventLoggerConfig{
					Name: "loki",
					Option: map[string]interface{}{
						"labels":   map[string]string{"app": "polaris", "env": "test"},
						"pushURL":  "http://localhost:3100/loki/api/v1/push",
						"tenantID": "test",
						"timeout":  "5s",
					},
				},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &discoverEvent{}
			got, err := d.newLogger(tt.args.cfg)
			assert.Equal(t, tt.wantErr, err)
			assert.NotNil(t, got)
		})
	}
}

func Test_discoverEvent_getEvents(t *testing.T) {
	tests := []struct {
		name   string
		events []model.DiscoverEvent
		want   int
	}{
		{
			name: "get buffer events",
			events: []model.DiscoverEvent{
				{
					Namespace:  "Polaris",
					Service:    "polaris.discover",
					Host:       "127.0.0.1",
					Port:       8091,
					EType:      model.EventInstanceOnline,
					CreateTime: time.Now(),
				},
				{
					Namespace:  "Polaris",
					Service:    "polaris.discover",
					Host:       "127.0.0.1",
					Port:       8091,
					EType:      model.EventInstanceOnline,
					CreateTime: time.Now(),
				},
			},
			want: 2,
		},
	}
	d := &discoverEvent{}
	conf := &plugin.ConfigEntry{
		Name: "discoverEvent",
		Option: map[string]interface{}{
			"queueSize": 1024,
			"logger": []interface{}{
				map[interface{}]interface{}{
					"name": "local",
					"option": map[interface{}]interface{}{
						"outputPath":         "./discover-event",
						"rotationMaxAge":     8,
						"rotationMaxBackups": 100,
						"rotationMaxSize":    500,
					},
				},
				map[interface{}]interface{}{
					"name": "loki",
					"option": map[interface{}]interface{}{
						"labels": map[interface{}]interface{}{"app": "polaris", "env": "test"}, "pushURL": "http://localhost:3100/loki/api/v1/push",
						"tenantID": "test",
						"timeout":  "5s",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := d.mockInitialize(conf)
			assert.Nil(t, err)
			for _, event := range tt.events {
				d.curEventBuffer.Put(event)
			}
			got := d.getEvents()
			assert.Equal(t, tt.want, len(got))
		})
	}
}

func (d *discoverEvent) mockInitialize(conf *plugin.ConfigEntry) error {
	confBytes, err := json.Marshal(conf.Option)
	if err != nil {
		return err
	}
	config := &model.DiscoverEventConfig{}
	if err := json.Unmarshal(confBytes, config); err != nil {
		return err
	}

	if err := config.Validate(); err != nil {
		return err
	}

	d.eventCh = make(chan model.DiscoverEvent, config.QueueSize)

	for _, cfg := range config.LoggerConfigs {
		logger, err := d.newLogger(cfg)
		if err != nil {
			return err
		}
		d.eventLoggers = append(d.eventLoggers, logger)
	}

	d.syncLock = &sync.Mutex{}
	d.bufferPool = &sync.Pool{
		New: func() interface{} {
			return newEventBufferHolder(defaultBufferSize)
		},
	}
	d.switchEventBuffer()
	return nil
}

func Test_discoverEvent_Run(t *testing.T) {
	type fields struct {
		eventCh        chan model.DiscoverEvent
		eventLoggers   []Logger
		bufferPool     *sync.Pool
		curEventBuffer *eventBufferHolder
		cursor         int
		syncLock       *sync.Mutex
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &discoverEvent{
				eventCh:        tt.fields.eventCh,
				eventLoggers:   tt.fields.eventLoggers,
				bufferPool:     tt.fields.bufferPool,
				curEventBuffer: tt.fields.curEventBuffer,
				cursor:         tt.fields.cursor,
				syncLock:       tt.fields.syncLock,
			}
			d.Run()
		})
	}
}
