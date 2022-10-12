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

package lrurate

import (
	"hash/crc32"

	lru "github.com/hashicorp/golang-lru"
	"golang.org/x/time/rate"
)

var (
	ipLruCache      *lru.Cache
	serviceLruCache *lru.Cache
)

// initEnv 初始化lru组件
func initEnv() error {
	var err error

	ipLruCache, err = lru.New(rateLimitIPLruSize)
	if err != nil {
		return err
	}

	serviceLruCache, err = lru.New(rateLimitServiceLruSize)
	if err != nil {
		return err
	}

	return nil
}

// hash crc32取字符串hash值
func hash(str string) uint32 {
	return crc32.ChecksumIEEE([]byte(str))
}

// allowIP ip限流
func allowIP(id string) bool {
	key := hash(id)
	ipLruCache.ContainsOrAdd(key, rate.NewLimiter(rate.Limit(rateLimitIPRate), rateLimitIPBurst))
	if value, ok := ipLruCache.Get(key); ok {
		return value.(*rate.Limiter).Allow()
	}

	return true
}

// allowService service限流
func allowService(id string) bool {
	key := hash(id)
	serviceLruCache.ContainsOrAdd(key, rate.NewLimiter(rate.Limit(rateLimitServiceRate), rateLimitServiceBurst))
	if value, ok := serviceLruCache.Get(key); ok {
		return value.(*rate.Limiter).Allow()
	}

	return true
}
