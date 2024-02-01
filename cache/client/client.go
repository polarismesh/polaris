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

package cache_client

import (
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"go.etcd.io/bbolt"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	"github.com/golang/protobuf/proto"
	types "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/common/eventhub"
	"github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/store"
	"github.com/polarismesh/specification/source/go/api/v1/service_manage"
)

var (
	_ types.ClientCache = (*clientCache)(nil)
)

// clientCache 客户端缓存的类
type clientCache struct {
	*types.BaseCache

	storage         store.Store
	lastMtimeLogged int64
	clientCount     int64
	// valueCache save ConfigFileRelease.Content into local file to reduce memory use
	clients        *bbolt.DB
	lock           sync.RWMutex
	singleFlight   *singleflight.Group
	lastUpdateTime time.Time
}

// name 获取资源名称
func (c *clientCache) Name() string {
	return types.ClientName
}

// LastMtime 最后一次更新时间
func (c *clientCache) LastMtime() time.Time {
	return c.BaseCache.LastMtime(c.Name())
}

// NewClientCache 新建一个clientCache
func NewClientCache(storage store.Store, cacheMgr types.CacheManager) types.ClientCache {
	return &clientCache{
		BaseCache: types.NewBaseCache(storage, cacheMgr),
		storage:   storage,
	}
}

// initialize 初始化函数
func (c *clientCache) Initialize(opt map[string]interface{}) error {
	c.singleFlight = &singleflight.Group{}
	c.lastUpdateTime = time.Unix(0, 0)
	var err error
	c.clients, err = c.openBoltCache(opt)
	if err != nil {
		return err
	}
	return nil
}

func (c *clientCache) openBoltCache(opt map[string]interface{}) (*bbolt.DB, error) {
	path, _ := opt["cachePath"].(string)
	if path == "" {
		path = "./data/cache/client"
	}
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return nil, err
	}
	dbFile := filepath.Join(path, "client_info.bolt")
	_ = os.Remove(dbFile)
	valueCache, err := bbolt.Open(dbFile, os.ModePerm, &bbolt.Options{})
	if err != nil {
		return nil, err
	}
	return valueCache, nil
}

// update 更新缓存函数
func (c *clientCache) Update() error {
	// 多个线程竞争，只有一个线程进行更新
	_, err, _ := c.singleFlight.Do(c.Name(), func() (interface{}, error) {
		defer func() {
			c.lastMtimeLogged = types.LogLastMtime(c.lastMtimeLogged, c.LastMtime().Unix(), "Client")
			c.reportMetricsInfo()
		}()
		return nil, c.DoCacheUpdate(c.Name(), c.realUpdate)
	})

	return err
}

func (c *clientCache) realUpdate() (map[string]time.Time, int64, error) {
	// 拉取diff前的所有数据
	start := time.Now()
	clients, err := c.storage.GetMoreClients(c.LastFetchTime(), c.IsFirstUpdate())
	if err != nil {
		log.Errorf("[Cache][Client] update get storage more err: %s", err.Error())
		return nil, -1, err
	}
	timeDiff := time.Since(start)
	lastMtimes, update, del := c.setClients(clients)
	if timeDiff > 1*time.Second {
		log.Info("[Cache][Client] get more clients",
			zap.Int("update", update), zap.Int("delete", del),
			zap.Time("last", c.LastMtime()), zap.Duration("used", time.Since(start)))
	}

	c.lastUpdateTime = time.Now()
	return lastMtimes, int64(len(clients)), nil
}

func (c *clientCache) getClient(id string) (*model.Client, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	var client *model.Client
	c.clients.View(func(tx *bbolt.Tx) error {
		val := tx.Bucket([]byte(id))
		if val == nil {
			return nil
		}
		saveData := val.Get([]byte("client"))
		msg := &service_manage.Client{}
		if err := proto.Unmarshal(saveData, msg); err != nil {
			return err
		}
		client = model.NewClient(msg)
		return nil
	})
	return client, client != nil
}

func (c *clientCache) deleteClient(id string) {
	atomic.AddInt64(&c.clientCount, -1)
	c.clients.Update(func(tx *bbolt.Tx) error {
		return tx.DeleteBucket([]byte(id))
	})
}

func (c *clientCache) storeClient(id string, client *model.Client) {
	atomic.AddInt64(&c.clientCount, 1)
	c.clients.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(id))
		if err != nil {
			return err
		}
		saveData, err := proto.Marshal(client.Proto())
		if err != nil {
			return err
		}
		bucket.Put([]byte("client"), saveData)
		return nil
	})
}

// setClients 保存client到内存中
// 返回：更新个数，删除个数
func (c *clientCache) setClients(clients map[string]*model.Client) (map[string]time.Time, int, int) {
	if len(clients) == 0 {
		return nil, 0, 0
	}

	lastMtime := c.LastMtime().Unix()
	update := 0
	del := 0
	progress := 0
	for _, client := range clients {
		progress++
		if progress%50000 == 0 {
			log.Infof("[Cache][Client] set clients progress: %d / %d", progress, len(clients))
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
				_ = eventhub.Publish(eventhub.CacheClientEventTopic, &eventhub.CacheClientEvent{
					Client:    client,
					EventType: eventhub.EventDeleted,
				})
			}
			continue
		}

		// 有修改或者新增的数据
		update++
		c.storeClient(id, client)
		if !itemExist {
			_ = eventhub.Publish(eventhub.CacheClientEventTopic, &eventhub.CacheClientEvent{
				Client:    client,
				EventType: eventhub.EventCreated,
			})
		} else {
			_ = eventhub.Publish(eventhub.CacheClientEventTopic, &eventhub.CacheClientEvent{
				Client:    client,
				EventType: eventhub.EventUpdated,
			})
		}
	}

	return map[string]time.Time{
		c.Name(): time.Unix(lastMtime, 0),
	}, update, del
}

// clear
func (c *clientCache) Clear() error {
	c.BaseCache.Clear()
	return nil
}

// GetClient get client
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
func (c *clientCache) IteratorClients(iterProc types.ClientIterProc) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	c.clients.View(func(tx *bbolt.Tx) error {
		cursor := tx.Cursor()
		for k, _ := cursor.First(); k != nil; k, _ = cursor.Next() {
			val := tx.Bucket(k)
			if val == nil {
				continue
			}
			saveData := val.Get([]byte("client"))
			msg := &service_manage.Client{}
			if err := proto.Unmarshal(saveData, msg); err != nil {
				return err
			}
			client := model.NewClient(msg)
			if !iterProc(string(k), client) {
				break
			}
		}
		return nil
	})
}

// GetClientsByFilter Query client information
func (c *clientCache) GetClientsByFilter(filters map[string]string, offset, limit uint32) (uint32,
	[]*model.Client, error) {
	var (
		ret                 = make([]*model.Client, 0, 16)
		host, hasHost       = filters["host"]
		clientType, hasType = filters["type"]
		version, hasVer     = filters["version"]
		id, hasId           = filters["id"]
	)
	c.IteratorClients(func(_ string, value *model.Client) bool {
		if hasHost && value.Proto().GetHost().GetValue() != host {
			return true
		}
		if hasType && value.Proto().GetType().String() != clientType {
			return true
		}
		if hasVer && value.Proto().GetVersion().GetValue() != version {
			return true
		}
		if hasId && value.Proto().GetId().GetValue() != id {
			return true
		}

		ret = append(ret, value)
		return true
	})

	return uint32(len(ret)), doClientPage(ret, offset, limit), nil
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

	clients = append(clients, ret...)
	sort.Slice(clients, func(i, j int) bool {
		return clients[i].ModifyTime().After(clients[j].ModifyTime())
	})

	return clients[beginIndex:endIndex]
}
