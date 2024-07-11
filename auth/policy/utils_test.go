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

package policy_test

import (
	"strings"
	"testing"

	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/polarismesh/polaris/auth/policy"
	"github.com/polarismesh/polaris/common/utils"
)

func TestCheckName(t *testing.T) {
	tests := []struct {
		name     string
		input    *wrappers.StringValue
		expected error
	}{
		{
			name:     "测试空名称",
			input:    nil,
			expected: errors.New(utils.NilErrString),
		},
		{
			name:     "测试空名称",
			input:    utils.NewStringValue(""),
			expected: errors.New(utils.EmptyErrString),
		},
		{
			name:     "测试非法用户名",
			input:    utils.NewStringValue("polariadmin"),
			expected: errors.New("illegal username"),
		},
		{
			name:     "测试名称长度超过限制",
			input:    utils.NewStringValue(strings.Repeat("a", utils.MaxNameLength+1)),
			expected: errors.New("name too long"),
		},
		{
			name:     "测试包含无效字符的名称",
			input:    utils.NewStringValue("invalid*name"),
			expected: errors.New("name contains invalid character"),
		},
		{
			name:     "测试有效的名称",
			input:    utils.NewStringValue("valid_name"),
			expected: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := policy.CheckName(tc.input)
			if tc.expected == nil && actual != nil {
				t.Fatal(tc.name)
			}
			if tc.expected != nil && actual == nil {
				t.Fatal(nil)
			}
			if tc.expected != nil && actual != nil {
				assert.Equal(t, tc.expected.Error(), actual.Error())
			}
		})
	}
}
