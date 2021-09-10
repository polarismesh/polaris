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
	 "math"
	 "reflect"
	 "sort"
	 "strings"
	 "time"
 
	 "github.com/polarismesh/polaris-server/common/log"
	 "github.com/polarismesh/polaris-server/common/model"
	 "github.com/polarismesh/polaris-server/common/utils"
	 "github.com/polarismesh/polaris-server/store"
 )
 
 const (
	 DataTypePlatform string = "platform"
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
 
	 dbOp := p.handler
 
	 if old, _ := p.GetPlatformById(platformKey); old != nil {
		 log.Errorf("[Store][platform] create platform(%s) duplicate", platform.ID)
		 return errors.New("Create Platform duplicate")
	 }
 
	 if err := dbOp.SaveValue(DataTypePlatform, platformKey, platform); err != nil {
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
 
	 dbOp := p.handler
 
	 if err := dbOp.SaveValue(DataTypePlatform, platformKey, platform); err != nil {
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
 
	 dbOp := p.handler
 
	 if err := dbOp.DeleteValues(DataTypePlatform, []string{platformKey}); err != nil {
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
 
	 result, err := dbOp.LoadValues(DataTypePlatform, []string{platformKey}, &model.Platform{})
	 if err != nil {
		 log.Errorf("[Store][platform] get platform by id(%s) err: %s", id, err.Error())
		 return nil, store.Error(err)
	 }
 
	 val := result[platformKey]
	 if val == nil {
		 return nil, nil
	 }
 
	 return val.(*model.Platform), nil
 }
 
 // GetPlatforms 根据过滤条件查询平台信息
 func (p *platformStore) GetPlatforms(
	 query map[string]string, offset uint32, limit uint32) (uint32, []*model.Platform, error) {

	 dbOp := p.handler
 
	 result, err := dbOp.LoadValuesByFilter(DataTypePlatform, utils.CollectFilterFields(query), &model.Platform{}, func(m map[string]interface{}) bool {
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
 