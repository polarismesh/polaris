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

package config

import (
	"reflect"
	"testing"

	"github.com/polarismesh/polaris/common/metrics"
)

func Test_cleanExpireConfigFileMetricLabel(t *testing.T) {
	metrics.InitMetrics()
	type args struct {
		pre  map[string]map[string]struct{}
		curr map[string]map[string]struct{}
	}
	tests := []struct {
		name  string
		args  args
		want  map[string]struct{}
		want1 map[string]map[string]struct{}
	}{
		{
			name: "01",
			args: args{
				pre: map[string]map[string]struct{}{
					"ns-1": {
						"group-1": {},
					},
				},
				curr: map[string]map[string]struct{}{
					"ns-2": {
						"group-2": {},
					},
				},
			},
			want: map[string]struct{}{
				"ns-1": {},
			},
			want1: map[string]map[string]struct{}{
				"ns-1": {
					"group-1": {},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := cleanExpireConfigFileMetricLabel(tt.args.pre, tt.args.curr)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("cleanExpireConfigFileMetricLabel() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("cleanExpireConfigFileMetricLabel() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
