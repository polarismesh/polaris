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
	"errors"
	"strings"
	"time"

	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store"
)

const (
	tblBusiness string = "business"
)

type businessStore struct {
	handler BoltHandler
}

// AddBusiness 增加一个业务集
func (bs *businessStore) AddBusiness(b *model.Business) error {
	if b.ID == "" || b.Name == "" || b.Token == "" || b.Owner == "" {
		log.Errorf("[Store][business] add business missing some params: %+v", b)
		return errors.New("Add Business missing some params")
	}

	dbOp := bs.handler

	tNow := time.Now()
	b.CreateTime = tNow
	b.ModifyTime = tNow

	if err := dbOp.SaveValue(tblBusiness, b.ID, b); err != nil {
		log.Errorf("[Store][business] add business err : %s", err.Error())
		return store.Error(err)
	}

	return nil
}

// DeleteBusiness 删除一个业务集
func (bs *businessStore) DeleteBusiness(bid string) error {
	if bid == "" {
		log.Errorf("[Store][business] delete business missing id")
		return errors.New("Delete Business missing some params")
	}

	dbOp := bs.handler

	if err := dbOp.DeleteValues(tblBusiness, []string{bid}); err != nil {
		log.Errorf("[Store][business] delete business err : %s", err.Error())
		return store.Error(err)
	}

	return nil
}

// UpdateBusiness 更新业务集
func (bs *businessStore) UpdateBusiness(b *model.Business) error {
	if b.ID == "" || b.Name == "" || b.Owner == "" {
		log.Errorf("[Store][business] update business missing some params: %+v", b)
		return errors.New("Update Business missing some params")
	}

	dbOp := bs.handler

	b.ModifyTime = time.Now()

	if err := dbOp.SaveValue(tblBusiness, b.ID, b); err != nil {
		log.Errorf("[Store][business] add business err : %s", err.Error())
		return store.Error(err)
	}

	return nil
}

// UpdateBusinessToken 更新业务集token
func (bs *businessStore) UpdateBusinessToken(bid string, token string) error {
	if bid == "" || token == "" {
		log.Errorf("[Store][business] update business token missing some params")
		return errors.New("Update Business Token missing some params")
	}

	dbOp := bs.handler

	if err := dbOp.UpdateValue(tblBusiness, bid, map[string]interface{}{
		"Token": token,
	}); err != nil {
		return store.Error(err)
	}

	return nil
}

// ListBusiness 查询owner下业务集
func (bs *businessStore) ListBusiness(owner string) ([]*model.Business, error) {
	if owner == "" {
		log.Errorf("[Store][business] list business missing owner")
		return nil, errors.New("List Business Mising param owner")
	}

	dbOp := bs.handler

	result, err := dbOp.LoadValuesByFilter(tblBusiness, []string{"Owner"}, &model.Business{}, func(m map[string]interface{}) bool {

		mO, ok := m["Owner"]
		if !ok {
			return false
		}

		return strings.Contains(mO.(string), owner)
	})
	if err != nil {
		log.Errorf("[Store][business] list business filter by Owner err : %s", err)
		return nil, store.Error(err)
	}

	ans := make([]*model.Business, 0)
	for _, v := range result {
		record := v.(*model.Business)
		ans = append(ans, record)
	}
	return ans, nil
}

// GetBusinessByID 根据业务集ID获取业务集详情
func (bs *businessStore) GetBusinessByID(id string) (*model.Business, error) {

	if id == "" {
		log.Errorf("[Store][business] get business missing id")
		return nil, errors.New("Get Business missing some params")
	}
	dbOp := bs.handler

	result, err := dbOp.LoadValues(tblBusiness, []string{id}, &model.Business{})
	if err != nil {
		return nil, store.Error(err)
	}

	val := result[id]
	if val != nil {
		return val.(*model.Business), nil
	}

	return nil, nil
}

// GetMoreBusiness 根据mtime获取增量数据
func (bs *businessStore) GetMoreBusiness(mtime time.Time) ([]*model.Business, error) {

	dbOp := bs.handler

	result, err := dbOp.LoadValuesByFilter(tblBusiness, []string{"ModifyTime"}, &model.Business{}, func(m map[string]interface{}) bool {
		mT := m["ModifyTime"].(time.Time)
		return mT.After(mtime)
	})
	if err != nil {
		log.Errorf("[Store][business] list business filter by mtime err : %s", err)
		return nil, store.Error(err)
	}

	ans := make([]*model.Business, 0)
	for _, v := range result {
		record := v.(*model.Business)
		ans = append(ans, record)
	}
	return ans, nil
}

// listAllBusiness 列出所有的 Business 信息
func (bs *businessStore) listAllBusiness() ([]*model.Business, error) {

	dbOp := bs.handler

	result, err := dbOp.LoadValues(tblBusiness, []string{}, &model.Business{})
	if err != nil {
		log.Errorf("[Store][business] list business by owner err : %s", err)
		return nil, store.Error(err)
	}

	ans := make([]*model.Business, 0)
	for _, v := range result {
		record := v.(*model.Business)
		ans = append(ans, record)
	}

	return ans, nil
}
