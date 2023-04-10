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
	"github.com/polarismesh/polaris/auth"
	"github.com/polarismesh/polaris/common/eventhub"
	"github.com/polarismesh/polaris/store/mock"
	"github.com/stretchr/testify/assert"
)

func Test_Initialize(t *testing.T) {
	t.Cleanup(func() {
		once = sync.Once{}
		finishInit = false
		eventhub.TestShutdownEventHub()
	})

	eventhub.TestInitEventHub()
	ctrl := gomock.NewController(t)
	s := mock.NewMockStore(ctrl)

	authSvr, err := auth.TestInitialize(context.Background(), &auth.Config{
		Name:   "defaultAuth",
		Option: map[string]interface{}{},
	}, s, nil)

	assert.NoError(t, err)
	err = Initialize(context.Background(), &Config{}, authSvr)
	assert.NoError(t, err)

	svr, err := GetOriginServer()
	assert.NoError(t, err)
	assert.NotNil(t, svr)

	dSvr, err := GetServer()
	assert.NoError(t, err)
	assert.NotNil(t, dSvr)
}
