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

package auth

import (
	"context"

	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
)

type acquireContextOption func(authCtx *AcquireContext)

var (
	_defaultAuthContextOptions []acquireContextOption = []acquireContextOption{
		WithFromConsole(),
	}
)

// AcquireContext 每次鉴权请求上下文信息
type AcquireContext struct {
	// RequestContext 请求上下文
	requestContext context.Context
	// Module 来自那个业务层（服务注册与服务治理、配置模块）
	module BzModule
	// Method 操作函数
	method ServerFunctionName
	// Operation 本次操作涉及的动作
	operation ResourceOperation
	// Resources 本次
	accessResources map[apisecurity.ResourceType][]ResourceEntry
	// Attachment 携带信息，用于操作完权限检查和资源操作的后置处理逻辑，解决信息需要二次查询问题
	attachment map[string]interface{}
	// fromClient 是否来自客户端的请求
	fromClient bool
	// allowAnonymous 是否允许匿名用户
	allowAnonymous bool
}

// NewAcquireContext 创建一个请求响应
//
//	@param options
//	@return *AcquireContext
func NewAcquireContext(options ...acquireContextOption) *AcquireContext {
	authCtx := &AcquireContext{
		attachment:      make(map[string]interface{}),
		accessResources: make(map[apisecurity.ResourceType][]ResourceEntry),
		module:          UnknowModule,
	}

	for index := range _defaultAuthContextOptions {
		opt := _defaultAuthContextOptions[index]
		opt(authCtx)
	}

	for index := range options {
		opt := options[index]
		opt(authCtx)
	}

	return authCtx
}

// WithRequestContext 设置请求上下文
//
//	@param ctx
//	@return acquireContextOption
func WithRequestContext(ctx context.Context) acquireContextOption {
	return func(authCtx *AcquireContext) {
		authCtx.requestContext = ctx
	}
}

// WithModule 设置本次请求的模块
//
//	@param module
//	@return acquireContextOption
func WithModule(module BzModule) acquireContextOption {
	return func(authCtx *AcquireContext) {
		authCtx.module = module
	}
}

// WithMethod 本次操作函数名称
func WithMethod(method ServerFunctionName) acquireContextOption {
	return func(authCtx *AcquireContext) {
		authCtx.method = method
	}
}

// WithOperation 设置本次的操作类型
//
//	@param operation
//	@return acquireContextOption
func WithOperation(operation ResourceOperation) acquireContextOption {
	return func(authCtx *AcquireContext) {
		authCtx.operation = operation
	}
}

// WithAccessResources 设置本次访问的资源
//
//	@param accessResources
//	@return acquireContextOption
func WithAccessResources(accessResources map[apisecurity.ResourceType][]ResourceEntry) acquireContextOption {
	return func(authCtx *AcquireContext) {
		authCtx.accessResources = accessResources
	}
}

// WithAttachment 设置本次请求的额外携带信息
//
//	@param attachment
//	@return acquireContextOption
func WithAttachment(attachment map[string]interface{}) acquireContextOption {
	return func(authCtx *AcquireContext) {
		for k, v := range attachment {
			authCtx.attachment[k] = v
		}
	}
}

// WithFromConsole 设置本次请求来自控制台
func WithFromConsole() acquireContextOption {
	return func(authCtx *AcquireContext) {
		authCtx.fromClient = false
	}
}

// WithFromClient 设置本次请求来自客户端
func WithFromClient() acquireContextOption {
	return func(authCtx *AcquireContext) {
		authCtx.fromClient = true
	}
}

// GetRequestContext 获取 context.Context
//
//	@receiver authCtx
//	@return context.Context
func (authCtx *AcquireContext) GetRequestContext() context.Context {
	return authCtx.requestContext
}

// SetRequestContext 重新设置 context.Context
//
//	@receiver authCtx
//	@param requestContext
func (authCtx *AcquireContext) SetRequestContext(requestContext context.Context) {
	authCtx.requestContext = requestContext
}

// GetModule 获取请求的模块
//
//	@receiver authCtx
//	@return BzModule
func (authCtx *AcquireContext) GetModule() BzModule {
	return authCtx.module
}

// GetOperation 获取本次操作的类型
//
//	@receiver authCtx
//	@return ResourceOperation
func (authCtx *AcquireContext) GetOperation() ResourceOperation {
	return authCtx.operation
}

// GetAccessResources 获取本次请求的资源
//
//	@receiver authCtx
//	@return map
func (authCtx *AcquireContext) GetAccessResources() map[apisecurity.ResourceType][]ResourceEntry {
	return authCtx.accessResources
}

// SetAccessResources 设置本次请求的资源
//
//	@receiver authCtx
//	@param accessRes
func (authCtx *AcquireContext) SetAccessResources(accessRes map[apisecurity.ResourceType][]ResourceEntry) {
	authCtx.accessResources = accessRes
}

// GetAttachments 获取本次请求的额外携带信息
func (authCtx *AcquireContext) GetAttachments() map[string]interface{} {
	return authCtx.attachment
}

// GetAttachment 按照 key 获取某一个附件信息
func (authCtx *AcquireContext) GetAttachment(key string) (interface{}, bool) {
	val, ok := authCtx.attachment[key]
	return val, ok
}

// SetAttachment 设置附件
func (authCtx *AcquireContext) SetAttachment(key string, val interface{}) {
	authCtx.attachment[key] = val
}

// GetMethod 获取本次请求涉及的操作函数
func (authCtx *AcquireContext) GetMethod() ServerFunctionName {
	return authCtx.method
}

// SetFromClient 本次请求来自客户端
func (authCtx *AcquireContext) SetFromClient() {
	authCtx.fromClient = true
}

// SetFromConsole 本次请求来自OpenAPI
func (authCtx *AcquireContext) SetFromConsole() {
	authCtx.fromClient = false
}

// IsFromClient 本次请求是否来自客户端
func (authCtx *AcquireContext) IsFromClient() bool {
	return authCtx.fromClient
}

// IsFromConsole 本次请求是否来自OpenAPI
func (authCtx *AcquireContext) IsFromConsole() bool {
	return !authCtx.IsFromClient()
}

// IsAccessResourceEmpty 判断当前待访问的资源，是否为空
func (authCtx *AcquireContext) IsAccessResourceEmpty() bool {
	nsEmpty := len(authCtx.accessResources[apisecurity.ResourceType_Namespaces]) == 0
	svcEmpty := len(authCtx.accessResources[apisecurity.ResourceType_Services]) == 0
	cfgEmpty := len(authCtx.accessResources[apisecurity.ResourceType_ConfigGroups]) == 0

	return nsEmpty && svcEmpty && cfgEmpty
}

// AllowAnonymous 本次请求是否允许匿名访问
func (authCtx *AcquireContext) IsAllowAnonymous() bool {
	return authCtx.allowAnonymous
}

// SetAllowAnonymous 本次请求是否允许匿名访问
func (authCtx *AcquireContext) SetAllowAnonymous(a bool) {
	authCtx.allowAnonymous = a
}

// ResourceOpInfo 资源的数据操作信息
type ResourceOpInfo struct {
	ResourceType apisecurity.ResourceType
	Namespace    string
	ResourceName string
	ResourceID   string
}
