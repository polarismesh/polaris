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
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	bolt "go.etcd.io/bbolt"
	"go.uber.org/zap"

	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

const (
	tblStrategy string = "strategy"

	StrategyFieldID              string = "ID"
	StrategyFieldName            string = "Name"
	StrategyFieldAction          string = "Action"
	StrategyFieldComment         string = "Comment"
	StrategyFieldUsersPrincipal  string = "Users"
	StrategyFieldGroupsPrincipal string = "Groups"
	StrategyFieldDefault         string = "Default"
	StrategyFieldOwner           string = "Owner"
	StrategyFieldNsResources     string = "NsResources"
	StrategyFieldSvcResources    string = "SvcResources"
	StrategyFieldCfgResources    string = "CfgResources"
	StrategyFieldValid           string = "Valid"
	StrategyFieldRevision        string = "Revision"
	StrategyFieldCreateTime      string = "CreateTime"
	StrategyFieldModifyTime      string = "ModifyTime"
)

var (
	ErrorMultiDefaultStrategy error = errors.New("multiple strategy found")
	ErrorStrategyNotFound     error = errors.New("strategy not fonud")
)

// StrategyStore
type strategyStore struct {
	handler BoltHandler
}

// AddStrategy add a new strategy
func (ss *strategyStore) AddStrategy(tx store.Tx, strategy *authcommon.StrategyDetail) error {
	if strategy.ID == "" || strategy.Name == "" || strategy.Owner == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"add auth_strategy missing some params, id is %s, name is %s, owner is %s",
			strategy.ID, strategy.Name, strategy.Owner))
	}

	initStrategy(strategy)
	dbTx := tx.GetDelegateTx().(*bolt.Tx)

	if err := ss.cleanInvalidStrategy(dbTx, strategy.Name, strategy.Owner); err != nil {
		log.Error("[Store][Strategy] clean invalid auth_strategy", zap.Error(err),
			zap.String("name", strategy.Name), zap.Any("owner", strategy.Owner))
		return err
	}

	if err := saveValue(dbTx, tblStrategy, strategy.ID, convertForStrategyStore(strategy)); err != nil {
		log.Error("[Store][Strategy] save auth_strategy", zap.Error(err),
			zap.String("name", strategy.Name), zap.String("owner", strategy.Owner))
		return err
	}
	return nil
}

// UpdateStrategy update a strategy
func (ss *strategyStore) UpdateStrategy(strategy *authcommon.ModifyStrategyDetail) error {
	if strategy.ID == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"update auth_strategy missing some params, id is %s", strategy.ID))
	}

	proxy, err := ss.handler.StartTx()
	if err != nil {
		return err
	}
	tx := proxy.GetDelegateTx().(*bolt.Tx)

	defer func() {
		_ = tx.Rollback()
	}()

	ret, err := loadStrategyById(tx, strategy.ID)
	if err != nil {
		return err
	}
	if ret == nil {
		return ErrorStrategyNotFound
	}

	return ss.updateStrategy(tx, strategy, ret)
}

// updateStrategy
func (ss *strategyStore) updateStrategy(tx *bolt.Tx, modify *authcommon.ModifyStrategyDetail,
	saveVal *strategyData) error {

	saveVal.Action = modify.Action
	saveVal.Comment = modify.Comment
	saveVal.Revision = utils.NewUUID()
	saveVal.CalleeFunctions = utils.MustJson(modify.CalleeMethods)
	saveVal.Conditions = utils.MustJson(modify.Conditions)

	computePrincipals(false, modify.AddPrincipals, saveVal)
	computePrincipals(true, modify.RemovePrincipals, saveVal)

	saveVal.computeResources(false, modify.AddResources)
	saveVal.computeResources(true, modify.RemoveResources)

	saveVal.ModifyTime = time.Now()

	if err := saveValue(tx, tblStrategy, saveVal.ID, saveVal); err != nil {
		log.Error("[Store][Strategy] update auth_strategy", zap.Error(err),
			zap.String("id", saveVal.ID))
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Error("[Store][Strategy] update auth_strategy tx commit", zap.Error(err),
			zap.String("id", saveVal.ID))
		return err
	}

	return nil
}

func computePrincipals(remove bool, principals []authcommon.Principal, saveVal *strategyData) {
	for i := range principals {
		principal := principals[i]
		if principal.PrincipalType == authcommon.PrincipalUser {
			if remove {
				delete(saveVal.Users, principal.PrincipalID)
			} else {
				saveVal.Users[principal.PrincipalID] = ""
			}
		} else {
			if remove {
				delete(saveVal.Groups, principal.PrincipalID)
			} else {
				saveVal.Groups[principal.PrincipalID] = ""
			}
		}
	}
}

// DeleteStrategy delete a strategy
func (ss *strategyStore) DeleteStrategy(id string) error {
	if id == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"delete auth_strategy missing some params, id is %s", id))
	}

	properties := make(map[string]interface{})
	properties[StrategyFieldValid] = false
	properties[StrategyFieldModifyTime] = time.Now()

	if err := ss.handler.UpdateValue(tblStrategy, id, properties); err != nil {
		log.Error("[Store][Strategy] delete auth_strategy", zap.Error(err), zap.String("id", id))
		return err
	}

	return nil
}

// RemoveStrategyResources 删除策略的资源数据信息
func (ss *strategyStore) RemoveStrategyResources(resources []authcommon.StrategyResource) error {
	return ss.operateStrategyResources(true, resources)
}

// LooseAddStrategyResources 松要求的添加鉴权策略的资源，允许忽略主键冲突的问题
func (ss *strategyStore) LooseAddStrategyResources(resources []authcommon.StrategyResource) error {
	return ss.operateStrategyResources(false, resources)
}

func (ss *strategyStore) operateStrategyResources(remove bool, resources []authcommon.StrategyResource) error {
	proxy, err := ss.handler.StartTx()
	if err != nil {
		return err
	}
	tx := proxy.GetDelegateTx().(*bolt.Tx)

	defer func() {
		_ = tx.Rollback()
	}()

	resMap := buildResMap(resources)
	for id, ress := range resMap {
		rule, err := loadStrategyById(tx, id)
		if err != nil {
			return err
		}
		if rule == nil {
			return ErrorStrategyNotFound
		}

		rule.computeResources(remove, ress)
		rule.ModifyTime = time.Now()
		if err := saveValue(tx, tblStrategy, rule.ID, rule); err != nil {
			log.Error("[Store][Strategy] operate strategy resource", zap.Error(err),
				zap.Bool("remove", remove), zap.String("id", id))
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		log.Error("[Store][Strategy] update auth_strategy resource tx commit",
			zap.Error(err), zap.Bool("remove", remove))
		return err
	}

	return nil
}

func loadStrategyById(tx *bolt.Tx, id string) (*strategyData, error) {
	values := make(map[string]interface{})

	if err := loadValues(tx, tblStrategy, []string{id}, &strategyData{}, values); err != nil {
		log.Error("[Store][Strategy] get auth_strategy by id", zap.Error(err),
			zap.String("id", id))
		return nil, err
	}

	if len(values) == 0 {
		return nil, nil
	}
	if len(values) > 1 {
		return nil, ErrorMultiDefaultStrategy
	}

	var ret *strategyData
	for _, v := range values {
		ret = v.(*strategyData)
		break
	}

	if !ret.Valid {
		return nil, nil
	}

	return ret, nil
}

func buildResMap(resources []authcommon.StrategyResource) map[string][]authcommon.StrategyResource {
	ret := make(map[string][]authcommon.StrategyResource)

	for i := range resources {
		resource := resources[i]
		if _, exist := ret[resource.StrategyID]; !exist {
			ret[resource.StrategyID] = make([]authcommon.StrategyResource, 0, 4)
		}

		val := ret[resource.StrategyID]
		val = append(val, resource)

		ret[resource.StrategyID] = val
	}

	return ret
}

// GetStrategyDetail 获取策略详情
func (ss *strategyStore) GetStrategyDetail(id string) (*authcommon.StrategyDetail, error) {
	proxy, err := ss.handler.StartTx()
	if err != nil {
		return nil, err
	}
	tx := proxy.GetDelegateTx().(*bolt.Tx)
	defer func() {
		_ = tx.Rollback()
	}()

	return ss.getStrategyDetail(tx, id)
}

// GetStrategyDetail
func (ss *strategyStore) getStrategyDetail(tx *bolt.Tx, id string) (*authcommon.StrategyDetail, error) {
	ret, err := loadStrategyById(tx, id)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return nil, nil
	}

	return convertForStrategyDetail(ret), nil
}

// GetStrategyResources 获取策略的资源
func (ss *strategyStore) GetStrategyResources(principalId string,
	principalRole authcommon.PrincipalType) ([]authcommon.StrategyResource, error) {

	fields := []string{StrategyFieldValid, StrategyFieldDefault, StrategyFieldUsersPrincipal}

	if principalRole == authcommon.PrincipalGroup {
		fields = []string{StrategyFieldValid, StrategyFieldDefault, StrategyFieldGroupsPrincipal}
	}

	values, err := ss.handler.LoadValuesByFilter(tblStrategy, fields, &strategyData{},
		func(m map[string]interface{}) bool {
			valid, ok := m[StrategyFieldValid].(bool)
			if ok && !valid {
				return false
			}

			var principals map[string]string

			if principalRole == authcommon.PrincipalUser {
				principals, _ = m[StrategyFieldUsersPrincipal].(map[string]string)
			} else {
				principals, _ = m[StrategyFieldGroupsPrincipal].(map[string]string)
			}

			_, exist := principals[principalId]

			return exist
		})

	if err != nil {
		return nil, err
	}

	ret := make([]authcommon.StrategyResource, 0, 4)

	for _, item := range values {
		rule := item.(*strategyData)
		ret = append(ret, rule.GetResources()...)
	}

	return ret, nil
}

// GetDefaultStrategyDetailByPrincipal 获取默认策略详情
func (ss *strategyStore) GetDefaultStrategyDetailByPrincipal(principalId string,
	principalType authcommon.PrincipalType) (*authcommon.StrategyDetail, error) {

	fields := []string{StrategyFieldValid, StrategyFieldDefault, StrategyFieldUsersPrincipal}

	if principalType == authcommon.PrincipalGroup {
		fields = []string{StrategyFieldValid, StrategyFieldDefault, StrategyFieldGroupsPrincipal, CommonFieldMetadata}
	}

	values, err := ss.handler.LoadValuesByFilter(tblStrategy, fields, &strategyData{},
		func(m map[string]interface{}) bool {
			valid, ok := m[StrategyFieldValid].(bool)
			if ok && !valid {
				return false
			}

			isDefault, _ := m[StrategyFieldDefault].(bool)
			if !isDefault {
				return false
			}

			var principals map[string]string

			if principalType == authcommon.PrincipalUser {
				principals, _ = m[StrategyFieldUsersPrincipal].(map[string]string)
			} else {
				principals, _ = m[StrategyFieldGroupsPrincipal].(map[string]string)
			}
			if _, exist := principals[principalId]; !exist {
				return false
			}

			metadata, _ := m[CommonFieldMetadata].(map[string]string)
			if val, exist := metadata[authcommon.MetadKeySystemDefaultPolicy]; exist && val == "true" {
				return false
			}
			return true
		})

	if err != nil {
		log.Error("[Store][Strategy] get default auth_strategy by principal", zap.Error(err),
			zap.String("principal-id", principalId), zap.String("principal", principalType.String()))
		return nil, err
	}
	if len(values) == 0 {
		return nil, ErrorStrategyNotFound
	}
	if len(values) > 1 {
		return nil, ErrorMultiDefaultStrategy
	}

	var ret *strategyData
	for _, v := range values {
		ret = v.(*strategyData)
		break
	}

	return convertForStrategyDetail(ret), nil
}

func compareResExist(resType, resId string, m map[string]interface{}) bool {
	saveNsRes, _ := m[StrategyFieldNsResources].(map[string]string)
	saveSvcRes, _ := m[StrategyFieldSvcResources].(map[string]string)
	saveCfgRes, _ := m[StrategyFieldCfgResources].(map[string]string)

	if strings.Compare(resType, "0") == 0 {
		_, exist := saveNsRes[resId]
		return exist
	}

	if strings.Compare(resType, "1") == 0 {
		_, exist := saveSvcRes[resId]
		return exist
	}

	if strings.Compare(resType, "2") == 0 {
		_, exist := saveCfgRes[resId]
		return exist
	}

	return true
}

func comparePrincipalExist(principalType, principalId string, m map[string]interface{}) bool {
	saveUsers, _ := m[StrategyFieldUsersPrincipal].(map[string]string)
	saveGroups, _ := m[StrategyFieldGroupsPrincipal].(map[string]string)

	if strings.Compare(principalType, "1") == 0 {
		_, exist := saveUsers[principalId]
		return exist
	}

	if strings.Compare(principalType, "2") == 0 {
		_, exist := saveGroups[principalId]
		return exist
	}

	return true
}

// GetMoreStrategies get strategy details for cache
func (ss *strategyStore) GetMoreStrategies(mtime time.Time, firstUpdate bool) ([]*authcommon.StrategyDetail, error) {

	ret, err := ss.handler.LoadValuesByFilter(tblStrategy, []string{StrategyFieldModifyTime}, &strategyData{},
		func(m map[string]interface{}) bool {
			mt := m[StrategyFieldModifyTime].(time.Time)
			isAfter := mt.After(mtime)
			return isAfter
		})
	if err != nil {
		log.Error("[Store][Strategy] get auth_strategy for cache", zap.Error(err))
		return nil, err
	}

	strategies := make([]*authcommon.StrategyDetail, 0, len(ret))

	for k := range ret {
		val := ret[k]
		strategies = append(strategies, convertForStrategyDetail(val.(*strategyData)))
	}

	return strategies, nil
}

func (ss *strategyStore) CleanPrincipalPolicies(tx store.Tx, p authcommon.Principal) error {
	fields := []string{StrategyFieldDefault, StrategyFieldUsersPrincipal, StrategyFieldGroupsPrincipal}
	values := make(map[string]interface{})

	dbTx := tx.GetDelegateTx().(*bolt.Tx)
	err := loadValuesByFilter(dbTx, tblStrategy, fields, &strategyData{},
		func(m map[string]interface{}) bool {
			isDefault := m[StrategyFieldDefault].(bool)
			if !isDefault {
				return false
			}

			var principals map[string]string
			if p.PrincipalType == authcommon.PrincipalUser {
				principals = m[StrategyFieldUsersPrincipal].(map[string]string)
			} else {
				principals = m[StrategyFieldGroupsPrincipal].(map[string]string)
			}

			if len(principals) != 1 {
				return false
			}
			_, exist := principals[p.PrincipalID]
			return exist
		}, values)

	if err != nil {
		log.Error("[Store][Strategy] load link auth_strategy", zap.Error(err), zap.String("principal", p.String()))
		return err
	}

	if len(values) == 0 {
		return nil
	}
	if len(values) > 1 {
		return ErrorMultiDefaultStrategy
	}

	for k := range values {

		properties := make(map[string]interface{})
		properties[StrategyFieldValid] = false
		properties[StrategyFieldModifyTime] = time.Now()

		if err := updateValue(dbTx, tblStrategy, k, properties); err != nil {
			log.Error("[Store][Strategy] clean link auth_strategy", zap.String("principal", p.String()), zap.Error(err))
			return err
		}
	}
	return nil
}

// cleanInvalidStrategy clean up authentication strategy by name
func (ss *strategyStore) cleanInvalidStrategy(tx *bolt.Tx, name, owner string) error {

	fields := []string{StrategyFieldName, StrategyFieldOwner, StrategyFieldValid}
	values := make(map[string]interface{})

	err := loadValuesByFilter(tx, tblStrategy, fields, &strategyData{},
		func(m map[string]interface{}) bool {
			valid, ok := m[StrategyFieldValid].(bool)
			// 如果数据是 valid 的，则不能被清理
			if ok && valid {
				return false
			}

			saveName := m[StrategyFieldName]
			saveOwner := m[StrategyFieldOwner]

			return saveName == name && saveOwner == owner
		}, values)

	if err != nil {
		log.Error("[Store][Strategy] clean invalid auth_strategy", zap.Error(err),
			zap.String("name", name), zap.Any("owner", owner))
		return err
	}

	if len(values) == 0 {
		return nil
	}

	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}

	return deleteValues(tx, tblStrategy, keys)
}

type strategyData struct {
	ID              string
	Name            string
	Action          string
	Comment         string
	Users           map[string]string
	Groups          map[string]string
	Default         bool
	Owner           string
	NsResources     map[string]string
	SvcResources    map[string]string
	CfgResources    map[string]string
	AllResources    string
	CalleeFunctions string
	Conditions      string
	Valid           bool
	Revision        string
	CreateTime      time.Time
	ModifyTime      time.Time
}

func (s *strategyData) computeResources(remove bool, resources []authcommon.StrategyResource) {
	saveVal := s.GetResources()

	tmp := make(map[string]authcommon.StrategyResource, 8)
	for i := range saveVal {
		tmp[saveVal[i].Key()] = saveVal[i]
	}
	for i := range resources {
		resource := resources[i]
		if remove {
			delete(tmp, resource.Key())
		} else {
			tmp[resource.Key()] = resource
		}
	}

	ret := make([]authcommon.StrategyResource, 0, 8)
	for i := range tmp {
		ret = append(ret, tmp[i])
	}

	s.AllResources = utils.MustJson(ret)
}

func (s *strategyData) GetResources() []authcommon.StrategyResource {
	ret := make([]authcommon.StrategyResource, 0, len(s.NsResources)+len(s.SvcResources)+len(s.CfgResources))

	for id := range s.NsResources {
		ret = append(ret, authcommon.StrategyResource{
			StrategyID: s.ID,
			ResType:    int32(apisecurity.ResourceType_Namespaces),
			ResID:      id,
		})
	}

	for id := range s.SvcResources {
		ret = append(ret, authcommon.StrategyResource{
			StrategyID: s.ID,
			ResType:    int32(apisecurity.ResourceType_Services),
			ResID:      id,
		})
	}

	for id := range s.CfgResources {
		ret = append(ret, authcommon.StrategyResource{
			StrategyID: s.ID,
			ResType:    int32(apisecurity.ResourceType_ConfigGroups),
			ResID:      id,
		})
	}
	if len(s.AllResources) != 0 {
		ret = make([]authcommon.StrategyResource, 0, 4)
		_ = json.Unmarshal([]byte(s.AllResources), &ret)
	}
	return ret
}

func convertForStrategyStore(strategy *authcommon.StrategyDetail) *strategyData {

	var (
		users      = make(map[string]string, 4)
		groups     = make(map[string]string, 4)
		principals = strategy.Principals
	)

	for i := range principals {
		principal := principals[i]
		if principal.PrincipalType == authcommon.PrincipalUser {
			users[principal.PrincipalID] = ""
		} else {
			groups[principal.PrincipalID] = ""
		}
	}

	return &strategyData{
		ID:              strategy.ID,
		Name:            strategy.Name,
		Action:          strategy.Action,
		Comment:         strategy.Comment,
		Users:           users,
		Groups:          groups,
		Default:         strategy.Default,
		Owner:           strategy.Owner,
		AllResources:    utils.MustJson(strategy.Resources),
		CalleeFunctions: utils.MustJson(strategy.CalleeMethods),
		Conditions:      utils.MustJson(strategy.Conditions),
		Valid:           strategy.Valid,
		Revision:        strategy.Revision,
		CreateTime:      strategy.CreateTime,
		ModifyTime:      strategy.ModifyTime,
	}
}

func convertForStrategyDetail(strategy *strategyData) *authcommon.StrategyDetail {

	principals := make([]authcommon.Principal, 0, len(strategy.Users)+len(strategy.Groups))

	for id := range strategy.Users {
		principals = append(principals, authcommon.Principal{
			StrategyID:    strategy.ID,
			PrincipalID:   id,
			PrincipalType: authcommon.PrincipalUser,
		})
	}
	for id := range strategy.Groups {
		principals = append(principals, authcommon.Principal{
			StrategyID:    strategy.ID,
			PrincipalID:   id,
			PrincipalType: authcommon.PrincipalGroup,
		})
	}

	ret := &authcommon.StrategyDetail{
		ID:         strategy.ID,
		Name:       strategy.Name,
		Action:     strategy.Action,
		Comment:    strategy.Comment,
		Principals: principals,
		Resources:  strategy.GetResources(),
		Default:    strategy.Default,
		Owner:      strategy.Owner,
		Valid:      strategy.Valid,
		Revision:   strategy.Revision,
		CreateTime: strategy.CreateTime,
		ModifyTime: strategy.ModifyTime,
	}

	if len(strategy.CalleeFunctions) != 0 {
		functions := make([]string, 0, 4)
		_ = json.Unmarshal([]byte(strategy.CalleeFunctions), &functions)
		ret.CalleeMethods = functions
	}
	if len(strategy.Conditions) != 0 {
		condition := make([]authcommon.Condition, 0, 4)
		_ = json.Unmarshal([]byte(strategy.Conditions), &condition)
		ret.Conditions = condition
	}
	return ret
}

func initStrategy(rule *authcommon.StrategyDetail) {
	if rule != nil {
		rule.Valid = true

		tn := time.Now()
		rule.CreateTime = tn
		rule.ModifyTime = tn

		for i := range rule.Resources {
			rule.Resources[i].StrategyID = rule.ID
		}
	}
}
