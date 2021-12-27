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

package naming

import (
	"context"

	api "github.com/polarismesh/polaris-server/common/api/v1"
)

func (svr *serverAuthAbility) CreateNamespaces(ctx context.Context, req []*api.Namespace) *api.BatchWriteResponse {

}

func (svr *serverAuthAbility) CreateNamespace(ctx context.Context, req *api.Namespace) *api.Response {

}

func (svr *serverAuthAbility) DeleteNamespaces(ctx context.Context, req []*api.Namespace) *api.BatchWriteResponse {

}

func (svr *serverAuthAbility) DeleteNamespace(ctx context.Context, req *api.Namespace) *api.Response {

}

func (svr *serverAuthAbility) UpdateNamespaces(ctx context.Context, req []*api.Namespace) *api.BatchWriteResponse {

}

func (svr *serverAuthAbility) UpdateNamespace(ctx context.Context, req *api.Namespace) *api.Response {

}

func (svr *serverAuthAbility) UpdateNamespaceToken(ctx context.Context, req *api.Namespace) *api.Response {

}

func (svr *serverAuthAbility) GetNamespaces(query map[string][]string) *api.BatchQueryResponse {

}

func (svr *serverAuthAbility) GetNamespaceToken(ctx context.Context, req *api.Namespace) *api.Response {

}
