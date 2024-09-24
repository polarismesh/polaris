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

package boltdb

import (
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/wrappers"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	"github.com/polarismesh/polaris/common/eventhub"
	"github.com/polarismesh/polaris/common/model"
	commontime "github.com/polarismesh/polaris/common/time"
)

func setup() {
	eventhub.InitEventHub()
}

func teardown() {
}

func TestAdminStore_BatchCleanDeletedClients(t *testing.T) {
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		handler.Close()
		_ = os.RemoveAll("./table.bolt")
	}()

	store := &adminStore{handler: handler}
	cStore := &clientStore{handler: handler}

	mockClients := createMockClients(5)
	err = cStore.BatchAddClients(mockClients)
	if err != nil {
		t.Fatal(err)
	}

	toDeleteClients := []string{
		mockClients[0].Proto().Id.GetValue(),
		mockClients[1].Proto().Id.GetValue(),
		mockClients[2].Proto().Id.GetValue(),
	}
	err = cStore.BatchDeleteClients(toDeleteClients)
	if err != nil {
		t.Fatal(err)
	}

	count, err := store.BatchCleanDeletedClients(0, 2)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("count not match, expect cnt=%d, actual cnt=%d", 2, count)
	}

	remainClients, err := cStore.GetMoreClients(time.Time{}, false)
	if err != nil {
		t.Fatal(err)
	}
	validCount := 0
	invalidCount := 0
	for _, v := range remainClients {
		if v.Valid() {
			validCount++
		} else {
			invalidCount++
		}
	}
	if validCount != 2 {
		t.Fatalf("count not match, expect cnt=%d, actual cnt=%d", 2, validCount)
	}
	if invalidCount != 1 {
		t.Fatalf("count not match, expect cnt=%d, actual cnt=%d", 1, invalidCount)
	}

}

func TestAdminStore_BatchCleanDeletedInstances(t *testing.T) {
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		handler.Close()
		_ = os.RemoveAll("./table.bolt")
	}()

	store := &adminStore{handler: handler}
	sStore := &serviceStore{handler: handler}
	insStore := &instanceStore{handler: handler}

	svcId := "svcid1"
	sStore.AddService(&model.Service{
		ID:        svcId,
		Name:      svcId,
		Namespace: svcId,
		Token:     svcId,
		Owner:     svcId,
		Valid:     true,
	})

	for i := 0; i < insCount; i++ {
		nowt := commontime.Time2String(time.Now())
		err := insStore.AddInstance(&model.Instance{
			Proto: &apiservice.Instance{
				Id:                &wrappers.StringValue{Value: "insid" + strconv.Itoa(i)},
				Host:              &wrappers.StringValue{Value: "1.1.1." + strconv.Itoa(i)},
				Port:              &wrappers.UInt32Value{Value: uint32(i + 1)},
				Protocol:          &wrappers.StringValue{Value: "grpc"},
				Weight:            &wrappers.UInt32Value{Value: uint32(i + 1)},
				EnableHealthCheck: &wrappers.BoolValue{Value: true},
				Healthy:           &wrappers.BoolValue{Value: true},
				Isolate:           &wrappers.BoolValue{Value: true},
				Metadata: map[string]string{
					"insk1": "insv1",
					"insk2": "insv2",
				},
				Ctime:    &wrappers.StringValue{Value: nowt},
				Mtime:    &wrappers.StringValue{Value: nowt},
				Revision: &wrappers.StringValue{Value: "revision" + strconv.Itoa(i)},
			},
			ServiceID:         svcId,
			ServicePlatformID: "svcPlatId1",
			Valid:             true,
			ModifyTime:        time.Now(),
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	toDeleteInstances := []interface{}{"insid1", "insid2", "insid3"}
	err = insStore.BatchDeleteInstances(toDeleteInstances)
	if err != nil {
		t.Fatal(err)
	}

	count, err := store.BatchCleanDeletedInstances(0, 2)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("count not match, expect cnt=%d, actual cnt=%d", 2, count)
	}

	count, err = insStore.GetInstancesCount()
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("count not match, expect cnt=%d, actual cnt=%d", 2, count)
	}

}

func TestAdminStore_StartLeaderElection(t *testing.T) {
	key := "TestElectKey"
	mstore := &adminStore{handler: nil, leMap: make(map[string]bool)}
	isLeader := mstore.IsLeader(key)
	if isLeader {
		t.Error("expect follower state")
	}

	mstore.StartLeaderElection(key)
	isLeader = mstore.IsLeader(key)
	if !isLeader {
		t.Error("expect leader state")
	}
}

func TestAdminStore_ReleaseLeaderElection(t *testing.T) {
	key := "TestElectKey"
	mstore := &adminStore{handler: nil, leMap: make(map[string]bool)}
	mstore.StartLeaderElection(key)
	mstore.ReleaseLeaderElection(key)
	isLeader := mstore.IsLeader(key)
	if isLeader {
		t.Error("expect follower state")
	}
}

func TestAdminStore_ListLeaderElections(t *testing.T) {
	key := "TestElectKey"
	mstore := &adminStore{handler: nil, leMap: make(map[string]bool)}
	mstore.StartLeaderElection(key)

	out, err := mstore.ListLeaderElections()
	if err != nil {
		t.Errorf("should not err: %v", err)
	}

	if len(out) != 1 {
		t.Error("expect one leader election")
	}

	if out[0].ElectKey != key {
		t.Errorf("expect key: %s, actual key: %s", key, out[0].ElectKey)
	}

}

func TestAdminStore_getUnHealthyInstancesBefore(t *testing.T) {
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		handler.Close()
		_ = os.RemoveAll("./table.bolt")
	}()

	store := &adminStore{handler: handler}
	sStore := &serviceStore{handler: handler}
	insStore := &instanceStore{handler: handler}

	svcId := "svcid1"
	sStore.AddService(&model.Service{
		ID:        svcId,
		Name:      svcId,
		Namespace: svcId,
		Token:     svcId,
		Owner:     svcId,
		Valid:     true,
	})

	mtime := time.Date(2023, 3, 4, 11, 0, 0, 0, time.Local)
	for i := 0; i < insCount; i++ {
		nowt := commontime.Time2String(mtime)
		err := insStore.AddInstance(&model.Instance{
			Proto: &apiservice.Instance{
				Id:                &wrappers.StringValue{Value: "insid" + strconv.Itoa(i)},
				Host:              &wrappers.StringValue{Value: "1.1.1." + strconv.Itoa(i)},
				Port:              &wrappers.UInt32Value{Value: uint32(i + 1)},
				Protocol:          &wrappers.StringValue{Value: "grpc"},
				Weight:            &wrappers.UInt32Value{Value: uint32(i + 1)},
				EnableHealthCheck: &wrappers.BoolValue{Value: true},
				Healthy:           &wrappers.BoolValue{Value: true},
				Isolate:           &wrappers.BoolValue{Value: true},
				Metadata: map[string]string{
					"insk1": "insv1",
					"insk2": "insv2",
				},
				Ctime:    &wrappers.StringValue{Value: nowt},
				Mtime:    &wrappers.StringValue{Value: nowt},
				Revision: &wrappers.StringValue{Value: "revision" + strconv.Itoa(i)},
			},
			ServiceID:         svcId,
			ServicePlatformID: "svcPlatId1",
			Valid:             true,
			ModifyTime:        time.Now(),
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	toUnHealthyInstances := []interface{}{"insid1", "insid2", "insid3"}
	err = insStore.BatchSetInstanceHealthStatus(toUnHealthyInstances, 0, "revision-11")
	if err != nil {
		t.Fatal(err)
	}

	beforeTime := time.Date(2023, 3, 4, 11, 1, 0, 0, time.Local)
	ids, err := store.getUnHealthyInstancesBefore(beforeTime, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 {
		t.Fatalf("count not match, expect cnt=%d, actual cnt=%d", 2, len(ids))
	}
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}
