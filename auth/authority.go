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
	"strings"
)

// @Desperate
// authority 鉴权数据来源类
type authority struct {
	global string
	open   bool
}

const (
	globalToken = "polaris@12345678"
	defaultOpen = true
)

// interface impl check
var _ Authority = (*authority)(nil)

// NewAuthority 新建一个缓存类
func NewAuthority(opt map[string]interface{}) (Authority, error) {
	global, _ := opt["global-token"].(string)
	if global == "" {
		global = globalToken
	}
	au := &authority{global: global, open: parseOpen(opt)}
	return au, nil
}

// parseOpen 解析鉴权功能是否打开的开关
func parseOpen(opt map[string]interface{}) bool {
	var (
		open bool
		ok   bool
	)

	open, ok = opt["open"].(bool)
	if !ok {
		return defaultOpen
	}
	return open
}

// VerifyToken 检查Token格式是否合法
func (a *authority) VerifyToken(actualToken string) bool {
	if !a.open {
		return true
	}
	return len(actualToken) > 0
}

// VerifyNamespace 校验命名空间是否合法
func (a *authority) VerifyNamespace(expectToken string, actualToken string) bool {
	return (!a.open || a.global == actualToken) || expectToken == actualToken
}

// VerifyService 校验服务是否合法
func (a *authority) VerifyService(expectToken string, actualToken string) bool {
	if !a.open || a.global == actualToken {
		return true
	}

	tokens := convertToken(expectToken)
	_, ok := tokens[actualToken]
	return ok
}

// VerifyInstance 校验实例是否合法
func (a *authority) VerifyInstance(expectToken string, actualToken string) bool {
	if !a.open || a.global == actualToken {
		return true
	}

	tokens := convertToken(expectToken)
	_, ok := tokens[actualToken]
	return ok
}

// VerifyRule 校验规则是否合法
func (a *authority) VerifyRule(expectToken string, actualToken string) bool {
	if !a.open || a.global == actualToken {
		return true
	}

	tokens := convertToken(expectToken)
	_, ok := tokens[actualToken]
	return ok
}

// VerifyMesh 校验网格规则是否合法
func (a *authority) VerifyMesh(expectToken string, actualToken string) bool {
	if !a.open || a.global == actualToken {
		return true
	}

	tokens := convertToken(expectToken)
	_, ok := tokens[actualToken]
	return ok
}

// VerifyPlatform 校验平台是否合法
func (a *authority) VerifyPlatform(expectToken string, actualToken string) bool {
	if !a.open || a.global == actualToken {
		return true
	}

	tokens := convertToken(expectToken)
	_, ok := tokens[actualToken]
	return ok
}

// convertToken 将string类型的token转化为map类型
func convertToken(token string) map[string]bool {
	if token == "" {
		return nil
	}

	strSlice := strings.Split(token, ",")
	strMap := make(map[string]bool)
	for _, value := range strSlice {
		strMap[value] = true
	}

	return strMap
}
