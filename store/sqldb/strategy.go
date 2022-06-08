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

	logger "github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/store"
)

var (
	RuleFilters map[string]string = map[string]string{
		"res_id":         "ar.res_id",
		"res_type":       "ar.res_type",
		"default":        "ag.default",
		"owner":          "ag.owner",
		"name":           "ag.name",
		"principal_id":   "ap.principal_id",
		"principal_type": "ap.principal_role",
	}

	RuleNeedLikeFilters map[string]struct{} = map[string]struct{}{
		"name": {},
	}
)

type strategyStore struct {
	master *BaseDB
	slave  *BaseDB
}

func (s *strategyStore) AddStrategy(strategy *model.StrategyDetail) error {
	if strategy.ID == "" || strategy.Name == "" || strategy.Owner == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"add auth_strategy missing some params, id is %s, name is %s, owner is %s",
			strategy.ID, strategy.Name, strategy.Owner))
	}

	// 先清理无效数据
	if err := s.cleanInvalidStrategy(strategy.Name, strategy.Owner); err != nil {
		return store.Error(err)
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

	if err := s.addStrategyPrincipals(tx, strategy.ID, strategy.Principals); err != nil {
		logger.StoreScope().Error("[Store][Strategy] add auth_strategy principals", zap.Error(err))
		return err
	}

	if err := s.addStrategyResources(tx, strategy.ID, strategy.Resources); err != nil {
		logger.StoreScope().Error("[Store][Strategy] add auth_strategy resources", zap.Error(err))
		return err
	}

	// 保存策略主信息
	saveMainSql := "INSERT INTO auth_strategy(`id`, `name`, `action`, `owner`, `comment`, `flag`, " +
		" `default`, `revision`) VALUES (?,?,?,?,?,?,?,?)"
	if _, err = tx.Exec(saveMainSql,
		[]interface{}{
			strategy.ID, strategy.Name, strategy.Action, strategy.Owner, strategy.Comment,
			0, isDefault, strategy.Revision}...,
	); err != nil {
		logger.StoreScope().Error("[Store][Strategy] add auth_strategy main info", zap.Error(err))
		return err
	}

	if err := tx.Commit(); err != nil {
		logger.StoreScope().Errorf("[Store][Strategy] add auth_strategy tx commit err: %s", err.Error())
		return err
	}

	return nil
}

// UpdateStrategy 更新鉴权规则
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

	// 调整 principal 信息
	if err := s.addStrategyPrincipals(tx, strategy.ID, strategy.AddPrincipals); err != nil {
		logger.StoreScope().Errorf("[Store][Strategy] add strategy principal err: %s", err.Error())
		return err
	}
	if err := s.deleteStrategyPrincipals(tx, strategy.ID, strategy.RemovePrincipals); err != nil {
		logger.StoreScope().Errorf("[Store][Strategy] remove strategy principal err: %s", err.Error())
		return err
	}

	// 调整鉴权资源信息
	if err := s.addStrategyResources(tx, strategy.ID, strategy.AddResources); err != nil {
		logger.StoreScope().Errorf("[Store][Strategy] add strategy resource err: %s", err.Error())
		return err
	}
	if err := s.deleteStrategyResources(tx, strategy.ID, strategy.RemoveResources); err != nil {
		logger.StoreScope().Errorf("[Store][Strategy] remove strategy resource err: %s", err.Error())
		return err
	}

	// 保存策略主信息
	saveMainSql := "UPDATE auth_strategy SET action = ?, comment = ?, mtime = sysdate() WHERE id = ?"
	if _, err = tx.Exec(saveMainSql, []interface{}{strategy.Action, strategy.Comment, strategy.ID}...); err != nil {
		logger.StoreScope().Error("[Store][Strategy] update strategy main info", zap.Error(err))
		return err
	}

	if err := tx.Commit(); err != nil {
		logger.StoreScope().Errorf("[Store][Strategy] update auth_strategy tx commit err: %s", err.Error())
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

	if _, err = tx.Exec("UPDATE auth_strategy SET flag = 1, mtime = sysdate() WHERE id = ?", []interface{}{
		id,
	}...); err != nil {
		return err
	}

	if _, err = tx.Exec("DELETE FROM auth_strategy_resource WHERE strategy_id = ?", []interface{}{
		id,
	}...); err != nil {
		return err
	}

	if _, err = tx.Exec("DELETE FROM auth_principal WHERE strategy_id = ?", []interface{}{
		id,
	}...); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		logger.StoreScope().Errorf("[Store][Strategy] delete auth_strategy tx commit err: %s", err.Error())
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

	logger.StoreScope().Debug("[Store][Strategy] add strategy principal", zap.String("sql", savePrincipalSql),
		zap.Any("args", args))

	_, err := tx.Exec(savePrincipalSql, args...)
	return err
}

// deleteStrategyPrincipals
func (s *strategyStore) deleteStrategyPrincipals(tx *BaseTx, id string,
	principals []model.Principal) error {

	if len(principals) == 0 {
		return nil
	}

	savePrincipalSql := "DELETE FROM auth_principal WHERE strategy_id = ? AND principal_id = ? " +
		" AND principal_role = ?"
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

	saveResSql := "REPLACE INTO auth_strategy_resource(strategy_id, res_type, res_id) VALUES "

	values := make([]string, 0)
	args := make([]interface{}, 0)

	for i := range resources {
		resource := resources[i]
		values = append(values, "(?,?,?)")
		args = append(args, resource.StrategyID, resource.ResType, resource.ResID)
	}

	if len(values) == 0 {
		return nil
	}

	saveResSql += strings.Join(values, ",")

	logger.StoreScope().Debug("[Store][Strategy] add strategy resources", zap.String("sql", saveResSql),
		zap.Any("args", args))
	_, err := tx.Exec(saveResSql, args...)
	return err
}

func (s *strategyStore) deleteStrategyResources(tx *BaseTx, id string,
	resources []model.StrategyResource) error {

	if len(resources) == 0 {
		return nil
	}

	for i := range resources {
		resource := resources[i]

		saveResSql := "DELETE FROM auth_strategy_resource WHERE strategy_id = ? AND res_id = ? AND res_type = ?"
		if _, err := tx.Exec(
			saveResSql,
			[]interface{}{resource.StrategyID, resource.ResID, resource.ResType}...,
		); err != nil {
			return err
		}

	}
	return nil
}

// LooseAddStrategyResources
func (s *strategyStore) LooseAddStrategyResources(resources []model.StrategyResource) error {
	tx, err := s.master.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// 保存策略的资源信息
	for i := range resources {
		resource := resources[i]

		saveResSql := "REPLACE INTO auth_strategy_resource(strategy_id, res_type, res_id) VALUES (?,?,?)"
		args := make([]interface{}, 0)
		args = append(args, resource.StrategyID, resource.ResType, resource.ResID)

		if _, err = tx.Exec(saveResSql, args...); err != nil {
			err = store.Error(err)
			if store.Code(err) == store.DuplicateEntryErr {
				continue
			}
			return err
		}

		// 主要是为了能够触发 StrategyCache 的刷新逻辑
		updateStrategySql := "UPDATE auth_strategy SET mtime = sysdate() WHERE id = ?"
		if _, err = tx.Exec(updateStrategySql, resource.StrategyID); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		logger.StoreScope().Errorf("[Store][Strategy] add auth_strategy tx commit err: %s", err.Error())
		return err
	}

	return nil
}

// RemoveStrategyResources
func (s *strategyStore) RemoveStrategyResources(resources []model.StrategyResource) error {
	tx, err := s.master.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for i := range resources {
		resource := resources[i]

		args := make([]interface{}, 0)
		saveResSql := "DELETE FROM auth_strategy_resource WHERE strategy_id = ? AND res_id = ? AND res_type = ?"
		args = append(args, resource.StrategyID, resource.ResID, resource.ResType)
		if resource.StrategyID == "" {
			saveResSql = "DELETE FROM auth_strategy_resource WHERE res_id = ? AND res_type = ?"
			args = append(args, resource.ResID, resource.ResType)
		}
		if _, err := tx.Exec(saveResSql, args...); err != nil {
			return err
		}
		// 主要是为了能够触发 StrategyCache 的刷新逻辑
		updateStrategySql := "UPDATE auth_strategy SET mtime = sysdate() WHERE id = ?"
		if _, err = tx.Exec(updateStrategySql, resource.StrategyID); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		logger.StoreScope().Errorf("[Store][Strategy] add auth_strategy tx commit err: %s", err.Error())
		return err
	}

	return nil
}

// GetStrategyDetail
func (s *strategyStore) GetStrategyDetail(id string) (*model.StrategyDetail, error) {
	if id == "" {
		return nil, store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"get auth_strategy missing some params, id is %s", id))
	}

	querySql := "SELECT ag.id, ag.name, ag.action, ag.owner, ag.default, ag.comment, ag.revision, ag.flag, " +
		" UNIX_TIMESTAMP(ag.ctime), UNIX_TIMESTAMP(ag.mtime) FROM auth_strategy AS ag WHERE ag.flag = 0 AND ag.id = ?"

	row := s.master.QueryRow(querySql, id)

	return s.getStrategyDetail(row)
}

// GetDefaultStrategyDetailByPrincipal
func (s *strategyStore) GetDefaultStrategyDetailByPrincipal(principalId string,
	principalType model.PrincipalType) (*model.StrategyDetail, error) {

	if principalId == "" {
		return nil, store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"get auth_strategy missing some params, principal_id is %s", principalId))
	}

	querySql := `
	 SELECT ag.id, ag.name, ag.action, ag.owner, ag.default
		 , ag.comment, ag.revision, ag.flag, UNIX_TIMESTAMP(ag.ctime)
		 , UNIX_TIMESTAMP(ag.mtime)
	 FROM auth_strategy ag
	 WHERE ag.flag = 0
		 AND ag.default = 1
		 AND ag.id IN (
			 SELECT DISTINCT strategy_id
			 FROM auth_principal
			 WHERE principal_id = ?
				 AND principal_role = ?
		 )
	 `

	row := s.master.QueryRow(querySql, principalId, int(principalType))

	return s.getStrategyDetail(row)
}

// getStrategyDetail
func (s *strategyStore) getStrategyDetail(row *sql.Row) (*model.StrategyDetail, error) {
	var (
		ctime, mtime    int64
		isDefault, flag int16
	)
	ret := new(model.StrategyDetail)
	if err := row.Scan(&ret.ID, &ret.Name, &ret.Action, &ret.Owner, &isDefault, &ret.Comment,
		&ret.Revision, &flag, &ctime, &mtime); err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, nil
		default:
			return nil, store.Error(err)
		}
	}

	ret.CreateTime = time.Unix(ctime, 0)
	ret.ModifyTime = time.Unix(mtime, 0)
	ret.Valid = flag == 0
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

// GetStrategies
func (s *strategyStore) GetStrategies(filters map[string]string, offset uint32, limit uint32) (uint32,
	[]*model.StrategyDetail, error) {

	showDetail := filters["show_detail"]
	delete(filters, "show_detail")

	filters["ag.flag"] = "0"

	return s.listStrategies(filters, offset, limit, showDetail == "true")

}

// listStrategies
func (s *strategyStore) listStrategies(filters map[string]string, offset uint32, limit uint32,
	showDetail bool) (uint32, []*model.StrategyDetail, error) {

	querySql :=
		`SELECT
			 ag.id,
			 ag.name,
			 ag.action,
			 ag.owner,
			 ag.comment,
			 ag.default,
			 ag.revision,
			 ag.flag,
			 UNIX_TIMESTAMP(ag.ctime),
			 UNIX_TIMESTAMP(ag.mtime)
		   FROM
			 (
			   auth_strategy ag
			   LEFT JOIN auth_strategy_resource ar ON ag.id = ar.strategy_id
			 )
			 LEFT JOIN auth_principal ap ON ag.id = ap.strategy_id `
	countSql := `
	 SELECT COUNT(DISTINCT ag.id)
	 FROM
	   (
		 auth_strategy ag
		 LEFT JOIN auth_strategy_resource ar ON ag.id = ar.strategy_id
	   )
	   LEFT JOIN auth_principal ap ON ag.id = ap.strategy_id
	 `

	return s.queryStrategies(s.master.Query, filters, RuleFilters, querySql, countSql,
		offset, limit, showDetail)
}

// queryStrategies 通用的查询策略列表
func (s *strategyStore) queryStrategies(
	handler QueryHandler,
	filters map[string]string, mapping map[string]string,
	querySqlPrefix string, countSqlPrefix string,
	offset uint32, limit uint32, showDetail bool) (uint32, []*model.StrategyDetail, error) {

	querySql := querySqlPrefix
	countSql := countSqlPrefix

	args := make([]interface{}, 0)
	if len(filters) != 0 {
		querySql += " WHERE "
		countSql += " WHERE "
		firstIndex := true
		for k, v := range filters {
			needLike := false
			if !firstIndex {
				querySql += " AND "
				countSql += " AND "
			}
			firstIndex = false

			if val, ok := mapping[k]; ok {
				if _, exist := RuleNeedLikeFilters[k]; exist {
					needLike = true
				}
				k = val
			}

			if needLike {
				if utils.IsWildName(v) {
					v = v[:len(v)-1]
				}
				querySql += (" " + k + " like ? ")
				countSql += (" " + k + " like ? ")
				args = append(args, "%"+v+"%")
			} else if k == "ag.owner" {
				querySql += " (ag.owner = ? OR (ap.principal_id = ? AND ap.principal_role = 1 )) "
				countSql += " (ag.owner = ? OR (ap.principal_id = ? AND ap.principal_role = 1 )) "
				args = append(args, v, v)
			} else {
				querySql += (" " + k + " = ? ")
				countSql += (" " + k + " = ? ")
				args = append(args, v)
			}
		}
	}

	count, err := queryEntryCount(s.master, countSql, args)
	if err != nil {
		return 0, nil, store.Error(err)
	}

	querySql += " GROUP BY ag.id ORDER BY ag.mtime LIMIT ?, ? "
	args = append(args, offset, limit)

	ret, err := s.collectStrategies(s.master.Query, querySql, args, showDetail)
	if err != nil {
		return 0, nil, err
	}

	return count, ret, nil
}

// collectStrategies 执行真正的 sql 并从 rows 中获取策略列表
func (s *strategyStore) collectStrategies(handler QueryHandler, querySql string,
	args []interface{}, showDetail bool) ([]*model.StrategyDetail, error) {

	logger.StoreScope().Debug("[Store][Strategy] get simple strategies", zap.String("query sql", querySql),
		zap.Any("args", args))

	rows, err := handler(querySql, args...)
	if err != nil {
		logger.StoreScope().Error("[Store][Strategy] get simple strategies", zap.String("query sql", querySql),
			zap.Any("args", args))
		return nil, store.Error(err)
	}
	defer rows.Close()

	idMap := make(map[string]struct{})

	ret := make([]*model.StrategyDetail, 0, 16)
	for rows.Next() {
		detail, err := fetchRown2StrategyDetail(rows)
		if err != nil {
			return nil, store.Error(err)
		}

		// 为了避免数据重复被加入到 slice 中，做一个 map 去重
		if _, ok := idMap[detail.ID]; ok {
			continue
		}
		idMap[detail.ID] = struct{}{}

		if showDetail {
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
		}

		ret = append(ret, detail)
	}

	return ret, nil

}

func (s *strategyStore) GetStrategyDetailsForCache(mtime time.Time,
	firstUpdate bool) ([]*model.StrategyDetail, error) {

	tx, err := s.slave.Begin()
	if err != nil {
		return nil, store.Error(err)
	}
	defer func() { _ = tx.Commit() }()

	args := make([]interface{}, 0)
	querySql := "SELECT ag.id, ag.name, ag.action, ag.owner, ag.comment, ag.default, ag.revision, ag.flag, " +
		" UNIX_TIMESTAMP(ag.ctime), UNIX_TIMESTAMP(ag.mtime) FROM auth_strategy ag "

	if !firstUpdate {
		querySql += " WHERE ag.mtime >= FROM_UNIXTIME(?)"
		args = append(args, timeToTimestamp(mtime))
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

// GetStrategyResources 获取对应 principal 能操作的所有资源
func (s *strategyStore) GetStrategyResources(principalId string,
	principalRole model.PrincipalType) ([]model.StrategyResource, error) {

	querySql := "SELECT res_id, res_type FROM auth_strategy_resource WHERE strategy_id IN (SELECT DISTINCT " +
		" ap.strategy_id FROM auth_principal ap join auth_strategy ar ON ap.strategy_id = ar.id WHERE ar.flag = 0 " +
		" AND ap.principal_id = ? AND ap.principal_role = ? )"

	rows, err := s.master.Query(querySql, principalId, principalRole)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, nil
		default:
			logger.StoreScope().Error("[Store][Strategy] get principal link resource", zap.String("sql", querySql),
				zap.String("principal-id", principalId), zap.Any("principal-type", principalRole))
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

func (s *strategyStore) getStrategyPrincipals(queryHander QueryHandler, id string) ([]model.Principal, error) {

	rows, err := queryHander("SELECT principal_id, principal_role FROM auth_principal WHERE strategy_id = ?", id)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			logger.StoreScope().Info("[Store][Strategy] not found link principals", zap.String("strategy-id", id))
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
	querySql := "SELECT res_id, res_type FROM auth_strategy_resource WHERE strategy_id = ?"
	rows, err := queryHander(querySql, id)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			logger.StoreScope().Info("[Store][Strategy] not found link resources", zap.String("strategy-id", id))
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
		ctime, mtime    int64
		isDefault, flag int16
	)
	ret := &model.StrategyDetail{
		Resources: make([]model.StrategyResource, 0),
	}

	if err := rows.Scan(&ret.ID, &ret.Name, &ret.Action, &ret.Owner, &ret.Comment, &isDefault, &ret.Revision, &flag,
		&ctime, &mtime); err != nil {
		return nil, store.Error(err)
	}

	ret.CreateTime = time.Unix(ctime, 0)
	ret.ModifyTime = time.Unix(mtime, 0)
	ret.Valid = flag == 0

	if isDefault == 1 {
		ret.Default = true
	}

	return ret, nil
}

// cleanInvalidStrategy 按名称清理鉴权策略
func (s *strategyStore) cleanInvalidStrategy(name, owner string) error {
	logger.StoreScope().Info("[Store][Strategy] clean invalid auth_strategy",
		zap.String("name", name), zap.String("owner", owner))

	tx, err := s.master.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	str := "delete from auth_strategy where name = ? and owner = ? and flag = 1"
	if _, err = tx.Exec(str, name, owner); err != nil {
		logger.StoreScope().Errorf("[Store][Strategy] clean invalid auth_strategy(%s) err: %s", name, err.Error())
		return err
	}
	if err := tx.Commit(); err != nil {
		logger.StoreScope().Errorf("[Store][Strategy] clean invalid auth_strategy tx commit err: %s", err.Error())
		return err
	}
	return nil
}

// cleanLinkStrategy 清理与自己相关联的鉴权信息
// step 1. 清理用户/用户组默认策略所关联的所有资源信息（直接走delete删除）
// step 2. 清理用户/用户组默认策略
// step 3. 清理用户/用户组所关联的其他鉴权策略的关联关系（直接走delete删除）
func cleanLinkStrategy(tx *BaseTx, role model.PrincipalType, principalId, owner string) error {

	// 清理默认策略对应的所有鉴权关联资源
	removeResSql := `
		 DELETE FROM auth_strategy_resource
		 WHERE strategy_id IN (
				 SELECT DISTINCT ag.id
				 FROM auth_strategy ag
				 WHERE ag.default = 1
					 AND ag.owner = ?
					 AND ag.id IN (
						 SELECT DISTINCT strategy_id
						 FROM auth_principal
						 WHERE principal_id = ?
							 AND principal_role = ?
					 )
			 )
		 `

	if _, err := tx.Exec(removeResSql, []interface{}{owner, principalId, role}...); err != nil {
		return err
	}

	// 清理默认策略
	cleanaRuleSql := `
		 UPDATE auth_strategy AS ag
		 SET ag.flag = 1
		 WHERE ag.id IN (
				 SELECT DISTINCT strategy_id
				 FROM auth_principal
				 WHERE principal_id = ?
					 AND principal_role = ?
			 )
			 AND ag.default = 1
			 AND ag.owner = ?
	 `

	if _, err := tx.Exec(cleanaRuleSql, []interface{}{principalId, role, owner}...); err != nil {
		return err
	}

	// 调整所关联的鉴权策略的 mtime 数据，保证cache刷新可以获取到变更的数据信息
	updateStrategySql := "UPDATE auth_strategy SET mtime = sysdate()  WHERE id IN (SELECT DISTINCT " +
		" strategy_id FROM auth_principal WHERE principal_id = ? AND principal_role = ?)"
	if _, err := tx.Exec(updateStrategySql, []interface{}{principalId, role}...); err != nil {
		return err
	}

	// 清理所在的所有鉴权principal
	cleanPrincipalSql := "DELETE FROM auth_principal WHERE principal_id = ? AND principal_role = ?"
	if _, err := tx.Exec(cleanPrincipalSql, []interface{}{principalId, role}...); err != nil {
		return err
	}

	return nil
}
