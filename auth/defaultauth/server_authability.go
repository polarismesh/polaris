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
	"errors"

	"github.com/polarismesh/polaris-server/auth"
	"github.com/polarismesh/polaris-server/cache"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/plugin"
	"github.com/polarismesh/polaris-server/store"
)

// serverAuthAbility 带鉴权能力的 server
type serverAuthAbility struct {
	authMgn *defaultAuthManager
	target  *server
}

// Initialize 执行初始化动作
func (svr *serverAuthAbility) Initialize(authOpt *auth.Config, storage store.Store, cacheMgn *cache.NamingCache) error {

	history := plugin.GetHistory()

	if history == nil {
		return errors.New("RecordPlugin not found")
	}

	authMgn := &defaultAuthManager{}
	if err := authMgn.Initialize(authOpt, cacheMgn); err != nil {
		return err
	}

	svr.authMgn = authMgn

	svr.target = &server{
		storage:  storage,
		history:  history,
		cacheMgn: cacheMgn,
	}

	return nil
}

// Login
func (svr *serverAuthAbility) Login(req *api.LoginRequest) *api.Response {
	return svr.target.Login(req)
}

// AfterResourceOperation
func (svr *serverAuthAbility) AfterResourceOperation(afterCtx *model.AcquireContext) {
	svr.target.AfterResourceOperation(afterCtx)
}

// GetAuthManager
func (svr *serverAuthAbility) GetAuthManager() auth.AuthManager {
	return svr.authMgn
}

// Name
func (svr *serverAuthAbility) Name() string {
	return PluginName
}
