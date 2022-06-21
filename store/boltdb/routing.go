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
	"sort"
	"strings"
	"time"

	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store"
)

type routingStore struct {
	handler BoltHandler
}

const (
	tblNameRouting         = "routing"
	routingFieldID         = "ID"
	routingFieldInBounds   = "InBounds"
	routingFieldOutBounds  = "OutBounds"
	routingFieldRevision   = "Revision"
	routingFieldModifyTime = "ModifyTime"
	routingFieldValid      = "Valid"
)

// CreateRoutingConfig Add a routing configuration
func (r *routingStore) CreateRoutingConfig(conf *model.RoutingConfig) error {
	if conf.ID == "" || conf.Revision == "" {
		log.Errorf("[Store][boltdb] create routing config missing service id or revision")
		return store.NewStatusError(store.EmptyParamsErr, "missing service id or revision")
	}
	if conf.InBounds == "" || conf.OutBounds == "" {
		log.Errorf("[Store][boltdb] create routing config missing params")
		return store.NewStatusError(store.EmptyParamsErr, "missing some params")
	}

	if err := r.cleanRoutingConfig(conf.ID); err != nil {
		return err
	}

	initRouting(conf)

	err := r.handler.SaveValue(tblNameRouting, conf.ID, conf)
	if err != nil {
		log.Errorf("add routing config to kv error, %v", err)
		return err
	}
	return nil
}

// cleanRoutingConfig 从数据库彻底清理路由配置
func (r *routingStore) cleanRoutingConfig(serviceID string) error {
	err := r.handler.DeleteValues(tblNameRouting, []string{serviceID}, false)
	if err != nil {
		log.Errorf("[Store][boltdb] delete invalid route config error, %v", err)
		return err
	}

	return nil
}

// UpdateRoutingConfig Update a routing configuration
func (r *routingStore) UpdateRoutingConfig(conf *model.RoutingConfig) error {

	if conf.ID == "" || conf.Revision == "" {
		log.Errorf("[Store][boltdb] update routing config missing service id or revision")
		return store.NewStatusError(store.EmptyParamsErr, "missing service id or revision")
	}
	if conf.InBounds == "" || conf.OutBounds == "" {
		log.Errorf("[Store][boltdb] update routing config missing params")
		return store.NewStatusError(store.EmptyParamsErr, "missing some params")
	}

	properties := make(map[string]interface{})
	properties[routingFieldInBounds] = conf.InBounds
	properties[routingFieldOutBounds] = conf.OutBounds
	properties[routingFieldRevision] = conf.Revision
	properties[routingFieldModifyTime] = time.Now()

	err := r.handler.UpdateValue(tblNameRouting, conf.ID, properties)
	if err != nil {
		log.Errorf("[Store][boltdb] update route config to kv error, %v", err)
		return err
	}
	return nil
}

// DeleteRoutingConfig Delete a routing configuration
func (r *routingStore) DeleteRoutingConfig(serviceID string) error {
	if serviceID == "" {
		log.Errorf("[Store][boltdb] delete routing config missing service id")
		return store.NewStatusError(store.EmptyParamsErr, "missing service id")
	}

	properties := make(map[string]interface{})
	properties[routingFieldValid] = false
	properties[routingFieldModifyTime] = time.Now()

	err := r.handler.UpdateValue(tblNameRouting, serviceID, properties)
	if err != nil {
		log.Errorf("[Store][boltdb] delete route config to kv error, %v", err)
		return err
	}

	return nil
}

// GetRoutingConfigsForCache Get incremental routing configuration information through mtime
func (r *routingStore) GetRoutingConfigsForCache(mtime time.Time, firstUpdate bool) ([]*model.RoutingConfig, error) {

	fields := []string{routingFieldModifyTime}

	routes, err := r.handler.LoadValuesByFilter(tblNameRouting, fields, &model.RoutingConfig{},
		func(m map[string]interface{}) bool {
			rMtime, ok := m[routingFieldModifyTime]
			if !ok {
				return false
			}
			routeMtime := rMtime.(time.Time)
			return !routeMtime.Before(mtime)
		})
	if err != nil {
		log.Errorf("[Store][boltdb] load route config from kv error, %v", err)
		return nil, err
	}

	return toRouteConf(routes), nil
}

// GetRoutingConfigWithService Get routing configuration based on service name and namespace
func (r *routingStore) GetRoutingConfigWithService(name string, namespace string) (*model.RoutingConfig, error) {

	dbOp := r.handler
	ss := &serviceStore{
		handler: dbOp,
	}

	// get service first
	service, err := ss.getServiceByNameAndNs(name, namespace)
	if err != nil {
		log.Errorf("[Store][boltdb] get service in route conf error, %v", err)
		return nil, err
	}

	if service == nil {
		return nil, nil
	}

	routeC, err := r.getWithID(service.ID)
	if err != nil {
		return nil, err
	}
	return routeC, nil
}

// GetRoutingConfigWithID Get routing configuration based on service ID
func (r *routingStore) GetRoutingConfigWithID(id string) (*model.RoutingConfig, error) {
	return r.getWithID(id)
}

func (r *routingStore) getWithID(id string) (*model.RoutingConfig, error) {
	fields := []string{routingFieldID}
	routeConf, err := r.handler.LoadValuesByFilter(tblNameRouting, fields, &model.RoutingConfig{},
		func(m map[string]interface{}) bool {
			return id == m[routingFieldID].(string)
		})
	if err != nil {
		log.Errorf("[Store][boltdb] load route config from kv error, %v", err)
		return nil, err
	}

	routeC, ok := routeConf[id].(*model.RoutingConfig)
	if !ok {
		return nil, nil
	}

	if !routeC.Valid {
		return nil, nil
	}

	return routeC, nil
}

// GetRoutingConfigs Get routing configuration list
func (r *routingStore) GetRoutingConfigs(
	filter map[string]string, offset uint32, limit uint32) (uint32, []*model.ExtendRoutingConfig, error) {

	// get all route config
	fields := []string{routingFieldInBounds, routingFieldOutBounds, routingFieldRevision, routingFieldValid}

	svcName, hasSvcName := filter["name"]
	svcNs, hasSvcNs := filter["namespace"]
	inBounds, isInBounds := filter["inBounds"]
	outBounds, isOutBounds := filter["outBounds"]
	revision, isRevision := filter["revision"]

	routeConf, err := r.handler.LoadValuesByFilter(tblNameRouting, fields, &model.RoutingConfig{},
		func(m map[string]interface{}) bool {
			if valid, _ := m[routingFieldValid].(bool); !valid {
				return false
			}

			if isInBounds {
				rIn, ok := m[routingFieldInBounds]
				if !ok {
					return false
				}
				if inBounds != rIn.(string) {
					return false
				}
			}
			if isOutBounds {
				rOut, ok := m[routingFieldOutBounds]
				if !ok {
					return false
				}
				if outBounds != rOut.(string) {
					return false
				}
			}
			if isRevision {
				rRe, ok := m[routingFieldRevision]
				if !ok {
					return false
				}
				if revision != rRe.(string) {
					return false
				}
			}
			return true
		})
	if err != nil {
		log.Errorf("[Store][boltdb] load route config from kv error, %v", err)
		return 0, nil, err
	}

	if len(routeConf) == 0 {
		return 0, nil, nil
	}

	// get service
	svcIds := make(map[string]bool)
	for k := range routeConf {
		svcIds[k] = true
	}

	fields = []string{SvcFieldID, SvcFieldName, SvcFieldNamespace, SvcFieldValid}

	services, err := r.handler.LoadValuesByFilter(tblNameService, fields, &model.Service{},
		func(m map[string]interface{}) bool {

			if valid, _ := m[SvcFieldValid].(bool); !valid {
				return false
			}

			rId, ok := m[SvcFieldID]
			if !ok {
				return false
			}

			savcName := m[SvcFieldName].(string)
			savcNamespace := m[SvcFieldNamespace].(string)

			if hasSvcName && savcName != svcName {
				return false
			}
			if hasSvcNs && savcNamespace != svcNs {
				return false
			}

			id := rId.(string)
			_, ok = svcIds[id]
			return ok
		})

	var out []*model.ExtendRoutingConfig

	for id, r := range routeConf {
		var temp model.ExtendRoutingConfig
		svc, ok := services[id].(*model.Service)
		if ok {
			temp.ServiceName = svc.Name
			temp.NamespaceName = svc.Namespace
		} else {
			log.Warnf("[Store][boltdb] get service in route conf error, service is nil, id: %s", id)
			continue
		}
		temp.Config = r.(*model.RoutingConfig)

		out = append(out, &temp)
	}

	return uint32(len(routeConf)), getRealRouteConfList(out, offset, limit), nil
}

func toRouteConf(m map[string]interface{}) []*model.RoutingConfig {
	var routeConf []*model.RoutingConfig
	for _, r := range m {
		routeConf = append(routeConf, r.(*model.RoutingConfig))
	}

	return routeConf
}

func getRealRouteConfList(routeConf []*model.ExtendRoutingConfig, offset, limit uint32) []*model.ExtendRoutingConfig {

	beginIndex := offset
	endIndex := beginIndex + limit
	totalCount := uint32(len(routeConf))
	// handle invalid offset, limit
	if totalCount == 0 {
		return routeConf
	}
	if beginIndex >= endIndex {
		return routeConf
	}
	if beginIndex >= totalCount {
		return routeConf
	}
	if endIndex > totalCount {
		endIndex = totalCount
	}

	sort.Slice(routeConf, func(i, j int) bool {
		// sort by modify time
		if routeConf[i].Config.ModifyTime.After(routeConf[j].Config.ModifyTime) {
			return true
		} else if routeConf[i].Config.ModifyTime.Before(routeConf[j].Config.ModifyTime) {
			return false
		} else {
			return strings.Compare(routeConf[i].Config.ID, routeConf[j].Config.ID) < 0
		}
	})

	return routeConf[beginIndex:endIndex]
}

func initRouting(r *model.RoutingConfig) {
	currTime := time.Now()
	r.CreateTime = currTime
	r.ModifyTime = currTime
	r.Valid = true
}
