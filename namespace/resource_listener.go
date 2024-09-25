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

package namespace

import (
	"context"

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"

	"github.com/polarismesh/polaris/common/model"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
)

// ResourceHook The listener is placed before and after the resource operation, only normal flow
type ResourceHook interface {

	// Before
	//  @param ctx
	//  @param resourceType
	Before(ctx context.Context, resourceType model.Resource)

	// After
	//  @param ctx
	//  @param resourceType
	//  @param res
	After(ctx context.Context, resourceType model.Resource, res *ResourceEvent) error
}

// ResourceEvent 资源事件
type ResourceEvent struct {
	ReqNamespace *apimodel.Namespace
	Namespace    *model.Namespace
	IsRemove     bool
}

// Before this function is called before the resource operation
func (svr *serverAuthAbility) Before(ctx context.Context, resourceType model.Resource) {
	// do nothing
}

// After this function is called after the resource operation
func (svr *serverAuthAbility) After(ctx context.Context, resourceType model.Resource, res *ResourceEvent) error {
	switch resourceType {
	case model.RNamespace:
		return svr.onNamespaceResource(ctx, res)
	default:
		return nil
	}
}

// onNamespaceResource
func (svr *serverAuthAbility) onNamespaceResource(ctx context.Context, res *ResourceEvent) error {
	authCtx, _ := ctx.Value(utils.ContextAuthContextKey).(*authcommon.AcquireContext)
	if authCtx == nil {
		log.Warn("[Namespace][ResourceHook] get auth context is nil, ignore", utils.RequestID(ctx))
		return nil
	}

	authCtx.SetAttachment(authcommon.ResourceAttachmentKey, map[apisecurity.ResourceType][]authcommon.ResourceEntry{
		apisecurity.ResourceType_Namespaces: {
			{
				ID:    res.Namespace.Name,
				Owner: utils.ParseOwnerID(ctx),
			},
		},
	})

	users := utils.ConvertStringValuesToSlice(res.ReqNamespace.UserIds)
	removeUses := utils.ConvertStringValuesToSlice(res.ReqNamespace.RemoveUserIds)

	groups := utils.ConvertStringValuesToSlice(res.ReqNamespace.GroupIds)
	removeGroups := utils.ConvertStringValuesToSlice(res.ReqNamespace.RemoveGroupIds)

	authCtx.SetAttachment(authcommon.LinkUsersKey, utils.StringSliceDeDuplication(users))
	authCtx.SetAttachment(authcommon.RemoveLinkUsersKey, utils.StringSliceDeDuplication(removeUses))

	authCtx.SetAttachment(authcommon.LinkGroupsKey, utils.StringSliceDeDuplication(groups))
	authCtx.SetAttachment(authcommon.RemoveLinkGroupsKey, utils.StringSliceDeDuplication(removeGroups))

	return svr.policySvr.AfterResourceOperation(authCtx)
}
