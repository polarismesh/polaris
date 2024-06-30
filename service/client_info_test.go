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

	"github.com/golang/mock/gomock"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/polarismesh/polaris/cache"
	api "github.com/polarismesh/polaris/common/api/v1"
	apiv1 "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/service"
	"github.com/polarismesh/polaris/service/batch"
	"github.com/polarismesh/polaris/store"
	"github.com/polarismesh/polaris/store/mock"
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
		t.Cleanup(func() {
			discoverSuit.cleanReportClient()
			discoverSuit.Destroy()
		})

		clients := mockReportClients(1)

		for i := range clients {
			resp := discoverSuit.DiscoverServer().ReportClient(discoverSuit.DefaultCtx, clients[i])
			assert.True(t, respSuccess(resp), resp.GetInfo().GetValue())
		}
	})

	t.Run("abnormal_scene", func(t *testing.T) {

		discoverSuit := &DiscoverTestSuit{}
		if err := discoverSuit.Initialize(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			discoverSuit.cleanReportClient()
			discoverSuit.Destroy()
		})

		client := mockReportClients(1)[0]

		svr := discoverSuit.OriginDiscoverServer().(*service.Server)

		t.Run("01_store_err", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			oldStore := svr.Store()
			oldBc := svr.GetBatchController()

			ctx, cancel := context.WithCancel(context.Background())
			defer func() {
				cancel()
				ctrl.Finish()
				svr.MockBatchController(oldBc)
				svr.TestSetStore(oldStore)
			}()

			mockStore := mock.NewMockStore(ctrl)
			mockBc, err := batch.NewBatchCtrlWithConfig(mockStore, discoverSuit.CacheMgr(), &batch.Config{
				ClientRegister: &batch.CtrlConfig{
					Open:          true,
					QueueSize:     1,
					WaitTime:      "32ms",
					Concurrency:   1,
					MaxBatchCount: 4,
				},
			})
			assert.NoError(t, err)
			mockBc.Start(ctx)

			svr.TestSetStore(mockStore)
			svr.MockBatchController(mockBc)

			mockStore.EXPECT().BatchAddClients(gomock.Any()).Return(store.NewStatusError(store.Unknown, "mock error")).AnyTimes()

			rsp := discoverSuit.DiscoverServer().ReportClient(discoverSuit.DefaultCtx, client)
			assert.False(t, api.IsSuccess(rsp), rsp.GetInfo().GetValue())
		})

		t.Run("02_exist_resource", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			oldStore := svr.Store()
			defer func() {
				ctrl.Finish()
				svr.TestSetStore(oldStore)
			}()

			mockStore := mock.NewMockStore(ctrl)
			svr.TestSetStore(mockStore)

			mockStore.EXPECT().BatchAddClients(gomock.Any()).Return(store.NewStatusError(store.DuplicateEntryErr, "mock error")).AnyTimes()

			rsp := discoverSuit.DiscoverServer().ReportClient(discoverSuit.DefaultCtx, client)
			assert.True(t, api.IsSuccess(rsp), rsp.GetInfo().GetValue())
		})
	})
}

func TestServer_GetReportClient(t *testing.T) {
	t.Run("客户端上报-查询客户端信息", func(t *testing.T) {
		discoverSuit := &DiscoverTestSuit{}
		if err := discoverSuit.Initialize(); err != nil {
			t.Fatal(err)
		}
		// 主动触发清理之前的 ReportClient 数据
		discoverSuit.cleanReportClient()
		// 强制触发缓存更新
		_ = discoverSuit.DiscoverServer().Cache().(*cache.CacheManager).TestUpdate()
		t.Log("finish sleep to wait cache refresh")

		t.Cleanup(func() {
			discoverSuit.cleanReportClient()
			discoverSuit.Destroy()
		})

		clients := mockReportClients(5)

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
		_ = discoverSuit.DiscoverServer().Cache().(*cache.CacheManager).TestUpdate()
		t.Log("finish sleep to wait cache refresh")

		resp := discoverSuit.DiscoverServer().GetPrometheusTargets(context.Background(), map[string]string{})
		t.Logf("get report clients result: %#v", resp)
		assert.Equal(t, apiv1.ExecuteSuccess, resp.Code)
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

		assert.Equal(t, resp.GetCode().GetValue(), uint32(apimodel.Code_ExecuteSuccess))
		// 强制刷新到 cache
		svr.Cache().(*cache.CacheManager).TestUpdate()

		originSvr := discoverSuit.OriginDiscoverServer().(*service.Server)
		qresp := originSvr.GetReportClients(discoverSuit.DefaultCtx, map[string]string{})
		assert.Equal(t, resp.GetCode().GetValue(), uint32(apimodel.Code_ExecuteSuccess))
		assert.Equal(t, qresp.GetAmount().GetValue(), uint32(1))
		assert.Equal(t, qresp.GetSize().GetValue(), uint32(1))
	})

	t.Run("invalid_search", func(t *testing.T) {
		originSvr := discoverSuit.OriginDiscoverServer().(*service.Server)
		resp := originSvr.GetReportClients(discoverSuit.DefaultCtx, map[string]string{
			"offset": "abc",
		})

		assert.False(t, api.IsSuccess(resp), resp.GetInfo().GetValue())
		assert.Equal(t, uint32(apimodel.Code_InvalidParameter), resp.GetCode().GetValue())

		resp = originSvr.GetReportClients(discoverSuit.DefaultCtx, map[string]string{
			"version_123": "abc",
		})

		assert.False(t, api.IsSuccess(resp), resp.GetInfo().GetValue())
		assert.Equal(t, uint32(apimodel.Code_InvalidParameter), resp.GetCode().GetValue())

	})
}

func Test_clientEquals(t *testing.T) {
	type args struct {
		client1 *apiservice.Client
		client2 *apiservice.Client
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "full_equal",
			args: args{
				client1: &apiservice.Client{
					Id:      wrapperspb.String("1"),
					Host:    wrapperspb.String("1.1.1.1"),
					Version: wrapperspb.String("Java-1.0.0"),
					Type:    apiservice.Client_SDK,
					Location: &apimodel.Location{
						Region: wrapperspb.String("region"),
						Zone:   wrapperspb.String("zone"),
						Campus: wrapperspb.String("campus"),
					},
					Stat: []*apiservice.StatInfo{
						{
							Target:   wrapperspb.String("prometheus"),
							Port:     wrapperspb.UInt32(28080),
							Path:     wrapperspb.String("/metrics"),
							Protocol: wrapperspb.String("http"),
						},
					},
				},
				client2: &apiservice.Client{
					Id:      wrapperspb.String("1"),
					Host:    wrapperspb.String("1.1.1.1"),
					Version: wrapperspb.String("Java-1.0.0"),
					Type:    apiservice.Client_SDK,
					Location: &apimodel.Location{
						Region: wrapperspb.String("region"),
						Zone:   wrapperspb.String("zone"),
						Campus: wrapperspb.String("campus"),
					},
					Stat: []*apiservice.StatInfo{
						{
							Target:   wrapperspb.String("prometheus"),
							Port:     wrapperspb.UInt32(28080),
							Path:     wrapperspb.String("/metrics"),
							Protocol: wrapperspb.String("http"),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "id_not_equal",
			args: args{
				client1: &apiservice.Client{
					Id:      wrapperspb.String("2"),
					Host:    wrapperspb.String("1.1.1.1"),
					Version: wrapperspb.String("Java-1.0.0"),
					Type:    apiservice.Client_SDK,
					Location: &apimodel.Location{
						Region: wrapperspb.String("region"),
						Zone:   wrapperspb.String("zone"),
						Campus: wrapperspb.String("campus"),
					},
					Stat: []*apiservice.StatInfo{
						{
							Target:   wrapperspb.String("prometheus"),
							Port:     wrapperspb.UInt32(28080),
							Path:     wrapperspb.String("/metrics"),
							Protocol: wrapperspb.String("http"),
						},
					},
				},
				client2: &apiservice.Client{
					Id:      wrapperspb.String("1"),
					Host:    wrapperspb.String("1.1.1.1"),
					Version: wrapperspb.String("Java-1.0.0"),
					Type:    apiservice.Client_SDK,
					Location: &apimodel.Location{
						Region: wrapperspb.String("region"),
						Zone:   wrapperspb.String("zone"),
						Campus: wrapperspb.String("campus"),
					},
					Stat: []*apiservice.StatInfo{
						{
							Target:   wrapperspb.String("prometheus"),
							Port:     wrapperspb.UInt32(28080),
							Path:     wrapperspb.String("/metrics"),
							Protocol: wrapperspb.String("http"),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "host_not_equal",
			args: args{
				client1: &apiservice.Client{
					Id:      wrapperspb.String("1"),
					Host:    wrapperspb.String("2.1.1.1"),
					Version: wrapperspb.String("Java-1.0.0"),
					Type:    apiservice.Client_SDK,
					Location: &apimodel.Location{
						Region: wrapperspb.String("region"),
						Zone:   wrapperspb.String("zone"),
						Campus: wrapperspb.String("campus"),
					},
					Stat: []*apiservice.StatInfo{
						{
							Target:   wrapperspb.String("prometheus"),
							Port:     wrapperspb.UInt32(28080),
							Path:     wrapperspb.String("/metrics"),
							Protocol: wrapperspb.String("http"),
						},
					},
				},
				client2: &apiservice.Client{
					Id:      wrapperspb.String("1"),
					Host:    wrapperspb.String("1.1.1.1"),
					Version: wrapperspb.String("Java-1.0.0"),
					Type:    apiservice.Client_SDK,
					Location: &apimodel.Location{
						Region: wrapperspb.String("region"),
						Zone:   wrapperspb.String("zone"),
						Campus: wrapperspb.String("campus"),
					},
					Stat: []*apiservice.StatInfo{
						{
							Target:   wrapperspb.String("prometheus"),
							Port:     wrapperspb.UInt32(28080),
							Path:     wrapperspb.String("/metrics"),
							Protocol: wrapperspb.String("http"),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "version_not_equal",
			args: args{
				client1: &apiservice.Client{
					Id:      wrapperspb.String("1"),
					Host:    wrapperspb.String("1.1.1.1"),
					Version: wrapperspb.String("Java-1.1.0"),
					Type:    apiservice.Client_SDK,
					Location: &apimodel.Location{
						Region: wrapperspb.String("region"),
						Zone:   wrapperspb.String("zone"),
						Campus: wrapperspb.String("campus"),
					},
					Stat: []*apiservice.StatInfo{
						{
							Target:   wrapperspb.String("prometheus"),
							Port:     wrapperspb.UInt32(28080),
							Path:     wrapperspb.String("/metrics"),
							Protocol: wrapperspb.String("http"),
						},
					},
				},
				client2: &apiservice.Client{
					Id:      wrapperspb.String("1"),
					Host:    wrapperspb.String("1.1.1.1"),
					Version: wrapperspb.String("Java-1.0.0"),
					Type:    apiservice.Client_SDK,
					Location: &apimodel.Location{
						Region: wrapperspb.String("region"),
						Zone:   wrapperspb.String("zone"),
						Campus: wrapperspb.String("campus"),
					},
					Stat: []*apiservice.StatInfo{
						{
							Target:   wrapperspb.String("prometheus"),
							Port:     wrapperspb.UInt32(28080),
							Path:     wrapperspb.String("/metrics"),
							Protocol: wrapperspb.String("http"),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "region_not_equal",
			args: args{
				client1: &apiservice.Client{
					Id:      wrapperspb.String("1"),
					Host:    wrapperspb.String("1.1.1.1"),
					Version: wrapperspb.String("Java-1.1.0"),
					Type:    apiservice.Client_SDK,
					Location: &apimodel.Location{
						Region: wrapperspb.String("region-1"),
						Zone:   wrapperspb.String("zone"),
						Campus: wrapperspb.String("campus"),
					},
					Stat: []*apiservice.StatInfo{
						{
							Target:   wrapperspb.String("prometheus"),
							Port:     wrapperspb.UInt32(28080),
							Path:     wrapperspb.String("/metrics"),
							Protocol: wrapperspb.String("http"),
						},
					},
				},
				client2: &apiservice.Client{
					Id:      wrapperspb.String("1"),
					Host:    wrapperspb.String("1.1.1.1"),
					Version: wrapperspb.String("Java-1.0.0"),
					Type:    apiservice.Client_SDK,
					Location: &apimodel.Location{
						Region: wrapperspb.String("region"),
						Zone:   wrapperspb.String("zone"),
						Campus: wrapperspb.String("campus"),
					},
					Stat: []*apiservice.StatInfo{
						{
							Target:   wrapperspb.String("prometheus"),
							Port:     wrapperspb.UInt32(28080),
							Path:     wrapperspb.String("/metrics"),
							Protocol: wrapperspb.String("http"),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "zone_not_equal",
			args: args{
				client1: &apiservice.Client{
					Id:      wrapperspb.String("1"),
					Host:    wrapperspb.String("1.1.1.1"),
					Version: wrapperspb.String("Java-1.1.0"),
					Type:    apiservice.Client_SDK,
					Location: &apimodel.Location{
						Region: wrapperspb.String("region"),
						Zone:   wrapperspb.String("zone-1"),
						Campus: wrapperspb.String("campus"),
					},
					Stat: []*apiservice.StatInfo{
						{
							Target:   wrapperspb.String("prometheus"),
							Port:     wrapperspb.UInt32(28080),
							Path:     wrapperspb.String("/metrics"),
							Protocol: wrapperspb.String("http"),
						},
					},
				},
				client2: &apiservice.Client{
					Id:      wrapperspb.String("1"),
					Host:    wrapperspb.String("1.1.1.1"),
					Version: wrapperspb.String("Java-1.0.0"),
					Type:    apiservice.Client_SDK,
					Location: &apimodel.Location{
						Region: wrapperspb.String("region"),
						Zone:   wrapperspb.String("zone"),
						Campus: wrapperspb.String("campus"),
					},
					Stat: []*apiservice.StatInfo{
						{
							Target:   wrapperspb.String("prometheus"),
							Port:     wrapperspb.UInt32(28080),
							Path:     wrapperspb.String("/metrics"),
							Protocol: wrapperspb.String("http"),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "campus_not_equal",
			args: args{
				client1: &apiservice.Client{
					Id:      wrapperspb.String("1"),
					Host:    wrapperspb.String("1.1.1.1"),
					Version: wrapperspb.String("Java-1.1.0"),
					Type:    apiservice.Client_SDK,
					Location: &apimodel.Location{
						Region: wrapperspb.String("region"),
						Zone:   wrapperspb.String("zone"),
						Campus: wrapperspb.String("campus-1"),
					},
					Stat: []*apiservice.StatInfo{
						{
							Target:   wrapperspb.String("prometheus"),
							Port:     wrapperspb.UInt32(28080),
							Path:     wrapperspb.String("/metrics"),
							Protocol: wrapperspb.String("http"),
						},
					},
				},
				client2: &apiservice.Client{
					Id:      wrapperspb.String("1"),
					Host:    wrapperspb.String("1.1.1.1"),
					Version: wrapperspb.String("Java-1.0.0"),
					Type:    apiservice.Client_SDK,
					Location: &apimodel.Location{
						Region: wrapperspb.String("region"),
						Zone:   wrapperspb.String("zone"),
						Campus: wrapperspb.String("campus"),
					},
					Stat: []*apiservice.StatInfo{
						{
							Target:   wrapperspb.String("prometheus"),
							Port:     wrapperspb.UInt32(28080),
							Path:     wrapperspb.String("/metrics"),
							Protocol: wrapperspb.String("http"),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "stat_target_not_equal",
			args: args{
				client1: &apiservice.Client{
					Id:      wrapperspb.String("1"),
					Host:    wrapperspb.String("1.1.1.1"),
					Version: wrapperspb.String("Java-1.1.0"),
					Type:    apiservice.Client_SDK,
					Location: &apimodel.Location{
						Region: wrapperspb.String("region"),
						Zone:   wrapperspb.String("zone"),
						Campus: wrapperspb.String("campus"),
					},
					Stat: []*apiservice.StatInfo{
						{
							Target:   wrapperspb.String("prometheus-1"),
							Port:     wrapperspb.UInt32(28080),
							Path:     wrapperspb.String("/metrics"),
							Protocol: wrapperspb.String("http"),
						},
					},
				},
				client2: &apiservice.Client{
					Id:      wrapperspb.String("1"),
					Host:    wrapperspb.String("1.1.1.1"),
					Version: wrapperspb.String("Java-1.0.0"),
					Type:    apiservice.Client_SDK,
					Location: &apimodel.Location{
						Region: wrapperspb.String("region"),
						Zone:   wrapperspb.String("zone"),
						Campus: wrapperspb.String("campus"),
					},
					Stat: []*apiservice.StatInfo{
						{
							Target:   wrapperspb.String("prometheus"),
							Port:     wrapperspb.UInt32(28080),
							Path:     wrapperspb.String("/metrics"),
							Protocol: wrapperspb.String("http"),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "stat_port_not_equal",
			args: args{
				client1: &apiservice.Client{
					Id:      wrapperspb.String("1"),
					Host:    wrapperspb.String("1.1.1.1"),
					Version: wrapperspb.String("Java-1.1.0"),
					Type:    apiservice.Client_SDK,
					Location: &apimodel.Location{
						Region: wrapperspb.String("region"),
						Zone:   wrapperspb.String("zone"),
						Campus: wrapperspb.String("campus"),
					},
					Stat: []*apiservice.StatInfo{
						{
							Target:   wrapperspb.String("prometheus"),
							Port:     wrapperspb.UInt32(28081),
							Path:     wrapperspb.String("/metrics"),
							Protocol: wrapperspb.String("http"),
						},
					},
				},
				client2: &apiservice.Client{
					Id:      wrapperspb.String("1"),
					Host:    wrapperspb.String("1.1.1.1"),
					Version: wrapperspb.String("Java-1.0.0"),
					Type:    apiservice.Client_SDK,
					Location: &apimodel.Location{
						Region: wrapperspb.String("region"),
						Zone:   wrapperspb.String("zone"),
						Campus: wrapperspb.String("campus"),
					},
					Stat: []*apiservice.StatInfo{
						{
							Target:   wrapperspb.String("prometheus"),
							Port:     wrapperspb.UInt32(28080),
							Path:     wrapperspb.String("/metrics"),
							Protocol: wrapperspb.String("http"),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "stat_path_not_equal",
			args: args{
				client1: &apiservice.Client{
					Id:      wrapperspb.String("1"),
					Host:    wrapperspb.String("1.1.1.1"),
					Version: wrapperspb.String("Java-1.1.0"),
					Type:    apiservice.Client_SDK,
					Location: &apimodel.Location{
						Region: wrapperspb.String("region"),
						Zone:   wrapperspb.String("zone"),
						Campus: wrapperspb.String("campus"),
					},
					Stat: []*apiservice.StatInfo{
						{
							Target:   wrapperspb.String("prometheus"),
							Port:     wrapperspb.UInt32(28080),
							Path:     wrapperspb.String("/v1/metrics"),
							Protocol: wrapperspb.String("http"),
						},
					},
				},
				client2: &apiservice.Client{
					Id:      wrapperspb.String("1"),
					Host:    wrapperspb.String("1.1.1.1"),
					Version: wrapperspb.String("Java-1.0.0"),
					Type:    apiservice.Client_SDK,
					Location: &apimodel.Location{
						Region: wrapperspb.String("region"),
						Zone:   wrapperspb.String("zone"),
						Campus: wrapperspb.String("campus"),
					},
					Stat: []*apiservice.StatInfo{
						{
							Target:   wrapperspb.String("prometheus"),
							Port:     wrapperspb.UInt32(28080),
							Path:     wrapperspb.String("/metrics"),
							Protocol: wrapperspb.String("http"),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "stat_protocol_not_equal",
			args: args{
				client1: &apiservice.Client{
					Id:      wrapperspb.String("1"),
					Host:    wrapperspb.String("1.1.1.1"),
					Version: wrapperspb.String("Java-1.1.0"),
					Type:    apiservice.Client_SDK,
					Location: &apimodel.Location{
						Region: wrapperspb.String("region"),
						Zone:   wrapperspb.String("zone"),
						Campus: wrapperspb.String("campus"),
					},
					Stat: []*apiservice.StatInfo{
						{
							Target:   wrapperspb.String("prometheus"),
							Port:     wrapperspb.UInt32(28080),
							Path:     wrapperspb.String("/metrics"),
							Protocol: wrapperspb.String("tcp"),
						},
					},
				},
				client2: &apiservice.Client{
					Id:      wrapperspb.String("1"),
					Host:    wrapperspb.String("1.1.1.1"),
					Version: wrapperspb.String("Java-1.0.0"),
					Type:    apiservice.Client_SDK,
					Location: &apimodel.Location{
						Region: wrapperspb.String("region"),
						Zone:   wrapperspb.String("zone"),
						Campus: wrapperspb.String("campus"),
					},
					Stat: []*apiservice.StatInfo{
						{
							Target:   wrapperspb.String("prometheus"),
							Port:     wrapperspb.UInt32(28080),
							Path:     wrapperspb.String("/metrics"),
							Protocol: wrapperspb.String("http"),
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := service.ClientEquals(tt.args.client1, tt.args.client2); got != tt.want {
				t.Errorf("clientEquals() = %v, want %v", got, tt.want)
			}
		})
	}
}
