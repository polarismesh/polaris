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

package service

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

const (
	ServiceContractName = "serviceContract"
)

func NewServiceContractCache(storage store.Store, cacheMgr types.CacheManager) types.ServiceContractCache {
	return &serviceContractCache{
		BaseCache: types.NewBaseCache(storage, cacheMgr),
	}
}

type serviceContractCache struct {
	*types.BaseCache

	lastMtimeLogged int64

	// data namespace/service/name/protocol/version -> *model.EnrichServiceContract
	data *utils.SyncMap[string, *model.EnrichServiceContract]
	// contracts 服务契约缓存，namespace -> service -> []*model.EnrichServiceContract
	contracts   *utils.SyncMap[string, *utils.SyncMap[string, *utils.SyncMap[string, *model.EnrichServiceContract]]]
	singleGroup *singleflight.Group
}

// Initialize
func (sc *serviceContractCache) Initialize(c map[string]interface{}) error {
	sc.singleGroup = &singleflight.Group{}
	sc.data = utils.NewSyncMap[string, *model.EnrichServiceContract]()
	sc.contracts = utils.NewSyncMap[string, *utils.SyncMap[string, *utils.SyncMap[string, *model.EnrichServiceContract]]]()
	return nil
}

// Update
func (sc *serviceContractCache) Update() error {
	err, _ := sc.singleUpdate()
	return err
}

func (sc *serviceContractCache) singleUpdate() (error, bool) {
	// 多个线程竞争，只有一个线程进行更新
	_, err, shared := sc.singleGroup.Do(sc.Name(), func() (interface{}, error) {
		defer func() {
			sc.lastMtimeLogged = types.LogLastMtime(sc.lastMtimeLogged, sc.LastMtime(sc.Name()).Unix(), "ServiceContract")
		}()
		return nil, sc.DoCacheUpdate(sc.Name(), sc.realUpdate)
	})
	return err, shared
}

func (sc *serviceContractCache) realUpdate() (map[string]time.Time, int64, error) {
	start := time.Now()
	values, err := sc.Store().GetMoreServiceContracts(sc.IsFirstUpdate(), sc.LastFetchTime())
	if err != nil {
		log.Errorf("[Cache][ServiceContract] update service_contract err: %s", err.Error())
		return nil, 0, err
	}

	lastMtimes, update, del := sc.setContracts(values)
	costTime := time.Since(start)
	log.Info(
		"[Cache][ServiceContract] get more service_contract", zap.Int("upsert", update), zap.Int("delete", del),
		zap.Time("last", sc.LastMtime(sc.Name())), zap.Duration("used", costTime))
	return lastMtimes, int64(len(values)), err
}

func (sc *serviceContractCache) setContracts(values []*model.EnrichServiceContract) (map[string]time.Time, int, int) {
	var (
		upsert, del int
		lastMtime   time.Time
	)
	for i := range values {
		item := values[i]
		namespace := item.Namespace
		service := item.Service

		if _, ok := sc.contracts.Load(namespace); !ok {
			sc.contracts.Store(namespace, utils.NewSyncMap[string, *utils.SyncMap[string, *model.EnrichServiceContract]]())
		}
		namespaceVal, _ := sc.contracts.Load(namespace)

		if _, ok := namespaceVal.Load(service); !ok {
			namespaceVal.Store(service, utils.NewSyncMap[string, *model.EnrichServiceContract]())
		}

		serviceVal, _ := namespaceVal.Load(service)
		if !item.Valid {
			del++
			sc.data.Delete(item.GetCacheKey())
			serviceVal.Delete(item.ID)
			continue
		}

		upsert++
		sc.data.Store(item.GetCacheKey(), item)
		serviceVal.Store(item.ID, item)
	}
	return map[string]time.Time{
		sc.Name(): lastMtime,
	}, upsert, del
}

// Clear
func (sc *serviceContractCache) Clear() error {
	sc.data = utils.NewSyncMap[string, *model.EnrichServiceContract]()
	sc.contracts = utils.NewSyncMap[string, *utils.SyncMap[string, *utils.SyncMap[string, *model.EnrichServiceContract]]]()
	return nil
}

// Name
func (sc *serviceContractCache) Name() string {
	return ServiceContractName
}

// forceQueryUpdate 为了确保读取的数据是最新的，这里需要做一个强制 update 的动作进行数据读取处理
func (sc *serviceContractCache) forceQueryUpdate() error {
	err, shared := sc.singleUpdate()
	// shared == true，表示当前已经有正在 update 执行的任务，这个任务不一定能够读取到最新的数据
	// 为了避免读取到脏数据，在发起一次 singleUpdate
	if shared {
		naminglog.Debug("[Server][ServiceContract][Query] force query update from store")
		err, _ = sc.singleUpdate()
	}
	return err
}

func (sc *serviceContractCache) Get(req *model.ServiceContract) *model.EnrichServiceContract {
	ret, _ := sc.data.Load(req.GetCacheKey())
	return ret
}

// Query .
func (sc *serviceContractCache) Query(filter map[string]string, offset, limit uint32) ([]*model.EnrichServiceContract, uint32, error) {
	if err := sc.forceQueryUpdate(); err != nil {
		return nil, 0, err
	}

	values := make([]*model.EnrichServiceContract, 0, 64)

	searchNamespace := filter["namespace"]
	searchService := filter["service"]
	searchName := filter["name"]
	searchProtocol := filter["protocol"]
	searchVersion := filter["version"]

	sc.contracts.ReadRange(func(namespace string, services *utils.SyncMap[string, *utils.SyncMap[string, *model.EnrichServiceContract]]) {
		if searchNamespace != "" {
			if !utils.IsWildMatch(namespace, searchNamespace) {
				return
			}
		}

		services.ReadRange(func(service string, contracts *utils.SyncMap[string, *model.EnrichServiceContract]) {
			if searchService != "" {
				if !utils.IsWildMatch(service, searchService) {
					return
				}
			}
			contracts.ReadRange(func(_ string, val *model.EnrichServiceContract) {
				if searchName != "" {
					names := strings.Split(searchName, ",")
					for i := range names {
						if !utils.IsWildMatch(val.Name, names[i]) {
							return
						}
					}
				}
				if searchProtocol != "" {
					if !utils.IsWildMatch(val.Protocol, searchProtocol) {
						return
					}
				}
				if searchVersion != "" {
					if !utils.IsWildMatch(val.Version, searchVersion) {
						return
					}
				}
				values = append(values, val)
				return
			})
		})
	})

	sort.Slice(values, func(i, j int) bool {
		return values[j].ModifyTime.Before(values[i].ModifyTime)
	})
	retVal, total := sc.toPage(values, offset, limit)
	return retVal, total, nil
}

// ListVersions .
func (sc *serviceContractCache) ListVersions(searchService, searchNamespace string) []*model.EnrichServiceContract {
	values := make([]*model.EnrichServiceContract, 0, 64)
	sc.contracts.Range(func(namespace string, services *utils.SyncMap[string, *utils.SyncMap[string, *model.EnrichServiceContract]]) {
		if searchNamespace != namespace {
			return
		}

		services.Range(func(service string, contracts *utils.SyncMap[string, *model.EnrichServiceContract]) {
			if searchService != service {
				return
			}
			contracts.Range(func(_ string, val *model.EnrichServiceContract) {
				values = append(values, &model.EnrichServiceContract{
					ServiceContract: &model.ServiceContract{
						ID:         val.ID,
						Namespace:  val.Namespace,
						Service:    val.Service,
						Name:       val.Name,
						Protocol:   val.Protocol,
						Version:    val.Version,
						Revision:   val.Revision,
						CreateTime: val.CreateTime,
						ModifyTime: val.ModifyTime,
					},
				})
			})
		})
	})
	sort.Slice(values, func(i, j int) bool {
		return values[j].ModifyTime.Before(values[i].ModifyTime)
	})
	return values
}

func (sc *serviceContractCache) toPage(values []*model.EnrichServiceContract, offset,
	limit uint32) ([]*model.EnrichServiceContract, uint32) {

	// 所有符合条件的服务数量
	amount := uint32(len(values))
	// 判断 offset 和 limit 是否允许返回对应的服务
	if offset >= amount || limit == 0 {
		return nil, amount
	}

	endIdx := offset + limit
	if endIdx > amount {
		endIdx = amount
	}
	return values[offset:endIdx], amount
}
