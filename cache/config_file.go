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
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

const (
	configFileCacheName = "configFile"
)

func init() {
	RegisterCache(configFileCacheName, CacheConfigFile)
}

// FileCache file cache
type FileCache interface {
	Cache
	// Get 通过ns,group,filename获取 Entry
	Get(namespace, group, fileName string) (*Entry, bool)
	// GetOrLoadIfAbsent
	GetOrLoadIfAbsent(namespace, group, fileName string) (*Entry, error)
	// Remove
	Remove(namespace, group, fileName string)
	// ReLoad
	ReLoad(namespace, group, fileName string) (*Entry, error)
	// GetOrLoadGroupByName
	GetOrLoadGroupByName(namespace, group string) (*model.ConfigFileGroup, error)
	// GetOrLoadGroupById
	GetOrLoadGroupById(id uint64) (*model.ConfigFileGroup, error)
	// CleanAll
	CleanAll()
}

// FileCache 文件缓存，使用 loading cache 懒加载策略。同时写入时设置过期时间，定时清理过期的缓存。
type fileCache struct {
	storage store.Store
	// fileId -> Entry
	files sync.Map
	// fileId -> lock
	fileLoadLocks sync.Map
	// loadCnt
	loadCnt int32
	// getCnt
	getCnt int32
	// removeCnt
	removeCnt int32
	// expireCnt
	expireCnt int32
	// configGroups
	configGroups *configFileGroupBucket
	//
	singleLoadGroup singleflight.Group
	// expireTimeAfterWrite
	expireTimeAfterWrite int
	// ctx
	ctx context.Context
}

// Entry 缓存实体对象
type Entry struct {
	Content string
	Md5     string
	Version uint64
	Tags    []*model.ConfigFileTag
	// 创建的时候，设置过期时间
	ExpireTime time.Time
	// 标识是否是空缓存
	Empty bool
}

func (e *Entry) GetDataKey() string {
	for _, tag := range e.Tags {
		if tag.Key == utils.ConfigFileTagKeyDataKey {
			return tag.Value
		}
	}
	return ""
}

func (e *Entry) GetEncryptAlgo() string {
	for _, tag := range e.Tags {
		if tag.Key == utils.ConfigFileTagKeyEncryptAlgo {
			return tag.Value
		}
	}
	return ""
}

func (e *Entry) Encrypted() bool {
	return e.GetDataKey() != ""
}

// newFileCache 创建文件缓存
func newFileCache(ctx context.Context, storage store.Store) FileCache {
	cache := &fileCache{
		storage:      storage,
		ctx:          ctx,
		configGroups: newConfigFileGroupBucket(),
	}
	return cache
}

// initialize
func (fc *fileCache) initialize(opt map[string]interface{}) error {
	fc.expireTimeAfterWrite, _ = opt["expireTimeAfterWrite"].(int)
	if fc.expireTimeAfterWrite == 0 {
		fc.expireTimeAfterWrite = 3600
	}

	go fc.configGroups.runCleanExpire(fc.ctx, time.Minute, int64(fc.expireTimeAfterWrite))
	go fc.startClearExpireEntryTask(fc.ctx)
	go fc.startLogStatusTask(fc.ctx)
	go fc.reportMetricsInfo(fc.ctx)
	return nil
}

// addListener 添加
func (fc *fileCache) addListener(_ []Listener) {

}

// update
func (fc *fileCache) update() error {
	return nil
}

// clear
func (fc *fileCache) clear() error {
	fc.CleanAll()
	fc.configGroups.clean()
	return nil
}

// name
func (fc *fileCache) name() string {
	return configFileCacheName
}

// Get 一般用于内部服务调用，所以不计入 metrics
func (fc *fileCache) Get(namespace, group, fileName string) (*Entry, bool) {
	fileId := utils.GenFileId(namespace, group, fileName)
	storedEntry, ok := fc.files.Load(fileId)
	if ok {
		entry := storedEntry.(*Entry)
		return entry, true
	}
	return nil, false
}

// GetOrLoadGroupByName 获取配置分组缓存
func (fc *fileCache) GetOrLoadGroupByName(namespace, group string) (*model.ConfigFileGroup, error) {
	item := fc.configGroups.getGroupByName(namespace, group)
	if item != nil {
		return item, nil
	}

	key := namespace + utils.FileIdSeparator + group

	ret, err, _ := fc.singleLoadGroup.Do(key, func() (interface{}, error) {
		data, err := fc.storage.GetConfigFileGroup(namespace, group)
		if err != nil {
			return nil, err
		}

		fc.configGroups.saveGroup(namespace, group, data)
		return data, nil
	})

	if err != nil {
		return nil, err
	}

	if ret != nil {
		return ret.(*model.ConfigFileGroup), nil
	}

	return nil, nil
}

func (fc *fileCache) GetOrLoadGroupById(id uint64) (*model.ConfigFileGroup, error) {
	item := fc.configGroups.getGroupById(id)
	if item != nil {
		return item, nil
	}

	key := fmt.Sprintf("config_group_%d", id)

	ret, err, _ := fc.singleLoadGroup.Do(key, func() (interface{}, error) {
		data, err := fc.storage.GetConfigFileGroupById(id)
		if err != nil {
			return nil, err
		}

		fc.configGroups.saveGroupById(id, data)
		return data, nil
	})

	if err != nil {
		return nil, err
	}

	if ret != nil {
		return ret.(*model.ConfigFileGroup), nil
	}

	return nil, nil
}

// GetOrLoadIfAbsent 获取缓存，如果缓存没命中则会从数据库中加载，如果数据库里获取不到数据，则会缓存一个空对象防止缓存一直被击穿
func (fc *fileCache) GetOrLoadIfAbsent(namespace, group, fileName string) (*Entry, error) {
	atomic.AddInt32(&fc.getCnt, 1)

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
	atomic.AddInt32(&fc.loadCnt, 1)

	file, tags, err := fc.getConfigFileReleaseAndTags(namespace, group, fileName)
	if err != nil {
		configLog.Error("[Config][Cache] load config file release and tags error.",
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("fileName", fileName),
			zap.Error(err))
		return nil, err
	}

	// 数据库中没有该对象, 为了避免对象不存在时，一直击穿数据库，所以缓存空对象
	if file == nil {
		configLog.Warn("[Config][Cache] load config file release not found.",
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
		Tags:       tags,
		ExpireTime: fc.getExpireTime(),
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

func (fc *fileCache) getConfigFileReleaseAndTags(
	namespace, group, fileName string) (*model.ConfigFileRelease, []*model.ConfigFileTag, error) {
	file, err := fc.storage.GetConfigFileRelease(nil, namespace, group, fileName)
	if err != nil {
		configLog.Error("[Config][Cache] load config file release error.",
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("fileName", fileName),
			zap.Error(err))
		return nil, nil, err
	}
	tags, err := fc.storage.QueryTagByConfigFile(namespace, group, fileName)
	if err != nil {
		configLog.Error("[Config][Cache] load config file tag error.",
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("fileName", fileName),
			zap.Error(err))
		return nil, nil, err
	}
	return file, tags, nil
}

// Remove 删除缓存对象
func (fc *fileCache) Remove(namespace, group, fileName string) {
	atomic.AddInt32(&fc.removeCnt, 1)
	fileId := utils.GenFileId(namespace, group, fileName)
	fc.files.Delete(fileId)
}

// ReLoad 重新加载缓存
func (fc *fileCache) ReLoad(namespace, group, fileName string) (*Entry, error) {
	fc.Remove(namespace, group, fileName)
	return fc.GetOrLoadIfAbsent(namespace, group, fileName)
}

// CleanAll 清空缓存，仅用于集成测试
func (fc *fileCache) CleanAll() {
	fc.files.Range(func(key, _ interface{}) bool {
		fc.files.Delete(key)
		return true
	})
}

// 缓存过期时间，为了避免集中失效，加上随机数。[60 ~ 70]分钟内随机失效
func (fc *fileCache) getExpireTime() time.Time {
	randTime := rand.Intn(10*60) + fc.expireTimeAfterWrite
	return time.Now().Add(time.Duration(randTime) * time.Second)
}

// 定时清理过期的缓存
func (fc *fileCache) startClearExpireEntryTask(ctx context.Context) {
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
				configLog.Info("[Config][Cache] clear expired file cache.", zap.Int("count", curExpiredFileCnt))
			}

			atomic.AddInt32(&fc.expireCnt, int32(curExpiredFileCnt))
		}
	}
}

// print cache status at fix rate
func (fc *fileCache) startLogStatusTask(ctx context.Context) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			configLog.Info("[Config][Cache] cache status:",
				zap.Int32("getCnt", atomic.LoadInt32(&fc.getCnt)),
				zap.Int32("loadCnt", atomic.LoadInt32(&fc.loadCnt)),
				zap.Int32("removeCnt", atomic.LoadInt32(&fc.removeCnt)),
				zap.Int32("expireCnt", atomic.LoadInt32(&fc.expireCnt)))
		}
	}
}
