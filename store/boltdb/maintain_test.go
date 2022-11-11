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
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	commontime "github.com/polarismesh/polaris/common/time"
)

func TestMaintainStore_BatchCleanDeletedInstances(t *testing.T) {
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		handler.Close()
		_ = os.RemoveAll("./table.bolt")
	}()

	store := &maintainStore{handler: handler}
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
			Proto: &api.Instance{
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

	count, err := store.BatchCleanDeletedInstances(2)
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
