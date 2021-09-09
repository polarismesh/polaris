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

package boltdbStore

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/store"
)

const (
	// rule 相关信息以及映射
	DataTypeCircuitBreaker string = "circuitbreaker_rule"

	// relation 相关信息以及映射信息
	DataTypeCircuitBreakerRelation string = "circuitbreaker_rule_relation"
	VersionForMaster               string = "master"
)

type circuitBreakerStore struct {
	handler      BoltHandler
	ruleLock     *sync.RWMutex // 负责 DataTypeCircuitBreaker 以及 DataTypeRuleIdVersionMapping
	relationLock *sync.RWMutex // 负责 DataTypeCircuitBreakerRelation 以及 DataTypeRuleServiceMapping
}

// CreateCircuitBreaker 新增熔断规则
func (c *circuitBreakerStore) CreateCircuitBreaker(cb *model.CircuitBreaker) error {
	dbOp := c.handler

	lock := c.ruleLock
	lock.Lock()
	defer lock.Unlock()

	if err := dbOp.SaveValue(DataTypeCircuitBreaker, c.buildKey(cb.ID, cb.Version), cb); err != nil {
		log.Errorf("[Store][circuitBreaker] create circuit breaker(%s, %s, %s) err: %s",
			cb.ID, cb.Name, cb.Version, err.Error())
		return store.Error(err)
	}
	return nil
}

// TagCircuitBreaker 标记熔断规则
func (c *circuitBreakerStore) TagCircuitBreaker(cb *model.CircuitBreaker) error {

	if err := c.tagCircuitBreaker(cb); err != nil {
		log.Errorf("[Store][circuitBreaker] create tag for circuit breaker(%s, %s) err: %s",
			cb.ID, cb.Version, err.Error())
		return store.Error(err)
	}

	return nil
}

// tagCircuitBreaker
func (c *circuitBreakerStore) tagCircuitBreaker(cb *model.CircuitBreaker) error {
	// first : Ensure that the master rule exists

	dbOp := c.handler
	key := c.buildKey(cb.ID, cb.Version)

	_, err := c.GetCircuitBreaker(cb.ID, VersionForMaster)
	if err != nil {
		log.Errorf("[Store][CircuitBreaker] get tag rule id(%s) version(%s) err : %s", cb.ID, VersionForMaster, err.Error())
		return store.Error(err)
	}

	tNow := time.Now()

	cb.CreateTime = tNow
	cb.ModifyTime = tNow

	lock := c.ruleLock
	lock.Lock()
	defer lock.Unlock()

	if err := dbOp.SaveValue(DataTypeCircuitBreaker, key, cb); err != nil {
		log.Errorf("[Store][circuitBreaker] tag rule breaker(%s, %s, %s) err: %s",
			cb.ID, cb.Name, cb.Version, err.Error())
		return store.Error(err)
	}

	return nil
}

// ReleaseCircuitBreaker 发布熔断规则
func (c *circuitBreakerStore) ReleaseCircuitBreaker(cbr *model.CircuitBreakerRelation) error {

	tNow := time.Now()

	cbr.CreateTime = tNow
	cbr.ModifyTime = tNow

	if err := c.releaseCircuitBreaker(cbr); err != nil {
		log.Errorf("[Store][CircuitBreaker] release rule err: %s", err.Error())
		return store.Error(err)
	}

	return nil
}

/**
 * @brief 发布熔断规则的内部函数
 * @note 可能存在服务的规则，由旧的更新到新的场景
 */
func (c *circuitBreakerStore) releaseCircuitBreaker(cbr *model.CircuitBreakerRelation) error {
	// 上层调用者保证 service 是已经存在的

	dbOp := c.handler

	if tRule, _ := c.GetCircuitBreaker(cbr.RuleID, cbr.RuleVersion); tRule == nil {
		return store.NewStatusError(store.NotFoundMasterConfig, "not found tag config")
	}

	lock := c.relationLock
	lock.Lock()
	defer lock.Unlock()

	// 如果之前存在，就直接覆盖上一次的 release 信息
	if err := dbOp.SaveValue(DataTypeCircuitBreakerRelation, cbr.ServiceID, cbr); err != nil {
		log.Errorf("[Store][circuitBreaker] tag rule relation(%s, %s, %s) err: %s",
			cbr.ServiceID, cbr.RuleID, cbr.RuleVersion, err.Error())
		return store.Error(err)
	}
	return nil
}

// UnbindCircuitBreaker 解绑熔断规则
func (c *circuitBreakerStore) UnbindCircuitBreaker(serviceID, ruleID, ruleVersion string) error {

	dbOp := c.handler

	lock := c.ruleLock
	lock.Lock()
	defer lock.Unlock()

	// 删除某个服务的熔断规则

	if err := dbOp.DeleteValues(DataTypeCircuitBreakerRelation, []string{serviceID}); err != nil {
		log.Errorf("[Store][circuitBreaker] tag rule relation(%s, %s, %s) err: %s",
			serviceID, ruleID, ruleVersion, err.Error())
		return store.Error(err)
	}
	return nil
}

// DeleteTagCircuitBreaker 删除已标记熔断规则
func (c *circuitBreakerStore) DeleteTagCircuitBreaker(id string, version string) error {

	dbOp := c.handler

	lock := c.ruleLock
	lock.Lock()
	defer lock.Unlock()

	if err := dbOp.DeleteValues(DataTypeCircuitBreaker, []string{c.buildKey(id, version)}); err != nil {
		log.Errorf("[Store][circuitBreaker] delete tag rule(%s, %s) err: %s", id, version, err.Error())
		return store.Error(err)
	}

	return nil
}

// DeleteMasterCircuitBreaker 删除master熔断规则
func (c *circuitBreakerStore) DeleteMasterCircuitBreaker(id string) error {
	return c.DeleteTagCircuitBreaker(id, VersionForMaster)
}

// UpdateCircuitBreaker 修改熔断规则
func (c *circuitBreakerStore) UpdateCircuitBreaker(cb *model.CircuitBreaker) error {

	dbOp := c.handler

	lock := c.ruleLock
	lock.Lock()
	defer lock.Unlock()

	if err := dbOp.SaveValue(DataTypeCircuitBreaker, c.buildKey(cb.ID, cb.Version), cb); err != nil {
		log.Errorf("[Store][CircuitBreaker] update rule(%s,%s) exec err: %s", cb.ID, cb.Version, err.Error())
		return store.Error(err)
	}

	return nil
}

// GetCircuitBreaker 获取熔断规则
func (c *circuitBreakerStore) GetCircuitBreaker(id, version string) (*model.CircuitBreaker, error) {

	lock := c.ruleLock
	lock.RLock()
	defer lock.RUnlock()

	return c.lockfreeGetCircuitBreaker(id, version)
}

// lockfreeGetCircuitBreaker 获取熔断规则
func (c *circuitBreakerStore) lockfreeGetCircuitBreaker(id, version string) (*model.CircuitBreaker, error) {

	dbOp := c.handler

	cbKey := c.buildKey(id, version)

	result, err := dbOp.LoadValues(DataTypeCircuitBreaker, []string{cbKey}, &model.CircuitBreaker{})
	if err != nil {
		log.Errorf("[Store][CircuitBreaker] get tag rule id(%s) version(%s) err : %s", id, version, err.Error())
		return nil, store.Error(err)
	}

	if len(result) != 1 {
		return nil, store.NewStatusError(store.NotFoundMasterConfig, "not found tag config")
	}
	return result[cbKey].(*model.CircuitBreaker), nil
}

// GetCircuitBreakerVersions 获取熔断规则的所有版本
func (c *circuitBreakerStore) GetCircuitBreakerVersions(id string) ([]string, error) {

	dbOp := c.handler

	lock := c.ruleLock
	lock.RLock()

	results, err := dbOp.LoadValuesByFilter(DataTypeCircuitBreaker, []string{"ID"}, &model.CircuitBreaker{}, func(m map[string]interface{}) bool {
		mV := m["ID"]
		return strings.Compare(mV.(string), id) == 0
	})
	if err != nil {
		lock.RUnlock()
		log.Errorf("[Store][CircuitBreaker] get rule_id(%s) links version err : %s", id, err.Error())
		return nil, store.Error(err)
	}
	lock.RUnlock()

	ans := make([]string, len(results))

	pos := 0
	for _, val := range results {
		record := val.(*model.CircuitBreaker)
		ans[pos] = record.Version
		pos++
	}

	return ans, nil
}

// GetCircuitBreakerMasterRelation 获取熔断规则master版本的绑定关系
func (c *circuitBreakerStore) GetCircuitBreakerMasterRelation(ruleID string) ([]*model.CircuitBreakerRelation, error) {
	return c.GetCircuitBreakerRelation(ruleID, VersionForMaster)
}

// GetCircuitBreakerRelation 获取已标记熔断规则的绑定关系
func (c *circuitBreakerStore) GetCircuitBreakerRelation(
	ruleID, ruleVersion string) ([]*model.CircuitBreakerRelation, error) {
	dbOp := c.handler

	lock := c.relationLock
	lock.RLock()
	defer lock.RUnlock()

	// first: get rule_id => service_ids
	serviceIds, err := c.getRuleToServiceMap(ruleID)
	if err != nil {
		log.Errorf("[Store][CircuitBreaker] get rule_id(%s) links service_ids err : %s", ruleID, err.Error())
		return nil, store.Error(err)
	}

	// second: get uniq keys
	ids := make([]string, len(serviceIds))
	pos := 0
	for serviceId := range serviceIds {
		ids[pos] = serviceId
		pos++
	}

	// third: batch get relation records
	relations := make([]*model.CircuitBreakerRelation, 0)

	results, err := dbOp.LoadValues(DataTypeCircuitBreakerRelation, ids, &model.CircuitBreakerRelation{})
	if err != nil {
		log.Errorf("[Store][CircuitBreaker] get rule_id(%s) relations err : %s", ruleID, err.Error())
		return nil, store.Error(err)
	}

	for _, val := range results {
		record := val.(*model.CircuitBreakerRelation)
		if strings.Compare(ruleVersion, record.RuleVersion) != 0 {
			continue
		}
		relations = append(relations, record)
	}

	return relations, nil
}

// GetCircuitBreakerForCache 根据修改时间拉取增量熔断规则
func (c *circuitBreakerStore) GetCircuitBreakerForCache(
	mtime time.Time, firstUpdate bool) ([]*model.ServiceWithCircuitBreaker, error) {

	dbOp := c.handler

	lock := c.ruleLock
	lock.RLock()

	relations, err := dbOp.LoadValuesByFilter(DataTypeCircuitBreakerRelation, []string{"ModifyTime"}, &model.CircuitBreakerRelation{}, func(m map[string]interface{}) bool {
		mt := m["ModifyTime"].(time.Time)
		isAfter := mt.After(mtime)
		return isAfter
	})
	if err != nil {
		lock.RUnlock()
		return nil, store.Error(err)
	}

	serviceToCbKey := make(map[string]string, 0)

	cbKeys := make([]string, 0)
	for k, v := range relations {
		rel := v.(*model.CircuitBreakerRelation)
		cbKeys = append(cbKeys, c.buildKey(rel.RuleID, rel.RuleVersion))
		serviceToCbKey[k] = c.buildKey(rel.RuleID, rel.RuleVersion)
	}

	cbs, err := dbOp.LoadValues(DataTypeCircuitBreaker, cbKeys, &model.CircuitBreaker{})

	if err != nil {
		lock.RUnlock()
		return nil, store.Error(err)
	}

	lock.RUnlock()

	results := make([]*model.ServiceWithCircuitBreaker, 0)
	for serviceId, cbKey := range serviceToCbKey {
		results = append(results, &model.ServiceWithCircuitBreaker{
			ServiceID:      serviceId,
			CircuitBreaker: cbs[cbKey].(*model.CircuitBreaker),
			CreateTime:     relations[serviceId].(*model.CircuitBreakerRelation).CreateTime,
			ModifyTime:     relations[serviceId].(*model.CircuitBreakerRelation).ModifyTime,
		})
	}

	return results, nil
}

// ListMasterCircuitBreakers 获取master熔断规则
func (c *circuitBreakerStore) ListMasterCircuitBreakers(
	filters map[string]string, offset uint32, limit uint32) (*model.CircuitBreakerDetail, error) {

	dbOp := c.handler

	fields := utils.CollectFilterFields(filters)
	fields = append(fields, "Version")

	lock := c.ruleLock
	lock.RLock()

	results, err := dbOp.LoadValuesByFilter(DataTypeCircuitBreaker, fields, &model.CircuitBreaker{}, func(m map[string]interface{}) bool {
		val := m["Version"].(string)
		if strings.Compare(val, VersionForMaster) != 0 {
			return false
		}
		for k, v := range filters {
			qV := m[k]
			if !reflect.DeepEqual(qV, v) {
				return false
			}
		}
		return true
	})
	if err != nil {
		lock.RUnlock()
		return nil, store.Error(err)
	}
	lock.RUnlock()

	// 内存中进行排序分页
	cbSlice := make([]*model.CircuitBreakerInfo, 0)
	for _, v := range results {
		record := v.(*model.CircuitBreaker)
		cbSlice = append(cbSlice, convertCircuitBreakerToInfo(record))
	}

	sort.Slice(cbSlice, func(i, j int) bool {
		a := cbSlice[i]
		b := cbSlice[j]
		return a.CircuitBreaker.ModifyTime.Before(b.CircuitBreaker.ModifyTime)
	})

	out := &model.CircuitBreakerDetail{
		Total:               uint32(len(results)),
		CircuitBreakerInfos: cbSlice[offset:int(math.Min(float64(offset+limit), float64(len(results))))],
	}

	return out, nil
}

// 获取已发布规则
func (c *circuitBreakerStore) ListReleaseCircuitBreakers(
	filters map[string]string, offset, limit uint32) (*model.CircuitBreakerDetail, error) {

	// 内存中进行分页
	dbOp := c.handler

	lock := c.ruleLock
	lock.RLock()

	results, err := dbOp.LoadValuesByFilter(DataTypeCircuitBreaker, utils.CollectFilterFields(filters), &model.CircuitBreaker{}, func(m map[string]interface{}) bool {
		for k, v := range filters {
			qV := m[k]
			if !reflect.DeepEqual(qV, v) {
				return false
			}
		}
		return true
	})
	if err != nil {
		lock.RUnlock()
		return nil, store.Error(err)
	}
	lock.RUnlock()

	// 内存中进行排序分页
	cbSlice := make([]*model.CircuitBreakerInfo, 0)
	for _, v := range results {
		record := v.(*model.CircuitBreaker)
		cbSlice = append(cbSlice, convertCircuitBreakerToInfo(record))
	}

	sort.Slice(cbSlice, func(i, j int) bool {
		a := cbSlice[i]
		b := cbSlice[j]
		return a.CircuitBreaker.ModifyTime.Before(b.CircuitBreaker.ModifyTime)
	})

	out := &model.CircuitBreakerDetail{
		Total:               uint32(len(results)),
		CircuitBreakerInfos: cbSlice[offset:int(math.Min(float64(offset+limit), float64(len(results))))],
	}

	return out, nil
}

// 根据服务获取熔断规则
func (c *circuitBreakerStore) GetCircuitBreakersByService(
	name string, namespace string) (*model.CircuitBreaker, error) {
	//TODO
	return nil, nil
}

func (c *circuitBreakerStore) getRuleToVersionMap(ruleID string) (map[string]struct{}, error) {

	dbOp := c.handler

	// 保存 rule_id => service_id 的映射
	mapKey := c.buildMapKey(ruleID)
	result, err := dbOp.LoadValuesByFilter(DataTypeCircuitBreaker, []string{"id"}, &model.CircuitBreaker{}, func(m map[string]interface{}) bool {
		id := m["id"].(string)
		return strings.Compare(id, ruleID) == 0
	})

	if err != nil {
		log.Errorf("[Store][CircuitBreaker] get tag rule id(%s) err : %s", ruleID, err.Error())
		return nil, store.Error(err)
	}

	if len(result) != 1 {
		return nil, store.NewStatusError(store.NotFoundMasterConfig, "not found tag config")
	}

	ruleVerMap := result[mapKey].(map[string]struct{})
	return ruleVerMap, nil
}

func (c *circuitBreakerStore) getRuleToServiceMap(ruleID string) (map[string]struct{}, error) {

	dbOp := c.handler

	// 保存 rule_id => service_id 的映射
	mapKey := c.buildMapKey(ruleID)
	result, err := dbOp.LoadValuesByFilter(DataTypeCircuitBreakerRelation, []string{"RuleID"}, &model.CircuitBreaker{}, func(m map[string]interface{}) bool {
		id := m["RuleID"].(string)
		return strings.Compare(id, ruleID) == 0
	})
	if err != nil {
		log.Errorf("[Store][CircuitBreaker] get tag rule id(%s) err : %s", ruleID, err.Error())
		return nil, store.Error(err)
	}

	if len(result) != 1 {
		return nil, store.NewStatusError(store.NotFoundMasterConfig, "not found tag config")
	}

	ruleSerMap := result[mapKey].(map[string]struct{})

	return ruleSerMap, nil
}

func (c *circuitBreakerStore) buildKey(id, version string) string {
	return fmt.Sprintf("%s_%s", id, version)
}

func (c *circuitBreakerStore) buildMapKey(id string) string {
	return fmt.Sprintf("map_%s", id)
}

func convertCircuitBreakerToInfo(record *model.CircuitBreaker) *model.CircuitBreakerInfo {
	return &model.CircuitBreakerInfo{
		CircuitBreaker: &model.CircuitBreaker{
			ID:         record.ID,
			Version:    record.Version,
			Name:       record.Name,
			Namespace:  record.Namespace,
			Business:   record.Business,
			Department: record.Department,
			Comment:    record.Comment,
			Inbounds:   record.Inbounds,
			Outbounds:  record.Outbounds,
			Token:      record.Token,
			Owner:      record.Owner,
			Revision:   record.Revision,
			CreateTime: record.CreateTime,
			ModifyTime: record.ModifyTime,
		},
		Services: []*model.Service{},
	}
}
