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

package healthcheck_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"github.com/stretchr/testify/assert"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
	"github.com/polarismesh/polaris/service/healthcheck"
	testsuit "github.com/polarismesh/polaris/test/suit"
)

func Test_serialSetInsDbStatus(t *testing.T) {
	testSuit := &testsuit.DiscoverTestSuit{}
	testSuit.Initialize()

	var (
		mockService   = "mock_service"
		mockNamespace = "mock_namespace"
		mockHost      = "127.0.0.1"
		mockPort      = 8080
	)

	t.Run("prepare_instance", func(t *testing.T) {
		resp := testSuit.DiscoverServer().RegisterInstance(testSuit.DefaultCtx, &service_manage.Instance{
			Service:   utils.NewStringValue(mockService),
			Namespace: utils.NewStringValue(mockNamespace),
			Host:      utils.NewStringValue(mockHost),
			Port:      utils.NewUInt32Value(uint32(mockPort)),
		})

		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), resp.GetCode().GetValue(), resp.GetInfo().GetValue())
		t.Logf("instacne-id: %s", resp.GetInstance().GetId().GetValue())
	})

	testFunc := func(t *testing.T, health bool, predicate func(t *testing.T, saveIns *model.Instance)) {
		mockSvr, err := healthcheck.NewHealthServer(context.TODO(), &healthcheck.Config{
			Open: utils.BoolPtr(true),
			Checkers: []plugin.ConfigEntry{
				{
					Name: "heartbeatMemory",
				},
			},
		}, healthcheck.WithStore(testSuit.Storage))
		if err != nil {
			t.Fatal(err)
		}

		instanceId, err := utils.CalculateInstanceID(mockNamespace, mockService, "", mockHost, uint32(mockPort))
		if err != nil {
			t.Fatal(err)
		}

		respCode := healthcheck.SerialSetInsDbStatus(mockSvr, &service_manage.Instance{
			Id:        utils.NewStringValue(instanceId),
			Service:   utils.NewStringValue(mockService),
			Namespace: utils.NewStringValue(mockNamespace),
			Host:      utils.NewStringValue(mockHost),
			Port:      utils.NewUInt32Value(uint32(mockPort)),
			Healthy:   utils.NewBoolValue(true),
		}, health, time.Now().Unix())

		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), uint32(respCode), fmt.Sprintf("%d", respCode))

		// 获取实例信息
		saveIns, err := testSuit.Storage.GetInstance(instanceId)
		if err != nil {
			t.Fatal(err)
		}

		predicate(t, saveIns)
	}

	t.Run("turn_unhealth", func(t *testing.T) {
		testFunc(t, false, func(t *testing.T, saveIns *model.Instance) {
			metadata := saveIns.Proto.GetMetadata()
			_, exist := metadata[model.MetadataInstanceLastHeartbeatTime]
			assert.True(t, exist, "internal-lastheartbeat must exist : %s", utils.MustJson(metadata))
		})
	})

	t.Run("turn_health", func(t *testing.T) {
		testFunc(t, true, func(t *testing.T, saveIns *model.Instance) {
			metadata := saveIns.Proto.GetMetadata()
			_, exist := metadata[model.MetadataInstanceLastHeartbeatTime]
			assert.False(t, exist, "internal-lastheartbeat must not exist : %s", utils.MustJson(metadata))
		})
	})
}
