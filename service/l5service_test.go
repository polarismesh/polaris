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

// . "github.com/smartystreets/goconvey/convey"

// // safe get cache data
// func safeSyncByAgentCmd(ctx context.Context, sbac *l5.Cl5SyncByAgentCmd) (*l5.Cl5SyncByAgentAckCmd, error) {
// 	time.Sleep(updateCacheInterval)
// 	return server.SyncByAgentCmd(ctx, sbac)
// }

// // get maxFlow from t_route
// func getMaxRouteFlow(t *testing.T) int {
// 	maxStr := "select IFNULL(max(fflow),0) from t_route"
// 	var maxFlow int
// 	err := db.QueryRow(maxStr).Scan(&maxFlow)
// 	switch {
// 	case err == sql.ErrNoRows:
// 		maxFlow = 0
// 	case err != nil:
// 		t.Fatalf("error: %s", err.Error())
// 	}

// 	return maxFlow + 1
// }

// // add l5 t_route
// func addL5Route(t *testing.T, ip, modID, cmdID int32, setID string) {
// 	maxFlow := getMaxRouteFlow(t)
// 	str := "replace into t_route(fip, fmodid, fcmdid, fsetId, fflag, fstamp, fflow) values(?,?,?,?,0,now(),?)"
// 	if _, err := db.Exec(str, ip, modID, cmdID, setID, maxFlow+1); err != nil {
// 		t.Fatalf("error: %s", err.Error())
// 	}
// }

// // 删除t_route
// func deleteL5Route(t *testing.T, ip, modID, cmdID int32) {
// 	maxFlow := getMaxRouteFlow(t)
// 	str := "update t_route set fflag = 1, fflow = ? where fip = ? and fmodid = ? and fcmdid = ?"
// 	if _, err := db.Exec(str, maxFlow, ip, modID, cmdID); err != nil {
// 		t.Fatalf("error: %s", err.Error())
// 	}
// }

// // 创建带SetID的实例列表
// // setID可以为空
// func createInstanceWithSetID(service *api.Service, index int, setIDs string, weights string) *api.Instance {
// 	instance := &api.Instance{
// 		Service:      service.GetName(),
// 		Namespace:    service.GetNamespace(),
// 		Host:         utils.NewStringValue(fmt.Sprintf("10.235.25.%d", index)),
// 		Port:         utils.NewUInt32Value(8080),
// 		ServiceToken: service.GetToken(),
// 	}
// 	if setIDs != "" {
// 		instance.Metadata = map[string]string{"internal-cl5-setId": setIDs}
// 	}
// 	if weights != "" {
// 		if instance.Metadata == nil {
// 			instance.Metadata = make(map[string]string)
// 		}
// 		instance.Metadata["internal-cl5-weight"] = weights
// 	}
// 	resp := server.CreateInstance(defaultCtx, instance)
// 	So(respSuccess(resp), ShouldEqual, true)
// 	return resp.GetInstance()
// }

// // 测试兼容l5协议的流程
// func TestSyncByAgentCmd(t *testing.T) {
// 	Convey("获取老Cl5的Sid数据", t, func() {
// 		reqCmd := &l5.Cl5SyncByAgentCmd{AgentIp: proto.Int32(11111), SyncFlow: proto.Int32(22222)}
// 		modID := int32(64850433)
// 		cmdID := int32(65540)
// 		service := &api.Service{
// 			Name:      utils.NewStringValue(fmt.Sprintf("%d:%d", modID, cmdID)),
// 			Namespace: utils.NewStringValue("default"),
// 			Owners:    utils.NewStringValue("aa"),
// 		}
// 		serviceResp := server.CreateService(defaultCtx, service)
// 		So(respSuccess(serviceResp), ShouldEqual, true)
// 		defer cleanServiceName(service.GetName().GetValue(), service.GetNamespace().GetValue())

// 		Convey("正常数据获取", func() {
// 			reqCmd.OptList = &l5.Cl5OptList{
// 				Opt: []*l5.Cl5OptObj{{ModId: proto.Int32(modID), CmdId: proto.Int32(cmdID)}},
// 			}

// 			ack, err := safeSyncByAgentCmd(defaultCtx, reqCmd)
// 			So(err, ShouldBeNil)
// 			So(ack.GetAgentIp(), ShouldEqual, reqCmd.GetAgentIp())
// 			So(ack.GetSyncFlow(), ShouldEqual, reqCmd.GetSyncFlow()+1)
// 			So(len(ack.GetServList().GetServ()), ShouldEqual, 0)

// 			for i := 0; i < 5; i++ {
// 				resp := createInstanceWithSetID(serviceResp.GetService(), i, "", "")
// 				defer cleanInstance(resp.GetId().GetValue())
// 			}

// 			ack, _ = safeSyncByAgentCmd(defaultCtx, reqCmd)
// 			So(len(ack.GetServList().GetServ()), ShouldEqual, 5)
// 		})
// 		Convey("一个实例属于一个set功能验证", func() {
// 			// 新建一些带set的被调
// 			for i := 0; i < 10; i++ {
// 				resp := createInstanceWithSetID(serviceResp.GetService(), i, fmt.Sprintf("SET_%d", i%2), "")
// 				defer cleanInstance(resp.GetId().GetValue())
// 			}

// 			ack, _ := safeSyncByAgentCmd(defaultCtx, reqCmd)
// 			So(len(ack.GetServList().GetServ()), ShouldEqual, 0)

// 			addL5Route(t, reqCmd.GetAgentIp(), modID, cmdID, "SET_1")
// 			defer deleteL5Route(t, reqCmd.GetAgentIp(), modID, cmdID)
// 			ack, _ = safeSyncByAgentCmd(defaultCtx, reqCmd)
// 			So(len(ack.GetServList().GetServ()), ShouldEqual, 5) // SET_0 SET_1 各一半
// 		})
// 		Convey("一个实例多个set功能验证", func() {
// 			// 新建一些带set的被调
// 			setIDs := "SET_X"
// 			weights := "0"
// 			for i := 0; i < 10; i++ {
// 				setIDs = setIDs + fmt.Sprintf(",SET_%d", i)
// 				weights = weights + fmt.Sprintf(",%d", (i+1)*100)
// 				resp := createInstanceWithSetID(serviceResp.GetService(), i, setIDs, weights)
// 				defer cleanInstance(resp.GetId().GetValue())
// 			}
// 			// SET_X,SET_0,  			0,100
// 			// SET_X,SET_0,SET_1 		0,100,200
// 			// SET_X,SET_0,SET_1,SET_2 	0,100,200,300
// 			// ...
// 			ack, _ := safeSyncByAgentCmd(defaultCtx, reqCmd)
// 			So(len(ack.GetServList().GetServ()), ShouldEqual, 0)
// 			for i := 0; i < 10; i++ {
// 				addL5Route(t, reqCmd.GetAgentIp(), modID, cmdID, fmt.Sprintf("SET_%d", i))
// 				ack, _ = safeSyncByAgentCmd(defaultCtx, reqCmd)
// 				So(len(ack.GetServList().GetServ()), ShouldEqual, 10-i)
// 				for _, callee := range ack.GetServList().GetServ() {
// 					So(callee.GetWeight(), ShouldEqual, (i+1)*100)
// 				}
// 			}

// 			// SET_X weight=0
// 			addL5Route(t, reqCmd.GetAgentIp(), modID, cmdID, "SET_X")
// 			ack, _ = safeSyncByAgentCmd(defaultCtx, reqCmd)
// 			So(len(ack.GetServList().GetServ()), ShouldEqual, 0)

// 			addL5Route(t, reqCmd.GetAgentIp(), modID, cmdID, fmt.Sprintf("SET_%d", 20))
// 			defer deleteL5Route(t, reqCmd.GetAgentIp(), modID, cmdID)
// 			ack, _ = safeSyncByAgentCmd(defaultCtx, reqCmd)
// 			So(len(ack.GetServList().GetServ()), ShouldEqual, 0)
// 		})
// 	})
// }

// // 测试根据埋点server获取到后端serverList的功能
// func TestCl5DiscoverTest(t *testing.T) {
// 	createDiscoverServer := func(name string) *api.Service {
// 		service := &api.Service{
// 			Name:      utils.NewStringValue(name),
// 			Namespace: utils.NewStringValue("Polaris"),
// 			Owners:    utils.NewStringValue("my"),
// 		}
// 		resp := server.CreateService(defaultCtx, service)
// 		So(respSuccess(resp), ShouldEqual, true)
// 		So(resp.GetService(), ShouldNotBeNil)
// 		return resp.GetService()
// 	}
// 	createDiscoverInstance := func(service *api.Service, index int) *api.Instance {
// 		instance := &api.Instance{
// 			Service:      service.GetName(),
// 			Namespace:    service.GetNamespace(),
// 			ServiceToken: service.GetToken(),
// 			Host:         utils.NewStringValue(fmt.Sprintf("10.0.0.%d", index)),
// 			Port:         utils.NewUInt32Value(7779),
// 			Protocol:     utils.NewStringValue("l5pb"),
// 			Healthy:      utils.NewBoolValue(true),
// 		}
// 		resp := server.CreateInstance(defaultCtx, instance)
// 		So(respSuccess(resp), ShouldEqual, true)
// 		So(resp.GetInstance(), ShouldNotBeNil)
// 		return resp.GetInstance()
// 	}

// 	Convey("测试根据埋点server获取到后端serverList的功能", t, func() {
// 		reqCmd := &l5.Cl5SyncByAgentCmd{
// 			AgentIp: proto.Int32(123),
// 			OptList: &l5.Cl5OptList{Opt: []*l5.Cl5OptObj{{ModId: proto.Int32(111), CmdId: proto.Int32(222)}}},
// 		}
// 		name := "test-api.cl5.discover"
// 		discover := createDiscoverServer(name)
// 		defer cleanServiceName(discover.GetName().GetValue(), discover.GetNamespace().GetValue())
// 		instance := createDiscoverInstance(discover, 0)
// 		defer cleanInstance(instance.GetId().GetValue())

// 		discover1 := createDiscoverServer(name + ".1")
// 		defer cleanServiceName(discover1.GetName().GetValue(), discover1.GetNamespace().GetValue())
// 		discover2 := createDiscoverServer(name + ".2")
// 		defer cleanServiceName(discover2.GetName().GetValue(), discover2.GetNamespace().GetValue())

// 		ctx := context.WithValue(defaultCtx, utils.Cl5ServerCluster{}, name)
// 		ctx = context.WithValue(ctx, utils.Cl5ServerProtocol{}, "l5pb")
// 		Convey("只有默认集群，则返回默认集群的数据", func() {
// 			ack, _ := safeSyncByAgentCmd(ctx, reqCmd)
// 			So(len(ack.GetL5SvrList().GetIp()), ShouldEqual, 1)
// 			t.Logf("%+v", ack)
// 		})
// 		Convey("不同请求IP获取到不同的集群", func() {
// 			discover.Metadata = map[string]string{"internal-cluster-count": "2"}
// 			So(respSuccess(server.UpdateService(defaultCtx, discover)), ShouldEqual, true)
// 			instance1 := createDiscoverInstance(discover1, 1)
// 			defer cleanInstance(instance1.GetId().GetValue())
// 			instance2 := createDiscoverInstance(discover1, 2)
// 			defer cleanInstance(instance2.GetId().GetValue())

// 			instance3 := createDiscoverInstance(discover2, 3)
// 			defer cleanInstance(instance3.GetId().GetValue())
// 			instance4 := createDiscoverInstance(discover2, 4)
// 			defer cleanInstance(instance4.GetId().GetValue())
// 			instance5 := createDiscoverInstance(discover2, 5)
// 			defer cleanInstance(instance5.GetId().GetValue())

// 			reqCmd.AgentIp = proto.Int32(56352420) // clusterIndex := ip %count + 1
// 			ack, _ := safeSyncByAgentCmd(ctx, reqCmd)
// 			So(len(ack.GetL5SvrList().GetIp()), ShouldEqual, 2) // cluster1

// 			reqCmd.AgentIp = proto.Int32(56352421)
// 			ack, _ = safeSyncByAgentCmd(ctx, reqCmd)
// 			So(len(ack.GetL5SvrList().GetIp()), ShouldEqual, 3) // cluster2
// 		})
// 	})
// }

// // 测试别名sid可以正常获取数据
// func TestCl5AliasSyncCmd(t *testing.T) {
// 	reqCmd := &l5.Cl5SyncByAgentCmd{
// 		AgentIp:  proto.Int32(11111),
// 		SyncFlow: proto.Int32(22222),
// 	}
// 	testFunc := func(namespace string) {
// 		Convey(fmt.Sprintf("%s, alias sid, discover ok", namespace), t, func() {
// 			service := &api.Service{
// 				Name:      utils.NewStringValue("my-name-for-alias"),
// 				Namespace: utils.NewStringValue(namespace),
// 				Owners:    utils.NewStringValue("aa"),
// 			}
// 			resp := server.CreateService(defaultCtx, service)
// 			So(respSuccess(resp), ShouldEqual, true)
// 			serviceResp := resp.Service
// 			defer cleanServiceName(serviceResp.Name.Value, serviceResp.Namespace.Value)

// 			resp = createCommonAlias(serviceResp, "", serviceResp.Namespace.GetValue(), api.AliasType_CL5SID)
// 			So(respSuccess(resp), ShouldEqual, true)
// 			defer cleanServiceName(resp.Alias.Alias.Value, serviceResp.Namespace.Value)
// 			modID, cmdID := parseStr2Sid(resp.Alias.Alias.Value)
// 			reqCmd.OptList = &l5.Cl5OptList{
// 				Opt: []*l5.Cl5OptObj{{ModId: proto.Int32(int32(modID)), CmdId: proto.Int32(int32(cmdID))}},
// 			}

// 			count := 5
// 			for i := 0; i < count; i++ {
// 				_, instanceResp := createCommonInstance(t, serviceResp, i)
// 				defer cleanInstance(instanceResp.GetId().GetValue())
// 			}
// 			time.Sleep(updateCacheInterval)

// 			ack, _ := server.SyncByAgentCmd(defaultCtx, reqCmd)
// 			So(ack.GetAgentIp(), ShouldEqual, reqCmd.GetAgentIp())
// 			So(ack.GetSyncFlow(), ShouldEqual, reqCmd.GetSyncFlow()+1)
// 			So(len(ack.GetServList().GetServ()), ShouldEqual, count)

// 		})
// 	}

// 	namespaces := []string{"default", "Polaris"}
// 	for _, entry := range namespaces {
// 		testFunc(entry)
// 	}
// }
