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
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"github.com/stretchr/testify/assert"

	types "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store/mock"
)

func newTestClientCache(t *testing.T) (*gomock.Controller, *mock.MockStore, *clientCache) {
	ctl := gomock.NewController(t)

	var cacheMgr types.CacheManager

	storage := mock.NewMockStore(ctl)
	rlc := NewClientCache(storage, cacheMgr)
	storage.EXPECT().GetUnixSecond(gomock.Any()).AnyTimes().Return(time.Now().Unix(), nil)
	var opt map[string]interface{}
	_ = rlc.Initialize(opt)
	return ctl, storage, rlc.(*clientCache)
}

func mockClients(cnt int) map[string]*model.Client {
	ret := make(map[string]*model.Client)

	for i := 0; i < cnt; i++ {

		id := utils.NewUUID()

		ret[id] = model.NewClient(&apiservice.Client{
			Host:    utils.NewStringValue(fmt.Sprintf("127.0.0.%d", i+1)),
			Type:    0,
			Version: utils.NewStringValue("v1.0.0"),
			Location: &apimodel.Location{
				Region: utils.NewStringValue("region"),
				Zone:   utils.NewStringValue("zone"),
				Campus: utils.NewStringValue("campus"),
			},
			Id:   utils.NewStringValue(id),
			Stat: []*apiservice.StatInfo{},
		})

		ret[id].SetValid(true)
	}

	return ret
}

func Test_clientCache_GetClient(t *testing.T) {

	t.Run("测试正常的client实例缓存获取", func(t *testing.T) {
		ctrl, store, clientCache := newTestClientCache(t)
		defer ctrl.Finish()

		ret := mockClients(10)

		id := ""
		for k := range ret {
			id = k
			break
		}

		store.EXPECT().GetMoreClients(gomock.Any(), gomock.Any()).Return(ret, nil)

		err := clientCache.Update()
		assert.NoError(t, err)

		item := clientCache.GetClient(id)
		assert.Equal(t, ret[id], item)
	})
}

func Test_clientCache_GetClientByFilter(t *testing.T) {

	t.Run("测试带条件的client过滤查询", func(t *testing.T) {
		ctrl, store, clientCache := newTestClientCache(t)
		defer ctrl.Finish()

		ret := mockClients(10)

		id := ""
		host := ""
		for k := range ret {
			id = k
			host = ret[id].Proto().Host.Value
			break
		}

		store.EXPECT().GetMoreClients(gomock.Any(), gomock.Any()).Return(ret, nil)

		err := clientCache.Update()
		assert.NoError(t, err)

		total, item, err := clientCache.GetClientsByFilter(map[string]string{
			"id":   id,
			"host": host,
		}, 0, 100)
		assert.NoError(t, err)
		assert.Equal(t, int32(1), int32(total))
		assert.Equal(t, int32(1), int32(len(item)))
		assert.Equal(t, ret[id], item[0])
	})
}
