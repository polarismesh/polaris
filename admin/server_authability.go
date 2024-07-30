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

package admin

import (
	"context"
	"errors"

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"

	"github.com/polarismesh/polaris/auth"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
)

// serverAuthAbility 带有鉴权能力的 maintainServer
type serverAuthAbility struct {
	targetServer *Server
	userMgn      auth.UserServer
	strategyMgn  auth.StrategyServer
}

func newServerAuthAbility(targetServer *Server,
	userMgn auth.UserServer, strategyMgn auth.StrategyServer) AdminOperateServer {
	proxy := &serverAuthAbility{
		targetServer: targetServer,
		userMgn:      userMgn,
		strategyMgn:  strategyMgn,
	}

	return proxy
}

func (svr *serverAuthAbility) collectMaintainAuthContext(ctx context.Context, resourceOp authcommon.ResourceOperation,
	methodName authcommon.ServerFunctionName) *authcommon.AcquireContext {
	return authcommon.NewAcquireContext(
		authcommon.WithRequestContext(ctx),
		authcommon.WithOperation(resourceOp),
		authcommon.WithModule(authcommon.MaintainModule),
		authcommon.WithMethod(methodName),
	)
}

func convertToErrCode(err error) apimodel.Code {
	if errors.Is(err, authcommon.ErrorTokenNotExist) {
		return apimodel.Code_TokenNotExisted
	}

	if errors.Is(err, authcommon.ErrorTokenDisabled) {
		return apimodel.Code_TokenDisabled
	}

	return apimodel.Code_NotAllowedAccess
}
