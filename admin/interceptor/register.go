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

package inteceptor

import (
	"context"

	"github.com/polarismesh/polaris/admin"
	admin_auth "github.com/polarismesh/polaris/admin/interceptor/auth"
	"github.com/polarismesh/polaris/auth"
)

type (
	ContextKeyUserSvr   struct{}
	ContextKeyPolicySvr struct{}
)

func init() {
	err := admin.RegisterServerProxy("auth", func(ctx context.Context,
		pre admin.AdminOperateServer) (admin.AdminOperateServer, error) {

		var userSvr auth.UserServer
		var policySvr auth.StrategyServer

		userSvrVal := ctx.Value(ContextKeyUserSvr{})
		if userSvrVal == nil {
			svr, err := auth.GetUserServer()
			if err != nil {
				return nil, err
			}
			userSvr = svr
		} else {
			userSvr = userSvrVal.(auth.UserServer)
		}

		policySvrVal := ctx.Value(ContextKeyPolicySvr{})
		if policySvrVal == nil {
			svr, err := auth.GetStrategyServer()
			if err != nil {
				return nil, err
			}
			policySvr = svr
		} else {
			policySvr = policySvrVal.(auth.StrategyServer)
		}

		return admin_auth.NewServer(pre, userSvr, policySvr), nil
	})
	if err != nil {
		panic(err)
	}
}
