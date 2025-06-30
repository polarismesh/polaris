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
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/wrappers"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	bolt "go.etcd.io/bbolt"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/polarismesh/polaris/common/model"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

type instanceStore struct {
	handler BoltHandler
}

const (
	tblNameInstance    = "instance"
	insFieldProto      = "Proto"
	insFieldServiceID  = "ServiceID"
	insFieldModifyTime = "ModifyTime"
	insFieldValid      = "Valid"
)

// AddInstance add an instance
func (i *instanceStore) AddInstance(instance *model.Instance) error {
	initInstance([]*model.Instance{instance})
	// Before adding new data, you must clean up the old data
	if err := i.handler.DeleteValues(tblNameInstance, []string{instance.ID()}); err != nil {
		log.Errorf("[Store][boltdb] delete instance to kv error, %v", err)
		return err
	}

	if err := i.handler.SaveValue(tblNameInstance, instance.ID(), instance); err != nil {
		log.Errorf("[Store][boltdb] save instance to kv error, %v", err)
		return err
	}

	return nil
}

// BatchAddInstances Add multiple instances
func (i *instanceStore) BatchAddInstances(instances []*model.Instance) error {

	if len(instances) == 0 {
		return nil
	}

	var insIds []string
	for _, instance := range instances {
		insIds = append(insIds, instance.ID())
	}

	// clear old instances
	if err := i.handler.DeleteValues(tblNameInstance, insIds); err != nil {
		log.Errorf("[Store][boltdb] save instance to kv error, %v", err)
		return err
	}

	initInstance(instances)
	for _, instance := range instances {
		if err := i.handler.SaveValue(tblNameInstance, instance.ID(), instance); err != nil {
			log.Errorf("[Store][boltdb] save instance to kv error, %v", err)
			return err
		}
	}

	return nil
}

// UpdateInstance Update instance
func (i *instanceStore) UpdateInstance(instance *model.Instance) error {

	properties := make(map[string]interface{})
	properties[insFieldProto] = instance.Proto
	curr := time.Now()
	properties[insFieldModifyTime] = curr
	instance.Proto.Mtime = &wrappers.StringValue{Value: commontime.Time2String(curr)}

	if err := i.handler.UpdateValue(tblNameInstance, instance.ID(), properties); err != nil {
		log.Errorf("[Store][boltdb] update instance to kv error, %v", err)
		return err
	}

	return nil
}

// DeleteInstance Delete an instance
func (i *instanceStore) DeleteInstance(instanceID string) error {

	properties := make(map[string]interface{})
	properties[insFieldValid] = false
	properties[insFieldModifyTime] = time.Now()

	if err := i.handler.UpdateValue(tblNameInstance, instanceID, properties); err != nil {
		log.Errorf("[Store][boltdb] delete instance from kv error, %v", err)
		return err
	}

	return nil
}

// BatchDeleteInstances Delete instances in batch
func (i *instanceStore) BatchDeleteInstances(ids []interface{}) error {

	if len(ids) == 0 {
		return nil
	}

	for _, id := range ids {

		properties := make(map[string]interface{})
		properties[insFieldValid] = false
		properties[insFieldModifyTime] = time.Now()

		if err := i.handler.UpdateValue(tblNameInstance, id.(string), properties); err != nil {
			log.Errorf("[Store][boltdb] batch delete instance from kv error, %v", err)
			return err
		}
	}
	return nil
}

// CleanInstance Delete an instance
func (i *instanceStore) CleanInstance(instanceID string) error {
	if err := i.handler.DeleteValues(tblNameInstance, []string{instanceID}); err != nil {
		log.Errorf("[Store][boltdb] delete instance from kv error, %v", err)
		return err
	}
	return nil
}

// BatchGetInstanceIsolate Check whether the ID exists, and return the query results of all IDs
func (i *instanceStore) BatchGetInstanceIsolate(ids map[string]bool) (map[string]bool, error) {

	if len(ids) == 0 {
		return nil, nil
	}

	keys := make([]string, len(ids))
	pos := 0
	for k := range ids {
		keys[pos] = k
		pos++
	}

	result, err := i.handler.LoadValues(tblNameInstance, keys, &model.Instance{})
	if err != nil {
		log.Errorf("[Store][boltdb] list instance in kv error, %v", err)
		return nil, err
	}

	if len(result) == 0 {
		return ids, nil
	}

	for id, val := range result {
		ins := val.(*model.Instance)
		if !ins.Valid {
			continue
		}

		ids[id] = ins.Proto.GetIsolate().GetValue()
	}

	return ids, nil
}

// GetInstancesBrief Get the token associated with the instance
func (i *instanceStore) GetInstancesBrief(ids map[string]bool) (map[string]*model.Instance, error) {

	if len(ids) == 0 {
		return nil, nil
	}

	fields := []string{insFieldProto, insFieldValid}

	// find all instances with given ids
	inss, err := i.handler.LoadValuesByFilter(tblNameInstance, fields, &model.Instance{},
		func(m map[string]interface{}) bool {
			valid, ok := m[insFieldValid]
			if ok && !valid.(bool) {
				return false
			}

			insProto, ok := m[insFieldProto]
			if !ok {
				return false
			}
			id := insProto.(*apiservice.Instance).GetId().GetValue()
			_, ok = ids[id]
			return ok
		})
	if err != nil {
		log.Errorf("[Store][boltdb] load instance error, %v", err)
		return nil, err
	}

	// find the service corresponding to the instance and get the serviceToken
	serviceIDs := make(map[string]bool)
	for _, ins := range inss {
		serviceID := ins.(*model.Instance).ServiceID
		serviceIDs[serviceID] = true
	}

	fields = []string{SvcFieldID, SvcFieldValid}
	services, err := i.handler.LoadValuesByFilter(tblNameService, fields, &Service{},
		func(m map[string]interface{}) bool {
			valid, ok := m[insFieldValid]
			if ok && !valid.(bool) {
				return false
			}

			svcId, ok := m[SvcFieldID]
			if !ok {
				return false
			}
			id := svcId.(string)
			_, ok = serviceIDs[id]
			return ok
		})

	// assemble return data
	out := make(map[string]*model.Instance, len(ids))
	var item model.ExpandInstanceStore
	var instance model.InstanceStore
	item.ServiceInstance = &instance

	for _, ins := range inss {
		tempIns := ins.(*model.Instance)
		svc, ok := services[tempIns.ServiceID]
		if !ok {
			log.Errorf("[Store][boltdb] can not find instance service , instanceId is %s", tempIns.ID())
			return nil, errors.New("can not find instance service")
		}
		tempService := svc.(*Service)
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

// GetInstance Query the details of an instance
func (i *instanceStore) GetInstance(instanceID string) (*model.Instance, error) {
	fields := []string{insFieldProto, insFieldValid}
	ins, err := i.handler.LoadValuesByFilter(tblNameInstance, fields, &model.Instance{},
		func(m map[string]interface{}) bool {
			if insValid, _ := m[insFieldValid].(bool); !insValid {
				return false
			}
			insProto, ok := m[insFieldProto]
			if !ok {
				return false
			}
			id := insProto.(*apiservice.Instance).GetId().GetValue()
			return id == instanceID
		})
	if err != nil {
		log.Errorf("[Store][boltdb] load instance from kv error, %v", err)
		return nil, err
	}
	instance, ok := ins[instanceID]
	if !ok {
		return nil, nil
	}
	return instance.(*model.Instance), nil
}

// GetInstancesCount Get the total number of instances
func (i *instanceStore) GetInstancesCount() (uint32, error) {

	count, err := i.handler.CountValues(tblNameInstance)
	if err != nil {
		log.Errorf("[Store][boltdb] get instance count error, %v", err)
		return 0, err
	}

	return uint32(count), nil
}

// GetInstancesCountTx Get the total number of instances
func (i *instanceStore) GetInstancesCountTx(tx store.Tx) (uint32, error) {
	dbTx, _ := tx.GetDelegateTx().(*bolt.Tx)
	count, err := countValues(dbTx, tblNameInstance)
	if err != nil {
		log.Errorf("[Store][boltdb] get instance count error, %v", err)
		return 0, err
	}

	return uint32(count), nil
}

// GetInstancesMainByService Get instances based on service and Host
func (i *instanceStore) GetInstancesMainByService(serviceID, host string) ([]*model.Instance, error) {

	// select by service_id and host
	fields := []string{insFieldServiceID, insFieldProto, insFieldValid}

	instances, err := i.handler.LoadValuesByFilter(tblNameInstance, fields, &model.Instance{},
		func(m map[string]interface{}) bool {
			valid, ok := m[insFieldValid]
			if ok && !valid.(bool) {
				return false
			}

			sId, ok := m[insFieldServiceID]
			if !ok {
				return false
			}
			insProto, ok := m[insFieldProto]
			if !ok {
				return false
			}

			svcId := sId.(string)
			h := insProto.(*apiservice.Instance).GetHost().GetValue()
			if svcId != serviceID {
				return false
			}
			if h != host {
				return false
			}
			return true
		})
	if err != nil {
		log.Errorf("[Store][boltdb] load instance from kv error, %v", err)
		return nil, err
	}

	return getRealInstancesList(instances, 0, uint32(len(instances))), nil
}

// GetExpandInstances View instance details and corresponding number according to filter conditions
func (i *instanceStore) GetExpandInstances(filter, metaFilter map[string]string,
	offset uint32, limit uint32) (uint32, []*model.Instance, error) {
	if limit == 0 {
		return 0, make([]*model.Instance, 0), nil
	}

	// find service
	name, isServiceName := filter["name"]
	namespace, isNamespace := filter["namespace"]

	svcIDFilterSet := make(map[string]struct{}, 0)
	if isNamespace || isServiceName {
		sStore := serviceStore{handler: i.handler}
		svcs, err := sStore.GetServiceByNameAndNamespace(name, namespace)
		if err != nil {
			log.Errorf("[Store][boltdb] find service error, %v", err)
			return 0, nil, err
		}
		for _, svc := range svcs {
			svcIDFilterSet[svc.ID] = struct{}{}
		}
		if len(svcIDFilterSet) == 0 {
			return 0, make([]*model.Instance, 0), nil
		}
	}

	svcIdsTmp := make(map[string]struct{})
	fields := []string{insFieldProto, insFieldServiceID, insFieldValid}

	instances, err := i.handler.LoadValuesByFilter(tblNameInstance, fields, &model.Instance{},
		func(m map[string]interface{}) bool {
			valid, ok := m[insFieldValid]
			if ok && !valid.(bool) {
				return false
			}

			insProto, ok := m[insFieldProto]
			if !ok {
				return false
			}
			ins := insProto.(*apiservice.Instance)
			host, isHost := filter["host"]
			port, isPort := filter["port"]
			protocol, isProtocol := filter["protocol"]
			version, isVersion := filter["version"]
			healthy, isHealthy := filter["health_status"]
			isolate, isIsolate := filter["isolate"]
			id, isId := filter["id"]

			if isId {
				if utils.IsWildName(id) {
					if !utils.IsWildMatch(ins.GetId().GetValue(), id) {
						return false
					}
				} else {
					if id != ins.GetId().GetValue() {
						return false
					}
				}
			}
			if isHost && host != ins.GetHost().GetValue() {
				return false
			}
			if isPort && port != strconv.Itoa(int(ins.GetPort().GetValue())) {
				return false
			}
			if isProtocol && protocol != ins.GetProtocol().GetValue() {
				return false
			}
			if isVersion && version != ins.GetVersion().GetValue() {
				return false
			}
			if isHealthy && compareParam2BoolNotEqual(healthy, ins.GetHealthy().GetValue()) {
				return false
			}
			if isIsolate && compareParam2BoolNotEqual(isolate, ins.GetIsolate().GetValue()) {
				return false
			}

			// 如果提供了 serviceName 或者 namespaceName 才过滤 serviceID
			if isServiceName || isNamespace {
				// filter serviceID
				sID, ok := m["ServiceID"]
				if !ok {
					return false
				}
				sIDStr, strOK := sID.(string)
				if !strOK {
					return false
				}
				if _, ok := svcIDFilterSet[sIDStr]; !ok {
					return false
				}
			}

			// filter metadata
			if len(metaFilter) > 0 {
				var key, value string
				for k, v := range metaFilter {
					key = k
					value = v
					break
				}

				insV, ok := ins.GetMetadata()[key]
				if !ok || insV != value {
					return false
				}
			}
			svcIdsTmp[m["ServiceID"].(string)] = struct{}{}
			return true
		})
	if err != nil {
		log.Errorf("[Store][boltdb] load instance from kv error, %v", err)
		return 0, nil, err
	}

	svcIds := make([]string, len(svcIdsTmp))
	pos := 0
	for k := range svcIdsTmp {
		svcIds[pos] = k
		pos++
	}
	svcRets, err := i.handler.LoadValuesAll(tblNameService, &Service{})
	if err != nil {
		log.Errorf("[Store][boltdb] load service from kv error, %v", err)
		return 0, nil, err
	}
	for _, v := range instances {
		ins := v.(*model.Instance)
		service, ok := svcRets[ins.ServiceID]
		if !ok {
			log.Errorf("[Store][boltdb] no found instance relate service, "+
				"instance-id: %s, service-id: %s", ins.ID(), ins.ServiceID)
			return 0, nil, errors.New("no found instance relate service")
		}
		ins.Proto.Service = wrapperspb.String(service.(*Service).Name)
		ins.Proto.Namespace = wrapperspb.String(service.(*Service).Namespace)
	}

	totalCount := uint32(len(instances))

	return totalCount, getRealInstancesList(instances, offset, limit), nil
}

// GetMoreInstances Get incremental instances according to mtime
func (i *instanceStore) GetMoreInstances(tx store.Tx, mtime time.Time, firstUpdate, needMeta bool,
	serviceID []string) (map[string]*model.Instance, error) {

	dbTx, _ := tx.GetDelegateTx().(*bolt.Tx)

	fields := []string{insFieldProto, insFieldServiceID, insFieldValid}
	svcIdMap := make(map[string]bool)
	for _, s := range serviceID {
		svcIdMap[s] = true
	}

	instances := make(map[string]interface{})
	err := loadValuesByFilter(dbTx, tblNameInstance, fields, &model.Instance{},
		func(m map[string]interface{}) bool {

			if firstUpdate {
				valid, ok := m[insFieldValid]
				if ok && !valid.(bool) {
					return false
				}
			}

			insProto, ok := m[insFieldProto]
			if !ok {
				return false
			}
			svcId, ok := m[insFieldServiceID]
			if !ok {
				return false
			}
			ins := insProto.(*apiservice.Instance)
			serviceId := svcId.(string)

			insMtime, err := time.Parse("2006-01-02 15:04:05", ins.GetMtime().GetValue())
			if err != nil {
				log.Errorf("[Store][boltdb] parse instance mtime error, %v", err)
				return false
			}

			if insMtime.Before(mtime) {
				return false
			}

			if len(svcIdMap) > 0 {
				_, ok = svcIdMap[serviceId]
				if !ok {
					return false
				}
			}

			return true
		}, instances)

	if err != nil {
		log.Errorf("[Store][boltdb] load instance from kv error, %v", err)
		return nil, err
	}

	return toInstance(instances), nil
}

// GetMoreInstancesConsoles 获取增量修改数据
func (ins *instanceStore) GetMoreInstanceConsoles(tx store.Tx, mtime time.Time, firstUpdate, needMeta bool,
	serviceID []string) (map[string]*model.InstanceConsole, error) {
	//Todo: get from boltdb
	return nil, nil
}

// DeleteInstanceConsole 逻辑删除instanceConsole
func (ins *instanceStore) DeleteInstanceConsole(instanceConsoleID string) error {
	//Todo: delete from boltdb
	return nil
}

// CleanInstanceConsole 物理删除instanceConsole
func (ins *instanceStore) CleanInstanceConsole(instanceConsoleID string) error {
	//Todo: clean from boltdb
	return nil
}

// UpdateInstanceConsole 更新instanceConsole
func (ins *instanceStore) UpdateInstanceConsole(instanceConsole *model.InstanceConsole) error {
	//Todo: update from boltdb
	return nil
}

// BatchSetInstanceHealthStatus 批量设置实例的健康状态
func (i *instanceStore) BatchSetInstanceHealthStatus(ids []interface{}, healthy int, revision string) error {
	for _, id := range ids {
		if err := i.SetInstanceHealthStatus(id.(string), healthy, revision); err != nil {
			return err
		}
	}
	return nil
}

// SetInstanceHealthStatus Set the health status of the instance
func (i *instanceStore) SetInstanceHealthStatus(instanceID string, flag int, revision string) error {

	// get instance
	fields := []string{insFieldProto}

	instances, err := i.handler.LoadValuesByFilter(tblNameInstance, fields, &model.Instance{},
		func(m map[string]interface{}) bool {
			insProto, ok := m[insFieldProto]
			if !ok {
				return false
			}
			insId := insProto.(*apiservice.Instance).GetId().GetValue()

			return insId == instanceID
		})
	if err != nil {
		log.Errorf("[Store][boltdb] load instance from kv error, %v", err)
		return err
	}
	if len(instances) == 0 {
		log.Errorf("cant not find instance in kv, %s", instanceID)
		return nil
	}

	// set status
	ins := instances[instanceID].(*model.Instance)
	var healthy bool
	if flag == 0 {
		healthy = false
	} else {
		healthy = true
	}
	ins.Proto.Healthy.Value = healthy
	ins.Proto.Revision.Value = revision

	properties := make(map[string]interface{})
	properties[insFieldProto] = ins.Proto
	curr := time.Now()
	properties[insFieldModifyTime] = curr
	ins.Proto.Mtime = &wrappers.StringValue{Value: commontime.Time2String(curr)}

	err = i.handler.UpdateValue(tblNameInstance, instanceID, properties)
	if err != nil {
		log.Errorf("[Store][boltdb] update instance error %v", err)
		return err
	}

	return nil
}

// BatchSetInstanceIsolate Modify the isolation status of instances in batches
func (i *instanceStore) BatchSetInstanceIsolate(ids []interface{}, isolate int, revision string) error {

	insIds := make(map[string]bool)
	for _, id := range ids {
		insIds[id.(string)] = true
	}
	var isolateStatus bool
	if isolate == 0 {
		isolateStatus = false
	} else {
		isolateStatus = true
	}

	fields := []string{insFieldProto}

	// get all instances by given ids
	instances, err := i.handler.LoadValuesByFilter(tblNameInstance, fields, &model.Instance{},
		func(m map[string]interface{}) bool {
			proto, ok := m[insFieldProto]
			if !ok {
				return false
			}
			insId := proto.(*apiservice.Instance).GetId().GetValue()

			_, ok = insIds[insId]
			return ok
		})
	if err != nil {
		log.Errorf("[Store][boltdb] get instance from kv error, %v", err)
		return err
	}
	if len(instances) == 0 {
		log.Errorf("cant not find instance in kv, %v", ids)
		return nil
	}

	for id, ins := range instances {
		instance := ins.(*model.Instance).Proto
		instance.Isolate.Value = isolateStatus
		instance.Revision.Value = revision

		properties := make(map[string]interface{})
		properties[insFieldProto] = instance
		curr := time.Now()
		properties[insFieldModifyTime] = curr
		instance.Mtime = &wrappers.StringValue{Value: commontime.Time2String(curr)}
		err = i.handler.UpdateValue(tblNameInstance, id, properties)
		if err != nil {
			log.Error("[Store][boltdb] update instance in set instance isolate error", zap.Error(err))
			return err
		}
	}

	return nil
}

// BatchAppendInstanceMetadata 追加实例 metadata
func (i *instanceStore) BatchAppendInstanceMetadata(requests []*store.InstanceMetadataRequest) error {
	if len(requests) == 0 {
		return nil
	}
	return i.handler.Execute(true, func(tx *bolt.Tx) error {
		values := map[string]interface{}{}
		fields := []string{insFieldProto, insFieldValid}
		if err := loadValuesByFilter(tx, tblNameInstance, fields, &model.Instance{},
			func(m map[string]interface{}) bool {
				valid, ok := m[insFieldValid]
				if ok && !valid.(bool) {
					return false
				}
				proto, ok := m[insFieldProto]
				if !ok {
					return false
				}
				insId := proto.(*apiservice.Instance).GetId().GetValue()
				for i := range requests {
					if requests[i].InstanceID == insId {
						return true
					}
				}
				return false
			}, values); err != nil {
			log.Errorf("[Store][boltdb] do batch append InstanceMetadata get instances error, %v", err)
			return err
		}
		if len(values) == 0 {
			return nil
		}
		for i := range requests {
			instanceID := requests[i].InstanceID
			val, ok := values[instanceID]
			if !ok {
				return nil
			}
			ins := val.(*model.Instance)
			if len(ins.Proto.GetMetadata()) == 0 {
				ins.Proto.Metadata = map[string]string{}
			}
			for k, v := range requests[i].Metadata {
				ins.Proto.Metadata[k] = v
			}
			properties := make(map[string]interface{})
			properties[insFieldProto] = ins.Proto
			properties[CommonFieldRevision] = requests[i].Revision
			properties[insFieldModifyTime] = time.Now()
			if err := updateValue(tx, tblNameInstance, instanceID, properties); err != nil {
				log.Errorf("[Store][boltdb] do batch append InstanceMetadata update instance by %s error, %v",
					instanceID, err)
				return err
			}
		}
		return nil
	})
}

// BatchRemoveInstanceMetadata 删除实例指定的 metadata
func (i *instanceStore) BatchRemoveInstanceMetadata(requests []*store.InstanceMetadataRequest) error {
	if len(requests) == 0 {
		return nil
	}
	return i.handler.Execute(true, func(tx *bolt.Tx) error {
		values := map[string]interface{}{}
		fields := []string{insFieldProto, insFieldValid}
		if err := loadValuesByFilter(tx, tblNameInstance, fields, &model.Instance{},
			func(m map[string]interface{}) bool {
				valid, ok := m[insFieldValid]
				if ok && !valid.(bool) {
					return false
				}
				proto, ok := m[insFieldProto]
				if !ok {
					return false
				}
				insId := proto.(*apiservice.Instance).GetId().GetValue()
				for i := range requests {
					if requests[i].InstanceID == insId {
						return true
					}
				}
				return false
			}, values); err != nil {
			log.Errorf("[Store][boltdb] do batch remove InstanceMetadata get instances error, %v", err)
			return err
		}
		if len(values) == 0 {
			return nil
		}
		for i := range requests {
			instanceID := requests[i].InstanceID
			val, ok := values[instanceID]
			if !ok {
				continue
			}
			ins := val.(*model.Instance)
			if len(ins.Proto.GetMetadata()) == 0 {
				ins.Proto.Metadata = map[string]string{}
			}
			for p := range requests[i].Keys {
				delete(ins.Proto.Metadata, requests[i].Keys[p])
			}
			properties := make(map[string]interface{})
			properties[insFieldProto] = ins.Proto
			properties[CommonFieldRevision] = requests[i].Revision
			properties[insFieldModifyTime] = time.Now()
			if err := updateValue(tx, tblNameInstance, instanceID, properties); err != nil {
				log.Errorf("[Store][boltdb] do batch remove InstanceMetadata update instance by %s error, %v",
					instanceID, err)
				return err
			}
		}
		return nil
	})
}

func toInstance(m map[string]interface{}) map[string]*model.Instance {
	insMap := make(map[string]*model.Instance)
	for k, v := range m {
		insMap[k] = v.(*model.Instance)
	}

	return insMap
}

func getRealInstancesList(originServices map[string]interface{}, offset, limit uint32) []*model.Instance {
	instances := make([]*model.Instance, 0)
	beginIndex := offset
	endIndex := beginIndex + limit
	totalCount := uint32(len(originServices))
	// handle invalid limit offset
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

	sort.Slice(instances, func(i, j int) bool {
		// sort by modify time
		if instances[i].ModifyTime.After(instances[j].ModifyTime) {
			return true
		} else if instances[i].ModifyTime.Before(instances[j].ModifyTime) {
			return false
		}
		return strings.Compare(instances[i].ID(), instances[j].ID()) < 0
	})

	return instances[beginIndex:endIndex]
}

func initInstance(instance []*model.Instance) {

	if len(instance) == 0 {
		return
	}

	for _, ins := range instance {
		if ins != nil {
			currT := time.Now()
			timeStamp := commontime.Time2String(currT)
			if ins.Proto != nil {
				if ins.Proto.GetMtime().GetValue() == "" {
					ins.Proto.Mtime = &wrappers.StringValue{Value: timeStamp}
				}
				if ins.Proto.GetCtime().GetValue() == "" {
					ins.Proto.Ctime = &wrappers.StringValue{Value: timeStamp}
				}
			}
			ins.Valid = true
			ins.ModifyTime = currT
		}
	}
}

func compareParam2BoolNotEqual(param string, b bool) bool {
	if param == "0" && !b {
		return false
	}
	if param == "1" && b {
		return false
	}
	return true
}
