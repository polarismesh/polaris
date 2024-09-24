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

// Resp ckv任务结果
type Resp struct {
	Value       string
	Values      []interface{}
	Err         error
	Exists      bool
	Compatible  bool
	shouldRetry bool
}

// RedisObject 序列化对象
type RedisObject interface {
	// Serialize 序列化成字符串
	Serialize(compatible bool) string
	// Deserialize 反序列为对象
	Deserialize(value string, compatible bool) error
}

type Pool interface {
	// Start 启动ckv连接池工作
	Start()
	// Sdd 使用连接池，向redis发起Sdd请求
	Sdd(id string, members []string) *Resp
	// Srem 使用连接池，向redis发起Srem请求
	Srem(id string, members []string) *Resp
	// Get 使用连接池，向redis发起Get请求
	Get(id string) *Resp
	// MGet 使用连接池，向redis发起MGet请求
	MGet(keys []string) *Resp
	// Set 使用连接池，向redis发起Set请求
	Set(id string, redisObj RedisObject) *Resp
	// Del 使用连接池，向redis发起Del请求
	Del(id string) *Resp
	// RecoverTimeSec the time second record when recover
	RecoverTimeSec() int64
}
