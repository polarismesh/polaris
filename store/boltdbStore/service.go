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
	"github.com/polarismesh/polaris-server/store/defaultStore"
	"sort"
	"strconv"
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
	InstanceStoreType = "instance"
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

	totalCount, services, err := ss.getServices(serviceFilters, serviceMetas, instanceFilters,offset, limit)
	if err != nil {
		return 0, nil, err
	}

	return totalCount, services, nil
}

// 获取所有服务总数
func (ss *serviceStore) GetServicesCount() (uint32, error) {

	count, err := ss.handler.CountValues(ServiceStoreType)
	if err != nil {
		log.Errorf("load service from kv error %v", err)
		return 0, err
	}

	return uint32(count), nil
}

// 获取增量services
func (ss *serviceStore) GetMoreServices(
	mtime time.Time, firstUpdate, disableBusiness, needMeta bool) (map[string]*model.Service, error) {

	// 不考虑 needMeta ，都返回
	fields := []string{"mtime"}
	if disableBusiness {
		fields = append(fields, "namespace")
	}
	
	services, err := ss.handler.LoadValuesByFilter(ServiceStoreType, fields, model.Service{},
	func(m map[string]interface{}) bool{
		if disableBusiness {
			if m["namespace"].(string) != defaultStore.SystemNamespace {
				return false
			}
		}
		if m["mtime"].(string) >= time2String(mtime) {
			return true
		}
		return false
	})

	if err != nil {
		log.Errorf("load service from kv error, %v", err)
		return nil, err
	}

	res := make(map[string]*model.Service)
	for k, v := range services {
		res[k] = v.(*model.Service)
	}

	return res, nil
}

// 获取服务别名列表
func (ss *serviceStore) GetServiceAliases(
	filter map[string]string, offset uint32, limit uint32) (uint32, []*model.ServiceAlias, error) {

	var totalCount uint32

	// 先通过传入的过滤条件，找到所有的 alias 服务
	fields := []string{"reference"}
	for k, _ := range filter {
		fields = append(fields, k)
	}

	services, err := ss.handler.LoadValuesByFilter(ServiceStoreType, fields, model.Service{},
	func(m map[string]interface{}) bool{
		// 通过是否有 reference 判断是不是 alias
		if m["reference"].(string) == "" {
			return false
		}
		// 判断传入的 filter
		for k, v := range filter {
			if v != m[k] {
				return false
			}
		}
		return true
	})
	if err != nil {
		log.Errorf("load service from kv error, %v", err)
		return 0, nil, err
	}
	if len(services) == 0 {
		return 0, []*model.ServiceAlias{}, nil
	}

	totalCount = uint32(len(services))

	// 找到每一个 alias 服务的 reference 服务
	var svcIds []string
	for _, s := range services {
		svcIds = append(svcIds, s.(model.Service).Reference)
	}
	fields = []string{"id"}

	refServiceName := make(map[string]string)

	refServices, err := ss.handler.LoadValuesByFilter(ServiceStoreType, fields, model.Service{},
	func(m map[string]interface{}) bool{
		if containsString(svcIds, m["id"].(string)){
			return true
		}
		return false
	})

	for _, i := range services {
		aliasSvc := i.(model.Service)
		refSvcId := aliasSvc.Reference
		refSvc, ok := refServices[refSvcId]
		if !ok {
			log.Errorf("can't find ref service for %s", aliasSvc.ID)
			continue
		}
		refServiceName[aliasSvc.ID] = refSvc.(model.Service).Name
	}

	// 排序，用 offset/limit 过滤
	s := getRealServicesList(services, offset, limit)

	// 将 service 组装为 ServiceAlias 并返回
	var serviceAlias []*model.ServiceAlias
	for _, service := range s {
		alias := model.ServiceAlias{}
		alias.ID = service.ID
		alias.Alias = service.Name
		alias.ServiceID = service.Reference
		alias.Service = refServiceName[alias.ID]
		alias.ModifyTime = service.ModifyTime
		alias.CreateTime = service.CreateTime
		alias.Comment = service.Comment
		alias.Namespace = service.Namespace
		alias.Owner = service.Owner

		serviceAlias = append(serviceAlias, &alias)
	}

	return totalCount, serviceAlias, nil
}

// 获取系统服务
func (ss *serviceStore) GetSystemServices() ([]*model.Service, error) {

	fields := []string{"namespace"}

	services, err := ss.handler.LoadValuesByFilter(ServiceStoreType, fields, model.Service{},
	func(m map[string]interface{}) bool{
		if m["namespace"].(string) == defaultStore.SystemNamespace {
			return true
		}
		return false
	})
	if err != nil {
		log.Errorf("load service from kv error, %v", err)
		return nil, err
	}

	return getRealServicesList(services, 0, uint32(len(services))), nil
}

// 批量获取服务id、负责人等信息
func (ss *serviceStore) GetServicesBatch(services []*model.Service) ([]*model.Service, error) {

	fields := []string{"name", "namespace"}
	var nameList []string
	var nsList []string

	svcs, err := ss.handler.LoadValuesByFilter(ServiceStoreType, fields, model.Service{},
		func(m map[string]interface{}) bool{
			if !containsString(nameList, m["name"].(string)) {
				return false
			}
			if !containsString(nsList, m["namespace"].(string)) {
				return false
			}
			return true
		})
	if err != nil {
		log.Errorf("load service from kv error, %v", err)
		return nil, err
	}

	return getRealServicesList(svcs, 0, uint32(len(services))), nil
}


func (ss *serviceStore) getServiceByNameAndNs(name string, namespace string) (*model.Service, error) {
	var out *model.Service

	fields := []string{"name", "namespace"}

	svc, err := ss.handler.LoadValuesByFilter(ServiceStoreType, fields, &model.Service{},
		func(m map[string]interface{}) bool{
			if m["name"].(string) == name && m["namespace"].(string) == namespace {
				return true
			}
			return false
		})
	if err != nil {
		return nil, err
	}

	if len(svc) > 1 {
		log.Errorf("multiple services found %v", svc)
		return nil, MultipleSvcFound
	}

	// 应该只能找到一个 service
	for _, v := range svc {
		out = v.(*model.Service)
	}

	return out, err
}

func (ss *serviceStore) getServiceByID(id string) (*model.Service, error) {
	var out model.Service

	fields := []string{"id"}

	svc, err := ss.handler.LoadValuesByFilter(ServiceStoreType, fields, model.Service{},
		func(m map[string]interface{}) bool{
			if m["id"].(string) == id {
				return true
			}
			return false
		})
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


func (ss *serviceStore) getServices(serviceFilters, serviceMetas map[string]string,
	instanceFilters *store.InstanceArgs, offset, limit uint32) (uint32, []*model.Service, error) {

	var insFiltersIds []string
	// int array to string array
	if instanceFilters != nil && (len(instanceFilters.Ports) > 0 || len(instanceFilters.Hosts) > 0) {

		portArray := make([]string, len(instanceFilters.Ports))
		for i, port := range instanceFilters.Ports {
			portArray[i] = strconv.Itoa(int(port))
		}

		// 从 instanceFilters 中得到过滤后的 serviceID 列表
		filter := []string{"host", "port"}

		inss, err := ss.handler.LoadValuesByFilter(InstanceStoreType, filter, model.Instance{},
			func(m map[string]interface{}) bool{
				insHost := m["host"].(string)
				insPort := m["port"].(uint32)

				ifHostFilter := false
				ifPortFilter := false
				if len(instanceFilters.Hosts) <= 0 {
					ifHostFilter = true
				}else {
					for _, h := range instanceFilters.Hosts {
						if h == insHost {
							ifHostFilter = true
							break
						}
					}
				}

				if len(instanceFilters.Ports) <= 0 {
					ifPortFilter = true
				}else {
					for _, p := range instanceFilters.Ports {
						if p == insPort {
							ifPortFilter = true
							break
						}
					}
				}

				if ifHostFilter && ifPortFilter {
					return true
				}
				return false
		})
		if err != nil {
			log.Errorf("load instance from kv error %v", err)
			return 0, nil, err
		}
		for _, i := range inss {
			insFiltersIds = append(insFiltersIds, i.(*model.Instance).ServiceID)
		}
	}

	var fields []string
	if len(insFiltersIds) > 0 {
		fields = append(fields, "id")
	}

	for k, _ := range serviceFilters {
		fields = append(fields, k)
	}

	svcs, err := ss.handler.LoadValuesByFilter(ServiceStoreType, fields, model.Service{},
	func(m map[string]interface{}) bool{
		// 判断 id
		if len(insFiltersIds) > 0 {
			if !containsString(insFiltersIds, m["id"].(string)){
				return false
			}
		}
		// 判断传入的 filter
		for k, v := range serviceFilters {
			if v != m[k] {
				return false
			}
		}
		return true
	})
	if err != nil {
		log.Errorf("load service from kv error %v", err)
		return 0, nil, err
	}
	totalCount := len(svcs)
	return uint32(totalCount), getRealServicesList(svcs, offset, limit), nil
}

// 将下层返回的全量的 service map 转为有序的 list，并根据 offset/limit 返回结果
func getRealServicesList(originServices map[string]interface{}, offset, limit uint32) []*model.Service {
	services := make([]*model.Service, 0)
	beginIndex := offset
	endIndex := beginIndex + limit
	totalCount := uint32(len(originServices))
	// 处理异常的 offset、 limit
	if totalCount == 0 {
		return services
	}
	if beginIndex >= endIndex {
		return services
	}
	if beginIndex >= totalCount {
		return services
	}
	if endIndex > totalCount {
		endIndex = totalCount
	}

	for _, s := range originServices {
		services = append(services, s.(*model.Service))
	}

	sort.Slice(services, func (i, j int) bool{
		// modifyTime 由近到远排序
		if services[i].ModifyTime.After(services[j].ModifyTime) {
			return true
		} else if services[i].ModifyTime.Before(services[j].ModifyTime){
			return false
		}else{
			// modifyTime 相同则比较id
			return services[i].ID < services[j].ID
		}
	})

	return services[beginIndex:endIndex]
}

func containsString(arr []string, key string) bool {
	if len(arr) == 0 {
		return false
	}

	for _, i := range arr {
		if i == key {
			return true
		}
	}

	return false
}

// time.Time转为字符串时间
func time2String(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

