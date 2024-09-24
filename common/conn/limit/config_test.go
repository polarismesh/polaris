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

package connlimit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestParseConnLimitConfig 可以正常解析配置测试
func TestParseConnLimitConfig(t *testing.T) {
	t.Run("可以正常解析配置", func(t *testing.T) {
		options := map[interface{}]interface{}{
			"openConnLimit":  true,
			"maxConnPerHost": 16,
			"maxConnLimit":   128,
			"whiteList":      "127.0.0.1,127.0.0.2,127.0.0.3",
			"readTimeout":    "120s",
		}
		config, err := ParseConnLimitConfig(options)
		assert.Nil(t, err)
		assert.True(t, config.OpenConnLimit)
		assert.Equal(t, config.MaxConnPerHost, 16)
		assert.Equal(t, config.MaxConnLimit, 128)
		assert.Equal(t, config.WhiteList, "127.0.0.1,127.0.0.2,127.0.0.3")
		assert.Equal(t, config.ReadTimeout, time.Second*120)
	})
}
