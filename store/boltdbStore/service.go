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
	"database/sql"
	"errors"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store"
	"time"
)

type serviceStore struct {
	handler BoltHandler
}

var (
	MultipleSvcFound = errors.New("multiple service find")
)

const (
	ServiceStoreType = "service"
)

// 保存一个服务
func (ss *serviceStore) AddService(s *model.Service) error {
	if s.ID == "" || s.Name == "" || s.Namespace == "" ||
		s.Owner == "" || s.Token == "" {
		return store.NewStatusError(store.EmptyParamsErr, "add Service missing some params")
	}

	err := ss.handler.SaveValue(ServiceStoreType, s.ID, s)

	return store.Error(err)
}

// 删除服务
func (ss *serviceStore) DeleteService(id, serviceName, namespaceName string) error {
	if id == "" {
		return store.NewStatusError(store.EmptyParamsErr, "delete Service missing some params")
	}
	err := ss.handler.DeleteValues(ServiceStoreType, []string{id})
	return store.Error(err)
}

// 删除服务别名
func (ss *serviceStore) DeleteServiceAlias(name string, namespace string) error {
	if name == "" || namespace == "" {
		return store.NewStatusError(store.EmptyParamsErr, "delete Service alias missing some params")
	}

	svc, err := ss.getServiceByNameAndNs(name, namespace)
	if err != nil {
		return err
	}
	ss.handler.DeleteValues(ServiceStoreType, []string{svc.ID})

	return store.Error(err)
}


// 修改服务别名
func (ss *serviceStore) UpdateServiceAlias(alias *model.Service, needUpdateOwner bool) error {

	err := ss.handler.SaveValue(ServiceStoreType, alias.ID, alias)

	return store.Error(err)
}

// 更新服务
func (ss *serviceStore) UpdateService(service *model.Service, needUpdateOwner bool) error {
	if service.ID == "" || service.Name == "" || service.Namespace == "" ||
		service.Token == "" || service.Owner == "" || service.Revision == "" {
		return store.NewStatusError(store.EmptyParamsErr, "Update Service missing some params")
	}

	err := ss.handler.SaveValue(ServiceStoreType, service.ID, service)

	serr := store.Error(err)
	if store.Code(serr) == store.DuplicateEntryErr {
		serr = store.NewStatusError(store.DataConflictErr, err.Error())
	}
	return serr
}

// 更新服务token
func (ss *serviceStore) UpdateServiceToken(serviceID string, token string, revision string) error {

	m := map[string]interface{}{
		token: token,
		revision: revision,
	}

	err := ss.handler.UpdateValue(ServiceStoreType, serviceID, m)

	return store.Error(err)
}

// 获取源服务的token信息
func (ss *serviceStore) GetSourceServiceToken(name string, namespace string) (*model.Service, error) {
	var out model.Service
	s, err := ss.getServiceByNameAndNs(name, namespace)
	switch {
	case err == sql.ErrNoRows:
		return nil, nil
	case err != nil:
		return nil, err
	default:
		out.ID = s.ID
		out.Token = s.Token
		out.PlatformID = s.PlatformID
		out.Name = name
		out.Namespace = namespace
		return &out, nil
	}

	return &out, nil
}

// 根据服务名和命名空间获取服务的详情
func (ss *serviceStore) GetService(name string, namespace string) (*model.Service, error) {
	s, err := ss.getServiceByNameAndNs(name, namespace)

	if err != nil{
		return nil, err
	}
	if s != nil && !s.Valid {
		return nil, nil
	}
	return s, nil
}

// 根据服务ID查询服务详情
func (ss *serviceStore) GetServiceByID(id string) (*model.Service, error) {
	service, err := ss.getServiceByID(id)
	if err != nil {
		return nil, err
	}
	if service != nil && !service.Valid {
		return nil, nil
	}

	return service, nil
}



// 根据相关条件查询对应服务及数目
func (ss *serviceStore) GetServices(serviceFilters, serviceMetas map[string]string,
	instanceFilters *store.InstanceArgs, offset, limit uint32) (uint32, []*model.Service, error) {
	//TODO
	return 0, nil, nil
}

// 获取所有服务总数
func (ss *serviceStore) GetServicesCount() (uint32, error) {
	//TODO
	return 0, nil
}

// 获取增量services
func (ss *serviceStore) GetMoreServices(
	mtime time.Time, firstUpdate, disableBusiness, needMeta bool) (map[string]*model.Service, error) {
	//TODO
	return nil, nil
}

// 获取服务别名列表
func (ss *serviceStore) GetServiceAliases(
	filter map[string]string, offset uint32, limit uint32) (uint32, []*model.ServiceAlias, error) {
	//TODO
	return 0, nil, nil
}

// 获取系统服务
func (ss *serviceStore) GetSystemServices() ([]*model.Service, error) {
	//TODO
	return nil, nil
}

// 批量获取服务id、负责人等信息
func (ss *serviceStore) GetServicesBatch(services []*model.Service) ([]*model.Service, error) {
	//TODO
	return nil, nil
}


func (ss *serviceStore) getServiceByNameAndNs(name string, namespace string) (*model.Service, error) {
	filter := map[string][]string{
		name: {name},
		namespace: {namespace},
	}
	return ss.getOneServiceByFilter(filter)
}

func (ss *serviceStore) getOneServiceByFilter(filter map[string][]string) (*model.Service, error) {
	var out model.Service

	svc, err := ss.handler.LoadValuesByFilter(ServiceStoreType, filter, model.Service{})
	if err != nil {
		return nil, err
	}

	if len(svc) > 1 {
		log.Errorf("multiple services found %v", svc)
		return nil, MultipleSvcFound
	}

	// 应该只能找到一个 service
	for _, v := range svc {
		out = v.(model.Service)
	}

	return &out, err
}

func (ss *serviceStore) getServiceByID(id string) (*model.Service, error) {
	filter := map[string][]string{
		id: {id},
	}
	return ss.getOneServiceByFilter(filter)
}


func (ss *serviceStore) getServices(serviceFilters, serviceMetas map[string]string,
	instanceFilters *store.InstanceArgs, offset, limit uint32) (uint32, []*model.Service, error) {

	svcs, err := ss.handler.LoadValuesByFilter(ServiceStoreType, getRealFilters(serviceFilters), model.Service{})



}

func getRealFilters(originFilters map[string]string) map[string][]string {
	realFilters := make(map[string][]string)
	for k, v := range originFilters {
		realFilters[k] = []string{v}
	}

	return realFilters
}

