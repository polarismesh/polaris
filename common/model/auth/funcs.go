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

import "strings"

/*
*
string_equal
string_not_equal
string_equal_ignore_case
string_not_equal_ignore_case
string_like
string_not_like
date_equal
date_not_equal
date_greater_than
date_greater_than_equal
date_less_than
date_less_than_equal
ip_equal
ip_not_equal
*/
var (
	ConditionCompareDict = map[string]func(string, string) bool{
		// string_equal
		"string_equal": func(s1, s2 string) bool {
			return s1 == s2
		},
		"for_any_value:string_equal": func(s1, s2 string) bool {
			return s1 == s2
		},
		// string_not_equal
		"string_not_equal": func(s1, s2 string) bool {
			return s1 != s2
		},
		"for_any_value:string_not_equal": func(s1, s2 string) bool {
			return s1 != s2
		},
		// string_equal_ignore_case
		"string_equal_ignore_case":               strings.EqualFold,
		"for_any_value:string_equal_ignore_case": strings.EqualFold,
		// string_not_equal_ignore_case
		"string_not_equal_ignore_case": func(s1, s2 string) bool {
			return !strings.EqualFold(s1, s2)
		},
		"for_any_value:string_not_equal_ignore_case": func(s1, s2 string) bool {
			return !strings.EqualFold(s1, s2)
		},
	}
)
