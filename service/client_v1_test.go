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

package service_test

import (
	"context"
	"sync"
	"testing"

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"github.com/stretchr/testify/assert"

	apiv1 "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/service"
)

func mockReportClients(cnt int) []*apiservice.Client {
	ret := make([]*apiservice.Client, 0, 4)

	for i := 0; i < cnt; i++ {
		ret = append(ret, &apiservice.Client{
			Host:     utils.NewStringValue("127.0.0.1"),
			Type:     apiservice.Client_SDK,
			Version:  utils.NewStringValue("v1.0.0"),
			Location: &apimodel.Location{},
			Id:       utils.NewStringValue(utils.NewUUID()),
			Stat: []*apiservice.StatInfo{
				{
					Target:   utils.NewStringValue(model.StatReportPrometheus),
					Port:     utils.NewUInt32Value(uint32(1000 + i)),
					Path:     utils.NewStringValue("/metrics"),
					Protocol: utils.NewStringValue("http"),
				},
			},
		})
	}

	return ret
}

func TestServer_ReportClient(t *testing.T) {
	t.Run("正常客户端上报", func(t *testing.T) {
		discoverSuit := &DiscoverTestSuit{}
		if err := discoverSuit.Initialize(); err != nil {
			t.Fatal(err)
		}
		defer discoverSuit.Destroy()

		clients := mockReportClients(1)
		defer discoverSuit.cleanReportClient()

		for i := range clients {
			resp := discoverSuit.DiscoverServer().ReportClient(discoverSuit.DefaultCtx, clients[i])
			assert.True(t, respSuccess(resp), resp.GetInfo().GetValue())
		}
	})
}

func TestServer_GetReportClient(t *testing.T) {
	t.Run("客户端上报-查询客户端信息", func(t *testing.T) {
		discoverSuit := &DiscoverTestSuit{}
		if err := discoverSuit.Initialize(); err != nil {
			t.Fatal(err)
		}
		defer discoverSuit.Destroy()

		clients := mockReportClients(5)
		defer discoverSuit.cleanReportClient()

		wait := sync.WaitGroup{}
		wait.Add(5)
		for i := range clients {
			go func(client *apiservice.Client) {
				defer wait.Done()
				resp := discoverSuit.DiscoverServer().ReportClient(discoverSuit.DefaultCtx, client)
				assert.True(t, respSuccess(resp), resp.GetInfo().GetValue())
				t.Logf("create one client success : %s", client.GetId().GetValue())
			}(clients[i])
		}

		wait.Wait()
		_ = discoverSuit.DiscoverServer().Cache().TestUpdate()
		t.Log("finish sleep to wait cache refresh")

		resp := discoverSuit.DiscoverServer().GetPrometheusTargets(context.Background(), map[string]string{})
		assert.Equal(t, apiv1.ExecuteSuccess, resp.Code)
		assert.True(t, len(resp.Response) >= 0 && len(resp.Response) <= 5)
	})
}

func TestServer_GetReportClients(t *testing.T) {
	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}

	t.Run("create client", func(t *testing.T) {
		svr := discoverSuit.OriginDiscoverServer()

		mockClientId := utils.NewUUID()
		resp := svr.ReportClient(context.Background(), &service_manage.Client{
			Host:    utils.NewStringValue("127.0.0.1"),
			Type:    service_manage.Client_SDK,
			Version: utils.NewStringValue("1.0.0"),
			Location: &apimodel.Location{
				Region: utils.NewStringValue("region"),
				Zone:   utils.NewStringValue("zone"),
				Campus: utils.NewStringValue("campus"),
			},
			Id: utils.NewStringValue(mockClientId),
			Stat: []*service_manage.StatInfo{
				{
					Target:   utils.NewStringValue("prometheus"),
					Port:     utils.NewUInt32Value(8080),
					Path:     utils.NewStringValue("/metrics"),
					Protocol: utils.NewStringValue("http"),
				},
			},
		})

		assert.Equal(t, resp.GetCode().GetValue(), apimodel.Code_ExecuteSuccess)
		// 强制刷新到 cache
		svr.Cache().TestUpdate()

		originSvr := discoverSuit.OriginDiscoverServer().(*service.Server)
		qresp := originSvr.GetReportClients(discoverSuit.DefaultCtx, map[string]string{})
		assert.Equal(t, resp.GetCode().GetValue(), apimodel.Code_ExecuteSuccess)
		assert.Equal(t, qresp.GetAmount().GetValue(), uint32(1))
		assert.Equal(t, qresp.GetSize().GetValue(), uint32(1))
	})
}
