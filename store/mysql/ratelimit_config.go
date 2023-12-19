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

package sqldb

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/store"
)

var _ store.RateLimitStore = (*rateLimitStore)(nil)

// rateLimitStore RateLimitStore的实现
type rateLimitStore struct {
	master *BaseDB
	slave  *BaseDB
}

// CreateRateLimit 新建限流规则
func (rls *rateLimitStore) CreateRateLimit(limit *model.RateLimit) error {
	if limit.ID == "" || limit.Revision == "" {
		return errors.New("[Store][database] create rate limit missing some params")
	}
	err := RetryTransaction("createRateLimit", func() error {
		return rls.createRateLimit(limit)
	})

	return store.Error(err)
}

func limitToEtimeStr(limit *model.RateLimit) string {
	etimeStr := "sysdate()"
	if limit.Disable {
		etimeStr = emptyEnableTime
	}
	return etimeStr
}

// createRateLimit
func (rls *rateLimitStore) createRateLimit(limit *model.RateLimit) error {
	tx, err := rls.master.Begin()
	if err != nil {
		log.Errorf("[Store][database] create rate limit(%+v) begin tx err: %s", limit, err.Error())
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	etimeStr := limitToEtimeStr(limit)
	// 新建限流规则
	str := fmt.Sprintf(`insert into ratelimit_config(
			id, name, disable, service_id, method, labels, priority, rule, revision, ctime, mtime, etime)
			values(?,?,?,?,?,?,?,?,?,sysdate(),sysdate(), %s)`, etimeStr)
	if _, err := tx.Exec(str, limit.ID, limit.Name, limit.Disable, limit.ServiceID, limit.Method, limit.Labels,
		limit.Priority, limit.Rule, limit.Revision); err != nil {
		log.Errorf("[Store][database] create rate limit(%+v), sql %s err: %s", limit, str, err.Error())
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Errorf("[Store][database] create rate limit(%+v) commit tx err: %s", limit, err.Error())
		return err
	}

	return nil
}

// UpdateRateLimit 更新限流规则
func (rls *rateLimitStore) UpdateRateLimit(limit *model.RateLimit) error {
	if limit.ID == "" || limit.Revision == "" {
		return errors.New("[Store][database] update rate limit missing some params")
	}

	err := RetryTransaction("updateRateLimit", func() error {
		return rls.updateRateLimit(limit)
	})

	return store.Error(err)
}

// EnableRateLimit 启用限流规则
func (rls *rateLimitStore) EnableRateLimit(limit *model.RateLimit) error {
	if limit.ID == "" || limit.Revision == "" {
		return errors.New("[Store][database] enable rate limit missing some params")
	}

	err := RetryTransaction("enableRateLimit", func() error {
		return rls.enableRateLimit(limit)
	})

	return store.Error(err)
}

// enableRateLimit
func (rls *rateLimitStore) enableRateLimit(limit *model.RateLimit) error {
	tx, err := rls.master.Begin()
	if err != nil {
		log.Errorf("[Store][database] update rate limit(%+v) begin tx err: %s", limit, err.Error())
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	etimeStr := limitToEtimeStr(limit)
	str := fmt.Sprintf(
		`update ratelimit_config set disable = ?, revision = ?, mtime = sysdate(), etime=%s where id = ?`, etimeStr)
	if _, err := tx.Exec(str, limit.Disable, limit.Revision, limit.ID); err != nil {
		log.Errorf("[Store][database] update rate limit(%+v), sql %s, err: %s", limit, str, err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Errorf("[Store][database] update rate limit(%+v) commit tx err: %s", limit, err.Error())
		return err
	}
	return nil
}

// updateRateLimit
func (rls *rateLimitStore) updateRateLimit(limit *model.RateLimit) error {
	tx, err := rls.master.Begin()
	if err != nil {
		log.Errorf("[Store][database] update rate limit(%+v) begin tx err: %s", limit, err.Error())
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	etimeStr := limitToEtimeStr(limit)
	str := fmt.Sprintf(`update ratelimit_config set name = ?, service_id=?, disable = ?, method= ?,
			labels = ?, priority = ?, rule = ?, revision = ?, mtime = sysdate(), etime=%s where id = ?`, etimeStr)
	if _, err := tx.Exec(str, limit.Name, limit.ServiceID, limit.Disable,
		limit.Method, limit.Labels, limit.Priority, limit.Rule, limit.Revision, limit.ID); err != nil {
		log.Errorf("[Store][database] update rate limit(%+v), sql %s, err: %s", limit, str, err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Errorf("[Store][database] update rate limit(%+v) commit tx err: %s", limit, err.Error())
		return err
	}
	return nil
}

// DeleteRateLimit 删除限流规则
func (rls *rateLimitStore) DeleteRateLimit(limit *model.RateLimit) error {
	if limit.ID == "" || limit.Revision == "" {
		return errors.New("[Store][database] delete rate limit missing some params")
	}

	err := RetryTransaction("deleteRateLimit", func() error {
		return rls.deleteRateLimit(limit)
	})

	return store.Error(err)
}

// deleteRateLimit
func (rls *rateLimitStore) deleteRateLimit(limit *model.RateLimit) error {
	tx, err := rls.master.Begin()
	if err != nil {
		log.Errorf("[Store][database] delete rate limit(%+v) begin tx err: %s", limit, err.Error())
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	str := `update ratelimit_config set flag = 1, mtime = sysdate() where id = ?`
	if _, err := tx.Exec(str, limit.ID); err != nil {
		log.Errorf("[Store][database] delete rate limit(%+v) err: %s", limit, err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Errorf("[Store][database] delete rate limit(%+v) commit tx err: %s", limit, err.Error())
		return err
	}
	return nil
}

// GetRateLimitWithID 根据限流规则ID获取限流规则
func (rls *rateLimitStore) GetRateLimitWithID(id string) (*model.RateLimit, error) {
	if id == "" {
		log.Errorf("[Store][database] get rate limit missing some params")
		return nil, errors.New("get rate limit missing some params")
	}

	str := `select id, name, disable, service_id, method, labels, priority, rule, revision, flag,
			unix_timestamp(ctime), unix_timestamp(mtime), unix_timestamp(etime)
			from ratelimit_config where id = ? and flag = 0`
	rows, err := rls.master.Query(str, id)
	if err != nil {
		log.Errorf("[Store][database] query rate limit with id(%s) err: %s", id, err.Error())
		return nil, err
	}
	out, err := fetchRateLimitRows(rows)
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out[0], nil
}

// fetchRateLimitRows 读取限流数据
func fetchRateLimitRows(rows *sql.Rows) ([]*model.RateLimit, error) {
	defer rows.Close()
	var out []*model.RateLimit
	for rows.Next() {
		var rateLimit model.RateLimit
		var flag int
		var ctime, mtime, etime int64
		err := rows.Scan(&rateLimit.ID, &rateLimit.Name, &rateLimit.Disable, &rateLimit.ServiceID, &rateLimit.Method,
			&rateLimit.Labels, &rateLimit.Priority, &rateLimit.Rule, &rateLimit.Revision, &flag, &ctime, &mtime, &etime)
		if err != nil {
			log.Errorf("[Store][database] fetch rate limit scan err: %s", err.Error())
			return nil, err
		}
		rateLimit.CreateTime = time.Unix(ctime, 0)
		rateLimit.ModifyTime = time.Unix(mtime, 0)
		rateLimit.EnableTime = time.Unix(etime, 0)
		rateLimit.Valid = true
		if flag == 1 {
			rateLimit.Valid = false
		}
		out = append(out, &rateLimit)
	}
	if err := rows.Err(); err != nil {
		log.Errorf("[Store][database] fetch rate limit next err: %s", err.Error())
		return nil, err
	}
	return out, nil
}

// GetRateLimitsForCache 根据修改时间拉取增量限流规则及最新版本号
func (rls *rateLimitStore) GetRateLimitsForCache(mtime time.Time,
	firstUpdate bool) ([]*model.RateLimit, error) {
	str := `select id, name, disable, ratelimit_config.service_id, method, labels, priority, rule, revision, flag,
			unix_timestamp(ratelimit_config.ctime), unix_timestamp(ratelimit_config.mtime), 
			unix_timestamp(ratelimit_config.etime) from ratelimit_config 
			where ratelimit_config.mtime > FROM_UNIXTIME(?)`
	if firstUpdate {
		str += " and flag != 1"
	}
	rows, err := rls.slave.Query(str, timeToTimestamp(mtime))
	if err != nil {
		log.Errorf("[Store][database] query rate limits with mtime err: %s", err.Error())
		return nil, err
	}
	rateLimits, err := fetchRateLimitCacheRows(rows)
	if err != nil {
		return nil, err
	}
	return rateLimits, nil
}

// fetchRateLimitCacheRows 读取限流数据以及最新版本号
func fetchRateLimitCacheRows(rows *sql.Rows) ([]*model.RateLimit, error) {
	defer rows.Close()

	var rateLimits []*model.RateLimit

	for rows.Next() {
		var (
			rateLimit           model.RateLimit
			ctime, mtime, etime int64
			serviceID           string
			flag                int
		)
		err := rows.Scan(&rateLimit.ID, &rateLimit.Name, &rateLimit.Disable, &serviceID, &rateLimit.Method, &rateLimit.Labels,
			&rateLimit.Priority, &rateLimit.Rule, &rateLimit.Revision, &flag, &ctime, &mtime, &etime)
		if err != nil {
			log.Errorf("[Store][database] fetch rate limit cache scan err: %s", err.Error())
			return nil, err
		}
		rateLimit.CreateTime = time.Unix(ctime, 0)
		rateLimit.ModifyTime = time.Unix(mtime, 0)
		rateLimit.EnableTime = time.Unix(etime, 0)
		rateLimit.Valid = true
		if flag == 1 {
			rateLimit.Valid = false
		}
		rateLimit.ServiceID = serviceID

		rateLimits = append(rateLimits, &rateLimit)
	}

	if err := rows.Err(); err != nil {
		log.Errorf("[Store][database] fetch rate limit cache next err: %s", err.Error())
		return nil, err
	}
	return rateLimits, nil
}

const (
	briefSearch = "brief"
)

// GetExtendRateLimits 根据过滤条件获取限流规则及数目
func (rls *rateLimitStore) GetExtendRateLimits(
	filter map[string]string, offset uint32, limit uint32) (uint32, []*model.ExtendRateLimit, error) {
	var out []*model.ExtendRateLimit
	var err error
	if bValue, ok := filter[briefSearch]; ok && strings.ToLower(bValue) == "true" {
		out, err = rls.getBriefRateLimits(filter, offset, limit)
	} else {
		out, err = rls.getExpandRateLimits(filter, offset, limit)
	}
	if err != nil {
		return 0, nil, err
	}
	num, err := rls.getExpandRateLimitsCount(filter)
	if err != nil {
		return 0, nil, err
	}
	return num, out, nil
}

// getBriefRateLimits 获取列表的概要信息
func (rls *rateLimitStore) getBriefRateLimits(
	filter map[string]string, offset uint32, limit uint32) ([]*model.ExtendRateLimit, error) {
	str := `select ratelimit_config.id, ratelimit_config.name, ratelimit_config.disable,
            ratelimit_config.service_id, ratelimit_config.method, unix_timestamp(ratelimit_config.ctime), 
			unix_timestamp(ratelimit_config.mtime), unix_timestamp(ratelimit_config.etime) 
			from ratelimit_config where ratelimit_config.flag = 0`

	queryStr, args := genFilterRateLimitSQL(filter)
	args = append(args, offset, limit)
	str = str + queryStr + ` order by ratelimit_config.mtime desc limit ?, ?`

	rows, err := rls.master.Query(str, args...)
	if err != nil {
		log.Errorf("[Store][database] query rate limits err: %s", err.Error())
		return nil, err
	}
	out, err := fetchBriefRateLimitRows(rows)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// fetchBriefRateLimitRows fetch the brief ratelimit list
func fetchBriefRateLimitRows(rows *sql.Rows) ([]*model.ExtendRateLimit, error) {
	defer rows.Close()
	var out []*model.ExtendRateLimit
	for rows.Next() {
		var expand model.ExtendRateLimit
		expand.RateLimit = &model.RateLimit{}
		var ctime, mtime, etime int64
		err := rows.Scan(
			&expand.RateLimit.ID,
			&expand.RateLimit.Name,
			&expand.RateLimit.Disable,
			&expand.RateLimit.ServiceID,
			&expand.RateLimit.Method, &ctime, &mtime, &etime)
		if err != nil {
			log.Errorf("[Store][database] fetch brief rate limit scan err: %s", err.Error())
			return nil, err
		}
		expand.RateLimit.CreateTime = time.Unix(ctime, 0)
		expand.RateLimit.ModifyTime = time.Unix(mtime, 0)
		expand.RateLimit.EnableTime = time.Unix(etime, 0)
		out = append(out, &expand)
	}
	if err := rows.Err(); err != nil {
		log.Errorf("[Store][database] fetch brief rate limit next err: %s", err.Error())
		return nil, err
	}
	return out, nil
}

// getExpandRateLimits 根据过滤条件获取限流规则
func (rls *rateLimitStore) getExpandRateLimits(
	filter map[string]string, offset uint32, limit uint32) ([]*model.ExtendRateLimit, error) {
	str := `select ratelimit_config.id, ratelimit_config.name, ratelimit_config.disable,
            ratelimit_config.service_id, ratelimit_config.method, ratelimit_config.labels, 
            ratelimit_config.priority, ratelimit_config.rule, ratelimit_config.revision, 
            unix_timestamp(ratelimit_config.ctime), unix_timestamp(ratelimit_config.mtime), 
			unix_timestamp(ratelimit_config.etime) 
			from ratelimit_config 
			where ratelimit_config.flag = 0`

	queryStr, args := genFilterRateLimitSQL(filter)
	args = append(args, offset, limit)
	str = str + queryStr + ` order by ratelimit_config.mtime desc limit ?, ?`

	rows, err := rls.master.Query(str, args...)
	if err != nil {
		log.Errorf("[Store][database] query rate limits err: %s", err.Error())
		return nil, err
	}
	out, err := fetchExpandRateLimitRows(rows)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// fetchExpandRateLimitRows 读取包含服务信息的限流数据
func fetchExpandRateLimitRows(rows *sql.Rows) ([]*model.ExtendRateLimit, error) {
	defer rows.Close()
	var out []*model.ExtendRateLimit
	for rows.Next() {
		var expand model.ExtendRateLimit
		expand.RateLimit = &model.RateLimit{}
		var ctime, mtime, etime int64
		err := rows.Scan(
			&expand.RateLimit.ID,
			&expand.RateLimit.Name,
			&expand.RateLimit.Disable,
			&expand.RateLimit.ServiceID,
			&expand.RateLimit.Method,
			&expand.RateLimit.Labels,
			&expand.RateLimit.Priority,
			&expand.RateLimit.Rule,
			&expand.RateLimit.Revision, &ctime, &mtime, &etime)
		if err != nil {
			log.Errorf("[Store][database] fetch expand rate limit scan err: %s", err.Error())
			return nil, err
		}
		expand.RateLimit.CreateTime = time.Unix(ctime, 0)
		expand.RateLimit.ModifyTime = time.Unix(mtime, 0)
		expand.RateLimit.EnableTime = time.Unix(etime, 0)
		out = append(out, &expand)
	}
	if err := rows.Err(); err != nil {
		log.Errorf("[Store][database] fetch expand rate limit next err: %s", err.Error())
		return nil, err
	}
	return out, nil
}

// getExpandRateLimitsCount 根据过滤条件获取限流规则数目
func (rls *rateLimitStore) getExpandRateLimitsCount(filter map[string]string) (uint32, error) {
	str := `select count(*) from ratelimit_config where ratelimit_config.flag = 0`

	queryStr, args := genFilterRateLimitSQL(filter)
	str = str + queryStr
	var total uint32
	err := rls.master.QueryRow(str, args...).Scan(&total)
	switch {
	case err == sql.ErrNoRows:
		return 0, nil
	case err != nil:
		log.Errorf("[Store][database] get expand rate limits count err: %s", err.Error())
		return 0, err
	default:
	}
	return total, nil
}

var queryKeyToDbColumn = map[string]string{
	"id":      "ratelimit_config.id",
	"name":    "ratelimit_config.name",
	"method":  "ratelimit_config.method",
	"labels":  "ratelimit_config.labels",
	"disable": "ratelimit_config.disable",
}

// genFilterRateLimitSQL 生成查询语句的过滤语句
func genFilterRateLimitSQL(query map[string]string) (string, []interface{}) {
	str := ""
	args := make([]interface{}, 0, len(query))
	for key, value := range query {
		var arg interface{}
		sqlKey := queryKeyToDbColumn[key]
		if len(sqlKey) == 0 {
			continue
		}
		if key == "name" || key == "method" || key == "labels" {
			str += fmt.Sprintf(" and %s like ?", sqlKey)
			arg = "%" + value + "%"
		} else if key == "disable" {
			str += fmt.Sprintf(" and %s = ?", sqlKey)
			arg, _ = strconv.ParseBool(value)
		} else {
			str += fmt.Sprintf(" and %s = ?", sqlKey)
			arg = value
		}
		args = append(args, arg)
	}
	return str, args
}
