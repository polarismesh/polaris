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
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"go.uber.org/zap"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	logger "github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/store"
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

type strategyForStore struct {
	ID           string
	Name         string
	Action       string
	Comment      string
	Users        map[string]string
	Groups       map[string]string
	Default      bool
	Owner        string
	NsResources  map[string]string
	SvcResources map[string]string
	CfgResources map[string]string
	Valid        bool
	Revision     string
	CreateTime   time.Time
	ModifyTime   time.Time
}

// StrategyStore
type strategyStore struct {
	handler BoltHandler
}

// AddStrategy add a new strategy
func (ss *strategyStore) AddStrategy(strategy *model.StrategyDetail) error {
	if strategy.ID == "" || strategy.Name == "" || strategy.Owner == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"add auth_strategy missing some params, id is %s, name is %s, owner is %s",
			strategy.ID, strategy.Name, strategy.Owner))
	}

	initStrategy(strategy)

	proxy, err := ss.handler.StartTx()
	if err != nil {
		return err
	}
	tx := proxy.GetDelegateTx().(*bolt.Tx)

	defer tx.Rollback()

	return ss.addStrategy(tx, strategy)
}

func (ss *strategyStore) addStrategy(tx *bolt.Tx, strategy *model.StrategyDetail) error {
	if err := ss.cleanInvalidStrategy(tx, strategy.Name, strategy.Owner); err != nil {
		logger.StoreScope().Error("[Store][Strategy] clean invalid auth_strategy", zap.Error(err),
			zap.String("name", strategy.Name), zap.Any("owner", strategy.Owner))
		return err
	}

	if err := saveValue(tx, tblStrategy, strategy.ID, convertForStrategyStore(strategy)); err != nil {
		logger.StoreScope().Error("[Store][Strategy] save auth_strategy", zap.Error(err),
			zap.String("name", strategy.Name), zap.String("owner", strategy.Owner))
		return err
	}

	if err := tx.Commit(); err != nil {
		logger.StoreScope().Error("[Store][Strategy] clean invalid auth_strategy tx commit", zap.Error(err),
			zap.String("name", strategy.Name), zap.String("owner", strategy.Owner))
		return err
	}

	return nil
}

// UpdateStrategy update a strategy
func (ss *strategyStore) UpdateStrategy(strategy *model.ModifyStrategyDetail) error {
	if strategy.ID == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"update auth_strategy missing some params, id is %s", strategy.ID))
	}

	proxy, err := ss.handler.StartTx()
	if err != nil {
		return err
	}
	tx := proxy.GetDelegateTx().(*bolt.Tx)

	defer tx.Rollback()

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
func (ss *strategyStore) updateStrategy(tx *bolt.Tx, modify *model.ModifyStrategyDetail,
	saveVal *strategyForStore) error {

	saveVal.Action = modify.Action
	saveVal.Comment = modify.Comment
	saveVal.Revision = utils.NewUUID()

	computePrincipals(false, modify.AddPrincipals, saveVal)
	computePrincipals(true, modify.RemovePrincipals, saveVal)

	computeResources(false, modify.AddResources, saveVal)
	computeResources(true, modify.RemoveResources, saveVal)

	if err := saveValue(tx, tblStrategy, saveVal.ID, saveVal); err != nil {
		logger.StoreScope().Error("[Store][Strategy] update auth_strategy", zap.Error(err),
			zap.String("id", saveVal.ID))
		return err
	}

	if err := tx.Commit(); err != nil {
		logger.StoreScope().Error("[Store][Strategy] update auth_strategy tx commit", zap.Error(err),
			zap.String("id", saveVal.ID))
		return err
	}

	return nil
}

func computePrincipals(remove bool, principals []model.Principal, saveVal *strategyForStore) {
	for i := range principals {
		principal := principals[i]
		if principal.PrincipalRole == model.PrincipalUser {
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

func computeResources(remove bool, resources []model.StrategyResource, saveVal *strategyForStore) {
	for i := range resources {
		resource := resources[i]
		if resource.ResType == int32(api.ResourceType_Namespaces) {
			if remove {
				delete(saveVal.NsResources, resource.ResID)
			} else {
				saveVal.NsResources[resource.ResID] = ""
			}
			continue
		}
		if resource.ResType == int32(api.ResourceType_Services) {
			if remove {
				delete(saveVal.SvcResources, resource.ResID)
			} else {
				saveVal.SvcResources[resource.ResID] = ""
			}
			continue
		}
		if resource.ResType == int32(api.ResourceType_ConfigGroups) {
			if remove {
				delete(saveVal.CfgResources, resource.ResID)
			} else {
				saveVal.CfgResources[resource.ResID] = ""
			}
			continue
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
		logger.StoreScope().Error("[Store][Strategy] delete auth_strategy", zap.Error(err), zap.String("id", id))
		return err
	}

	return nil
}

// RemoveStrategyResources 删除策略的资源数据信息
func (ss *strategyStore) RemoveStrategyResources(resources []model.StrategyResource) error {
	return ss.operateStrategyResources(true, resources)
}

// LooseAddStrategyResources 松要求的添加鉴权策略的资源，允许忽略主键冲突的问题
func (ss *strategyStore) LooseAddStrategyResources(resources []model.StrategyResource) error {
	return ss.operateStrategyResources(false, resources)
}

func (ss *strategyStore) operateStrategyResources(remove bool, resources []model.StrategyResource) error {
	proxy, err := ss.handler.StartTx()
	if err != nil {
		return err
	}
	tx := proxy.GetDelegateTx().(*bolt.Tx)

	defer tx.Rollback()

	resMap := buildResMap(resources)

	for id, ress := range resMap {
		rule, err := loadStrategyById(tx, id)
		if err != nil {
			return err
		}
		if rule == nil {
			return ErrorStrategyNotFound
		}

		computeResources(remove, ress, rule)
		if err := saveValue(tx, tblStrategy, rule.ID, rule); err != nil {
			logger.StoreScope().Error("[Store][Strategy] operate strategy resource", zap.Error(err),
				zap.Bool("remove", remove), zap.String("id", id))
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		logger.StoreScope().Error("[Store][Strategy] update auth_strategy resource tx commit",
			zap.Error(err), zap.Bool("remove", remove))
		return err
	}

	return nil
}

func loadStrategyById(tx *bolt.Tx, id string) (*strategyForStore, error) {
	values := make(map[string]interface{})

	if err := loadValues(tx, tblStrategy, []string{id}, &strategyForStore{}, values); err != nil {
		logger.StoreScope().Error("[Store][Strategy] get auth_strategy by id", zap.Error(err),
			zap.String("id", id))
		return nil, err
	}

	if len(values) == 0 {
		return nil, nil
	}
	if len(values) > 1 {
		return nil, ErrorMultiDefaultStrategy
	}

	var ret *strategyForStore
	for _, v := range values {
		ret = v.(*strategyForStore)
		break
	}

	if !ret.Valid {
		return nil, nil
	}

	return ret, nil
}

func buildResMap(resources []model.StrategyResource) map[string][]model.StrategyResource {
	ret := make(map[string][]model.StrategyResource)

	for i := range resources {
		resource := resources[i]
		if _, exist := ret[resource.StrategyID]; !exist {
			ret[resource.StrategyID] = make([]model.StrategyResource, 0, 4)
		}

		val := ret[resource.StrategyID]
		val = append(val, resource)

		ret[resource.StrategyID] = val
	}

	return ret
}

// GetStrategyDetail 获取策略详情
func (ss *strategyStore) GetStrategyDetail(id string) (*model.StrategyDetail, error) {
	proxy, err := ss.handler.StartTx()
	if err != nil {
		return nil, err
	}
	tx := proxy.GetDelegateTx().(*bolt.Tx)

	defer tx.Rollback()

	return ss.getStrategyDetail(tx, id)
}

// GetStrategyDetail
func (ss *strategyStore) getStrategyDetail(tx *bolt.Tx, id string) (*model.StrategyDetail, error) {
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
	principalRole model.PrincipalType) ([]model.StrategyResource, error) {

	fields := []string{StrategyFieldValid, StrategyFieldDefault, StrategyFieldUsersPrincipal}

	if principalRole == model.PrincipalGroup {
		fields = []string{StrategyFieldValid, StrategyFieldDefault, StrategyFieldGroupsPrincipal}
	}

	values, err := ss.handler.LoadValuesByFilter(tblStrategy, fields, &strategyForStore{},
		func(m map[string]interface{}) bool {
			valid, ok := m[StrategyFieldValid].(bool)
			if ok && !valid {
				return false
			}

			var principals map[string]string

			if principalRole == model.PrincipalUser {
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

	ret := make([]model.StrategyResource, 0, 4)

	for _, item := range values {
		rule := item.(*strategyForStore)
		ret = append(ret, collectStrategyResources(rule)...)
	}

	return ret, nil
}

func collectStrategyResources(rule *strategyForStore) []model.StrategyResource {
	ret := make([]model.StrategyResource, 0, len(rule.NsResources)+len(rule.SvcResources)+len(rule.CfgResources))

	for id := range rule.NsResources {
		ret = append(ret, model.StrategyResource{
			StrategyID: rule.ID,
			ResType:    int32(api.ResourceType_Namespaces),
			ResID:      id,
		})
	}

	for id := range rule.SvcResources {
		ret = append(ret, model.StrategyResource{
			StrategyID: rule.ID,
			ResType:    int32(api.ResourceType_Services),
			ResID:      id,
		})
	}

	for id := range rule.CfgResources {
		ret = append(ret, model.StrategyResource{
			StrategyID: rule.ID,
			ResType:    int32(api.ResourceType_ConfigGroups),
			ResID:      id,
		})
	}

	return ret
}

// GetDefaultStrategyDetailByPrincipal 获取默认策略详情
func (ss *strategyStore) GetDefaultStrategyDetailByPrincipal(principalId string,
	principalType model.PrincipalType) (*model.StrategyDetail, error) {

	fields := []string{StrategyFieldValid, StrategyFieldDefault, StrategyFieldUsersPrincipal}

	if principalType == model.PrincipalGroup {
		fields = []string{StrategyFieldValid, StrategyFieldDefault, StrategyFieldGroupsPrincipal}
	}

	values, err := ss.handler.LoadValuesByFilter(tblStrategy, fields, &strategyForStore{},
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

			if principalType == model.PrincipalUser {
				principals, _ = m[StrategyFieldUsersPrincipal].(map[string]string)
			} else {
				principals, _ = m[StrategyFieldGroupsPrincipal].(map[string]string)
			}

			_, exist := principals[principalId]

			return exist
		})

	if err != nil {
		logger.StoreScope().Error("[Store][Strategy] get default auth_strategy by principal", zap.Error(err),
			zap.String("principal-id", principalId), zap.String("principal", principalType.String()))
		return nil, err
	}
	if len(values) == 0 {
		return nil, ErrorStrategyNotFound
	}
	if len(values) > 1 {
		return nil, ErrorMultiDefaultStrategy
	}

	var ret *strategyForStore
	for _, v := range values {
		ret = v.(*strategyForStore)
		break
	}

	return convertForStrategyDetail(ret), nil
}

// GetStrategies 查询鉴权策略列表
func (ss *strategyStore) GetStrategies(filters map[string]string, offset uint32, limit uint32) (uint32,
	[]*model.StrategyDetail, error) {

	showDetail := filters["show_detail"]
	delete(filters, "show_detail")

	return ss.listStrategies(filters, offset, limit, showDetail == "true")
}

func (ss *strategyStore) listStrategies(filters map[string]string, offset uint32, limit uint32,
	showDetail bool) (uint32, []*model.StrategyDetail, error) {

	fields := []string{StrategyFieldValid, StrategyFieldName, StrategyFieldUsersPrincipal,
		StrategyFieldGroupsPrincipal, StrategyFieldNsResources, StrategyFieldSvcResources,
		StrategyFieldCfgResources, StrategyFieldOwner, StrategyFieldDefault}

	values, err := ss.handler.LoadValuesByFilter(tblStrategy, fields, &strategyForStore{},
		func(m map[string]interface{}) bool {
			valid, ok := m[StrategyFieldValid].(bool)
			if ok && !valid {
				return false
			}

			saveName, _ := m[StrategyFieldName].(string)
			saveDefault, _ := m[StrategyFieldDefault].(bool)
			saveOwner, _ := m[StrategyFieldOwner].(string)

			if name, ok := filters["name"]; ok {
				if utils.IsWildName(name) {
					name = name[:len(name)-1]
				}
				if !strings.Contains(saveName, name) {
					return false
				}
			}

			if owner, ok := filters["owner"]; ok {
				if strings.Compare(saveOwner, owner) != 0 {
					if principalId, ok := filters["principal_id"]; ok {
						principalType := filters["principal_type"]
						if !comparePrincipalExist(principalType, principalId, m) {
							return false
						}
					}
				}
			}

			if isDefault, ok := filters["default"]; ok {
				compareParam2BoolNotEqual := func(param string, b bool) bool {
					if param == "0" && !b {
						return true
					}
					if param == "1" && b {
						return true
					}
					return false
				}
				if !compareParam2BoolNotEqual(isDefault, saveDefault) {
					return false
				}
			}

			if resType, ok := filters["res_type"]; ok {
				resId := filters["res_id"]
				if !compareResExist(resType, resId, m) {
					return false
				}
			}

			if principalId, ok := filters["principal_id"]; ok {
				principalType := filters["principal_type"]
				if !comparePrincipalExist(principalType, principalId, m) {
					return false
				}
			}

			return true
		})

	if err != nil {
		logger.StoreScope().Error("[Store][Strategy] get auth_strategy for list", zap.Error(err))
		return 0, nil, err
	}

	return uint32(len(values)), doStrategyPage(values, offset, limit, showDetail), nil
}

func doStrategyPage(ret map[string]interface{}, offset, limit uint32, showDetail bool) []*model.StrategyDetail {
	rules := make([]*model.StrategyDetail, 0, len(ret))

	beginIndex := offset
	endIndex := beginIndex + limit
	totalCount := uint32(len(ret))

	if totalCount == 0 {
		return rules
	}
	if beginIndex >= endIndex {
		return rules
	}
	if beginIndex >= totalCount {
		return rules
	}
	if endIndex > totalCount {
		endIndex = totalCount
	}

	emptyPrincipals := make([]model.Principal, 0)
	emptyResources := make([]model.StrategyResource, 0)

	for k := range ret {
		rule := convertForStrategyDetail(ret[k].(*strategyForStore))
		if !showDetail {
			rule.Principals = emptyPrincipals
			rule.Resources = emptyResources
		}
		rules = append(rules, rule)
	}

	sort.Slice(rules, func(i, j int) bool {
		return rules[i].ModifyTime.After(rules[j].ModifyTime)
	})

	return rules[beginIndex:endIndex]
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

// GetStrategyDetailsForCache get strategy details for cache
func (ss *strategyStore) GetStrategyDetailsForCache(mtime time.Time,
	firstUpdate bool) ([]*model.StrategyDetail, error) {

	ret, err := ss.handler.LoadValuesByFilter(tblStrategy, []string{StrategyFieldModifyTime}, &strategyForStore{},
		func(m map[string]interface{}) bool {
			mt := m[StrategyFieldModifyTime].(time.Time)
			isAfter := mt.After(mtime)
			return isAfter
		})
	if err != nil {
		logger.StoreScope().Error("[Store][Strategy] get auth_strategy for cache", zap.Error(err))
		return nil, err
	}

	strategies := make([]*model.StrategyDetail, 0, len(ret))

	for k := range ret {
		val := ret[k]
		strategies = append(strategies, convertForStrategyDetail(val.(*strategyForStore)))
	}

	return strategies, nil
}

// cleanInvalidStrategy clean up authentication strategy by name
func (ss *strategyStore) cleanInvalidStrategy(tx *bolt.Tx, name, owner string) error {

	fields := []string{StrategyFieldName, StrategyFieldOwner, StrategyFieldValid}
	values := make(map[string]interface{})

	err := loadValuesByFilter(tx, tblStrategy, fields, &strategyForStore{},
		func(m map[string]interface{}) bool {
			valid, ok := m[StrategyFieldValid].(bool)
			// 如果数据是 valid 的，则不能被清理
			if ok && valid {
				return false
			}

			saveName, _ := m[StrategyFieldName]
			saveOwner, _ := m[StrategyFieldOwner]

			return saveName == name && saveOwner == owner
		}, values)

	if err != nil {
		logger.StoreScope().Error("[Store][Strategy] clean invalid auth_strategy", zap.Error(err),
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

	return deleteValues(tx, tblStrategy, keys, false)
}

func createDefaultStrategy(tx *bolt.Tx, role model.PrincipalType, principalId, name, owner string) error {
	strategy := &model.StrategyDetail{
		ID:        utils.NewUUID(),
		Name:      model.BuildDefaultStrategyName(role, name),
		Action:    api.AuthAction_READ_WRITE.String(),
		Default:   true,
		Owner:     owner,
		Revision:  utils.NewUUID(),
		Resources: []model.StrategyResource{},
		Valid:     true,
		Principals: []model.Principal{
			{
				PrincipalID:   principalId,
				PrincipalRole: role,
			},
		},
		Comment: "Default Strategy",
	}

	return saveValue(tx, tblStrategy, strategy.ID, convertForStrategyStore(strategy))
}

func cleanLinkStrategy(tx *bolt.Tx, role model.PrincipalType, principalId, owner string) error {

	fields := []string{StrategyFieldDefault, StrategyFieldUsersPrincipal, StrategyFieldGroupsPrincipal}
	values := make(map[string]interface{})

	err := loadValuesByFilter(tx, tblStrategy, fields, &strategyForStore{},
		func(m map[string]interface{}) bool {
			isDefault := m[StrategyFieldDefault].(bool)
			if !isDefault {
				return false
			}

			var principals map[string]string
			if role == model.PrincipalUser {
				principals = m[StrategyFieldUsersPrincipal].(map[string]string)
			} else {
				principals = m[StrategyFieldGroupsPrincipal].(map[string]string)
			}

			if len(principals) != 1 {
				return false
			}
			_, exist := principals[principalId]
			return exist
		}, values)

	if err != nil {
		logger.StoreScope().Error("[Store][Strategy] load link auth_strategy", zap.Error(err),
			zap.String("principal-id", principalId), zap.Any("principal-type", role))
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

		if err := updateValue(tx, tblStrategy, k, properties); err != nil {
			logger.StoreScope().Error("[Store][Strategy] clean link auth_strategy", zap.Error(err),
				zap.String("principal-id", principalId), zap.Any("principal-type", role))
			return err
		}
	}

	return nil
}

func convertForStrategyStore(strategy *model.StrategyDetail) *strategyForStore {

	var (
		users      = make(map[string]string, 4)
		groups     = make(map[string]string, 4)
		principals = strategy.Principals
	)

	for i := range principals {
		principal := principals[i]
		if principal.PrincipalRole == model.PrincipalUser {
			users[principal.PrincipalID] = ""
		} else {
			groups[principal.PrincipalID] = ""
		}
	}

	ns := make(map[string]string, 4)
	svc := make(map[string]string, 4)
	cfg := make(map[string]string, 4)

	resources := strategy.Resources

	for i := range resources {
		res := resources[i]
		switch res.ResType {
		case int32(api.ResourceType_Namespaces):
			ns[res.ResID] = ""
		case int32(api.ResourceType_Services):
			svc[res.ResID] = ""
		case int32(api.ResourceType_ConfigGroups):
			cfg[res.ResID] = ""
		}
	}

	return &strategyForStore{
		ID:           strategy.ID,
		Name:         strategy.Name,
		Action:       strategy.Action,
		Comment:      strategy.Comment,
		Users:        users,
		Groups:       groups,
		Default:      strategy.Default,
		Owner:        strategy.Owner,
		NsResources:  ns,
		SvcResources: svc,
		CfgResources: cfg,
		Valid:        strategy.Valid,
		Revision:     strategy.Revision,
		CreateTime:   strategy.CreateTime,
		ModifyTime:   strategy.ModifyTime,
	}
}

func convertForStrategyDetail(strategy *strategyForStore) *model.StrategyDetail {

	principals := make([]model.Principal, 0, len(strategy.Users)+len(strategy.Groups))
	resources := make([]model.StrategyResource, 0, len(strategy.NsResources)+
		len(strategy.SvcResources)+len(strategy.CfgResources))

	for id := range strategy.Users {
		principals = append(principals, model.Principal{
			StrategyID:    strategy.ID,
			PrincipalID:   id,
			PrincipalRole: model.PrincipalUser,
		})
	}
	for id := range strategy.Groups {
		principals = append(principals, model.Principal{
			StrategyID:    strategy.ID,
			PrincipalID:   id,
			PrincipalRole: model.PrincipalGroup,
		})
	}

	fillRes := func(idMap map[string]string, resType api.ResourceType) []model.StrategyResource {
		res := make([]model.StrategyResource, 0, len(idMap))

		for id := range idMap {
			res = append(res, model.StrategyResource{
				StrategyID: strategy.ID,
				ResType:    int32(resType),
				ResID:      id,
			})
		}

		return res
	}

	resources = append(resources, fillRes(strategy.NsResources, api.ResourceType_Namespaces)...)
	resources = append(resources, fillRes(strategy.SvcResources, api.ResourceType_Services)...)
	resources = append(resources, fillRes(strategy.CfgResources, api.ResourceType_ConfigGroups)...)

	return &model.StrategyDetail{
		ID:         strategy.ID,
		Name:       strategy.Name,
		Action:     strategy.Action,
		Comment:    strategy.Comment,
		Principals: principals,
		Resources:  resources,
		Default:    strategy.Default,
		Owner:      strategy.Owner,
		Valid:      strategy.Valid,
		Revision:   strategy.Revision,
		CreateTime: strategy.CreateTime,
		ModifyTime: strategy.ModifyTime,
	}
}

func initStrategy(rule *model.StrategyDetail) {
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
