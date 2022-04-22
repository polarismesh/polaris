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
	"errors"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store/mock"
)

// 创建一个测试mock userCache
func newTestUserCache(t *testing.T) (*gomock.Controller, *mock.MockStore, *userCache) {
	ctl := gomock.NewController(t)

	storage := mock.NewMockStore(ctl)
	uc := newUserCache(storage, make(chan interface{}, 4))
	opt := map[string]interface{}{}
	_ = uc.initialize(opt)

	return ctl, storage, uc.(*userCache)
}

// 生成测试数据
func genModelUsers(label string, total int) map[string]*model.User {
	if total%10 != 0 {
		panic(errors.New("total must like 10, 20, 30, 40, ..."))
	}

	out := make(map[string]*model.User)

	var owner *model.User

	for i := 0; i < total; i++ {
		if i%10 == 0 {
			owner = &model.User{
				ID:       fmt.Sprintf("owner-user-%d", i),
				Name:     fmt.Sprintf("owner-user-%d", i),
				Password: fmt.Sprintf("owner-user-%d", i),
				Owner:    "",
				Source:   "Polaris",
				Type:     model.OwnerUserRole,
				Token:    fmt.Sprintf("owner-user-%d", i),
				Valid:    true,
			}
		}

		entry := &model.User{
			ID:       fmt.Sprintf("sub-user-%d", i),
			Name:     fmt.Sprintf("sub-user-%d", i),
			Password: fmt.Sprintf("sub-user-%d", i),
			Owner:    owner.ID,
			Source:   "Polaris",
			Type:     model.SubAccountUserRole,
			Token:    fmt.Sprintf("sub-user-%d", i),
			Valid:    true,
		}

		out[entry.ID] = entry
	}
	return out
}


func TestUserCache_UpdateNormal(t *testing.T) {
}
