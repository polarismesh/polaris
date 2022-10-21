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
	"time"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/store"
)

// circuitBreakerStore 的实现
type circuitBreakerStore struct {
	master *BaseDB
	slave  *BaseDB
}

// CreateCircuitBreaker 创建一个新的熔断规则
func (c *circuitBreakerStore) CreateCircuitBreaker(cb *model.CircuitBreaker) error {
	if err := c.cleanCircuitBreaker(cb.ID, cb.Version); err != nil {
		log.Errorf("[Store][circuitBreaker] clean master for circuit breaker(%s, %s) err: %s",
			cb.ID, cb.Version, err.Error())
		return store.Error(err)
	}

	str := `insert into circuitbreaker_rule
			(id, version, name, namespace, business, department, comment, inbounds, 
			outbounds, token, owner, revision, flag, ctime, mtime)
			values(?,?,?,?,?,?,?,?,?,?,?,?,?,sysdate(),sysdate())`
	if _, err := c.master.Exec(str, cb.ID, cb.Version, cb.Name, cb.Namespace, cb.Business, cb.Department,
		cb.Comment, cb.Inbounds, cb.Outbounds, cb.Token, cb.Owner, cb.Revision, 0); err != nil {
		log.Errorf("[Store][circuitBreaker] create circuit breaker(%s, %s, %s) err: %s",
			cb.ID, cb.Name, cb.Version, err.Error())
		return store.Error(err)
	}

	return nil
}

// TagCircuitBreaker 给master熔断规则打一个version tag
func (c *circuitBreakerStore) TagCircuitBreaker(cb *model.CircuitBreaker) error {
	if err := c.cleanCircuitBreaker(cb.ID, cb.Version); err != nil {
		log.Errorf("[Store][circuitBreaker] clean tag for circuit breaker(%s, %s) err: %s",
			cb.ID, cb.Version, err.Error())
		return store.Error(err)
	}

	if err := c.tagCircuitBreaker(cb); err != nil {
		log.Errorf("[Store][circuitBreaker] create tag for circuit breaker(%s, %s) err: %s",
			cb.ID, cb.Version, err.Error())
		return store.Error(err)
	}

	return nil
}

// tagCircuitBreaker 给master熔断规则打一个version tag的内部函数
func (c *circuitBreakerStore) tagCircuitBreaker(cb *model.CircuitBreaker) error {
	// 需要保证master规则存在
	str := `insert into circuitbreaker_rule
			(id, version, name, namespace, business, department, comment, inbounds, 
			outbounds, token, owner, revision, ctime, mtime) 
			select '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', 
			'%s', '%s', '%s', '%s', sysdate(), sysdate() from circuitbreaker_rule 
			where id = ? and version = 'master'`
	str = fmt.Sprintf(str, cb.ID, cb.Version, cb.Name, cb.Namespace, cb.Business, cb.Department, cb.Comment,
		cb.Inbounds, cb.Outbounds, cb.Token, cb.Owner, cb.Revision)
	result, err := c.master.Exec(str, cb.ID)
	if err != nil {
		log.Errorf("[Store][CircuitBreaker] exec create tag sql(%s) err: %s", str, err.Error())
		return err
	}

	if err := checkDataBaseAffectedRows(result, 1); err != nil {
		if store.Code(err) == store.AffectedRowsNotMatch {
			return store.NewStatusError(store.NotFoundMasterConfig, "not found master config")
		}
		log.Errorf("[Store][CircuitBreaker] tag rule affected rows err: %s", err.Error())
		return err
	}

	return nil
}

// ReleaseCircuitBreaker 发布熔断规则
func (c *circuitBreakerStore) ReleaseCircuitBreaker(cbr *model.CircuitBreakerRelation) error {
	if err := c.cleanCircuitBreakerRelation(cbr); err != nil {
		return store.Error(err)
	}

	if err := c.releaseCircuitBreaker(cbr); err != nil {
		log.Errorf("[Store][CircuitBreaker] release rule err: %s", err.Error())
		return store.Error(err)
	}

	return nil
}

// releaseCircuitBreaker 发布熔断规则的内部函数
// @note 可能存在服务的规则，由旧的更新到新的场景
func (c *circuitBreakerStore) releaseCircuitBreaker(cbr *model.CircuitBreakerRelation) error {
	// 发布规则时，需要保证规则已经被标记
	str := `insert into circuitbreaker_rule_relation(service_id, rule_id, rule_version, flag, ctime, mtime)
		select '%s', '%s', '%s', 0, sysdate(), sysdate() from service, circuitbreaker_rule 
		where service.id = ? and service.flag = 0 
		and circuitbreaker_rule.id = ? and circuitbreaker_rule.version = ? 
		and circuitbreaker_rule.flag = 0 
		on DUPLICATE key update 
		rule_id = ?, rule_version = ?, flag = 0, mtime = sysdate()`
	str = fmt.Sprintf(str, cbr.ServiceID, cbr.RuleID, cbr.RuleVersion)
	log.Infof("[Store][CircuitBreaker] exec release sql(%s)", str)
	result, err := c.master.Exec(str, cbr.ServiceID, cbr.RuleID, cbr.RuleVersion, cbr.RuleID, cbr.RuleVersion)
	if err != nil {
		log.Errorf("[Store][CircuitBreaker] release exec sql(%s) err: %s", str, err.Error())
		return err
	}
	if err := checkDataBaseAffectedRows(result, 1, 2); err != nil {
		if store.Code(err) == store.AffectedRowsNotMatch {
			return store.NewStatusError(store.NotFoundTagConfigOrService, "not found tag config or service")
		}
		log.Errorf("[Store][CircuitBreaker] release rule affected rows err: %s", err.Error())
		return err
	}

	return nil
}

// UnbindCircuitBreaker 解绑熔断规则
func (c *circuitBreakerStore) UnbindCircuitBreaker(serviceID, ruleID, ruleVersion string) error {
	str := `update circuitbreaker_rule_relation set flag = 1, mtime = sysdate() where service_id = ? and rule_id = ?
					and rule_version = ?`
	if _, err := c.master.Exec(str, serviceID, ruleID, ruleVersion); err != nil {
		log.Errorf("[Store][CircuitBreaker] delete relation(%s) err: %s", serviceID, err.Error())
		return err
	}

	return nil
}

// DeleteTagCircuitBreaker 删除非master熔断规则
func (c *circuitBreakerStore) DeleteTagCircuitBreaker(id string, version string) error {
	// 需要保证规则无绑定服务
	str := `update circuitbreaker_rule set flag = 1, mtime = sysdate()
			where id = ? and version = ? 
			and id not in 
			(select DISTINCT(rule_id) from circuitbreaker_rule_relation 
				where rule_id = ? and rule_version = ? and flag = 0)`
	log.Infof("[Store][circuitBreaker] delete rule id(%s) version(%s), sql(%s)", id, version, str)
	if _, err := c.master.Exec(str, id, version, id, version); err != nil {
		log.Errorf("[Store][CircuitBreaker] delete tag rule(%s, %s) exec err: %s", id, version, err.Error())
		return err
	}

	return nil
}

// DeleteMasterCircuitBreaker 删除master熔断规则
func (c *circuitBreakerStore) DeleteMasterCircuitBreaker(id string) error {
	// 需要保证所有已标记的规则无绑定服务
	str := `update circuitbreaker_rule set flag = 1, mtime = sysdate()
			where id = ? and version = 'master'
			and id not in 
			(select DISTINCT(rule_id) from circuitbreaker_rule_relation 
				where rule_id = ? and flag = 0)`
	log.Infof("[Store][CircuitBreaker] delete master rule(%s) sql(%s)", id, str)
	if _, err := c.master.Exec(str, id, id); err != nil {
		log.Errorf("[Store][CircuitBreaker] delete master rule(%s) exec err: %s", id, err.Error())
		return err
	}

	return nil
}

// UpdateCircuitBreaker 修改熔断规则
// @note 只允许修改master熔断规则
func (c *circuitBreakerStore) UpdateCircuitBreaker(cb *model.CircuitBreaker) error {
	str := `update circuitbreaker_rule set business = ?, department = ?, comment = ?,
			inbounds = ?, outbounds = ?, token = ?, owner = ?, revision = ?, mtime = sysdate() 
			where id = ? and version = ?`

	if _, err := c.master.Exec(str, cb.Business, cb.Department, cb.Comment, cb.Inbounds,
		cb.Outbounds, cb.Token, cb.Owner, cb.Revision, cb.ID, cb.Version); err != nil {
		log.Errorf("[Store][CircuitBreaker] update rule(%s,%s) exec err: %s", cb.ID, cb.Version, err.Error())
		return err
	}

	return nil
}

// GetCircuitBreaker 获取熔断规则
func (c *circuitBreakerStore) GetCircuitBreaker(id, version string) (*model.CircuitBreaker, error) {
	str := `select id, version, name, namespace, IFNULL(business, ""), IFNULL(department, ""), IFNULL(comment, ""),
			inbounds, outbounds, token, owner, revision, flag, unix_timestamp(ctime), unix_timestamp(mtime) 
			from circuitbreaker_rule 
			where id = ? and version = ? and flag = 0`
	rows, err := c.master.Query(str, id, version)
	if err != nil {
		log.Errorf("[Store][CircuitBreaker] query circuitbreaker_rule with id(%s) and version(%s) err: %s",
			id, version, err.Error())
		return nil, err
	}

	out, err := fetchCircuitBreakerRows(rows)
	if err != nil {
		return nil, err
	}

	if len(out) == 0 {
		return nil, nil
	}

	return out[0], nil
}

// GetCircuitBreakerRelation 获取已标记熔断规则的绑定关系
func (c *circuitBreakerStore) GetCircuitBreakerRelation(ruleID, ruleVersion string) (
	[]*model.CircuitBreakerRelation, error) {
	str := genQueryCircuitBreakerRelation()
	str += `where rule_id = ? and rule_version = ? and flag = 0`
	rows, err := c.master.Query(str, ruleID, ruleVersion)
	if err != nil {
		log.Errorf("[Store][CircuitBreaker] query circuitbreaker_rule_relation "+
			"with rule_id(%s) and rule_version(%s) err: %s",
			ruleID, ruleVersion, err.Error())
		return nil, err
	}

	out, err := fetchCircuitBreakerRelationRows(rows)
	if err != nil {
		return nil, err
	}

	return out, nil
}

// GetCircuitBreakerMasterRelation 获取熔断规则master版本的绑定关系
func (c *circuitBreakerStore) GetCircuitBreakerMasterRelation(ruleID string) (
	[]*model.CircuitBreakerRelation, error) {
	str := genQueryCircuitBreakerRelation()
	str += `where rule_id = ? and flag = 0`
	rows, err := c.master.Query(str, ruleID)
	if err != nil {
		log.Errorf("[Store][CircuitBreaker] query circuitbreaker_rule_relation with rule_id(%s) err: %s",
			ruleID, err.Error())
		return nil, err
	}

	out, err := fetchCircuitBreakerRelationRows(rows)
	if err != nil {
		return nil, err
	}

	return out, nil
}

// GetCircuitBreakerForCache 根据修改时间拉取增量熔断规则
func (c *circuitBreakerStore) GetCircuitBreakerForCache(mtime time.Time, firstUpdate bool) (
	[]*model.ServiceWithCircuitBreaker, error) {
	str := genQueryCircuitBreakerWithServiceID()
	str += `where circuitbreaker_rule_relation.mtime > FROM_UNIXTIME(?) and rule_id = id and rule_version = version
			and circuitbreaker_rule.flag = 0`
	if firstUpdate {
		str += ` and circuitbreaker_rule_relation.flag != 1`
	}
	rows, err := c.slave.Query(str, timeToTimestamp(mtime))
	if err != nil {
		log.Errorf("[Store][CircuitBreaker] query circuitbreaker_rule_relation with mtime err: %s",
			err.Error())
		return nil, err
	}
	circuitBreakers, err := fetchCircuitBreakerAndServiceRows(rows)
	if err != nil {
		return nil, err
	}
	return circuitBreakers, nil
}

// GetCircuitBreakerVersions 获取熔断规则的所有版本
func (c *circuitBreakerStore) GetCircuitBreakerVersions(id string) ([]string, error) {
	str := `select version from circuitbreaker_rule where id = ? and flag = 0 order by mtime desc`
	rows, err := c.master.Query(str, id)
	if err != nil {
		log.Errorf("[Store][CircuitBreaker] get circuit breaker(%s) versions query err: %s", id, err.Error())
		return nil, err
	}

	var versions []string
	var version string
	for rows.Next() {
		if err := rows.Scan(&version); err != nil {
			log.Errorf("[Store][CircuitBreaker] get circuit breaker(%s) versions scan err: %s", id, err.Error())
			return nil, err
		}

		versions = append(versions, version)
	}
	if err := rows.Err(); err != nil {
		log.Errorf("[Store][CircuitBreaker] get circuit breaker(%s) versions next err: %s", id, err.Error())
		return nil, err
	}

	return versions, nil
}

// ListMasterCircuitBreakers 获取master熔断规则
func (c *circuitBreakerStore) ListMasterCircuitBreakers(filters map[string]string, offset uint32, limit uint32) (
	*model.CircuitBreakerDetail, error) {
	// 获取master熔断规则
	selectStr := `select rule.id, rule.version, rule.name, rule.namespace, IFNULL(rule.business, ""),
				IFNULL(rule.department, ""), IFNULL(rule.comment, ""), rule.inbounds, rule.outbounds, 
				rule.owner, rule.revision, 
				unix_timestamp(rule.ctime), unix_timestamp(rule.mtime) from circuitbreaker_rule as rule `
	countStr := `select count(*) from circuitbreaker_rule as rule `
	whereStr := "where rule.version = 'master' and rule.flag = 0 "
	orderStr := "order by rule.mtime desc "
	pageStr := "limit ?, ? "

	var args []interface{}
	filterStr, filterArgs := genRuleFilterSQL("rule", filters)
	if filterStr != "" {
		whereStr += "and " + filterStr
		args = append(args, filterArgs...)
	}

	out := &model.CircuitBreakerDetail{
		Total:               0,
		CircuitBreakerInfos: make([]*model.CircuitBreakerInfo, 0),
	}
	err := c.master.QueryRow(countStr+whereStr, args...).Scan(&out.Total)
	switch {
	case err == sql.ErrNoRows:
		out.Total = 0
		return out, nil
	case err != nil:
		log.Errorf("[Store][CircuitBreaker] list master circuitbreakers query count err: %s", err.Error())
		return nil, err
	default:
	}

	args = append(args, offset)
	args = append(args, limit)

	rows, err := c.master.Query(selectStr+whereStr+orderStr+pageStr, args...)
	if err != nil {
		log.Errorf("[Store][CircuitBreaker] list master circuitbreaker query err: %s", err.Error())
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var ctime, mtime int64
	for rows.Next() {
		var entry model.CircuitBreaker
		if err := rows.Scan(&entry.ID, &entry.Version, &entry.Name, &entry.Namespace, &entry.Business,
			&entry.Department, &entry.Comment, &entry.Inbounds, &entry.Outbounds, &entry.Owner, &entry.Revision,
			&ctime, &mtime); err != nil {
			log.Errorf("[Store][CircuitBreaker] list master circuitbreakers rows scan err: %s", err.Error())
			return nil, err
		}

		entry.CreateTime = time.Unix(ctime, 0)
		entry.ModifyTime = time.Unix(mtime, 0)
		cbEntry := &model.CircuitBreakerInfo{CircuitBreaker: &entry}
		out.CircuitBreakerInfos = append(out.CircuitBreakerInfos, cbEntry)
	}
	if err := rows.Err(); err != nil {
		log.Errorf("[Store][CircuitBreaker] list master circuitbreakers rows next err: %s", err.Error())
		return nil, err
	}

	return out, nil
}

// ListReleaseCircuitBreakers 获取已发布规则及服务
func (c *circuitBreakerStore) ListReleaseCircuitBreakers(filters map[string]string, offset, limit uint32) (
	*model.CircuitBreakerDetail, error) {
	selectStr := `select rule_id, rule_version, unix_timestamp(relation.ctime), unix_timestamp(relation.mtime),
				name, namespace, service.owner from circuitbreaker_rule_relation as relation, service `
	whereStr := `where relation.flag = 0 and relation.service_id = service.id `
	orderStr := "order by relation.mtime desc "
	pageStr := "limit ?, ?"

	countStr := `select count(*) from circuitbreaker_rule_relation as relation where relation.flag = 0 `

	var args []interface{}
	filterStr, filterArgs := genRuleFilterSQL("relation", filters)
	if filterStr != "" {
		countStr += "and " + filterStr
		whereStr += "and " + filterStr
		args = append(args, filterArgs...)
	}

	out := &model.CircuitBreakerDetail{
		Total:               0,
		CircuitBreakerInfos: make([]*model.CircuitBreakerInfo, 0),
	}

	err := c.master.QueryRow(countStr, args...).Scan(&out.Total)
	switch {
	case err == sql.ErrNoRows:
		out.Total = 0
		return out, nil
	case err != nil:
		log.Errorf("[Store][CircuitBreaker] list tag circuitbreakers query count err: %s", err.Error())
		return nil, err
	default:
	}

	args = append(args, offset)
	args = append(args, limit)

	rows, err := c.master.Query(selectStr+whereStr+orderStr+pageStr, args...)
	if err != nil {
		log.Errorf("[Store][CircuitBreaker] list tag circuitBreakers query err: %s", err.Error())
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var ctime, mtime int64
	for rows.Next() {
		var entry model.CircuitBreaker
		var service model.Service
		if err := rows.Scan(&entry.ID, &entry.Version, &ctime, &mtime, &service.Name, &service.Namespace,
			&service.Owner); err != nil {
			log.Errorf("[Store][CircuitBreaker] list tag circuitBreakers scan err: %s", err.Error())
			return nil, err
		}

		service.CreateTime = time.Unix(ctime, 0)
		service.ModifyTime = time.Unix(mtime, 0)

		info := &model.CircuitBreakerInfo{
			CircuitBreaker: &entry,
			Services: []*model.Service{
				&service,
			},
		}

		out.CircuitBreakerInfos = append(out.CircuitBreakerInfos, info)
	}

	return out, nil
}

// GetCircuitBreakersByService 根据服务获取熔断规则
func (c *circuitBreakerStore) GetCircuitBreakersByService(name string, namespace string) (
	*model.CircuitBreaker, error) {
	str := `select rule.id, rule.version, rule.name, rule.namespace, IFNULL(rule.business, ""),
			IFNULL(rule.comment, ""), IFNULL(rule.department, ""),
			rule.inbounds, rule.outbounds, rule.owner, rule.revision,
			unix_timestamp(rule.ctime), unix_timestamp(rule.mtime) 
			from circuitbreaker_rule as rule, circuitbreaker_rule_relation as relation, service 
			where service.id = relation.service_id 
			and relation.rule_id = rule.id and relation.rule_version = rule.version
			and relation.flag = 0 and service.flag = 0 and rule.flag = 0 
			and service.name = ? and service.namespace = ?`
	var breaker model.CircuitBreaker
	var ctime, mtime int64
	err := c.master.QueryRow(str, name, namespace).Scan(&breaker.ID, &breaker.Version, &breaker.Name,
		&breaker.Namespace, &breaker.Business, &breaker.Comment, &breaker.Department,
		&breaker.Inbounds, &breaker.Outbounds, &breaker.Owner, &breaker.Revision, &ctime, &mtime)
	switch {
	case err == sql.ErrNoRows:
		return nil, nil
	case err != nil:
		log.Errorf("[Store][CircuitBreaker] get tag circuitbreaker with service(%s, %s) err: %s",
			name, namespace, err.Error())
		return nil, err
	default:
		breaker.CreateTime = time.Unix(ctime, 0)
		breaker.ModifyTime = time.Unix(mtime, 0)
		return &breaker, nil
	}
}

// cleanCircuitBreakerRelation 清理无效的熔断规则关系
func (c *circuitBreakerStore) cleanCircuitBreakerRelation(cbr *model.CircuitBreakerRelation) error {
	log.Infof("[Store][CircuitBreaker] clean relation for service(%s)", cbr.ServiceID)
	str := `delete from circuitbreaker_rule_relation where service_id = ? and flag = 1`
	if _, err := c.master.Exec(str, cbr.ServiceID); err != nil {
		log.Errorf("[Store][CircuitBreaker] clean relation service(%s) err: %s",
			cbr.ServiceID, err.Error())
		return err
	}

	return nil
}

// cleanCircuitBreaker 彻底清理熔断规则
func (c *circuitBreakerStore) cleanCircuitBreaker(id string, version string) error {
	str := `delete from circuitbreaker_rule where id = ? and version = ? and flag = 1`
	if _, err := c.master.Exec(str, id, version); err != nil {
		log.Errorf("[Store][database] clean circuit breaker(%s) err: %s", id, err.Error())
		return store.Error(err)
	}
	return nil
}

// fetchCircuitBreakerRows 读取circuitbreaker_rule的数据
func fetchCircuitBreakerRows(rows *sql.Rows) ([]*model.CircuitBreaker, error) {
	defer rows.Close()
	var out []*model.CircuitBreaker
	for rows.Next() {
		var entry model.CircuitBreaker
		var flag int
		var ctime, mtime int64
		err := rows.Scan(&entry.ID, &entry.Version, &entry.Name, &entry.Namespace, &entry.Business, &entry.Department,
			&entry.Comment, &entry.Inbounds, &entry.Outbounds, &entry.Token, &entry.Owner, &entry.Revision,
			&flag, &ctime, &mtime)
		if err != nil {
			log.Errorf("[Store][CircuitBreaker] fetch circuitbreaker_rule scan err: %s", err.Error())
			return nil, err
		}

		entry.CreateTime = time.Unix(ctime, 0)
		entry.ModifyTime = time.Unix(mtime, 0)
		entry.Valid = true
		if flag == 1 {
			entry.Valid = false
		}

		out = append(out, &entry)
	}
	if err := rows.Err(); err != nil {
		log.Errorf("[Store][CircuitBreaker] fetch circuitbreaker_rule next err: %s", err.Error())
		return nil, err
	}

	return out, nil
}

// fetchCircuitBreakerRelationRows 读取circuitbreaker_rule_relation的数据
func fetchCircuitBreakerRelationRows(rows *sql.Rows) ([]*model.CircuitBreakerRelation, error) {
	defer rows.Close()
	var out []*model.CircuitBreakerRelation
	for rows.Next() {
		var entry model.CircuitBreakerRelation
		var flag int
		var ctime, mtime int64
		err := rows.Scan(&entry.ServiceID, &entry.RuleID, &entry.RuleVersion, &flag, &ctime, &mtime)
		if err != nil {
			log.Errorf("[Store][CircuitBreaker] fetch circuitbreaker_rule_relation scan err: %s", err.Error())
			return nil, err
		}

		entry.CreateTime = time.Unix(ctime, 0)
		entry.ModifyTime = time.Unix(mtime, 0)
		entry.Valid = true
		if flag == 1 {
			entry.Valid = false
		}

		out = append(out, &entry)
	}

	if err := rows.Err(); err != nil {
		log.Errorf("[Store][CircuitBreaker] fetch circuitbreaker_rule_relation next err: %s", err.Error())
		return nil, err
	}

	return out, nil
}

// fetchCircuitBreakerAndServiceRows 读取circuitbreaker_rule和circuitbreaker_rule_relation的数据
func fetchCircuitBreakerAndServiceRows(rows *sql.Rows) ([]*model.ServiceWithCircuitBreaker, error) {
	defer rows.Close()
	var out []*model.ServiceWithCircuitBreaker
	for rows.Next() {
		var entry model.ServiceWithCircuitBreaker
		var rule model.CircuitBreaker
		var relationFlag, ruleFlag int
		var relationCtime, relationMtime, ruleCtime, ruleMtime int64
		err := rows.Scan(&entry.ServiceID, &rule.ID, &rule.Version, &relationFlag, &relationCtime, &relationMtime,
			&rule.Name, &rule.Namespace, &rule.Business, &rule.Department, &rule.Comment, &rule.Inbounds, &rule.Outbounds,
			&rule.Token, &rule.Owner, &rule.Revision, &ruleFlag, &ruleCtime, &ruleMtime)
		if err != nil {
			log.Errorf("[Store][CircuitBreaker] fetch circuitbreaker_rule and relation scan err: %s",
				err.Error())
			return nil, err
		}
		entry.CreateTime = time.Unix(relationCtime, 0)
		entry.ModifyTime = time.Unix(relationMtime, 0)
		entry.Valid = true
		if relationFlag == 1 {
			entry.Valid = false
		}
		rule.CreateTime = time.Unix(ruleCtime, 0)
		rule.ModifyTime = time.Unix(ruleMtime, 0)
		rule.Valid = true
		if ruleFlag == 1 {
			rule.Valid = false
		}
		entry.CircuitBreaker = &rule
		out = append(out, &entry)
	}
	if err := rows.Err(); err != nil {
		log.Errorf("[Store][CircuitBreaker] fetch circuitbreaker_rule and relation next err: %s", err.Error())
		return nil, err
	}

	return out, nil
}

// genQueryCircuitBreakerRelation 查询熔断规则绑定关系表的语句
func genQueryCircuitBreakerRelation() string {
	str := `select service_id, rule_id, rule_version, flag, unix_timestamp(ctime), unix_timestamp(mtime)
			from circuitbreaker_rule_relation `
	return str
}

// genQueryCircuitBreakerWithServiceID 根据服务id查询熔断规则的查询语句
func genQueryCircuitBreakerWithServiceID() string {
	str := `select service_id, rule_id, rule_version, circuitbreaker_rule_relation.flag,
			unix_timestamp(circuitbreaker_rule_relation.ctime), unix_timestamp(circuitbreaker_rule_relation.mtime), 
			name, namespace, IFNULL(business, ""), IFNULL(department, ""), IFNULL(comment, ""), inbounds, outbounds, 
			token, owner, revision, circuitbreaker_rule.flag, 
			unix_timestamp(circuitbreaker_rule.ctime), unix_timestamp(circuitbreaker_rule.mtime) 
			from circuitbreaker_rule_relation, circuitbreaker_rule `
	return str
}
