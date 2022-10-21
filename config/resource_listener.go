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

package config

import (
	"context"
	"strconv"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
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
	ConfigGroup *api.ConfigFileGroup
}

// Before this function is called before the resource operation
func (s *serverAuthability) Before(ctx context.Context, resourceType model.Resource) {
	// do nothing
}

// After this function is called after the resource operation
func (s *serverAuthability) After(ctx context.Context, resourceType model.Resource, res *ResourceEvent) error {
	switch resourceType {
	case model.RConfigGroup:
		return s.onConfigGroupResource(ctx, res)
	default:
		return nil
	}
}

// onConfigGroupResource
func (s *serverAuthability) onConfigGroupResource(ctx context.Context, res *ResourceEvent) error {
	authCtx := ctx.Value(utils.ContextAuthContextKey).(*model.AcquireContext)

	authCtx.SetAttachment(model.ResourceAttachmentKey, map[api.ResourceType][]model.ResourceEntry{
		api.ResourceType_ConfigGroups: {
			{
				ID:    strconv.FormatUint(res.ConfigGroup.Id.GetValue(), 10),
				Owner: utils.ParseOwnerID(ctx),
			},
		},
	})

	users := utils.ConvertStringValuesToSlice(res.ConfigGroup.UserIds)
	removeUses := utils.ConvertStringValuesToSlice(res.ConfigGroup.RemoveUserIds)

	groups := utils.ConvertStringValuesToSlice(res.ConfigGroup.GroupIds)
	removeGroups := utils.ConvertStringValuesToSlice(res.ConfigGroup.RemoveGroupIds)

	authCtx.SetAttachment(model.LinkUsersKey, utils.StringSliceDeDuplication(users))
	authCtx.SetAttachment(model.RemoveLinkUsersKey, utils.StringSliceDeDuplication(removeUses))

	authCtx.SetAttachment(model.LinkGroupsKey, utils.StringSliceDeDuplication(groups))
	authCtx.SetAttachment(model.RemoveLinkGroupsKey, utils.StringSliceDeDuplication(removeGroups))

	return s.authSvr.AfterResourceOperation(authCtx)
}
