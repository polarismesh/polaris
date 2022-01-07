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

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/auth"
)

// verifyAuth token
func verifyAuth(ctx context.Context, authMgn *defaultAuthManager, token string, needOwner bool) (context.Context, TokenInfo, *api.Response) {
	tokenInfo, err := authMgn.ParseToken(token)
	if err != nil {
		return ctx, tokenInfo, api.NewResponseWithMsg(api.ExecuteException, err.Error())
	}

	if tokenInfo.Role == auth.RoleForUserGroup {
		return ctx, tokenInfo, api.NewResponseWithMsg(api.NotAllowedAccess, "only user can access this API")
	}

	if err = authMgn.checkToken(tokenInfo); err != nil {
		return ctx, tokenInfo, api.NewResponseWithMsg(api.NotAllowedAccess, err.Error())
	}

	if needOwner && !tokenInfo.IsOwner {
		return ctx, tokenInfo, api.NewResponseWithMsg(api.NotAllowedAccess, "only main account can access this API")
	}

	ctx = context.WithValue(ctx, utils.StringContext("is-owner"), tokenInfo.IsOwner)
	ctx = context.WithValue(ctx, utils.StringContext("user-id"), tokenInfo.ID)

	return ctx, tokenInfo, nil
}
