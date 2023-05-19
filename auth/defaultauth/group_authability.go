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

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	api "github.com/polarismesh/polaris/common/api/v1"
)

type groupAuthAbility struct {
	authMgn *defaultAuthChecker
	target  *server
}

// CreateGroup creates a group.
func (svr *groupAuthAbility) CreateGroup(ctx context.Context, group *apisecurity.UserGroup) *apiservice.Response {
	ctx, rsp := verifyAuth(ctx, WriteOp, MustOwner, svr.authMgn)
	if rsp != nil {
		rsp.UserGroup = group
		return rsp
	}

	return svr.target.CreateGroup(ctx, group)
}

// UpdateGroups updates groups.
func (svr *groupAuthAbility) UpdateGroups(ctx context.Context,
	reqs []*apisecurity.ModifyUserGroup) *apiservice.BatchWriteResponse {
	ctx, rsp := verifyAuth(ctx, WriteOp, MustOwner, svr.authMgn)
	if rsp != nil {
		resp := api.NewAuthBatchWriteResponse(apimodel.Code_ExecuteSuccess)
		api.Collect(resp, rsp)
		return resp
	}

	return svr.target.UpdateGroups(ctx, reqs)
}

// DeleteGroups deletes groups.
func (svr *groupAuthAbility) DeleteGroups(ctx context.Context,
	reqs []*apisecurity.UserGroup) *apiservice.BatchWriteResponse {
	ctx, rsp := verifyAuth(ctx, WriteOp, MustOwner, svr.authMgn)
	if rsp != nil {
		resp := api.NewAuthBatchWriteResponse(apimodel.Code_ExecuteSuccess)
		api.Collect(resp, rsp)
		return resp
	}

	return svr.target.DeleteGroups(ctx, reqs)
}

// GetGroups 查看用户组列表
func (svr *groupAuthAbility) GetGroups(ctx context.Context,
	query map[string]string) *apiservice.BatchQueryResponse {
	ctx, rsp := verifyAuth(ctx, ReadOp, NotOwner, svr.authMgn)
	if rsp != nil {
		return api.NewAuthBatchQueryResponseWithMsg(apimodel.Code(rsp.GetCode().Value), rsp.Info.Value)
	}

	return svr.target.GetGroups(ctx, query)
}

// GetGroup 查看用户组
func (svr *groupAuthAbility) GetGroup(ctx context.Context, req *apisecurity.UserGroup) *apiservice.Response {
	ctx, rsp := verifyAuth(ctx, WriteOp, NotOwner, svr.authMgn)
	if rsp != nil {
		return rsp
	}

	return svr.target.GetGroup(ctx, req)
}

// GetGroupToken 获取用户组token
func (svr *groupAuthAbility) GetGroupToken(ctx context.Context, req *apisecurity.UserGroup) *apiservice.Response {
	ctx, rsp := verifyAuth(ctx, ReadOp, NotOwner, svr.authMgn)
	if rsp != nil {
		return rsp
	}

	return svr.target.GetGroupToken(ctx, req)
}

// UpdateGroupToken 更新用户组token
func (svr *groupAuthAbility) UpdateGroupToken(ctx context.Context, group *apisecurity.UserGroup) *apiservice.Response {
	ctx, rsp := verifyAuth(ctx, WriteOp, MustOwner, svr.authMgn)
	if rsp != nil {
		rsp.UserGroup = group
		return rsp
	}

	return svr.target.UpdateGroupToken(ctx, group)
}

// ResetGroupToken 重置用户组token
func (svr *groupAuthAbility) ResetGroupToken(ctx context.Context, group *apisecurity.UserGroup) *apiservice.Response {
	ctx, rsp := verifyAuth(ctx, WriteOp, MustOwner, svr.authMgn)
	if rsp != nil {
		rsp.UserGroup = group
		return rsp
	}

	return svr.target.ResetGroupToken(ctx, group)
}
