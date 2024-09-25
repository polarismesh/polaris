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

	"go.uber.org/zap"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

type laneStore struct {
	master *BaseDB
	slave  *BaseDB
}

// AddLaneGroup 添加泳道组
func (l *laneStore) AddLaneGroup(tx store.Tx, item *model.LaneGroup) error {
	if err := l.cleanSoftDeletedRules(); err != nil {
		return err
	}

	dbTx := tx.GetDelegateTx().(*BaseTx)
	// 先清理无效的泳道组
	if _, err := dbTx.Exec("DELETE FROM lane_group WHERE name = ? AND flag = 1", item.Name); err != nil {
		log.Error("[Store][Lane] clean invalid lane group", zap.String("id", item.ID),
			zap.String("name", item.Name), zap.Error(err))
		return err
	}
	args := []interface{}{
		item.ID,
		item.Name,
		item.Rule,
		item.Revision,
		item.Description,
	}

	addSql := `
INSERT INTO lane_group (id, name, rule, revision, description, flag
	, ctime, mtime)
VALUES (?, ?, ?, ?, ?, 0, sysdate(), sysdate())
`
	if _, err := dbTx.Exec(addSql, args...); err != nil {
		log.Error("[Store][Lane] add lane group", zap.String("id", item.ID),
			zap.String("name", item.Name), zap.Error(err))
		return store.Error(err)
	}
	return l.upsertLaneRules(dbTx, item, item.LaneRules)
}

// UpdateLaneGroup 更新泳道组
func (l *laneStore) UpdateLaneGroup(tx store.Tx, item *model.LaneGroup) error {
	if err := l.cleanSoftDeletedRules(); err != nil {
		return err
	}

	dbTx := tx.GetDelegateTx().(*BaseTx)
	args := []interface{}{
		item.Rule,
		item.Revision,
		item.Description,
		item.ID,
	}

	addSql := "UPDATE lane_group SET rule = ?, revision = ?, description = ?, mtime = sysdate() WHERE id = ?"
	if _, err := dbTx.Exec(addSql, args...); err != nil {
		log.Error("[Store][Lane] update lane group", zap.String("id", item.ID),
			zap.String("name", item.Name), zap.Error(err))
		return store.Error(err)
	}
	return l.upsertLaneRules(dbTx, item, item.LaneRules)
}

// GetLaneGroup 查询泳道组
func (l *laneStore) GetLaneGroup(name string) (*model.LaneGroup, error) {
	querySql := `
SELECT id, name, rule, description, revision, flag, UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime) FROM lane_group WHERE flag = 0 AND name = ?
`
	result := make([]*model.LaneGroup, 0, 1)
	err := l.master.processWithTransaction("GetLaneGroup", func(tx *BaseTx) error {
		rows, err := tx.Query(querySql, name)
		if err != nil {
			log.Error("[Store][Lane] select one lane group", zap.String("querySql", querySql), zap.Error(err))
			return err
		}
		if err := transferLaneGroups(rows, func(group *model.LaneGroup) {
			result = append(result, group)
		}); err != nil {
			log.Error("[Store][Lane] transfer one lane group row", zap.Error(err))
			return err
		}
		return tx.Commit()
	})
	if err != nil {
		return nil, store.Error(err)
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result[0], nil
}

// GetLaneGroupByID .
func (l *laneStore) GetLaneGroupByID(id string) (*model.LaneGroup, error) {
	querySql := `
SELECT id, name, rule, description, revision, flag, UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime) FROM lane_group WHERE flag = 0 AND id = ?
`
	result := make([]*model.LaneGroup, 0, 1)
	err := l.master.processWithTransaction("GetLaneGroupByID", func(tx *BaseTx) error {
		rows, err := tx.Query(querySql, id)
		if err != nil {
			log.Error("[Store][Lane] select one lane group", zap.String("querySql", querySql), zap.Error(err))
			return err
		}
		if err := transferLaneGroups(rows, func(group *model.LaneGroup) {
			result = append(result, group)
		}); err != nil {
			log.Error("[Store][Lane] transfer one lane group row", zap.Error(err))
			return err
		}
		return tx.Commit()
	})
	if err != nil {
		return nil, store.Error(err)
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result[0], nil
}

func (l *laneStore) LockLaneGroup(tx store.Tx, name string) (*model.LaneGroup, error) {
	querySql := `
SELECT id, name, rule, description, revision
	, flag, UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime)
FROM lane_group
WHERE flag = 0
	AND name = ?
FOR UPDATE
`
	dbTx := tx.GetDelegateTx().(*BaseTx)
	result := make([]*model.LaneGroup, 0, 1)
	rows, err := dbTx.Query(querySql, name)
	if err != nil {
		log.Error("[Store][Lane] select one lane group", zap.String("querySql", querySql), zap.String("name", name),
			zap.Error(err))
		return nil, err
	}
	if err := transferLaneGroups(rows, func(group *model.LaneGroup) {
		result = append(result, group)
	}); err != nil {
		log.Error("[Store][Lane] transfer one lane group row", zap.String("name", name), zap.Error(err))
		return nil, store.Error(err)
	}
	if len(result) == 0 {
		return nil, nil
	}
	rules, err := l.getLaneRulesByGroup(dbTx, []string{name})
	if err != nil {
		log.Error("[Store][Lane] load lane_group all lane_rule", zap.String("name", name), zap.Error(err))
		return nil, store.Error(err)
	}
	if len(rules) != 0 {
		result[0].LaneRules = rules[name]
	}
	return result[0], nil
}

// GetLaneGroups 查询泳道组
func (l *laneStore) GetLaneGroups(filter map[string]string, offset, limit uint32) (uint32, []*model.LaneGroup, error) {
	countSql := `
SELECT COUNT(*) FROM lane_group WHERE flag = 0 
`
	querySql := `
SELECT id, name, rule, description, revision, flag, UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime) FROM lane_group WHERE flag = 0
`
	conditions := []string{}
	args := []interface{}{}
	for k, v := range filter {
		switch k {
		case "name":
			if v, ok := utils.ParseWildName(v); ok {
				conditions = append(conditions, "name = ?")
				args = append(args, v)
			} else {
				conditions = append(conditions, "name LIKE ?")
				args = append(args, "%"+v+"%")
			}
		case "id":
			conditions = append(conditions, "id = ?")
			args = append(args, v)
		}
	}
	if len(conditions) > 0 {
		countSql += " AND " + strings.Join(conditions, " AND ")
		querySql += " AND " + strings.Join(conditions, " AND ")
	}

	querySql += fmt.Sprintf(" ORDER BY %s %s LIMIT ?, ? ", filter["order_field"], filter["order_type"])

	var count int64
	var result []*model.LaneGroup

	err := l.master.processWithTransaction("GetLaneGroups", func(tx *BaseTx) error {
		row := tx.QueryRow(countSql, args...)
		if err := row.Scan(&count); err != nil {
			log.Error("[Store][Lane] count lane group", zap.String("countSql", countSql), zap.Error(err))
			return err
		}

		// count 阶段不需要分页参数，因此留到这里在进行追加
		args = append(args, offset, limit)
		rows, err := tx.Query(querySql, args...)
		if err != nil {
			log.Error("[Store][Lane] select lane group", zap.String("querySql", querySql), zap.Error(err))
			return err
		}
		if err := transferLaneGroups(rows, func(group *model.LaneGroup) {
			result = append(result, group)
		}); err != nil {
			log.Error("[Store][Lane] transfer lane group row", zap.Error(err))
			return err
		}
		brief := filter[briefSearch] == "true"
		if !brief {
			names := make([]string, 0, len(result))
			for i := range result {
				names = append(names, result[i].Name)
			}
			rules, err := l.getLaneRulesByGroup(tx, names)
			if err != nil {
				return err
			}
			for i := range result {
				item := result[i]
				item.LaneRules = rules[item.Name]
			}
		}
		return tx.Commit()
	})
	if err != nil {
		return 0, nil, store.Error(err)
	}
	return uint32(count), result, nil
}

// DeleteLaneGroup 删除泳道组
func (l *laneStore) DeleteLaneGroup(id string) error {
	err := l.master.processWithTransaction("DeleteLaneGroup", func(tx *BaseTx) error {
		args := []interface{}{
			id,
		}

		addSql := "UPDATE lane_rule SET flag = 1, mtime = sysdate() WHERE group_name IN (SELECT name FROM lane_group WHERE id = ?)"
		if _, err := tx.Exec(addSql, args...); err != nil {
			log.Error("[Store][Lane] delete lane group", zap.String("id", id), zap.Error(err))
			return err
		}

		addSql = "UPDATE lane_group SET flag = 1, mtime = sysdate() WHERE id = ?"
		if _, err := tx.Exec(addSql, args...); err != nil {
			log.Error("[Store][Lane] delete lane group", zap.String("id", id), zap.Error(err))
			return err
		}
		return tx.Commit()
	})
	return store.Error(err)
}

// getLaneRulesByGroup .
func (l *laneStore) getLaneRulesByGroup(tx *BaseTx, names []string) (map[string]map[string]*model.LaneRule, error) {
	if len(names) == 0 {
		return map[string]map[string]*model.LaneRule{}, nil
	}

	querySql := `
SELECT id, name, group_name, rule, revision, priority, description, enable, flag, UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(etime), UNIX_TIMESTAMP(mtime)
	FROM lane_rule WHERE flag = 0 AND group_name IN (%s)
`
	querySql = fmt.Sprintf(querySql, placeholders(len(names)))

	rows, err := tx.Query(querySql, StringsToArgs(names)...)
	if err != nil {
		log.Error("[Store][Lane] fetch lane group all lane_rules", zap.String("sql", querySql), zap.Error(err))
		return nil, store.Error(err)
	}
	result := make(map[string]map[string]*model.LaneRule, len(names))
	if err := transferLaneRules(rows, func(rule *model.LaneRule) {
		if _, ok := result[rule.LaneGroup]; !ok {
			result[rule.LaneGroup] = make(map[string]*model.LaneRule, 32)
		}
		result[rule.LaneGroup][rule.ID] = rule
	}); err != nil {
		return nil, store.Error(err)
	}
	return result, nil
}

// upsertLaneRules 添加通道规则
func (l *laneStore) upsertLaneRules(tx *BaseTx, group *model.LaneGroup, items map[string]*model.LaneRule) error {
	// 先清理到不再 model.Lane[] 中的泳道规则
	if len(items) > 0 {
		// 如果 items.size > 0，只清理不再 items 里面的泳道规则
		args := make([]interface{}, 0, len(items))
		args = append(args, group.Name)
		for i := range items {
			args = append(args, items[i].Name)
		}

		cleanSql := fmt.Sprintf("UPDATE lane_rule SET flag = 1 WHERE group_name = ? AND name NOT IN (%s)", placeholders(len(items)))
		if _, err := tx.Exec(cleanSql, args...); err != nil {
			log.Error("[Store][Lane] clean invalid lane rule", zap.String("sql", cleanSql), zap.Any("args", args), zap.Error(err))
			return store.Error(err)
		}
	} else {
		// 如果 items.size == 0, 则直接清空所有的泳道规则
		if _, err := tx.Exec("UPDATE lane_rule SET flag = 1 WHERE group_name = ?", group.Name); err != nil {
			log.Error("[Store][Lane] clean invalid lane rule", zap.String("group", group.Name), zap.Error(err))
			return store.Error(err)
		}
	}

	for i := range items {
		item := items[i]
		var args []interface{}

		var upsertSql string
		if item.IsAdd() {
			args = []interface{}{
				item.ID,
				item.Name,
				item.LaneGroup,
				item.Rule,
				item.Revision,
				item.Priority,
				item.Description,
				item.Enable,
			}
			addSql := `
INSERT INTO lane_rule (id, name, group_name, rule, revision, priority, description, enable, flag
	, ctime, etime, mtime)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0
	, sysdate(), %s, sysdate())
`
			etimeStr := "sysdate()"
			if !item.Enable {
				etimeStr = emptyEnableTime
			}
			upsertSql = fmt.Sprintf(addSql, etimeStr)
		} else {
			args = []interface{}{
				item.Rule,
				item.Revision,
				item.Priority,
				item.Description,
				item.Enable,
				item.ID,
			}
			if item.IsChangeEnable() {
				addSql := `
UPDATE lane_rule SET rule = ?, revision = ?, priority = ?, description = ?, enable = ?
	, etime = %s, mtime = sysdate() WHERE id = ?
`
				etimeStr := "sysdate()"
				if !item.Enable {
					etimeStr = emptyEnableTime
				}
				upsertSql = fmt.Sprintf(addSql, etimeStr)
			} else {
				upsertSql = `
UPDATE lane_rule SET rule = ?, revision = ?, priority = ?, description = ?, enable = ?
	, mtime = sysdate() WHERE id = ?
`
			}
		}
		if _, err := tx.Exec(upsertSql, args...); err != nil {
			log.Error("[Store][Lane] add lane rule", zap.String("id", item.ID), zap.String("sql", upsertSql),
				zap.String("group", item.LaneGroup), zap.String("name", item.Name), zap.Error(err))
			return store.Error(err)
		}
	}
	return nil
}

// GetMoreLaneGroups 获取泳道规则列表到缓存层
func (l *laneStore) GetMoreLaneGroups(mtime time.Time, firstUpdate bool) (map[string]*model.LaneGroup, error) {
	if firstUpdate {
		mtime = time.Unix(0, 1)
	}
	deltaGroupSql := `
SELECT id, name, rule, description
	, revision, flag, UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime)
FROM lane_group
WHERE mtime >= FROM_UNIXTIME(?)
`

	deltaRuleSql := `
SELECT
  lr.id,
  lr.name,
  group_name,
  lr.rule,
  lr.revision,
  lr.priority,
  lr.description,
  enable,
  lr.flag,
  UNIX_TIMESTAMP(lr.ctime),
  UNIX_TIMESTAMP(lr.etime),
  UNIX_TIMESTAMP(lr.mtime)
FROM
  lane_rule lr
  LEFT JOIN lane_group lg ON lr.group_name = lg.name
WHERE
  lg.mtime >= FROM_UNIXTIME(?)
`
	deltaGroups := map[string]*model.LaneGroup{}
	var deltaRules []*model.LaneRule

	err := l.slave.processWithTransaction("GetMoreLaneGroups", func(tx *BaseTx) error {
		rows, err := tx.Query(deltaGroupSql, mtime)
		if err != nil {
			log.Error("[Store][Lane] delta lane group", zap.String("querySql", deltaGroupSql), zap.Error(err))
			return err
		}
		if err := transferLaneGroups(rows, func(group *model.LaneGroup) {
			group.LaneRules = make(map[string]*model.LaneRule)
			deltaGroups[group.Name] = group
		}); err != nil {
			log.Error("[Store][Lane] transfer lane group row", zap.Error(err))
			return err
		}
		if len(deltaGroups) > 0 {
			// 走 join 操作获取每个 group 下的 lane_rule 列表
			rows, err := tx.Query(deltaRuleSql, mtime)
			if err != nil {
				log.Error("[Store][Lane] delta lane rule", zap.String("querySql", deltaRuleSql), zap.Error(err))
				return err
			}
			if err := transferLaneRules(rows, func(rule *model.LaneRule) {
				deltaRules = append(deltaRules, rule)
			}); err != nil {
				log.Error("[Store][Lane] transfer lane rule row", zap.Error(err))
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, store.Error(err)
	}
	for i := range deltaRules {
		item := deltaRules[i]
		group, ok := deltaGroups[item.LaneGroup]
		if !ok {
			continue
		}
		group.LaneRules[item.ID] = item
	}
	return deltaGroups, nil
}

// GetLaneRuleMaxPriority 获取当前泳道规则的最大优先级 ID
func (l *laneStore) GetLaneRuleMaxPriority() (int32, error) {
	var maxPriority int32
	err := l.master.processWithTransaction("GetLaneRuleMaxPriority", func(tx *BaseTx) error {
		addSql := "SELECT IFNULL(max(priority), 0) FROM lane_rule WHERE flag = 0"
		row := tx.QueryRow(addSql)
		if err := row.Scan(&maxPriority); err != nil {
			log.Error("[Store][Lane] get current lane_rule max priority", zap.Error(err))
		}
		return tx.Commit()
	})
	return maxPriority, store.Error(err)
}

// cleanSoftDeletedRules .
func (l *laneStore) cleanSoftDeletedRules() error {
	err := l.master.processWithTransaction("cleanSoftDeletedRules", func(tx *BaseTx) error {
		if _, err := tx.Exec("DELETE FROM lane_rule WHERE flag = 1"); err != nil {
			log.Error("[Store][Lane] clean soft delete lane_rule", zap.Error(err))
		}
		return tx.Commit()
	})
	return store.Error(err)
}

func transferLaneGroups(rows *sql.Rows, op func(group *model.LaneGroup)) error {
	if rows == nil {
		return nil
	}
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		item := &model.LaneGroup{}
		var ctime, mtime int64
		var flag int

		if err := rows.Scan(&item.ID, &item.Name, &item.Rule, &item.Description, &item.Revision, &flag, &ctime, &mtime); err != nil {
			return err
		}
		item.Valid = flag == 0
		item.CreateTime = time.Unix(ctime, 0)
		item.ModifyTime = time.Unix(mtime, 0)
		op(item)
	}
	return nil
}

func transferLaneRules(rows *sql.Rows, op func(rule *model.LaneRule)) error {
	if rows == nil {
		return nil
	}
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		item := &model.LaneRule{}
		var ctime, etime, mtime int64
		var flag, enable int

		if err := rows.Scan(&item.ID, &item.Name, &item.LaneGroup, &item.Rule, &item.Revision, &item.Priority, &item.Description,
			&enable, &flag, &ctime, &etime, &mtime); err != nil {
			return err
		}
		item.Valid = flag == 0
		item.Enable = enable == 1
		item.CreateTime = time.Unix(ctime, 0)
		item.EnableTime = time.Unix(etime, 0)
		item.ModifyTime = time.Unix(mtime, 0)
		op(item)
	}

	return nil
}
