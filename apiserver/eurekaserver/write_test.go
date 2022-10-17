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
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store/mock"
)

func TestEurekaServer_renew(t *testing.T) {

	ins := &model.Instance{
		ServiceID: utils.NewUUID(),
		Proto: &api.Instance{
			Service:   utils.NewStringValue("echo"),
			Namespace: utils.NewStringValue("default"),
			Host:      utils.NewStringValue("127.0.0.1"),
			Port:      utils.NewUInt32Value(8080),
			HealthCheck: &api.HealthCheck{
				Type: api.HealthCheck_HEARTBEAT,
				Heartbeat: &api.HeartbeatHealthCheck{
					Ttl: &wrapperspb.UInt32Value{
						Value: 5,
					},
				},
			},
		},
		Valid: true,
	}

	insId, resp := utils.CheckInstanceTetrad(ins.Proto)
	if resp != nil {
		t.Fatal(resp.GetInfo().GetValue())
		return
	}

	ins.Proto.Id = utils.NewStringValue(insId)

	disableBeatIns := &model.Instance{
		ServiceID: utils.NewUUID(),
		Proto: &api.Instance{
			Service:   utils.NewStringValue("echo"),
			Namespace: utils.NewStringValue("default"),
			Host:      utils.NewStringValue("127.0.0.2"),
			Port:      utils.NewUInt32Value(8081),
			HealthCheck: &api.HealthCheck{
				Type: api.HealthCheck_HEARTBEAT,
				Heartbeat: &api.HeartbeatHealthCheck{
					Ttl: &wrapperspb.UInt32Value{
						Value: 5,
					},
				},
			},
		},
		Valid: true,
	}

	disableBeatInsId, resp := utils.CheckInstanceTetrad(disableBeatIns.Proto)
	if resp != nil {
		t.Fatal(resp.GetInfo().GetValue())
		return
	}

	disableBeatIns.Proto.Id = utils.NewStringValue(disableBeatInsId)

	eurekaSuit := &EurekaTestSuit{}
	if err := eurekaSuit.initialize(t, func(t *testing.T, s *mock.MockStore) error {
		s.
			EXPECT().
			GetMoreInstances(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			AnyTimes().
			Return(map[string]*model.Instance{
				insId:            ins,
				disableBeatInsId: disableBeatIns,
			}, nil)
		s.
			EXPECT().
			GetMoreServices(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			AnyTimes().
			Return(map[string]*model.Service{
				ins.ServiceID: {
					ID:        ins.ServiceID,
					Name:      ins.Proto.GetService().GetValue(),
					Namespace: ins.Proto.GetNamespace().GetValue(),
				},
			}, nil)

		s.EXPECT().GetInstancesCount().AnyTimes().Return(uint32(1), nil)
		s.EXPECT().GetUnixSecond().AnyTimes().Return(time.Now().Unix(), nil)
		s.EXPECT().Destroy().Return(nil)
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	defer eurekaSuit.Destroy()

	t.Run("eureka客户端心跳上报-实例正常且开启心跳", func(t *testing.T) {
		svr := &EurekaServer{
			healthCheckServer: eurekaSuit.healthSvr,
		}
		code := svr.renew(context.Background(), "", insId)
		assert.Equalf(t, api.ExecuteSuccess, code, "code need success, actual : %d", code)
	})

	t.Run("eureka客户端心跳上报-实例未开启心跳", func(t *testing.T) {
		svr := &EurekaServer{
			healthCheckServer: eurekaSuit.healthSvr,
		}
		code := svr.renew(context.Background(), "", disableBeatInsId)
		assert.Equalf(t, api.ExecuteSuccess, code, "code need success, actual : %d", code)
	})

	t.Run("eureka客户端心跳上报-实例不存在", func(t *testing.T) {
		svr := &EurekaServer{
			healthCheckServer: eurekaSuit.healthSvr,
		}
		code := svr.renew(context.Background(), "", utils.NewUUID())
		assert.Equalf(t, api.NotFoundResource, code, "code need notfound, actual : %d", code)
	})

}
