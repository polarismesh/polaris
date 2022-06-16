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
	"sort"
	"sync"
	"time"

	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

func init() {
	RegisterCache(ClientName, CacheClient)
}

const (
	// ClientName client cache name
	ClientName = "client"
)

// ClientIterProc client iter proc func
type ClientIterProc func(key string, value *model.Client) bool

// ClientCache 客户端的 Cache 接口
type ClientCache interface {
	Cache

	// GetClient
	GetClient(id string) *model.Client

	// IteratorClients 迭代
	IteratorClients(iterProc ClientIterProc)

	// GetClientsByFilter Query client information
	GetClientsByFilter(filters map[string]string, offset, limit uint32) (uint32, []*model.Client, error)
}

// clientCache 客户端缓存的类
type clientCache struct {
	*baseCache

	storage         store.Store
	lastMtime       int64
	lastMtimeLogged int64
	firstUpdate     bool
	ids             *sync.Map // instanceid -> instance
	singleFlight    *singleflight.Group
	lastUpdateTime  time.Time
}

// name 获取资源名称
func (cc *clientCache) name() string {
	return InstanceName
}

// LastMtime 最后一次更新时间
func (cc *clientCache) LastMtime() time.Time {
	return time.Unix(cc.lastMtime, 0)
}

// newClientCache 新建一个clientCache
func newClientCache(storage store.Store) *clientCache {
	return &clientCache{
		baseCache: newBaseCache(),
		storage:   storage,
	}
}

// initialize 初始化函数
func (cc *clientCache) initialize(opt map[string]interface{}) error {
	cc.singleFlight = new(singleflight.Group)
	cc.ids = new(sync.Map)
	cc.lastMtime = 0
	cc.lastUpdateTime = time.Unix(0, 0)
	cc.firstUpdate = true
	return nil
}

// update 更新缓存函数
func (cc *clientCache) update(arg Args) error {
	// 一分钟update一次
	timeDiff := time.Now().Sub(cc.lastUpdateTime).Minutes()
	if !cc.firstUpdate && 1 > timeDiff {
		log.CacheScope().Debug("[Cache][Client] update get storage ignore", zap.Float64("time-diff", timeDiff))
		return nil
	}

	// 多个线程竞争，只有一个线程进行更新
	_, err, _ := cc.singleFlight.Do(InstanceName, func() (interface{}, error) {
		defer func() {
			cc.lastMtimeLogged = logLastMtime(cc.lastMtimeLogged, cc.lastMtime, "Client")
		}()
		return nil, cc.realUpdate(arg)
	})
	return err
}

func (cc *clientCache) realUpdate(arg Args) error {
	// 拉取diff前的所有数据
	
	start := time.Now()
	lastMtime := cc.LastMtime().Add(arg.StoreTimeRollbackSec)

	clients, err := cc.storage.GetMoreClients(lastMtime, cc.firstUpdate)
	if err != nil {
		log.CacheScope().Errorf("[Cache][Client] update get storage more err: %s", err.Error())
		return err
	}

	cc.firstUpdate = false
	update, del := cc.setClients(clients)
	timeDiff := time.Since(start)
	if timeDiff > 1*time.Second {
		log.CacheScope().Info("[Cache][Client] get more clients",
			zap.Int("update", update), zap.Int("delete", del),
			zap.Time("last", lastMtime), zap.Duration("used", time.Since(start)))
	}

	cc.lastUpdateTime = time.Now()

	return nil
}

// setClients 保存client到内存中
// 返回：更新个数，删除个数
func (cc *clientCache) setClients(clients map[string]*model.Client) (int, int) {
	if len(clients) == 0 {
		return 0, 0
	}

	lastMtime := cc.lastMtime
	update := 0
	del := 0
	progress := 0
	for _, client := range clients {
		progress++
		if progress%50000 == 0 {
			log.CacheScope().Infof("[Cache][Client] set clients progress: %d / %d", progress, len(clients))
		}
		modifyTime := client.ModifyTime().Unix()
		if lastMtime < modifyTime {
			lastMtime = modifyTime
		}
		id := client.Proto().GetId().GetValue()
		_, itemExist := cc.ids.Load(id)
		// 待删除的instance
		if !client.Valid() {
			del++
			cc.ids.Delete(id)
			if itemExist {
				cc.manager.onEvent(client, EventDeleted)
			}
			continue
		}
		// 有修改或者新增的数据
		update++
		cc.ids.Store(id, client)
		if !itemExist {
			cc.manager.onEvent(client, EventCreated)
		} else {
			cc.manager.onEvent(client, EventUpdated)
		}
	}

	if cc.lastMtime != lastMtime {
		log.CacheScope().Infof("[Cache][Client] Client lastMtime update from %s to %s",
			time.Unix(cc.lastMtime, 0), time.Unix(lastMtime, 0))
		cc.lastMtime = lastMtime
	}
	return update, del
}

// clear
//  @return error
func (cc *clientCache) clear() error {
	cc.ids = new(sync.Map)
	cc.lastMtime = 0
	return nil
}

// GetClient
//  @param id
//  @return *model.Client
func (cc *clientCache) GetClient(id string) *model.Client {
	if id == "" {
		return nil
	}

	value, ok := cc.ids.Load(id)
	if !ok {
		return nil
	}

	return value.(*model.Client)
}

// IteratorClients 迭代
func (cc *clientCache) IteratorClients(iterProc ClientIterProc) {
	cc.ids.Range(func(key, value interface{}) bool {
		return iterProc(key.(string), value.(*model.Client))
	})
}

// GetClientsByFilter Query client information
func (cc *clientCache) GetClientsByFilter(filters map[string]string, offset, limit uint32) (uint32,
	[]*model.Client, error) {

	ret := make([]*model.Client, 0, 16)

	host, hasHost := filters["host"]
	clientType, hasType := filters["type"]
	version, hasVer := filters["version"]

	cc.IteratorClients(func(_ string, value *model.Client) bool {
		if hasHost && value.Proto().GetHost().GetValue() != host {
			return true
		}
		if hasType && value.Proto().GetType().String() != clientType {
			return true
		}
		if hasVer && value.Proto().GetVersion().String() != version {
			return true
		}

		ret = append(ret, value)
		return true
	})

	amount := uint32(len(ret))
	return amount, doClientPage(ret, offset, limit), nil
}

// doClientPage 进行分页, 仅用于控制台查询时的排序
func doClientPage(ret []*model.Client, offset, limit uint32) []*model.Client {

	clients := make([]*model.Client, 0, len(ret))

	beginIndex := offset
	endIndex := beginIndex + limit
	totalCount := uint32(len(ret))

	if totalCount == 0 {
		return clients
	}
	if beginIndex >= endIndex {
		return clients
	}
	if beginIndex >= totalCount {
		return clients
	}
	if endIndex > totalCount {
		endIndex = totalCount
	}
	for i := range ret {
		clients = append(clients, ret[i])
	}

	sort.Slice(clients, func(i, j int) bool {
		return clients[i].ModifyTime().After(clients[j].ModifyTime())
	})

	return clients[beginIndex:endIndex]

}
