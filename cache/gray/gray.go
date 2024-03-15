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
	"encoding/json"
	"time"

	regexp "github.com/dlclark/regexp2"
	"github.com/golang/protobuf/jsonpb"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	types "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

var (
	_ types.GrayCache = (*grayCache)(nil)
)

type grayCache struct {
	*types.BaseCache
	storage       store.Store
	grayResources *utils.SyncMap[string, []*apimodel.ClientLabel]
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
	gc.grayResources = utils.NewSyncMap[string, []*apimodel.ClientLabel]()
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
	lastMtimes, err := gc.setGrayResources(grayResources)
	if err != nil {
		return nil, 0, err
	}
	log.Info("[Cache][Gray] get more gray resource", zap.Int("total", len(grayResources)))
	return lastMtimes, int64(len(grayResources)), nil
}

func (gc *grayCache) setGrayResources(grayResources []*model.GrayResource) (map[string]time.Time, error) {
	lastMtime := gc.LastMtime(gc.Name()).Unix()
	for _, grayResource := range grayResources {
		modifyUnix := grayResource.ModifyTime.Unix()
		if modifyUnix > lastMtime {
			lastMtime = modifyUnix
		}
		clientLabels := []*apimodel.ClientLabel{}
		jsonDecoder := json.NewDecoder(bytes.NewBuffer([]byte(grayResource.MatchRule)))
		// read open bracket
		if _, err := jsonDecoder.Token(); err != nil {
			return nil, err
		}
		for jsonDecoder.More() {
			protoMessage := &apimodel.ClientLabel{}
			if err := jsonpb.UnmarshalNext(jsonDecoder, protoMessage); err != nil {
				return nil, err
			}
			clientLabels = append(clientLabels, protoMessage)
		}
		gc.grayResources.Store(grayResource.Name, clientLabels)
	}

	return map[string]time.Time{
		gc.Name(): time.Unix(lastMtime, 0),
	}, nil
}

// Clear clear cache
func (gc *grayCache) Clear() error {
	gc.BaseCache.Clear()
	gc.grayResources = utils.NewSyncMap[string, []*apimodel.ClientLabel]()
	return nil
}

// Name return gray name
func (gc *grayCache) Name() string {
	return types.GrayName
}

// GetGrayRule get gray rule
func (gc *grayCache) GetGrayRule(name string) []*apimodel.ClientLabel {
	val, ok := gc.grayResources.Load(name)
	if !ok {
		return nil
	}
	return val
}

func (gc *grayCache) HitGrayRule(name string, labels map[string]string) bool {
	rule, ok := gc.grayResources.Load(name)
	if !ok {
		return false
	}

	return grayMatch(rule, labels)
}

func grayMatch(rule []*apimodel.ClientLabel, labels map[string]string) bool {
	for i := range rule {
		clientLabel := rule[i]
		labelKey := clientLabel.Key
		actualVal, ok := labels[labelKey]
		if !ok {
			return false
		}
		isMatch := utils.MatchString(actualVal, clientLabel.Value, func(s string) *regexp.Regexp {
			regex, err := regexp.Compile(s, regexp.RE2)
			if err != nil {
				log.Error("[Cache][Gray] compile regex failed", zap.Error(err))
				return nil
			}
			return regex
		})
		if !isMatch {
			return false
		}
	}
	return true
}
