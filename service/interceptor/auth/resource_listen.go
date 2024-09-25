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
	// 资源删除，触发所有关联的策略进行一个 update 操作更新
	return svr.onChangeResource(ctx, res)
}

// onChangeResource 服务资源的处理，只处理服务，namespace 只由 namespace 相关的进行处理，
func (svr *Server) onChangeResource(ctx context.Context, res *service.ResourceEvent) error {
	authCtx := ctx.Value(utils.ContextAuthContextKey).(*authcommon.AcquireContext)

	authCtx.SetAttachment(authcommon.ResourceAttachmentKey, map[apisecurity.ResourceType][]authcommon.ResourceEntry{
		res.Resource.Type: {
			res.Resource,
		},
	})

	var users, removeUsers []string
	var groups, removeGroups []string

	for i := range res.AddPrincipals {
		switch res.AddPrincipals[i].PrincipalType {
		case authcommon.PrincipalUser:
			users = append(users, res.AddPrincipals[i].PrincipalID)
		case authcommon.PrincipalGroup:
			groups = append(groups, res.AddPrincipals[i].PrincipalID)
		}
	}
	for i := range res.DelPrincipals {
		switch res.DelPrincipals[i].PrincipalType {
		case authcommon.PrincipalUser:
			removeUsers = append(removeUsers, res.DelPrincipals[i].PrincipalID)
		case authcommon.PrincipalGroup:
			removeGroups = append(removeGroups, res.DelPrincipals[i].PrincipalID)
		}
	}

	authCtx.SetAttachment(authcommon.LinkUsersKey, users)
	authCtx.SetAttachment(authcommon.RemoveLinkUsersKey, removeUsers)

	authCtx.SetAttachment(authcommon.LinkGroupsKey, groups)
	authCtx.SetAttachment(authcommon.RemoveLinkGroupsKey, removeGroups)

	return svr.policySvr.AfterResourceOperation(authCtx)
}
