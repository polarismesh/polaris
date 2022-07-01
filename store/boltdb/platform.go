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
	"math"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/store"
)

const (
	tblPlatform string = "platform"

	PlatformFieldID         string = "ID"
	PlatformFieldName       string = "Name"
	PlatformFieldDomain     string = "Domain"
	PlatformFieldQPS        string = "QPS"
	PlatformFieldToken      string = "Token"
	PlatformFieldOwner      string = "Owner"
	PlatformFieldDepartment string = "Department"
	PlatformFieldComment    string = "Comment"
	PlatformFieldValid      string = "Valid"
	PlatformFieldCreateTime string = "CreateTime"
	PlatformFieldModifyTime string = "ModifyTime"
)

type platformStore struct {
	handler BoltHandler
}

// CreatePlatform 新增平台信息
func (p *platformStore) CreatePlatform(platform *model.Platform) error {
	if platform.ID == "" {
		return errors.New("create platform missing id")
	}

	tNow := time.Now()

	platformKey := platform.ID
	platform.CreateTime = tNow
	platform.ModifyTime = tNow
	platform.Valid = true

	dbOp := p.handler

	if old, _ := p.GetPlatformById(platformKey); old != nil {
		log.Errorf("[Store][platform] create platform(%s) duplicate", platform.ID)
		return errors.New("create Platform duplicate")
	}

	if err := dbOp.SaveValue(tblPlatform, platformKey, platform); err != nil {
		log.Errorf("[Store][platform] create platform(%s) err: %s", platform.ID, err.Error())
		return store.Error(err)
	}

	return nil
}

// UpdatePlatform 更新平台信息
func (p *platformStore) UpdatePlatform(platform *model.Platform) error {
	if platform.ID == "" {
		return errors.New("create platform missing id")
	}

	platformKey := platform.ID
	platform.ModifyTime = time.Now()
	platform.Valid = true

	dbOp := p.handler

	if err := dbOp.SaveValue(tblPlatform, platformKey, platform); err != nil {
		log.Errorf("[Store][platform] update platform(%+v) err: %s", platform, err.Error())
		return store.Error(err)
	}

	return nil
}

// DeletePlatform 删除平台信息
func (p *platformStore) DeletePlatform(id string) error {
	if strings.Compare(id, "") == 0 {
		return errors.New("delete platform missing id")
	}

	platformKey := id

	properties := make(map[string]interface{})
	properties[PlatformFieldValid] = false
	properties[PlatformFieldModifyTime] = time.Now()

	if err := p.handler.UpdateValue(tblPlatform, platformKey, properties); err != nil {
		log.Errorf("[Store][platform] delete platform(%s) err: %s", id, err.Error())
		return store.Error(err)
	}

	return nil
}

// GetPlatformById 查询平台信息
func (p *platformStore) GetPlatformById(id string) (*model.Platform, error) {
	if strings.Compare(id, "") == 0 {
		return nil, errors.New("GetPlatformById platform missing id")
	}

	platformKey := id

	dbOp := p.handler

	result, err := dbOp.LoadValues(tblPlatform, []string{platformKey}, &model.Platform{})
	if err != nil {
		log.Errorf("[Store][platform] get platform by id(%s) err: %s", id, err.Error())
		return nil, store.Error(err)
	}

	val, ok := result[platformKey]
	if !ok {
		return nil, nil
	}

	ret, ok := val.(*model.Platform)
	if !ok {
		return nil, nil
	}

	if !ret.Valid {
		return nil, nil
	}

	return ret, nil
}

// GetPlatforms 根据过滤条件查询平台信息
func (p *platformStore) GetPlatforms(
	query map[string]string, offset uint32, limit uint32) (uint32, []*model.Platform, error) {

	dbOp := p.handler

	result, err := dbOp.LoadValuesByFilter(tblPlatform, utils.CollectMapKeys(query), &model.Platform{}, func(m map[string]interface{}) bool {
		for k, v := range query {
			qV := m[k]
			if !reflect.DeepEqual(qV, v) {
				return false
			}
		}
		return true
	})
	if err != nil {
		log.Errorf("[Store][platform] get platform by query(%#v) err: %s", query, err.Error())
		return 0, nil, store.Error(err)
	}

	total := len(result)

	platformSlice := make([]*model.Platform, 0)
	for _, v := range result {
		platformSlice = append(platformSlice, v.(*model.Platform))
	}

	sort.Slice(platformSlice, func(i, j int) bool {
		a := platformSlice[i]
		b := platformSlice[j]
		return a.ModifyTime.After(b.ModifyTime)
	})

	return uint32(total), platformSlice[offset:int(math.Min(float64(offset+limit), float64(total)))], nil
}
