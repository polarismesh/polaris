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
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store"
	"sort"
	"strconv"
	"strings"
	"time"
)

type routingStore struct {
	handler BoltHandler
}

const routingStoreType = "routing"

// CreateRoutingConfig Add a routing configuration
func (r *routingStore) CreateRoutingConfig(conf *model.RoutingConfig) error {
	if conf.ID == "" || conf.Revision == "" {
		log.Errorf("[Store][database] create routing config missing service id or revision")
		return store.NewStatusError(store.EmptyParamsErr, "missing service id or revision")
	}
	if conf.InBounds == "" || conf.OutBounds == "" {
		log.Errorf("[Store][database] create routing config missing params")
		return store.NewStatusError(store.EmptyParamsErr, "missing some params")
	}

	err := r.handler.SaveValue(routingStoreType, conf.ID, conf)
	if err != nil {
		log.Errorf("add routing config to kv error, %v", err)
		return err
	}
	return nil
}

// UpdateRoutingConfig Update a routing configuration
func (r *routingStore) UpdateRoutingConfig(conf *model.RoutingConfig) error {

	if conf.ID == "" || conf.Revision == "" {
		log.Errorf("[Store][database] update routing config missing service id or revision")
		return store.NewStatusError(store.EmptyParamsErr, "missing service id or revision")
	}
	if conf.InBounds == "" || conf.OutBounds == "" {
		log.Errorf("[Store][database] update routing config missing params")
		return store.NewStatusError(store.EmptyParamsErr, "missing some params")
	}

	properties := make(map[string]interface{})
	properties["InBounds"] = conf.InBounds
	properties["OutBounds"] = conf.OutBounds
	properties["Revision"] = conf.Revision

	err := r.handler.UpdateValue(routingStoreType, conf.ID, properties)
	if err != nil {
		log.Errorf("update route config to kv error, %v", err)
		return err
	}
	return nil
}

// DeleteRoutingConfig Delete a routing configuration
func (r *routingStore) DeleteRoutingConfig(serviceID string) error {
	if serviceID == "" {
		log.Errorf("[Store][database] delete routing config missing service id")
		return store.NewStatusError(store.EmptyParamsErr, "missing service id")
	}

	err := r.handler.DeleteValues(routingStoreType, []string{serviceID})
	if err != nil {
		log.Errorf("delete route config to kv error, %v", err)
		return err
	}

	return nil
}

// GetRoutingConfigsForCache Get incremental routing configuration information through mtime
func (r *routingStore) GetRoutingConfigsForCache(mtime time.Time, firstUpdate bool) ([]*model.RoutingConfig, error) {

	fields := []string{"ModifyTime"}

	routes, err := r.handler.LoadValuesByFilter(routingStoreType, fields, &model.RoutingConfig{},
		func(m map[string]interface{}) bool{
			routeMtime, err := time.Parse("2006-01-02 15:04:05", m["ModifyTime"].(string))
			if err != nil {
				log.Errorf("parse route conf time error, %v", err)
				return false
			}
			if routeMtime.Before(mtime) {
				return false
			}

			return true
		})
	if err != nil {
		log.Errorf("get route config from kv error, %v", err)
		return nil, err
	}

	return toRouteConf(routes), nil
}

// GetRoutingConfigWithService Get routing configuration based on service name and namespace
func (r *routingStore) GetRoutingConfigWithService(name string, namespace string) (*model.RoutingConfig, error) {

	// get service first
	service, err := GetServiceByNameAndNs(name, name, r.handler)
	if err != nil {
		log.Errorf("get service in route conf error, %v", err)
		return nil, err
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
	fields := []string{"ID"}
	routeConf, err := r.handler.LoadValuesByFilter(routingStoreType, fields, &model.RoutingConfig{},
		func(m map[string]interface{}) bool{
			if id != m["ID"].(string) {
				return false
			}

			return true
		})
	if err != nil {
		log.Errorf("get route config from kv error, %v", err)
		return nil, err
	}

	routeC, ok := routeConf[id].(*model.RoutingConfig)
	if !ok {
		return nil, nil
	}
	return routeC, nil
}

// GetRoutingConfigs Get routing configuration list
func (r *routingStore) GetRoutingConfigs(
	filter map[string]string, offset uint32, limit uint32) (uint32, []*model.ExtendRoutingConfig, error) {

	// get all route config
	fields := []string{"InBounds", "OutBounds", "Revision", "Valid"}

	inBounds, isInBounds := filter["inBounds"]
	outBounds, isOutBounds := filter["outBounds"]
	revision, isRevision := filter["revision"]
	valid, isValid := filter["valid"]

	routeConf, err := r.handler.LoadValuesByFilter(routingStoreType, fields, &model.RoutingConfig{},
		func(m map[string]interface{}) bool{
			if isInBounds && inBounds != m["InBounds"].(string) {
				return false
			}
			if isOutBounds && outBounds != m["outBounds"].(string) {
				return false
			}
			if isRevision && revision != m["revision"].(string) {
				return false
			}
			if isValid && valid != strconv.FormatBool(m["valid"].(bool)) {
				return false
			}
			return true
		})
	if err != nil {
		log.Errorf("get route config from kv error, %v", err)
		return 0, nil, err
	}

	if len(routeConf) == 0 {
		return 0, nil, nil
	}

	// get service
	svcIds := make(map[string]bool)
	for k, _ := range routeConf {
		svcIds[k] = true
	}

	fields = []string{"ID"}

	services, err := r.handler.LoadValuesByFilter(ServiceStoreType, fields, &model.Service{},
		func(m map[string]interface{}) bool{
			id := m["ID"].(string)
			_, ok := svcIds[id]
			if !ok {
				return false
			}

			return true
		})

	var out []*model.ExtendRoutingConfig

	for id, r := range routeConf {
		var temp model.ExtendRoutingConfig
		svc, ok := services[id].(*model.Service)
		if ok {
			temp.ServiceName = svc.Name
			temp.NamespaceName = svc.Namespace
		} else {
			log.Warnf("get service in route conf error, service is nil, id: %s", id)
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
	// 处理异常的 offset、 limit
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

	sort.Slice(routeConf, func (i, j int) bool{
		// sort by modify time
		if routeConf[i].Config.ModifyTime.After(routeConf[j].Config.ModifyTime) {
			return true
		} else if routeConf[i].Config.ModifyTime.Before(routeConf[j].Config.ModifyTime){
			return false
		}else{
			return strings.Compare(routeConf[i].Config.ID, routeConf[j].Config.ID) < 0
		}
	})

	return routeConf[beginIndex:endIndex]
}