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
	"context"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/polarismesh/polaris-server/cache"
	"github.com/polarismesh/polaris-server/plugin"
	"github.com/polarismesh/polaris-server/store/mock"
)

func Test_defaultAuthManager_ParseToken(t *testing.T) {
	AuthOption.Salt = "polaris@a7b068ce3235442b"
	token := "orRm9Zt7sMqQaAM5b7yHLXnhWsr5dfPT0jpRlQ+C0tdy2UmuDa/X3uFG"

	authMgn := &defaultAuthChecker{}

	tokenInfo, err := authMgn.DecodeToken(token)

	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%#v", tokenInfo)
}

func initDefaultAuth() {
	// 设置好默认的鉴权策略插件
	plugin.SetPluginConfig(&plugin.Config{
		Auth: plugin.ConfigEntry{
			Name: "defaultAuth",
		},
	})
}

func initCache(ctrl *gomock.Controller) (*cache.NamingCache, error) {
	/*
	   - name: service # 加载服务数据
	     option:
	       disableBusiness: false # 不加载业务服务
	       needMeta: true # 加载服务元数据
	   - name: instance # 加载实例数据
	     option:
	       disableBusiness: false # 不加载业务服务实例
	       needMeta: true # 加载实例元数据
	   - name: routingConfig # 加载路由数据
	   - name: rateLimitConfig # 加载限流数据
	   - name: circuitBreakerConfig # 加载熔断数据
	   - name: l5 # 加载l5数据
	   - name: users
	   - name: strategyRule
	   - name: namespace
	*/
	cfg := &cache.Config{
		Open: true,
		Resources: []cache.ConfigEntry{
			{
				Name: "service",
				Option: map[string]interface{}{
					"disableBusiness": false,
					"needMeta":        true,
				},
			},
			{
				Name: "instance",
				Option: map[string]interface{}{
					"disableBusiness": false,
					"needMeta":        true,
				},
			},
			{
				Name: "users",
			},
			{
				Name: "strategyRule",
			},
			{
				Name: "namespace",
			},
		},
	}

	cache.Initialize(context.Background(), cfg, mock.NewMockStore(ctrl), nil)

	return cache.GetCacheManager()
}

func Test_defaultAuthChecker_VerifyToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	initDefaultAuth()
	cacheMgn, err := initCache(ctrl)
	if err != nil {
		t.Fatal(err)
	}

	checker := &defaultAuthChecker{}

	checker.cacheMgn = cacheMgn
	checker.authPlugin = plugin.GetAuth()

}

func Test_defaultAuthChecker_removeNoStrategyResources(t *testing.T) {

}
