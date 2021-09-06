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
	"github.com/polarismesh/polaris-server/common/model"
	"time"
)

type instanceStore struct {
	handler BoltHandler
}

// 增加一个实例
func (i *instanceStore) AddInstance(instance *model.Instance) error {
	//TODO
	return nil
}

// 增加多个实例
func (i *instanceStore) BatchAddInstances(instances []*model.Instance) error {
	//TODO
	return nil
}

// 更新实例
func (i *instanceStore) UpdateInstance(instance *model.Instance) error {
	//TODO
	return nil
}

// 删除一个实例，实际是把valid置为false
func (i *instanceStore) DeleteInstance(instanceID string) error {
	//TODO
	return nil
}

// 批量删除实例，flag=1
func (i *instanceStore) BatchDeleteInstances(ids []interface{}) error {
	//TODO
	return nil
}

// 清空一个实例，真正删除
func (i *instanceStore) CleanInstance(instanceID string) error {
	//TODO
	return nil
}

// 检查ID是否存在，并且返回所有ID的查询结果
func (i *instanceStore) CheckInstancesExisted(ids map[string]bool) (map[string]bool, error) {
	//TODO
	return nil, nil
}

// 获取实例关联的token
func (i *instanceStore) GetInstancesBrief(ids map[string]bool) (map[string]*model.Instance, error) {
	//TODO
	return nil, nil
}

// 查询一个实例的详情，只返回有效的数据
func (i *instanceStore) GetInstance(instanceID string) (*model.Instance, error) {
	//TODO
	return nil, nil
}

// 获取有效的实例总数
func (i *instanceStore) GetInstancesCount() (uint32, error) {
	//TODO
	return 0, nil
}

// 根据服务和Host获取实例（不包括metadata）
func (i *instanceStore) GetInstancesMainByService(serviceID, host string) ([]*model.Instance, error) {
	//TODO
	return nil, nil
}

// 根据过滤条件查看实例详情及对应数目
func (i *instanceStore) GetExpandInstances(filter, metaFilter map[string]string,
	offset uint32, limit uint32) (uint32, []*model.Instance, error) {
	//TODO
	return 0, nil, nil
}

// 根据mtime获取增量instances，返回所有store的变更信息
func (i *instanceStore) GetMoreInstances(
	mtime time.Time, firstUpdate, needMeta bool, serviceID []string) (map[string]*model.Instance, error) {
	//TODO
	return nil, nil
}

// 设置实例的健康状态
func (i *instanceStore) SetInstanceHealthStatus(instanceID string, flag int, revision string) error {
	//TODO
	return nil
}

// 批量修改实例的隔离状态
func (i *instanceStore) BatchSetInstanceIsolate(ids []interface{}, isolate int, revision string) error {
	//TODO
	return nil
}