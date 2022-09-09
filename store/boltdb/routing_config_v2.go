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

	v2 "github.com/polarismesh/polaris-server/common/model/v2"
	"github.com/polarismesh/polaris-server/store"
)

const (
	tblNameRoutingV2 = "routing_config_v2"

	routingV2FieldID         = "ID"
	routingV2FieldName       = "Name"
	routingV2FieldPolicy     = "Policy"
	routingV2FieldConfig     = "Config"
	routingV2FieldEnable     = "Enable"
	routingV2FieldRevision   = "Revision"
	routingV2FieldCreateTime = "CreateTime"
	routingV2FieldModifyTime = "ModifyTime"
	routingV2FieldEnableTime = "EnableTime"
	routingV2FieldValid      = "Valid"
)

type routingStoreV2 struct {
	handler BoltHandler
}

// CreateRoutingConfigV2 新增一个路由配置
func (r *routingStoreV2) CreateRoutingConfigV2(conf *v2.RoutingConfig) error {
	if conf.ID == "" || conf.Revision == "" {
		log.Errorf("[Store][boltdb] create routing config v2 missing id or revision")
		return store.NewStatusError(store.EmptyParamsErr, "missing id or revision")
	}
	if conf.Name == "" || conf.Config == "" {
		log.Errorf("[Store][boltdb] create routing config v2 missing params")
		return store.NewStatusError(store.EmptyParamsErr, "missing some params")
	}

	if err := r.cleanRoutingConfig(conf.ID); err != nil {
		return err
	}

	currTime := time.Now()
	conf.CreateTime = currTime
	conf.ModifyTime = currTime
	conf.EnableTime = time.Time{}
	conf.Valid = true
	conf.Enable = false

	err := r.handler.SaveValue(tblNameRoutingV2, conf.ID, conf)
	if err != nil {
		log.Errorf("[Store][boltdb] add routing config v2 to kv error, %v", err)
		return err
	}
	return nil
}

// cleanRoutingConfig 从数据库彻底清理路由配置
func (r *routingStoreV2) cleanRoutingConfig(ruleID string) error {
	err := r.handler.DeleteValues(tblNameRoutingV2, []string{ruleID}, false)
	if err != nil {
		log.Errorf("[Store][boltdb] delete invalid route config v2 error, %v", err)
		return err
	}
	return nil
}

// UpdateRoutingConfigV2 更新一个路由配置
func (r *routingStoreV2) UpdateRoutingConfigV2(conf *v2.RoutingConfig) error {
	if conf.ID == "" || conf.Revision == "" {
		log.Errorf("[Store][boltdb] update routing config v2 missing id or revision")
		return store.NewStatusError(store.EmptyParamsErr, "missing id or revision")
	}
	if conf.Name == "" || conf.Config == "" {
		log.Errorf("[Store][boltdb] create routing config v2 missing params")
		return store.NewStatusError(store.EmptyParamsErr, "missing some params")
	}

	properties := make(map[string]interface{})
	properties[routingV2FieldEnable] = conf.Enable
	properties[routingV2FieldEnableTime] = conf.EnableTime
	properties[routingV2FieldPolicy] = conf.Policy
	properties[routingV2FieldConfig] = conf.Config
	properties[routingV2FieldRevision] = conf.Revision
	properties[routingV2FieldModifyTime] = time.Now()

	err := r.handler.UpdateValue(tblNameRouting, conf.ID, properties)
	if err != nil {
		log.Errorf("[Store][boltdb] update route config v2 to kv error, %v", err)
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

	err := r.handler.UpdateValue(tblNameRouting, ruleID, properties)
	if err != nil {
		log.Errorf("[Store][boltdb] update route config v2 to kv error, %v", err)
		return err
	}
	return nil
}

// GetRoutingConfigsV2ForCache 通过mtime拉取增量的路由配置信息
// 此方法用于 cache 增量更新，需要注意 mtime 应为数据库时间戳
func (r *routingStoreV2) GetRoutingConfigsV2ForCache(mtime time.Time, firstUpdate bool) ([]*v2.RoutingConfig, error) {
	if firstUpdate {
		mtime = time.Time{}
	}

	

}

// GetRoutingConfigV2WithID 根据服务ID拉取路由配置
func (r *routingStoreV2) GetRoutingConfigV2WithID(id string) (*v2.RoutingConfig, error) {}

// GetRoutingConfigsV2 查询路由配置列表
func (r *routingStoreV2) GetRoutingConfigsV2(filter map[string]string, offset uint32,
	limit uint32) (uint32, []*v2.RoutingConfig, error) {
}
