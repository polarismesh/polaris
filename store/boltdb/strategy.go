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
	"time"

	"github.com/polarismesh/polaris-server/common/model"
)

// StrategyStore
type strategyStore struct {
	handler BoltHandler
}

// AddStrategy
func (ss *strategyStore) AddStrategy(strategy *model.StrategyDetail) error {
	return errors.New("implement me")
}

// UpdateStrategy
//  @param strategy
//  @return error
func (ss *strategyStore) UpdateStrategy(strategy *model.ModifyStrategyDetail) error {
	return errors.New("implement me")
}

// DeleteStrategy
//  @param id
//  @return error
func (ss *strategyStore) DeleteStrategy(id string) error {
	return errors.New("implement me")
}

// AddStrategyResources
//  @param resources
//  @return error
func (ss *strategyStore) AddStrategyResources(resources []model.StrategyResource) error {
	return errors.New("implement me")
}

// RemoveStrategyResources
//  @param resources
//  @return error
func (ss *strategyStore) RemoveStrategyResources(resources []model.StrategyResource) error {
	return errors.New("implement me")
}

// LooseAddStrategyResources 松要求的添加鉴权策略的资源，允许忽略主键冲突的问题
//  @param resources
//  @return error
func (ss *strategyStore) LooseAddStrategyResources(resources []model.StrategyResource) error {
	return errors.New("implement me")
}

// GetStrategyDetail
//  @param id
//  @return *model.StrategyDetail
//  @return error
func (ss *strategyStore) GetStrategyDetail(id string) (*model.StrategyDetail, error) {
	return nil, errors.New("implement me")
}

// GetStrategyDetailByName
//  @receiver ss
//  @param owner
//  @param name
//  @return *model.StrategyDetail
//  @return error
func (ss *strategyStore) GetStrategyDetailByName(owner, name string) (*model.StrategyDetail, error) {
	return nil, errors.New("implement me")
}

// GetStrategySimpleByName
//  @receiver ss
//  @param owner
//  @param name
//  @return *model.Strategy
//  @return error
func (ss *strategyStore) GetStrategySimpleByName(owner, name string) (*model.Strategy, error) {
	return nil, errors.New("implement me")
}

// GetSimpleStrategies
//  @param filters
//  @param offset
//  @param limit
//  @return uint32
//  @return []*model.StrategyDetail
//  @return error
func (ss *strategyStore) GetSimpleStrategies(filters map[string]string, offset uint32, limit uint32) (uint32, []*model.StrategyDetail, error) {
	return 0, nil, errors.New("implement me")
}

// GetStrategyDetailsForCache
//  @param mtime
//  @param firstUpdate
//  @return []*model.StrategyDetail
//  @return error
func (ss *strategyStore) GetStrategyDetailsForCache(mtime time.Time, firstUpdate bool) ([]*model.StrategyDetail, error) {
	return nil, errors.New("implement me")
}
