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

// Package utils contains common utility functions
package utils

// StringSet is a set of strings
type StringSet interface {
	Add(val string)

	Remove(val string)

	ToSlice() []string

	Range(func(val string) bool)
}

// NewStringSet returns a new StringSet
func NewStringSet() StringSet {
	return &stringSet{
		container: make(map[string]struct{}),
	}
}

type stringSet struct {
	container map[string]struct{}
}

// Add adds a string to the set
func (set *stringSet) Add(val string) {
	set.container[val] = struct{}{}
}

// Remove removes a string from the set
func (set *stringSet) Remove(val string) {
	delete(set.container, val)
}

func (set *stringSet) ToSlice() []string {
	ret := make([]string, 0, len(set.container))

	for k := range set.container {
		ret = append(ret, k)
	}

	return ret
}

func (set *stringSet) Range(fn func(val string) bool) {

	for k := range set.container {
		if !fn(k) {
			break
		}
	}

}
