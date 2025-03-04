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
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
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

func (s *strategyStore) AddStrategy(tx store.Tx, strategy *authcommon.StrategyDetail) error {
	if strategy.ID == "" || strategy.Name == "" || strategy.Owner == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"add auth_strategy missing some params, id is %s, name is %s, owner is %s",
			strategy.ID, strategy.Name, strategy.Owner))
	}

	dbTx := tx.GetDelegateTx().(*BaseTx)

	// 先清理无效数据
	log.Info("[Store][Strategy] clean invalid auth_strategy", zap.String("name", strategy.Name),
		zap.String("owner", strategy.Owner))

	str := "delete from auth_strategy where name = ? and owner = ? and flag = 1"
	if _, err := dbTx.Exec(str, strategy.Name, strategy.Owner); err != nil {
		log.Errorf("[Store][Strategy] clean invalid auth_strategy(%s) err: %s", strategy.Name, err.Error())
		return err
	}

	isDefault := 0
	if strategy.Default {
		isDefault = 1
	}

	if err := s.addPolicyPrincipals(dbTx, strategy.ID, strategy.Principals); err != nil {
		log.Error("[Store][Strategy] add auth_strategy principals", zap.Error(err))
		return err
	}
	if err := s.addPolicyResources(dbTx, strategy.ID, strategy.Resources); err != nil {
		log.Error("[Store][Strategy] add auth_strategy resources", zap.Error(err))
		return err
	}
	if err := s.savePolicyFunctions(dbTx, strategy.ID, strategy.CalleeMethods); err != nil {
		log.Error("[Store][Strategy] save auth_strategy functions", zap.Error(err))
		return err
	}
	if err := s.savePolicyConditions(dbTx, strategy.ID, strategy.Conditions); err != nil {
		log.Error("[Store][Strategy] save auth_strategy conditions", zap.Error(err))
		return err
	}

	// 保存策略主信息
	saveMainSql := "INSERT INTO auth_strategy(`id`, `name`, `action`, `owner`, `comment`, `flag`, " +
		" `default`, `revision`, `source`, `metadata`) VALUES (?,?,?,?,?,?,?,?,?,?)"
	if _, err := dbTx.Exec(saveMainSql,
		[]interface{}{
			strategy.ID, strategy.Name, strategy.Action, strategy.Owner, strategy.Comment,
			0, isDefault, strategy.Revision, strategy.Source, utils.MustJson(strategy.Metadata)}...,
	); err != nil {
		log.Error("[Store][Strategy] add auth_strategy main info", zap.Error(err))
		return err
	}
	return nil
}

// UpdateStrategy 更新鉴权规则
func (s *strategyStore) UpdateStrategy(strategy *authcommon.ModifyStrategyDetail) error {
	if strategy.ID == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"update auth_strategy missing some params, id is %s", strategy.ID))
	}

	err := RetryTransaction("updateStrategy", func() error {
		return s.updateStrategy(strategy)
	})
	return store.Error(err)
}

func (s *strategyStore) updateStrategy(strategy *authcommon.ModifyStrategyDetail) error {
	tx, err := s.master.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// 调整 principal 信息
	if err = s.addPolicyPrincipals(tx, strategy.ID, strategy.AddPrincipals); err != nil {
		log.Errorf("[Store][Strategy] add strategy principal err: %s", err.Error())
		return err
	}
	if err = s.deletePolicyPrincipals(tx, strategy.ID, strategy.RemovePrincipals); err != nil {
		log.Errorf("[Store][Strategy] remove strategy principal err: %s", err.Error())
		return err
	}

	// 调整鉴权资源信息
	if err = s.addPolicyResources(tx, strategy.ID, strategy.AddResources); err != nil {
		log.Errorf("[Store][Strategy] add strategy resource err: %s", err.Error())
		return err
	}
	if err = s.deletePolicyResources(tx, strategy.ID, strategy.RemoveResources); err != nil {
		log.Errorf("[Store][Strategy] remove strategy resource err: %s", err.Error())
		return err
	}

	if err = s.savePolicyFunctions(tx, strategy.ID, strategy.CalleeMethods); err != nil {
		log.Error("[Store][Strategy] save auth_strategy functions", zap.Error(err))
		return err
	}
	if err = s.savePolicyConditions(tx, strategy.ID, strategy.Conditions); err != nil {
		log.Error("[Store][Strategy] save auth_strategy conditions", zap.Error(err))
		return err
	}

	// 保存策略主信息
	saveMainSql := "UPDATE auth_strategy SET action = ?, comment = ?, mtime = sysdate() WHERE id = ?"
	if _, err = tx.Exec(saveMainSql, []interface{}{strategy.Action, strategy.Comment, strategy.ID}...); err != nil {
		log.Error("[Store][Strategy] update strategy main info", zap.Error(err))
		return err
	}

	if err = tx.Commit(); err != nil {
		log.Errorf("[Store][Strategy] update auth_strategy tx commit err: %s", err.Error())
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

	if _, err = tx.Exec("UPDATE auth_strategy SET flag = 1, mtime = sysdate() WHERE id = ?", id); err != nil {
		return err
	}
	if _, err = tx.Exec("DELETE FROM auth_strategy_resource WHERE strategy_id = ?", id); err != nil {
		return err
	}
	if _, err = tx.Exec("DELETE FROM auth_principal WHERE strategy_id = ?", id); err != nil {
		return err
	}
	if _, err = tx.Exec("DELETE FROM auth_strategy_function WHERE strategy_id = ?", id); err != nil {
		return err
	}
	if _, err = tx.Exec("DELETE FROM auth_strategy_label WHERE strategy_id = ?", id); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Errorf("[Store][Strategy] delete auth_strategy tx commit err: %s", err.Error())
		return err
	}
	return nil
}

// savePolicyFunctions
func (s *strategyStore) savePolicyFunctions(tx *BaseTx, id string, functions []string) error {
	if len(functions) == 0 {
		return nil
	}

	if _, err := tx.Exec("DELETE FROM auth_strategy_function WHERE strategy_id = ?", id); err != nil {
		return err
	}

	savePrincipalSql := "INSERT IGNORE INTO auth_strategy_function(`strategy_id`, `function`) VALUES "
	values := make([]string, 0)
	args := make([]interface{}, 0)

	for i := range functions {
		values = append(values, "(?,?)")
		args = append(args, id, functions[i])
	}

	savePrincipalSql += strings.Join(values, ",")

	log.Debug("[Store][Strategy] save policy functions", zap.String("sql", savePrincipalSql),
		zap.Any("args", args))

	_, err := tx.Exec(savePrincipalSql, args...)
	return err
}

// savePolicyConditions
func (s *strategyStore) savePolicyConditions(tx *BaseTx, id string, conditions []authcommon.Condition) error {
	if len(conditions) == 0 {
		return nil
	}

	if _, err := tx.Exec("DELETE FROM auth_strategy_label WHERE strategy_id = ?", id); err != nil {
		return err
	}

	savePrincipalSql := "INSERT IGNORE INTO auth_strategy_label(`strategy_id`, `key`, `value`, `compare_type`) VALUES "
	values := make([]string, 0)
	args := make([]interface{}, 0)

	for i := range conditions {
		item := conditions[i]
		values = append(values, "(?,?,?,?)")
		args = append(args, id, item.Key, item.Value, item.CompareFunc)
	}

	savePrincipalSql += strings.Join(values, ",")

	log.Debug("[Store][Strategy] save policy conditions", zap.String("sql", savePrincipalSql),
		zap.Any("args", args))

	_, err := tx.Exec(savePrincipalSql, args...)
	return err
}

// addPolicyPrincipals
func (s *strategyStore) addPolicyPrincipals(tx *BaseTx, id string, principals []authcommon.Principal) error {
	if len(principals) == 0 {
		return nil
	}

	savePrincipalSql := "INSERT IGNORE INTO auth_principal(strategy_id, principal_id, principal_role, IFNULL(extend_info, '')) VALUES "
	values := make([]string, 0)
	args := make([]interface{}, 0)

	for i := range principals {
		principal := principals[i]
		values = append(values, "(?,?,?,?)")
		args = append(args, id, principal.PrincipalID, principal.PrincipalType, utils.MustJson(principal.Extend))
	}

	savePrincipalSql += strings.Join(values, ",")

	log.Debug("[Store][Strategy] add strategy principal", zap.String("sql", savePrincipalSql),
		zap.Any("args", args))

	_, err := tx.Exec(savePrincipalSql, args...)
	return err
}

// deletePolicyPrincipals
func (s *strategyStore) deletePolicyPrincipals(tx *BaseTx, id string,
	principals []authcommon.Principal) error {
	if len(principals) == 0 {
		return nil
	}

	savePrincipalSql := "DELETE FROM auth_principal WHERE strategy_id = ? AND principal_id = ? " +
		" AND principal_role = ?"
	for i := range principals {
		principal := principals[i]
		if _, err := tx.Exec(savePrincipalSql, []interface{}{
			id, principal.PrincipalID, principal.PrincipalType,
		}...); err != nil {
			return err
		}
	}

	return nil
}

// addPolicyResources .
func (s *strategyStore) addPolicyResources(tx *BaseTx, id string, resources []authcommon.StrategyResource) error {
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
	log.Debug("[Store][Strategy] add strategy resources", zap.String("sql", saveResSql),
		zap.Any("args", args))
	_, err := tx.Exec(saveResSql, args...)
	return err
}

// deletePolicyResources .
func (s *strategyStore) deletePolicyResources(tx *BaseTx, id string,
	resources []authcommon.StrategyResource) error {

	if len(resources) == 0 {
		return nil
	}

	for i := range resources {
		resource := resources[i]
		saveResSql := "DELETE FROM auth_strategy_resource WHERE strategy_id = ? AND res_id = ? AND res_type = ?"
		if _, err := tx.Exec(
			saveResSql, []interface{}{resource.StrategyID, resource.ResID, resource.ResType}...,
		); err != nil {
			return err
		}
	}
	return nil
}

// LooseAddStrategyResources loose add strategy resources
func (s *strategyStore) LooseAddStrategyResources(resources []authcommon.StrategyResource) error {
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
		log.Errorf("[Store][Strategy] add auth_strategy tx commit err: %s", err.Error())
		return err
	}

	return nil
}

// RemoveStrategyResources 删除策略的资源
func (s *strategyStore) RemoveStrategyResources(resources []authcommon.StrategyResource) error {
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
		if _, err = tx.Exec(saveResSql, args...); err != nil {
			return err
		}
		// 主要是为了能够触发 StrategyCache 的刷新逻辑
		updateStrategySql := "UPDATE auth_strategy SET mtime = sysdate() WHERE id = ?"
		if _, err = tx.Exec(updateStrategySql, resource.StrategyID); err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		log.Errorf("[Store][Strategy] add auth_strategy tx commit err: %s", err.Error())
		return err
	}

	return nil
}

// GetStrategyDetail 获取策略详情
func (s *strategyStore) GetStrategyDetail(id string) (*authcommon.StrategyDetail, error) {
	if id == "" {
		return nil, store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"get auth_strategy missing some params, id is %s", id))
	}

	querySql := "SELECT ag.id, ag.name, ag.action, ag.owner, ag.default, ag.comment, ag.revision, ag.flag, " +
		" UNIX_TIMESTAMP(ag.ctime), UNIX_TIMESTAMP(ag.mtime) FROM auth_strategy AS ag WHERE ag.flag = 0 AND ag.id = ?"

	row := s.master.QueryRow(querySql, id)

	return s.getStrategyDetail(row)
}

// GetDefaultStrategyDetailByPrincipal 获取默认策略
func (s *strategyStore) GetDefaultStrategyDetailByPrincipal(principalId string,
	principalType authcommon.PrincipalType) (*authcommon.StrategyDetail, error) {

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

	rows, err := s.master.Query(querySql, principalId, int(principalType))
	if err != nil {
		return nil, store.Error(err)
	}
	defer rows.Close()

	for rows.Next() {
		detail, err := fetchRown2StrategyDetail(rows)
		if err != nil {
			return nil, store.Error(err)
		}
		if detail.Metadata[authcommon.MetadKeySystemDefaultPolicy] == "true" {
			continue
		}
		return detail, nil
	}
	return nil, nil
}

// getStrategyDetail
func (s *strategyStore) getStrategyDetail(row *sql.Row) (*authcommon.StrategyDetail, error) {
	var (
		ctime, mtime    int64
		isDefault, flag int16
	)
	ret := new(authcommon.StrategyDetail)
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

func (s *strategyStore) GetMoreStrategies(mtime time.Time, firstUpdate bool) ([]*authcommon.StrategyDetail, error) {
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
	defer func() {
		_ = rows.Close()
	}()

	ret := make([]*authcommon.StrategyDetail, 0)
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
		conditions, err := s.getStrategyConditions(s.slave.Query, detail.ID)
		if err != nil {
			return nil, store.Error(err)
		}
		functions, err := s.getStrategyFunctions(s.slave.Query, detail.ID)
		if err != nil {
			return nil, store.Error(err)
		}

		detail.Resources = resArr
		detail.Principals = principals
		detail.CalleeMethods = functions
		detail.Conditions = conditions

		ret = append(ret, detail)
	}

	return ret, nil
}

// GetStrategyResources 获取对应 principal 能操作的所有资源
func (s *strategyStore) GetStrategyResources(principalId string,
	principalRole authcommon.PrincipalType) ([]authcommon.StrategyResource, error) {

	querySql := "SELECT res_id, res_type FROM auth_strategy_resource WHERE strategy_id IN (SELECT DISTINCT " +
		" ap.strategy_id FROM auth_principal ap join auth_strategy ar ON ap.strategy_id = ar.id WHERE ar.flag = 0 " +
		" AND ap.principal_id = ? AND ap.principal_role = ? )"

	rows, err := s.master.Query(querySql, principalId, principalRole)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, nil
		default:
			log.Error("[Store][Strategy] get principal link resource", zap.String("sql", querySql),
				zap.String("principal-id", principalId), zap.Any("principal-type", principalRole))
			return nil, store.Error(err)
		}
	}

	defer rows.Close()

	resArr := make([]authcommon.StrategyResource, 0)

	for rows.Next() {
		res := new(authcommon.StrategyResource)
		if err := rows.Scan(&res.ResID, &res.ResType); err != nil {
			return nil, store.Error(err)
		}
		resArr = append(resArr, *res)
	}

	return resArr, nil
}

func (s *strategyStore) getStrategyPrincipals(queryHander QueryHandler, id string) ([]authcommon.Principal, error) {

	rows, err := queryHander("SELECT principal_id, principal_role, IFNULL(extend_info, '') FROM auth_principal WHERE strategy_id = ?", id)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			log.Info("[Store][Strategy] not found link principals", zap.String("strategy-id", id))
			return nil, nil
		default:
			return nil, store.Error(err)
		}
	}
	defer rows.Close()

	principals := make([]authcommon.Principal, 0)

	for rows.Next() {
		res := new(authcommon.Principal)
		var extend string
		if err := rows.Scan(&res.PrincipalID, &res.PrincipalType, &extend); err != nil {
			return nil, store.Error(err)
		}
		res.Extend = map[string]string{}
		_ = json.Unmarshal([]byte(extend), &res.Extend)
		principals = append(principals, *res)
	}

	return principals, nil
}

func (s *strategyStore) getStrategyConditions(queryHander QueryHandler, id string) ([]authcommon.Condition, error) {

	rows, err := queryHander("SELECT `key`, `value`, `compare_type` FROM auth_strategy_label WHERE strategy_id = ?", id)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			log.Info("[Store][Strategy] not found link condition", zap.String("strategy-id", id))
			return nil, nil
		default:
			return nil, store.Error(err)
		}
	}
	defer rows.Close()

	conditions := make([]authcommon.Condition, 0)

	for rows.Next() {
		res := new(authcommon.Condition)
		if err := rows.Scan(&res.Key, &res.Value, &res.CompareFunc); err != nil {
			return nil, store.Error(err)
		}
		conditions = append(conditions, *res)
	}

	return conditions, nil
}

func (s *strategyStore) getStrategyFunctions(queryHander QueryHandler, id string) ([]string, error) {

	rows, err := queryHander("SELECT `function` FROM auth_strategy_function WHERE strategy_id = ?", id)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			log.Info("[Store][Strategy] not found link functions", zap.String("strategy-id", id))
			return nil, nil
		default:
			return nil, store.Error(err)
		}
	}
	defer rows.Close()

	functions := make([]string, 0)

	for rows.Next() {
		var item string
		if err := rows.Scan(&item); err != nil {
			return nil, store.Error(err)
		}
		functions = append(functions, item)
	}

	return functions, nil
}

func (s *strategyStore) getStrategyResources(queryHander QueryHandler, id string) ([]authcommon.StrategyResource, error) {
	querySql := "SELECT res_id, res_type FROM auth_strategy_resource WHERE strategy_id = ?"
	rows, err := queryHander(querySql, id)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			log.Info("[Store][Strategy] not found link resources", zap.String("strategy-id", id))
			return nil, nil
		default:
			return nil, store.Error(err)
		}
	}
	defer rows.Close()

	resArr := make([]authcommon.StrategyResource, 0)

	for rows.Next() {
		res := new(authcommon.StrategyResource)
		if err := rows.Scan(&res.ResID, &res.ResType); err != nil {
			return nil, store.Error(err)
		}
		resArr = append(resArr, *res)
	}

	return resArr, nil
}

func fetchRown2StrategyDetail(rows *sql.Rows) (*authcommon.StrategyDetail, error) {
	var (
		ctime, mtime    int64
		isDefault, flag int16
	)
	ret := &authcommon.StrategyDetail{
		Resources: make([]authcommon.StrategyResource, 0),
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

// CleanPrincipalPolicies 清理与自己相关联的鉴权信息
// step 1. 清理用户/用户组默认策略所关联的所有资源信息（直接走delete删除）
// step 2. 清理用户/用户组默认策略
// step 3. 清理用户/用户组所关联的其他鉴权策略的关联关系（直接走delete删除）
func (s *strategyStore) CleanPrincipalPolicies(tx store.Tx, p authcommon.Principal) error {
	dbTx := tx.GetDelegateTx().(*BaseTx)

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

	if _, err := dbTx.Exec(removeResSql, []interface{}{p.Owner, p.PrincipalID, p.PrincipalType}...); err != nil {
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

	if _, err := dbTx.Exec(cleanaRuleSql, []interface{}{p.PrincipalID, p.PrincipalType, p.Owner}...); err != nil {
		return err
	}

	// 调整所关联的鉴权策略的 mtime 数据，保证cache刷新可以获取到变更的数据信息
	updateStrategySql := "UPDATE auth_strategy SET mtime = sysdate()  WHERE id IN (SELECT DISTINCT " +
		" strategy_id FROM auth_principal WHERE principal_id = ? AND principal_role = ?)"
	if _, err := dbTx.Exec(updateStrategySql, []interface{}{p.PrincipalID, p.PrincipalType}...); err != nil {
		return err
	}

	// 清理所在的所有鉴权principal
	cleanPrincipalSql := "DELETE FROM auth_principal WHERE principal_id = ? AND principal_role = ?"
	if _, err := dbTx.Exec(cleanPrincipalSql, []interface{}{p.PrincipalID, p.PrincipalType}...); err != nil {
		return err
	}

	return nil
}
