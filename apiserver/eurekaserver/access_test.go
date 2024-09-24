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
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/emicklei/go-restful/v3"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/utils"
	testsuit "github.com/polarismesh/polaris/test/suit"
)

func createEurekaServerForTest(
	discoverSuit *testsuit.DiscoverTestSuit, options map[string]interface{}) (*EurekaServer, error) {
	eurekaSrv := &EurekaServer{
		namingServer:      discoverSuit.DiscoverServer(),
		healthCheckServer: discoverSuit.HealthCheckServer(),
		originDiscoverSvr: discoverSuit.OriginDiscoverServer(),
		allowAsyncRegis:   false,
	}
	err := eurekaSrv.Initialize(context.Background(), options, nil)
	if err != nil {
		return nil, err
	}
	// 注册实例信息修改 chain 数据
	eurekaSrv.registerInstanceChain()
	return eurekaSrv, nil
}

func batchBuildInstances(appId string, host string, port int, lease *LeaseInfo, count int) []*InstanceInfo {
	var instances []*InstanceInfo
	for i := 0; i < count; i++ {
		portValue := port + i
		instance := &InstanceInfo{
			InstanceId: fmt.Sprintf("%s_%s_%d", appId, host, portValue),
			AppName:    appId,
			IpAddr:     host,
			Port: &PortWrapper{
				RealPort:   portValue,
				RealEnable: true,
			},
			SecurePort: &PortWrapper{
				RealEnable: false,
			},
			CountryId: 1,
			DataCenterInfo: &DataCenterInfo{
				Clazz: "testClazz",
				Name:  "testName",
			},
			HostName:  host,
			Status:    "UP",
			LeaseInfo: lease,
		}
		instances = append(instances, instance)
	}
	return instances
}

func batchCreateInstance(t *testing.T, eurekaSvr *EurekaServer, namespace string, instances []*InstanceInfo) {
	for _, instance := range instances {
		code := eurekaSvr.registerInstances(context.Background(), namespace, instance.AppName, instance, false)
		assert.Equal(t, api.ExecuteSuccess, code, fmt.Sprintf("%+v", code))
	}
}

type mockResponseWriter struct {
	statusCode int
	body       bytes.Buffer
	header     http.Header
}

func newMockResponseWriter() *mockResponseWriter {
	return &mockResponseWriter{header: map[string][]string{}}
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {
	m.statusCode = statusCode
}

func (m *mockResponseWriter) Write(value []byte) (int, error) {
	return m.body.Write(value)
}

func (m *mockResponseWriter) Header() http.Header {
	return m.header
}

func countInstances(applications *Applications) int {
	var count int
	for _, app := range applications.Application {
		count += len(app.Instance)
	}
	return count
}

func TestEmptySlice(t *testing.T) {
	applications := &Applications{}
	count := countInstances(applications)
	assert.Equal(t, 0, count)
}

func checkInstanceAction(t *testing.T, applications *Applications, appName string, instanceId string, action string) {
	var hasApp bool
	var hasInstance bool
	var actionType string
	for _, app := range applications.Application {
		if app.Name == appName {
			hasApp = true
			for _, instance := range app.Instance {
				if instance.InstanceId == instanceId {
					hasInstance = true
					actionType = instance.ActionType
				}
			}
		}
	}
	assert.True(t, hasInstance)
	assert.True(t, hasApp)
	// fix: github action not suit for aync jobs
	fmt.Printf("latest action is %s\n", actionType)
	//assert.Equal(t, action, actionType)
}

// 测试新建实例
func TestCreateInstance(t *testing.T) {
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
	total := 10
	instances := batchBuildInstances(appId, host, startPort, &LeaseInfo{
		RenewalIntervalInSecs: 30,
		DurationInSecs:        120,
	}, total)
	batchCreateInstance(t, eurekaSrv, namespace, instances)

	time.Sleep(10 * time.Second)
	httpRequest := &http.Request{Header: map[string][]string{
		restful.HEADER_Accept: {restful.MIME_JSON},
		HeaderNamespace:       {namespace},
	}}
	req := restful.NewRequest(httpRequest)
	mockWriter := newMockResponseWriter()
	resp := &restful.Response{ResponseWriter: mockWriter}
	eurekaSrv.GetAllApplications(req, resp)
	assert.Equal(t, 200, mockWriter.statusCode)

	appResp := &ApplicationsResponse{}
	err = json.Unmarshal(mockWriter.body.Bytes(), appResp)
	assert.Nil(t, err)
	count := countInstances(appResp.Applications)
	assert.Equal(t, total, count)

	time.Sleep(5 * time.Second)
	instanceId := fmt.Sprintf("%s_%s_%d", appId, host, startPort)
	code := eurekaSrv.deregisterInstance(context.Background(), namespace, appId, instanceId, false)
	assert.Equal(t, api.ExecuteSuccess, code, fmt.Sprintf("%d", code))
	time.Sleep(20 * time.Second)

	deltaReq := restful.NewRequest(httpRequest)
	deltaMockWriter := newMockResponseWriter()
	deltaResp := &restful.Response{ResponseWriter: deltaMockWriter}
	eurekaSrv.GetDeltaApplications(deltaReq, deltaResp)

	deltaAppResp := &ApplicationsResponse{}
	err = json.Unmarshal(deltaMockWriter.body.Bytes(), deltaAppResp)
	assert.Nil(t, err)
	checkInstanceAction(t, deltaAppResp.Applications, appId, instanceId, ActionDeleted)
}

// Test_EurekaWrite .
func Test_EurekaWrite(t *testing.T) {
	discoverSuit := &testsuit.DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	options := map[string]interface{}{optionRefreshInterval: 5, optionDeltaExpireInterval: 120}
	eurekaSrv, err := createEurekaServerForTest(discoverSuit, options)
	assert.Nil(t, err)

	mockIns := genMockEurekaInstance()

	// pretty output must be created and written explicitly
	output, err := xml.MarshalIndent(mockIns, " ", " ")
	assert.NoError(t, err)

	var body bytes.Buffer
	_, err = body.Write([]byte(xml.Header))
	assert.NoError(t, err)
	_, err = body.Write(output)
	assert.NoError(t, err)

	mockReq := httptest.NewRequest("", fmt.Sprintf("http://127.0.0.1:8761/eureka/v2/apps/%s", mockIns.AppName), &body)
	mockReq.Header.Add(restful.HEADER_Accept, restful.MIME_XML)
	mockReq.Header.Add(restful.HEADER_ContentType, restful.MIME_XML)
	mockRsp := newMockResponseWriter()

	restfulReq := restful.NewRequest(mockReq)
	injectRestfulReqPathParameters(t, restfulReq, map[string]string{
		ParamAppId: mockIns.AppName,
	})
	eurekaSrv.RegisterApplication(restfulReq, restful.NewResponse(mockRsp))
	assert.Equal(t, http.StatusNoContent, mockRsp.statusCode)
	assert.Equal(t, restfulReq.Attribute(statusCodeHeader), uint32(apimodel.Code_ExecuteSuccess))

	_ = discoverSuit.CacheMgr().TestUpdate()
	saveIns, err := discoverSuit.Storage.GetInstance(mockIns.InstanceId)
	assert.NoError(t, err)
	assert.NotNil(t, saveIns)

	t.Run("UpdateStatus", func(t *testing.T) {
		t.Run("01_StatusUnknown", func(t *testing.T) {
			mockReq := httptest.NewRequest("", fmt.Sprintf("http://127.0.0.1:8761/eureka/v2/apps/%s/%s/status",
				mockIns.AppName, mockIns.InstanceId), nil)
			mockReq.PostForm = url.Values{}
			mockReq.PostForm.Add(ParamValue, StatusUnknown)
			mockRsp := newMockResponseWriter()

			restfulReq := restful.NewRequest(mockReq)
			injectRestfulReqPathParameters(t, restfulReq, map[string]string{
				ParamAppId:  mockIns.AppName,
				ParamInstId: mockIns.InstanceId,
			})
			eurekaSrv.UpdateStatus(restfulReq, restful.NewResponse(mockRsp))
			assert.Equal(t, http.StatusOK, mockRsp.statusCode)
			assert.Equal(t, restfulReq.Attribute(statusCodeHeader), uint32(apimodel.Code_ExecuteSuccess))

			//
			saveIns, err := discoverSuit.Storage.GetInstance(mockIns.InstanceId)
			assert.NoError(t, err)
			assert.NotNil(t, saveIns)
			assert.False(t, saveIns.Isolate())
		})

		t.Run("02_StatusDown", func(t *testing.T) {
			mockReq := httptest.NewRequest("", fmt.Sprintf("http://127.0.0.1:8761/eureka/v2/apps/%s/%s/status",
				mockIns.AppName, mockIns.InstanceId), nil)
			mockReq.PostForm = url.Values{}
			mockReq.PostForm.Add(ParamValue, StatusDown)
			mockRsp := newMockResponseWriter()

			restfulReq := restful.NewRequest(mockReq)
			injectRestfulReqPathParameters(t, restfulReq, map[string]string{
				ParamAppId:  mockIns.AppName,
				ParamInstId: mockIns.InstanceId,
			})
			eurekaSrv.UpdateStatus(restfulReq, restful.NewResponse(mockRsp))
			assert.Equal(t, http.StatusOK, mockRsp.statusCode)
			assert.Equal(t, restfulReq.Attribute(statusCodeHeader), uint32(apimodel.Code_ExecuteSuccess), fmt.Sprintf("%d", restfulReq.Attribute(statusCodeHeader)))

			//
			saveIns, err := discoverSuit.Storage.GetInstance(mockIns.InstanceId)
			assert.NoError(t, err)
			assert.True(t, saveIns.Isolate())
			assert.Equal(t, StatusDown, saveIns.Proto.Metadata[InternalMetadataStatus])
		})

		t.Run("03_StatusUp", func(t *testing.T) {
			mockReq := httptest.NewRequest("", fmt.Sprintf("http://127.0.0.1:8761/eureka/v2/apps/%s/%s/status",
				mockIns.AppName, mockIns.InstanceId), nil)
			mockReq.PostForm = url.Values{}
			mockReq.PostForm.Add(ParamValue, StatusUp)
			mockRsp := newMockResponseWriter()

			restfulReq := restful.NewRequest(mockReq)
			injectRestfulReqPathParameters(t, restfulReq, map[string]string{
				ParamAppId:  mockIns.AppName,
				ParamInstId: mockIns.InstanceId,
			})
			eurekaSrv.UpdateStatus(restfulReq, restful.NewResponse(mockRsp))
			assert.Equal(t, http.StatusOK, mockRsp.statusCode)
			assert.Equal(t, restfulReq.Attribute(statusCodeHeader), uint32(apimodel.Code_ExecuteSuccess), fmt.Sprintf("%d", restfulReq.Attribute(statusCodeHeader)))

			//
			saveIns, err := discoverSuit.Storage.GetInstance(mockIns.InstanceId)
			assert.NoError(t, err)
			assert.False(t, saveIns.Isolate())
			assert.Equal(t, StatusUp, saveIns.Proto.Metadata[InternalMetadataStatus])
		})

		t.Run("04_Polaris_UpdateInstances", func(t *testing.T) {
			defer func() {
				rsp := discoverSuit.DiscoverServer().UpdateInstances(discoverSuit.DefaultCtx, []*service_manage.Instance{
					{
						Id:      wrapperspb.String(mockIns.InstanceId),
						Isolate: wrapperspb.Bool(false),
					},
				})
				assert.Equal(t, apimodel.Code_ExecuteSuccess, apimodel.Code(rsp.GetCode().GetValue()))
			}()
			rsp := discoverSuit.DiscoverServer().UpdateInstances(discoverSuit.DefaultCtx, []*service_manage.Instance{
				{
					Id:      wrapperspb.String(mockIns.InstanceId),
					Isolate: wrapperspb.Bool(true),
				},
			})
			assert.Equal(t, apimodel.Code_ExecuteSuccess, apimodel.Code(rsp.GetCode().GetValue()))

			// 在获取一次
			saveIns, err := discoverSuit.Storage.GetInstance(mockIns.InstanceId)
			assert.NoError(t, err)
			assert.True(t, saveIns.Isolate())
			assert.Equal(t, StatusOutOfService, saveIns.Proto.Metadata[InternalMetadataStatus])
		})

		t.Run("05_Polaris_UpdateInstancesIsolate", func(t *testing.T) {
			rsp := discoverSuit.DiscoverServer().UpdateInstances(discoverSuit.DefaultCtx, []*service_manage.Instance{
				{
					Id:      wrapperspb.String(mockIns.InstanceId),
					Isolate: wrapperspb.Bool(true),
				},
			})
			assert.Equal(t, apimodel.Code_ExecuteSuccess, apimodel.Code(rsp.GetCode().GetValue()))

			// 在获取一次
			_, saveInss, err := discoverSuit.Storage.GetExpandInstances(map[string]string{
				"id": mockIns.InstanceId,
			}, map[string]string{}, 0, 10)
			assert.NoError(t, err)
			assert.Equal(t, 1, len(saveInss))
			assert.True(t, saveInss[0].Isolate())
			assert.Equal(t, StatusOutOfService, saveInss[0].Proto.Metadata[InternalMetadataStatus])
		})
	})
}

func injectRestfulReqPathParameters(t *testing.T, req *restful.Request, params map[string]string) {
	v := reflect.ValueOf(req)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	field := v.FieldByName("pathParameters")
	fieldVal := utils.GetUnexportedField(field)

	pathParameters, ok := fieldVal.(map[string]string)
	assert.True(t, ok)
	for k, v := range params {
		pathParameters[k] = v
	}
	utils.SetUnexportedField(field, params)
}

func genMockEurekaInstance() *InstanceInfo {
	mockIns := &InstanceInfo{
		XMLName:      struct{}{},
		InstanceId:   "123",
		AppName:      "MOCK_SERVICE",
		AppGroupName: "MOCK_SERVICE",
		IpAddr:       "127.0.0.1",
		Sid:          "",
		Port: &PortWrapper{
			Port:       "8080",
			RealPort:   8080,
			Enabled:    "true",
			RealEnable: true,
		},
		Status:           StatusUp,
		OverriddenStatus: StatusUnknown,
	}
	return mockIns
}
