//go:build integration
// +build integration

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

package test

import (
	"testing"
	"time"

	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/test/integrate/grpc"
	"github.com/polarismesh/polaris/test/integrate/http"
	"github.com/polarismesh/polaris/test/integrate/resource"
)

/**
 * @brief 测试客户端GRPC接口
 */
func TestClientGRPC_DiscoverInstance(t *testing.T) {
	DiscoveryRunAndInitResource(t,
		func(t *testing.T, clientHttp *http.Client, namespaces []*apimodel.Namespace, services []*apiservice.Service) {
			clientGRPC, err := grpc.NewClient(grpcServerAddress)
			if err != nil {
				t.Fatalf("new grpc client fail")
			}
			defer clientGRPC.Close()

			client := resource.CreateClient(0)
			instances := resource.CreateInstances(services[0])

			t.Run("GRPC——上报SDK客户端信息", func(t *testing.T) {
				// 上报客户端信息
				err = clientGRPC.ReportClient(client)
				if err != nil {
					t.Fatalf("report client fail")
				}
				t.Log("report client success")
			})

			t.Run("GRPC——实例注册", func(t *testing.T) {
				// 注册服务实例
				err = clientGRPC.RegisterInstance(instances[0])
				if err != nil {
					t.Fatalf("register instance fail")
				}
				t.Log("register instance success")
			})

			time.Sleep(3 * time.Second) // 延迟

			var revision string
			t.Run("GRPC——首次发现实例", func(t *testing.T) {
				// 查询服务实例
				err = clientGRPC.Discover(apiservice.DiscoverRequest_INSTANCE, services[0], func(resp *apiservice.DiscoverResponse) {
					assert.Equal(t, api.ExecuteSuccess, resp.Code.GetValue(), "discover instance must be success")
					assert.Equal(t, 1, len(resp.Instances), "instance size must not be zero")
					revision = resp.Service.GetRevision().GetValue()
				})
				if err != nil {
					t.Fatalf("discover instance fail")
				}
				t.Log("discover instance success")
			})

			time.Sleep(time.Second)
			t.Run("GRPC——二次发现实例", func(t *testing.T) {
				// 查询服务实例
				copySvc := &(*services[0])
				copySvc.Revision = utils.NewStringValue(revision)

				err = clientGRPC.Discover(apiservice.DiscoverRequest_INSTANCE, copySvc, func(resp *apiservice.DiscoverResponse) {
					assert.Equal(t, api.DataNoChange, resp.Code.GetValue(), "discover instance must be datanotchange")
				})
				if err != nil {
					t.Fatalf("discover instance fail")
				}
				t.Log("discover instance success")
			})

			t.Run("GRPC——客户端心跳上报", func(t *testing.T) {
				// 上报心跳
				if err = clientGRPC.Heartbeat(instances[0]); err != nil {
					t.Fatalf("instance heartbeat fail")
				}
				t.Log("instance heartbeat success")
			})

			t.Run("GRPC——控制台修改实例信息", func(t *testing.T) {

				testMeta := map[string]string{
					"internal-mock-label": "polaris-auto-mock-test",
				}

				copyIns := &(*instances[0])
				copyIns.Metadata = testMeta

				if err := clientHttp.UpdateInstances([]*apiservice.Instance{copyIns}); err != nil {
					t.Fatalf("update instance fail : %s", err)
				}

				time.Sleep(5 * time.Second)

				err = clientGRPC.Discover(apiservice.DiscoverRequest_INSTANCE, services[0], func(resp *apiservice.DiscoverResponse) {
					assert.Equal(t, api.ExecuteSuccess, resp.Code.GetValue(), "discover instance must be success")
					assert.Equal(t, 1, len(resp.Instances), "instance size must not be zero")

					testMeta["version"] = instances[0].Version.GetValue()
					testMeta["protocol"] = instances[0].Protocol.GetValue()

					assert.True(t, resp.Instances[0].Metadata["internal-mock-label"] == "polaris-auto-mock-test", "instance metadata actual : %v", instances[0].Metadata)
				})
				if err != nil {
					t.Fatalf("discover instance fail")
				}
				t.Log("discover after update instance success")
			})

			t.Run("GRPC——反注册实例", func(t *testing.T) {
				// 反注册服务实例
				err = clientGRPC.DeregisterInstance(instances[0])
				if err != nil {
					t.Fatalf("deregister instance fail")
				}

				time.Sleep(2 * time.Second)

				err = clientGRPC.Discover(apiservice.DiscoverRequest_INSTANCE, services[0], func(resp *apiservice.DiscoverResponse) {
					assert.Equalf(t, api.ExecuteSuccess, resp.Code.GetValue(),
						"discover instance must success, actual code: %d", resp.GetCode().GetValue())
					assert.Equal(t, 0, len(resp.Instances), "instance size must be zero")
				})

				t.Log("deregister instance success")
			})
		})
}

func TestClientGRPC_DiscoverServices(t *testing.T) {
	DiscoveryRunAndInitResource(t,
		func(t *testing.T, clientHttp *http.Client, namespaces []*apimodel.Namespace, services []*apiservice.Service) {
			clientGRPC, err := grpc.NewClient(grpcServerAddress)
			if err != nil {
				t.Fatalf("new grpc client fail")
			}
			defer clientGRPC.Close()

			newNs := resource.CreateNamespaces()
			newSvcs := resource.CreateServices(newNs[0])

			t.Run("命名空间下未创建服务", func(t *testing.T) {
				resp, err := clientGRPC.DiscoverRequest(&apiservice.DiscoverRequest{
					Type: apiservice.DiscoverRequest_SERVICES,
					Service: &apiservice.Service{
						Namespace: &wrapperspb.StringValue{Value: newNs[0].Name.Value},
					},
				})
				if err != nil {
					t.Fatalf("discover services fail")
				}

				assert.True(t, len(resp.Services) == 0, "discover services response need empty")
			})

			t.Run("命名空间下创建了服务", func(t *testing.T) {

				// 创建服务
				_, err := clientHttp.CreateServices(newSvcs)
				if err != nil {
					t.Fatalf("create services fail")
				}
				t.Log("create services success")

				time.Sleep(5 * time.Second)

				resp, err := clientGRPC.DiscoverRequest(&apiservice.DiscoverRequest{
					Type: apiservice.DiscoverRequest_SERVICES,
					Service: &apiservice.Service{
						Namespace: &wrapperspb.StringValue{Value: newNs[0].Name.Value},
					},
				})
				if err != nil {
					t.Fatalf("discover services fail")
				}

				assert.False(t, len(resp.Services) == 0, "discover services response not empty")
				assert.Truef(t, len(newSvcs) == len(resp.Services),
					"discover services size not equal, expect : %d, actual : %s", len(newSvcs), len(resp.Services))
			})

		})
}

func Test_QueryGroups(t *testing.T) {
	ConfigCenterRunAndInitResource(t,
		func(t *testing.T, clientHttp *http.Client, namespaces []*apimodel.Namespace, configGroups []*apiconfig.ConfigFileGroup) {

		})
}
