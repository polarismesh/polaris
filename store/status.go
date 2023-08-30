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

package store

import (
	"strings"
)

// StatusCode 存储层的状态码
type StatusCode uint32

// 状态码定义
const (
	Ok StatusCode = iota
	// 参数不合法
	EmptyParamsErr
	// 数据不合法，比如越级了，超过了字段大小
	OutOfRangeErr
	// 数据冲突，在并发更新metadata的时候可能会出现
	DataConflictErr
	// 找不到namespace，service插入依赖namespace是否存在
	NotFoundNamespace
	// 找不到service，在instance等资源插入的时候依赖service是否存在
	NotFoundService
	// 在标记规则前，需要保证规则的master版本存在
	NotFoundMasterConfig
	// 在发布规则前，需要保证规则已标记且服务存在
	NotFoundTagConfigOrService
	// 在删除规则时，发现存在已经发布的版本
	ExistReleasedConfig
	// 操作的行数与预期不符合
	AffectedRowsNotMatch
	// 主键重复，一般是资源已存在了，提醒用户资源存在
	DuplicateEntryErr
	// 外键错误，一般是操作不当导致的
	ForeignKeyErr
	// 数据库死锁
	DeadlockErr
	// 网格订阅服务的时候，网格或者服务不存在
	NotFoundMeshOrService
	// 更新订阅服务的时候，订阅服务不存在
	NotFoundMeshService
	// Failed to find target CircuitBreaker
	NotFoundCircuitBreaker
	// Failed to find fuse breaker information associated with service
	NotFoundReleaseCircuitBreaker
	Unknown
	// 用户不存在
	NotFoundUser
	// 用户组不存在
	NotFoundUserGroup
	// 非法的用户ID列表
	InvalidUserIDSlice
	NotFoundResource
)

// Error 普通error转StatusError
func Error(err error) error {
	if err == nil {
		return nil
	}

	// 已经是StatusError了，不再转换
	if _, ok := err.(*StatusError); ok {
		return err
	}

	s := &StatusError{message: err.Error()}
	if strings.Contains(s.message, "Data too long") {
		s.code = OutOfRangeErr
	} else if strings.Contains(s.message, "Duplicate entry") {
		s.code = DuplicateEntryErr
	} else if strings.Contains(s.message, "a foreign key constraint fails") {
		s.code = ForeignKeyErr
	} else if strings.Contains(s.message, "Deadlock") {
		s.code = DeadlockErr
	} else {
		s.code = Unknown
	}

	return s
}

// NewStatusError 根据code和message创建StatusError
func NewStatusError(code StatusCode, message string) error {
	return &StatusError{
		code:    code,
		message: message,
	}
}

// Code 根据error接口，获取状态码
func Code(err error) StatusCode {
	if err == nil {
		return Ok
	}

	se, ok := err.(*StatusError)
	if ok {
		return se.code
	}

	return Unknown
}

// StatusError 包括了状态码的error接口
type StatusError struct {
	code    StatusCode
	message string
}

// Error 实现error接口
func (s *StatusError) Error() string {
	if s == nil {
		return ""
	}

	return s.message
}
