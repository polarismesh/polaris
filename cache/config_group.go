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
	"sort"
	"strings"
	"time"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

const (
	configGroupCacheName = "configGroup"
)

func init() {
	RegisterCache(configGroupCacheName, CacheConfigFile)
}

type ConfigGroupArgs struct {
	Namespace  string
	Name       string
	Business   string
	Department string
	Metadata   map[string]string
	Offset     uint32
	Limit      uint32
	// OrderField Sort field
	OrderField string
	// OrderType Sorting rules
	OrderType string
}

// ConfigGroupCache file cache
type ConfigGroupCache interface {
	Cache
	// GetGroupByName
	GetGroupByName(namespace, name string) *model.ConfigFileGroup
	// GetGroupByID
	GetGroupByID(id uint64) *model.ConfigFileGroup
	// Query
	Query(args *ConfigGroupArgs) (uint32, []*model.ConfigFileGroup)
}

type configGroupCache struct {
	*baseCache
	storage store.Store
	// files config_file_group.id -> model.ConfigFileGroup
	groups *utils.SyncMap[uint64, *model.ConfigFileGroup]
	// name2files config_file.<namespace, group> -> model.ConfigFileGroup
	name2groups *utils.SyncMap[string, *utils.SyncMap[string, *model.ConfigFileGroup]]
	// singleGroup
	singleGroup *singleflight.Group
}

// newFileCache 创建文件缓存
func newConfigGroupCache(ctx context.Context, storage store.Store) ConfigGroupCache {
	cache := &configGroupCache{
		baseCache: newBaseCache(storage),
		storage:   storage,
	}
	return cache
}

// initialize
func (fc *configGroupCache) initialize(opt map[string]interface{}) error {
	fc.groups = utils.NewSyncMap[uint64, *model.ConfigFileGroup]()
	fc.name2groups = utils.NewSyncMap[string, *utils.SyncMap[string, *model.ConfigFileGroup]]()
	fc.singleGroup = &singleflight.Group{}
	return nil
}

// update 更新缓存函数
func (fc *configGroupCache) update() error {
	err, _ := fc.singleUpdate()
	return err
}

func (fc *configGroupCache) singleUpdate() (error, bool) {
	// 多个线程竞争，只有一个线程进行更新
	_, err, shared := fc.singleGroup.Do(fc.name(), func() (interface{}, error) {
		return nil, fc.doCacheUpdate(fc.name(), fc.realUpdate)
	})
	return err, shared
}

func (fc *configGroupCache) realUpdate() (map[string]time.Time, int64, error) {
	start := time.Now()
	groups, err := fc.storage.GetMoreConfigGroup(fc.isFirstUpdate(), fc.LastFetchTime())
	if err != nil {
		return nil, 0, err
	}
	lastMimes, update, del := fc.setConfigGroups(groups)
	log.Info("[Cache][ConfigGroup] get more config_groups",
		zap.Int("update", update), zap.Int("delete", del),
		zap.Time("last", fc.LastMtime()), zap.Duration("used", time.Since(start)))
	return lastMimes, int64(len(groups)), err
}

func (fc *configGroupCache) LastMtime() time.Time {
	return fc.baseCache.LastMtime(fc.name())
}

func (fc *configGroupCache) setConfigGroups(groups []*model.ConfigFileGroup) (map[string]time.Time, int, int) {
	lastMtime := fc.LastMtime().Unix()
	update := 0
	del := 0

	affect := map[string]struct{}{}

	for i := range groups {
		item := groups[i]

		if !item.Valid {
			del++
			fc.groups.Delete(item.Id)
			nsBucket, ok := fc.name2groups.Load(item.Namespace)
			if ok {
				nsBucket.Delete(item.Name)
			}
		} else {
			update++
			fc.groups.Store(item.Id, item)
			if _, ok := fc.name2groups.Load(item.Namespace); !ok {
				fc.name2groups.Store(item.Namespace, utils.NewSyncMap[string, *model.ConfigFileGroup]())
			}
			nsBucket, _ := fc.name2groups.Load(item.Namespace)
			nsBucket.Store(item.Name, item)
		}

		affect[item.Namespace] = struct{}{}

		modifyUnix := item.ModifyTime.Unix()
		if modifyUnix > lastMtime {
			lastMtime = modifyUnix
		}
	}

	fc.postProcessUpdatedGroups(affect)

	return map[string]time.Time{
		fc.name(): time.Unix(lastMtime, 0),
	}, update, del
}

// clear
func (fc *configGroupCache) postProcessUpdatedGroups(affect map[string]struct{}) {
	for ns := range affect {
		nsBucket, _ := fc.name2groups.Load(ns)
		count := nsBucket.Len()
		fc.reportMetricsInfo(ns, count)
	}
}

// clear
func (fc *configGroupCache) clear() error {
	fc.groups = utils.NewSyncMap[uint64, *model.ConfigFileGroup]()
	fc.name2groups = utils.NewSyncMap[string, *utils.SyncMap[string, *model.ConfigFileGroup]]()
	fc.singleGroup = &singleflight.Group{}
	return nil
}

// name
func (fc *configGroupCache) name() string {
	return configGroupCacheName
}

// GetGroupByName
func (fc *configGroupCache) GetGroupByName(namespace, name string) *model.ConfigFileGroup {
	nsBucket, ok := fc.name2groups.Load(namespace)
	if !ok {
		return nil
	}

	val, _ := nsBucket.Load(name)
	return val
}

// GetGroupByID
func (fc *configGroupCache) GetGroupByID(id uint64) *model.ConfigFileGroup {
	val, _ := fc.groups.Load(id)
	return val
}

// Query
func (fc *configGroupCache) Query(args *ConfigGroupArgs) (uint32, []*model.ConfigFileGroup) {
	values := make([]*model.ConfigFileGroup, 0, 8)
	fc.name2groups.Range(func(namespce string, groups *utils.SyncMap[string, *model.ConfigFileGroup]) bool {
		if args.Namespace != "" && !utils.IsWildMatch(namespce, args.Namespace) {
			return true
		}
		groups.Range(func(name string, group *model.ConfigFileGroup) bool {
			if args.Name != "" && !utils.IsWildMatch(name, args.Name) {
				return true
			}
			if args.Business != "" && !utils.IsWildMatch(group.Business, args.Business) {
				return true
			}
			if args.Department != "" && !utils.IsWildMatch(group.Department, args.Department) {
				return true
			}
			if len(args.Metadata) > 0 {
				for k, v := range args.Metadata {
					sv, ok := group.Metadata[k]
					if !ok || sv != v {
						return true
					}
				}
			}
			values = append(values, group)
			return true
		})
		return true
	})

	sort.Slice(values, func(i, j int) bool {
		asc := strings.ToLower(args.OrderType) == "asc" || args.OrderType == ""
		if strings.ToLower(args.OrderField) == "name" {
			return orderByConfigGroupName(values[i], values[j], asc)
		}
		return orderByConfigGroupMtime(values[i], values[j], asc)
	})

	return uint32(len(values)), doPageConfigGroups(values, args.Offset, args.Limit)
}

func orderByConfigGroupName(a, b *model.ConfigFileGroup, asc bool) bool {
	if a.Name < b.Name {
		return asc
	}
	if a.Name > b.Name {
		// false && asc always false
		return false
	}
	return a.Id < b.Id && asc
}

func orderByConfigGroupMtime(a, b *model.ConfigFileGroup, asc bool) bool {
	if a.ModifyTime.After(b.ModifyTime) {
		return asc
	}
	if a.ModifyTime.Before(b.ModifyTime) {
		// false && asc always false
		return false
	}
	return a.Id < b.Id && asc
}

func doPageConfigGroups(ret []*model.ConfigFileGroup, offset, limit uint32) []*model.ConfigFileGroup {
	amount := uint32(len(ret))
	if offset >= amount || limit == 0 {
		return nil
	}
	endIdx := offset +limit
	if endIdx > amount {
		endIdx = amount
	}
	return ret[offset:endIdx]
}
