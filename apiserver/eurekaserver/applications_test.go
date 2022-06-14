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

package eurekaserver

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/polarismesh/polaris-server/service"

	"github.com/golang/protobuf/ptypes/wrappers"
	api "github.com/polarismesh/polaris-server/common/api/v1"

	"bou.ke/monkey"
	"github.com/google/uuid"
	"github.com/polarismesh/polaris-server/common/model"
)

const (
	defaultSvcCount     = 10
	productionSvcCount  = 5
	unhealthySvcCount   = 5
	instanceCount       = 20
	namespaceDefault    = "default"
	namespaceProduction = "production"
	hostPrefix          = "127.0.0."
	port                = 10011
)

type svcName struct {
	name      string
	namespace string
}

var (
	mockServices           = map[svcName]*model.Service{}
	mockInstances          = map[string]map[string]*model.Instance{}
	mockUnhealthyServices  = map[svcName]*model.Service{}
	mockUnhealthyInstances = map[string]map[string]*model.Instance{}
)

func buildServices(count int, namespace string, services map[svcName]*model.Service) {
	for i := 0; i < count; i++ {
		name := svcName{
			name:      namespace + "_svc_" + strconv.Itoa(i),
			namespace: namespace,
		}
		services[name] = &model.Service{
			ID:        uuid.NewString(),
			Name:      name.name,
			Namespace: name.namespace,
		}
	}
}

func buildMockInstance(idx int, svc *model.Service, healthy bool, vipAddresses string, svipAddresses string) *model.Instance {
	instance := &model.Instance{
		Proto: &api.Instance{
			Id:                &wrappers.StringValue{Value: uuid.NewString()},
			Service:           &wrappers.StringValue{Value: svc.Name},
			Namespace:         &wrappers.StringValue{Value: svc.Namespace},
			Host:              &wrappers.StringValue{Value: hostPrefix + strconv.Itoa(idx)},
			Port:              &wrappers.UInt32Value{Value: port},
			Protocol:          &wrappers.StringValue{Value: InsecureProtocol},
			Version:           &wrappers.StringValue{Value: "1.0.0"},
			Weight:            &wrappers.UInt32Value{Value: 100},
			EnableHealthCheck: &wrappers.BoolValue{Value: true},
			HealthCheck: &api.HealthCheck{Type: api.HealthCheck_HEARTBEAT, Heartbeat: &api.HeartbeatHealthCheck{
				Ttl: nil,
			}},
			Healthy: &wrappers.BoolValue{Value: healthy},
			Isolate: &wrappers.BoolValue{Value: false},
			Location: &api.Location{
				Region: &wrappers.StringValue{Value: "South China"},
				Zone:   &wrappers.StringValue{Value: "ShangHai"},
				Campus: &wrappers.StringValue{Value: "CampusOne"},
			},
			Metadata: map[string]string{
				MetadataRegisterFrom:        ServerEureka,
				MetadataCountryId:           DefaultCountryId,
				MetadataHostName:            svc.Namespace + "." + svc.Name + "." + strconv.Itoa(port) + "." + strconv.Itoa(idx),
				MetadataInsecurePort:        strconv.Itoa(port),
				MetadataInsecurePortEnabled: "true",
				MetadataSecurePortEnabled:   "false",
			},
		},
		ServiceID:  svc.ID,
		Valid:      true,
		ModifyTime: time.Now(),
	}
	if len(vipAddresses) > 0 {
		instance.Proto.Metadata[MetadataVipAddress] = vipAddresses
	}
	if len(svipAddresses) > 0 {
		instance.Proto.Metadata[MetadataSecureVipAddress] = svipAddresses
	}
	return instance
}

func buildMockSvcInstances() {
	buildServices(defaultSvcCount, namespaceDefault, mockServices)
	buildServices(productionSvcCount, namespaceProduction, mockServices)
	idx := 0
	for _, svc := range mockServices {
		instances := make(map[string]*model.Instance, instanceCount)
		for i := 0; i < instanceCount; i++ {
			idx++
			instance := buildMockInstance(idx, svc, true, "", "")
			instances[instance.ID()] = instance
		}
		mockInstances[svc.ID] = instances
	}
}

func buildMockUnhealthyInstances() {
	buildServices(unhealthySvcCount, namespaceDefault, mockUnhealthyServices)
	idx := 0
	for _, svc := range mockUnhealthyServices {
		var allUnhealthy bool
		if idx%(3*instanceCount) == 0 {
			allUnhealthy = true
		}
		instances := make(map[string]*model.Instance, instanceCount)
		for i := 0; i < instanceCount; i++ {
			idx++
			instance := buildMockInstance(idx, svc, !allUnhealthy, "", "")
			instances[instance.ID()] = instance
		}
		mockUnhealthyInstances[svc.ID] = instances
	}

}

func mockGetCacheServices(namingServer service.DiscoverServer, namespace string) map[string]*model.Service {
	var newServices = make(map[string]*model.Service)
	for _, svc := range mockServices {
		if namespace == svc.Namespace {
			newServices[svc.ID] = svc
		}
	}
	return newServices
}

func mockGetCacheInstances(namingServer service.DiscoverServer, svcId string) ([]*model.Instance, string, error) {
	instances := mockInstances[svcId]
	var retValue = make([]*model.Instance, 0, len(instances))
	if len(instances) == 0 {
		return retValue, uuid.NewString(), nil
	}
	for _, instance := range instances {
		retValue = append(retValue, instance)
	}
	return retValue, uuid.NewString(), nil
}

func doFunctionMock() {
	buildMockSvcInstances()
	monkey.Patch(getCacheServices, mockGetCacheServices)
	monkey.Patch(getCacheInstances, mockGetCacheInstances)
}

// TestApplicationsBuilder_BuildApplications testing method for application builder
func TestApplicationsBuilder_BuildApplications(t *testing.T) {
	doFunctionMock()
	builder := &ApplicationsBuilder{
		namespace:              DefaultNamespace,
		enableSelfPreservation: true,
	}
	appResCache := builder.BuildApplications(nil)
	applications := appResCache.AppsResp.Applications.Application
	assert.Equal(t, defaultSvcCount, len(applications))
	for _, application := range applications {
		serviceName := svcName{
			name:      application.Name,
			namespace: DefaultNamespace,
		}
		svc, ok := mockServices[serviceName]
		assert.True(t, ok)
		instances := application.Instance
		mInstances := mockInstances[svc.ID]
		for _, instance := range instances {
			mInstance, ok := mInstances[instance.InstanceId]
			assert.True(t, ok)
			assert.Equal(t, instance.Port.Port, int(mInstance.Port()))
			assert.Equal(t, instance.IpAddr, mInstance.Host())
			assert.Equal(t, mInstance.Location().GetRegion().GetValue(), instance.Metadata.Meta[KeyRegion])
			assert.Equal(t, mInstance.Location().GetZone().GetValue(), instance.Metadata.Meta[keyZone])
			assert.Equal(t, mInstance.Location().GetCampus().GetValue(), instance.Metadata.Meta[keyCampus])
		}
	}
}

func doUnhealthyFunctionMock() {
	buildMockUnhealthyInstances()
	monkey.Patch(getCacheServices, mockGetUnhealthyServices)
	monkey.Patch(getCacheInstances, mockGetUnhealthyInstances)
}

func mockGetUnhealthyServices(namingServer service.DiscoverServer, namespace string) map[string]*model.Service {
	var newServices = make(map[string]*model.Service)
	for _, svc := range mockUnhealthyServices {
		if namespace == svc.Namespace {
			newServices[svc.ID] = svc
		}
	}
	return newServices
}

func mockGetUnhealthyInstances(namingServer service.DiscoverServer, svcId string) ([]*model.Instance, string, error) {
	instances := mockUnhealthyInstances[svcId]
	var retValue = make([]*model.Instance, 0, len(instances))
	if len(instances) == 0 {
		return retValue, uuid.NewString(), nil
	}
	for _, instance := range instances {
		retValue = append(retValue, instance)
	}
	return retValue, uuid.NewString(), nil
}

// TestApplicationsBuilder_BuildApplicationsSelfPreservation test selfPreservation ability
func TestApplicationsBuilder_BuildApplicationsSelfPreservation(t *testing.T) {
	doUnhealthyFunctionMock()
	builder := &ApplicationsBuilder{
		namespace:              DefaultNamespace,
		enableSelfPreservation: true,
	}
	appResCache := builder.BuildApplications(nil)
	applications := appResCache.AppsResp.Applications.Application
	assert.Equal(t, unhealthySvcCount, len(applications))
	for _, application := range applications {
		serviceName := svcName{
			name:      application.Name,
			namespace: DefaultNamespace,
		}
		svc, ok := mockUnhealthyServices[serviceName]
		assert.True(t, ok)
		instances := application.Instance
		mInstances := mockUnhealthyInstances[svc.ID]
		assert.Equal(t, len(mInstances), len(instances))
	}
}
