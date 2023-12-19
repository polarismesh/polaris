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
	"sort"
	"strconv"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/store"
)

const (
	// rule 相关信息以及映射
	tblCircuitBreakerRule string = "circuitbreaker_rule_v2"
)

const (
	CbFieldLevel        = "Level"
	CbFieldSrcService   = "SrcService"
	CbFieldSrcNamespace = "SrcNamespace"
	CbFieldDstService   = "DstService"
	CbFieldDstNamespace = "DstNamespace"
	CbFieldDstMethod    = "DstMethod"
	CbFieldRule         = "Rule"
)

const (
	// rule 相关信息以及映射
	tblCircuitBreaker string = "circuitbreaker_rule"

	// relation 相关信息以及映射信息
	tblCircuitBreakerRelation string = "circuitbreaker_rule_relation"
	VersionForMaster          string = "master"
	CBFieldNameValid          string = "Valid"
	CBFieldNameVersion        string = "Version"
	CBFieldNameID             string = "ID"
	CBFieldNameModifyTime     string = "ModifyTime"

	CBRFieldNameServiceID   string = "ServiceID"
	CBRFieldNameRuleID      string = "RuleID"
	CBRFieldNameRuleVersion string = "RuleVersion"

	CBRelationFieldServiceID   string = "ServiceID"
	CBRelationFieldRuleID      string = "RuleID"
	CBRelationFieldRuleVersion string = "RuleVersion"
	CBRelationFieldValid       string = "Valid"
	CBRelationFieldCreateTime  string = "CreateTime"
	CBRelationFieldModifyTime  string = "ModifyTime"
)

type circuitBreakerStore struct {
	handler BoltHandler
}

func initCircuitBreakerRule(cb *model.CircuitBreakerRule) {
	cb.Valid = true
	cb.CreateTime = time.Now()
	cb.ModifyTime = time.Now()
}

// cleanCircuitBreaker 彻底清理熔断规则
func (c *circuitBreakerStore) cleanCircuitBreakerRule(id string) error {
	if err := c.handler.DeleteValues(tblCircuitBreakerRule, []string{id}); err != nil {
		log.Errorf("[Store][circuitBreaker] clean invalid circuit-breaker rule(%s) err: %s",
			id, err.Error())
		return store.Error(err)
	}

	return nil
}

// CreateCircuitBreakerRule create general circuitbreaker rule
func (c *circuitBreakerStore) CreateCircuitBreakerRule(cbRule *model.CircuitBreakerRule) error {
	dbOp := c.handler

	initCircuitBreakerRule(cbRule)
	if err := c.cleanCircuitBreakerRule(cbRule.ID); err != nil {
		log.Errorf("[Store][circuitBreaker] clean circuit breaker rule(%s) err: %s",
			cbRule.ID, err.Error())
		return store.Error(err)
	}
	if err := dbOp.SaveValue(tblCircuitBreakerRule, cbRule.ID, cbRule); err != nil {
		log.Errorf("[Store][circuitBreaker] create circuit breaker(%s, %s) err: %s",
			cbRule.ID, cbRule.Name, err.Error())
		return store.Error(err)
	}

	return nil
}

// UpdateCircuitBreakerRule update general circuitbreaker rule
func (c *circuitBreakerStore) UpdateCircuitBreakerRule(cbRule *model.CircuitBreakerRule) error {
	dbOp := c.handler
	properties := map[string]interface{}{
		CommonFieldName:        cbRule.Name,
		CommonFieldNamespace:   cbRule.Namespace,
		CommonFieldRevision:    cbRule.Revision,
		CommonFieldDescription: cbRule.Description,
		CommonFieldModifyTime:  time.Now(),
		CbFieldLevel:           cbRule.Level,
		CbFieldSrcService:      cbRule.SrcService,
		CbFieldSrcNamespace:    cbRule.SrcNamespace,
		CbFieldDstService:      cbRule.DstService,
		CbFieldDstNamespace:    cbRule.DstNamespace,
		CbFieldDstMethod:       cbRule.DstMethod,
		CbFieldRule:            cbRule.Rule,
	}
	if cbRule.Enable {
		properties[CommonFieldEnableTime] = time.Now()
	} else {
		properties[CommonFieldEnableTime] = time.Unix(0, 0)
	}
	if err := dbOp.UpdateValue(tblCircuitBreakerRule, cbRule.ID, properties); err != nil {
		log.Errorf("[Store][CircuitBreaker] update rule(%s) exec err: %s", cbRule.ID, err.Error())
		return store.Error(err)
	}
	return nil
}

// DeleteCircuitBreakerRule delete general circuitbreaker rule
func (c *circuitBreakerStore) DeleteCircuitBreakerRule(id string) error {
	handler := c.handler
	return handler.Execute(true, func(tx *bolt.Tx) error {

		properties := make(map[string]interface{})
		properties[CommonFieldValid] = false
		properties[CommonFieldModifyTime] = time.Now()

		if err := updateValue(tx, tblCircuitBreakerRule, id, properties); err != nil {
			log.Errorf("[Store][CircuitBreaker] delete rule(%s) err: %s", id, err.Error())
			return err
		}

		return nil
	})
}

// getCircuitBreakerRuleWithID 根据规则ID拉取熔断规则
func (c *circuitBreakerStore) getCircuitBreakerRuleWithID(id string) (*model.CircuitBreakerRule, error) {
	if id == "" {
		return nil, ErrBadParam
	}

	handler := c.handler
	result, err := handler.LoadValues(tblCircuitBreakerRule, []string{id}, &model.CircuitBreakerRule{})

	if err != nil {
		log.Errorf("[Store][boltdb] get rate limit fail : %s", err.Error())
		return nil, err
	}

	if len(result) > 1 {
		return nil, ErrMultipleResult
	}

	if len(result) == 0 {
		return nil, nil
	}

	cbRule := result[id].(*model.CircuitBreakerRule)
	if cbRule.Valid {
		return cbRule, nil
	}

	return nil, nil
}

// HasCircuitBreakerRule check circuitbreaker rule exists
func (c *circuitBreakerStore) HasCircuitBreakerRule(id string) (bool, error) {
	cbRule, err := c.getCircuitBreakerRuleWithID(id)
	if nil != err {
		return false, err
	}
	return cbRule != nil, nil
}

// HasCircuitBreakerRuleByName check circuitbreaker rule exists for name
func (c *circuitBreakerStore) HasCircuitBreakerRuleByName(name string, namespace string) (bool, error) {
	filter := map[string]string{
		exactName:   name,
		"namespace": namespace,
	}
	total, _, err := c.GetCircuitBreakerRules(filter, 0, 10)
	if nil != err {
		return false, err
	}
	return total > 0, nil
}

// HasCircuitBreakerRuleByNameExcludeId check circuitbreaker rule exists for name not this id
func (c *circuitBreakerStore) HasCircuitBreakerRuleByNameExcludeId(
	name string, namespace string, id string) (bool, error) {
	filter := map[string]string{
		exactName:   name,
		"namespace": namespace,
		excludeId:   id,
	}
	total, _, err := c.GetCircuitBreakerRules(filter, 0, 10)
	if nil != err {
		return false, err
	}
	return total > 0, nil
}

var (
	cbSearchFields = []string{CommonFieldID, CommonFieldName, CommonFieldNamespace, CommonFieldDescription,
		CbFieldLevel, CbFieldSrcService, CbFieldSrcNamespace, CbFieldDstService, CbFieldDstNamespace,
		CbFieldDstMethod, CommonFieldEnable, CommonFieldValid,
	}
	cbBlurSearchFields = map[string]bool{
		CommonFieldName:        true,
		CommonFieldDescription: true,
		CbFieldSrcService:      true,
		CbFieldDstService:      true,
		CbFieldDstMethod:       true,
	}
)

// GetCircuitBreakerRules get all circuitbreaker rules by query and limit
func (c *circuitBreakerStore) GetCircuitBreakerRules(
	filter map[string]string, offset uint32, limit uint32) (uint32, []*model.CircuitBreakerRule, error) {
	svc, hasSvc := filter[svcSpecificQueryKeyService]
	delete(filter, svcSpecificQueryKeyService)
	svcNs, hasSvcNs := filter[svcSpecificQueryKeyNamespace]
	delete(filter, svcSpecificQueryKeyNamespace)
	exactNameValue, hasExactName := filter[exactName]
	delete(filter, exactName)
	excludeIdValue, hasExcludeId := filter[excludeId]
	delete(filter, excludeId)
	delete(filter, "brief")
	lowerFilter := make(map[string]string, len(filter))
	for k, v := range filter {
		lowerFilter[strings.ToLower(k)] = v
	}
	result, err := c.handler.LoadValuesByFilter(tblCircuitBreakerRule, cbSearchFields, &model.CircuitBreakerRule{},
		func(m map[string]interface{}) bool {
			validVal, ok := m[CommonFieldValid]
			if ok && !validVal.(bool) {
				return false
			}
			if hasSvcNs {
				srcNsValue := m[CbFieldSrcNamespace]
				dstNsValue := m[CbFieldDstNamespace]
				if !((srcNsValue == "*" || srcNsValue == svcNs) || (dstNsValue == "*" || dstNsValue == svcNs)) {
					return false
				}
			}
			if hasSvc {
				srcSvcValue := m[CbFieldSrcService]
				dstSvcValue := m[CbFieldDstService]
				if !((srcSvcValue == svc || srcSvcValue == "*") || (dstSvcValue == svc || dstSvcValue == "*")) {
					return false
				}
			}
			if hasExactName {
				if exactNameValue != m[CommonFieldName] {
					return false
				}
			}
			if hasExcludeId {
				if excludeIdValue == m[CBFieldNameID] {
					return false
				}
			}
			if len(lowerFilter) == 0 {
				return true
			}
			var matched = true
			for fieldKey, fieldValue := range m {
				lowerKey := strings.ToLower(fieldKey)
				filterValue, ok := lowerFilter[lowerKey]
				if !ok {
					continue
				}
				_, isBlur := cbBlurSearchFields[fieldKey]
				if isBlur {
					if !strings.Contains(fieldValue.(string), filterValue) {
						matched = false
						break
					}
				} else if fieldKey == CommonFieldEnable {
					filterEnable, _ := strconv.ParseBool(filterValue)
					if filterEnable != fieldValue.(bool) {
						matched = false
						break
					}
				} else if fieldKey == CbFieldLevel {
					levels := strings.Split(filterValue, ",")
					var inLevel = false
					for _, level := range levels {
						levelInt, _ := strconv.Atoi(level)
						if int64(levelInt) == fieldValue.(int64) {
							inLevel = true
							break
						}
					}
					if !inLevel {
						matched = false
						break
					}
				} else {
					if filterValue != fieldValue.(string) {
						matched = false
						break
					}
				}
			}
			return matched
		})
	if nil != err {
		return 0, nil, err
	}
	out := make([]*model.CircuitBreakerRule, 0, len(result))
	for _, value := range result {
		out = append(out, value.(*model.CircuitBreakerRule))
	}
	return uint32(len(out)), sublistCircuitBreakerRules(out, offset, limit), nil
}

func sublistCircuitBreakerRules(cbRules []*model.CircuitBreakerRule, offset, limit uint32) []*model.CircuitBreakerRule {
	beginIndex := offset
	endIndex := beginIndex + limit
	totalCount := uint32(len(cbRules))
	// handle invalid offset, limit
	if totalCount == 0 {
		return cbRules
	}
	if beginIndex >= endIndex {
		return cbRules
	}
	if beginIndex >= totalCount {
		return cbRules
	}
	if endIndex > totalCount {
		endIndex = totalCount
	}

	sort.Slice(cbRules, func(i, j int) bool {
		// sort by modify time
		if cbRules[i].ModifyTime.After(cbRules[j].ModifyTime) {
			return true
		} else if cbRules[i].ModifyTime.Before(cbRules[j].ModifyTime) {
			return false
		}
		return strings.Compare(cbRules[i].ID, cbRules[j].ID) < 0
	})

	return cbRules[beginIndex:endIndex]
}

// GetCircuitBreakerRulesForCache get increment circuitbreaker rules
func (c *circuitBreakerStore) GetCircuitBreakerRulesForCache(
	mtime time.Time, firstUpdate bool) ([]*model.CircuitBreakerRule, error) {
	handler := c.handler

	if firstUpdate {
		mtime = time.Time{}
	}

	results, err := handler.LoadValuesByFilter(
		tblCircuitBreakerRule, []string{CommonFieldModifyTime}, &model.CircuitBreakerRule{},
		func(m map[string]interface{}) bool {
			mt := m[CommonFieldModifyTime].(time.Time)
			isAfter := !mt.Before(mtime)
			return isAfter
		})

	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return []*model.CircuitBreakerRule{}, nil
	}

	out := make([]*model.CircuitBreakerRule, 0, len(results))
	for _, value := range results {
		out = append(out, value.(*model.CircuitBreakerRule))
	}

	return out, nil
}

// EnableCircuitBreakerRule enable specific circuitbreaker rule
func (c *circuitBreakerStore) EnableCircuitBreakerRule(cbRule *model.CircuitBreakerRule) error {
	handler := c.handler
	return handler.Execute(true, func(tx *bolt.Tx) error {
		properties := make(map[string]interface{})
		properties[CommonFieldEnable] = cbRule.Enable
		properties[CommonFieldRevision] = cbRule.Revision
		properties[CommonFieldModifyTime] = time.Now()
		if cbRule.Enable {
			properties[CommonFieldEnableTime] = time.Now()
		} else {
			properties[CommonFieldEnableTime] = time.Unix(0, 0)
		}
		// create ratelimit_config
		if err := updateValue(tx, tblCircuitBreakerRule, cbRule.ID, properties); err != nil {
			log.Errorf("[Store][RateLimit] update circuitbreaker rule(%s) err: %s",
				cbRule.ID, err.Error())
			return err
		}
		return nil
	})
}
