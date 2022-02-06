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

package cache

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/store"
)

var (
	loadCnt   = 0
	getCnt    = 0
	removeCnt = 0
	expireCnt = 0
)

type FileCacheParam struct {
	ExpireTimeAfterWrite int
}

// FileCache use the lazy loading strategy. At the same time,
// setting the expiration time when creating to clear the expired cache.
type FileCache struct {
	params  FileCacheParam
	storage store.Store
	//fileId -> Entry
	files *sync.Map

	singleFlight *singleflight.Group
}

// Entry is cache entity objects
type Entry struct {
	Content string
	Md5     string
	Version uint64
	//ExpireTime sets the expiration time when creating to clear the expired cache.
	ExpireTime time.Time
	//Empty identifies whether the cache is empty
	Empty bool
}

// NewFileCache creates a new FileCache
func NewFileCache(ctx context.Context, storage store.Store, param FileCacheParam) *FileCache {
	cache := &FileCache{
		params:       param,
		storage:      storage,
		files:        new(sync.Map),
		singleFlight: new(singleflight.Group),
	}

	go cache.startClearExpireEntryTask(ctx)
	go cache.startLogStatusTask(ctx)

	return cache
}

// Get generally used for internal service calls, so it is not included in metrics
func (fc *FileCache) Get(namespace, group, fileName string) (*Entry, bool) {
	fileId := utils.GenFileId(namespace, group, fileName)
	storedEntry, ok := fc.files.Load(fileId)
	if ok {
		entry := storedEntry.(*Entry)
		return entry, true
	}
	return nil, false
}

// GetOrLoadIfAbsent gets fileCache.If the cache misses, it will be loaded from the database.
// If the data can't be obtained from the database, an empty object will be cached
// to prevent the cache from being broken down all the time.
func (fc *FileCache) GetOrLoadIfAbsent(namespace, group, fileName string) (*Entry, error) {
	getCnt++

	fileId := utils.GenFileId(namespace, group, fileName)
	storedEntry, ok := fc.files.Load(fileId)
	if ok {
		return storedEntry.(*Entry), nil
	}
	// Cache miss, load data from database.
	// To avoid the database being overwhelmed in the case of large concurrency,
	// use singleFlight to ensure concurrency safety.
	storedEntry, err, _ := fc.singleFlight.Do(fileId, func() (interface{}, error) {
		storedEntry, ok = fc.files.Load(fileId)
		if ok {
			return storedEntry.(*Entry), nil
		}

		loadCnt++
		// load data from database
		file, err := fc.storage.GetConfigFileRelease(nil, namespace, group, fileName)
		if err != nil {
			log.ConfigScope().Error("[Config][Cache] load config file release error.",
				zap.String("namespace", namespace),
				zap.String("group", group),
				zap.String("fileName", fileName),
				zap.Error(err))
			return nil, err
		}

		//data can't be obtained from the database, an empty object will be cached
		if file == nil {
			emptyEntry := &Entry{
				ExpireTime: fc.getExpireTime(),
				Empty:      true,
			}
			fc.files.Store(fileId, emptyEntry)
			return emptyEntry, nil
		}

		//data in the database, update the cache
		newEntry := &Entry{
			Content:    file.Content,
			Md5:        file.Md5,
			Version:    file.Version,
			ExpireTime: fc.getExpireTime(),
			Empty:      false,
		}

		//cache isn't exist, it is directly stored in the cache
		if !ok {
			fc.files.Store(fileId, newEntry)
			return newEntry, nil
		}

		//cache exists, the idempotent judgment can only be stored in the cache with a larger version number
		oldEntry := storedEntry.(*Entry)
		if oldEntry.Empty || newEntry.Version > oldEntry.Version {
			fc.files.Store(fileId, newEntry)
		}
		return newEntry, nil
	})

	return storedEntry.(*Entry), err

}

// Remove deletes fileCache
func (fc *FileCache) Remove(namespace, group, fileName string) {
	removeCnt++
	fileId := utils.GenFileId(namespace, group, fileName)
	fc.files.Delete(fileId)
}

// ReLoad reloads fileCache
func (fc *FileCache) ReLoad(namespace, group, fileName string) (*Entry, error) {
	fc.Remove(namespace, group, fileName)
	return fc.GetOrLoadIfAbsent(namespace, group, fileName)
}

// getExpireTime sets random expire time to avoid expiring simultaneously.
// the random expireTime is between 60 minutes and 70 minutes.
func (fc *FileCache) getExpireTime() time.Time {
	randTime := rand.Intn(10*60) + fc.params.ExpireTimeAfterWrite
	return time.Now().Add(time.Duration(randTime) * time.Second)
}

// startClearExpireEntryTask regular cleans expired cache.
func (fc *FileCache) startClearExpireEntryTask(ctx context.Context) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			curExpiredFileCnt := 0
			fc.files.Range(func(fileId, entry interface{}) bool {
				if time.Now().After(entry.(*Entry).ExpireTime) {
					fc.files.Delete(fileId)
					curExpiredFileCnt++
				}
				return true
			})

			if curExpiredFileCnt > 0 {
				log.ConfigScope().Info("[Config][Cache] clear expired file cache.", zap.Int("count", curExpiredFileCnt))
			}

			expireCnt += curExpiredFileCnt
		}
	}
}

// startLogStatusTask prints cache status at fix rate
func (fc *FileCache) startLogStatusTask(ctx context.Context) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			log.ConfigScope().Info("[Config][Cache] cache status:",
				zap.Int("getCnt", getCnt),
				zap.Int("loadCnt", loadCnt),
				zap.Int("removeCnt", removeCnt),
				zap.Int("expireCnt", expireCnt))
		}
	}
}
