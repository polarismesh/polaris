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
	"github.com/polarismesh/polaris-server/common/model"
	"time"
)

type circuitBreakerStore struct {
	handler BoltHandler
}

// 新增熔断规则
func (c *circuitBreakerStore) CreateCircuitBreaker(circuitBreaker *model.CircuitBreaker) error {
	//TODO
	return nil
}

// 标记熔断规则
func (c *circuitBreakerStore) TagCircuitBreaker(circuitBreaker *model.CircuitBreaker) error {
	//TODO
	return nil
}

// 发布熔断规则
func (c *circuitBreakerStore) ReleaseCircuitBreaker(circuitBreakerRelation *model.CircuitBreakerRelation) error {
	//TODO
	return nil
}

// 解绑熔断规则
func (c *circuitBreakerStore) UnbindCircuitBreaker(serviceID, ruleID, ruleVersion string) error {
	//TODO
	return nil
}

// 删除已标记熔断规则
func (c *circuitBreakerStore) DeleteTagCircuitBreaker(id string, version string) error {
	//TODO
	return nil
}

// 删除master熔断规则
func (c *circuitBreakerStore) DeleteMasterCircuitBreaker(id string) error {
	//TODO
	return nil
}

// 修改熔断规则
func (c *circuitBreakerStore) UpdateCircuitBreaker(circuitBreaker *model.CircuitBreaker) error {
	//TODO
	return nil
}

// 获取熔断规则
func (c *circuitBreakerStore) GetCircuitBreaker(id, version string) (*model.CircuitBreaker, error) {
	//TODO
	return nil, nil
}

// 获取熔断规则的所有版本
func (c *circuitBreakerStore) GetCircuitBreakerVersions(id string) ([]string, error) {
	//TODO
	return nil, nil
}

// 获取熔断规则master版本的绑定关系
func (c *circuitBreakerStore) GetCircuitBreakerMasterRelation(ruleID string) ([]*model.CircuitBreakerRelation, error) {
	//TODO
	return nil, nil
}

// 获取已标记熔断规则的绑定关系
func (c *circuitBreakerStore) GetCircuitBreakerRelation(
	ruleID, ruleVersion string) ([]*model.CircuitBreakerRelation, error) {
	//TODO
	return nil, nil
}

// 根据修改时间拉取增量熔断规则
func (c *circuitBreakerStore) GetCircuitBreakerForCache(
	mtime time.Time, firstUpdate bool) ([]*model.ServiceWithCircuitBreaker, error) {
	//TODO
	return nil, nil
}

// 获取master熔断规则
func (c *circuitBreakerStore) ListMasterCircuitBreakers(
	filters map[string]string, offset uint32, limit uint32) (*model.CircuitBreakerDetail, error) {
	//TODO
	return nil, nil
}

// 获取已发布规则
func (c *circuitBreakerStore) ListReleaseCircuitBreakers(
	filters map[string]string, offset, limit uint32) (*model.CircuitBreakerDetail, error) {
	//TODO
	return nil, nil
}

// 根据服务获取熔断规则
func (c *circuitBreakerStore) GetCircuitBreakersByService(
	name string, namespace string) (*model.CircuitBreaker, error) {
	//TODO
	return nil, nil
}