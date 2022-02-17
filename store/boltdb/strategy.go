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
	"time"

	"github.com/polarismesh/polaris-server/common/model"
)

const (
	tblStrategy string = "strategy"

	StrategyFieldModifyTime string = "ModifyTime"
)

// StrategyStore
type strategyStore struct {
	handler BoltHandler
}

// AddStrategy
func (ss *strategyStore) AddStrategy(strategy *model.StrategyDetail) error {
	return nil
}

// UpdateStrategy
func (ss *strategyStore) UpdateStrategy(strategy *model.ModifyStrategyDetail) error {
	return nil
}

// DeleteStrategy
func (ss *strategyStore) DeleteStrategy(id string) error {
	return nil
}

// AddStrategyResources
func (ss *strategyStore) AddStrategyResources(resources []model.StrategyResource) error {
	return nil
}

// RemoveStrategyResources
func (ss *strategyStore) RemoveStrategyResources(resources []model.StrategyResource) error {
	return nil
}

// LooseAddStrategyResources 松要求的添加鉴权策略的资源，允许忽略主键冲突的问题
func (ss *strategyStore) LooseAddStrategyResources(resources []model.StrategyResource) error {
	return nil
}

// GetStrategyDetail
func (ss *strategyStore) GetStrategyDetail(id string, isDefault bool) (*model.StrategyDetail, error) {
	return nil, nil
}

// GetStrategyResources
func (ss *strategyStore) GetStrategyResources(principalId string,
	principalRole model.PrincipalType) ([]model.StrategyResource, error) {

	return nil, nil
}

// GetDefaultStrategyDetailByPrincipal
func (ss *strategyStore) GetDefaultStrategyDetailByPrincipal(principalId string,
	principalType int) (*model.StrategyDetail, error) {

	return nil, nil
}

// GetStrategies
func (ss *strategyStore) GetStrategies(filters map[string]string, offset uint32, limit uint32) (uint32,
	[]*model.StrategyDetail, error) {
	return 0, nil, nil
}

// GetStrategyDetailsForCache
func (ss *strategyStore) GetStrategyDetailsForCache(mtime time.Time,
	firstUpdate bool) ([]*model.StrategyDetail, error) {

	ret, err := ss.handler.LoadValuesByFilter(tblStrategy, []string{StrategyFieldModifyTime}, &model.StrategyDetail{},
		func(m map[string]interface{}) bool {
			mt := m[StrategyFieldModifyTime].(time.Time)
			isAfter := mt.After(mtime)
			return isAfter
		})
	if err != nil {
		return nil, err
	}

	strategies := make([]*model.StrategyDetail, 0, len(ret))

	for k := range ret {
		val := ret[k]
		strategies = append(strategies, val.(*model.StrategyDetail))
	}

	return strategies, nil
}
