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
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/store"
)

type faultDetectStore struct {
	handler BoltHandler
}

const (
	// rule 相关信息以及映射
	tblFaultDetectRule string = "faultdetect_rule"
)

func initFaultDetectRule(cb *model.FaultDetectRule) {
	cb.Valid = true
	cb.CreateTime = time.Now()
	cb.ModifyTime = time.Now()
}

// cleanCircuitBreaker 彻底清理熔断规则
func (c *faultDetectStore) cleanFaultDetectRule(id string) error {
	if err := c.handler.DeleteValues(tblFaultDetectRule, []string{id}); err != nil {
		log.Errorf("[Store][fault-detect] clean invalid fault-detect rule(%s) err: %s",
			id, err.Error())
		return store.Error(err)
	}

	return nil
}

// CreateFaultDetectRule create fault detect rule
func (c *faultDetectStore) CreateFaultDetectRule(fdRule *model.FaultDetectRule) error {
	dbOp := c.handler

	initFaultDetectRule(fdRule)
	if err := c.cleanFaultDetectRule(fdRule.ID); err != nil {
		log.Errorf("[Store][fault-detect] clean fault-detect rule(%s) err: %s",
			fdRule.ID, err.Error())
		return store.Error(err)
	}
	if err := dbOp.SaveValue(tblFaultDetectRule, fdRule.ID, fdRule); err != nil {
		log.Errorf("[Store][fault-detect] create fault-detect(%s, %s) err: %s",
			fdRule.ID, fdRule.Name, err.Error())
		return store.Error(err)
	}

	return nil
}

// UpdateFaultDetectRule update fault detect rule
func (c *faultDetectStore) UpdateFaultDetectRule(fdRule *model.FaultDetectRule) error {
	dbOp := c.handler
	fdRule.Valid = true
	fdRule.ModifyTime = time.Now()

	if err := dbOp.SaveValue(tblFaultDetectRule, fdRule.ID, fdRule); err != nil {
		log.Errorf("[Store][fault-detect] update rule(%s) exec err: %s", fdRule.ID, err.Error())
		return store.Error(err)
	}

	return nil
}

// DeleteFaultDetectRule delete fault detect rule
func (c *faultDetectStore) DeleteFaultDetectRule(id string) error {
	handler := c.handler
	return handler.Execute(true, func(tx *bolt.Tx) error {

		properties := make(map[string]interface{})
		properties[CommonFieldValid] = false
		properties[CommonFieldModifyTime] = time.Now()

		if err := updateValue(tx, tblFaultDetectRule, id, properties); err != nil {
			log.Errorf("[Store][fault-detect] delete rule(%s) err: %s", id, err.Error())
			return err
		}

		return nil
	})
}

func (c *faultDetectStore) getFaultDetectRuleWithID(id string) (*model.FaultDetectRule, error) {
	if id == "" {
		return nil, ErrBadParam
	}

	handler := c.handler
	result, err := handler.LoadValues(tblFaultDetectRule, []string{id}, &model.FaultDetectRule{})

	if err != nil {
		log.Errorf("[Store][fault-detect] get rule fail : %s", err.Error())
		return nil, err
	}

	if len(result) > 1 {
		return nil, ErrMultipleResult
	}

	if len(result) == 0 {
		return nil, nil
	}

	cbRule := result[id].(*model.FaultDetectRule)
	if cbRule.Valid {
		return cbRule, nil
	}

	return nil, nil
}

// HasFaultDetectRule check fault detect rule exists
func (c *faultDetectStore) HasFaultDetectRule(id string) (bool, error) {
	cbRule, err := c.getFaultDetectRuleWithID(id)
	if nil != err {
		return false, err
	}
	return cbRule != nil, nil
}

// HasFaultDetectRuleByName check fault detect rule exists by name
func (c *faultDetectStore) HasFaultDetectRuleByName(name string, namespace string) (bool, error) {
	filter := map[string]string{
		exactName:   name,
		"namespace": namespace,
	}
	total, _, err := c.GetFaultDetectRules(filter, 0, 10)
	if nil != err {
		return false, err
	}
	return total > 0, nil
}

// HasFaultDetectRuleByNameExcludeId check fault detect rule exists by name not this id
func (c *faultDetectStore) HasFaultDetectRuleByNameExcludeId(name string, namespace string, id string) (bool, error) {
	filter := map[string]string{
		exactName:   name,
		"namespace": namespace,
		excludeId:   id,
	}
	total, _, err := c.GetFaultDetectRules(filter, 0, 10)
	if nil != err {
		return false, err
	}
	return total > 0, nil
}

const (
	fdFieldDstService   = "DstService"
	fdFieldDstNamespace = "DstNamespace"
	fdFieldDstMethod    = "DstMethod"
)

var (
	fdSearchFields = []string{
		CommonFieldID, CommonFieldName, CommonFieldNamespace, CommonFieldDescription, fdFieldDstService,
		fdFieldDstNamespace, fdFieldDstMethod, CommonFieldEnable, CommonFieldValid,
	}
	fdBlurSearchFields = map[string]bool{
		CommonFieldName:        true,
		CommonFieldDescription: true,
		fdFieldDstService:      true,
		fdFieldDstNamespace:    true,
		fdFieldDstMethod:       true,
	}
)

// GetFaultDetectRules get all circuitbreaker rules by query and limit
func (c *faultDetectStore) GetFaultDetectRules(
	filter map[string]string, offset uint32, limit uint32) (uint32, []*model.FaultDetectRule, error) {
	svc, hasSvc := filter[svcSpecificQueryKeyService]
	delete(filter, svcSpecificQueryKeyService)
	svcNs, hasSvcNs := filter[svcSpecificQueryKeyNamespace]
	delete(filter, svcSpecificQueryKeyNamespace)
	exactNameValue, hasExactName := filter[exactName]
	delete(filter, exactName)
	excludeIdValue, hasExcludeId := filter[excludeId]
	delete(filter, excludeId)
	delete(filter, "brief")
	result, err := c.handler.LoadValuesByFilter(tblFaultDetectRule, fdSearchFields, &model.FaultDetectRule{},
		func(m map[string]interface{}) bool {
			validVal, ok := m[CommonFieldValid]
			if ok && !validVal.(bool) {
				return false
			}
			if hasSvc && hasSvcNs {
				dstServiceValue := m[fdFieldDstService]
				dstNamespaceValue := m[fdFieldDstNamespace]
				if !(dstServiceValue == svc && dstNamespaceValue == svcNs) {
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
			if len(filter) == 0 {
				return true
			}
			var matched = true
			for fieldKey, fieldValue := range m {
				lowerKey := strings.ToLower(fieldKey)
				filterValue, ok := filter[lowerKey]
				if !ok {
					continue
				}
				_, isBlur := fdBlurSearchFields[fieldKey]
				if isBlur {
					if !strings.Contains(fieldValue.(string), filterValue) {
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
	out := make([]*model.FaultDetectRule, 0, len(result))
	for _, value := range result {
		out = append(out, value.(*model.FaultDetectRule))
	}
	return uint32(len(out)), sublistFaultDetectRules(out, offset, limit), nil
}

func sublistFaultDetectRules(cbRules []*model.FaultDetectRule, offset, limit uint32) []*model.FaultDetectRule {
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

// GetFaultDetectRulesForCache get increment circuitbreaker rules
func (c *faultDetectStore) GetFaultDetectRulesForCache(
	mtime time.Time, firstUpdate bool) ([]*model.FaultDetectRule, error) {
	handler := c.handler

	if firstUpdate {
		mtime = time.Time{}
	}

	results, err := handler.LoadValuesByFilter(
		tblFaultDetectRule, []string{CommonFieldModifyTime}, &model.FaultDetectRule{},
		func(m map[string]interface{}) bool {
			mt := m[CommonFieldModifyTime].(time.Time)
			isAfter := !mt.Before(mtime)
			return isAfter
		})

	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return []*model.FaultDetectRule{}, nil
	}

	out := make([]*model.FaultDetectRule, 0, len(results))
	for _, value := range results {
		out = append(out, value.(*model.FaultDetectRule))
	}

	return out, nil
}
