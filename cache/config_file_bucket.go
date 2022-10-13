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
	"sync"
	"time"

	"github.com/polarismesh/polaris/common/model"
)

type (
	configGroupEntry struct {
		item          *model.ConfigFileGroup
		expireTimeSec int64
		empty         bool
	}
)

func newConfigFileGroupBucket() *configFileGroupBucket {
	b := &configFileGroupBucket{
		id2groups:   map[uint64]*configGroupEntry{},
		name2groups: map[string]*subConfigFileGroupBucket{},
	}

	return b
}

type configFileGroupBucket struct {
	lock        sync.RWMutex
	name2groups map[string]*subConfigFileGroupBucket

	idlock    sync.RWMutex
	id2groups map[uint64]*configGroupEntry
}

func (b *configFileGroupBucket) saveGroupById(id uint64, item *model.ConfigFileGroup) {
	b.idlock.Lock()
	defer b.idlock.Unlock()

	b.id2groups[id] = &configGroupEntry{
		item:          item,
		empty:         item == nil,
		expireTimeSec: time.Now().Add(time.Minute).Unix(),
	}

	if item == nil {
		b.id2groups[id].expireTimeSec = time.Now().Add(10 * time.Second).Unix()
	}
}

func (b *configFileGroupBucket) saveGroup(namespace, group string, item *model.ConfigFileGroup) {
	func() {
		b.lock.Lock()
		defer b.lock.Unlock()

		if _, ok := b.name2groups[namespace]; !ok {
			b.name2groups[namespace] = &subConfigFileGroupBucket{
				name2groups: make(map[string]*configGroupEntry),
			}
		}
	}()

	b.lock.RLock()
	defer b.lock.RUnlock()

	sub := b.name2groups[namespace]
	sub.saveGroup(namespace, group, item)
}

func (b *configFileGroupBucket) getGroupByName(namespace, group string) *model.ConfigFileGroup {
	b.lock.RLock()
	defer b.lock.RUnlock()

	sub, ok := b.name2groups[namespace]
	if !ok {
		return nil
	}

	return sub.getGroupByName(group)
}

func (b *configFileGroupBucket) getGroupById(id uint64) *model.ConfigFileGroup {
	b.lock.RLock()
	defer b.lock.RUnlock()

	item, ok := b.id2groups[id]
	if !ok {
		return nil
	}

	return item.item
}

func (b *configFileGroupBucket) runCleanExpire(ctx context.Context, interval time.Duration, expireTime int64) {
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			cleanId2Group := func() {
				b.idlock.Lock()
				defer b.idlock.Unlock()

				tn := time.Now().Unix()
				for i := range b.id2groups {
					item := b.id2groups[i]
					if item.expireTimeSec < tn {
						delete(b.id2groups, i)
					}
				}
			}

			cleanName2Group := func() {
				b.lock.RLock()
				defer b.lock.RUnlock()

				for namespace := range b.name2groups {
					sub := b.name2groups[namespace]
					sub.cleanExpire(expireTime)
				}
			}

			cleanId2Group()
			cleanName2Group()
		case <-ctx.Done():
			return
		}
	}
}

func (b *configFileGroupBucket) clean() {
	b.name2groups = map[string]*subConfigFileGroupBucket{}
}

type subConfigFileGroupBucket struct {
	lock sync.RWMutex

	name2groups map[string]*configGroupEntry
}

func (b *subConfigFileGroupBucket) saveGroup(namespace, group string, item *model.ConfigFileGroup) {
	b.lock.Lock()
	defer b.lock.Unlock()

	entry := &configGroupEntry{
		item:          item,
		empty:         item == nil,
		expireTimeSec: time.Now().Add(time.Minute).Unix(),
	}

	if entry.empty {
		entry.item = &model.ConfigFileGroup{
			Name:      group,
			Namespace: namespace,
		}
		entry.expireTimeSec = time.Now().Add(10 * time.Second).Unix()
	}

	b.name2groups[group] = entry
}

func (b *subConfigFileGroupBucket) getGroupByName(group string) *model.ConfigFileGroup {
	b.lock.RLock()
	defer b.lock.RUnlock()

	entry, ok := b.name2groups[group]
	if !ok {
		return nil
	}
	if entry.empty {
		return nil
	}
	return entry.item
}

func (b *subConfigFileGroupBucket) cleanExpire(expire int64) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	tn := time.Now().Unix()
	for i := range b.name2groups {
		item := b.name2groups[i]

		if item.expireTimeSec < tn {
			delete(b.name2groups, item.item.Name)
		}
	}
}
