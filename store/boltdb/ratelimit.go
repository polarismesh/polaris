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
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

var _ store.RateLimitStore = (*rateLimitStore)(nil)

var (
	ErrBadParam       = errors.New("missing some params")
	ErrMultipleResult = errors.New("multiple ratelimit find")
)

const (
	// rule 相关信息以及映射
	tblRateLimitConfig       string = "ratelimit_config"
	RateLimitFieldID         string = "ID"
	RateLimitFieldServiceID  string = "ServiceID"
	RateLimitFieldClusterID  string = "ClusterID"
	RateLimitFieldEnableTime string = "EnableTime"
	RateLimitFieldName       string = "Name"
	RateLimitFieldDisable    string = "Disable"
	RateLimitFieldMethod     string = "Method"
	RateLimitFieldLabels     string = "Labels"
	RateLimitFieldPriority   string = "Priority"
	RateLimitFieldRule       string = "Rule"
	RateLimitFieldRevision   string = "Revision"
	RateLimitFieldValid      string = "Valid"
	RateLimitFieldCreateTime string = "CreateTime"
	RateLimitFieldModifyTime string = "ModifyTime"
	RateConfFieldMtime       string = "ModifyTime"
	RateConfFieldServiceID   string = "ServiceID"
	RateConfFieldValid       string = "Valid"
)

type rateLimitStore struct {
	handler BoltHandler
}

// CreateRateLimit 新增限流规则
func (r *rateLimitStore) CreateRateLimit(limit *model.RateLimit) error {
	if limit.ID == "" || limit.Revision == "" {
		log.Error("[Store][boltdb] create ratelimit missing some params")
		return ErrBadParam
	}

	return r.createRateLimit(limit)
}

// UpdateRateLimit 更新限流规则
func (r *rateLimitStore) UpdateRateLimit(limit *model.RateLimit) error {
	if limit.ID == "" || limit.Revision == "" {
		log.Error("[Store][boltdb] update ratelimit missing some params")
		return ErrBadParam
	}

	return r.updateRateLimit(limit)
}

// EnableRateLimit 激活限流规则
func (r *rateLimitStore) EnableRateLimit(limit *model.RateLimit) error {
	if limit.ID == "" || limit.Revision == "" {
		log.Error("[Store][boltdb] update ratelimit missing some params")
		return ErrBadParam
	}
	return r.enableRateLimit(limit)
}

// DeleteRateLimit 删除限流规则
func (r *rateLimitStore) DeleteRateLimit(limit *model.RateLimit) error {
	if limit.ID == "" || limit.Revision == "" {
		log.Error("[Store][boltdb] delete ratelimit missing some params")
		return ErrBadParam
	}

	return r.deleteRateLimit(limit)
}

// GetExtendRateLimits 根据过滤条件拉取限流规则
func (r *rateLimitStore) GetExtendRateLimits(
	query map[string]string, offset uint32, limit uint32) (uint32, []*model.ExtendRateLimit, error) {

	handler := r.handler
	fields := append(utils.CollectMapKeys(query), RateConfFieldServiceID, RateConfFieldValid)

	result, err := handler.LoadValuesByFilter(tblRateLimitConfig, fields, &model.RateLimit{},
		func(m map[string]interface{}) bool {
			validVal, ok := m[RateConfFieldValid]
			if ok && !validVal.(bool) {
				return false
			}
			delete(m, RateConfFieldValid)

			for k, v := range query {
				if k == "name" || k == "method" || k == "labels" {
					if !strings.Contains(m[k].(string), v) {
						return false
					}
				} else if k == "disable" {
					if v != strconv.FormatBool(m[RateLimitFieldDisable].(bool)) {
						return false
					}
				} else {
					qV := m[k]
					if !reflect.DeepEqual(qV, v) {
						return false
					}
				}
			}
			return true
		})

	if err != nil {
		return 0, nil, err
	}
	if len(result) == 0 {
		return 0, []*model.ExtendRateLimit{}, nil
	}

	out := make([]*model.ExtendRateLimit, 0, len(result))
	for _, r := range result {
		var temp model.ExtendRateLimit
		temp.RateLimit = r.(*model.RateLimit)

		out = append(out, &temp)
	}

	return uint32(len(result)), getRealRateConfList(out, offset, limit), nil
}

// GetRateLimitWithID 根据限流ID拉取限流规则
func (r *rateLimitStore) GetRateLimitWithID(id string) (*model.RateLimit, error) {
	if id == "" {
		return nil, ErrBadParam
	}

	handler := r.handler
	result, err := handler.LoadValues(tblRateLimitConfig, []string{id}, &model.RateLimit{})

	if err != nil {
		log.Errorf("[Store][boltdb] get rate limit fail : %s", err.Error())
		return nil, err
	}

	if len(result) > 1 {
		return nil, ErrMultipleResult
	}

	if len(result) == 0 {
		return nil, nil
	}

	rateLimitRet := result[id].(*model.RateLimit)
	if rateLimitRet.Valid {
		return rateLimitRet, nil
	}

	return nil, nil
}

// GetRateLimitsForCache 根据修改时间拉取增量限流规则及最新版本号
func (r *rateLimitStore) GetRateLimitsForCache(mtime time.Time,
	firstUpdate bool) ([]*model.RateLimit, error) {
	handler := r.handler

	if firstUpdate {
		mtime = time.Time{}
	}

	var (
		fields = []string{RateConfFieldMtime, RateConfFieldServiceID}
	)
	limitResults, err := handler.LoadValuesByFilter(tblRateLimitConfig, fields, &model.RateLimit{},
		func(m map[string]interface{}) bool {
			mt := m[RateConfFieldMtime].(time.Time)
			isAfter := !mt.Before(mtime)
			return isAfter
		})

	if err != nil {
		return nil, err
	}

	if len(limitResults) == 0 {
		return []*model.RateLimit{}, nil
	}

	limits := make([]*model.RateLimit, 0, len(limitResults))

	for i := range limitResults {
		rule := limitResults[i].(*model.RateLimit)
		limits = append(limits, rule)
	}

	return limits, nil
}

// createRateLimit save model.RateLimit and model.RateLimitRevision
//
//	@receiver r *rateLimitStore
//	@param limit current limiting configuration data to be saved
//	@return error
func (r *rateLimitStore) createRateLimit(limit *model.RateLimit) error {
	handler := r.handler
	tNow := time.Now()
	limit.CreateTime = tNow
	limit.ModifyTime = tNow
	if !limit.Disable {
		limit.EnableTime = tNow
	} else {
		limit.EnableTime = time.Unix(0, 0)
	}
	limit.Valid = true
	return handler.Execute(true, func(tx *bolt.Tx) error {
		// create ratelimit_config
		if err := saveValue(tx, tblRateLimitConfig, limit.ID, limit); err != nil {
			log.Errorf("[Store][RateLimit] create rate_limit(%s, %s), %+v, err: %s",
				limit.ID, limit.ServiceID, limit, err.Error())
			return err
		}
		return nil
	})
}

// enableRateLimit
//
//	@receiver r
//	@param limit
//	@return error
func (r *rateLimitStore) enableRateLimit(limit *model.RateLimit) error {
	handler := r.handler
	return handler.Execute(true, func(tx *bolt.Tx) error {
		properties := make(map[string]interface{})
		properties[RateLimitFieldDisable] = limit.Disable
		properties[RateLimitFieldRevision] = limit.Revision
		properties[RateLimitFieldModifyTime] = time.Now()
		if limit.Disable {
			properties[RateLimitFieldEnableTime] = time.Unix(0, 0)
		} else {
			properties[RateLimitFieldEnableTime] = time.Now()
		}
		// create ratelimit_config
		if err := updateValue(tx, tblRateLimitConfig, limit.ID, properties); err != nil {
			log.Errorf("[Store][RateLimit] update rate_limit(%s, %s) err: %s",
				limit.ID, limit.ServiceID, err.Error())
			return err
		}
		return nil
	})
}

// updateRateLimit
//
//	@receiver r
//	@param limit
//	@return error
func (r *rateLimitStore) updateRateLimit(limit *model.RateLimit) error {
	handler := r.handler
	return handler.Execute(true, func(tx *bolt.Tx) error {
		properties := make(map[string]interface{})
		properties[RateLimitFieldName] = limit.Name
		properties[RateLimitFieldServiceID] = limit.ServiceID
		properties[RateLimitFieldMethod] = limit.Method
		properties[RateLimitFieldDisable] = limit.Disable
		properties[RateLimitFieldLabels] = limit.Labels
		properties[RateLimitFieldPriority] = limit.Priority
		properties[RateLimitFieldRule] = limit.Rule
		properties[RateLimitFieldRevision] = limit.Revision
		properties[RateLimitFieldModifyTime] = time.Now()
		if limit.Disable {
			properties[RateLimitFieldEnableTime] = time.Unix(0, 0)
		} else {
			properties[RateLimitFieldEnableTime] = time.Now()
		}
		// create ratelimit_config
		if err := updateValue(tx, tblRateLimitConfig, limit.ID, properties); err != nil {
			log.Errorf("[Store][RateLimit] update rate_limit(%s, %s) err: %s",
				limit.ID, limit.ServiceID, err.Error())
			return err
		}
		return nil
	})
}

// deleteRateLimit
//
//	@receiver r
//	@param limit
//	@return error
func (r *rateLimitStore) deleteRateLimit(limit *model.RateLimit) error {
	handler := r.handler

	return handler.Execute(true, func(tx *bolt.Tx) error {

		properties := make(map[string]interface{})
		properties[RateLimitFieldValid] = false
		properties[RateLimitFieldModifyTime] = time.Now()

		if err := updateValue(tx, tblRateLimitConfig, limit.ID, properties); err != nil {
			log.Errorf("[Store][RateLimit] delete rate_limit(%s, %s) err: %s",
				limit.ID, limit.ServiceID, err.Error())
			return err
		}
		return nil
	})
}

func getRealRateConfList(routeConf []*model.ExtendRateLimit, offset, limit uint32) []*model.ExtendRateLimit {

	beginIndex := offset
	endIndex := beginIndex + limit
	totalCount := uint32(len(routeConf))
	// handle invalid offset, limit
	if totalCount == 0 {
		return routeConf
	}
	if beginIndex >= endIndex {
		return routeConf
	}
	if beginIndex >= totalCount {
		return routeConf
	}
	if endIndex > totalCount {
		endIndex = totalCount
	}

	sort.Slice(routeConf, func(i, j int) bool {
		// sort by modify time
		if routeConf[i].RateLimit.ModifyTime.After(routeConf[j].RateLimit.ModifyTime) {
			return true
		} else if routeConf[i].RateLimit.ModifyTime.Before(routeConf[j].RateLimit.ModifyTime) {
			return false
		}
		return strings.Compare(routeConf[i].RateLimit.ID, routeConf[j].RateLimit.ID) < 0
	})

	return routeConf[beginIndex:endIndex]
}
