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

package service

import (
	"testing"

	"github.com/polarismesh/polaris/common/model"
)

func Test_matchServiceFilter_ignoreServiceCI(t *testing.T) {
	type args struct {
		svc       *model.Service
		svcFilter map[string]string
		matchName bool
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test_1",
			args: args{
				svc: &model.Service{
					Namespace: "default",
					Name:      "TestCaseService",
				},
				svcFilter: map[string]string{
					"name": "TestCaseService*",
				},
				matchName: true,
			},
			want: true,
		},
		{
			name: "Test_1",
			args: args{
				svc: &model.Service{
					Namespace: "default",
					Name:      "TestCaseService",
				},
				svcFilter: map[string]string{
					"name": "testCaseService*",
				},
				matchName: true,
			},
			want: true,
		},
		{
			name: "Test_1",
			args: args{
				svc: &model.Service{
					Namespace: "default",
					Name:      "TestCaseService",
				},
				svcFilter: map[string]string{
					"name": "testCase*",
				},
				matchName: true,
			},
			want: true,
		},
		{
			name: "Test_1",
			args: args{
				svc: &model.Service{
					Namespace: "default",
					Name:      "TestCaseService",
				},
				svcFilter: map[string]string{
					"name": "Testcase*",
				},
				matchName: true,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchServiceFilter(tt.args.svc, tt.args.svcFilter, tt.args.matchName, false); got != tt.want {
				t.Errorf("matchServiceFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_matchServiceFilter_business(t *testing.T) {
	type args struct {
		svc       *model.Service
		svcFilter map[string]string
		matchName bool
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test_1",
			args: args{
				svc: &model.Service{
					Namespace: "default",
					Business:  "TSE",
				},
				svcFilter: map[string]string{
					"business": "TSE",
				},
				matchName: false,
			},
			want: true,
		},
		{
			name: "Test_1",
			args: args{
				svc: &model.Service{
					Namespace: "default",
					Business:  "TSE",
				},
				svcFilter: map[string]string{
					"business": "tse",
				},
				matchName: false,
			},
			want: true,
		},
		{
			name: "Test_1",
			args: args{
				svc: &model.Service{
					Namespace: "default",
					Business:  "TSe",
				},
				svcFilter: map[string]string{
					"business": "tse",
				},
				matchName: false,
			},
			want: true,
		},
		{
			name: "Test_1",
			args: args{
				svc: &model.Service{
					Namespace: "default",
					Business:  "Te",
				},
				svcFilter: map[string]string{
					"business": "tse",
				},
				matchName: false,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchServiceFilter(tt.args.svc, tt.args.svcFilter, tt.args.matchName, false); got != tt.want {
				t.Errorf("matchServiceFilter() = %v, want %v, args %#v", got, tt.want, tt.args)
			}
		})
	}
}
