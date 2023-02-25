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
	"sync"

	"github.com/polarismesh/polaris/common/model"
)

type strategyBucket struct {
	lock       sync.RWMutex
	strategies map[string]*model.StrategyDetailCache
}

func (s *strategyBucket) save(key string, val *model.StrategyDetailCache) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.strategies[key] = val
}

func (s *strategyBucket) delete(key string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.strategies, key)
}

func (s *strategyBucket) get(key string) (*model.StrategyDetailCache, bool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	val, ok := s.strategies[key]
	return val, ok
}

type strategyIdBucket struct {
	lock sync.RWMutex
	ids  map[string]struct{}
}

func (s *strategyIdBucket) save(key string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.ids[key] = struct{}{}
}

func (s *strategyIdBucket) delete(key string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.ids, key)
}

func (s *strategyIdBucket) toSlice() []string {
	s.lock.RLock()
	defer s.lock.RUnlock()

	ret := make([]string, 0, len(s.ids))

	for k := range s.ids {
		ret = append(ret, k)
	}

	return ret
}

type strategyLinkBucket struct {
	lock       sync.RWMutex
	strategies map[string]*strategyIdBucket
}

func (s *strategyLinkBucket) save(linkId, strategyId string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.strategies[linkId]; !ok {
		s.strategies[linkId] = &strategyIdBucket{
			lock: sync.RWMutex{},
			ids:  make(map[string]struct{}),
		}
	}

	s.strategies[linkId].save(strategyId)
}

func (s *strategyLinkBucket) deleteAllLink(linkId string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.strategies, linkId)
}

func (s *strategyLinkBucket) delete(linkId, strategyID string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	v, ok := s.strategies[linkId]
	if !ok {
		return
	}

	v.delete(strategyID)

	if len(v.ids) == 0 {
		delete(s.strategies, linkId)
	}
}

func (s *strategyLinkBucket) get(key string) ([]string, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	val, ok := s.strategies[key]
	if !ok {
		return []string{}, ok
	}
	return val.toSlice(), ok
}
