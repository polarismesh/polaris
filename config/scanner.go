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
	"time"

	"go.uber.org/zap"

	"github.com/polarismesh/polaris-server/cache"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store"
)

const (
	DefaultScanTimeOffset = -10 * time.Second
	FirstScanTimeOffset   = -time.Minute * 10
	MessageExpireTime     = -5 * time.Second
)

// releaseMessageScanner 发布事件扫描器，根据发布时间获取发布事件，并通过 EventCenter 广播事件
type releaseMessageScanner struct {
	storage store.Store

	lastScannerTime time.Time

	scanInterval time.Duration

	fileCache *cache.FileCache

	eventCenter *Center
}

func initReleaseMessageScanner(ctx context.Context, storage store.Store, fileCache *cache.FileCache,
	eventCenter *Center, scanInterval time.Duration) error {
	scanner := &releaseMessageScanner{
		storage:      storage,
		fileCache:    fileCache,
		eventCenter:  eventCenter,
		scanInterval: scanInterval,
	}

	err := scanner.scanAtFirstTime()
	if err != nil {
		return err
	}

	go scanner.startScanTask(ctx)

	return nil
}

func (s *releaseMessageScanner) scanAtFirstTime() error {
	t := time.Now().Add(FirstScanTimeOffset)
	s.lastScannerTime = t

	releases, err := s.storage.FindConfigFileReleaseByModifyTimeAfter(t)
	if err != nil {
		log.ConfigScope().Error("[Config][Scanner] scan config file release error.", zap.Error(err))
		return err
	}

	if len(releases) == 0 {
		return nil
	}

	log.ConfigScope().Info("[Config][Scanner] scan config file release count at first time. ", zap.Int("count", len(releases)))

	err = s.handlerReleases(true, releases)
	if err != nil {
		log.ConfigScope().Error("[Config][Scanner] handler release message error.", zap.Error(err))
		return err
	}

	return nil
}

func (s *releaseMessageScanner) startScanTask(ctx context.Context) {
	t := time.NewTicker(s.scanInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			// 为了避免丢失消息，扫描发布消息的时间点往前拨10s。因为处理消息是幂等的，所以即使捞出重复消息也能够正常处理
			scanIdx := s.lastScannerTime.Add(DefaultScanTimeOffset)
			releases, err := s.storage.FindConfigFileReleaseByModifyTimeAfter(scanIdx)

			if err != nil {
				log.ConfigScope().Error("[Config][Scanner] scan config file release error.", zap.Error(err))
				continue
			}

			err = s.handlerReleases(false, releases)
			if err != nil {
				log.ConfigScope().Error("[Config][Scanner] handler release message error.", zap.Error(err))
			}
		}
	}
}

func (s *releaseMessageScanner) handlerReleases(firstTime bool, releases []*model.ConfigFileRelease) error {
	if len(releases) == 0 {
		return nil
	}

	maxModifyTime := s.lastScannerTime
	newReleaseCnt := 0

	for _, release := range releases {
		if release.ModifyTime.After(maxModifyTime) {
			maxModifyTime = release.ModifyTime
			newReleaseCnt++
		}

		entry, ok := s.fileCache.Get(release.Namespace, release.Group, release.FileName)

		// 缓存不存在，或者缓存的版本号落后数据库的版本号则处理消息. 因为有版本号判断，所以能够幂等处理重复消息
		if !ok || entry.Empty || release.Version > entry.Version {
			if release.Flag == 1 {
				// 删除的发布消息，因为缓存被清除了，所以会一直判断为新消息，所以通过判断消息是否过期来避免一直重复消费
				if isExpireMessage(release) {
					continue
				}
				// 删除配置文件，删除缓存
				s.fileCache.Remove(release.Namespace, release.Group, release.FileName)
			} else {
				// 正常配置发布，更新缓存
				_, _ = s.fileCache.ReLoad(release.Namespace, release.Group, release.FileName)
			}

			isExpire := isExpireMessage(release)

			if !firstTime && !isExpire {
				s.eventCenter.handleEvent(Event{
					EventType: eventTypePublishConfigFile,
					Message:   release,
				})
			}
		}
	}

	s.lastScannerTime = maxModifyTime

	if newReleaseCnt > 0 {
		log.ConfigScope().Info("[Config][Scanner] scan config file release count. ", zap.Int("count", len(releases)))
	}

	return nil
}

func isExpireMessage(release *model.ConfigFileRelease) bool {
	return release.ModifyTime.Before(time.Now().Add(MessageExpireTime))
}
