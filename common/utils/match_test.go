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
	"testing"

	regexp "github.com/dlclark/regexp2"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestMatchString(t *testing.T) {
	type args struct {
		srcMetaValue   string
		matchValule    *apimodel.MatchString
		regexToPattern func(string) *regexp.Regexp
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// 测试等值匹配
		{
			name: "等式匹配",
			args: args{
				srcMetaValue: "123",
				matchValule: &apimodel.MatchString{
					Type: apimodel.MatchString_EXACT,
					Value: &wrapperspb.StringValue{
						Value: "123",
					},
					ValueType: apimodel.MatchString_TEXT,
				},
				regexToPattern: func(s string) *regexp.Regexp {
					return regexp.MustCompile(s, regexp.RE2)
				},
			},
			want: true,
		},
		{
			name: "等式匹配",
			args: args{
				srcMetaValue: "1234",
				matchValule: &apimodel.MatchString{
					Type: apimodel.MatchString_EXACT,
					Value: &wrapperspb.StringValue{
						Value: "123",
					},
					ValueType: apimodel.MatchString_TEXT,
				},
				regexToPattern: func(s string) *regexp.Regexp {
					return regexp.MustCompile(s, regexp.RE2)
				},
			},
			want: false,
		},
		// 不等式匹配
		{
			name: "不等式匹配",
			args: args{
				srcMetaValue: "1234",
				matchValule: &apimodel.MatchString{
					Type: apimodel.MatchString_NOT_EQUALS,
					Value: &wrapperspb.StringValue{
						Value: "123",
					},
					ValueType: apimodel.MatchString_TEXT,
				},
				regexToPattern: func(s string) *regexp.Regexp {
					return regexp.MustCompile(s, regexp.RE2)
				},
			},
			want: true,
		},
		{
			name: "不等式匹配",
			args: args{
				srcMetaValue: "123",
				matchValule: &apimodel.MatchString{
					Type: apimodel.MatchString_NOT_EQUALS,
					Value: &wrapperspb.StringValue{
						Value: "123",
					},
					ValueType: apimodel.MatchString_TEXT,
				},
				regexToPattern: func(s string) *regexp.Regexp {
					return regexp.MustCompile(s, regexp.RE2)
				},
			},
			want: false,
		},
		// 包含匹配
		{
			name: "包含匹配",
			args: args{
				srcMetaValue: "123",
				matchValule: &apimodel.MatchString{
					Type: apimodel.MatchString_IN,
					Value: &wrapperspb.StringValue{
						Value: "123,456,123456",
					},
					ValueType: apimodel.MatchString_TEXT,
				},
				regexToPattern: func(s string) *regexp.Regexp {
					return regexp.MustCompile(s, regexp.RE2)
				},
			},
			want: true,
		},
		{
			name: "包含匹配",
			args: args{
				srcMetaValue: "123",
				matchValule: &apimodel.MatchString{
					Type: apimodel.MatchString_IN,
					Value: &wrapperspb.StringValue{
						Value: "456,123456",
					},
					ValueType: apimodel.MatchString_TEXT,
				},
				regexToPattern: func(s string) *regexp.Regexp {
					return regexp.MustCompile(s, regexp.RE2)
				},
			},
			want: false,
		},
		{
			name: "包含匹配",
			args: args{
				srcMetaValue: "123",
				matchValule: &apimodel.MatchString{
					Type: apimodel.MatchString_IN,
					Value: &wrapperspb.StringValue{
						Value: "456,asdad",
					},
					ValueType: apimodel.MatchString_TEXT,
				},
				regexToPattern: func(s string) *regexp.Regexp {
					return regexp.MustCompile(s, regexp.RE2)
				},
			},
			want: false,
		},
		// 不包含匹配
		{
			name: "不包含匹配",
			args: args{
				srcMetaValue: "123",
				matchValule: &apimodel.MatchString{
					Type: apimodel.MatchString_NOT_IN,
					Value: &wrapperspb.StringValue{
						Value: "123,456,123456",
					},
					ValueType: apimodel.MatchString_TEXT,
				},
				regexToPattern: func(s string) *regexp.Regexp {
					return regexp.MustCompile(s, regexp.RE2)
				},
			},
			want: false,
		},
		{
			name: "不包含匹配",
			args: args{
				srcMetaValue: "123",
				matchValule: &apimodel.MatchString{
					Type: apimodel.MatchString_NOT_IN,
					Value: &wrapperspb.StringValue{
						Value: "456,asdad",
					},
					ValueType: apimodel.MatchString_TEXT,
				},
				regexToPattern: func(s string) *regexp.Regexp {
					return regexp.MustCompile(s, regexp.RE2)
				},
			},
			want: true,
		},
		{
			name: "不包含匹配",
			args: args{
				srcMetaValue: "123",
				matchValule: &apimodel.MatchString{
					Type: apimodel.MatchString_NOT_IN,
					Value: &wrapperspb.StringValue{
						Value: "456,123456",
					},
					ValueType: apimodel.MatchString_TEXT,
				},
				regexToPattern: func(s string) *regexp.Regexp {
					return regexp.MustCompile(s, regexp.RE2)
				},
			},
			want: true,
		},
		// 范围匹配
		{
			name: "范围匹配",
			args: args{
				srcMetaValue: "123",
				matchValule: &apimodel.MatchString{
					Type: apimodel.MatchString_RANGE,
					Value: &wrapperspb.StringValue{
						Value: "123~123456",
					},
					ValueType: apimodel.MatchString_TEXT,
				},
				regexToPattern: func(s string) *regexp.Regexp {
					return regexp.MustCompile(s, regexp.RE2)
				},
			},
			want: true,
		},
		{
			name: "范围匹配",
			args: args{
				srcMetaValue: "1234",
				matchValule: &apimodel.MatchString{
					Type: apimodel.MatchString_RANGE,
					Value: &wrapperspb.StringValue{
						Value: "123~123456",
					},
					ValueType: apimodel.MatchString_TEXT,
				},
				regexToPattern: func(s string) *regexp.Regexp {
					return regexp.MustCompile(s, regexp.RE2)
				},
			},
			want: true,
		},
		{
			name: "范围匹配",
			args: args{
				srcMetaValue: "12",
				matchValule: &apimodel.MatchString{
					Type: apimodel.MatchString_RANGE,
					Value: &wrapperspb.StringValue{
						Value: "123~123456",
					},
					ValueType: apimodel.MatchString_TEXT,
				},
				regexToPattern: func(s string) *regexp.Regexp {
					return regexp.MustCompile(s, regexp.RE2)
				},
			},
			want: false,
		},
		{
			name: "范围匹配",
			args: args{
				srcMetaValue: "12345643",
				matchValule: &apimodel.MatchString{
					Type: apimodel.MatchString_RANGE,
					Value: &wrapperspb.StringValue{
						Value: "123~123456",
					},
					ValueType: apimodel.MatchString_TEXT,
				},
				regexToPattern: func(s string) *regexp.Regexp {
					return regexp.MustCompile(s, regexp.RE2)
				},
			},
			want: false,
		},
		{
			name: "范围匹配",
			args: args{
				srcMetaValue: "12345643",
				matchValule: &apimodel.MatchString{
					Type: apimodel.MatchString_RANGE,
					Value: &wrapperspb.StringValue{
						Value: "a~123456",
					},
					ValueType: apimodel.MatchString_TEXT,
				},
				regexToPattern: func(s string) *regexp.Regexp {
					return regexp.MustCompile(s, regexp.RE2)
				},
			},
			want: false,
		},
		{
			name: "范围匹配",
			args: args{
				srcMetaValue: "12345643",
				matchValule: &apimodel.MatchString{
					Type: apimodel.MatchString_RANGE,
					Value: &wrapperspb.StringValue{
						Value: "1231~3a",
					},
					ValueType: apimodel.MatchString_TEXT,
				},
				regexToPattern: func(s string) *regexp.Regexp {
					return regexp.MustCompile(s, regexp.RE2)
				},
			},
			want: false,
		},
		{
			name: "范围匹配",
			args: args{
				srcMetaValue: "wx33",
				matchValule: &apimodel.MatchString{
					Type: apimodel.MatchString_RANGE,
					Value: &wrapperspb.StringValue{
						Value: "1231~123123",
					},
					ValueType: apimodel.MatchString_TEXT,
				},
				regexToPattern: func(s string) *regexp.Regexp {
					return regexp.MustCompile(s, regexp.RE2)
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MatchString(tt.args.srcMetaValue, tt.args.matchValule, tt.args.regexToPattern); got != tt.want {
				t.Errorf("MatchString() = %v, want %v", got, tt.want)
			}
		})
	}
}
