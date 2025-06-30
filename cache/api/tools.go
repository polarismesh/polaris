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

package api

import "time"

func NewExpireEntry[T any](t T, maxAlive time.Duration) *ExpireEntry[T] {
	return &ExpireEntry[T]{
		data:     t,
		maxAlive: maxAlive,
	}
}

func EmptyExpireEntry[T any](t T, maxAlive time.Duration) *ExpireEntry[T] {
	return &ExpireEntry[T]{
		empty:    true,
		maxAlive: maxAlive,
	}
}

type ExpireEntry[T any] struct {
	empty      bool
	data       T
	lastAccess time.Time
	maxAlive   time.Duration
}

func (e *ExpireEntry[T]) Get() T {
	if e.empty {
		return e.data
	}
	e.lastAccess = time.Now()
	return e.data
}

func (e *ExpireEntry[T]) IsExpire() bool {
	return time.Since(e.lastAccess) > e.maxAlive
}
