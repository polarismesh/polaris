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

package service_auth

import (
	"context"

	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"

	"github.com/polarismesh/polaris/common/model"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/service"
)

// Before this function is called before the resource operation
func (svr *Server) Before(ctx context.Context, resourceType model.Resource) {
	// do nothing
}

// After this function is called after the resource operation
func (svr *Server) After(ctx context.Context, resourceType model.Resource, res *service.ResourceEvent) error {
	switch resourceType {
	case model.RService:
		return svr.onServiceResource(ctx, res)
	default:
		return nil
	}
}

// onServiceResource 服务资源的处理，只处理服务，namespace 只由 namespace 相关的进行处理，
func (svr *Server) onServiceResource(ctx context.Context, res *service.ResourceEvent) error {
	authCtx := ctx.Value(utils.ContextAuthContextKey).(*authcommon.AcquireContext)
	ownerId := utils.ParseOwnerID(ctx)

	authCtx.SetAttachment(authcommon.ResourceAttachmentKey, map[apisecurity.ResourceType][]authcommon.ResourceEntry{
		apisecurity.ResourceType_Services: {
			{
				ID:       res.Service.ID,
				Owner:    ownerId,
				Metadata: res.Service.Meta,
			},
		},
	})

	users := utils.ConvertStringValuesToSlice(res.ReqService.UserIds)
	removeUses := utils.ConvertStringValuesToSlice(res.ReqService.RemoveUserIds)

	groups := utils.ConvertStringValuesToSlice(res.ReqService.GroupIds)
	removeGroups := utils.ConvertStringValuesToSlice(res.ReqService.RemoveGroupIds)

	authCtx.SetAttachment(authcommon.LinkUsersKey, utils.StringSliceDeDuplication(users))
	authCtx.SetAttachment(authcommon.RemoveLinkUsersKey, utils.StringSliceDeDuplication(removeUses))

	authCtx.SetAttachment(authcommon.LinkGroupsKey, utils.StringSliceDeDuplication(groups))
	authCtx.SetAttachment(authcommon.RemoveLinkGroupsKey, utils.StringSliceDeDuplication(removeGroups))

	return svr.policySvr.AfterResourceOperation(authCtx)
}
