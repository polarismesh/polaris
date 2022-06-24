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

var _ ClientCache = (*clientCache)(nil)

// ClientIterProc client iter proc func
type ClientIterProc func(key string, value *model.Client) bool

// ClientCache 客户端的 Cache 接口
type ClientCache interface {
	Cache

	// GetClient get client
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
	clients         map[string]*model.Client // instance id -> instance
	lock            sync.RWMutex
	singleFlight    *singleflight.Group
	lastUpdateTime  time.Time
}

// name 获取资源名称
func (c *clientCache) name() string {
	return InstanceName
}

// LastMtime 最后一次更新时间
func (c *clientCache) LastMtime() time.Time {
	return time.Unix(c.lastMtime, 0)
}

// newClientCache 新建一个clientCache
func newClientCache(storage store.Store) *clientCache {
	return &clientCache{
		baseCache: newBaseCache(),
		storage:   storage,
		clients:   map[string]*model.Client{},
	}
}

// initialize 初始化函数
func (c *clientCache) initialize(_ map[string]interface{}) error {
	c.singleFlight = &singleflight.Group{}
	c.lastMtime = 0
	c.lastUpdateTime = time.Unix(0, 0)
	c.firstUpdate = true

	return nil
}

// update 更新缓存函数
func (c *clientCache) update(storeRollbackSec time.Duration) error {
	// 一分钟update一次
	timeDiff := time.Now().Sub(c.lastUpdateTime).Minutes()
	if !c.firstUpdate && 1 > timeDiff {
		log.CacheScope().Debug("[Cache][Client] update get storage ignore", zap.Float64("time-diff", timeDiff))
		return nil
	}

	// 多个线程竞争，只有一个线程进行更新
	_, err, _ := c.singleFlight.Do(InstanceName, func() (interface{}, error) {
		defer func() {
			c.lastMtimeLogged = logLastMtime(c.lastMtimeLogged, c.lastMtime, "Client")
		}()

		return nil, c.realUpdate(storeRollbackSec)
	})

	return err
}

func (c *clientCache) realUpdate(storeRollbackSec time.Duration) error {
	// 拉取diff前的所有数据
	start := time.Now()
	lastMtime := c.LastMtime().Add(storeRollbackSec)
	clients, err := c.storage.GetMoreClients(lastMtime, c.firstUpdate)
	if err != nil {
		log.CacheScope().Errorf("[Cache][Client] update get storage more err: %s", err.Error())
		return err
	}

	c.firstUpdate = false
	update, del := c.setClients(clients)
	timeDiff := time.Since(start)
	if timeDiff > 1*time.Second {
		log.CacheScope().Info("[Cache][Client] get more clients",
			zap.Int("update", update), zap.Int("delete", del),
			zap.Time("last", lastMtime), zap.Duration("used", time.Since(start)))
	}

	c.lastUpdateTime = time.Now()

	return nil
}

func (c *clientCache) getClient(id string) (*model.Client, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	client, ok := c.clients[id]
	return client, ok
}

func (c *clientCache) deleteClient(id string) {
	c.lock.Lock()
	delete(c.clients, id)
	c.lock.Unlock()
}

func (c *clientCache) storeClient(id string, client *model.Client) {
	c.lock.Lock()
	c.clients[id] = client
	c.lock.Unlock()
}

// setClients 保存client到内存中
// 返回：更新个数，删除个数
func (c *clientCache) setClients(clients map[string]*model.Client) (int, int) {
	if len(clients) == 0 {
		return 0, 0
	}

	lastMtime := c.lastMtime
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
		_, itemExist := c.getClient(id)
		// 待删除的instance
		if !client.Valid() {
			del++
			c.deleteClient(id)
			if itemExist {
				c.manager.onEvent(client, EventDeleted)
			}

			continue
		}

		// 有修改或者新增的数据
		update++
		c.storeClient(id, client)
		if !itemExist {
			c.manager.onEvent(client, EventCreated)
		} else {
			c.manager.onEvent(client, EventUpdated)
		}
	}

	if c.lastMtime != lastMtime {
		log.CacheScope().Infof("[Cache][Client] Client lastMtime update from %s to %s",
			time.Unix(c.lastMtime, 0), time.Unix(lastMtime, 0))
		c.lastMtime = lastMtime
	}

	return update, del
}

// clear
//  @return error
func (c *clientCache) clear() error {
	c.lock.Lock()
	c.clients = map[string]*model.Client{}
	c.lock.Unlock()

	c.lastMtime = 0
	return nil
}

// GetClient get client
// @param id
// @return *model.Client
func (c *clientCache) GetClient(id string) *model.Client {
	if id == "" {
		return nil
	}

	value, ok := c.getClient(id)
	if !ok {
		return nil
	}

	return value
}

// IteratorClients 迭代
func (c *clientCache) IteratorClients(iterProc ClientIterProc) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	for key := range c.clients {
		if !iterProc(key, c.clients[key]) {
			break
		}
	}
}

// GetClientsByFilter Query client information
func (c *clientCache) GetClientsByFilter(filters map[string]string, offset, limit uint32) (uint32,
	[]*model.Client, error) {

	ret := make([]*model.Client, 0, 16)
	host, hasHost := filters["host"]
	clientType, hasType := filters["type"]
	version, hasVer := filters["version"]

	c.IteratorClients(func(_ string, value *model.Client) bool {
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
	if totalCount == 0 || beginIndex >= endIndex || beginIndex >= totalCount {
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
