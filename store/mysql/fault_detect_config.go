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
	"strings"
	"time"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/store"
)

var _ store.FaultDetectRuleStore = (*faultDetectRuleStore)(nil)

type faultDetectRuleStore struct {
	master *BaseDB
	slave  *BaseDB
}

const (
	labelCreateFaultDetectRule = "createFaultDetectRule"
	labelUpdateFaultDetectRule = "updateFaultDetectRule"
	labelDeleteFaultDetectRule = "deleteFaultDetectRule"
)

const (
	insertFaultDetectSql = `insert into fault_detect_rule(
			id, name, namespace, revision, description, dst_service, dst_namespace, dst_method, config, ctime, mtime)
			values(?,?,?,?,?,?,?,?,?, sysdate(),sysdate())`
	updateFaultDetectSql = `update fault_detect_rule set name = ?, namespace = ?, revision = ?, description = ?,
			dst_service = ?, dst_namespace = ?, dst_method = ?, config = ?, mtime = sysdate() where id = ?`
	deleteFaultDetectSql    = `update fault_detect_rule set flag = 1, mtime = sysdate() where id = ?`
	countFaultDetectSql     = `select count(*) from fault_detect_rule where flag = 0`
	queryFaultDetectFullSql = `select id, name, namespace, revision, description, dst_service, 
			dst_namespace, dst_method, config, unix_timestamp(ctime), unix_timestamp(mtime)
            from fault_detect_rule where flag = 0`
	queryFaultDetectBriefSql = `select id, name, namespace, revision, description, dst_service, 
			dst_namespace, dst_method, unix_timestamp(ctime), unix_timestamp(mtime)
            from fault_detect_rule where flag = 0`
	queryFaultDetectCacheSql = `select id, name, namespace, revision, description, dst_service, 
			dst_namespace, dst_method, config, flag, unix_timestamp(ctime), unix_timestamp(mtime)
			from fault_detect_rule where mtime > FROM_UNIXTIME(?)`
)

// CreateFaultDetectRule create fault detect rule
func (f *faultDetectRuleStore) CreateFaultDetectRule(fdRule *model.FaultDetectRule) error {
	err := RetryTransaction(labelCreateFaultDetectRule, func() error {
		return f.createFaultDetectRule(fdRule)
	})
	return store.Error(err)
}

func (f *faultDetectRuleStore) createFaultDetectRule(fdRule *model.FaultDetectRule) error {
	return f.master.processWithTransaction(labelCreateFaultDetectRule, func(tx *BaseTx) error {
		if _, err := tx.Exec(insertFaultDetectSql, fdRule.ID, fdRule.Name, fdRule.Namespace, fdRule.Revision,
			fdRule.Description, fdRule.DstService, fdRule.DstNamespace, fdRule.DstMethod, fdRule.Rule); err != nil {
			log.Errorf("[Store][database] fail to %s exec sql, rule(%+v), err: %s",
				labelCreateFaultDetectRule, fdRule, err.Error())
			return err
		}

		if err := tx.Commit(); err != nil {
			log.Errorf("[Store][database] fail to %s commit tx, rule(%+v), err: %s",
				labelCreateFaultDetectRule, fdRule, err.Error())
			return err
		}
		return nil
	})
}

// UpdateFaultDetectRule update fault detect rule
func (f *faultDetectRuleStore) UpdateFaultDetectRule(fdRule *model.FaultDetectRule) error {
	err := RetryTransaction(labelUpdateFaultDetectRule, func() error {
		return f.updateFaultDetectRule(fdRule)
	})
	return store.Error(err)
}

func (f *faultDetectRuleStore) updateFaultDetectRule(fdRule *model.FaultDetectRule) error {
	return f.master.processWithTransaction(labelUpdateFaultDetectRule, func(tx *BaseTx) error {
		if _, err := tx.Exec(updateFaultDetectSql, fdRule.Name, fdRule.Namespace, fdRule.Revision,
			fdRule.Description, fdRule.DstService, fdRule.DstNamespace, fdRule.DstMethod, fdRule.Rule, fdRule.ID); err != nil {
			log.Errorf("[Store][database] fail to %s exec sql, rule(%+v), err: %s",
				labelUpdateFaultDetectRule, fdRule, err.Error())
			return err
		}

		if err := tx.Commit(); err != nil {
			log.Errorf("[Store][database] fail to %s commit tx, rule(%+v), err: %s",
				labelUpdateFaultDetectRule, fdRule, err.Error())
			return err
		}
		return nil
	})
}

// DeleteFaultDetectRule delete fault detect rule
func (f *faultDetectRuleStore) DeleteFaultDetectRule(id string) error {
	err := RetryTransaction(labelDeleteFaultDetectRule, func() error {
		return f.deleteFaultDetectRule(id)
	})
	return store.Error(err)
}

func (f *faultDetectRuleStore) deleteFaultDetectRule(id string) error {
	return f.master.processWithTransaction(labelDeleteFaultDetectRule, func(tx *BaseTx) error {
		if _, err := tx.Exec(deleteFaultDetectSql, id); err != nil {
			log.Errorf("[Store][database] fail to %s exec sql, rule(%s), err: %s",
				labelDeleteFaultDetectRule, id, err.Error())
			return err
		}

		if err := tx.Commit(); err != nil {
			log.Errorf("[Store][database] fail to %s commit tx, rule(%s), err: %s",
				labelDeleteFaultDetectRule, id, err.Error())
			return err
		}
		return nil
	})
}

// HasFaultDetectRule check fault detect rule exists
func (f *faultDetectRuleStore) HasFaultDetectRule(id string) (bool, error) {
	queryParams := map[string]string{"id": id}
	count, err := f.getFaultDetectRulesCount(queryParams)
	if nil != err {
		return false, err
	}
	return count > 0, nil
}

// HasFaultDetectRuleByName check fault detect rule exists by name
func (f *faultDetectRuleStore) HasFaultDetectRuleByName(name string, namespace string) (bool, error) {
	queryParams := map[string]string{exactName: name, "namespace": namespace}
	count, err := f.getFaultDetectRulesCount(queryParams)
	if nil != err {
		return false, err
	}
	return count > 0, nil
}

// HasFaultDetectRuleByNameExcludeId check fault detect rule exists by name not this id
func (f *faultDetectRuleStore) HasFaultDetectRuleByNameExcludeId(
	name string, namespace string, id string) (bool, error) {
	queryParams := map[string]string{exactName: name, "namespace": namespace, excludeId: id}
	count, err := f.getFaultDetectRulesCount(queryParams)
	if nil != err {
		return false, err
	}
	return count > 0, nil
}

// GetFaultDetectRules get all fault detect rules by query and limit
func (f *faultDetectRuleStore) GetFaultDetectRules(
	filter map[string]string, offset uint32, limit uint32) (uint32, []*model.FaultDetectRule, error) {
	var out []*model.FaultDetectRule
	var err error

	bValue, ok := filter[briefSearch]
	var isBrief = ok && strings.ToLower(bValue) == "true"
	delete(filter, briefSearch)

	if isBrief {
		out, err = f.getBriefFaultDetectRules(filter, offset, limit)
	} else {
		out, err = f.getFullFaultDetectRules(filter, offset, limit)
	}
	if err != nil {
		return 0, nil, err
	}
	num, err := f.getFaultDetectRulesCount(filter)
	if err != nil {
		return 0, nil, err
	}
	return num, out, nil
}

// GetFaultDetectRulesForCache get increment circuitbreaker rules
func (f *faultDetectRuleStore) GetFaultDetectRulesForCache(
	mtime time.Time, firstUpdate bool) ([]*model.FaultDetectRule, error) {
	str := queryFaultDetectCacheSql
	if firstUpdate {
		str += " and flag != 1"
	}
	rows, err := f.slave.Query(str, timeToTimestamp(mtime))
	if err != nil {
		log.Errorf("[Store][database] query fault detect rules with mtime err: %s", err.Error())
		return nil, err
	}
	fdRules, err := fetchFaultDetectRulesRows(rows)
	if err != nil {
		return nil, err
	}
	return fdRules, nil
}

func fetchFaultDetectRulesRows(rows *sql.Rows) ([]*model.FaultDetectRule, error) {
	defer rows.Close()
	var out []*model.FaultDetectRule
	for rows.Next() {
		var fdRule model.FaultDetectRule
		var flag int
		var ctime, mtime int64
		err := rows.Scan(&fdRule.ID, &fdRule.Name, &fdRule.Namespace, &fdRule.Revision,
			&fdRule.Description, &fdRule.DstService, &fdRule.DstNamespace,
			&fdRule.DstMethod, &fdRule.Rule, &flag, &ctime, &mtime)
		if err != nil {
			log.Errorf("[Store][database] fetch brief fault detect rule scan err: %s", err.Error())
			return nil, err
		}
		fdRule.CreateTime = time.Unix(ctime, 0)
		fdRule.ModifyTime = time.Unix(mtime, 0)
		fdRule.Valid = true
		if flag == 1 {
			fdRule.Valid = false
		}
		out = append(out, &fdRule)
	}
	if err := rows.Err(); err != nil {
		log.Errorf("[Store][database] fetch brief fault detect rule next err: %s", err.Error())
		return nil, err
	}
	return out, nil
}

func genFaultDetectRuleSQL(query map[string]string) (string, []interface{}) {
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
		str += " and (dst_service = ? or dst_service = '*')"
		args = append(args, svcQueryValue)
	}
	if len(svcNamespaceQueryValue) > 0 {
		str += " and (dst_namespace = ? or dst_namespace = '*')"
		args = append(args, svcNamespaceQueryValue)
	}
	return str, args
}

func (f *faultDetectRuleStore) getFaultDetectRulesCount(filter map[string]string) (uint32, error) {
	queryStr, args := genFaultDetectRuleSQL(filter)
	str := countFaultDetectSql + queryStr
	var total uint32
	err := f.master.QueryRow(str, args...).Scan(&total)
	switch {
	case err == sql.ErrNoRows:
		return 0, nil
	case err != nil:
		log.Errorf("[Store][database] get fault detect rule count err: %s", err.Error())
		return 0, err
	default:
	}
	return total, nil
}

func (f *faultDetectRuleStore) getBriefFaultDetectRules(
	filter map[string]string, offset uint32, limit uint32) ([]*model.FaultDetectRule, error) {
	queryStr, args := genFaultDetectRuleSQL(filter)
	args = append(args, offset, limit)
	str := queryFaultDetectBriefSql + queryStr + ` order by mtime desc limit ?, ?`

	rows, err := f.master.Query(str, args...)
	if err != nil {
		log.Errorf("[Store][database] query brief fault detect rule rules err: %s", err.Error())
		return nil, err
	}
	out, err := fetchBriefFaultDetectRules(rows)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func fetchBriefFaultDetectRules(rows *sql.Rows) ([]*model.FaultDetectRule, error) {
	defer rows.Close()
	var out []*model.FaultDetectRule
	for rows.Next() {
		var fdRule model.FaultDetectRule
		var ctime, mtime int64
		err := rows.Scan(&fdRule.ID, &fdRule.Name, &fdRule.Namespace, &fdRule.Revision,
			&fdRule.Description, &fdRule.DstService, &fdRule.DstNamespace,
			&fdRule.DstMethod, &ctime, &mtime)
		if err != nil {
			log.Errorf("[Store][database] fetch brief fault detect rule scan err: %s", err.Error())
			return nil, err
		}
		fdRule.CreateTime = time.Unix(ctime, 0)
		fdRule.ModifyTime = time.Unix(mtime, 0)
		out = append(out, &fdRule)
	}
	if err := rows.Err(); err != nil {
		log.Errorf("[Store][database] fetch brief fault detect rule next err: %s", err.Error())
		return nil, err
	}
	return out, nil
}

func (f *faultDetectRuleStore) getFullFaultDetectRules(
	filter map[string]string, offset uint32, limit uint32) ([]*model.FaultDetectRule, error) {
	queryStr, args := genFaultDetectRuleSQL(filter)
	args = append(args, offset, limit)
	str := queryFaultDetectFullSql + queryStr + ` order by mtime desc limit ?, ?`

	rows, err := f.master.Query(str, args...)
	if err != nil {
		log.Errorf("[Store][database] query brief fault detect rules err: %s", err.Error())
		return nil, err
	}
	out, err := fetchFullFaultDetectRules(rows)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func fetchFullFaultDetectRules(rows *sql.Rows) ([]*model.FaultDetectRule, error) {
	defer rows.Close()
	var out []*model.FaultDetectRule
	for rows.Next() {
		var fdRule model.FaultDetectRule
		var ctime, mtime int64
		err := rows.Scan(&fdRule.ID, &fdRule.Name, &fdRule.Namespace, &fdRule.Revision,
			&fdRule.Description, &fdRule.DstService, &fdRule.DstNamespace,
			&fdRule.DstMethod, &fdRule.Rule, &ctime, &mtime)
		if err != nil {
			log.Errorf("[Store][database] fetch brief fault detect rule scan err: %s", err.Error())
			return nil, err
		}
		fdRule.CreateTime = time.Unix(ctime, 0)
		fdRule.ModifyTime = time.Unix(mtime, 0)
		out = append(out, &fdRule)
	}
	if err := rows.Err(); err != nil {
		log.Errorf("[Store][database] fetch brief fault detect rule next err: %s", err.Error())
		return nil, err
	}
	return out, nil
}
