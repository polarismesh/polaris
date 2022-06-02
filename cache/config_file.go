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

// FileCache 文件缓存，使用 loading cache 懒加载策略。同时写入时设置过期时间，定时清理过期的缓存。
type FileCache struct {
	params  FileCacheParam
	storage store.Store
	// fileId -> Entry
	files *sync.Map
	// fileId -> lock
	fileLoadLocks *sync.Map
}

// Entry 缓存实体对象
type Entry struct {
	Content string
	Md5     string
	Version uint64
	// 创建的时候，设置过期时间
	ExpireTime time.Time
	// 标识是否是空缓存
	Empty bool
}

// NewFileCache 创建文件缓存
func NewFileCache(ctx context.Context, storage store.Store, param FileCacheParam) *FileCache {
	cache := &FileCache{
		params:        param,
		storage:       storage,
		files:         new(sync.Map),
		fileLoadLocks: new(sync.Map),
	}

	go cache.startClearExpireEntryTask(ctx)
	go cache.startLogStatusTask(ctx)

	return cache
}

// Get 一般用于内部服务调用，所以不计入 metrics
func (fc *FileCache) Get(namespace, group, fileName string) (*Entry, bool) {
	fileId := utils.GenFileId(namespace, group, fileName)
	storedEntry, ok := fc.files.Load(fileId)
	if ok {
		entry := storedEntry.(*Entry)
		return entry, true
	}
	return nil, false
}

// GetOrLoadIfAbsent 获取缓存，如果缓存没命中则会从数据库中加载，如果数据库里获取不到数据，则会缓存一个空对象防止缓存一直被击穿
func (fc *FileCache) GetOrLoadIfAbsent(namespace, group, fileName string) (*Entry, error) {
	getCnt++

	fileId := utils.GenFileId(namespace, group, fileName)
	storedEntry, ok := fc.files.Load(fileId)
	if ok {
		return storedEntry.(*Entry), nil
	}

	// 缓存未命中，则从数据库里加载数据

	// 为了避免在大并发量的情况下，数据库被压垮，所以增加锁。同时为了提高性能，减小锁粒度
	lockObj, _ := fc.fileLoadLocks.LoadOrStore(fileId, new(sync.Mutex))
	loadLock := lockObj.(*sync.Mutex)
	loadLock.Lock()
	defer loadLock.Unlock()

	// double check
	storedEntry, ok = fc.files.Load(fileId)
	if ok {
		return storedEntry.(*Entry), nil
	}

	// 从数据库中加载数据
	loadCnt++

	file, err := fc.storage.GetConfigFileRelease(nil, namespace, group, fileName)
	if err != nil {
		log.ConfigScope().Error("[Config][Cache] load config file release error.",
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("fileName", fileName),
			zap.Error(err))
		return nil, err
	}

	// 数据库中没有该对象, 为了避免对象不存在时，一直击穿数据库，所以缓存空对象
	if file == nil {
		log.ConfigScope().Warn("[Config][Cache] load config file release not found.",
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("fileName", fileName),
			zap.Error(err))
		emptyEntry := &Entry{
			ExpireTime: fc.getExpireTime(),
			Empty:      true,
		}
		fc.files.Store(fileId, emptyEntry)
		return emptyEntry, nil
	}

	// 数据库中有对象，更新缓存
	newEntry := &Entry{
		Content:    file.Content,
		Md5:        file.Md5,
		Version:    file.Version,
		ExpireTime: fc.getExpireTime(),
		Empty:      false,
	}

	// 缓存不存在，则直接存入缓存
	if !ok {
		fc.files.Store(fileId, newEntry)
		return newEntry, nil
	}

	// 缓存存在，幂等判断只能存入版本号更大的
	oldEntry := storedEntry.(*Entry)
	if oldEntry.Empty || newEntry.Version > oldEntry.Version {
		fc.files.Store(fileId, newEntry)
	}

	return newEntry, nil
}

// Remove 删除缓存对象
func (fc *FileCache) Remove(namespace, group, fileName string) {
	removeCnt++
	fileId := utils.GenFileId(namespace, group, fileName)
	fc.files.Delete(fileId)
}

// ReLoad 重新加载缓存
func (fc *FileCache) ReLoad(namespace, group, fileName string) (*Entry, error) {
	fc.Remove(namespace, group, fileName)
	return fc.GetOrLoadIfAbsent(namespace, group, fileName)
}

// Clear 清空缓存，仅用于集成测试
func (fc *FileCache) Clear() {
	fc.files.Range(func(key, _ interface{}) bool {
		fc.files.Delete(key)
		return true
	})
}

// 缓存过期时间，为了避免集中失效，加上随机数。[60 ~ 70]分钟内随机失效
func (fc *FileCache) getExpireTime() time.Time {
	randTime := rand.Intn(10*60) + fc.params.ExpireTimeAfterWrite
	return time.Now().Add(time.Duration(randTime) * time.Second)
}

// 定时清理过期的缓存
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

// print cache status at fix rate
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
