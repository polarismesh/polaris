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
	api "github.com/polarismesh/polaris-server/common/api/v1"
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

// AddService save a service
func (ss *serviceStore) AddService(s *model.Service) error {
	if s.ID == "" || s.Name == "" || s.Namespace == "" ||
		s.Owner == "" || s.Token == "" {
		return store.NewStatusError(store.EmptyParamsErr, "add Service missing some params")
	}

	err := ss.handler.SaveValue(ServiceStoreType, s.ID, s)

	return store.Error(err)
}

// DeleteService delete a service
func (ss *serviceStore) DeleteService(id, serviceName, namespaceName string) error {
	if id == "" {
		return store.NewStatusError(store.EmptyParamsErr, "delete Service missing some params")
	}
	err := ss.handler.DeleteValues(ServiceStoreType, []string{id})
	return store.Error(err)
}

// DeleteServiceAlias delete a service alias
func (ss *serviceStore) DeleteServiceAlias(name string, namespace string) error {
	if name == "" || namespace == "" {
		return store.NewStatusError(store.EmptyParamsErr, "delete Service alias missing some params")
	}

	svc, err := GetServiceByNameAndNs(name, namespace, ss.handler)
	if err != nil {
		log.Errorf("get service alias error, %v", err)
		return err
	}

	err = ss.handler.DeleteValues(ServiceStoreType, []string{svc.ID})
	if err != nil  {
		log.Errorf("delete service alias error, %v", err)
	}

	return store.Error(err)
}


// UpdateServiceAlias update service alias
func (ss *serviceStore) UpdateServiceAlias(alias *model.Service, needUpdateOwner bool) error {

	if alias.ID == "" || alias.Name == "" || alias.Namespace == "" ||
		alias.Token == "" || alias.Owner == "" || alias.Revision == "" || alias.Reference == "" {
		return store.NewStatusError(store.EmptyParamsErr, "Update Service Alias missing some params")
	}

	properties := make(map[string]interface{})
	properties["ID"] = alias.ID
	properties["Name"] = alias.Name
	properties["Namespace"] = alias.Namespace
	properties["Token"] = alias.Token
	properties["Owner"] = alias.Owner
	properties["Revision"] = alias.Revision
	properties["Reference"] = alias.Reference

	err := ss.handler.UpdateValue(ServiceStoreType, alias.ID, properties)

	return store.Error(err)
}

// UpdateService update service
func (ss *serviceStore) UpdateService(service *model.Service, needUpdateOwner bool) error {
	if service.ID == "" || service.Name == "" || service.Namespace == "" ||
		service.Token == "" || service.Owner == "" || service.Revision == "" {
		return store.NewStatusError(store.EmptyParamsErr, "Update Service missing some params")
	}

	properties := make(map[string]interface{})
	properties["ID"] = service.ID
	properties["Name"] = service.Name
	properties["Namespace"] = service.Namespace
	properties["Token"] = service.Token
	properties["Owner"] = service.Owner
	properties["Revision"] = service.Revision

	err := ss.handler.UpdateValue(ServiceStoreType, service.ID, properties)

	serr := store.Error(err)
	if store.Code(serr) == store.DuplicateEntryErr {
		serr = store.NewStatusError(store.DataConflictErr, err.Error())
	}
	return serr
}

// UpdateServiceToken update service token
func (ss *serviceStore) UpdateServiceToken(serviceID string, token string, revision string) error {

	properties := make(map[string]interface{})
	properties["Token"] = token
	properties["Revision"] = revision

	err := ss.handler.UpdateValue(ServiceStoreType, serviceID, properties)

	return store.Error(err)
}

// GetSourceServiceToken get source service token
func (ss *serviceStore) GetSourceServiceToken(name string, namespace string) (*model.Service, error) {
	var out model.Service
	s, err := GetServiceByNameAndNs(name, namespace, ss.handler)
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

// GetService get service details based on service name and namespace
func (ss *serviceStore) GetService(name string, namespace string) (*model.Service, error) {
	s, err := GetServiceByNameAndNs(name, namespace, ss.handler)

	if err != nil{
		return nil, err
	}
	if s != nil && !s.Valid {
		return nil, nil
	}
	return s, nil
}

// GetServiceByID get service detail by service id
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



// GetServices query corresponding services and numbers according to relevant conditions
func (ss *serviceStore) GetServices(serviceFilters, serviceMetas map[string]string,
	instanceFilters *store.InstanceArgs, offset, limit uint32) (uint32, []*model.Service, error) {

	totalCount, services, err := ss.getServices(serviceFilters, serviceMetas, instanceFilters,offset, limit)
	if err != nil {
		return 0, nil, err
	}

	return totalCount, services, nil
}

// GetServicesCount get the total number of all services
func (ss *serviceStore) GetServicesCount() (uint32, error) {

	count, err := ss.handler.CountValues(ServiceStoreType)
	if err != nil {
		log.Errorf("load service from kv error %v", err)
		return 0, err
	}

	return uint32(count), nil
}

// GetMoreServices get incremental services
func (ss *serviceStore) GetMoreServices(
	mtime time.Time, firstUpdate, disableBusiness, needMeta bool) (map[string]*model.Service, error) {

	fields := []string{"ModifyTime"}
	if disableBusiness {
		fields = append(fields, "Namespace")
	}
	
	services, err := ss.handler.LoadValuesByFilter(ServiceStoreType, fields, &model.Service{},
	func(m map[string]interface{}) bool{
		if disableBusiness {
			if m["Namespace"].(string) != defaultStore.SystemNamespace {
				return false
			}
		}

		serviceMtime, err := time.Parse("2006-01-02 15:04:05", m["ModifyTime"].(string))
		if err != nil {
			log.Errorf("parse time in get more service error, %v", err)
			return false
		}
		if serviceMtime.Before(mtime) {
			return false
		}
		return true
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

// GetServiceAliases get list of service aliases
func (ss *serviceStore) GetServiceAliases(
	filter map[string]string, offset uint32, limit uint32) (uint32, []*model.ServiceAlias, error) {

	var totalCount uint32

	// find all alias service with filters
	fields := []string{"Reference", "Meta", "Department", "Business"}
	for k, _ := range filter {
		fields = append(fields, k)
	}

	referenceService := make(map[string]bool)
	services, err := ss.handler.LoadValuesByFilter(ServiceStoreType, fields, &model.Service{},
	func(m map[string]interface{}) bool{
		// judge whether it is alias by whether there is a reference
		if m["Reference"].(string) == "" {
			return false
		}

		name, isName := filter["name"]
		keys, isKeys := filter["keys"]
		values, isValues := filter["values"]
		department, isDepartment := filter["department"]
		business, isBusiness := filter["business"]

		// filter by other
		if isName && m["Name"].(string) != name {
			return false
		}

		if isKeys {
			metaValue, ok := m["Meta"].(map[string]string)[keys]
			if !ok {
				return false
			}
			if isValues && values != metaValue {
				return false
			}
		}

		if isDepartment && department != m["Department"].(string) {
			return false
		}
		if isBusiness && business != m["Business"].(string) {
			return false
		}
		referenceService[m["Reference"].(string)] = true
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

	// find source service for every alias
	fields = []string{"ID"}

	refServices, err := ss.handler.LoadValuesByFilter(ServiceStoreType, fields, &model.Service{},
	func(m map[string]interface{}) bool{
		_, ok := referenceService[m["ID"].(string)]
		if !ok {
			return false
		}
		return true
	})

	// sort and limit
	s := getRealServicesList(services, offset, limit)

	var serviceAlias []*model.ServiceAlias
	for _, service := range s {
		alias := model.ServiceAlias{}
		alias.ID = service.ID
		alias.Alias = service.Name
		alias.ServiceID = service.Reference
		alias.Service = refServices[service.Reference].(*model.Service).Name
		alias.ModifyTime = service.ModifyTime
		alias.CreateTime = service.CreateTime
		alias.Comment = service.Comment
		alias.Namespace = service.Namespace
		alias.Owner = service.Owner

		serviceAlias = append(serviceAlias, &alias)
	}

	return totalCount, serviceAlias, nil
}

// GetSystemServices get system services
func (ss *serviceStore) GetSystemServices() ([]*model.Service, error) {

	fields := []string{"Namespace"}

	services, err := ss.handler.LoadValuesByFilter(ServiceStoreType, fields, &model.Service{},
	func(m map[string]interface{}) bool{
		if m["Namespace"].(string) == defaultStore.SystemNamespace {
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

// GetServicesBatch get service id, person in charge and other information in batch
func (ss *serviceStore) GetServicesBatch(services []*model.Service) ([]*model.Service, error) {

	fields := []string{"Name", "Namespace"}

	serviceInfo := make(map[string]string)

	for _, service := range services {
		serviceInfo[service.Name] = service.Namespace
	}

	svcs, err := ss.handler.LoadValuesByFilter(ServiceStoreType, fields, &model.Service{},
		func(m map[string]interface{}) bool{
			name := m["Name"].(string)
			namespace := m["Namespace"].(string)
			for s, n := range serviceInfo {
				if name != s || n != namespace {
					return false
				}
			}
			return true
		})
	if err != nil {
		log.Errorf("load service from kv error, %v", err)
		return nil, err
	}

	return getRealServicesList(svcs, 0, uint32(len(services))), nil
}


func GetServiceByNameAndNs(name string, namespace string,
	handler BoltHandler) (*model.Service, error) {
	var out *model.Service

	fields := []string{"Name", "Namespace"}

	svc, err := handler.LoadValuesByFilter(ServiceStoreType, fields, &model.Service{},
		func(m map[string]interface{}) bool{
			if m["Name"].(string) == name && m["Namespace"].(string) == namespace {
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

	// should only find one service
	for _, v := range svc {
		out = v.(*model.Service)
	}

	return out, err
}

func (ss *serviceStore) getServiceByID(id string) (*model.Service, error) {
	var out *model.Service

	fields := []string{"ID"}

	svc, err := ss.handler.LoadValuesByFilter(ServiceStoreType, fields, &model.Service{},
		func(m map[string]interface{}) bool{
			if m["ID"].(string) == id {
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

	// should only find one service
	for _, v := range svc {
		out = v.(*model.Service)
	}

	return out, err
}


func (ss *serviceStore) getServices(serviceFilters, serviceMetas map[string]string,
	instanceFilters *store.InstanceArgs, offset, limit uint32) (uint32, []*model.Service, error) {

	insFiltersIds := make(map[string]bool)
	// int array to string array
	if instanceFilters != nil && (len(instanceFilters.Ports) > 0 || len(instanceFilters.Hosts) > 0) {

		portArray := make([]string, len(instanceFilters.Ports))
		for i, port := range instanceFilters.Ports {
			portArray[i] = strconv.Itoa(int(port))
		}

		// get the filtered list of serviceIDs from instanceFilters
		filter := []string{"Proto"}

		inss, err := ss.handler.LoadValuesByFilter(InstanceStoreType, filter, &model.Instance{},
			func(m map[string]interface{}) bool{
				ins := m["Proto"].(*api.Instance)
				insHost := ins.GetHost().GetValue()
				insPort := ins.GetPort().GetValue()

				if len(instanceFilters.Hosts) > 0 {
					ifFound := false
					for _, h := range instanceFilters.Hosts {
						if h == insHost {
							ifFound = true
							break
						}
					}
					if !ifFound {
						return false
					}
				}
				if len(instanceFilters.Ports) > 0 {
					ifFound := false
					for _, p := range instanceFilters.Ports {
						if p == insPort {
							ifFound = true
							break
						}
					}
					if !ifFound {
						return false
					}
				}
				return true
		})
		if err != nil {
			log.Errorf("load instance from kv error %v", err)
			return 0, nil, err
		}
		for _, i := range inss {
			insFiltersIds[i.(*model.Instance).ServiceID] = true
		}
	}

	fields := []string{"Name", "Meta", "Department", "Business"}
	if len(insFiltersIds) > 0 {
		fields = append(fields, "ID")
	}

	name, isName := serviceFilters["name"]
	keys, isKeys := serviceFilters["keys"]
	values, isValues := serviceFilters["values"]
	department, isDepartment := serviceFilters["department"]
	business, isBusiness := serviceFilters["business"]

	svcs, err := ss.handler.LoadValuesByFilter(ServiceStoreType, fields, &model.Service{},
	func(m map[string]interface{}) bool{
		// filter by id
		if len(insFiltersIds) > 0 {
			_, ok := insFiltersIds[m["ID"].(string)]
			if !ok {
				return false
			}
		}
		// filter by other
		if isName && m["Name"].(string) != name {
			return false
		}

		if isKeys {
			metaValue, ok := m["Meta"].(map[string]string)[keys]
			if !ok {
				return false
			}
			if isValues && values != metaValue {
				return false
			}
		}

		if isDepartment && department != m["Department"].(string) {
			return false
		}
		if isBusiness && business != m["Business"].(string) {
			return false
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

func getRealServicesList(originServices map[string]interface{}, offset, limit uint32) []*model.Service {
	services := make([]*model.Service, 0)
	beginIndex := offset
	endIndex := beginIndex + limit
	totalCount := uint32(len(originServices))
	// handle special offset, limit
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
		// sort by modifyTime
		if services[i].ModifyTime.After(services[j].ModifyTime) {
			return true
		} else if services[i].ModifyTime.Before(services[j].ModifyTime){
			return false
		}else{
			// compare id if modifyTime is the same
			return services[i].ID < services[j].ID
		}
	})

	return services[beginIndex:endIndex]
}