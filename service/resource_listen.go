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

package service

import (
	"context"
	"strings"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
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
	After(ctx context.Context, resourceType model.Resource, res *ResourceEvent)
}

// ResourceEvent
type ResourceEvent struct {
	ReqNamespace *api.Namespace
	Namespace    *model.Namespace
	ReqService   *api.Service
	Service      *model.Service
	IsRemove     bool
}

// Before
func (svr *serverAuthAbility) Before(ctx context.Context, resourceType model.Resource) {
	// do nothing
}

// After
func (svr *serverAuthAbility) After(ctx context.Context, resourceType model.Resource, res *ResourceEvent) {
	switch resourceType {
	case model.RNamespace:
		svr.onNamespaceResource(ctx, res)
	case model.RService:
		svr.onServiceResource(ctx, res)
	}
}

// onNamespaceResource
func (svr *serverAuthAbility) onNamespaceResource(ctx context.Context, res *ResourceEvent) {
	authCtx := ctx.Value(utils.ContextAuthContextKey).(*model.AcquireContext)
	ownerId := utils.ParseOwnerID(ctx)

	ns := res.Namespace
	authCtx.GetAttachment()[model.ResourceAttachmentKey] = map[api.ResourceType][]model.ResourceEntry{
		api.ResourceType_Namespaces: {
			{
				ID:    ns.Name,
				Owner: ownerId,
			},
		},
	}

	users := make([]string, 0, 4)
	if len(res.ReqNamespace.UserIds) > 0 {
		for index := range res.ReqNamespace.UserIds {
			id := res.ReqNamespace.UserIds[index]
			if strings.TrimSpace(id.GetValue()) == "" {
				continue
			}
			users = append(users, id.GetValue())
		}
	}

	groups := make([]string, 0, 4)
	if len(res.ReqNamespace.GroupIds) > 0 {
		for index := range res.ReqNamespace.GroupIds {
			id := res.ReqNamespace.GroupIds[index]
			if strings.TrimSpace(id.GetValue()) == "" {
				continue
			}
			groups = append(groups, id.GetValue())
		}
	}

	authCtx.GetAttachment()[model.LinkUsersKey] = utils.StringSliceDeDuplication(append(users, utils.ParseUserID(ctx)))
	authCtx.GetAttachment()[model.LinkGroupsKey] = utils.StringSliceDeDuplication(groups)

	svr.authSvr.AfterResourceOperation(authCtx)

}

// onServiceResource
func (svr *serverAuthAbility) onServiceResource(ctx context.Context, res *ResourceEvent) {
	authCtx := ctx.Value(utils.ContextAuthContextKey).(*model.AcquireContext)
	ownerId := utils.ParseOwnerID(ctx)

	authCtx.GetAttachment()[model.ResourceAttachmentKey] = map[api.ResourceType][]model.ResourceEntry{
		api.ResourceType_Namespaces: {
			{
				ID:    res.Namespace.Name,
				Owner: ownerId,
			},
		},
		api.ResourceType_Services: {
			{
				ID:    res.Service.ID,
				Owner: ownerId,
			},
		},
	}

	users := make([]string, 0, 4)
	if len(res.ReqService.UserIds) > 0 {
		for index := range res.ReqService.UserIds {
			id := res.ReqService.UserIds[index]
			if strings.TrimSpace(id.GetValue()) == "" {
				continue
			}
			users = append(users, id.GetValue())
		}
	}

	groups := make([]string, 0, 4)
	if len(res.ReqService.GroupIds) > 0 {
		for index := range res.ReqService.GroupIds {
			id := res.ReqService.GroupIds[index]
			if strings.TrimSpace(id.GetValue()) == "" {
				continue
			}
			groups = append(groups, id.GetValue())
		}
	}

	authCtx.GetAttachment()[model.LinkUsersKey] = utils.StringSliceDeDuplication(append(users, utils.ParseUserID(ctx)))
	authCtx.GetAttachment()[model.LinkGroupsKey] = utils.StringSliceDeDuplication(groups)

	svr.authSvr.AfterResourceOperation(authCtx)
}

// onConfigGroupResource
func (svr *serverAuthAbility) onConfigGroupResource(ctx context.Context, res *ResourceEvent) {
	authCtx := ctx.Value(utils.ContextAuthContextKey).(*model.AcquireContext)

	svr.authSvr.AfterResourceOperation(authCtx)
}
