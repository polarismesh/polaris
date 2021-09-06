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
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store"
	"time"
)

type serviceStore struct {
	handler BoltHandler
}

// 保存一个服务
func (s *serviceStore) AddService(service *model.Service) error {
	//TODO
	return nil
}

// 删除服务
func (s *serviceStore) DeleteService(id, serviceName, namespaceName string) error {
	//TODO
	return nil
}

// 删除服务别名
func (s *serviceStore) DeleteServiceAlias(name string, namespace string) error {
	//TODO
	return nil
}

// 修改服务别名
func (s *serviceStore) UpdateServiceAlias(alias *model.Service, needUpdateOwner bool) error {
	//TODO
	return nil
}

// 更新服务
func (s *serviceStore) UpdateService(service *model.Service, needUpdateOwner bool) error {
	//TODO
	return nil
}

// 更新服务token
func (s *serviceStore) UpdateServiceToken(serviceID string, token string, revision string) error {
	//TODO
	return nil
}

// 获取源服务的token信息
func (s *serviceStore) GetSourceServiceToken(name string, namespace string) (*model.Service, error) {
	//TODO
	return nil, nil
}

// 根据服务名和命名空间获取服务的详情
func (s *serviceStore) GetService(name string, namespace string) (*model.Service, error) {
	//TODO
	return nil, nil
}

// 根据服务ID查询服务详情
func (s *serviceStore) GetServiceByID(id string) (*model.Service, error) {
	//TODO
	return nil, nil
}

// 根据相关条件查询对应服务及数目
func (s *serviceStore) GetServices(serviceFilters, serviceMetas map[string]string,
	instanceFilters *store.InstanceArgs, offset, limit uint32) (uint32, []*model.Service, error) {
	//TODO
	return 0, nil, nil
}

// 获取所有服务总数
func (s *serviceStore) GetServicesCount() (uint32, error) {
	//TODO
	return 0, nil
}

// 获取增量services
func (s *serviceStore) GetMoreServices(
	mtime time.Time, firstUpdate, disableBusiness, needMeta bool) (map[string]*model.Service, error) {
	//TODO
	return nil, nil
}

// 获取服务别名列表
func (s *serviceStore) GetServiceAliases(
	filter map[string]string, offset uint32, limit uint32) (uint32, []*model.ServiceAlias, error) {
	//TODO
	return 0, nil, nil
}

// 获取系统服务
func (s *serviceStore) GetSystemServices() ([]*model.Service, error) {
	//TODO
	return nil, nil
}

// 批量获取服务id、负责人等信息
func (s *serviceStore) GetServicesBatch(services []*model.Service) ([]*model.Service, error) {
	//TODO
	return nil, nil
}