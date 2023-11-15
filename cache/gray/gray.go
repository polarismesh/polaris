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

package gray

import (
	"bytes"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	"github.com/golang/protobuf/jsonpb"
	types "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
)

var (
	_ types.GrayCache = (*grayCache)(nil)
)

type grayCache struct {
	*types.BaseCache
	storage       store.Store
	grayResources *utils.SyncMap[string, *apimodel.MatchTerm]
	updater       *singleflight.Group
}

// NewGrayCache create gray cache obj
func NewGrayCache(storage store.Store, cacheMgr types.CacheManager) types.GrayCache {
	return &grayCache{
		BaseCache: types.NewBaseCache(storage, cacheMgr),
		storage:   storage,
	}
}

// Initialize init gray cache
func (gc *grayCache) Initialize(opt map[string]interface{}) error {
	gc.grayResources = utils.NewSyncMap[string, *apimodel.MatchTerm]()
	gc.updater = &singleflight.Group{}
	return nil
}

// Update update cache
func (gc *grayCache) Update() error {
	// 多个线程竞争，只有一个线程进行更新
	_, err, _ := gc.updater.Do(gc.Name(), func() (interface{}, error) {
		return nil, gc.DoCacheUpdate(gc.Name(), gc.realUpdate)
	})
	return err
}

func (gc *grayCache) realUpdate() (map[string]time.Time, int64, error) {
	grayResources, err := gc.storage.GetMoreGrayResouces(gc.IsFirstUpdate(), gc.LastFetchTime())

	if err != nil {
		log.Error("[Cache][Gray] get storage more", zap.Error(err))
		return nil, -1, err
	}
	if len(grayResources) == 0 {
		return nil, 0, nil
	}
	lastMtimes := gc.setGrayResources(grayResources)
	log.Info("[Cache][Gray] get more gray resource",
		zap.Int("total", len(grayResources)))
	return lastMtimes, int64(len(grayResources)), nil
}

func (gc *grayCache) setGrayResources(grayResources []*model.GrayResource) map[string]time.Time {
	lastMtime := gc.LastMtime(gc.Name()).Unix()
	for _, grayResource := range grayResources {
		modifyUnix := grayResource.ModifyTime.Unix()
		if modifyUnix > lastMtime {
			lastMtime = modifyUnix
		}
		grayRule := &apimodel.MatchTerm{}
		reader := bytes.NewReader([]byte(grayResource.MatchRule))
		err := jsonpb.Unmarshal(reader, grayRule)
		if err != nil {
			log.Error("[Cache][Gray] setGrayResources unmarshal gray rule fail.",
				zap.String("name", grayResource.Name), zap.Error(err))
			continue
		}
		gc.grayResources.Store(grayResource.Name, grayRule)
	}

	return map[string]time.Time{
		gc.Name(): time.Unix(lastMtime, 0),
	}
}

// Clear clear cache
func (gc *grayCache) Clear() error {
	gc.BaseCache.Clear()
	gc.grayResources = utils.NewSyncMap[string, *apimodel.MatchTerm]()
	return nil
}

// Name return gray name
func (gc *grayCache) Name() string {
	return types.GrayName
}

// GetGrayRule get gray rule
func (gc *grayCache) GetGrayRule(name string) *apimodel.MatchTerm {
	val, ok := gc.grayResources.Load(name)
	if !ok {
		return nil
	}
	return val
}
