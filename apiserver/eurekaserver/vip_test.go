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
	"testing"

	"github.com/stretchr/testify/assert"

	"bou.ke/monkey"
	"github.com/google/uuid"
	"github.com/polarismesh/polaris-server/service"

	"github.com/polarismesh/polaris-server/common/model"
)

const (
	vipSvcCount       = 20
	vipInstanceCount  = 6
	svipInstanceCount = 10
	vipAddress1       = "vip.address.one"
	vipAddress2       = "vip.address.two"
	vipOneCount       = vipInstanceCount / 2
	svipAddress1      = "svip.address.one"
	svipAddress2      = "svip.address.two"
	svipOneCount      = svipInstanceCount / 2
)

var (
	mockVipServices   = map[svcName]*model.Service{}
	mockVipInstances  = map[string]map[string]*model.Instance{}
	mockSvipServices  = map[svcName]*model.Service{}
	mockSvipInstances = map[string]map[string]*model.Instance{}
)

func buildMockVipInstances() {
	buildServices(vipSvcCount, namespaceDefault, mockVipServices)
	idx := 0
	for _, svc := range mockVipServices {
		instances := make(map[string]*model.Instance, vipInstanceCount)
		for i := 0; i < vipInstanceCount; i++ {
			idx++
			var vipAddress string
			if i < vipOneCount {
				vipAddress = vipAddress1
			} else if i == vipOneCount {
				vipAddress = vipAddress1 + "," + vipAddress2
			} else {
				vipAddress = vipAddress2
			}
			instance := buildMockInstance(idx, svc, true, vipAddress, "")
			instances[instance.ID()] = instance
		}
		mockVipInstances[svc.ID] = instances
	}
}

func buildMockSvipInstances() {
	buildServices(vipSvcCount, namespaceDefault, mockSvipServices)
	idx := 0
	for _, svc := range mockSvipServices {
		instances := make(map[string]*model.Instance, svipInstanceCount)
		for i := 0; i < svipInstanceCount; i++ {
			idx++
			var vipAddress string
			if i < svipOneCount {
				vipAddress = svipAddress1
			} else if i == svipOneCount {
				vipAddress = svipAddress1 + "," + svipAddress2
			} else {
				vipAddress = svipAddress2
			}
			instance := buildMockInstance(idx, svc, true, "", vipAddress)
			instances[instance.ID()] = instance
		}
		mockSvipInstances[svc.ID] = instances
	}
}

func doVipFunctionMock() {
	buildMockVipInstances()
	monkey.Patch(getCacheServices, mockGetVipServices)
	monkey.Patch(getCacheInstances, mockGetVipInstances)
}

func mockGetVipServices(namingServer service.DiscoverServer, namespace string) map[string]*model.Service {
	var newServices = make(map[string]*model.Service)
	for _, svc := range mockVipServices {
		if namespace == svc.Namespace {
			newServices[svc.ID] = svc
		}
	}
	return newServices
}

func mockGetVipInstances(namingServer service.DiscoverServer, svcId string) ([]*model.Instance, string, error) {
	instances := mockVipInstances[svcId]
	var retValue = make([]*model.Instance, 0, len(instances))
	if len(instances) == 0 {
		return retValue, uuid.NewString(), nil
	}
	for _, instance := range instances {
		retValue = append(retValue, instance)
	}
	return retValue, uuid.NewString(), nil
}

//TestBuildApplicationsForVip test method for BuildApplicationsForVip
func TestBuildApplicationsForVip(t *testing.T) {
	doVipFunctionMock()
	builder := &ApplicationsBuilder{
		namespace:              DefaultNamespace,
		enableSelfPreservation: true,
	}
	appResCache := builder.BuildApplications(nil)
	vipAddress1Apps := BuildApplicationsForVip(&VipCacheKey{
		entityType:       entityTypeVip,
		targetVipAddress: vipAddress1,
	}, appResCache)
	applications1 := vipAddress1Apps.AppsResp.Applications.Application
	assert.Equal(t, vipSvcCount, len(applications1))
	for _, application := range applications1 {
		assert.Equal(t, vipOneCount+1, len(application.Instance))
	}
	vipAddress2Apps := BuildApplicationsForVip(&VipCacheKey{
		entityType:       entityTypeVip,
		targetVipAddress: vipAddress2,
	}, appResCache)
	applications2 := vipAddress2Apps.AppsResp.Applications.Application
	assert.Equal(t, vipSvcCount, len(applications2))
	for _, application := range applications2 {
		assert.Equal(t, vipInstanceCount-vipOneCount, len(application.Instance))
	}
}

func doSVipFunctionMock() {
	buildMockSvipInstances()
	monkey.Patch(getCacheServices, mockGetSvipServices)
	monkey.Patch(getCacheInstances, mockGetSvipInstances)
}

func mockGetSvipServices(namingServer service.DiscoverServer, namespace string) map[string]*model.Service {
	var newServices = make(map[string]*model.Service)
	for _, svc := range mockSvipServices {
		if namespace == svc.Namespace {
			newServices[svc.ID] = svc
		}
	}
	return newServices
}

func mockGetSvipInstances(namingServer service.DiscoverServer, svcId string) ([]*model.Instance, string, error) {
	instances := mockSvipInstances[svcId]
	var retValue = make([]*model.Instance, 0, len(instances))
	if len(instances) == 0 {
		return retValue, uuid.NewString(), nil
	}
	for _, instance := range instances {
		retValue = append(retValue, instance)
	}
	return retValue, uuid.NewString(), nil
}

//TestBuildApplicationsForSVip test method for BuildApplicationsForVip
func TestBuildApplicationsForSvip(t *testing.T) {
	doSVipFunctionMock()
	builder := &ApplicationsBuilder{
		namespace:              DefaultNamespace,
		enableSelfPreservation: true,
	}
	appResCache := builder.BuildApplications(nil)
	vipAddress1Apps := BuildApplicationsForVip(&VipCacheKey{
		entityType:       entityTypeSVip,
		targetVipAddress: svipAddress1,
	}, appResCache)
	applications1 := vipAddress1Apps.AppsResp.Applications.Application
	assert.Equal(t, vipSvcCount, len(applications1))
	for _, application := range applications1 {
		assert.Equal(t, svipOneCount+1, len(application.Instance))
	}
	vipAddress2Apps := BuildApplicationsForVip(&VipCacheKey{
		entityType:       entityTypeSVip,
		targetVipAddress: svipAddress2,
	}, appResCache)
	applications2 := vipAddress2Apps.AppsResp.Applications.Application
	assert.Equal(t, vipSvcCount, len(applications2))
	for _, application := range applications2 {
		assert.Equal(t, svipInstanceCount-svipOneCount, len(application.Instance))
	}
}
