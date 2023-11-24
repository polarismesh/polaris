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

package gray

import (
	"testing"

	"github.com/golang/protobuf/ptypes/wrappers"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"github.com/stretchr/testify/assert"
)

func TestMatch(t *testing.T) {
	// 1. 全匹配
	matchKv := []*apimodel.ClientLabel{
		{
			Key: "ip",
			Value: &apimodel.MatchString{
				Type:  apimodel.MatchString_EXACT,
				Value: &wrappers.StringValue{Value: "127.0.0.1"},
			},
		},
	}

	ok := grayMatch(matchKv, map[string]string{
		"ip": "127.0.0.1",
	})
	assert.Equal(t, ok, true)

	// 2. in 匹配
	matchKv = []*apimodel.ClientLabel{
		{
			Key: "ip",
			Value: &apimodel.MatchString{
				Type:  apimodel.MatchString_IN,
				Value: &wrappers.StringValue{Value: "127.0.0.1,196.10.10.1"},
			},
		},
	}

	ok = grayMatch(matchKv, map[string]string{
		"ip": "127.0.0.1",
	})
	assert.Equal(t, ok, true)
}
