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

package boltdb

import (
	"github.com/polarismesh/polaris-server/common/model"
	"time"
)

type rateLimitStore struct {
	handler BoltHandler
}

// 新增限流规则
func (r *rateLimitStore) CreateRateLimit(limiting *model.RateLimit) error {
	//TODO
	return nil
}

// 更新限流规则
func (r *rateLimitStore) UpdateRateLimit(limiting *model.RateLimit) error {
	//TODO
	return nil
}

// 删除限流规则
func (r *rateLimitStore) DeleteRateLimit(limiting *model.RateLimit) error {
	//TODO
	return nil
}

// 根据过滤条件拉取限流规则
func (r *rateLimitStore) GetExtendRateLimits(
	query map[string]string, offset uint32, limit uint32) (uint32, []*model.ExtendRateLimit, error) {
	//TODO
	return 0, nil, nil
}

// 根据限流ID拉取限流规则
func (r *rateLimitStore) GetRateLimitWithID(id string) (*model.RateLimit, error) {
	//TODO
	return nil, nil
}

// 根据修改时间拉取增量限流规则及最新版本号
func (r *rateLimitStore) GetRateLimitsForCache(mtime time.Time, firstUpdate bool) ([]*model.RateLimit, []*model.RateLimitRevision, error) {
	//TODO
	return nil, nil, nil
}