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
)

// CreateGroup
func (svr *serverAuthAbility) CreateGroup(ctx context.Context, group *api.UserGroup) *api.Response {
	ctx, errResp := svr.verifyAuth(ctx, WriteOp, MustOwner)
	if errResp != nil {
		errResp.UserGroup = group
		return errResp
	}

	return svr.target.CreateGroup(ctx, group)
}

// UpdateGroups
func (svr *serverAuthAbility) UpdateGroups(ctx context.Context,
	reqs []*api.ModifyUserGroup) *api.BatchWriteResponse {

	ctx, errResp := svr.verifyAuth(ctx, WriteOp, MustOwner)
	if errResp != nil {
		resp := api.NewBatchWriteResponse(api.ExecuteSuccess)
		resp.Collect(errResp)
		return resp
	}

	return svr.target.UpdateGroups(ctx, reqs)
}

// DeleteGroup
func (svr *serverAuthAbility) DeleteGroups(ctx context.Context,
	reqs []*api.UserGroup) *api.BatchWriteResponse {

	ctx, errResp := svr.verifyAuth(ctx, WriteOp, MustOwner)
	if errResp != nil {
		resp := api.NewBatchWriteResponse(api.ExecuteSuccess)
		resp.Collect(errResp)
		return resp
	}

	return svr.target.DeleteGroups(ctx, reqs)
}

// GetGroups 查看用户组
func (svr *serverAuthAbility) GetGroups(ctx context.Context,
	query map[string]string) *api.BatchQueryResponse {

	ctx, errResp := svr.verifyAuth(ctx, ReadOp, NotOwner)
	if errResp != nil {
		return api.NewBatchQueryResponseWithMsg(errResp.GetCode().Value, errResp.Info.Value)
	}

	return svr.target.GetGroups(ctx, query)
}

// GetGroupUsers
func (svr *serverAuthAbility) GetGroup(ctx context.Context, req *api.UserGroup) *api.Response {

	ctx, errResp := svr.verifyAuth(ctx, WriteOp, NotOwner)
	if errResp != nil {
		return errResp
	}

	return svr.target.GetGroup(ctx, req)
}

// GetUserGroupToken
func (svr *serverAuthAbility) GetGroupToken(ctx context.Context, req *api.UserGroup) *api.Response {

	ctx, errResp := svr.verifyAuth(ctx, ReadOp, NotOwner)
	if errResp != nil {
		return errResp
	}

	return svr.target.GetGroupToken(ctx, req)
}

// UpdateGroupToken
func (svr *serverAuthAbility) UpdateGroupToken(ctx context.Context, group *api.UserGroup) *api.Response {
	ctx, errResp := svr.verifyAuth(ctx, WriteOp, MustOwner)
	if errResp != nil {
		errResp.UserGroup = group
		return errResp
	}

	return svr.target.UpdateGroupToken(ctx, group)
}

// ResetGroupToken
func (svr *serverAuthAbility) ResetGroupToken(ctx context.Context, group *api.UserGroup) *api.Response {
	ctx, errResp := svr.verifyAuth(ctx, WriteOp, MustOwner)
	if errResp != nil {
		errResp.UserGroup = group
		return errResp
	}

	return svr.target.ResetGroupToken(ctx, group)
}
