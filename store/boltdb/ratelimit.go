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
	"strings"
	"time"

	"github.com/boltdb/bolt"

	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
)

var (
	BadParamError       = errors.New("missing some params")
	MultipleResultError = errors.New("multiple ratelimit find")
)

const (
	// rule 相关信息以及映射
	tblRateLimitConfig   string = "ratelimit_config"
	tblRateLimitRevision string = "ratelimit_revision"

	RateLimitFieldID         string = "ID"
	RateLimitFieldServiceID  string = "ServiceID"
	RateLimitFieldClusterID  string = "ClusterID"
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

	RateLimitReviFieldServiceID    string = "ServiceID"
	RateLimitReviFieldLastRevision string = "LastRevision"
	RateLimitReviFieldModifyTime   string = "ModifyTime"
)

type rateLimitStore struct {
	handler BoltHandler
}

// CreateRateLimit 新增限流规则
func (r *rateLimitStore) CreateRateLimit(limit *model.RateLimit) error {
	if limit.ID == "" || limit.ServiceID == "" || limit.Revision == "" {
		log.Error("[Store][boltdb] create ratelimit missing some params")
		return BadParamError
	}

	tNow := time.Now()

	limit.CreateTime = tNow
	limit.ModifyTime = tNow
	limit.Valid = true

	return r.createRateLimit(limit)
}

// UpdateRateLimit 更新限流规则
func (r *rateLimitStore) UpdateRateLimit(limit *model.RateLimit) error {
	if limit.ID == "" || limit.ServiceID == "" || limit.Revision == "" {
		log.Error("[Store][boltdb] update ratelimit missing some params")
		return BadParamError
	}

	return r.updateRateLimit(limit)
}

// DeleteRateLimit 删除限流规则
func (r *rateLimitStore) DeleteRateLimit(limit *model.RateLimit) error {
	if limit.ID == "" || limit.ServiceID == "" || limit.Revision == "" {
		log.Error("[Store][boltdb] delete ratelimit missing some params")
		return BadParamError
	}

	return r.deleteRateLimit(limit)
}

// GetExtendRateLimits 根据过滤条件拉取限流规则
func (r *rateLimitStore) GetExtendRateLimits(
	query map[string]string, offset uint32, limit uint32) (uint32, []*model.ExtendRateLimit, error) {

	svcName, hasSvcName := query["name"]
	svcNs, hasSvcNamespace := query["namespace"]

	handler := r.handler
	fields := []string{SvcFieldName, SvcFieldNamespace, SvcFieldValid}
	services, err := r.handler.LoadValuesByFilter(tblNameService, fields, &model.Service{},
		func(m map[string]interface{}) bool {
			validVal, ok := m[SvcFieldValid]
			if ok && !validVal.(bool) {
				return false
			}

			if hasSvcName && svcName != m[SvcFieldName].(string) {
				return false
			}
			if hasSvcNamespace && svcNs != m[SvcFieldNamespace].(string) {
				return false
			}
			return true
		})

	// Remove query parameters for the service
	delete(query, strings.ToLower(SvcFieldName))
	delete(query, strings.ToLower(svcFieldNamespace))

	fields = append(utils.CollectMapKeys(query), RateConfFieldServiceID, RateConfFieldValid)

	result, err := handler.LoadValuesByFilter(tblRateLimitConfig, fields, &model.RateLimit{},
		func(m map[string]interface{}) bool {
			validVal, ok := m[RateConfFieldValid]
			if ok && !validVal.(bool) {
				return false
			}
			rSvcId := m[RateConfFieldServiceID]
			if _, ok := services[rSvcId.(string)]; !ok {
				return false
			}

			delete(m, RateConfFieldValid)

			for k, v := range query {
				if k == "labels" {
					if !strings.Contains(m[k].(string), v) {
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

	var out []*model.ExtendRateLimit

	for id, r := range result {
		var temp model.ExtendRateLimit
		svc, ok := services[r.(*model.RateLimit).ServiceID].(*model.Service)
		if ok {
			temp.ServiceName = svc.Name
			temp.NamespaceName = svc.Namespace
		} else {
			log.Warnf("[Store][boltdb] get service in ratelimit conf error, service is nil, id: %s", id)
		}
		temp.RateLimit = r.(*model.RateLimit)

		out = append(out, &temp)
	}

	return uint32(len(result)), getRealRateConfList(out, offset, limit), nil
}

// GetRateLimitWithID 根据限流ID拉取限流规则
func (r *rateLimitStore) GetRateLimitWithID(id string) (*model.RateLimit, error) {
	if id == "" {
		return nil, BadParamError
	}

	handler := r.handler
	result, err := handler.LoadValues(tblRateLimitConfig, []string{id}, &model.RateLimit{})

	if err != nil {
		log.Errorf("[Store][boltdb] get rate limit fail : %s", err.Error())
		return nil, err
	}

	if len(result) > 1 {
		return nil, MultipleResultError
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
func (r *rateLimitStore) GetRateLimitsForCache(mtime time.Time, firstUpdate bool) ([]*model.RateLimit, []*model.RateLimitRevision, error) {
	handler := r.handler

	serviceIds := make(map[string]struct{})
	limitResults, err := handler.LoadValuesByFilter(tblRateLimitConfig, []string{RateConfFieldMtime, RateConfFieldServiceID}, &model.RateLimit{},
		func(m map[string]interface{}) bool {
			mt := m[RateConfFieldMtime].(time.Time)
			isAfter := mt.After(mtime)
			if isAfter {
				serviceIds[m[RateConfFieldServiceID].(string)] = struct{}{}
			}
			return isAfter
		})

	if err != nil {
		return nil, nil, err
	}

	if len(limitResults) == 0 {
		return []*model.RateLimit{}, []*model.RateLimitRevision{}, nil
	}

	svcIds := make([]string, len(serviceIds))
	pos := 0
	for k := range serviceIds {
		svcIds[pos] = k
		pos++
	}

	revisionResults, err := handler.LoadValues(tblRateLimitRevision, svcIds, &model.RateLimitRevision{})
	if err != nil {
		return nil, nil, err
	}

	if len(revisionResults) == 0 {
		return []*model.RateLimit{}, []*model.RateLimitRevision{}, nil
	}

	if len(limitResults) != len(revisionResults) {
		return nil, nil, errors.New("ratelimit conf size must be equal to ratelimit revision size")
	}

	limits := make([]*model.RateLimit, len(limitResults))
	versions := make([]*model.RateLimitRevision, len(revisionResults))

	pos = 0
	for i := range limitResults {
		limits[pos] = limitResults[i].(*model.RateLimit)
		pos++
	}

	pos = 0
	for i := range revisionResults {
		versions[pos] = revisionResults[i].(*model.RateLimitRevision)
		pos++
	}

	return limits, versions, nil
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
			return err
		}

		// create ratelimit_version
		lastVer := &model.RateLimitRevision{
			ServiceID:    limit.ServiceID,
			LastRevision: limit.Revision,
			ModifyTime:   time.Now(),
		}

		if err := saveValue(tx, tblRateLimitRevision, lastVer.ServiceID, lastVer); err != nil {
			log.Errorf("[Store][RateLimit] create ratelimit_revision(%s, %s) err: %s",
				limit.ID, limit.ServiceID, err.Error())
			return err
		}

		return nil
	})

}

// updateRateLimit
//  @receiver r
//  @param limit
//  @return error
func (r *rateLimitStore) updateRateLimit(limit *model.RateLimit) error {
	handler := r.handler
	return handler.Execute(true, func(tx *bolt.Tx) error {
		tNow := time.Now()

		limit.ModifyTime = tNow
		limit.Valid = true

		// create ratelimit_config
		if err := saveValue(tx, tblRateLimitConfig, limit.ID, limit); err != nil {
			log.Errorf("[Store][RateLimit] update rate_limit(%s, %s) err: %s",
				limit.ID, limit.ServiceID, err.Error())
			return err
		}

		// create ratelimit_version
		lastVer := &model.RateLimitRevision{
			ServiceID:    limit.ServiceID,
			LastRevision: limit.Revision,
			ModifyTime:   tNow,
		}

		if err := saveValue(tx, tblRateLimitRevision, lastVer.ServiceID, lastVer); err != nil {
			log.Errorf("[Store][RateLimit] update ratelimit_revision(%s, %s) err: %s",
				limit.ID, limit.ServiceID, err.Error())
			return err
		}

		return nil
	})

}

// deleteRateLimit
//  @receiver r
//  @param limit
//  @return error
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

		revisionProperties := make(map[string]interface{})
		revisionProperties[RateLimitReviFieldServiceID] = limit.ServiceID
		revisionProperties[RateLimitReviFieldLastRevision] = limit.Revision
		revisionProperties[RateLimitReviFieldModifyTime] = time.Now()

		if err := updateValue(tx, tblRateLimitRevision, limit.ServiceID, revisionProperties); err != nil {
			log.Errorf("[Store][RateLimit] delete ratelimit_version(%s, %s) err: %s",
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
		} else {
			return strings.Compare(routeConf[i].RateLimit.ID, routeConf[j].RateLimit.ID) < 0
		}
	})

	return routeConf[beginIndex:endIndex]
}
