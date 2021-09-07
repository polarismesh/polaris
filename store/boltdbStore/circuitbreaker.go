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
	DataTypeCircuitBreaker         string = "circuitbreaker_rule"
	DataTypeRuleIdVersionMapping   string = "circuitbreaker_rule_id_to_version"
	DataTypeCircuitBreakerRelation string = "circuitbreaker_rule_relation"
	DataTypeRuleServiceMapping     string = "circuitbreaker_rule_to_service"
	VersionForMaster               string = "master"
)

type circuitBreakerStore struct {
	handler      BoltHandler
	ruleLock     *sync.RWMutex
	relationLock *sync.RWMutex
}

// CreateCircuitBreaker 新增熔断规则
func (c *circuitBreakerStore) CreateCircuitBreaker(cb *model.CircuitBreaker) error {
	if err := c.cleanCircuitBreaker(cb.ID, cb.Version); err != nil {
		return err
	}

	dbOp := c.handler

	lock := c.ruleLock
	lock.Lock()
	defer lock.Unlock()

	if err := dbOp.SaveValue(DataTypeCircuitBreaker, c.buildKey(cb.ID, cb.Version), cb); err != nil {
		log.Errorf("[Store][circuitBreaker] create circuit breaker(%s, %s, %s) err: %s",
			cb.ID, cb.Name, cb.Version, err.Error())
		return store.Error(err)
	}

	// TODO save rule_id => versions，如果 BoltHandler 提供了 key 前缀搜索的话，这里就可以省略

	return nil
}

// TagCircuitBreaker 标记熔断规则
func (c *circuitBreakerStore) TagCircuitBreaker(cb *model.CircuitBreaker) error {
	lock := c.ruleLock
	lock.Lock()
	defer lock.Unlock()

	if err := c.tagCircuitBreaker(cb); err != nil {
		log.Errorf("[Store][circuitBreaker] create tag for circuit breaker(%s, %s) err: %s",
			cb.ID, cb.Version, err.Error())
		return store.Error(err)
	}

	return nil
}

// tagCircuitBreaker
func (c *circuitBreakerStore) tagCircuitBreaker(cb *model.CircuitBreaker) error {
	// 需要保证master规则存在

	dbOp := c.handler
	key := c.buildKey(cb.ID, VersionForMaster)

	result, err := dbOp.LoadValues(DataTypeCircuitBreaker, []string{key})
	if err != nil {
		log.Errorf("[Store][CircuitBreaker] get tag rule id(%s) version(%s) err : %s", cb.ID, VersionForMaster, err.Error())
		return store.Error(err)
	}

	if len(result) != 1 {
		return store.NewStatusError(store.NotFoundMasterConfig, "not found master config")
	}

	data := result[key].(*model.CircuitBreaker)
	tNow := time.Now()

	data.CreateTime = tNow
	data.ModifyTime = tNow

	if err := dbOp.SaveValue(DataTypeCircuitBreaker, key, data); err != nil {
		log.Errorf("[Store][circuitBreaker] tag rule breaker(%s, %s, %s) err: %s",
			cb.ID, cb.Name, cb.Version, err.Error())
		return store.Error(err)
	}

	return nil
}

// ReleaseCircuitBreaker 发布熔断规则
func (c *circuitBreakerStore) ReleaseCircuitBreaker(cbr *model.CircuitBreakerRelation) error {
	lock := c.relationLock
	lock.Lock()
	defer lock.Unlock()

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

	// 需要记录 RuleID => ServiceID 的映射关系
	if err := c.saveRuleToServiceMap(cbr.RuleID, cbr.ServiceID); err != nil {
		log.Errorf("[Store][circuitBreaker] save RuleID map ServiceID(%s, %s) err: %s",
			cbr.RuleID, cbr.ServiceID, err.Error())
		return store.Error(err)
	}

	tNow := time.Now()

	cbr.CreateTime = tNow
	cbr.ModifyTime = tNow

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

	// find circuitbreaker_rule_relation
	crbKey := serviceID

	lock := c.ruleLock
	lock.Lock()
	defer lock.Unlock()

	// 删除某个服务的熔断规则

	if err := dbOp.DeleteValues(DataTypeCircuitBreakerRelation, []string{crbKey}); err != nil {
		log.Errorf("[Store][circuitBreaker] tag rule relation(%s, %s, %s) err: %s",
			serviceID, ruleID, ruleVersion, err.Error())
		return store.Error(err)
	}

	// 清除 rule_id => service_id 的映射关系
	if err := c.cancelRuleToServiceMap(ruleID, serviceID); err != nil {
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

	cbKey := c.buildKey(id, version)

	if err := dbOp.DeleteValues(DataTypeCircuitBreaker, []string{cbKey}); err != nil {
		log.Errorf("[Store][circuitBreaker] delete tag rule(%s, %s) err: %s", id, version, err.Error())
		return store.Error(err)
	}

	// TODO remove rule_id => versions

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

	cbKey := c.buildKey(cb.ID, cb.Version)

	if err := dbOp.SaveValue(DataTypeCircuitBreaker, cbKey, cb); err != nil {
		log.Errorf("[Store][CircuitBreaker] update rule(%s,%s) exec err: %s", cb.ID, cb.Version, err.Error())
		return store.Error(err)
	}

	return nil
}

// GetCircuitBreaker 获取熔断规则
func (c *circuitBreakerStore) GetCircuitBreaker(id, version string) (*model.CircuitBreaker, error) {

	lock := c.ruleLock
	lock.Lock()
	defer lock.Unlock()

	return c.lockfreeGetCircuitBreaker(id, version)
}

// lockfreeGetCircuitBreaker 获取熔断规则
func (c *circuitBreakerStore) lockfreeGetCircuitBreaker(id, version string) (*model.CircuitBreaker, error) {

	dbOp := c.handler

	cbKey := c.buildKey(id, version)

	result, err := dbOp.LoadValues(DataTypeCircuitBreaker, []string{cbKey})
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

	idVersionsKey := id

	results, err := dbOp.LoadValues(DataTypeRuleIdVersionMapping, []string{idVersionsKey})
	if err != nil {
		log.Errorf("[Store][CircuitBreaker] get rule_id(%s) links version err : %s", id, err.Error())
		return nil, store.Error(err)
	}

	versions := results[idVersionsKey].(map[string]struct{})

	ans := make([]string, len(versions))

	pos := 0
	for k := range versions {
		ans[pos] = k
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

	// first: get rule_id => service_ids
	results, err := dbOp.LoadValues(DataTypeRuleServiceMapping, []string{ruleID})
	if err != nil {
		log.Errorf("[Store][CircuitBreaker] get rule_id(%s) links service_ids err : %s", ruleID, err.Error())
		return nil, store.Error(err)
	}

	serviceIds := results[ruleID].(map[string]struct{})

	// second: get uniq keys

	ids := make([]string, len(serviceIds))
	pos := 0
	for serviceId := range serviceIds {
		ids[pos] = serviceId
		pos++
	}

	// third: batch get relation records

	relations := make([]*model.CircuitBreakerRelation, 0)

	results, err = dbOp.LoadValues(DataTypeCircuitBreakerRelation, ids)
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
	//TODO
	return nil, nil
}

// ListMasterCircuitBreakers 获取master熔断规则
func (c *circuitBreakerStore) ListMasterCircuitBreakers(
	filters map[string]string, offset uint32, limit uint32) (*model.CircuitBreakerDetail, error) {

	dbOp := c.handler

	results, err := dbOp.LoadValuesByFilter(DataTypeCircuitBreaker, utils.ConvertFilter(filters))
	if err != nil {
		return nil, store.Error(err)
	}

	// 内存中进行排序分页
	cbSlice := make([]*model.CircuitBreakerInfo, 0)
	for _, v := range results {
		record := v.(*model.CircuitBreaker)
		cbSlice = append(cbSlice, &model.CircuitBreakerInfo{
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
		})
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

	return nil, nil
}

// 根据服务获取熔断规则
func (c *circuitBreakerStore) GetCircuitBreakersByService(
	name string, namespace string) (*model.CircuitBreaker, error) {
	//TODO
	return nil, nil
}

/**
 * @brief 清理无效的熔断规则关系
 */
func (c *circuitBreakerStore) cleanCircuitBreakerRelation(cbr *model.CircuitBreakerRelation) error {
	lock := c.relationLock

	lock.Lock()
	defer lock.Unlock()

	return nil
}

/**
 * @brief 彻底清理熔断规则
 */
func (c *circuitBreakerStore) cleanCircuitBreaker(id string, version string) error {
	lock := c.ruleLock

	lock.Lock()
	defer lock.Unlock()

	return nil
}

func (c *circuitBreakerStore) saveRuleToServiceMap(ruleID, serviceID string) error {

	dbOp := c.handler

	// 保存 rule_id => service_id 的映射
	mapKey := c.buildMapKey(ruleID)
	result, err := dbOp.LoadValues(DataTypeRuleServiceMapping, []string{mapKey})
	if err != nil {
		log.Errorf("[Store][CircuitBreaker] get tag rule id(%s) err : %s", ruleID, err.Error())
		return store.Error(err)
	}

	if len(result) != 1 {
		return store.NewStatusError(store.NotFoundMasterConfig, "not found tag config")
	}

	ruleSerMap := result[mapKey].(map[string]struct{})
	ruleSerMap[serviceID] = struct{}{}

	if err := dbOp.SaveValue(DataTypeRuleServiceMapping, mapKey, ruleSerMap); err != nil {
		return store.Error(err)
	}

	return nil
}

func (c *circuitBreakerStore) cancelRuleToServiceMap(ruleID, serviceID string) error {

	dbOp := c.handler

	// 保存 rule_id => service_id 的映射
	mapKey := c.buildMapKey(ruleID)
	result, err := dbOp.LoadValues(DataTypeRuleServiceMapping, []string{mapKey})
	if err != nil {
		log.Errorf("[Store][CircuitBreaker] get tag rule id(%s) err : %s", ruleID, err.Error())
		return store.Error(err)
	}

	if len(result) != 1 {
		return store.NewStatusError(store.NotFoundMasterConfig, "not found tag config")
	}

	ruleSerMap := result[mapKey].(map[string]struct{})
	delete(ruleSerMap, serviceID)

	if err := dbOp.SaveValue(DataTypeRuleServiceMapping, mapKey, ruleSerMap); err != nil {
		return store.Error(err)
	}

	return nil
}

func (c *circuitBreakerStore) buildKey(id, version string) string {
	return fmt.Sprintf("%s_%s", id, version)
}

func (c *circuitBreakerStore) buildMapKey(id string) string {
	return fmt.Sprintf("map_%s", id)
}
