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
	"strconv"
	"strings"

	regexp "github.com/dlclark/regexp2"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
)

const (
	// EmptyErrString empty error string
	EmptyErrString = "empty"
	// NilErrString null pointer error string
	NilErrString = "nil"
	// MatchAll rule match all service or namespace value
	MatchAll = "*"
)

func IsMatchAll(v string) bool {
	return v == "" || v == MatchAll
}

func MatchString(srcMetaValue string, matchValule *apimodel.MatchString, regexToPattern func(string) *regexp.Regexp) bool {
	rawMetaValue := matchValule.GetValue().GetValue()
	if IsMatchAll(rawMetaValue) {
		return true
	}

	switch matchValule.Type {
	case apimodel.MatchString_REGEX:
		matchExp := regexToPattern(rawMetaValue)
		if matchExp == nil {
			return false
		}
		match, err := matchExp.MatchString(srcMetaValue)
		if err != nil {
			return false
		}
		return match
	case apimodel.MatchString_NOT_EQUALS:
		return srcMetaValue != rawMetaValue
	case apimodel.MatchString_EXACT:
		return srcMetaValue == rawMetaValue
	case apimodel.MatchString_IN:
		find := false
		tokens := strings.Split(rawMetaValue, ",")
		for _, token := range tokens {
			if token == srcMetaValue {
				find = true
				break
			}
		}
		return find
	case apimodel.MatchString_NOT_IN:
		tokens := strings.Split(rawMetaValue, ",")
		for _, token := range tokens {
			if token == srcMetaValue {
				return false
			}
		}
		return true
	case apimodel.MatchString_RANGE:
		// range 模式只支持数字
		tokens := strings.Split(rawMetaValue, "~")
		if len(tokens) != 2 {
			return false
		}
		left, err := strconv.ParseInt(tokens[0], 10, 64)
		if err != nil {
			return false
		}
		right, err := strconv.ParseInt(tokens[1], 10, 64)
		if err != nil {
			return false
		}
		srcVal, err := strconv.ParseInt(srcMetaValue, 10, 64)
		if err != nil {
			return false
		}
		return srcVal >= left && srcVal <= right
	}
	return true
}
