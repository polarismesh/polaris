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
	"github.com/polarismesh/polaris/common/model"
)

// serverAuthAbility 带有鉴权能力的 maintainServer
type serverAuthAbility struct {
	targetServer *Server
	userMgn      auth.UserOperator
	strategyMgn  auth.StrategyOperator
}

func newServerAuthAbility(targetServer *Server, userMgn auth.UserOperator, strategyMgn auth.StrategyOperator) AdminOperateServer {
	proxy := &serverAuthAbility{
		targetServer: targetServer,
		userMgn:      userMgn,
		strategyMgn:  strategyMgn,
	}

	return proxy
}

func (svr *serverAuthAbility) collectMaintainAuthContext(ctx context.Context, resourceOp model.ResourceOperation,
	methodName string) *model.AcquireContext {
	return model.NewAcquireContext(
		model.WithRequestContext(ctx),
		model.WithOperation(resourceOp),
		model.WithModule(model.MaintainModule),
		model.WithMethod(methodName),
	)
}

func convertToErrCode(err error) apimodel.Code {
	if errors.Is(err, model.ErrorTokenNotExist) {
		return apimodel.Code_TokenNotExisted
	}

	if errors.Is(err, model.ErrorTokenDisabled) {
		return apimodel.Code_TokenDisabled
	}

	return apimodel.Code_NotAllowedAccess
}
