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

	bolt "go.etcd.io/bbolt"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/store"
)

var _ store.RoutingConfigStoreV2 = (*routingStoreV2)(nil)

var (
	// ErrMultipleRoutingV2Found 多个路由配置
	ErrMultipleRoutingV2Found = errors.New("multiple routing v2 found")
)

const (
	tblNameRoutingV2 = "routing_config_v2"

	routingV2FieldID          = "ID"
	routingV2FieldName        = "Name"
	routingV2FieldNamespace   = "Namespace"
	routingV2FieldPolicy      = "Policy"
	routingV2FieldConfig      = "Config"
	routingV2FieldEnable      = "Enable"
	routingV2FieldRevision    = "Revision"
	routingV2FieldCreateTime  = "CreateTime"
	routingV2FieldModifyTime  = "ModifyTime"
	routingV2FieldEnableTime  = "EnableTime"
	routingV2FieldValid       = "Valid"
	routingV2FieldPriority    = "Priority"
	routingV2FieldDescription = "Description"
)

type routingStoreV2 struct {
	handler BoltHandler
}

// CreateRoutingConfigV2 新增一个路由配置
func (r *routingStoreV2) CreateRoutingConfigV2(conf *model.RouterConfig) error {
	if conf.ID == "" || conf.Revision == "" {
		log.Errorf("[Store][boltdb] create routing config v2 missing id or revision")
		return store.NewStatusError(store.EmptyParamsErr, "missing id or revision")
	}
	if conf.Policy == "" || conf.Config == "" {
		log.Errorf("[Store][boltdb] create routing config v2 missing params")
		return store.NewStatusError(store.EmptyParamsErr, "missing some params")
	}

	return r.handler.Execute(true, func(tx *bolt.Tx) error {
		return r.createRoutingConfigV2(tx, conf)
	})
}

// cleanRoutingConfig 从数据库彻底清理路由配置
func (r *routingStoreV2) cleanRoutingConfig(tx *bolt.Tx, ruleID string) error {
	err := deleteValues(tx, tblNameRoutingV2, []string{ruleID})
	if err != nil {
		log.Errorf("[Store][boltdb] delete invalid route config v2 error, %v", err)
		return err
	}
	return nil
}

func (r *routingStoreV2) CreateRoutingConfigV2Tx(tx store.Tx, conf *model.RouterConfig) error {
	if tx == nil {
		return errors.New("transaction is nil")
	}

	dbTx := tx.GetDelegateTx().(*bolt.Tx)
	return r.createRoutingConfigV2(dbTx, conf)
}

func (r *routingStoreV2) createRoutingConfigV2(tx *bolt.Tx, conf *model.RouterConfig) error {
	if err := r.cleanRoutingConfig(tx, conf.ID); err != nil {
		return err
	}

	currTime := time.Now()
	conf.CreateTime = currTime
	conf.ModifyTime = currTime
	conf.EnableTime = time.Time{}
	conf.Valid = true

	if conf.Enable {
		conf.EnableTime = time.Now()
	} else {
		conf.EnableTime = time.Time{}
	}

	err := saveValue(tx, tblNameRoutingV2, conf.ID, conf)
	if err != nil {
		log.Errorf("[Store][boltdb] add routing config v2 to kv error, %v", err)
		return err
	}
	return nil
}

// UpdateRoutingConfigV2 更新一个路由配置
func (r *routingStoreV2) UpdateRoutingConfigV2(conf *model.RouterConfig) error {
	if conf.ID == "" || conf.Revision == "" {
		log.Errorf("[Store][boltdb] update routing config v2 missing id or revision")
		return store.NewStatusError(store.EmptyParamsErr, "missing id or revision")
	}
	if conf.Policy == "" || conf.Config == "" {
		log.Errorf("[Store][boltdb] create routing config v2 missing params")
		return store.NewStatusError(store.EmptyParamsErr, "missing some params")
	}

	return r.handler.Execute(true, func(tx *bolt.Tx) error {
		return r.updateRoutingConfigV2Tx(tx, conf)
	})
}

func (r *routingStoreV2) UpdateRoutingConfigV2Tx(tx store.Tx, conf *model.RouterConfig) error {
	if tx == nil {
		return errors.New("tx is nil")
	}

	dbTx := tx.GetDelegateTx().(*bolt.Tx)
	return r.updateRoutingConfigV2Tx(dbTx, conf)
}

func (r *routingStoreV2) updateRoutingConfigV2Tx(tx *bolt.Tx, conf *model.RouterConfig) error {
	properties := make(map[string]interface{})
	properties[routingV2FieldEnable] = conf.Enable
	properties[routingV2FieldName] = conf.Name
	properties[routingV2FieldPolicy] = conf.Policy
	properties[routingV2FieldConfig] = conf.Config
	properties[routingV2FieldPriority] = conf.Priority
	properties[routingV2FieldRevision] = conf.Revision
	properties[routingV2FieldDescription] = conf.Description
	properties[routingV2FieldModifyTime] = time.Now()

	err := updateValue(tx, tblNameRoutingV2, conf.ID, properties)
	if err != nil {
		log.Errorf("[Store][boltdb] update route config v2 to kv error, %v", err)
		return err
	}
	return nil
}

// EnableRouting
func (r *routingStoreV2) EnableRouting(conf *model.RouterConfig) error {
	if conf.ID == "" || conf.Revision == "" {
		return errors.New("[Store][database] enable routing config v2 missing some params")
	}

	if conf.Enable {
		conf.EnableTime = time.Now()
	} else {
		conf.EnableTime = time.Time{}
	}

	properties := make(map[string]interface{})
	properties[routingV2FieldEnable] = conf.Enable
	properties[routingV2FieldEnableTime] = conf.EnableTime
	properties[routingV2FieldRevision] = conf.Revision
	properties[routingV2FieldModifyTime] = time.Now()

	err := r.handler.UpdateValue(tblNameRoutingV2, conf.ID, properties)
	if err != nil {
		log.Errorf("[Store][boltdb] enable route config v2 to kv error, %v", err)
		return err
	}
	return nil
}

// DeleteRoutingConfigV2 删除一个路由配置
func (r *routingStoreV2) DeleteRoutingConfigV2(ruleID string) error {
	if ruleID == "" {
		log.Errorf("[Store][boltdb] update routing config v2 missing id")
		return store.NewStatusError(store.EmptyParamsErr, "missing id")
	}
	properties := make(map[string]interface{})
	properties[routingV2FieldValid] = false
	properties[routingV2FieldModifyTime] = time.Now()

	err := r.handler.UpdateValue(tblNameRoutingV2, ruleID, properties)
	if err != nil {
		log.Errorf("[Store][boltdb] update route config v2 to kv error, %v", err)
		return err
	}
	return nil
}

// GetRoutingConfigsV2ForCache 通过mtime拉取增量的路由配置信息
// 此方法用于 cache 增量更新，需要注意 mtime 应为数据库时间戳
func (r *routingStoreV2) GetRoutingConfigsV2ForCache(mtime time.Time, firstUpdate bool) ([]*model.RouterConfig, error) {
	if firstUpdate {
		mtime = time.Time{}
	}

	fields := []string{routingV2FieldModifyTime}

	routes, err := r.handler.LoadValuesByFilter(tblNameRoutingV2, fields, &model.RouterConfig{},
		func(m map[string]interface{}) bool {
			rMtime, ok := m[routingV2FieldModifyTime]
			if !ok {
				return false
			}
			routeMtime := rMtime.(time.Time)
			return !routeMtime.Before(mtime)
		})
	if err != nil {
		log.Errorf("[Store][boltdb] load route config v2 from kv error, %v", err)
		return nil, err
	}

	return toRouteConfV2(routes), nil
}

func toRouteConfV2(m map[string]interface{}) []*model.RouterConfig {
	var routeConf []*model.RouterConfig
	for _, r := range m {
		routeConf = append(routeConf, r.(*model.RouterConfig))
	}

	return routeConf
}

// GetRoutingConfigV2WithID 根据服务ID拉取路由配置
func (r *routingStoreV2) GetRoutingConfigV2WithID(id string) (*model.RouterConfig, error) {
	tx, err := r.handler.StartTx()
	if err != nil {
		return nil, err
	}

	boldTx := tx.GetDelegateTx().(*bolt.Tx)
	defer func() {
		_ = boldTx.Rollback()
	}()

	return r.getRoutingConfigV2WithIDTx(boldTx, id)
}

// GetRoutingConfigV2WithIDTx 根据服务ID拉取路由配置
func (r *routingStoreV2) GetRoutingConfigV2WithIDTx(tx store.Tx, id string) (*model.RouterConfig, error) {

	if tx == nil {
		return nil, errors.New("tx is nil")
	}

	boldTx := tx.GetDelegateTx().(*bolt.Tx)
	return r.getRoutingConfigV2WithIDTx(boldTx, id)
}

func (r *routingStoreV2) getRoutingConfigV2WithIDTx(tx *bolt.Tx, id string) (*model.RouterConfig, error) {
	ret := make(map[string]interface{})
	if err := loadValues(tx, tblNameRoutingV2, []string{id}, &model.RouterConfig{}, ret); err != nil {
		log.Error("[Store][boltdb] load route config v2 from kv", zap.String("routing-id", id), zap.Error(err))
		return nil, err
	}

	if len(ret) == 0 {
		return nil, nil
	}

	if len(ret) > 1 {
		return nil, ErrMultipleRoutingV2Found
	}

	val := ret[id].(*model.RouterConfig)
	if !val.Valid {
		return nil, nil
	}

	return val, nil
}
