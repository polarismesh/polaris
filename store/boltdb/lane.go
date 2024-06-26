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
	"encoding/json"
	"sort"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

const (
	tblLaneGroup string = "lane_group"

	FieldLaneRuleText    = "Rule"
	FieldLaneDescription = "Description"
	FieldLaneGroupName   = "LaneGroup"
	FieldLaneRules       = "LaneRules"
)

type laneStore struct {
	handler BoltHandler
}

// AddLaneGroup 添加泳道组
func (l *laneStore) AddLaneGroup(tx store.Tx, item *model.LaneGroup) error {
	dbTx := tx.GetDelegateTx().(*bolt.Tx)
	// Before adding new data, you must clean up the old data
	if err := deleteValues(dbTx, tblLaneGroup, []string{item.ID}); err != nil {
		log.Errorf("[Store][Lane] delete lane group to kv error, %v", err)
		return store.Error(err)
	}

	for i := range item.LaneRules {
		rule := item.LaneRules[i]
		if rule.IsAdd() {
			rule.CreateTime = time.Now()
			rule.ModifyTime = time.Now()
		} else {
			rule.ModifyTime = time.Now()
		}

		if rule.IsChangeEnable() {
			rule.EnableTime = time.Now()
		} else {
			rule.EnableTime = time.Unix(0, 1)
		}
	}

	tn := time.Now()
	item.CreateTime = tn
	item.ModifyTime = tn
	item.Valid = true

	if err := saveValue(dbTx, tblLaneGroup, item.ID, toLaneGroupStore(item)); err != nil {
		log.Errorf("[Store][Lane] save lane group to kv error, %v", err)
		return store.Error(err)
	}
	return nil
}

// UpdateLaneGroup 更新泳道组
func (l *laneStore) UpdateLaneGroup(tx store.Tx, item *model.LaneGroup) error {
	dbTx := tx.GetDelegateTx().(*bolt.Tx)

	for i := range item.LaneRules {
		rule := item.LaneRules[i]
		if rule.IsAdd() {
			rule.CreateTime = time.Now()
			rule.ModifyTime = time.Now()
		} else {
			rule.ModifyTime = time.Now()
		}

		if rule.IsChangeEnable() {
			rule.EnableTime = time.Now()
		} else {
			rule.EnableTime = time.Unix(0, 1)
		}
	}

	properties := map[string]interface{}{
		FieldLaneDescription:  item.Description,
		FieldLaneRuleText:     item.Rule,
		CommonFieldModifyTime: time.Now(),
		CommonFieldRevision:   item.Revision,
		FieldLaneRules:        utils.MustJson(item.LaneRules),
	}

	if err := updateValue(dbTx, tblLaneGroup, item.ID, properties); err != nil {
		log.Error("[Store][Lane] update lane group to kv", zap.String("name", item.Name), zap.Error(err))
		return store.Error(err)
	}

	return nil
}

// LockLaneGroup 锁住一个泳道分组
func (l *laneStore) LockLaneGroup(tx store.Tx, name string) (*model.LaneGroup, error) {
	dbTx := tx.GetDelegateTx().(*bolt.Tx)
	ret, err := l.getLaneGroup(dbTx, name, false)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (l *laneStore) GetLaneGroup(name string) (*model.LaneGroup, error) {
	var ret *model.LaneGroup
	var err error
	err = l.handler.Execute(false, func(tx *bolt.Tx) error {
		ret, err = l.getLaneGroup(tx, name, true)
		return err
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (l *laneStore) GetLaneGroupByID(id string) (*model.LaneGroup, error) {
	var ret *model.LaneGroup
	var err error
	err = l.handler.Execute(false, func(tx *bolt.Tx) error {
		ret, err = l.getLaneGroup(tx, id, true)
		return err
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (l *laneStore) getLaneGroup(tx *bolt.Tx, name string, brief bool) (*model.LaneGroup, error) {
	fields := []string{
		CommonFieldValid, CommonFieldName, CommonFieldID,
	}
	result := make(map[string]interface{})
	err := loadValuesByFilter(tx, tblLaneGroup, fields, &LaneGroup{},
		func(m map[string]interface{}) bool {
			validVal, ok := m[CommonFieldValid]
			if ok && !validVal.(bool) {
				return false
			}
			saveId, _ := m[CommonFieldID].(string)
			saveName, _ := m[CommonFieldName].(string)
			return name == saveName || saveId == name
		}, result)
	if err != nil {
		log.Errorf("[Store][Lane] select one lane group to kv error, %v", err)
		return nil, store.Error(err)
	}
	for _, v := range result {
		saveData := toLaneGroupModel(v.(*LaneGroup))
		return saveData, nil
	}
	return nil, nil
}

// GetLaneGroups 查询泳道组
func (l *laneStore) GetLaneGroups(filter map[string]string, offset, limit uint32) (uint32, []*model.LaneGroup, error) {
	fields := []string{
		CommonFieldValid, CommonFieldID, CommonFieldName,
	}
	searchName, hasName := filter["name"]
	searchId, hasId := filter["id"]
	result, err := l.handler.LoadValuesByFilter(tblLaneGroup, fields, &LaneGroup{}, func(m map[string]interface{}) bool {
		validVal, ok := m[CommonFieldValid]
		if ok && !validVal.(bool) {
			return false
		}
		if hasName {
			if !utils.IsWildMatch(m[CommonFieldName].(string), searchName) {
				return false
			}
		}
		if hasId {
			if m[CommonFieldID].(string) != searchId {
				return false
			}
		}
		return true
	})
	if err != nil {
		log.Errorf("[Store][Lane] select lane group to kv error, %v", err)
		return 0, nil, store.Error(err)
	}
	groups := make([]*model.LaneGroup, 0, len(result))
	for _, v := range result {
		group := toLaneGroupModel(v.(*LaneGroup))
		groups = append(groups, group)
	}
	return uint32(len(result)), pageLaneGroups(groups, offset, limit, filter), nil
}

func pageLaneGroups(items []*model.LaneGroup, offset, limit uint32, order map[string]string) []*model.LaneGroup {
	orderField := order["order_field"]
	asc := strings.ToLower(order["order_type"]) == "asc"

	switch orderField {
	case "name":
		// 按照名称排序
		sort.Slice(items, func(i, j int) bool {
			if asc {
				return items[i].Name < items[j].Name
			}
			return items[i].Name > items[j].Name
		})
	default:
		// 默认按照更新时间排序
		sort.Slice(items, func(i, j int) bool {
			if asc {
				return items[i].ModifyTime.Before(items[j].ModifyTime)
			}
			return items[i].ModifyTime.After(items[j].ModifyTime)
		})
	}

	amount := uint32(len(items))
	endIdx := offset + limit
	if endIdx > amount {
		endIdx = amount
	}

	return items[offset:endIdx]
}

// DeleteLaneGroup 删除泳道组
func (l *laneStore) DeleteLaneGroup(id string) error {
	err := l.handler.Execute(true, func(tx *bolt.Tx) error {
		properties := map[string]interface{}{
			CommonFieldModifyTime: time.Now(),
			CommonFieldValid:      false,
		}
		if err := updateValue(tx, tblLaneGroup, id, properties); err != nil {
			log.Error("[Store][Lane] delete lane_group from kv", zap.Error(err))
			return err
		}
		return nil
	})
	return store.Error(err)
}

// GetMoreLaneGroups 获取泳道规则列表到缓存层
func (l *laneStore) GetMoreLaneGroups(mtime time.Time, firstUpdate bool) (map[string]*model.LaneGroup, error) {
	if firstUpdate {
		mtime = time.Unix(0, 0)
	}
	fields := []string{
		CommonFieldModifyTime,
	}

	groups := make(map[string]*model.LaneGroup, 32)
	err := l.handler.Execute(false, func(tx *bolt.Tx) error {
		result := make(map[string]interface{})
		err := loadValuesByFilter(tx, tblLaneGroup, fields, &LaneGroup{}, func(m map[string]interface{}) bool {
			val, ok := m[CommonFieldModifyTime]
			if !ok {
				return true
			}
			saveMtime := val.(time.Time)
			return !saveMtime.Before(mtime)
		}, result)
		if err != nil {
			log.Errorf("[Store][Lane] get more lane rule for cache, %v", err)
			return store.Error(err)
		}
		for _, v := range result {
			item := toLaneGroupModel(v.(*LaneGroup))
			groups[item.ID] = item
		}
		return nil
	})
	if err != nil {
		log.Error("[Store][Lane] get more lane_group for cache update", zap.Error(err))
		return nil, store.Error(err)
	}
	return groups, nil
}

func (l *laneStore) GetLaneRuleMaxPriority() (int32, error) {
	var maxPriority int32
	fields := []string{
		FieldLaneRules,
		CommonFieldValid,
	}
	_, err := l.handler.LoadValuesByFilter(tblLaneGroup, fields, &model.LaneGroup{}, func(m map[string]interface{}) bool {
		valid, _ := m[CommonFieldValid].(bool)
		if !valid {
			return false
		}
		rules := make(map[string]*model.LaneRule)
		_ = json.Unmarshal([]byte(m[FieldLaneRules].(string)), &rules)
		for _, rule := range rules {
			curPriority := rule.Priority
			if maxPriority <= int32(curPriority) {
				maxPriority = int32(curPriority)
			}
		}
		return false
	})
	if err != nil {
		log.Error("[Store][Lane] get current lane_rule max priority", zap.Error(err))
	}
	return maxPriority, err
}

func toLaneGroupStore(data *model.LaneGroup) *LaneGroup {
	return &LaneGroup{
		ID:          data.ID,
		Name:        data.Name,
		Rule:        data.Rule,
		Revision:    data.Revision,
		Description: data.Description,
		Valid:       data.Valid,
		CreateTime:  data.CreateTime,
		ModifyTime:  data.ModifyTime,
		LaneRules:   utils.MustJson(data.LaneRules),
	}
}

func toLaneGroupModel(data *LaneGroup) *model.LaneGroup {
	ret := &model.LaneGroup{
		ID:          data.ID,
		Name:        data.Name,
		Rule:        data.Rule,
		Revision:    data.Revision,
		Description: data.Description,
		Valid:       data.Valid,
		CreateTime:  data.CreateTime,
		ModifyTime:  data.ModifyTime,
		LaneRules:   map[string]*model.LaneRule{},
	}

	_ = json.Unmarshal([]byte(data.Name), &ret.LaneRules)
	return ret
}

type LaneGroup struct {
	ID          string
	Name        string
	Rule        string
	Revision    string
	Description string
	Valid       bool
	CreateTime  time.Time
	ModifyTime  time.Time
	LaneRules   string
}
