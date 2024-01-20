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

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/polarismesh/polaris/auth"
	cachemock "github.com/polarismesh/polaris/cache/mock"
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
