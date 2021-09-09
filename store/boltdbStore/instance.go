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
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"sort"
	"strings"
	"time"
)

type instanceStore struct {
	handler BoltHandler
}

// 增加一个实例
func (i *instanceStore) AddInstance(instance *model.Instance) error {

	// 新增数据之前，必须先清理老数据
	if err := i.handler.DeleteValues(InstanceStoreType, []string{instance.ID()}); err != nil {
		log.Errorf("delete instance to kv error, %v", err)
		return err
	}

	if err := i.handler.SaveValue(InstanceStoreType, instance.ID(), instance); err != nil{
		log.Errorf("save instance to kv error, %v", err)
		return err
	}

	return nil
}

// 增加多个实例
func (i *instanceStore) BatchAddInstances(instances []*model.Instance) error {

	// 得到 id list
	if len(instances) == 0 {
		return nil
	}

	var insIds []string
	for _, instance := range instances {
		insIds = append(insIds, instance.ID())
	}

	// 直接清理所有的老数据
	if err := i.handler.DeleteValues(InstanceStoreType, insIds); err != nil{
		log.Errorf("save instance to kv error, %v", err)
		return err
	}

	for _, instance := range instances {
		if err := i.handler.SaveValue(InstanceStoreType, instance.ID(), instance); err != nil{
			// 遇到错误就中断，返回错误
			log.Errorf("save instance to kv error, %v", err)
			return err
		}
	}

	return nil
}

// 更新实例
func (i *instanceStore) UpdateInstance(instance *model.Instance) error {

	if err := i.handler.SaveValue(InstanceStoreType, instance.ID(), instance); err != nil{
		log.Errorf("update instance to kv error, %v", err)
		return err
	}

	return nil
}

// 删除一个实例，实际是把valid置为false
func (i *instanceStore) DeleteInstance(instanceID string) error {

	if err := i.handler.DeleteValues(InstanceStoreType, []string{instanceID}); err != nil{
		log.Errorf("update instance to kv error, %v", err)
		return err
	}

	return nil
}

// 批量删除实例，flag=1
func (i *instanceStore) BatchDeleteInstances(ids []interface{}) error {

	if len(ids) == 0 {
		return nil
	}

	var realIds []string

	for _, id := range ids {
		realIds = append(realIds, id.(string))
	}

	if err := i.handler.DeleteValues(InstanceStoreType, realIds); err != nil{
		log.Errorf("update instance to kv error, %v", err)
		return err
	}
	return nil
}

// 清空一个实例，真正删除
func (i *instanceStore) CleanInstance(instanceID string) error {
	if err := i.handler.DeleteValues(InstanceStoreType, []string{instanceID}); err != nil{
		log.Errorf("update instance to kv error, %v", err)
		return err
	}
	return nil
}

// 检查ID是否存在，并且返回所有ID的查询结果
func (i *instanceStore) CheckInstancesExisted(ids map[string]bool) (map[string]bool, error) {

	if len(ids) == 0 {
		return nil, nil
	}

	//err := i.handler.IterateFields(InstanceStoreType, "id", func(ins interface{}){
	//	instance := ins.(model.Instance)
	//
	//	_, ok := ids[instance.ID()]
	//	if ok {
	//		ids[instance.ID()] = true
	//	}
	//})
	//
	//if err != nil {
	//	log.Errorf("list instance in kv error, %v", err)
	//	return nil, err
	//}

	return ids, nil
}

// 获取实例关联的token
func (i *instanceStore) GetInstancesBrief(ids map[string]bool) (map[string]*model.Instance, error) {

	if len(ids) == 0 {
		return nil, nil
	}

	fields := []string{"id"}

	// 找到全部的实例
	inss, err := i.handler.LoadValuesByFilter(InstanceStoreType, fields, model.Instance{},
		func(m map[string]interface{}) bool{
			id := m["id"].(string)
			_, ok := ids[id]
			if ok {
				return true
			}
			return false
		})
	if err != nil {
		log.Errorf("load instance error, %v", err)
		return nil, err
	}

	// 找到实例对应的 service，得到 serviceToken
	serviceIDs := make(map[string]bool)
	for _, ins := range inss {
		serviceID := ins.(model.Instance).ServiceID
		serviceIDs[serviceID] = true
	}

	services, err := i.handler.LoadValuesByFilter(ServiceStoreType, fields, model.Service{},
		func(m map[string]interface{}) bool{
			id := m["id"].(string)
			_, ok := serviceIDs[id]
			if ok {
				return true
			}
			return false
		})

	// 组合数据
	out := make(map[string]*model.Instance, len(ids))
	var item model.ExpandInstanceStore
	var instance model.InstanceStore
	item.ServiceInstance = &instance

	for _, ins := range inss {
		tempIns := ins.(model.Instance)
		svc, ok := services[tempIns.ServiceID]
		if !ok {
			log.Errorf("can not find instance service , instanceId is %s", tempIns.ID())
			return nil, errors.New("can not find instance service")
		}
		tempService := svc.(model.Service)
		instance.ID = tempIns.ID()
		instance.Host = tempIns.Host()
		instance.Port = tempIns.Port()
		item.ServiceName = tempService.Name
		item.Namespace = tempService.Namespace
		item.ServiceToken = tempService.Token
		item.ServicePlatformID = tempService.PlatformID

		out[instance.ID] = model.ExpandStore2Instance(&item)
	}


	return out, nil
}

// 查询一个实例的详情，只返回有效的数据
func (i *instanceStore) GetInstance(instanceID string) (*model.Instance, error) {

	fields := []string{"id"}

	ins, err := i.handler.LoadValuesByFilter(InstanceStoreType, fields, model.Instance{},
		func(m map[string]interface{}) bool{
			id := m["id"].(string)
			if id == instanceID {
				return true
			}
			return false
		})
	if err != nil {
		log.Errorf("load instance from kv error, %v", err)
		return nil, err
	}
	instance := ins[instanceID].(model.Instance)
	return &instance, nil
}

// 获取有效的实例总数
func (i *instanceStore) GetInstancesCount() (uint32, error) {

	count, err := i.handler.CountValues(InstanceStoreType)
	if err != nil {
		log.Errorf("get instance count error, %v", err)
		return 0, err
	}

	return uint32(count), nil
}

// 根据服务和Host获取实例（不包括metadata）
func (i *instanceStore) GetInstancesMainByService(serviceID, host string) ([]*model.Instance, error) {

	// select by service_id and host
	fields := []string{"service_id", "host"}

	instances, err := i.handler.LoadValuesByFilter(InstanceStoreType, fields, model.Instance{},
		func(m map[string]interface{}) bool{
			svcId := m["service_id"].(string)
			h := m["host"].(string)
			if svcId != serviceID {
				return false
			}
			if h != host {
				return false
			}
			return true
		})
	if err != nil {
		log.Errorf("load instance from kv error, %v", err)
		return nil, err
	}

	return getRealInstancesList(instances, 0, uint32(len(instances))), nil
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


// 将下层返回的全量的 service map 转为有序的 list，并根据 offset/limit 返回结果
func getRealInstancesList(originServices map[string]interface{}, offset, limit uint32) []*model.Instance {
	instances := make([]*model.Instance, 0)
	beginIndex := offset
	endIndex := beginIndex + limit
	totalCount := uint32(len(originServices))
	// 处理异常的 offset、 limit
	if totalCount == 0 {
		return instances
	}
	if beginIndex >= endIndex {
		return instances
	}
	if beginIndex >= totalCount {
		return instances
	}
	if endIndex > totalCount {
		endIndex = totalCount
	}

	for _, s := range originServices {
		instances = append(instances, s.(*model.Instance))
	}

	sort.Slice(instances, func (i, j int) bool{
		// modifyTime 由近到远排序
		if instances[i].ModifyTime.After(instances[j].ModifyTime) {
			return true
		} else if instances[i].ModifyTime.Before(instances[j].ModifyTime){
			return false
		}else{
			// modifyTime 相同则比较id
			return strings.Compare(instances[i].ID(), instances[j].ID()) < 0
		}
	})

	return instances[beginIndex:endIndex]
}