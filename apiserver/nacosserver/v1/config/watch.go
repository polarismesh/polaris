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

package config

import (
	"context"
	"sync"
	"time"

	"github.com/polarismesh/specification/source/go/api/v1/config_manage"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/config"
)

type LongPollWatchContext struct {
	clientId         string
	labels           map[string]string
	once             sync.Once
	finishTime       time.Time
	finishChan       chan *config_manage.ConfigClientResponse
	watchConfigFiles map[string]*config_manage.ClientConfigFileInfo
	betaMatcher      config.BetaReleaseMatcher
}

func (c *LongPollWatchContext) ClientLabels() map[string]string {
	return c.labels
}

// GetNotifieResult .
func (c *LongPollWatchContext) GetNotifieResult() *config_manage.ConfigClientResponse {
	return <-c.finishChan
}

// GetNotifieResultWithTime .
func (c *LongPollWatchContext) GetNotifieResultWithTime(timeout time.Duration) (*config_manage.ConfigClientResponse, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case ret := <-c.finishChan:
		return ret, nil
	case <-timer.C:
		return nil, context.DeadlineExceeded
	}
}

// IsOnce
func (c *LongPollWatchContext) IsOnce() bool {
	return true
}

// ShouldExpire .
func (c *LongPollWatchContext) ShouldExpire(now time.Time) bool {
	return now.After(c.finishTime)
}

// ClientID .
func (c *LongPollWatchContext) ClientID() string {
	return c.clientId
}

// ShouldNotify .
func (c *LongPollWatchContext) ShouldNotify(event *model.SimpleConfigFileRelease) bool {
	key := event.FileKey()
	watchFile, ok := c.watchConfigFiles[key]
	if !ok {
		return false
	}
	// 删除操作，直接通知
	if !event.Valid {
		return true
	}
	isChange := watchFile.GetMd5().GetValue() != event.Md5
	return isChange
}

func (c *LongPollWatchContext) ListWatchFiles() []*config_manage.ClientConfigFileInfo {
	ret := make([]*config_manage.ClientConfigFileInfo, 0, len(c.watchConfigFiles))
	for _, v := range c.watchConfigFiles {
		ret = append(ret, v)
	}
	return ret
}

// AppendInterest .
func (c *LongPollWatchContext) AppendInterest(item *config_manage.ClientConfigFileInfo) {
	key := model.BuildKeyForClientConfigFileInfo(item)
	c.watchConfigFiles[key] = item
}

// RemoveInterest .
func (c *LongPollWatchContext) RemoveInterest(item *config_manage.ClientConfigFileInfo) {
	key := model.BuildKeyForClientConfigFileInfo(item)
	delete(c.watchConfigFiles, key)
}

// Close .
func (c *LongPollWatchContext) Close() error {
	c.once.Do(func() {
		close(c.finishChan)
	})
	return nil
}

func (c *LongPollWatchContext) Reply(rsp *config_manage.ConfigClientResponse) {
	c.once.Do(func() {
		c.finishChan <- rsp
		close(c.finishChan)
	})
}
