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
	"strings"

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
)

// Match 查找tags 中寻找匹配term 的tag
func Match(term *apimodel.MatchTerm, tags []*apimodel.Tag) bool {
	for _, tag := range tags {
		if term.GetKey().GetValue() == tag.GetKey().GetValue() {
			val := term.GetValue()
			express := val.GetValue().GetValue()
			tagValue := tag.GetValue().GetValue()
			switch val.GetType() {
			case apimodel.MatchString_EXACT:
				if tagValue == express {
					return true
				}
			case apimodel.MatchString_IN:
				fields := strings.Split(express, ",")
				for _, field := range fields {
					if tagValue == field {
						return true
					}
				}
			}
			return false
		}
	}
	return false
}
