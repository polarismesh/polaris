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

package maintain

import (
	"context"
	"errors"

	"github.com/polarismesh/polaris-server/auth"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
)

// serverAuthAbility 带有鉴权能力的 maintainServer
type serverAuthAbility struct {
	targetServer *Server
	authSvr      auth.AuthServer
	authMgn      auth.AuthChecker
}

func newServerAuthAbility(targetServer *Server, authSvr auth.AuthServer) MaintainOperateServer {
	proxy := &serverAuthAbility{
		targetServer: targetServer,
		authSvr:      authSvr,
		authMgn:      authSvr.GetAuthChecker(),
	}

	return proxy
}

func (svr *serverAuthAbility) collectMaintainAuthContext(ctx context.Context, resourceOp model.ResourceOperation,
	methodName string) *model.AcquireContext {
	return model.NewAcquireContext(
		model.WithRequestContext(ctx),
		model.WithOperation(resourceOp),
		model.WithToken(utils.ParseAuthToken(ctx)),
		model.WithModule(model.MaintainModule),
		model.WithMethod(methodName),
	)
}

func convertToErrCode(err error) uint32 {
	if errors.Is(err, model.ErrorTokenNotExist) {
		return api.TokenNotExisted
	}

	if errors.Is(err, model.ErrorTokenDisabled) {
		return api.TokenDisabled
	}
	
	return api.NotAllowedAccess
}
