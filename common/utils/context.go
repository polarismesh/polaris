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

package utils

import "context"

type (
	// StringContext is a context key that carries a string.
	StringContext string
	// localhostCtx is a context key that carries localhost info.
	localhostCtx struct{}
	// ContextAPIServerSlot
	ContextAPIServerSlot struct{}
	// WatchTimeoutCtx .
	WatchTimeoutCtx struct{}
)

// WithLocalhost 存储localhost
func WithLocalhost(ctx context.Context, localhost string) context.Context {
	return context.WithValue(ctx, localhostCtx{}, localhost)
}

// ValueLocalhost 获取localhost
func ValueLocalhost(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	value, ok := ctx.Value(localhostCtx{}).(string)
	if !ok {
		return ""
	}

	return value
}
