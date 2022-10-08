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

	api "github.com/polarismesh/polaris/common/api/v1"
)

// NamespaceOperateServer Namespace related operation
type NamespaceOperateServer interface {
	// CreateNamespace Create a single name space
	CreateNamespace(ctx context.Context, req *api.Namespace) *api.Response
	// CreateNamespaces Batch creation namespace
	CreateNamespaces(ctx context.Context, req []*api.Namespace) *api.BatchWriteResponse
	// DeleteNamespace Delete a single namespace
	DeleteNamespace(ctx context.Context, req *api.Namespace) *api.Response
	// DeleteNamespaces Batch delete namespace
	DeleteNamespaces(ctx context.Context, req []*api.Namespace) *api.BatchWriteResponse
	// UpdateNamespaces Batch update naming space
	UpdateNamespaces(ctx context.Context, req []*api.Namespace) *api.BatchWriteResponse
	// UpdateNamespaceToken Update token with namespace
	UpdateNamespaceToken(ctx context.Context, req *api.Namespace) *api.Response
	// GetNamespaces Get a list of namespaces
	GetNamespaces(ctx context.Context, query map[string][]string) *api.BatchQueryResponse
	// GetNamespaceToken Get the token with namespace
	GetNamespaceToken(ctx context.Context, req *api.Namespace) *api.Response
	// CreateNamespaceIfAbsent Create a single name space
	CreateNamespaceIfAbsent(ctx context.Context, req *api.Namespace) error
}
