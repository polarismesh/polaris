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

package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	apiv1 "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

func mockReportClients(cnt int) []*apiv1.Client {
	ret := make([]*apiv1.Client, 0, 4)

	for i := 0; i < cnt; i++ {
		ret = append(ret, &apiv1.Client{
			Host:     utils.NewStringValue("127.0.0.1"),
			Type:     apiv1.Client_SDK,
			Version:  utils.NewStringValue("v1.0.0"),
			Location: &apiv1.Location{},
			Id:       utils.NewStringValue(NewUUID()),
			Stat: []*apiv1.StatInfo{
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
		if err := discoverSuit.initialize(); err != nil {
			t.Fatal(err)
		}
		defer discoverSuit.Destroy()

		clients := mockReportClients(1)
		defer discoverSuit.cleanReportClient()

		for i := range clients {
			resp := discoverSuit.server.ReportClient(discoverSuit.defaultCtx, clients[i])
			assert.True(t, respSuccess(resp), resp.GetInfo().GetValue())
		}
	})
}

func TestServer_GetReportClient(t *testing.T) {
	t.Run("客户端上报-查询客户端信息", func(t *testing.T) {
		discoverSuit := &DiscoverTestSuit{}
		if err := discoverSuit.initialize(); err != nil {
			t.Fatal(err)
		}
		defer discoverSuit.Destroy()

		clients := mockReportClients(5)
		defer discoverSuit.cleanReportClient()

		wait := sync.WaitGroup{}
		wait.Add(5)
		for i := range clients {
			go func(client *apiv1.Client) {
				defer wait.Done()
				resp := discoverSuit.server.ReportClient(discoverSuit.defaultCtx, client)
				assert.True(t, respSuccess(resp), resp.GetInfo().GetValue())
				t.Logf("create one client success : %s", client.GetId().GetValue())
			}(clients[i])
		}

		wait.Wait()
		time.Sleep(discoverSuit.updateCacheInterval * 5)
		t.Log("finish sleep to wait cache refresh")

		resp := discoverSuit.server.GetReportClientWithCache(context.Background(), map[string]string{})
		assert.Equal(t, apiv1.ExecuteSuccess, resp.Code)
		assert.True(t, len(resp.Response) >= 0 && len(resp.Response) <= 5)
	})
}
