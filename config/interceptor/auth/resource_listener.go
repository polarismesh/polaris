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

package config_auth

import (
	"context"
	"strconv"

	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/config"
)

// Before this function is called before the resource operation
func (s *ServerAuthability) Before(ctx context.Context, resourceType model.Resource) {
	// do nothing
}

// After this function is called after the resource operation
func (s *ServerAuthability) After(ctx context.Context, resourceType model.Resource, res *config.ResourceEvent) error {
	switch resourceType {
	case model.RConfigGroup:
		return s.onConfigGroupResource(ctx, res)
	default:
		return nil
	}
}

// onConfigGroupResource
func (s *ServerAuthability) onConfigGroupResource(ctx context.Context, res *config.ResourceEvent) error {
	authCtx := ctx.Value(utils.ContextAuthContextKey).(*auth.AcquireContext)

	authCtx.SetAttachment(auth.ResourceAttachmentKey, map[apisecurity.ResourceType][]auth.ResourceEntry{
		apisecurity.ResourceType_ConfigGroups: {
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

	authCtx.SetAttachment(auth.LinkUsersKey, utils.StringSliceDeDuplication(users))
	authCtx.SetAttachment(auth.RemoveLinkUsersKey, utils.StringSliceDeDuplication(removeUses))

	authCtx.SetAttachment(auth.LinkGroupsKey, utils.StringSliceDeDuplication(groups))
	authCtx.SetAttachment(auth.RemoveLinkGroupsKey, utils.StringSliceDeDuplication(removeGroups))

	return s.policyMgr.AfterResourceOperation(authCtx)
}
