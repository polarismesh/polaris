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
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/store"
)

const (
	labelCreateCircuitBreakerRule = "createCircuitBreakerRule"
	labelUpdateCircuitBreakerRule = "updateCircuitBreakerRule"
	labelDeleteCircuitBreakerRule = "deleteCircuitBreakerRule"
	labelEnableCircuitBreakerRule = "enableCircuitBreakerRule"
)

const (
	insertCircuitBreakerRuleSql = `insert into circuitbreaker_rule_v2(
			id, name, namespace, enable, revision, description, level, src_service, src_namespace, 
			dst_service, dst_namespace, dst_method, config, ctime, mtime, etime)
			values(?,?,?,?,?,?,?,?,?,?,?,?,?, sysdate(),sysdate(), %s)`
	updateCircuitBreakerRuleSql = `update circuitbreaker_rule_v2 set name = ?, namespace=?, enable = ?, revision= ?,
			description = ?, level = ?, src_service = ?, src_namespace = ?,
            dst_service = ?, dst_namespace = ?, dst_method = ?,
			config = ?, mtime = sysdate(), etime=%s where id = ?`
	deleteCircuitBreakerRuleSql = `update circuitbreaker_rule_v2 set flag = 1, mtime = sysdate() where id = ?`
	enableCircuitBreakerRuleSql = `update circuitbreaker_rule_v2 set enable = ?, revision = ?, mtime = sysdate(), 
			etime=%s where id = ?`
	countCircuitBreakerRuleSql     = `select count(*) from circuitbreaker_rule_v2 where flag = 0`
	queryCircuitBreakerRuleFullSql = `select id, name, namespace, enable, revision, description, level, src_service, 
			src_namespace, dst_service, dst_namespace, dst_method, config, unix_timestamp(ctime), unix_timestamp(mtime), 
			unix_timestamp(etime) from circuitbreaker_rule_v2 where flag = 0`
	queryCircuitBreakerRuleBriefSql = `select id, name, namespace, enable, revision, level, src_service, src_namespace, 
			dst_service, dst_namespace, dst_method, unix_timestamp(ctime), unix_timestamp(mtime), unix_timestamp(etime)
			from circuitbreaker_rule_v2 where flag = 0`
	queryCircuitBreakerRuleCacheSql = `select id, name, namespace, enable, revision, description, level, src_service, 
			src_namespace, dst_service, dst_namespace, dst_method, config, flag, unix_timestamp(ctime), 
			unix_timestamp(mtime), unix_timestamp(etime) from circuitbreaker_rule_v2 where mtime > FROM_UNIXTIME(?)`
)

const (
	labelCreateCircuitBreakerRuleOld    = "createCircuitBreakerRuleOld"
	labelTagCircuitBreakerRuleOld       = "tagCircuitBreakerRuleOld"
	labelDeleteTagCircuitBreakerRuleOld = "deleteTagCircuitBreakerRuleOld"
	labelReleaseCircuitBreakerRuleOld   = "releaseCircuitBreakerRuleOld"
	labelUnbindCircuitBreakerRuleOld    = "unbindCircuitBreakerRuleOld"
	labelUpdateCircuitBreakerRuleOld    = "updateCircuitBreakerRuleOld"
	labelDeleteCircuitBreakerRuleOld    = "deleteCircuitBreakerRuleOld"
)

// circuitBreakerStore 的实现
type circuitBreakerStore struct {
	master *BaseDB
	slave  *BaseDB
}

func (c *circuitBreakerStore) CreateCircuitBreakerRule(cbRule *model.CircuitBreakerRule) error {
	err := RetryTransaction(labelCreateCircuitBreakerRule, func() error {
		return c.createCircuitBreakerRule(cbRule)
	})

	return store.Error(err)
}

func (c *circuitBreakerStore) createCircuitBreakerRule(cbRule *model.CircuitBreakerRule) error {
	return c.master.processWithTransaction(labelCreateCircuitBreakerRule, func(tx *BaseTx) error {
		etimeStr := buildEtimeStr(cbRule.Enable)
		str := fmt.Sprintf(insertCircuitBreakerRuleSql, etimeStr)
		if _, err := tx.Exec(str, cbRule.ID, cbRule.Name, cbRule.Namespace, cbRule.Enable, cbRule.Revision,
			cbRule.Description, cbRule.Level, cbRule.SrcService, cbRule.SrcNamespace, cbRule.DstService,
			cbRule.DstNamespace, cbRule.DstMethod, cbRule.Rule); err != nil {
			log.Errorf("[Store][database] fail to %s exec sql, err: %s", labelCreateCircuitBreakerRule, err.Error())
			return err
		}
		if err := tx.Commit(); err != nil {
			log.Errorf("[Store][database] fail to %s commit tx, rule(%+v) commit tx err: %s",
				labelCreateCircuitBreakerRule, cbRule, err.Error())
			return err
		}
		return nil
	})
}

// UpdateCircuitBreakerRule 更新熔断规则
func (c *circuitBreakerStore) UpdateCircuitBreakerRule(cbRule *model.CircuitBreakerRule) error {
	err := RetryTransaction(labelUpdateCircuitBreakerRule, func() error {
		return c.updateCircuitBreakerRule(cbRule)
	})

	return store.Error(err)
}

func (c *circuitBreakerStore) updateCircuitBreakerRule(cbRule *model.CircuitBreakerRule) error {
	return c.master.processWithTransaction(labelUpdateCircuitBreakerRule, func(tx *BaseTx) error {
		etimeStr := buildEtimeStr(cbRule.Enable)
		str := fmt.Sprintf(updateCircuitBreakerRuleSql, etimeStr)
		if _, err := tx.Exec(str, cbRule.Name, cbRule.Namespace, cbRule.Enable,
			cbRule.Revision, cbRule.Description, cbRule.Level, cbRule.SrcService, cbRule.SrcNamespace,
			cbRule.DstService, cbRule.DstNamespace, cbRule.DstMethod, cbRule.Rule, cbRule.ID); err != nil {
			log.Errorf("[Store][database] fail to %s exec sql, err: %s", labelUpdateCircuitBreakerRule, err.Error())
			return err
		}

		if err := tx.Commit(); err != nil {
			log.Errorf("[Store][database] fail to %s commit tx, rule(%+v) commit tx err: %s",
				labelUpdateCircuitBreakerRule, cbRule, err.Error())
			return err
		}

		return nil
	})
}

// DeleteCircuitBreakerRule 删除熔断规则
func (c *circuitBreakerStore) DeleteCircuitBreakerRule(id string) error {
	err := RetryTransaction("deleteCircuitBreakerRule", func() error {
		return c.deleteCircuitBreakerRule(id)
	})

	return store.Error(err)
}

func (c *circuitBreakerStore) deleteCircuitBreakerRule(id string) error {
	return c.master.processWithTransaction(labelDeleteCircuitBreakerRule, func(tx *BaseTx) error {
		if _, err := tx.Exec(deleteCircuitBreakerRuleSql, id); err != nil {
			log.Errorf(
				"[Store][database] fail to %s exec sql, err: %s", labelDeleteCircuitBreakerRule, err.Error())
			return err
		}

		if err := tx.Commit(); err != nil {
			log.Errorf("[Store][database] fail to %s commit tx, rule(%s) commit tx err: %s",
				labelDeleteCircuitBreakerRule, id, err.Error())
			return err
		}
		return nil
	})
}

// HasCircuitBreakerRule check circuitbreaker rule exists
func (c *circuitBreakerStore) HasCircuitBreakerRule(id string) (bool, error) {
	queryParams := map[string]string{"id": id}
	count, err := c.getCircuitBreakerRulesCount(queryParams)
	if nil != err {
		return false, err
	}
	return count > 0, nil
}

// HasCircuitBreakerRuleByName check circuitbreaker rule exists by name
func (c *circuitBreakerStore) HasCircuitBreakerRuleByName(name string, namespace string) (bool, error) {
	queryParams := map[string]string{exactName: name, "namespace": namespace}
	count, err := c.getCircuitBreakerRulesCount(queryParams)
	if nil != err {
		return false, err
	}
	return count > 0, nil
}

// HasCircuitBreakerRuleByNameExcludeId check circuitbreaker rule exists by name exclude id
func (c *circuitBreakerStore) HasCircuitBreakerRuleByNameExcludeId(
	name string, namespace string, id string) (bool, error) {
	queryParams := map[string]string{exactName: name, "namespace": namespace, excludeId: id}
	count, err := c.getCircuitBreakerRulesCount(queryParams)
	if nil != err {
		return false, err
	}
	return count > 0, nil
}

func fetchCircuitBreakerRuleRows(rows *sql.Rows) ([]*model.CircuitBreakerRule, error) {
	defer rows.Close()
	var out []*model.CircuitBreakerRule
	for rows.Next() {
		var cbRule model.CircuitBreakerRule
		var flag int
		var ctime, mtime, etime int64
		err := rows.Scan(&cbRule.ID, &cbRule.Name, &cbRule.Namespace, &cbRule.Enable, &cbRule.Revision,
			&cbRule.Description, &cbRule.Level, &cbRule.SrcService, &cbRule.SrcNamespace, &cbRule.DstService,
			&cbRule.DstNamespace, &cbRule.DstMethod, &cbRule.Rule, &flag, &ctime, &mtime, &etime)
		if err != nil {
			log.Errorf("[Store][database] fetch circuitbreaker rule scan err: %s", err.Error())
			return nil, err
		}
		cbRule.CreateTime = time.Unix(ctime, 0)
		cbRule.ModifyTime = time.Unix(mtime, 0)
		cbRule.EnableTime = time.Unix(etime, 0)
		cbRule.Valid = true
		if flag == 1 {
			cbRule.Valid = false
		}
		out = append(out, &cbRule)
	}
	if err := rows.Err(); err != nil {
		log.Errorf("[Store][database] fetch circuitbreaker rule next err: %s", err.Error())
		return nil, err
	}
	return out, nil
}

func (c *circuitBreakerStore) GetCircuitBreakerRules(
	filter map[string]string, offset uint32, limit uint32) (uint32, []*model.CircuitBreakerRule, error) {
	var out []*model.CircuitBreakerRule
	var err error

	bValue, ok := filter[briefSearch]
	var isBrief = ok && strings.ToLower(bValue) == "true"
	delete(filter, briefSearch)

	if isBrief {
		out, err = c.getBriefCircuitBreakerRules(filter, offset, limit)
	} else {
		out, err = c.getFullCircuitBreakerRules(filter, offset, limit)
	}
	if err != nil {
		return 0, nil, err
	}
	num, err := c.getCircuitBreakerRulesCount(filter)
	if err != nil {
		return 0, nil, err
	}
	return num, out, nil
}

func (c *circuitBreakerStore) getBriefCircuitBreakerRules(
	filter map[string]string, offset uint32, limit uint32) ([]*model.CircuitBreakerRule, error) {
	queryStr, args := genCircuitBreakerRuleSQL(filter)
	args = append(args, offset, limit)
	str := queryCircuitBreakerRuleBriefSql + queryStr + ` order by mtime desc limit ?, ?`

	rows, err := c.master.Query(str, args...)
	if err != nil {
		log.Errorf("[Store][database] query brief circuitbreaker rules err: %s", err.Error())
		return nil, err
	}
	out, err := fetchBriefCircuitBreakerRules(rows)
	if err != nil {
		return nil, err
	}
	return out, nil
}

var blurQueryKeys = map[string]bool{
	"name":         true,
	"description":  true,
	"srcService":   true,
	"srcNamespace": true,
	"dstService":   true,
	"dstNamespace": true,
	"dstMethod":    true,
}

const (
	svcSpecificQueryKeyService   = "service"
	svcSpecificQueryKeyNamespace = "serviceNamespace"
	exactName                    = "exactName"
	excludeId                    = "excludeId"
)

func placeholders(n int) string {
	var b strings.Builder
	for i := 0; i < n-1; i++ {
		b.WriteString("?,")
	}
	if n > 0 {
		b.WriteString("?")
	}
	return b.String()
}

func genCircuitBreakerRuleSQL(query map[string]string) (string, []interface{}) {
	str := ""
	args := make([]interface{}, 0, len(query))
	var svcNamespaceQueryValue string
	var svcQueryValue string
	for key, value := range query {
		if len(value) == 0 {
			continue
		}
		if key == svcSpecificQueryKeyService {
			svcQueryValue = value
			continue
		}
		if key == svcSpecificQueryKeyNamespace {
			svcNamespaceQueryValue = value
			continue
		}
		storeKey := toUnderscoreName(key)
		if _, ok := blurQueryKeys[key]; ok {
			str += fmt.Sprintf(" and %s like ?", storeKey)
			args = append(args, "%"+value+"%")
		} else if key == "enable" {
			str += fmt.Sprintf(" and %s = ?", storeKey)
			arg, _ := strconv.ParseBool(value)
			args = append(args, arg)
		} else if key == "level" {
			tokens := strings.Split(value, ",")
			str += fmt.Sprintf(" and %s in (%s)", storeKey, placeholders(len(tokens)))
			for _, token := range tokens {
				args = append(args, token)
			}
		} else if key == exactName {
			str += " and name = ?"
			args = append(args, value)
		} else if key == excludeId {
			str += " and id != ?"
			args = append(args, value)
		} else {
			str += fmt.Sprintf(" and %s = ?", storeKey)
			args = append(args, value)
		}
	}
	if len(svcQueryValue) > 0 {
		str += " and (dst_service = ? or dst_service = '*' or src_service = ? or src_service = '*')"
		args = append(args, svcQueryValue, svcQueryValue)
	}
	if len(svcNamespaceQueryValue) > 0 {
		str += " and (dst_namespace = ? or dst_namespace = '*' or dst_namespace = ? or dst_namespace = '*')"
		args = append(args, svcNamespaceQueryValue, svcNamespaceQueryValue)
	}
	return str, args
}

// fetchBriefRateLimitRows fetch the brief ratelimit list
func fetchBriefCircuitBreakerRules(rows *sql.Rows) ([]*model.CircuitBreakerRule, error) {
	defer rows.Close()
	var out []*model.CircuitBreakerRule
	for rows.Next() {
		var cbRule model.CircuitBreakerRule
		var ctime, mtime, etime int64
		err := rows.Scan(&cbRule.ID, &cbRule.Name, &cbRule.Namespace, &cbRule.Enable, &cbRule.Revision,
			&cbRule.Level, &cbRule.SrcService, &cbRule.SrcNamespace, &cbRule.DstService, &cbRule.DstNamespace,
			&cbRule.DstMethod, &ctime, &mtime, &etime)
		if err != nil {
			log.Errorf("[Store][database] fetch brief circuitbreaker rule scan err: %s", err.Error())
			return nil, err
		}
		cbRule.CreateTime = time.Unix(ctime, 0)
		cbRule.ModifyTime = time.Unix(mtime, 0)
		cbRule.EnableTime = time.Unix(etime, 0)
		out = append(out, &cbRule)
	}
	if err := rows.Err(); err != nil {
		log.Errorf("[Store][database] fetch brief circuitbreaker rule next err: %s", err.Error())
		return nil, err
	}
	return out, nil
}

func (c *circuitBreakerStore) getFullCircuitBreakerRules(
	filter map[string]string, offset uint32, limit uint32) ([]*model.CircuitBreakerRule, error) {
	queryStr, args := genCircuitBreakerRuleSQL(filter)
	args = append(args, offset, limit)
	str := queryCircuitBreakerRuleFullSql + queryStr + ` order by mtime desc limit ?, ?`

	rows, err := c.master.Query(str, args...)
	if err != nil {
		log.Errorf("[Store][database] query brief circuitbreaker rules err: %s", err.Error())
		return nil, err
	}
	out, err := fetchFullCircuitBreakerRules(rows)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func fetchFullCircuitBreakerRules(rows *sql.Rows) ([]*model.CircuitBreakerRule, error) {
	defer rows.Close()
	var out []*model.CircuitBreakerRule
	for rows.Next() {
		var cbRule model.CircuitBreakerRule
		var ctime, mtime, etime int64
		err := rows.Scan(&cbRule.ID, &cbRule.Name, &cbRule.Namespace, &cbRule.Enable, &cbRule.Revision,
			&cbRule.Description, &cbRule.Level, &cbRule.SrcService, &cbRule.SrcNamespace, &cbRule.DstService,
			&cbRule.DstNamespace, &cbRule.DstMethod, &cbRule.Rule, &ctime, &mtime, &etime)
		if err != nil {
			log.Errorf("[Store][database] fetch full circuitbreaker rule scan err: %s", err.Error())
			return nil, err
		}
		cbRule.CreateTime = time.Unix(ctime, 0)
		cbRule.ModifyTime = time.Unix(mtime, 0)
		cbRule.EnableTime = time.Unix(etime, 0)
		out = append(out, &cbRule)
	}
	if err := rows.Err(); err != nil {
		log.Errorf("[Store][database] fetch full circuitbreaker rule next err: %s", err.Error())
		return nil, err
	}
	return out, nil
}

func (c *circuitBreakerStore) getCircuitBreakerRulesCount(filter map[string]string) (uint32, error) {
	queryStr, args := genCircuitBreakerRuleSQL(filter)
	str := countCircuitBreakerRuleSql + queryStr
	var total uint32
	err := c.master.QueryRow(str, args...).Scan(&total)
	switch {
	case err == sql.ErrNoRows:
		return 0, nil
	case err != nil:
		log.Errorf("[Store][database] get circuitbreaker rule count err: %s", err.Error())
		return 0, err
	default:
	}
	return total, nil
}

// GetCircuitBreakerRulesForCache list circuitbreaker rules by query
func (c *circuitBreakerStore) GetCircuitBreakerRulesForCache(
	mtime time.Time, firstUpdate bool) ([]*model.CircuitBreakerRule, error) {
	str := queryCircuitBreakerRuleCacheSql
	if firstUpdate {
		str += " and flag != 1"
	}
	rows, err := c.slave.Query(str, timeToTimestamp(mtime))
	if err != nil {
		log.Errorf("[Store][database] query circuitbreaker rules with mtime err: %s", err.Error())
		return nil, err
	}
	cbRules, err := fetchCircuitBreakerRuleRows(rows)
	if err != nil {
		return nil, err
	}
	return cbRules, nil
}

// EnableCircuitBreakerRule enable circuitbreaker rule
func (c *circuitBreakerStore) EnableCircuitBreakerRule(cbRule *model.CircuitBreakerRule) error {
	err := RetryTransaction("enableCircuitbreaker", func() error {
		return c.enableCircuitBreakerRule(cbRule)
	})

	return store.Error(err)
}

func (c *circuitBreakerStore) enableCircuitBreakerRule(cbRule *model.CircuitBreakerRule) error {
	return c.master.processWithTransaction(labelEnableCircuitBreakerRule, func(tx *BaseTx) error {

		etimeStr := buildEtimeStr(cbRule.Enable)
		str := fmt.Sprintf(enableCircuitBreakerRuleSql, etimeStr)
		if _, err := tx.Exec(str, cbRule.Enable, cbRule.Revision, cbRule.ID); err != nil {
			log.Errorf(
				"[Store][database] fail to %s exec sql, err: %s", labelEnableCircuitBreakerRule, err.Error())
			return err
		}

		if err := tx.Commit(); err != nil {
			log.Errorf("[Store][database] fail to %s commit tx, rule(%+v) commit tx err: %s",
				labelEnableCircuitBreakerRule, cbRule, err.Error())
			return err
		}
		return nil
	})
}
