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

import (
	"encoding/hex"

	"github.com/google/uuid"
)

var emptyVal = struct{}{}

// ConvertFilter map[string]string to  map[string][]string
func ConvertFilter(filters map[string]string) map[string][]string {
	newFilters := make(map[string][]string)

	for k, v := range filters {
		val := make([]string, 0)
		val = append(val, v)
		newFilters[k] = val
	}

	return newFilters
}

// CollectMapKeys collect filters key to slice
func CollectMapKeys(filters map[string]string) []string {
	fields := make([]string, 0, len(filters))
	for k := range filters {
		fields = append(fields, k)
	}

	return fields
}

// IsWildName 判断名字是否为通配名字，只支持前缀索引(名字最后为*)
func IsWildName(name string) bool {
	length := len(name)
	return length >= 1 && name[length-1:length] == "*"
}

// NewUUID 返回一个随机的UUID
func NewUUID() string {
	uuidBytes := uuid.New()
	return hex.EncodeToString(uuidBytes[:])
}

// NewUUID 返回一个随机的UUID
func NewRoutingV2UUID() string {
	uuidBytes := uuid.New()
	return hex.EncodeToString(uuidBytes[:])
}

// NewV2Revision 返回一个随机的UUID
func NewV2Revision() string {
	uuidBytes := uuid.New()
	return "v2-" + hex.EncodeToString(uuidBytes[:])
}

// StringSliceDeDuplication 字符切片去重
func StringSliceDeDuplication(s []string) []string {
	m := make(map[string]struct{}, len(s))
	res := make([]string, 0, len(s))
	for k := range s {
		if _, ok := m[s[k]]; !ok {
			m[s[k]] = emptyVal
			res = append(res, s[k])
		}
	}

	return res
}
