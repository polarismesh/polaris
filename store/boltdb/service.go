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
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

type serviceStore struct {
	handler BoltHandler
}

var (
	ErrMultipleSvcFound = errors.New("multiple service find")
)

const (
	tblNameService     string = "service"
	SvcFieldID         string = "ID"
	SvcFieldName       string = "Name"
	SvcFieldNamespace  string = "Namespace"
	SvcFieldBusiness   string = "Business"
	SvcFieldPorts      string = "Ports"
	SvcFieldMeta       string = "Meta"
	SvcFieldComment    string = "Comment"
	SvcFieldDepartment string = "Department"
	SvcFieldModifyTime string = "ModifyTime"
	SvcFieldToken      string = "Token"
	SvcFieldOwner      string = "Owner"
	SvcFieldRevision   string = "Revision"
	SvcFieldReference  string = "Reference"
	SvcFieldValid      string = "Valid"
	SvcFieldCmdbMod1   string = "CmdbMod1"
	SvcFieldCmdbMod2   string = "CmdbMod2"
	SvcFieldCmdbMod3   string = "CmdbMod3"
	SvcFieldExportTo   string = "ExportTo"
)

// AddService save a service
func (ss *serviceStore) AddService(s *model.Service) error {

	// 删除之前同名的服务
	if err := ss.cleanInValidService(s.Name, s.Namespace); err != nil {
		return err
	}

	initService(s)

	if s.ID == "" || s.Name == "" || s.Namespace == "" {
		return store.NewStatusError(store.EmptyParamsErr, "add Service missing some params")
	}

	err := ss.handler.SaveValue(tblNameService, s.ID, toStoreService(s))
	return store.Error(err)
}

// DeleteService delete a service
func (ss *serviceStore) DeleteService(id, serviceName, namespaceName string) error {
	if id == "" {
		return store.NewStatusError(store.EmptyParamsErr, "delete Service missing some params")
	}

	properties := make(map[string]interface{})
	properties[SvcFieldValid] = false
	properties[SvcFieldModifyTime] = time.Now()

	err := ss.handler.UpdateValue(tblNameService, id, properties)
	return store.Error(err)
}

// DeleteServiceAlias delete a service alias
func (ss *serviceStore) DeleteServiceAlias(name string, namespace string) error {
	if name == "" || namespace == "" {
		return store.NewStatusError(store.EmptyParamsErr, "delete Service alias missing some params")
	}

	svc, err := ss.getServiceByNameAndNs(name, namespace)
	if err != nil {
		log.Error("[Store][boltdb] get service alias error", zap.Error(err))
		return err
	}
	if svc == nil {
		return nil
	}

	properties := make(map[string]interface{})
	properties[SvcFieldValid] = false
	properties[SvcFieldModifyTime] = time.Now()

	if err = ss.handler.UpdateValue(tblNameService, svc.ID, properties); err != nil {
		log.Errorf("[Store][boltdb] delete service alias error, %v", err)
	}

	return store.Error(err)
}

// UpdateServiceAlias update service alias
func (ss *serviceStore) UpdateServiceAlias(alias *model.Service, needUpdateOwner bool) error {
	if alias.ID == "" || alias.Name == "" || alias.Namespace == "" ||
		alias.Revision == "" || alias.Reference == "" || (needUpdateOwner && alias.Owner == "") {
		return store.NewStatusError(store.EmptyParamsErr, "update Service Alias missing some params")
	}

	properties := make(map[string]interface{})
	properties[SvcFieldName] = alias.Name
	properties[SvcFieldNamespace] = alias.Namespace
	properties[SvcFieldComment] = alias.Comment
	properties[SvcFieldRevision] = alias.Revision
	properties[SvcFieldToken] = alias.Token
	properties[SvcFieldOwner] = alias.Owner
	properties[SvcFieldReference] = alias.Reference
	properties[SvcFieldExportTo] = utils.MustJson(alias.ExportTo)
	properties[SvcFieldModifyTime] = time.Now()

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

	properties[SvcFieldName] = service.Name
	properties[SvcFieldNamespace] = service.Namespace
	properties[SvcFieldDepartment] = service.Department
	properties[SvcFieldBusiness] = service.Business
	properties[SvcFieldMeta] = service.Meta
	properties[SvcFieldComment] = service.Comment
	properties[SvcFieldRevision] = service.Revision
	properties[SvcFieldToken] = service.Token
	properties[SvcFieldOwner] = service.Owner
	properties[SvcFieldPorts] = service.Ports
	properties[SvcFieldReference] = service.Reference
	properties[SvcFieldCmdbMod1] = service.CmdbMod1
	properties[SvcFieldCmdbMod2] = service.CmdbMod2
	properties[SvcFieldCmdbMod3] = service.CmdbMod3
	properties[SvcFieldExportTo] = utils.MustJson(service.ExportTo)
	properties[SvcFieldModifyTime] = time.Now()

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
	properties[SvcFieldModifyTime] = time.Now()

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
	case s == nil:
		return nil, nil
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

	if s == nil {
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

	fields := []string{SvcFieldModifyTime, SvcFieldValid}
	if disableBusiness {
		fields = append(fields, SvcFieldNamespace)
	}

	services, err := ss.handler.LoadValuesByFilter(tblNameService, fields, &Service{},
		func(m map[string]interface{}) bool {
			if disableBusiness {
				serviceNs, ok := m[SvcFieldNamespace]
				if !ok {
					return false
				}
				if serviceNs.(string) != SystemNamespace {
					return false
				}
			}

			svcMTime, ok := m[SvcFieldModifyTime]
			if !ok {
				return false
			}

			serviceMtime := svcMTime.(time.Time)

			return !serviceMtime.Before(mtime)
		})

	if err != nil {
		log.Errorf("[Store][boltdb] load service from kv error, %v", err)
		return nil, err
	}

	res := make(map[string]*model.Service)
	for k, v := range services {
		res[k] = toModelService(v.(*Service))
	}

	return res, nil
}

// GetServiceAliases get list of service aliases
func (ss *serviceStore) GetServiceAliases(
	filter map[string]string, offset uint32, limit uint32) (uint32, []*model.ServiceAlias, error) {

	// find all alias service with filters
	fields := []string{SvcFieldReference, SvcFieldValid, SvcFieldName, SvcFieldNamespace,
		SvcFieldMeta, SvcFieldDepartment, SvcFieldBusiness}
	for k := range filter {
		fields = append(fields, k)
	}

	referenceService, services, err := ss.getServiceAliases(filter, fields)
	if err != nil {
		return 0, nil, err
	}

	// find source service for every alias
	fields = []string{SvcFieldID, SvcFieldName, SvcFieldNamespace, SvcFieldValid}

	svcName, hasSvcName := filter["service"]
	svcNs, hasSvcNs := filter["namespace"]

	refServices, err := ss.handler.LoadValuesByFilter(tblNameService, fields, &Service{},
		func(m map[string]interface{}) bool {
			if valid, _ := m[SvcFieldValid].(bool); !valid {
				return false
			}

			if hasSvcName && m[SvcFieldName].(string) != svcName {
				return false
			}

			if hasSvcNs && m[SvcFieldNamespace].(string) != svcNs {
				return false
			}

			_, ok := referenceService[m[SvcFieldID].(string)]
			return ok
		})

	var serviceAlias []*model.ServiceAlias
	for _, service := range services {

		if _, ok := refServices[service.Reference]; !ok {
			continue
		}

		alias := model.ServiceAlias{}
		alias.ID = service.ID
		alias.Alias = service.Name
		alias.AliasNamespace = service.Namespace
		alias.ServiceID = service.Reference
		alias.Service = refServices[service.Reference].(*Service).Name
		alias.ModifyTime = service.ModifyTime
		alias.CreateTime = service.CreateTime
		alias.Comment = service.Comment
		alias.Namespace = refServices[service.Reference].(*Service).Namespace
		alias.Owner = service.Owner
		alias.ExportTo = service.ExportTo
		serviceAlias = append(serviceAlias, &alias)
	}

	return uint32(len(serviceAlias)), doPageAliasServices(serviceAlias, offset, limit), nil
}

func (ss *serviceStore) getServiceAliases(
	filter map[string]string, fields []string) (map[string]bool, map[string]*model.Service, error) {
	aliasName, isAliasName := filter["alias"]
	aliasNamespace, isAliasNamespace := filter["alias_namespace"]
	keys, isKeys := filter["keys"]
	values, isValues := filter["values"]
	department, isDepartment := filter["department"]
	business, isBusiness := filter["business"]

	referenceService := make(map[string]bool)
	services, err := ss.handler.LoadValuesByFilter(tblNameService, fields, &Service{},
		func(m map[string]interface{}) bool {
			if valid, _ := m[SvcFieldValid].(bool); !valid {
				return false
			}

			// judge whether it is alias by whether there is a reference
			if reference, _ := m[SvcFieldReference].(string); reference == "" {
				return false
			}

			// filter by other
			if isAliasName {
				svcName, _ := m[SvcFieldName].(string)
				aliasName, isWild := utils.ParseWildName(aliasName)
				if isWild && !strings.Contains(svcName, aliasName) {
					return false
				}
				if svcName != aliasName {
					return false
				}
			}
			if isAliasNamespace {
				svcNamespace, _ := m[SvcFieldNamespace].(string)
				aliasNamespace, isWild := utils.ParseWildName(aliasNamespace)
				if isWild && !strings.Contains(svcNamespace, aliasNamespace) {
					return false
				}
				if svcNamespace != aliasNamespace {
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
		return nil, nil, err
	}
	if len(services) == 0 {
		return referenceService, map[string]*model.Service{}, nil
	}

	ret := make(map[string]*model.Service, len(services))
	for k := range services {
		ret[k] = toModelService(services[k].(*Service))
	}

	return referenceService, ret, nil
}

// GetSystemServices get system services
func (ss *serviceStore) GetSystemServices() ([]*model.Service, error) {

	fields := []string{SvcFieldNamespace}

	services, err := ss.handler.LoadValuesByFilter(tblNameService, fields, &Service{},
		func(m map[string]interface{}) bool {
			svcNamespace, ok := m[SvcFieldNamespace]
			if !ok {
				return false
			}
			if svcNamespace.(string) == SystemNamespace {
				return true
			}
			return false
		})
	if err != nil {
		log.Errorf("[Store][boltdb] load service from kv error, %v", err)
		return nil, err
	}

	ret := make(map[string]*model.Service, len(services))
	for k := range services {
		ret[k] = toModelService(services[k].(*Service))
	}

	return getRealServicesList(ret, 0, uint32(len(services))), nil
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

	svcs, err := ss.handler.LoadValuesByFilter(tblNameService, fields, &Service{},
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

	ret := make(map[string]*model.Service, len(svcs))
	for k := range svcs {
		ret[k] = toModelService(svcs[k].(*Service))
	}

	return getRealServicesList(ret, 0, uint32(len(services))), nil
}

func (ss *serviceStore) getServiceByNameAndNs(name string, namespace string) (*model.Service, error) {

	out, err := ss.getServiceByNameAndNsCommon(name, namespace, true)
	if err != nil {
		return nil, err
	}

	if out == nil || len(out) == 0 {
		return nil, nil
	}

	return out[0], err
}

// getServiceByNameAndNsCommon 根据服务名和命名空间查询服务，支持模糊查询
func (ss *serviceStore) getServiceByNameAndNsCommon(name string, namespace string, forceValid bool) (
	[]*model.Service, error) {

	var out []*model.Service
	fields := []string{svcFieldName, SvcFieldNamespace, SvcFieldValid}

	isNameWild := utils.IsWildName(name)
	isNamespaceWild := utils.IsWildName(namespace)

	svcSlice, err := ss.handler.LoadValuesByFilter(tblNameService, fields, &Service{},
		func(m map[string]interface{}) bool {
			// valid field filter
			if forceValid {
				valid, ok := m[SvcFieldValid]
				if ok && !valid.(bool) {
					return false
				}
			}

			// service name field filter
			svcName, ok := m[SvcFieldName]
			if !ok {
				return false
			}
			if len(name) > 0 {
				if isNameWild {
					if !utils.IsWildMatch(svcName.(string), name) {
						return false
					}
				} else if svcName.(string) != name {
					return false
				}
			}

			// namespace field filter
			svcNs, ok := m[SvcFieldNamespace]
			if !ok {
				return false
			}
			if len(namespace) > 0 {
				if isNamespaceWild {
					if !utils.IsWildMatch(svcNs.(string), namespace) {
						return false
					}
				} else if svcNs.(string) != namespace {
					return false
				}
			}
			return true
		})
	if err != nil {
		return nil, err
	}

	if len(svcSlice) == 0 {
		return nil, nil
	}

	out = make([]*model.Service, 0, len(svcSlice))
	for _, v := range svcSlice {
		svc := v.(*Service)
		if !svc.Valid {
			continue
		}
		out = append(out, toModelService(v.(*Service)))
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, err
}

func (ss *serviceStore) getServiceByNameAndNsIgnoreValid(name string, namespace string) (*model.Service, error) {
	var out *model.Service

	fields := []string{SvcFieldName, SvcFieldNamespace, SvcFieldValid}

	svc, err := ss.handler.LoadValuesByFilter(tblNameService, fields, &Service{},
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
		return nil, ErrMultipleSvcFound
	}

	if len(svc) == 0 {
		return nil, nil
	}

	// should only find one service
	for _, v := range svc {
		out = toModelService(v.(*Service))
	}

	return out, err
}

func (ss *serviceStore) getServiceByID(id string) (*model.Service, error) {

	fields := []string{SvcFieldID, svcFieldValid}

	svc, err := ss.handler.LoadValuesByFilter(tblNameService, fields, &Service{},
		func(m map[string]interface{}) bool {
			valid, ok := m[SvcFieldValid]
			if ok && !valid.(bool) {
				return false
			}

			svcId, ok := m[SvcFieldID]
			if !ok {
				return false
			}
			if svcId.(string) != id {
				return false
			}
			return true
		})
	if err != nil {
		return nil, err
	}

	if len(svc) > 1 {
		log.Errorf("[Store][boltdb] multiple services found %v", svc)
		return nil, ErrMultipleSvcFound
	}
	if len(svc) == 0 {
		return nil, nil
	}
	if _, ok := svc[id]; !ok {
		return nil, nil
	}

	svcRet := toModelService(svc[id].(*Service))
	if svcRet.Valid {
		return svcRet, nil
	}

	return nil, err
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
		filter := []string{insFieldProto, insFieldValid}

		inss, err := ss.handler.LoadValuesByFilter(tblNameInstance, filter, &model.Instance{},
			func(m map[string]interface{}) bool {
				valid, ok := m[SvcFieldValid]
				if ok && !valid.(bool) {
					return false
				}
				insPorto, ok := m[insFieldProto]
				if !ok {
					return false
				}
				ins := insPorto.(*apiservice.Instance)
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

	fields := []string{SvcFieldValid, SvcFieldNamespace, SvcFieldName, SvcFieldMeta, SvcFieldDepartment,
		SvcFieldBusiness, SvcFieldReference}
	if len(insFiltersIds) > 0 {
		fields = append(fields, SvcFieldID)
	}

	isKeys := true
	isValues := true
	var keys string
	var values string

	if len(serviceMetas) == 0 {
		isKeys = false
		isValues = false
	} else {
		for k, v := range serviceMetas {
			keys = k
			values = v
			if values == "" {
				isValues = false
			}
			break
		}
	}

	name, isName := serviceFilters["name"]
	department, isDepartment := serviceFilters["department"]
	business, isBusiness := serviceFilters["business"]
	namespace, isNamespace := serviceFilters["namespace"]

	svcs, err := ss.handler.LoadValuesByFilter(tblNameService, fields, &Service{},
		func(m map[string]interface{}) bool {
			valid, ok := m[SvcFieldValid]
			if ok && !valid.(bool) {
				return false
			}
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

			if isNamespace && namespace != "" {
				svcNs, ok := m[SvcFieldNamespace]
				if !ok {
					return false
				}
				if utils.IsPrefixWildName(namespace) {
					return strings.Contains(svcNs.(string), namespace[0:len(namespace)-1])
				}
				if svcNs.(string) != namespace {
					return false
				}
			}

			// filter by other
			if isName && name != "" {
				svcName, ok := m[SvcFieldName]
				if !ok {
					return false
				}
				if utils.IsPrefixWildName(name) {
					return strings.Contains(svcName.(string), name[0:len(name)-1])
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

			if isDepartment && department != "" {
				svcDepartment, ok := m[SvcFieldDepartment]
				if !ok {
					return false
				}
				if utils.IsPrefixWildName(department) {
					return strings.Contains(svcDepartment.(string), department[0:len(department)-1])
				}
				if svcDepartment.(string) != department {
					return false
				}
			}

			if isBusiness && business != "" {
				svcBusiness, ok := m[SvcFieldBusiness]
				if !ok {
					return false
				}
				if utils.IsPrefixWildName(business) {
					return strings.Contains(svcBusiness.(string), business[0:len(business)-1])
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

	ret := make(map[string]*model.Service, len(svcs))
	for k := range svcs {
		ret[k] = toModelService(svcs[k].(*Service))
	}

	return uint32(totalCount), getRealServicesList(ret, offset, limit), nil
}

func (ss *serviceStore) cleanInValidService(name, namespace string) error {
	old, err := ss.getServiceByNameAndNsIgnoreValid(name, namespace)

	if err != nil {
		return err
	}

	if old == nil {
		return nil
	}

	if err := ss.handler.DeleteValues(tblNameService, []string{old.ID}); err != nil {
		log.Errorf("[Store][boltdb] delete invalid service error, %+v", err)
		return err
	}

	return nil
}

func (ss *serviceStore) GetServiceByNameAndNamespace(name string, namespace string) ([]*model.Service, error) {
	return ss.getServiceByNameAndNsCommon(name, namespace, true)
}

func getRealServicesList(originServices map[string]*model.Service, offset, limit uint32) []*model.Service {
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
		services = append(services, s)
	}

	sort.Slice(services, func(i, j int) bool {
		// sort by modifyTime
		if services[i].ModifyTime.After(services[j].ModifyTime) {
			return true
		} else if services[i].ModifyTime.Before(services[j].ModifyTime) {
			return false
		}
		// compare id if modifyTime is the same
		return services[i].ID < services[j].ID
	})

	return services[beginIndex:endIndex]
}

func doPageAliasServices(originServices []*model.ServiceAlias, offset, limit uint32) []*model.ServiceAlias {
	services := make([]*model.ServiceAlias, 0)
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

	services = append(services, originServices...)
	sort.Slice(services, func(i, j int) bool {
		// sort by modifyTime
		if services[i].ModifyTime.After(services[j].ModifyTime) {
			return true
		} else if services[i].ModifyTime.Before(services[j].ModifyTime) {
			return false
		}
		// compare id if modifyTime is the same
		return services[i].ID < services[j].ID
	})

	return services[beginIndex:endIndex]
}

func initService(s *model.Service) {
	current := time.Now()
	if s != nil {
		s.CreateTime = current
		s.ModifyTime = current
		s.Valid = true
	}
}

func toModelService(data *Service) *model.Service {
	export := make(map[string]struct{})
	_ = json.Unmarshal([]byte(data.ExportTo), &export)
	return &model.Service{
		ID:          data.ID,
		Name:        data.Name,
		Namespace:   data.Namespace,
		Ports:       data.Ports,
		Meta:        data.Meta,
		Comment:     data.Comment,
		Business:    data.Business,
		Department:  data.Department,
		CmdbMod1:    data.CmdbMod1,
		CmdbMod2:    data.CmdbMod2,
		CmdbMod3:    data.CmdbMod3,
		Token:       data.Token,
		Owner:       data.Owner,
		ExportTo:    export,
		Revision:    data.Revision,
		Reference:   data.Reference,
		ReferFilter: data.ReferFilter,
		PlatformID:  data.PlatformID,
		Valid:       data.Valid,
		CreateTime:  data.CreateTime,
		ModifyTime:  data.ModifyTime,
		Mtime:       data.Mtime,
		Ctime:       data.Ctime,
	}
}

func toStoreService(data *model.Service) *Service {
	return &Service{
		ID:          data.ID,
		Name:        data.Name,
		Namespace:   data.Namespace,
		Ports:       data.Ports,
		Meta:        data.Meta,
		Comment:     data.Comment,
		Business:    data.Business,
		Department:  data.Department,
		CmdbMod1:    data.CmdbMod1,
		CmdbMod2:    data.CmdbMod2,
		CmdbMod3:    data.CmdbMod3,
		Token:       data.Token,
		Owner:       data.Owner,
		ExportTo:    utils.MustJson(data.ExportTo),
		Revision:    data.Revision,
		Reference:   data.Reference,
		ReferFilter: data.ReferFilter,
		PlatformID:  data.PlatformID,
		Valid:       data.Valid,
		CreateTime:  data.CreateTime,
		ModifyTime:  data.ModifyTime,
		Mtime:       data.Mtime,
		Ctime:       data.Ctime,
	}
}

type Service struct {
	ID         string
	Name       string
	Namespace  string
	Ports      string
	Meta       map[string]string
	Comment    string
	Business   string
	Department string
	CmdbMod1   string
	CmdbMod2   string
	CmdbMod3   string
	Token      string
	Owner      string
	// ExportTo 服务可见性暴露设置
	ExportTo    string
	Revision    string
	Reference   string
	ReferFilter string
	PlatformID  string
	Valid       bool
	CreateTime  time.Time
	ModifyTime  time.Time
	Mtime       int64
	Ctime       int64
}
