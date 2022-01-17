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

	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	commontime "github.com/polarismesh/polaris-server/common/time"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/store"
	"go.uber.org/zap"
)

var (
	RuleLinkUserFilters map[string]string = map[string]string{
		"principal_id":   "ap.principal_id",
		"principal_type": "ap.principal_role",
	}
)

type strategyStore struct {
	master *BaseDB
	slave  *BaseDB
}

func (s *strategyStore) AddStrategy(strategy *model.StrategyDetail) error {
	if strategy.ID == "" || strategy.Name == "" || strategy.Owner == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"add auth_strategy missing some params, id is %s, name is %s, owner is %s", strategy.ID, strategy.Name, strategy.Owner))
	}

	// 先清理无效数据
	if err := s.cleanInvalidStrategy(strategy.Name); err != nil {
		return err
	}

	err := RetryTransaction("addStrategy", func() error {
		return s.addStrategy(strategy)
	})
	return store.Error(err)
}

func (s *strategyStore) addStrategy(strategy *model.StrategyDetail) error {

	tx, err := s.master.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	isDefault := 0
	if strategy.Default {
		isDefault = 1
	}

	// 保存策略主信息
	saveMainSql := "INSERT INTO auth_strategy(`id`, `name`, `action`, `owner`, `comment`, `flag`, `default`, `revision`) VALUES (?,?,?,?,?,?,?,?)"
	if _, err = tx.Exec(saveMainSql, []interface{}{strategy.ID, strategy.Name, strategy.Action, strategy.Owner, strategy.Comment, 0, isDefault, strategy.Revision}...); err != nil {
		return err
	}

	if err := s.addStrategyPrincipals(tx, strategy.ID, strategy.Principals); err != nil {
		return err
	}

	if err := s.addStrategyResources(tx, strategy.ID, strategy.Resources); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		log.GetAuthLogger().Errorf("[Store][database] add auth_strategy tx commit err: %s", err.Error())
		return err
	}

	return nil
}

// UpdateStrategyMain 更新鉴权规则的主体信息
//  @receiver s
//  @param strategy
//  @return error
func (s *strategyStore) UpdateStrategy(strategy *model.ModifyStrategyDetail) error {
	if strategy.ID == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"update auth_strategy missing some params, id is %s", strategy.ID))
	}

	err := RetryTransaction("updateStrategy", func() error {
		return s.updateStrategy(strategy)
	})
	return store.Error(err)
}

func (s *strategyStore) updateStrategy(strategy *model.ModifyStrategyDetail) error {

	tx, err := s.master.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// 保存策略主信息
	saveMainSql := "UPDATE auth_strategy SET action = ?, comment = ? WHERE id = ?"
	if _, err = tx.Exec(saveMainSql, []interface{}{strategy.Action, strategy.Comment, strategy.ID}...); err != nil {
		return err
	}

	// 调整 principal 信息
	if err := s.addStrategyPrincipals(tx, strategy.ID, strategy.AddPrincipals); err != nil {
		log.GetAuthLogger().Errorf("[Store][Strategy] add strategy principal err: %s", err.Error())
		return err
	}
	if err := s.deleteStrategyPrincipals(tx, strategy.ID, strategy.RemovePrincipals); err != nil {
		log.GetAuthLogger().Errorf("[Store][database] remove strategy principal err: %s", err.Error())
		return err
	}

	// 调整鉴权资源信息
	if err := s.addStrategyResources(tx, strategy.ID, strategy.AddResources); err != nil {
		log.GetAuthLogger().Errorf("[Store][database] add strategy resource err: %s", err.Error())
		return err
	}
	if err := s.deleteStrategyResources(tx, strategy.ID, strategy.RemoveResources); err != nil {
		log.GetAuthLogger().Errorf("[Store][database] remove strategy resource err: %s", err.Error())
		return err
	}

	if err := tx.Commit(); err != nil {
		log.GetAuthLogger().Errorf("[Store][database] update auth_strategy tx commit err: %s", err.Error())
		return err
	}

	return nil
}

func (s *strategyStore) DeleteStrategy(id string) error {
	if id == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"delete auth_strategy missing some params, id is %s", id))
	}

	err := RetryTransaction("deleteStrategy", func() error {
		return s.deleteStrategy(id)
	})
	return store.Error(err)
}

func (s *strategyStore) deleteStrategy(id string) error {

	tx, err := s.master.Begin()
	if err != nil {
		return err
	}

	defer func() { _ = tx.Rollback() }()

	if _, err = tx.Exec("UPDATE auth_strategy SET flag = 1 WHERE id = ?", []interface{}{
		id,
	}...); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		log.GetAuthLogger().Errorf("[Store][database] delete auth_strategy tx commit err: %s", err.Error())
		return err
	}
	return nil
}

// addStrategyPrincipals
func (s *strategyStore) addStrategyPrincipals(tx *BaseTx, id string, principals []model.Principal) error {

	if len(principals) == 0 {
		return nil
	}

	savePrincipalSql := "INSERT IGNORE INTO auth_principal(strategy_id, principal_id, principal_role) VALUES "
	values := make([]string, 0)
	args := make([]interface{}, 0)

	for i := range principals {
		principal := principals[i]
		values = append(values, "(?,?,?)")
		args = append(args, id, principal.PrincipalID, principal.PrincipalRole)
	}

	savePrincipalSql += strings.Join(values, ",")

	log.GetAuthLogger().Debug("add strategy principal", zap.String("sql", savePrincipalSql), zap.Any("args", args))

	_, err := tx.Exec(savePrincipalSql, args...)
	return err
}

// deleteStrategyPrincipals
func (s *strategyStore) deleteStrategyPrincipals(tx *BaseTx, id string, principals []model.Principal) error {
	if len(principals) == 0 {
		return nil
	}

	savePrincipalSql := "DELETE FROM auth_principal WHERE strategy_id = ? AND principal_id = ? AND principal_role = ?"
	for i := range principals {
		principal := principals[i]
		if _, err := tx.Exec(savePrincipalSql, []interface{}{
			id, principal.PrincipalID, principal.PrincipalRole,
		}...); err != nil {
			return err
		}
	}

	return nil
}

func (s *strategyStore) addStrategyResources(tx *BaseTx, id string, resources []model.StrategyResource) error {
	if len(resources) == 0 {
		return nil
	}

	saveResSql := "REPLACE INTO auth_strategy_resource(strategy_id, res_type, res_id, flag) VALUES "
	// 保存策略的资源信息

	values := make([]string, 0)
	args := make([]interface{}, 0)

	for i := range resources {
		resource := resources[i]
		values = append(values, "(?,?,?,?)")
		args = append(args, resource.StrategyID, resource.ResType, resource.ResID, 0)
	}

	if len(values) == 0 {
		return nil
	}

	saveResSql += strings.Join(values, ",")

	log.GetAuthLogger().Debug("add strategy resources", zap.String("sql", saveResSql), zap.Any("args", args))
	_, err := tx.Exec(saveResSql, args...)
	return err
}

func (s *strategyStore) deleteStrategyResources(tx *BaseTx, id string, resources []model.StrategyResource) error {
	if len(resources) == 0 {
		return nil
	}

	for i := range resources {
		resource := resources[i]

		saveResSql := "UPDATE auth_strategy_resource SET flag = 1 WHERE strategy_id = ? AND res_id = ? AND res_type = ?"
		_, err := tx.Exec(saveResSql, []interface{}{resource.StrategyID, resource.ResID, resource.ResType}...)

		if err != nil {
			return err
		}

	}
	return nil
}

// LooseAddStrategyResources
//  @param resources
//  @return error
func (s *strategyStore) LooseAddStrategyResources(resources []model.StrategyResource) error {
	tx, err := s.master.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// 保存策略的资源信息
	for i := range resources {
		saveResSql := "INSERT INTO auth_strategy_resource(strategy_id, res_type, res_id, flag) VALUES (?,?,?,?)"
		args := make([]interface{}, 0)
		resource := resources[i]
		args = append(args, resource.StrategyID, resource.ResType, resource.ResID, 0)

		_, err = tx.Exec(saveResSql, args...)
		if err != nil {
			err = store.Error(err)
			if store.Code(err) == store.DuplicateEntryErr {
				continue
			}
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		log.GetAuthLogger().Errorf("[Store][database] add auth_strategy tx commit err: %s", err.Error())
		return err
	}

	return nil
}

func (s *strategyStore) RemoveStrategyResources(resources []model.StrategyResource) error {
	tx, err := s.master.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for i := range resources {
		args := make([]interface{}, 0)
		resource := resources[i]
		saveResSql := "UPDATE auth_strategy_resource SET flag = 1 WHERE strategy_id = ? AND res_id = ? AND res_type = ?"
		args = append(args, resource.ResID, resource.ResID, resource.ResType)
		_, err := tx.Exec(saveResSql, args...)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		log.GetAuthLogger().Errorf("[Store][database] add auth_strategy tx commit err: %s", err.Error())
		return err
	}

	return nil
}

// GetStrategyDetail
//  @receiver s
//  @param id
//  @return *model.StrategyDetail
//  @return error
func (s *strategyStore) GetStrategyDetail(id string) (*model.StrategyDetail, error) {
	if id == "" {
		return nil, store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"get auth_strategy missing some params, id is %s", id))
	}

	tx, err := s.slave.Begin()
	if err != nil {
		return nil, err
	}

	defer tx.Commit()

	querySql := "SELECT `id`, `name`, `action`, `owner`, `default`, `comment`, `revision`, UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime) FROM auth_strategy WHERE flag = 0 AND id = ?"
	row := tx.QueryRow(querySql, id)

	return s.getStrategyDetail(tx, row)
}

// GetStrategyDetailByName
//  @receiver s
//  @param owner
//  @param name
//  @return *model.StrategyDetail
//  @return error
func (s *strategyStore) GetStrategyDetailByName(owner, name string) (*model.StrategyDetail, error) {
	if name == "" {
		return nil, store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"get auth_strategy missing some params, name is %s", name))
	}

	tx, err := s.slave.Begin()
	if err != nil {
		return nil, err
	}

	defer tx.Commit()

	querySql := "SELECT `id`, `name`, `action`, `owner`, `default`, `comment`, `revision`, UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime) FROM auth_strategy WHERE flag = 0 AND owner = ? AND name = ?"
	row := tx.QueryRow(querySql, owner, name)

	return s.getStrategyDetail(tx, row)
}

// getStrategyDetail
//  @receiver s
//  @param tx
//  @param row
//  @return *model.StrategyDetail
//  @return error
func (s *strategyStore) getStrategyDetail(tx *BaseTx, row *sql.Row) (*model.StrategyDetail, error) {
	var (
		ctime, mtime int64
		isDefault    int16
	)
	ret := new(model.StrategyDetail)
	if err := row.Scan(&ret.ID, &ret.Name, &ret.Action, &ret.Owner, &isDefault, &ret.Comment, &ret.Revision, &ctime, &mtime); err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, nil
		default:
			return nil, store.Error(err)
		}
	}

	ret.CreateTime = time.Unix(ctime, 0)
	ret.ModifyTime = time.Unix(mtime, 0)
	ret.Valid = true
	ret.Default = isDefault == 1

	resArr, err := s.getStrategyResources(s.slave.Query, ret.ID)
	if err != nil {
		return nil, store.Error(err)
	}
	principals, err := s.getStrategyPrincipals(s.slave.Query, ret.ID)
	if err != nil {
		return nil, store.Error(err)
	}

	ret.Resources = resArr
	ret.Principals = principals
	return ret, nil
}

// GetStrategySimpleByName
//  @receiver s
//  @param owner
//  @param name
//  @return *model.Strategy
//  @return error
func (s *strategyStore) GetStrategySimpleByName(owner, name string) (*model.Strategy, error) {
	if owner == "" && name == "" {
		return nil, store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"get auth_strategy missing some params, owner is %s, name is %s", owner, name))
	}

	tx, err := s.slave.Begin()
	if err != nil {
		return nil, err
	}
	querySql := "SELECT `id`, `name`, `action`, `owner`, `default`, `comment`, UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime) FROM auth_strategy WHERE flag = 0 AND owner = ? AND name = ?"
	row := tx.QueryRow(querySql, owner, name)

	var (
		ctime, mtime int64
		isDefault    int16
	)
	ret := new(model.Strategy)
	if err := row.Scan(&ret.ID, &ret.Name, &ret.Action, &ret.Owner, &isDefault, &ret.Comment, &ctime, &mtime); err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, nil
		default:
			return nil, store.Error(err)
		}
	}

	ret.CreateTime = time.Unix(ctime, 0)
	ret.ModifyTime = time.Unix(mtime, 0)
	ret.Valid = true
	ret.Default = isDefault == 1

	return ret, nil
}

func (s *strategyStore) GetSimpleStrategies(filters map[string]string, offset uint32, limit uint32) (uint32, []*model.StrategyDetail, error) {

	if _, ok := filters["principal_id"]; ok {
		return s.listStrategySimpleByUserId(filters, offset, limit)
	} else {
		return s.listStrategySimple(filters, offset, limit)
	}

}

// listStrategySimple
func (s *strategyStore) listStrategySimple(filters map[string]string, offset uint32, limit uint32) (uint32, []*model.StrategyDetail, error) {
	tx, err := s.slave.Begin()
	if err != nil {
		return 0, nil, err
	}

	defer func() { _ = tx.Commit() }()

	querySql := "SELECT `id`, `name`, `action`, `owner`, `comment`, `default`, `revision`, UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime) FROM auth_strategy "
	countSql := "SELECT COUNT(*) FROM auth_strategy"

	args := make([]interface{}, 0)
	if len(filters) != 0 {
		querySql += " WHERE "
		countSql += " WHERE "
		firstIndex := true
		for k, v := range filters {
			if !firstIndex {
				querySql += " AND "
				countSql += " AND "
			}
			firstIndex = false

			if utils.IsWildName(v) {
				querySql += (" " + k + " like ? ")
				countSql += (" " + k + " like ? ")
				args = append(args, v[:len(v)-1]+"%")
			} else {
				querySql += (" " + k + " = ? ")
				countSql += (" " + k + " = ? ")
				args = append(args, v)
			}

		}
	}

	log.GetAuthLogger().Debug("get simple strategies", zap.String("count sql", countSql), zap.Any("args", args))
	count, err := queryEntryCount(s.master, countSql, args)
	if err != nil {
		return 0, nil, store.Error(err)
	}

	querySql += " ORDER BY mtime LIMIT ?, ? "
	args = append(args, offset, limit)

	ret, err := s.collectStrategies(s.master.Query, querySql, args)
	if err != nil {
		return 0, nil, err
	}
	return count, ret, nil
}

// listStrategySimpleByUserId
func (s *strategyStore) listStrategySimpleByUserId(filters map[string]string, offset uint32,
	limit uint32) (uint32, []*model.StrategyDetail, error) {
	tx, err := s.slave.Begin()
	if err != nil {
		return 0, nil, err
	}

	defer func() { _ = tx.Commit() }()

	querySql := "SELECT `id`, `name`, `action`, `owner`, `comment`, `default`, `revision`, UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime) FROM auth_principal ap LEFT JOIN auth_strategy ag ON ap.strategy_id = ag.id "
	countSql := "SELECT COUNT(*) FROM  auth_principal ap LEFT JOIN auth_strategy ag ON ap.strategy_id = ag.id "

	args := make([]interface{}, 0)
	if len(filters) != 0 {
		querySql += " WHERE "
		countSql += " WHERE "
		firstIndex := true
		for k, v := range filters {
			if !firstIndex {
				querySql += " AND "
				countSql += " AND "
			}
			firstIndex = false

			if val, ok := RuleLinkUserFilters[k]; ok {
				k = val
			}

			querySql += (" " + k + " = ? ")
			countSql += (" " + k + " = ? ")
			args = append(args, v)
		}
	}

	log.GetAuthLogger().Debug("ListStrategySimpleByUserId", zap.String("count sql", countSql), zap.Any("args", args))
	count, err := queryEntryCount(s.master, countSql, args)
	if err != nil {
		return 0, nil, store.Error(err)
	}

	querySql += " ORDER BY ag.mtime LIMIT ?, ? "
	args = append(args, offset, limit)

	ret, err := s.collectStrategies(s.master.Query, querySql, args)
	if err != nil {
		return 0, nil, err
	}

	return count, ret, nil
}

func (s *strategyStore) collectStrategies(handler QueryHandler, querySql string, args []interface{}) ([]*model.StrategyDetail, error) {
	log.GetAuthLogger().Debug("get simple strategies", zap.String("query sql", querySql), zap.Any("args", args))
	rows, err := handler(querySql, args...)
	if err != nil {
		return nil, store.Error(err)
	}
	defer rows.Close()

	ret := make([]*model.StrategyDetail, 0, 16)
	for rows.Next() {
		detail, err := fetchRown2StrategyDetail(rows)
		if err != nil {
			return nil, store.Error(err)
		}
		ret = append(ret, detail)
	}

	return ret, nil

}

func (s *strategyStore) GetStrategyDetailsForCache(mtime time.Time, firstUpdate bool) ([]*model.StrategyDetail, error) {
	tx, err := s.slave.Begin()
	if err != nil {
		return nil, store.Error(err)
	}
	defer func() { _ = tx.Commit() }()

	args := make([]interface{}, 0)
	querySql := "SELECT id, name, action, owner, comment, `default`, `revision`, UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime) FROM auth_strategy"

	if !firstUpdate {
		querySql += " WHERE mtime >= ?"
		args = append(args, commontime.Time2String(mtime))
	}

	rows, err := tx.Query(querySql, args...)
	if err != nil {
		return nil, store.Error(err)
	}
	defer rows.Close()

	ret := make([]*model.StrategyDetail, 0)
	for rows.Next() {
		detail, err := fetchRown2StrategyDetail(rows)
		if err != nil {
			return nil, store.Error(err)
		}

		resArr, err := s.getStrategyResources(s.slave.Query, detail.ID)
		if err != nil {
			return nil, store.Error(err)
		}
		principals, err := s.getStrategyPrincipals(s.slave.Query, detail.ID)
		if err != nil {
			return nil, store.Error(err)
		}

		detail.Resources = resArr
		detail.Principals = principals

		ret = append(ret, detail)
	}

	return ret, nil
}

func (s *strategyStore) getStrategyPrincipals(queryHander QueryHandler, id string) ([]model.Principal, error) {
	rows, err := queryHander("SELECT principal_id, principal_role FROM auth_principal WHERE strategy_id = ?", id)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, nil
		default:
			return nil, store.Error(err)
		}
	}
	defer rows.Close()

	principals := make([]model.Principal, 0)

	for rows.Next() {
		res := new(model.Principal)
		if err := rows.Scan(&res.PrincipalID, &res.PrincipalRole); err != nil {
			return nil, store.Error(err)
		}
		principals = append(principals, *res)
	}

	return principals, nil
}

func (s *strategyStore) getStrategyResources(queryHander QueryHandler, id string) ([]model.StrategyResource, error) {
	rows, err := queryHander("SELECT res_id, res_type FROM auth_strategy_resource WHERE strategy_id = ? AND flag = 0", id)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, nil
		default:
			return nil, store.Error(err)
		}
	}
	defer rows.Close()

	resArr := make([]model.StrategyResource, 0)

	for rows.Next() {
		res := new(model.StrategyResource)
		if err := rows.Scan(&res.ResID, &res.ResType); err != nil {
			return nil, store.Error(err)
		}
		resArr = append(resArr, *res)
	}

	return resArr, nil
}

func fetchRown2StrategyDetail(rows *sql.Rows) (*model.StrategyDetail, error) {
	var (
		ctime, mtime int64
		isDefault    int16
	)
	ret := &model.StrategyDetail{
		Resources: make([]model.StrategyResource, 0),
	}
	if err := rows.Scan(&ret.ID, &ret.Name, &ret.Action, &ret.Owner, &ret.Comment, &isDefault, &ret.Revision, &ctime, &mtime); err != nil {
		return nil, store.Error(err)
	}

	ret.CreateTime = time.Unix(ctime, 0)
	ret.ModifyTime = time.Unix(mtime, 0)
	ret.Valid = true

	if isDefault == 1 {
		ret.Default = true
	}

	return ret, nil
}

func (s *strategyStore) cleanInvalidStrategy(name string) error {
	log.GetAuthLogger().Infof("[Store][database] clean invalid auth_strategy(%s)", name)

	tx, err := s.master.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	str := "delete from auth_strategy_resource where strategy_id = (select id from auth_strategy where  name = ? and flag = 1) and flag = 1"
	_, err = tx.Exec(str, name)
	if err != nil {
		log.GetAuthLogger().Errorf("[Store][database] clean invalid auth_strategy(%s) err: %s", name, err.Error())
		return err
	}

	str = "delete from auth_strategy where name = ? and flag = 1"
	_, err = tx.Exec(str, name)
	if err != nil {
		log.GetAuthLogger().Errorf("[Store][database] clean invalid auth_strategy(%s) err: %s", name, err.Error())
		return err
	}
	if err := tx.Commit(); err != nil {
		log.GetAuthLogger().Errorf("[Store][database] clean invalid auth_strategy tx commit err: %s", err.Error())
		return err
	}
	return nil
}
