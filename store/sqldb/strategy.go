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
	"github.com/polarismesh/polaris-server/store"
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
	saveMainSql := "INSERT INTO auth_strategy(`id`, `name`, `principal`, `action`, `owner`, `comment`, `flag`, `default`) VALUES (?,?,?,?,?,?,?,?)"
	_, err = tx.Exec(saveMainSql, []interface{}{strategy.ID, strategy.Name, strategy.Principal, strategy.Action, strategy.Owner, strategy.Comment, 0, isDefault}...)

	if err != nil {
		return err
	}

	saveResSql := "INSERT INTO auth_strategy_resource(strategy_id, res_type, res_id) VALUES "
	// 保存策略的资源信息
	resources := strategy.Resources

	values := make([]string, 0)
	args := make([]interface{}, 0)

	for i := range resources {
		resource := resources[i]
		values = append(values, "(?,?,?)")
		args = append(args, strategy.ID, resource.ResType, resource.ResID)
	}

	saveResSql += strings.Join(values, ",")

	_, err = tx.Exec(saveResSql, args...)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Errorf("[Store][database] add auth_strategy tx commit err: %s", err.Error())
		return err
	}

	return nil
}

func (s *strategyStore) UpdateStrategyMain(strategy *model.StrategyDetail) error {
	if strategy.ID == "" || strategy.Name == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"update auth_strategy missing some params, id is %s, name is %s, ", strategy.ID, strategy.Name))
	}

	err := RetryTransaction("updateStrategy", func() error {
		return s.updateStrategyMain(strategy)
	})
	return store.Error(err)
}

func (s *strategyStore) updateStrategyMain(strategy *model.StrategyDetail) error {

	tx, err := s.master.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// 保存策略主信息
	saveMainSql := "UPDATE auth_strategy SET principal = ?, action = ?, comment = ? WHERE id = ?"
	_, err = tx.Exec(saveMainSql, []interface{}{strategy.Principal, strategy.Action, strategy.Comment, strategy.ID}...)

	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Errorf("[Store][database] update auth_strategy tx commit err: %s", err.Error())
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

	delSql := "UPDATE auth_strategy SET flag = 1 WHERE id = ?"

	_, err = tx.Exec(delSql, []interface{}{
		id,
	}...)

	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		log.Errorf("[Store][database] delete auth_strategy tx commit err: %s", err.Error())
		return err
	}
	return nil
}

func (s *strategyStore) AddStrategyResources(resources []*model.StrategyResource) error {
	tx, err := s.master.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	saveResSql := "INSERT INTO auth_strategy_resource(strategy_id, res_type, res_id) VALUES "
	// 保存策略的资源信息

	values := make([]string, 0)
	args := make([]interface{}, 0)

	for i := range resources {
		resource := resources[i]
		values = append(values, "(?,?,?)")
		args = append(args, resource.ResID, resource.ResType, resource.ResID)
	}

	saveResSql += strings.Join(values, ",")

	_, err = tx.Exec(saveResSql, args...)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Errorf("[Store][database] add auth_strategy tx commit err: %s", err.Error())
		return err
	}

	return nil
}

func (s *strategyStore) DeleteStrategyResources(resources []*model.StrategyResource) error {
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
		_, err = tx.Exec(saveResSql, args...)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		log.Errorf("[Store][database] add auth_strategy tx commit err: %s", err.Error())
		return err
	}

	return nil
}

// LooseAddStrategyResources
//  @param resources
//  @return error
func (s *strategyStore) LooseAddStrategyResources(resources []*model.StrategyResource) error {
	tx, err := s.master.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// 保存策略的资源信息
	for i := range resources {
		saveResSql := "INSERT INTO auth_strategy_resource(strategy_id, res_type, res_id) VALUES (?,?,?)"
		args := make([]interface{}, 0)
		resource := resources[i]
		args = append(args, resource.ResID, resource.ResType, resource.ResID)

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
		log.Errorf("[Store][database] add auth_strategy tx commit err: %s", err.Error())
		return err
	}

	return nil
}

func (s *strategyStore) GetStrategyDetail(id string) (*model.StrategyDetail, error) {

	return s.getStrategyDetailByIDOrName(id, "")
}

func (s *strategyStore) GetStrategyDetailByName(name string) (*model.StrategyDetail, error) {

	return s.getStrategyDetailByIDOrName("", name)
}

func (s *strategyStore) getStrategyDetailByIDOrName(id, name string) (*model.StrategyDetail, error) {
	if id == "" && name == "" {
		return nil, store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"get auth_strategy missing some params, id is %s, name is %s", id, name))
	}

	tx, err := s.slave.Begin()
	if err != nil {
		return nil, err
	}

	arg := id
	querySql := "SELECT id, name, principal, action, owner, default, comment, UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime) FROM auth_strategy WHERE flag = 0 AND id = ?"
	if id == "" {
		querySql = "SELECT id, name, principal, action, owner, default, comment, UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime) FROM auth_strategy WHERE flag = 0 AND name = ?"
		arg = name
	}

	row := tx.QueryRow(querySql, arg)

	var (
		ctime, mtime int64
		isDefault    int16
	)
	ret := new(model.StrategyDetail)
	if err := row.Scan(&ret.ID, &ret.Name, &ret.Principal, &ret.Action, &ret.Owner, &isDefault, &ret.Comment, &ctime, &mtime); err != nil {
		return nil, store.Error(err)
	}

	ret.CreateTime = time.Unix(ctime, 0)
	ret.ModifyTime = time.Unix(mtime, 0)
	ret.Valid = true

	if isDefault == 1 {
		ret.Default = true
	}

	// query all link resource
	queryResSql := "SELECT res_type, res_id, UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime) FROM auth_strategy_resource WHERE flag = 0 AND strategy_id = ?"
	rows, err := tx.Query(queryResSql, ret.ID)

	resources := make([]model.StrategyResource, 0)
	for rows.Next() {
		res := &model.StrategyResource{StrategyID: ret.ID, Valid: true}
		if err := rows.Scan(&res.ResType, &res.ResID, &ctime, &mtime); err != nil {
			return nil, store.Error(err)
		}
		resources = append(resources, *res)
	}

	ret.Resources = resources
	return ret, nil
}

func (s *strategyStore) ListStrategyDetails(filters map[string]string, offset uint32, limit uint32) (uint32, []*model.StrategyDetail, error) {

	args := make([]interface{}, 0)

	tx, err := s.slave.Begin()
	if err != nil {
		return 0, nil, err
	}

	defer func() { _ = tx.Commit() }()

	querySql := "SELECT id, name, principal, action, owner, comment, default, UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime) FROM auth_strategy"
	countSql := "SELECT COUNT(*) FROM auth_strategy"

	if len(filters) != 0 {
		querySql += " WHERE "
		countSql += " WHERE "
		firstIndex := true
		for k, v := range filters {
			if !firstIndex {
				querySql += " AND "
				countSql += " AND "
			}

			querySql += (" " + k + " = ? ")
			countSql += (" " + k + " = ? ")
			args = append(args, v)
		}

		args = append(args)
	}

	querySql += " ORDER BY mtime LIMIT ?, ? "
	args = append(args, offset, limit)

	count, err := queryEntryCount(s.master, countSql, args)
	if err != nil {
		return 0, nil, store.Error(err)
	}

	rows, err := tx.Query(querySql, args)
	if err != nil {
		return 0, nil, store.Error(err)
	}

	ret := make([]*model.StrategyDetail, 0, 16)
	for rows.Next() {
		detail, err := fetchRown2StrategyDetail(rows)
		if err != nil {
			return 0, nil, store.Error(err)
		}
		pullAllRes := "SELECT res_id, res_type FROM auth_strategy_resource WHERE strategy_id = ? AND flag = 0"

		resRows, err := tx.Query(pullAllRes, detail.ID)

		for resRows.Next() {
			res := new(model.StrategyResource)
			if err := resRows.Scan(&res.ResID, &res.ResType); err != nil {
				return 0, nil, store.Error(err)
			}
			detail.Resources = append(detail.Resources, *res)
		}

		ret = append(ret, detail)
	}

	return count, ret, nil
}

func (s *strategyStore) GetStrategyDetailsForCache(mtime time.Time, firstUpdate bool) ([]*model.StrategyDetail, error) {

	args := make([]interface{}, 0)

	tx, err := s.slave.Begin()
	if err != nil {
		return nil, err
	}

	defer func() { _ = tx.Commit() }()

	querySql := "SELECT id, name, principal, action, owner, comment, default, UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime) FROM auth_strategy"
	if !firstUpdate {
		querySql += " WHERE mtime >= ?"
		args = append(args, commontime.Time2String(mtime))
	}

	rows, err := tx.Query(querySql, args)
	if err != nil {
		return nil, store.Error(err)
	}

	ret := make([]*model.StrategyDetail, 0)
	for rows.Next() {
		detail, err := fetchRown2StrategyDetail(rows)
		if err != nil {
			return nil, store.Error(err)
		}
		pullAllRes := "SELECT res_id, res_type FROM auth_strategy_resource WHERE strategy_id = ? AND flag = 0"

		resRows, err := tx.Query(pullAllRes, detail.ID)

		for resRows.Next() {
			res := new(model.StrategyResource)
			if err := resRows.Scan(&res.ResID, &res.ResType); err != nil {
				return nil, store.Error(err)
			}
			detail.Resources = append(detail.Resources, *res)
		}

		ret = append(ret, detail)
	}

	return ret, nil
}

func fetchRown2StrategyDetail(rows *sql.Rows) (*model.StrategyDetail, error) {
	var (
		ctime, mtime int64
		isDefault    int16
	)
	ret := &model.StrategyDetail{
		Resources: make([]model.StrategyResource, 0),
	}
	if err := rows.Scan(&ret.ID, &ret.Name, &ret.Principal, &ret.Action, &ret.Owner, &ret.Comment, &isDefault, &ctime, &mtime); err != nil {
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
	log.Infof("[Store][database] clean invalid auth_strategy(%s)", name)

	tx, err := s.master.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	str := "delete from auth_strategy_resource where strategy_id = (select id from auth_strategy where  name = ? and flag = 1) and flag = 1"
	_, err = tx.Exec(str, name)
	if err != nil {
		log.Errorf("[Store][database] clean invalid auth_strategy(%s) err: %s", name, err.Error())
		return err
	}

	str = "delete from auth_strategy where name = ? and flag = 1"
	_, err = tx.Exec(str, name)
	if err != nil {
		log.Errorf("[Store][database] clean invalid auth_strategy(%s) err: %s", name, err.Error())
		return err
	}
	if err := tx.Commit(); err != nil {
		log.Errorf("[Store][database] clean invalid auth_strategy tx commit err: %s", err.Error())
		return err
	}
	return nil
}
