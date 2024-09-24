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
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	types "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

type configGroupCache struct {
	*types.BaseCache
	storage store.Store
	// files config_file_group.id -> model.ConfigFileGroup
	groups *utils.SyncMap[uint64, *model.ConfigFileGroup]
	// name2files config_file.<namespace, group> -> model.ConfigFileGroup
	name2groups *utils.SyncMap[string, *utils.SyncMap[string, *model.ConfigFileGroup]]
	// revisions namespace -> [revision]
	revisions *utils.SyncMap[string, string]
	// singleGroup
	singleGroup *singleflight.Group
}

// NewConfigGroupCache 创建文件缓存
func NewConfigGroupCache(storage store.Store, cacheMgr types.CacheManager) types.ConfigGroupCache {
	gc := &configGroupCache{
		storage: storage,
	}
	gc.BaseCache = types.NewBaseCacheWithRepoerMetrics(storage, cacheMgr, gc.reportMetricsInfo)
	return gc
}

// Initialize
func (fc *configGroupCache) Initialize(opt map[string]interface{}) error {
	fc.groups = utils.NewSyncMap[uint64, *model.ConfigFileGroup]()
	fc.name2groups = utils.NewSyncMap[string, *utils.SyncMap[string, *model.ConfigFileGroup]]()
	fc.singleGroup = &singleflight.Group{}
	fc.revisions = utils.NewSyncMap[string, string]()
	return nil
}

// Update 更新缓存函数
func (fc *configGroupCache) Update() error {
	err, _ := fc.singleUpdate()
	return err
}

func (fc *configGroupCache) singleUpdate() (error, bool) {
	// 多个线程竞争，只有一个线程进行更新
	_, err, shared := fc.singleGroup.Do(fc.Name(), func() (interface{}, error) {
		return nil, fc.DoCacheUpdate(fc.Name(), fc.realUpdate)
	})
	return err, shared
}

func (fc *configGroupCache) realUpdate() (map[string]time.Time, int64, error) {
	start := time.Now()
	groups, err := fc.storage.GetMoreConfigGroup(fc.IsFirstUpdate(), fc.LastFetchTime())
	if err != nil {
		return nil, 0, err
	}
	if len(groups) == 0 {
		return nil, 0, nil
	}
	lastMimes, update, del := fc.setConfigGroups(groups)
	log.Info("[Cache][ConfigGroup] get more config_groups",
		zap.Int("update", update), zap.Int("delete", del),
		zap.Time("last", fc.LastMtime()), zap.Duration("used", time.Since(start)))
	return lastMimes, int64(len(groups)), err
}

func (fc *configGroupCache) LastMtime() time.Time {
	return fc.BaseCache.LastMtime(fc.Name())
}

func (fc *configGroupCache) setConfigGroups(groups []*model.ConfigFileGroup) (map[string]time.Time, int, int) {
	lastMtime := fc.LastMtime().Unix()
	update := 0
	del := 0

	affect := map[string]struct{}{}

	for i := range groups {
		item := groups[i]
		affect[item.Namespace] = struct{}{}

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

		modifyUnix := item.ModifyTime.Unix()
		if modifyUnix > lastMtime {
			lastMtime = modifyUnix
		}
	}

	fc.postProcessUpdatedGroups(affect)

	return map[string]time.Time{
		fc.Name(): time.Unix(lastMtime, 0),
	}, update, del
}

// postProcessUpdatedGroups
func (fc *configGroupCache) postProcessUpdatedGroups(affect map[string]struct{}) {
	for ns := range affect {
		nsBucket, ok := fc.name2groups.Load(ns)
		if !ok {
			continue
		}
		count := nsBucket.Len()
		revisions := make([]string, 0, count)
		nsBucket.Range(func(key string, val *model.ConfigFileGroup) {
			revisions = append(revisions, val.Revision)
		})

		revision, err := types.CompositeComputeRevision(revisions)
		if err != nil {
			revision = utils.NewUUID()
		}
		fc.revisions.Store(ns, revision)
	}
}

// Clear
func (fc *configGroupCache) Clear() error {
	fc.groups = utils.NewSyncMap[uint64, *model.ConfigFileGroup]()
	fc.name2groups = utils.NewSyncMap[string, *utils.SyncMap[string, *model.ConfigFileGroup]]()
	fc.singleGroup = &singleflight.Group{}
	fc.revisions = utils.NewSyncMap[string, string]()
	return nil
}

// Name
func (fc *configGroupCache) Name() string {
	return types.ConfigGroupCacheName
}

func (fc *configGroupCache) ListGroups(namespace string) ([]*model.ConfigFileGroup, string) {
	nsBucket, ok := fc.name2groups.Load(namespace)
	if !ok {
		return nil, ""
	}
	ret := make([]*model.ConfigFileGroup, 0, nsBucket.Len())
	nsBucket.Range(func(key string, val *model.ConfigFileGroup) {
		ret = append(ret, val)
	})

	revision, ok := fc.revisions.Load(namespace)
	if !ok {
		revision = utils.NewUUID()
	}

	return ret, revision
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

// forceQueryUpdate 为了确保读取的数据是最新的，这里需要做一个强制 update 的动作进行数据读取处理
func (fc *configGroupCache) forceQueryUpdate() error {
	err, shared := fc.singleUpdate()
	// shared == true，表示当前已经有正在 update 执行的任务，这个任务不一定能够读取到最新的数据
	// 为了避免读取到脏数据，在发起一次 singleUpdate
	if shared {
		configLog.Debug("[Config][Group][Query] force query update second")
		err, _ = fc.singleUpdate()
	}
	return err
}

// Query
func (fc *configGroupCache) Query(args *types.ConfigGroupArgs) (uint32, []*model.ConfigFileGroup, error) {
	if err := fc.forceQueryUpdate(); err != nil {
		return 0, nil, err
	}

	values := make([]*model.ConfigFileGroup, 0, 8)
	fc.name2groups.ReadRange(func(namespce string, groups *utils.SyncMap[string, *model.ConfigFileGroup]) {
		if args.Namespace != "" && utils.IsWildNotMatch(namespce, args.Namespace) {
			return
		}
		groups.ReadRange(func(name string, group *model.ConfigFileGroup) {
			if args.Name != "" && utils.IsWildNotMatch(name, args.Name) {
				return
			}
			if args.Business != "" && utils.IsWildNotMatch(group.Business, args.Business) {
				return
			}
			if args.Department != "" && utils.IsWildNotMatch(group.Department, args.Department) {
				return
			}
			if len(args.Metadata) > 0 {
				for k, v := range args.Metadata {
					sv, ok := group.Metadata[k]
					if !ok || sv != v {
						return
					}
				}
			}
			values = append(values, group)
		})
	})

	sort.Slice(values, func(i, j int) bool {
		asc := strings.ToLower(args.OrderType) == "asc" || args.OrderType == ""
		if strings.ToLower(args.OrderField) == "name" {
			return orderByConfigGroupName(values[i], values[j], asc)
		}
		return orderByConfigGroupMtime(values[i], values[j], asc)
	})

	return uint32(len(values)), doPageConfigGroups(values, args.Offset, args.Limit), nil
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
	endIdx := offset + limit
	if endIdx > amount {
		endIdx = amount
	}
	return ret[offset:endIdx]
}
