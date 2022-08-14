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
 * CONDITIONS OF ANY KIND, either express or Serveried. See the License for the
 * specific language governing permissions and limitations under the License.
 */

package config

import (
	"reflect"
	"sort"
	"testing"
)

func Test_diffTags(t *testing.T) {
	type args struct {
		a map[string]map[string]struct{}
		b map[string]map[string]struct{}
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "",
			args: struct {
				a map[string]map[string]struct{}
				b map[string]map[string]struct{}
			}{
				a: map[string]map[string]struct{}{
					"key-1": {
						"key-1-val-1": {},
						"key-1-val-2": {},
					},
					"key-2": {
						"key-2-val-1": {},
					},
				},
				b: map[string]map[string]struct{}{},
			},
			want: []string{"key-1", "key-1-val-1", "key-1", "key-1-val-2", "key-2", "key-2-val-1"},
		},
		{
			name: "",
			args: struct {
				a map[string]map[string]struct{}
				b map[string]map[string]struct{}
			}{
				a: map[string]map[string]struct{}{
					"key-1": {
						"key-1-val-1": {},
						"key-1-val-2": {},
					},
					"key-2": {
						"key-2-val-1": {},
					},
				},
				b: map[string]map[string]struct{}{
					"key-1": {
						"key-1-val-1": {},
						"key-1-val-2": {},
					},
				},
			},
			want: []string{"key-2", "key-2-val-1"},
		},
		{
			name: "",
			args: struct {
				a map[string]map[string]struct{}
				b map[string]map[string]struct{}
			}{
				a: map[string]map[string]struct{}{
					"key-1": {
						"key-1-val-1": {},
						"key-1-val-2": {},
					},
				},
				b: map[string]map[string]struct{}{
					"key-1": {
						"key-1-val-1": {},
						"key-1-val-2": {},
					},
				},
			},
			want: []string{},
		},
		{
			name: "",
			args: struct {
				a map[string]map[string]struct{}
				b map[string]map[string]struct{}
			}{
				a: map[string]map[string]struct{}{},
				b: map[string]map[string]struct{}{
					"key-1": {
						"key-1-val-1": {},
						"key-1-val-2": {},
					},
				},
			},
			want: []string{},
		},
	}

	// 由于diffTags 返回的切片是由mp遍历得来, 不保证顺序
	diffTagsEqual := func(a, b []string) bool {
		sort.Strings(a)
		sort.Strings(b)
		return reflect.DeepEqual(a, b)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := diffTags(tt.args.a, tt.args.b); !diffTagsEqual(got, tt.want) {
				t.Errorf("diffTags() = %v, want %v", got, tt.want)
			}
		})
	}
}
