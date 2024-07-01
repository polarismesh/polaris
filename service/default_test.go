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
	"context"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/polarismesh/polaris/auth"
	"github.com/polarismesh/polaris/cache"
	cachemock "github.com/polarismesh/polaris/cache/mock"
	"github.com/polarismesh/polaris/namespace"
	"github.com/polarismesh/polaris/service/batch"
	"github.com/polarismesh/polaris/service/healthcheck"
	"github.com/polarismesh/polaris/store/mock"
)

func Test_Initialize(t *testing.T) {
	t.Cleanup(func() {
		once = sync.Once{}
		finishInit = false
	})

	ctrl := gomock.NewController(t)
	s := mock.NewMockStore(ctrl)
	cacheMgr := cachemock.NewMockCacheManager(ctrl)
	cacheMgr.EXPECT().OpenResourceCache(gomock.Any()).Return(nil).AnyTimes()
	cacheMgr.EXPECT().GetReportInterval().Return(time.Second).AnyTimes()
	cacheMgr.EXPECT().GetUpdateCacheInterval().Return(time.Second).AnyTimes()

	_, _, err := auth.TestInitialize(context.Background(), &auth.Config{
		Option: map[string]interface{}{},
	}, s, cacheMgr)
	assert.NoError(t, err)

	err = Initialize(context.Background(), &Config{
		Interceptors: GetChainOrder(),
	})
	assert.NoError(t, err)

	svr, err := GetOriginServer()
	assert.NoError(t, err)
	assert.NotNil(t, svr)

	dSvr, err := GetServer()
	assert.NoError(t, err)
	assert.NotNil(t, dSvr)
}

func Test_Server(t *testing.T) {
	t.Run("cache_entries", func(t *testing.T) {
		ret := GetAllCaches()
		assert.True(t, len(ret) > 0)
	})

	t.Run("with_test", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		t.Cleanup(func() {
			ctrl.Finish()
		})

		svr := &Server{}

		opt := []InitOption{}

		mockCacheMgr := cachemock.NewMockCacheManager(ctrl)
		mockCacheMgr.EXPECT().OpenResourceCache(gomock.Any()).Return(nil).AnyTimes()
		mockCacheMgr.EXPECT().GetReportInterval().Return(time.Second).AnyTimes()
		mockCacheMgr.EXPECT().GetUpdateCacheInterval().Return(time.Second).AnyTimes()

		opt = append(opt, WithBatchController(&batch.Controller{}))
		opt = append(opt, WithNamespaceSvr(&namespace.Server{}))
		opt = append(opt, WithCacheManager(&cache.Config{}, mockCacheMgr))
		opt = append(opt, WithHealthCheckSvr(&healthcheck.Server{}))
		opt = append(opt, WithStorage(mock.NewMockStore(ctrl)))

		for i := range opt {
			opt[i](svr)
		}

		assert.NotNil(t, svr.bc)
		assert.NotNil(t, svr.namespaceSvr)
		assert.NotNil(t, svr.caches)
		assert.NotNil(t, svr.healthServer)
		assert.NotNil(t, svr.storage)

	})
}
