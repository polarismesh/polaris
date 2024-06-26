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
	"context"
	"encoding/json"
	"sort"
	"time"

	bolt "go.etcd.io/bbolt"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

const (
	tblServiceContract = "service_contract"

	ContractFieldID         = "ID"
	ContractFieldNamespace  = "Namespace"
	ContractFieldService    = "Service"
	ContractFieldType       = "Type"
	ContractFieldProtocol   = "Protocol"
	ContractFieldVersion    = "Version"
	ContractFieldRevision   = "Revision"
	ContractFieldContent    = "Content"
	ContractFieldInterfaces = "Interfaces"
	ContractFieldModifyTime = "ModifyTime"
	ContractFieldValid      = "Valid"
)

type serviceContractStore struct {
	handler BoltHandler
}

// CreateServiceContract 创建服务契约
func (s *serviceContractStore) CreateServiceContract(contract *model.ServiceContract) error {
	tn := time.Now()
	contract.Valid = true
	contract.CreateTime = tn
	contract.ModifyTime = tn

	if err := s.handler.SaveValue(tblServiceContract, contract.ID, s.toStore(&model.EnrichServiceContract{
		ServiceContract: contract,
	})); err != nil {
		return err
	}
	return nil
}

// UpdateServiceContract .
func (s *serviceContractStore) UpdateServiceContract(contract *model.ServiceContract) error {
	properties := map[string]interface{}{
		ContractFieldRevision:   contract.Revision,
		ContractFieldContent:    contract.Content,
		ContractFieldModifyTime: time.Now(),
	}

	if err := s.handler.UpdateValue(tblServiceContract, contract.ID, properties); err != nil {
		return err
	}
	return nil
}

// DeleteServiceContract 删除服务契约
func (s *serviceContractStore) DeleteServiceContract(contract *model.ServiceContract) error {
	properties := map[string]interface{}{
		ContractFieldValid:      false,
		ContractFieldModifyTime: time.Now(),
	}

	if err := s.handler.UpdateValue(tblServiceContract, contract.ID, properties); err != nil {
		return err
	}
	return nil
}

// GetServiceContract .
func (s *serviceContractStore) GetServiceContract(id string) (*model.EnrichServiceContract, error) {
	values, err := s.handler.LoadValues(tblServiceContract, []string{id}, &ServiceContract{})
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, nil
	}
	ret := values[id].(*ServiceContract)
	if !ret.Valid {
		return nil, nil
	}
	return s.toModel(ret), nil
}

// AddServiceContractInterfaces 创建服务契约API接口
func (s *serviceContractStore) AddServiceContractInterfaces(contract *model.EnrichServiceContract) error {
	return s.handler.Execute(true, func(tx *bolt.Tx) error {
		values := map[string]interface{}{}
		if err := loadValues(tx, tblServiceContract, []string{contract.ID}, &ServiceContract{}, values); err != nil {
			return err
		}
		if len(values) == 0 {
			return store.NewStatusError(store.NotFoundResource, "not found target service_contract")
		}
		tN := time.Now()
		for i := range contract.Interfaces {
			contract.Interfaces[i].CreateTime = tN
			contract.Interfaces[i].ModifyTime = tN
		}

		record := values[contract.ID].(*ServiceContract)
		enrichRecord := s.toModel(record)
		enrichRecord.Interfaces = contract.Interfaces
		enrichRecord.Revision = contract.Revision
		enrichRecord.ModifyTime = time.Now()

		return saveValue(tx, tblServiceContract, contract.ID, s.toStore(enrichRecord))
	})
}

// AppendServiceContractInterfaces 追加服务契约API接口
func (s *serviceContractStore) AppendServiceContractInterfaces(contract *model.EnrichServiceContract) error {
	return s.handler.Execute(true, func(tx *bolt.Tx) error {
		values := map[string]interface{}{}
		if err := loadValues(tx, tblServiceContract, []string{contract.ID}, &ServiceContract{}, values); err != nil {
			return err
		}
		if len(values) == 0 {
			return store.NewStatusError(store.NotFoundResource, "not found target service_contract")
		}

		record := values[contract.ID].(*ServiceContract)
		enrichRecord := s.toModel(record)

		interfaceMap := make(map[string]*model.InterfaceDescriptor, len(enrichRecord.Interfaces))
		for i := range enrichRecord.Interfaces {
			interfaceMap[enrichRecord.Interfaces[i].ID] = enrichRecord.Interfaces[i]
		}
		tN := time.Now()
		for i := range contract.Interfaces {
			contract.Interfaces[i].ModifyTime = tN
			contract.Interfaces[i].CreateTime = tN
			interfaceMap[contract.Interfaces[i].ID] = contract.Interfaces[i]
		}
		interfaceSlice := make([]*model.InterfaceDescriptor, 0, len(interfaceMap))
		for i := range interfaceMap {
			interfaceSlice = append(interfaceSlice, interfaceMap[i])
		}

		enrichRecord.Interfaces = interfaceSlice
		enrichRecord.Revision = contract.Revision
		enrichRecord.ModifyTime = time.Now()

		return saveValue(tx, tblServiceContract, contract.ID, s.toStore(enrichRecord))
	})
}

// DeleteServiceContractInterfaces 删除服务契约API接口
func (s *serviceContractStore) DeleteServiceContractInterfaces(contract *model.EnrichServiceContract) error {
	return s.handler.Execute(true, func(tx *bolt.Tx) error {
		values := map[string]interface{}{}
		if err := loadValues(tx, tblServiceContract, []string{contract.ID}, &ServiceContract{}, values); err != nil {
			return err
		}
		if len(values) == 0 {
			return store.NewStatusError(store.NotFoundResource, "not found target service_contract")
		}

		record := values[contract.ID].(*ServiceContract)
		enrichRecord := s.toModel(record)

		interfaceMap := make(map[string]*model.InterfaceDescriptor, len(enrichRecord.Interfaces))
		for i := range enrichRecord.Interfaces {
			interfaceMap[enrichRecord.Interfaces[i].ID] = enrichRecord.Interfaces[i]
		}
		for i := range contract.Interfaces {
			delete(interfaceMap, contract.Interfaces[i].ID)
		}
		interfaceSlice := make([]*model.InterfaceDescriptor, 0, len(interfaceMap))
		for i := range interfaceMap {
			interfaceSlice = append(interfaceSlice, interfaceMap[i])
		}

		enrichRecord.Interfaces = interfaceSlice
		enrichRecord.Revision = contract.Revision
		enrichRecord.ModifyTime = time.Now()

		return saveValue(tx, tblServiceContract, contract.ID, s.toStore(enrichRecord))
	})
}

var (
	searchContractFields = []string{
		ContractFieldType,
		ContractFieldNamespace,
		ContractFieldService,
		ContractFieldVersion,
		ContractFieldProtocol,
		ContractFieldValid,
	}

	searchInterfaceFields = []string{
		ContractFieldType,
		ContractFieldNamespace,
		ContractFieldService,
		ContractFieldVersion,
		ContractFieldProtocol,
		ContractFieldValid,
		ContractFieldInterfaces,
	}
)

func (s *serviceContractStore) GetServiceContracts(ctx context.Context, filter map[string]string, offset,
	limit uint32) (uint32, []*model.EnrichServiceContract, error) {

	values, err := s.handler.LoadValuesByFilter(tblServiceContract, searchContractFields, &ServiceContract{},
		func(m map[string]interface{}) bool {
			valid, _ := m[ContractFieldValid].(bool)
			if !valid {
				return false
			}
			if searchNs, ok := filter["namespace"]; ok {
				saveNs, _ := m[ContractFieldNamespace].(string)
				if !utils.IsWildMatch(saveNs, searchNs) {
					return false
				}
			}
			if searchSvc, ok := filter["service"]; ok {
				saveSvc, _ := m[ContractFieldService].(string)
				if !utils.IsWildMatch(saveSvc, searchSvc) {
					return false
				}
			}
			if searchProtocol, ok := filter["protocol"]; ok {
				saveProtocol, _ := m[ContractFieldProtocol].(string)
				if !utils.IsWildMatch(saveProtocol, searchProtocol) {
					return false
				}
			}
			if searchVer, ok := filter["version"]; ok {
				saveVer, _ := m[ContractFieldVersion].(string)
				if !utils.IsWildMatch(saveVer, searchVer) {
					return false
				}
			}
			if searchType, ok := filter["type"]; ok {
				saveType, _ := m[ContractFieldType].(string)
				if !utils.IsWildMatch(saveType, searchType) {
					return false
				}
			}
			return true
		})
	if err != nil {
		return 0, nil, store.Error(err)
	}

	ret := make([]*model.EnrichServiceContract, 0, len(values))
	for _, v := range values {
		ret = append(ret, s.toModel(v.(*ServiceContract)))
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[j].ModifyTime.Before(ret[i].ModifyTime)
	})
	return uint32(len(values)), toServiceContractPage(ret, offset, limit), nil
}

func toServiceContractPage(result []*model.EnrichServiceContract, offset, limit uint32) []*model.EnrichServiceContract {
	// 所有符合条件的服务数量
	amount := uint32(len(result))
	// 判断 offset 和 limit 是否允许返回对应的服务
	if offset >= amount || limit == 0 {
		return nil
	}

	endIdx := offset + limit
	if endIdx > amount {
		endIdx = amount
	}
	return result[offset:endIdx]
}

// GetInterfaceDescriptors 查询服务接口列表
func (s *serviceContractStore) GetInterfaceDescriptors(ctx context.Context, filter map[string]string, offset,
	limit uint32) (uint32, []*model.InterfaceDescriptor, error) {

	result := make([]*model.InterfaceDescriptor, 0, limit)
	_, err := s.handler.LoadValuesByFilter(tblServiceContract, searchInterfaceFields, &ServiceContract{},
		func(m map[string]interface{}) bool {
			valid, _ := m[ContractFieldValid].(bool)
			if !valid {
				return false
			}
			if searchNs, ok := filter["namespace"]; ok {
				saveNs, _ := m[ContractFieldNamespace].(string)
				if saveNs != searchNs {
					return false
				}
			}
			if searchSvc, ok := filter["service"]; ok {
				saveSvc, _ := m[ContractFieldService].(string)
				if saveSvc != searchSvc {
					return false
				}
			}
			if searchProtocol, ok := filter["protocol"]; ok {
				saveProtocol, _ := m[ContractFieldProtocol].(string)
				if saveProtocol != searchProtocol {
					return false
				}
			}
			if searchVer, ok := filter["version"]; ok {
				saveVer, _ := m[ContractFieldVersion].(string)
				if searchVer != saveVer {
					return false
				}
			}
			interfaces := make([]*model.InterfaceDescriptor, 0, 4)
			_ = json.Unmarshal([]byte(m[ContractFieldInterfaces].(string)), &interfaces)
			for i := range interfaces {
				if searchType, ok := filter["type"]; ok {
					if searchType != interfaces[i].Type {
						continue
					}
				}
				if searchPath, ok := filter["path"]; ok {
					if searchPath != interfaces[i].Path {
						continue
					}
				}
				if searchMethod, ok := filter["method"]; ok {
					if searchMethod != interfaces[i].Method {
						continue
					}
				}
				if searchSource, ok := filter["source"]; ok {
					if searchSource != interfaces[i].Source.String() {
						continue
					}
				}
				result = append(result, interfaces[i])
			}

			return true
		})
	if err != nil {
		return 0, nil, store.Error(err)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[j].ModifyTime.Before(result[i].ModifyTime)
	})
	return uint32(len(result)), toServiceInterfacesPage(result, offset, limit), err
}

func toServiceInterfacesPage(result []*model.InterfaceDescriptor, offset, limit uint32) []*model.InterfaceDescriptor {
	// 所有符合条件的服务数量
	amount := uint32(len(result))
	// 判断 offset 和 limit 是否允许返回对应的服务
	if offset >= amount || limit == 0 {
		return nil
	}

	endIdx := offset + limit
	if endIdx > amount {
		endIdx = amount
	}
	return result[offset:endIdx]
}

// ListVersions .
func (s *serviceContractStore) ListVersions(ctx context.Context, service, namespace string) ([]*model.ServiceContract, error) {
	fields := []string{ContractFieldValid, ContractFieldNamespace, ContractFieldService}
	values, err := s.handler.LoadValuesByFilter(tblServiceContract, fields, &ServiceContract{},
		func(m map[string]interface{}) bool {
			valid, _ := m[ContractFieldValid].(bool)
			if !valid {
				return false
			}

			saveNs, _ := m[ContractFieldNamespace].(string)
			saveSvc, _ := m[ContractFieldService].(string)

			return saveNs == namespace && saveSvc == service
		})
	if err != nil {
		return nil, store.Error(err)
	}

	ret := make([]*model.EnrichServiceContract, 0, len(values))
	for _, v := range values {
		data := s.toModel(v.(*ServiceContract))
		data.Interfaces = nil
		data.Content = ""
		ret = append(ret, data)
	}
	return nil, nil
}

// GetMoreServiceContracts .
func (s *serviceContractStore) GetMoreServiceContracts(firstUpdate bool, mtime time.Time) ([]*model.EnrichServiceContract, error) {
	if firstUpdate {
		mtime = time.Unix(0, 0)
	}

	fields := []string{ContractFieldValid, ContractFieldModifyTime}
	values, err := s.handler.LoadValuesByFilter(tblServiceContract, fields, &ServiceContract{},
		func(m map[string]interface{}) bool {
			if firstUpdate {
				valid, _ := m[ContractFieldValid].(bool)
				if !valid {
					return false
				}
			}
			saveMtime, _ := m[ContractFieldModifyTime].(time.Time)
			return !saveMtime.Before(mtime)
		})
	if err != nil {
		return nil, store.Error(err)
	}

	ret := make([]*model.EnrichServiceContract, 0, len(values))
	for _, v := range values {
		ret = append(ret, s.toModel(v.(*ServiceContract)))
	}
	return ret, nil
}

func (s *serviceContractStore) toModel(data *ServiceContract) *model.EnrichServiceContract {
	interfaces := make([]*model.InterfaceDescriptor, 0, 4)
	_ = json.Unmarshal([]byte(data.Interfaces), &interfaces)
	ret := &model.EnrichServiceContract{
		ServiceContract: &model.ServiceContract{
			ID:         data.ID,
			Namespace:  data.Namespace,
			Service:    data.Service,
			Type:       data.Type,
			Protocol:   data.Protocol,
			Version:    data.Version,
			Revision:   data.Revision,
			Content:    data.Content,
			CreateTime: data.CreateTime,
			ModifyTime: data.ModifyTime,
			Valid:      data.Valid,
		},
		Interfaces: interfaces,
	}
	ret.Format()
	return ret
}

func (s *serviceContractStore) toStore(data *model.EnrichServiceContract) *ServiceContract {
	return &ServiceContract{
		ID:         data.ID,
		Namespace:  data.Namespace,
		Service:    data.Service,
		Type:       data.Type,
		Protocol:   data.Protocol,
		Version:    data.Version,
		Revision:   data.Revision,
		Content:    data.Content,
		Interfaces: utils.MustJson(data.Interfaces),
		CreateTime: data.CreateTime,
		ModifyTime: data.ModifyTime,
		Valid:      data.Valid,
	}
}

type ServiceContract struct {
	ID string
	// 所属命名空间
	Namespace string
	// 所属服务名称
	Service string
	// 契约名称
	Type string
	// 协议，http/grpc/dubbo/thrift
	Protocol string
	// 契约版本
	Version string
	// 信息摘要
	Revision string
	// 额外描述
	Content string
	// 接口描述信息
	Interfaces string
	// 创建时间
	CreateTime time.Time
	// 更新时间
	ModifyTime time.Time
	// 是否有效
	Valid bool
}
