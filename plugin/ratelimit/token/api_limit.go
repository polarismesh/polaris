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

package token

import (
	"errors"
	"sync"

	"golang.org/x/time/rate"
)

// apiRatelimit 接口限流类
type apiRatelimit struct {
	rules  map[string]*BucketRatelimit // 存储规则
	apis   sync.Map                    // 存储api -> apiLimiter
	config *APILimitConfig
}

// newAPIRatelimit 新建一个接口限流类
func newAPIRatelimit(config *APILimitConfig) (*apiRatelimit, error) {
	art := &apiRatelimit{}
	if err := art.initialize(config); err != nil {
		return nil, err
	}

	return art, nil
}

// initialize 接口限流具体实现
func (art *apiRatelimit) initialize(config *APILimitConfig) error {
	art.config = config
	if config == nil || !config.Open {
		log.Infof("[Plugin][%s] api rate limit is not open", PluginName)
		return nil
	}

	log.Infof("[Plugin][%s] api ratelimit open", PluginName)
	if err := art.parseRules(config.Rules); err != nil {
		return err
	}
	if err := art.parseApis(config.Apis); err != nil {
		return err
	}
	return nil
}

// parseRules 解析限流规则
func (art *apiRatelimit) parseRules(rules []*RateLimitRule) error {
	if len(rules) == 0 {
		return errors.New("invalid api rate limit config, rules are empty")
	}

	art.rules = make(map[string]*BucketRatelimit, len(rules))
	for _, entry := range rules {
		if entry.Name == "" {
			return errors.New("invalid api rate limit config, some rules name are empty")
		}
		if entry.Limit == nil {
			return errors.New("invalid api rate limit config, some rules limit are null")
		}
		if entry.Limit.Open && (entry.Limit.Bucket <= 0 || entry.Limit.Rate <= 0) {
			return errors.New("invalid api rate limit config, rules bucket or rate is more than 0")
		}
		art.rules[entry.Name] = entry.Limit
	}

	return nil
}

// parseApis 解析每个api的限流
func (art *apiRatelimit) parseApis(apis []*APILimitInfo) error {
	if len(apis) == 0 {
		return errors.New("invalid api rate limit config, apis are empty")
	}

	for _, entry := range apis {
		if entry.Name == "" {
			return errors.New("invalid api rate limit config, api name is empty")
		}
		if entry.Rule == "" {
			return errors.New("invalid api rate limit config, api rule is empty")
		}

		limit, ok := art.rules[entry.Rule]
		if !ok {
			return errors.New("invalid api rate limit config, api rule is not found")
		}
		art.createLimiter(entry.Name, limit)
	}

	return nil
}

// createLimiter 创建一个私有limiter
func (art *apiRatelimit) createLimiter(name string, limit *BucketRatelimit) *apiLimiter {
	limiter := newAPILimiter(name, limit.Open, limit.Rate, limit.Bucket)
	art.apis.Store(name, limiter)
	return limiter
}

// 获取limiter
func (art *apiRatelimit) acquireLimiter(name string) *apiLimiter {
	if value, ok := art.apis.Load(name); ok {
		return value.(*apiLimiter)
	}

	return nil
}

// 系统是否开启API限流
func (art *apiRatelimit) isOpen() bool {
	return art.config != nil && art.config.Open
}

// 令牌桶限流
func (art *apiRatelimit) allow(name string) bool {
	// 检查系统是否开启API限流
	// 系统不开启API限流，则返回true通过
	if !art.isOpen() {
		return true
	}

	limiter := art.acquireLimiter(name)
	if limiter == nil {
		// 找不到limiter，默认返回true
		return true
	}

	return limiter.Allow()
}

// 封装rate.Limiter
// 每个API接口对应一个apiLimiter
type apiLimiter struct {
	open          bool   // 该接口是否开启限流
	name          string // 接口名
	*rate.Limiter        // 令牌桶对象
}

// newAPILimiter 新建一个apiLimiter
func newAPILimiter(name string, open bool, r int, b int) *apiLimiter {
	limiter := &apiLimiter{
		open:    false,
		name:    name,
		Limiter: nil,
	}
	if !open {
		return limiter
	}

	limiter.open = true
	limiter.Limiter = rate.NewLimiter(rate.Limit(r), b)
	return limiter
}

// Allow 继承rate.Limiter.Allow函数
func (a *apiLimiter) Allow() bool {
	// 当前接口不开启限流
	if !a.open {
		return true
	}

	return a.Limiter.Allow()
}
