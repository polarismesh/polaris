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
	"errors"
	"fmt"
	"time"

	"github.com/boltdb/bolt"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store"
)

const (
	// rule 相关信息以及映射
	tblRateLimitConfig   string = "ratelimit_config"
	tblRateLimitRevision string = "ratelimit_revision"
)

type rateLimitStore struct {
	handler BoltHandler
}

// CreateRateLimit 新增限流规则
func (r *rateLimitStore) CreateRateLimit(limit *model.RateLimit) error {
	if limit.ID == "" || limit.ServiceID == "" || limit.Revision == "" {
		return errors.New("[Store][database] create rate limit missing some params")
	}

	return r.createRateLimit(limit)
}

// UpdateRateLimit 更新限流规则
func (r *rateLimitStore) UpdateRateLimit(limiting *model.RateLimit) error {
	//TODO
	return nil
}

// DeleteRateLimit 删除限流规则
func (r *rateLimitStore) DeleteRateLimit(limiting *model.RateLimit) error {
	//TODO
	return nil
}

// GetExtendRateLimits 根据过滤条件拉取限流规则
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

// createRateLimit save model.RateLimit and model.RateLimitRevision
//  @receiver r *rateLimitStore
//  @param limit current limiting configuration data to be saved
//  @return error
func (r *rateLimitStore) createRateLimit(limit *model.RateLimit) error {
	handler := r.handler
	return handler.Execute(true, func(tx *bolt.Tx) error {
		// create ratelimit_config
		if err := saveValue(tx, tblRateLimitConfig, limit.ID, limit); err != nil {
			log.Errorf("[Store][RateLimit] create rate_limit(%s, %s) err: %s",
				limit.ID, limit.ServiceID, err.Error())
			return store.Error(err)
		}

		// create ratelimit_version
		lastVer := &model.RateLimitRevision{
			ServiceID:    limit.ServiceID,
			LastRevision: limit.Revision,
			ModifyTime:   time.Now(),
		}

		recordKey := fmt.Sprintf("%s@%s", lastVer.ServiceID, lastVer.LastRevision)

		if err := saveValue(tx, tblRateLimitRevision, recordKey, lastVer); err != nil {
			log.Errorf("[Store][RateLimit] create ratelimit_revision(%s, %s) err: %s",
				limit.ID, limit.ServiceID, err.Error())
			return store.Error(err)
		}

		return nil
	})

}

func (r *rateLimitStore) updateRateLimit(limit *model.RateLimit) error {
	handler := r.handler
	return handler.Execute(true, func(tx *bolt.Tx) error {

		limit.ModifyTime = time.Now()
		// create ratelimit_config
		if err := saveValue(tx, tblRateLimitConfig, limit.ID, limit); err != nil {
			log.Errorf("[Store][RateLimit] create rate_limit(%s, %s) err: %s",
				limit.ID, limit.ServiceID, err.Error())
			return store.Error(err)
		}

		// create ratelimit_version
		lastVer := &model.RateLimitRevision{
			ServiceID:    limit.ServiceID,
			LastRevision: limit.Revision,
			ModifyTime:   time.Now(),
		}

		recordKey := fmt.Sprintf("%s@%s", lastVer.ServiceID, lastVer.LastRevision)

		if err := saveValue(tx, tblRateLimitRevision, recordKey, lastVer); err != nil {
			log.Errorf("[Store][RateLimit] create ratelimit_revision(%s, %s) err: %s",
				limit.ID, limit.ServiceID, err.Error())
			return store.Error(err)
		}

		return nil
	})

}
