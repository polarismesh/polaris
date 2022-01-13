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

import "github.com/polarismesh/polaris-server/common/model"

type StringSet interface {
	Add(val string)

	Remove(val string)

	ToSlice() []string

	Range(func(val string) bool)
}

func NewStringSet() StringSet {
	return &stringSet{
		container: make(map[string]struct{}),
	}
}

type stringSet struct {
	container map[string]struct{}
}

func (set *stringSet) Add(val string) {
	set.container[val] = emptyVal
}

func (set *stringSet) Remove(val string) {
	delete(set.container, val)
}

func (set *stringSet) ToSlice() []string {
	ret := make([]string, len(set.container))

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

type ServiceSet struct {
	container map[string]*model.Service
}

func NewServiceSet() *ServiceSet {
	return &ServiceSet{
		container: make(map[string]*model.Service),
	}
}

func (set *ServiceSet) Add(val *model.Service) {
	set.container[val.ID] = val
}

func (set *ServiceSet) Remove(val *model.Service) {
	delete(set.container, val.ID)
}

func (set *ServiceSet) ToSlice() []*model.Service {
	ret := make([]*model.Service, len(set.container))

	for _, v := range set.container {
		ret = append(ret, v)
	}

	return ret
}

func (set *ServiceSet) Range(fn func(val *model.Service) bool) {
	for _, v := range set.container {
		if !fn(v) {
			break
		}
	}
}
