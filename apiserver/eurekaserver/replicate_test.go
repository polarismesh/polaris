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
	"time"

	"github.com/stretchr/testify/assert"

	api "github.com/polarismesh/polaris/common/api/v1"
	testsuit "github.com/polarismesh/polaris/test/suit"
)

func TestDispatchHeartbeat(t *testing.T) {
	discoverSuit := &testsuit.DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	options := map[string]interface{}{optionRefreshInterval: 5, optionDeltaExpireInterval: 120}
	eurekaSrv, err := createEurekaServerForTest(discoverSuit, options)
	assert.Nil(t, err)
	eurekaSrv.workers = NewApplicationsWorkers(eurekaSrv.refreshInterval, eurekaSrv.deltaExpireInterval,
		eurekaSrv.enableSelfPreservation, eurekaSrv.namingServer, eurekaSrv.healthCheckServer, eurekaSrv.namespace)

	namespace := "default"
	appId := "TESTAPP"
	startPort := 8900
	host := "127.0.1.1"
	total := 30
	instances := batchBuildInstances(appId, host, startPort, &LeaseInfo{
		RenewalIntervalInSecs: 30,
		DurationInSecs:        120,
	}, total)

	var replicateInstances = &ReplicationList{}

	for i, instance := range instances {
		eurekalog.Infof("replicate test: register %d", i)
		replicateInstances.ReplicationList = append(replicateInstances.ReplicationList, &ReplicationInstance{
			AppName:      appId,
			Id:           instance.InstanceId,
			InstanceInfo: instance,
			Action:       actionRegister,
		})
	}
	_, code := eurekaSrv.doBatchReplicate(replicateInstances, "", namespace)
	assert.Equal(t, api.ExecuteSuccess, code)

	time.Sleep(10 * time.Second)
	for i := 0; i < 5; i++ {
		eurekalog.Infof("replicate test: heartbeat %d", i)
		replicateInstances = &ReplicationList{}
		for _, instance := range instances {
			replicateInstances.ReplicationList = append(replicateInstances.ReplicationList, &ReplicationInstance{
				AppName: appId,
				Id:      instance.InstanceId,
				Action:  actionHeartbeat,
			})
		}
		_, code := eurekaSrv.doBatchReplicate(replicateInstances, "", namespace)
		assert.Equal(t, api.ExecuteSuccess, code)
	}
}
