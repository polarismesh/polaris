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

package defaultauth

import (
	"testing"

	_ "github.com/polarismesh/polaris-server/plugin/auth/defaultauth"
)

func Test_GetPrincipalResources(t *testing.T) {
	// reset(false)
	// ctrl := gomock.NewController(t)
	// defer ctrl.Finish()

	// users := createMockUser(10)
	// groups := createMockUserGroup(users)

	// namespaces := createMockNamespace(len(users)+len(groups)+10, users[0].ID)
	// services := createMockService(namespaces)
	// serviceMap := convertServiceSliceToMap(services)
	// strategies := createMockStrategy(users, groups, services[:len(users)+len(groups)])

	// storage := storemock.NewMockStore(ctrl)

	// storage.EXPECT().GetUsersForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(users, nil)
	// storage.EXPECT().GetGroupsForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(groups, nil)
	// storage.EXPECT().GetStrategyDetailsForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(strategies, nil)
	// storage.EXPECT().GetMoreNamespaces(gomock.Any()).AnyTimes().Return(namespaces, nil)
	// storage.EXPECT().GetMoreServices(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(serviceMap, nil)

	// cfg, storage := initCache(ctrl)

	// ctx, cancel := context.WithCancel(context.Background())
	// if err := cache.TestCacheInitialize(ctx, cfg, storage, nil); err != nil {
	// 	t.Fatal(err)
	// }

	// cacheMgn, err := cache.GetCacheManager()
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// defer func() {
	// 	cancel()
	// 	cacheMgn.Clear()
	// 	time.Sleep(2 * time.Second)
	// }()
	// time.Sleep(time.Second)

	// checker := &defaultAuthChecker{}
	// checker.cacheMgn = cacheMgn
	// checker.authPlugin = plugin.GetAuth()

	// svr := &serverAuthAbility{
	// 	authMgn: checker,
	// 	target: &server{
	// 		storage:  storage,
	// 		cacheMgn: &cache.NamingCache{},
	// 		authMgn:  checker,
	// 	},
	// }

	// ret := svr.GetPrincipalResources(context.Background(), map[string]string{
	// 	"principal_id":   users[1].ID,
	// 	"principal_type": "user",
	// })

	// assert.EqualValues(t, api.ExecuteSuccess, ret.Code, "need query success")
	// resources := ret.Resources
	// assert.Equal(t, 2, len(resources.Services), "need query 2 service resources")
}
