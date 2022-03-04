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
	"fmt"
	lru "github.com/hashicorp/golang-lru"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/plugin"
	"golang.org/x/time/rate"
)

// 资源限制器
type resourceRatelimit struct {
	typStr    string
	resources *lru.Cache
	whiteList map[string]bool
	config    *ResourceLimitConfig
}

// 新建资源限制器
func newResourceRatelimit(typ plugin.RatelimitType, config *ResourceLimitConfig) (*resourceRatelimit, error) {
	r := &resourceRatelimit{typStr: plugin.RatelimitStr[typ]}
	if err := r.initialize(config); err != nil {
		return nil, err
	}

	return r, nil
}

// initialize
func (r *resourceRatelimit) initialize(config *ResourceLimitConfig) error {
	r.config = config
	if config == nil || !config.Open {
		log.Infof("[Plugin][%s] resource(%s) ratelimit is not open", PluginName, r.typStr)
		return nil
	}

	if config.Global == nil {
		return fmt.Errorf("resource(%s) global ratelimit rule is empty", r.typStr)
	}
	if config.Global.Bucket <= 0 || config.Global.Rate <= 0 {
		return fmt.Errorf("resource(%s) ratelimit global bucket or rate invalid", r.typStr)
	}
	if config.MaxResourceCacheAmount <= 0 {
		return fmt.Errorf("resource(%s) max resource amount is invalid", r.typStr)
	}

	cache, err := lru.New(config.MaxResourceCacheAmount)
	if err != nil {
		log.Errorf("[Plugin][%s] resource(%s) ratelimit create new lru cache err: %s",
			PluginName, r.typStr, err.Error())
		return err
	}
	r.resources = cache

	r.whiteList = make(map[string]bool)
	for _, item := range config.WhiteList {
		r.whiteList[item] = true
	}

	log.Infof("[Plugin][%s] resource(%s) ratelimit open", PluginName, r.typStr)
	return nil
}

// 限流是否开启
func (r *resourceRatelimit) isOpen() bool {
	return r.config != nil && r.config.Open
}

// 检查是否属于白名单，属于的话，则不限流
func (r *resourceRatelimit) isWhiteList(key string) bool {
	_, ok := r.whiteList[key]
	return ok
}

// 实现limiter
func (r *resourceRatelimit) allow(key string) bool {
	if ok := r.isOpen(); !ok {
		return true
	}
	if ok := r.isWhiteList(key); ok {
		return true
	}

	value, ok := r.resources.Get(key)
	if !ok {
		r.resources.ContainsOrAdd(key,
			rate.NewLimiter(rate.Limit(r.config.Global.Rate), r.config.Global.Bucket))
		// 上面已经加了value，这里正常情况会有value
		value, ok = r.resources.Get(key)
		if !ok {
			// 还找不到，打印日志，返回true
			log.Warnf("[Plugin][%s] not found the resources(%s) key(%s) in the cache",
				PluginName, r.typStr, key)
			return true
		}
	}

	return value.(*rate.Limiter).Allow()
}
