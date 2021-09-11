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
	"sort"
	"strconv"
	"time"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store"
	"github.com/polarismesh/polaris-server/store/defaultStore"
)

type serviceStore struct {
	handler BoltHandler
}

var (
	MultipleSvcFound = errors.New("multiple service find")
)

const (
	tblNameService     = "service"
	SvcFieldID         = "ID"
	SvcFieldName       = "Name"
	SvcFieldNamespace  = "Namespace"
	SvcFieldBusiness   = "Business"
	SvcFieldPorts      = "Ports"
	SvcFieldMeta       = "Meta"
	SvcFieldComment    = "Comment"
	SvcFieldDepartment = "Department"
	SvcFieldModifyTime = "ModifyTime"
	SvcFieldToken      = "Token"
	SvcFieldOwner      = "Owner"
	SvcFieldRevision   = "Revision"
	SvcFieldReference  = "Reference"
)

// AddService save a service
func (ss *serviceStore) AddService(s *model.Service) error {
	if s.ID == "" || s.Name == "" || s.Namespace == "" ||
		s.Owner == "" || s.Token == "" {
		return store.NewStatusError(store.EmptyParamsErr, "add Service missing some params")
	}

	err := ss.handler.SaveValue(tblNameService, s.ID, s)

	return store.Error(err)
}

// DeleteService delete a service
func (ss *serviceStore) DeleteService(id, serviceName, namespaceName string) error {
	if id == "" {
		return store.NewStatusError(store.EmptyParamsErr, "delete Service missing some params")
	}
	err := ss.handler.DeleteValues(tblNameService, []string{id})
	return store.Error(err)
}

// DeleteServiceAlias delete a service alias
func (ss *serviceStore) DeleteServiceAlias(name string, namespace string) error {
	if name == "" || namespace == "" {
		return store.NewStatusError(store.EmptyParamsErr, "delete Service alias missing some params")
	}

	svc, err := ss.getServiceByNameAndNs(name, namespace)
	if err != nil {
		log.Errorf("[Store][boltdb] get service alias error, %v", err)
		return err
	}

	err = ss.handler.DeleteValues(tblNameService, []string{svc.ID})
	if err != nil {
		log.Errorf("[Store][boltdb] delete service alias error, %v", err)
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
	properties[SvcFieldID] = alias.ID
	properties[SvcFieldName] = alias.Name
	properties[SvcFieldNamespace] = alias.Namespace
	properties[SvcFieldToken] = alias.Token
	properties[SvcFieldOwner] = alias.Owner
	properties[SvcFieldRevision] = alias.Revision
	properties[SvcFieldReference] = alias.Reference

	err := ss.handler.UpdateValue(tblNameService, alias.ID, properties)

	return store.Error(err)
}

// UpdateService update service
func (ss *serviceStore) UpdateService(service *model.Service, needUpdateOwner bool) error {
	if service.ID == "" || service.Name == "" || service.Namespace == "" ||
		service.Token == "" || service.Owner == "" || service.Revision == "" {
		return store.NewStatusError(store.EmptyParamsErr, "Update Service missing some params")
	}

	properties := make(map[string]interface{})
	properties[SvcFieldID] = service.ID
	properties[svcFieldName] = service.Name
	properties[svcFieldNamespace] = service.Namespace
	properties[SvcFieldToken] = service.Token
	properties[SvcFieldOwner] = service.Owner
	properties[SvcFieldRevision] = service.Revision

	err := ss.handler.UpdateValue(tblNameService, service.ID, properties)

	serr := store.Error(err)
	if store.Code(serr) == store.DuplicateEntryErr {
		serr = store.NewStatusError(store.DataConflictErr, err.Error())
	}
	return serr
}

// UpdateServiceToken update service token
func (ss *serviceStore) UpdateServiceToken(serviceID string, token string, revision string) error {

	properties := make(map[string]interface{})
	properties[SvcFieldToken] = token
	properties[SvcFieldRevision] = revision

	err := ss.handler.UpdateValue(tblNameService, serviceID, properties)

	return store.Error(err)
}

// GetSourceServiceToken get source service token
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

// GetService get service details based on service name and namespace
func (ss *serviceStore) GetService(name string, namespace string) (*model.Service, error) {
	s, err := ss.getServiceByNameAndNs(name, namespace)

	if err != nil {
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

	totalCount, services, err := ss.getServices(serviceFilters, serviceMetas, instanceFilters, offset, limit)
	if err != nil {
		return 0, nil, err
	}

	return totalCount, services, nil
}

// GetServicesCount get the total number of all services
func (ss *serviceStore) GetServicesCount() (uint32, error) {

	count, err := ss.handler.CountValues(tblNameService)
	if err != nil {
		log.Errorf("[Store][boltdb] load service from kv error %v", err)
		return 0, err
	}

	return uint32(count), nil
}

// GetMoreServices get incremental services
func (ss *serviceStore) GetMoreServices(
	mtime time.Time, firstUpdate, disableBusiness, needMeta bool) (map[string]*model.Service, error) {

	fields := []string{SvcFieldModifyTime}
	if disableBusiness {
		fields = append(fields, SvcFieldNamespace)
	}

	services, err := ss.handler.LoadValuesByFilter(tblNameService, fields, &model.Service{},
		func(m map[string]interface{}) bool {
			if disableBusiness {
				serviceNs, ok := m[SvcFieldNamespace]
				if !ok {
					return false
				}
				if serviceNs.(string) != defaultStore.SystemNamespace {
					return false
				}
			}

			svcMTime, ok := m[SvcFieldModifyTime]
			if !ok {
				return false
			}

			serviceMtime := svcMTime.(time.Time)
			if serviceMtime.Before(mtime) {
				return false
			}
			return true
		})

	if err != nil {
		log.Errorf("[Store][boltdb] load service from kv error, %v", err)
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
	fields := []string{SvcFieldReference, SvcFieldMeta, SvcFieldDepartment, SvcFieldBusiness}
	for k, _ := range filter {
		fields = append(fields, k)
	}

	referenceService := make(map[string]bool)
	services, err := ss.handler.LoadValuesByFilter(tblNameService, fields, &model.Service{},
		func(m map[string]interface{}) bool {
			// judge whether it is alias by whether there is a reference
			reference, err := m[SvcFieldReference]
			if !err {
				return false
			}
			if reference.(string) == "" {
				return false
			}

			name, isName := filter["name"]
			keys, isKeys := filter["keys"]
			values, isValues := filter["values"]
			department, isDepartment := filter["department"]
			business, isBusiness := filter["business"]

			// filter by other
			if isName {
				svcName, ok := m[SvcFieldName]
				if !ok {
					return false
				}
				if svcName.(string) != name {
					return false
				}
			}

			if isKeys {
				svcMeta, ok := m[SvcFieldMeta]
				if !ok {
					return false
				}
				metaValue, ok := svcMeta.(map[string]string)[keys]
				if !ok {
					return false
				}
				if isValues && values != metaValue {
					return false
				}
			}

			if isDepartment {
				svcDepartment, ok := m[SvcFieldDepartment]
				if !ok {
					return false
				}
				if department != svcDepartment.(string) {
					return false
				}
			}
			if isBusiness && business != m[SvcFieldBusiness].(string) {
				svcBusiness, ok := m[SvcFieldBusiness]
				if !ok {
					return false
				}
				if business != svcBusiness.(string) {
					return false
				}
			}
			referenceService[m[SvcFieldReference].(string)] = true
			return true
		})
	if err != nil {
		log.Errorf("[Store][boltdb] load service from kv error, %v", err)
		return 0, nil, err
	}
	if len(services) == 0 {
		return 0, []*model.ServiceAlias{}, nil
	}

	totalCount = uint32(len(services))

	// find source service for every alias
	fields = []string{SvcFieldID}

	refServices, err := ss.handler.LoadValuesByFilter(tblNameService, fields, &model.Service{},
		func(m map[string]interface{}) bool {
			_, ok := referenceService[m[SvcFieldID].(string)]
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

	fields := []string{SvcFieldNamespace}

	services, err := ss.handler.LoadValuesByFilter(tblNameService, fields, &model.Service{},
		func(m map[string]interface{}) bool {
			svcNamespace, ok := m[SvcFieldNamespace]
			if !ok {
				return false
			}
			if svcNamespace.(string) == defaultStore.SystemNamespace {
				return true
			}
			return false
		})
	if err != nil {
		log.Errorf("[Store][boltdb] load service from kv error, %v", err)
		return nil, err
	}

	return getRealServicesList(services, 0, uint32(len(services))), nil
}

// GetServicesBatch get service id and other information in batch
func (ss *serviceStore) GetServicesBatch(services []*model.Service) ([]*model.Service, error) {

	if len(services) == 0 {
		return nil, nil
	}

	fields := []string{SvcFieldName, SvcFieldNamespace}

	serviceInfo := make(map[string]string)

	for _, service := range services {
		serviceInfo[service.Name] = service.Namespace
	}

	svcs, err := ss.handler.LoadValuesByFilter(tblNameService, fields, &model.Service{},
		func(m map[string]interface{}) bool {

			svcName, ok := m[SvcFieldName]
			if !ok {
				return false
			}
			svcNs, ok := m[SvcFieldNamespace]
			if !ok {
				return false
			}

			name := svcName.(string)
			namespace := svcNs.(string)
			ns, ok := serviceInfo[name]
			if !ok {
				return false
			}
			if ns != namespace {
				return false
			}
			return true
		})
	if err != nil {
		log.Errorf("[Store][boltdb] load service from kv error, %v", err)
		return nil, err
	}

	return getRealServicesList(svcs, 0, uint32(len(services))), nil
}

func (ss *serviceStore) getServiceByNameAndNs(name string, namespace string) (*model.Service, error) {
	var out *model.Service

	fields := []string{SvcFieldName, SvcFieldNamespace}

	svc, err := ss.handler.LoadValuesByFilter(tblNameService, fields, &model.Service{},
		func(m map[string]interface{}) bool {

			svcName, ok := m[SvcFieldName]
			if !ok {
				return false
			}
			svcNs, ok := m[SvcFieldNamespace]
			if !ok {
				return false
			}

			if svcName.(string) == name && svcNs.(string) == namespace {
				return true
			}
			return false
		})
	if err != nil {
		return nil, err
	}

	if len(svc) > 1 {
		log.Errorf("[Store][boltdb] multiple services found %v", svc)
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

	fields := []string{SvcFieldID}

	svc, err := ss.handler.LoadValuesByFilter(tblNameService, fields, &model.Service{},
		func(m map[string]interface{}) bool {
			svcId, ok := m[SvcFieldID]
			if !ok {
				return false
			}
			if svcId.(string) == id {
				return true
			}
			return false
		})
	if err != nil {
		return nil, err
	}

	if len(svc) > 1 {
		log.Errorf("[Store][boltdb] multiple services found %v", svc)
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
		filter := []string{insFieldProto}

		inss, err := ss.handler.LoadValuesByFilter(tblNameInstance, filter, &model.Instance{},
			func(m map[string]interface{}) bool {
				insPorto, ok := m[insFieldProto]
				if !ok {
					return false
				}
				ins := insPorto.(*api.Instance)
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
			log.Errorf("[Store][boltdb] load instance from kv error %v", err)
			return 0, nil, err
		}
		for _, i := range inss {
			insFiltersIds[i.(*model.Instance).ServiceID] = true
		}
	}

	fields := []string{SvcFieldName, SvcFieldMeta, SvcFieldDepartment, SvcFieldBusiness}
	if len(insFiltersIds) > 0 {
		fields = append(fields, SvcFieldID)
	}

	name, isName := serviceFilters["name"]
	keys, isKeys := serviceFilters["keys"]
	values, isValues := serviceFilters["values"]
	department, isDepartment := serviceFilters["department"]
	business, isBusiness := serviceFilters["business"]

	svcs, err := ss.handler.LoadValuesByFilter(tblNameService, fields, &model.Service{},
		func(m map[string]interface{}) bool {
			// filter by id
			if len(insFiltersIds) > 0 {
				svcId, ok := m[SvcFieldID]
				if !ok {
					return false
				}
				_, ok = insFiltersIds[svcId.(string)]
				if !ok {
					return false
				}
			}
			// filter by other
			if isName {
				svcName, ok := m[SvcFieldName]
				if !ok {
					return false
				}
				if svcName.(string) != name {
					return false
				}
			}

			if isKeys {
				svcMeta, ok := m[SvcFieldMeta]
				if !ok {
					return false
				}
				metaValue, ok := svcMeta.(map[string]string)[keys]
				if !ok {
					return false
				}
				if isValues && values != metaValue {
					return false
				}
			}

			if isDepartment && department != m[SvcFieldDepartment].(string) {
				svcDepartment, ok := m[SvcFieldDepartment]
				if !ok {
					return false
				}
				if svcDepartment.(string) != department {
					return false
				}
			}
			if isBusiness && business != m[SvcFieldBusiness].(string) {
				svcBusiness, ok := m[SvcFieldBusiness]
				if !ok {
					return false
				}
				if svcBusiness.(string) != business {
					return false
				}
			}

			return true
		})
	if err != nil {
		log.Errorf("[Store][boltdb] load service from kv error %v", err)
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

	sort.Slice(services, func(i, j int) bool {
		// sort by modifyTime
		if services[i].ModifyTime.After(services[j].ModifyTime) {
			return true
		} else if services[i].ModifyTime.Before(services[j].ModifyTime) {
			return false
		} else {
			// compare id if modifyTime is the same
			return services[i].ID < services[j].ID
		}
	})

	return services[beginIndex:endIndex]
}
