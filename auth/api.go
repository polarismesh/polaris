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

// Authority 内部鉴权接口
// 内部鉴权分为两大类：命名空间和服务的资源鉴权；请求鉴权，
// 比如对于OSS操作，需要全局放通
type Authority interface {
	// VerifyToken 检查Token格式是否合法
	VerifyToken(actualToken string) bool

	// VerifyNamespace 校验命名空间是否合法
	VerifyNamespace(expectToken string, actualToken string) bool

	// VerifyService 校验服务是否合法
	VerifyService(expectToken string, actualToken string) bool

	// VerifyInstance 校验实例是否合法
	VerifyInstance(expectToken string, actualToken string) bool

	// VerifyRule 校验规则是否合法
	VerifyRule(expectToken string, actualToken string) bool

	// VerifyPlatform 校验平台是否合法
	VerifyPlatform(expectToken string, actualToken string) bool

	// VerifyMesh 校验网格权限是否合法
	VerifyMesh(expectToken string, actualToken string) bool
}
