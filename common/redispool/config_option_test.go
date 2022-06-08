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
package redispool

import (
	"testing"
	"time"

	commontime "github.com/polarismesh/polaris-server/common/time"
	"github.com/stretchr/testify/assert"
)

func TestNewRedisClient(t *testing.T) {
	config := DefaultConfig()

	t.Log("before config: ", config)

	// mock config read
	config.KvAddr = "127.0.0.1:6379"
	config.MaxConnAge = commontime.Duration(1000 * time.Second)
	config.MinIdleConns = 30

	_ = NewRedisClient(WithConfig(config))
	assert.Equal(t, config.MaxConnAge, commontime.Duration(1000*time.Second))
	assert.Equal(t, config.MinIdleConns, 30)

	t.Log("after config: ", config)

	// client := NewRedisClient(WithConfig(config))
	// err := client.Set(context.Background(), "polaris", 1, 60*time.Second).Err()
	// if err != nil {
	// 	t.Fatalf("test redis client error:%v", err)
	// }

	t.Log("test success")
}
