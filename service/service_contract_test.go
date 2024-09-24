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
	"fmt"
	"testing"

	"github.com/golang/protobuf/ptypes"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/polarismesh/polaris/common/utils"
)

func TestServer_CreateServiceContracts(t *testing.T) {
	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}

	discoverSuit.CleanServiceContract()
	t.Cleanup(func() {
		discoverSuit.CleanServiceContract()
		discoverSuit.Destroy()
	})

	expectTotal := 10
	mockData := mockServiceContracts(expectTotal, true)

	t.Run("01-正常创建一个服务契约配置", func(t *testing.T) {
		resp := discoverSuit.DiscoverServer().CreateServiceContracts(discoverSuit.DefaultCtx, mockData)
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), resp.GetCode().GetValue(), resp.String())

		_ = discoverSuit.CacheMgr().TestUpdate()

		t.Run("该接口不会保存Interfaces信息", func(t *testing.T) {
			queryRsp := discoverSuit.DiscoverServer().GetServiceContracts(discoverSuit.DefaultCtx, map[string]string{
				"name": "mock-name*",
			})
			assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), queryRsp.GetCode().GetValue(), queryRsp.String())
			assert.Equal(t, expectTotal, len(queryRsp.GetData()))
			specData, err := unmarshalServiceContratcs(queryRsp.GetData())
			assert.Nil(t, err)
			for i := range specData {
				assert.Equal(t, 0, len(specData[i].GetInterfaces()))
			}
		})

		t.Run("更新Content", func(t *testing.T) {
			mockData[0].Content = "new content"
			uRsp := discoverSuit.DiscoverServer().CreateServiceContracts(discoverSuit.DefaultCtx, []*service_manage.ServiceContract{mockData[0]})
			assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), uRsp.GetCode().GetValue(), uRsp.String())

			mockId, _ := utils.CheckContractTetrad(mockData[0])
			discoverSuit.Storage.GetServiceContract(mockId)

			_ = discoverSuit.CacheMgr().TestUpdate()
			queryRsp := discoverSuit.DiscoverServer().GetServiceContracts(discoverSuit.DefaultCtx, map[string]string{
				"name": mockData[0].Name,
			})
			assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), queryRsp.GetCode().GetValue(), queryRsp.String())
			assert.Equal(t, 1, len(queryRsp.GetData()))
			specData, err := unmarshalServiceContratcs(queryRsp.GetData())
			assert.Nil(t, err)
			assert.Equal(t, mockData[0].Content, specData[0].Content)
		})

		t.Run("测试模糊匹配", func(t *testing.T) {
			queryRsp := discoverSuit.DiscoverServer().GetServiceContracts(discoverSuit.DefaultCtx, map[string]string{
				"name": "mock-name*",
			})
			assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), queryRsp.GetCode().GetValue(), resp.String())
			assert.Equal(t, expectTotal, len(queryRsp.GetData()))
		})

		t.Run("测试模糊匹配-分页测试", func(t *testing.T) {
			queryRsp := discoverSuit.DiscoverServer().GetServiceContracts(discoverSuit.DefaultCtx, map[string]string{
				"name":   "mock-name*",
				"offset": "0",
				"limit":  "1",
			})
			assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), queryRsp.GetCode().GetValue(), resp.String())
			assert.Equal(t, 1, len(queryRsp.GetData()))
		})

		t.Run("测试模糊匹配-分页测试", func(t *testing.T) {
			queryRsp := discoverSuit.DiscoverServer().GetServiceContracts(discoverSuit.DefaultCtx, map[string]string{
				"name":   "mock-name*",
				"offset": "100",
				"limit":  "1",
			})
			assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), queryRsp.GetCode().GetValue(), resp.String())
			assert.Equal(t, 0, len(queryRsp.GetData()))
		})

		t.Run("测试模糊匹配-匹配失败", func(t *testing.T) {
			queryRsp := discoverSuit.DiscoverServer().GetServiceContracts(discoverSuit.DefaultCtx, map[string]string{
				"name": "naame*",
			})
			assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), queryRsp.GetCode().GetValue(), resp.String())
			assert.Equal(t, 0, len(queryRsp.GetData()))
		})

		t.Run("测试全匹配", func(t *testing.T) {
			queryRsp := discoverSuit.DiscoverServer().GetServiceContracts(discoverSuit.DefaultCtx, map[string]string{
				"name": "mock-name-0",
			})
			assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), queryRsp.GetCode().GetValue(), resp.String())
			assert.Equal(t, 1, len(queryRsp.GetData()))
		})
	})

	t.Run("02-更新服务契约", func(t *testing.T) {
		copyData := mockData[expectTotal-1]

		resp := discoverSuit.DiscoverServer().DeleteServiceContractInterfaces(discoverSuit.DefaultCtx, copyData)
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), resp.GetCode().GetValue(), resp.String())

		_ = discoverSuit.CacheMgr().TestUpdate()

		queryRsp := discoverSuit.DiscoverServer().GetServiceContracts(discoverSuit.DefaultCtx, map[string]string{
			"name": copyData.Name,
		})
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), queryRsp.GetCode().GetValue(), resp.String())
		assert.Equal(t, 1, len(queryRsp.GetData()))

		specData, err := unmarshalServiceContratcs(queryRsp.GetData())
		assert.Nil(t, err)
		assert.Equal(t, 1, len(specData))
		assert.Equal(t, 0, len(specData[0].GetInterfaces()))

		t.Run("手动追加服务契约接口", func(t *testing.T) {
			copyData = mockData[expectTotal-1]
			mockServiceContractInterfaces(copyData)
			resp = discoverSuit.DiscoverServer().AppendServiceContractInterfaces(discoverSuit.DefaultCtx, copyData, service_manage.InterfaceDescriptor_Manual)
			assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), resp.GetCode().GetValue(), resp.String())
			_ = discoverSuit.CacheMgr().TestUpdate()

			queryRsp = discoverSuit.DiscoverServer().GetServiceContracts(discoverSuit.DefaultCtx, map[string]string{
				"name": copyData.Name,
			})
			assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), queryRsp.GetCode().GetValue(), resp.String())
			assert.Equal(t, 1, len(queryRsp.GetData()))

			specData, err = unmarshalServiceContratcs(queryRsp.GetData())
			assert.Nil(t, err)
			assert.Equal(t, 1, len(specData))
			assert.Equal(t, 4, len(specData[0].GetInterfaces()))
			for i := range specData[0].GetInterfaces() {
				item := specData[0].GetInterfaces()[i]
				assert.Equal(t, service_manage.InterfaceDescriptor_Manual, item.Source)
			}
		})

		t.Run("客户端追加服务契约接口", func(t *testing.T) {
			copyData = mockData[expectTotal-1]
			mockServiceContractInterfaces(copyData)
			resp = discoverSuit.DiscoverServer().AppendServiceContractInterfaces(discoverSuit.DefaultCtx, copyData, service_manage.InterfaceDescriptor_Client)
			assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), resp.GetCode().GetValue(), resp.String())
			_ = discoverSuit.CacheMgr().TestUpdate()

			queryRsp = discoverSuit.DiscoverServer().GetServiceContracts(discoverSuit.DefaultCtx, map[string]string{
				"name": copyData.Name,
			})
			assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), queryRsp.GetCode().GetValue(), resp.String())
			assert.Equal(t, 1, len(queryRsp.GetData()))

			specData, err = unmarshalServiceContratcs(queryRsp.GetData())
			assert.Nil(t, err)
			assert.Equal(t, 1, len(specData))
			assert.Equal(t, 4, len(specData[0].GetInterfaces()))
			manualCount := 0
			clientCount := 0
			for i := range specData[0].GetInterfaces() {
				item := specData[0].GetInterfaces()[i]
				switch item.Source {
				case service_manage.InterfaceDescriptor_Client:
					clientCount++
				case service_manage.InterfaceDescriptor_Manual:
					manualCount++
				}
			}

			assert.Equal(t, 0, clientCount)
			assert.Equal(t, 4, manualCount)
		})

		t.Run("客户端追加新服务契约接口", func(t *testing.T) {
			copyData = mockData[expectTotal-1]
			mockServiceContractInterfaces(copyData)
			for i := range copyData.Interfaces {
				item := copyData.Interfaces[i]
				item.Path = "/api/" + item.Path
				copyData.Interfaces[i] = item
			}
			resp = discoverSuit.DiscoverServer().AppendServiceContractInterfaces(discoverSuit.DefaultCtx, copyData, service_manage.InterfaceDescriptor_Client)
			assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), resp.GetCode().GetValue(), resp.String())
			_ = discoverSuit.CacheMgr().TestUpdate()

			queryRsp = discoverSuit.DiscoverServer().GetServiceContracts(discoverSuit.DefaultCtx, map[string]string{
				"name": copyData.Name,
			})
			assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), queryRsp.GetCode().GetValue(), resp.String())
			assert.Equal(t, 1, len(queryRsp.GetData()))

			specData, err = unmarshalServiceContratcs(queryRsp.GetData())
			assert.Nil(t, err)
			assert.Equal(t, 1, len(specData))
			assert.Equal(t, 8, len(specData[0].GetInterfaces()))
			manualCount := 0
			clientCount := 0
			for i := range specData[0].GetInterfaces() {
				item := specData[0].GetInterfaces()[i]
				switch item.Source {
				case service_manage.InterfaceDescriptor_Client:
					clientCount++
				case service_manage.InterfaceDescriptor_Manual:
					manualCount++
				}
			}

			assert.Equal(t, 4, clientCount)
			assert.Equal(t, 4, manualCount)
		})
	})

	t.Run("03-删除服务契约", func(t *testing.T) {
		resp := discoverSuit.DiscoverServer().DeleteServiceContracts(discoverSuit.DefaultCtx, mockData)
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), resp.GetCode().GetValue(), resp.String())
		assert.Equal(t, expectTotal, len(resp.GetResponses()), resp.String())
		_ = discoverSuit.CacheMgr().TestUpdate()

		queryRsp := discoverSuit.DiscoverServer().GetServiceContracts(discoverSuit.DefaultCtx, map[string]string{
			"name": "mock-name*",
		})
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), queryRsp.GetCode().GetValue(), queryRsp.String())
		assert.Equal(t, 0, len(queryRsp.GetData()))
	})
}

func mockServiceContractInterfaces(mockData *service_manage.ServiceContract) {
	mockData.Interfaces = make([]*service_manage.InterfaceDescriptor, 0, 4)
	for j := 0; j < 4; j++ {
		mockData.Interfaces = append(mockData.Interfaces, &service_manage.InterfaceDescriptor{
			Method:  fmt.Sprintf("new-mock-method-%d", j),
			Path:    fmt.Sprintf("new-mock-path-%d", j),
			Content: fmt.Sprintf("new-mock-content-%d", j),
			Source:  service_manage.InterfaceDescriptor_Client,
		})
	}
}

func mockServiceContracts(total int, needInterfaces bool) []*service_manage.ServiceContract {
	ret := make([]*service_manage.ServiceContract, 0, total)
	for i := 0; i < total; i++ {
		mockData := &service_manage.ServiceContract{
			Name:      fmt.Sprintf("mock-name-%d", i),
			Namespace: fmt.Sprintf("mock-namespace-%d", i),
			Service:   fmt.Sprintf("mock-service-%d", i),
			Protocol:  fmt.Sprintf("mock-protocol-%d", i),
			Version:   fmt.Sprintf("mock-version-%d", i),
			Content:   fmt.Sprintf("mock-content-%d", i),
		}
		if needInterfaces {
			mockData.Interfaces = make([]*service_manage.InterfaceDescriptor, 0, 4)
			for j := 0; j < 4; j++ {
				mockData.Interfaces = append(mockData.Interfaces, &service_manage.InterfaceDescriptor{
					Method:  fmt.Sprintf("mock-method-%d-%d", i, j),
					Path:    fmt.Sprintf("mock-path-%d-%d", i, j),
					Content: fmt.Sprintf("mock-content-%d-%d", i, j),
					Source:  service_manage.InterfaceDescriptor_Client,
				})
			}
		}
		ret = append(ret, mockData)
	}
	return ret
}

// unmarshalServiceContratcs 转换为 []*service_manage.ServiceContract 数组
func unmarshalServiceContratcs(routings []*anypb.Any) ([]*service_manage.ServiceContract, error) {
	ret := make([]*service_manage.ServiceContract, 0, len(routings))

	for i := range routings {
		entry := routings[i]

		msg := &service_manage.ServiceContract{}
		if err := ptypes.UnmarshalAny(entry, msg); err != nil {
			return nil, err
		}

		ret = append(ret, msg)
	}
	return ret, nil
}
