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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/grafana/loki/pkg/logproto"
	json "github.com/json-iterator/go"

	"github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

const (
	contentType    = "application/x-protobuf"
	defaultTimeout = 10 * time.Second
)

// LokiLoggerConfig Loki 日志器配置
type LokiLoggerConfig struct {
	PushURL  string            `json:"pushURL"`
	TenantID string            `json:"tenantID"`
	Labels   map[string]string `json:"labels"`
	Timeout  time.Duration     `json:"timeout"`
}

// UnmarshalJSON Loki 日志器配置 json unmarshal 方法
func (c *LokiLoggerConfig) UnmarshalJSON(data []byte) (err error) {
	var tmp struct {
		PushURL  string            `json:"pushURL"`
		TenantID string            `json:"tenantID"`
		Labels   map[string]string `json:"labels"`
		Timeout  string            `json:"timeout"`
	}
	if err = json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	c.PushURL = tmp.PushURL
	c.TenantID = tmp.TenantID
	c.Labels = tmp.Labels
	if len(tmp.Timeout) > 0 {
		c.Timeout, err = time.ParseDuration(tmp.Timeout)
	}
	return err
}

// Validate Loki 日志器配置校验配置是否合法
func (c *LokiLoggerConfig) Validate() error {
	if c.PushURL == "" {
		return errors.New("PushURL is empty")
	}
	return nil
}

// LokiLogger Loki 日志器
type LokiLogger struct {
	pushURL  string            // Loki push api 地址
	tenantID string            // 租户ID
	labels   map[string]string // 标签
	client   *http.Client      // http 客户端
	timeout  time.Duration     // http 请求超时时间
}

// newLokiLogger 根据配置新建 Loki 日志器
func newLokiLogger(opt map[string]interface{}) (*LokiLogger, error) {
	data, err := json.Marshal(opt)
	if err != nil {
		return nil, err
	}
	conf := &LokiLoggerConfig{}
	if err := json.Unmarshal(data, conf); err != nil {
		return nil, err
	}
	if err := conf.Validate(); err != nil {
		return nil, err
	}
	if conf.Labels == nil {
		conf.Labels = make(map[string]string)
	}
	conf.Labels["source"] = PluginName
	if conf.Timeout == 0 {
		conf.Timeout = defaultTimeout
	}
	lokiLogger := &LokiLogger{
		pushURL:  conf.PushURL,
		tenantID: conf.TenantID,
		labels:   conf.Labels,
		client:   &http.Client{},
		timeout:  conf.Timeout,
	}
	return lokiLogger, nil
}

// Log Loki 日志器记录服务发现事件日志
func (l *LokiLogger) Log(events []model.DiscoverEvent) {
	// 按事件类型分组
	eventType2entries := make(map[model.DiscoverEventType][]logproto.Entry)
	for _, event := range events {
		entry := logproto.Entry{
			Timestamp: event.CreateTime,
			Line: fmt.Sprintf("%s|%s|%s|%d|%s|%d|%s",
				event.Namespace,
				event.Service,
				event.Host,
				event.Port,
				event.EType,
				event.CreateTime.Unix(),
				utils.LocalHost),
		}
		eventType2entries[event.EType] = append(eventType2entries[event.EType], entry)
	}
	// 将事件类型设置为标签
	var streams []logproto.Stream
	for etype, entries := range eventType2entries {
		l.labels["event_type"] = string(etype)
		streams = append(streams, logproto.Stream{
			Labels:  genLabels(l.labels),
			Entries: entries,
		})
	}

	req := logproto.PushRequest{
		Streams: streams,
	}
	buf, err := proto.Marshal(&req)
	if err != nil {
		log.Errorf("[Discoverevent][LokiLogger] marshal push request error: %v", err)
		return
	}

	buf = snappy.Encode(nil, buf)
	resp, err := l.send(context.Background(), buf)
	if err != nil {
		log.Errorf("[Discoverevent][LokiLogger] send request error: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("[Discoverevent][LokiLogger] read resp body error: %v", err)
			return
		}
		log.Errorf("[Discoverevent][LokiLogger] send request return status code: %d, message: %s", resp.StatusCode, body)
		return
	}
}

// send loki 日志器发送日志 http 请求
func (l *LokiLogger) send(ctx context.Context, reqBody []byte) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, l.timeout)
	defer cancel()
	req, err := http.NewRequest(http.MethodPost, l.pushURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", contentType)
	if l.tenantID != "" {
		req.Header.Set("X-Scope-OrgID", l.tenantID)
	}
	resp, err := l.client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// genLables 生成标签
func genLabels(labels map[string]string) string {
	var sb strings.Builder
	sb.WriteString("{")
	var i int
	for k, v := range labels {
		i++
		sb.WriteString(fmt.Sprintf("%s=\"%s\"", k, v))
		if i != len(labels) {
			sb.WriteString(",")
		}
	}
	sb.WriteString("}")
	return sb.String()
}
