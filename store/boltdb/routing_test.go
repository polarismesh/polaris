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
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/polarismesh/polaris/common/model"
)

const (
	routeCount = 5
)

func TestRoutingStore_CreateRoutingConfig(t *testing.T) {
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if err != nil {
		t.Fatal(err)
	}

	defer handler.Close()

	rStore := &routingStore{handler: handler}

	for i := 0; i < routeCount; i++ {
		rStore.CreateRoutingConfig(&model.RoutingConfig{
			ID:         "testid" + strconv.Itoa(i),
			InBounds:   "v1" + strconv.Itoa(i),
			OutBounds:  "v2" + strconv.Itoa(i),
			Revision:   "revision" + strconv.Itoa(i),
			Valid:      true,
			CreateTime: time.Now(),
			ModifyTime: time.Now(),
		})
	}

	if err != nil {
		t.Fatal(err)
	}
}

func TestRoutingStore_GetRoutingConfigs(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "RoutingConfig", func(t *testing.T, handler BoltHandler) {

		rStore := &routingStore{handler: handler}
		sStore := &serviceStore{handler: handler}

		for i := 0; i < routeCount; i++ {
			err := sStore.AddService(&model.Service{
				ID:        "testid" + strconv.Itoa(i),
				Name:      "testid" + strconv.Itoa(i),
				Namespace: "testid" + strconv.Itoa(i),
			})
			assert.NoError(t, err)

			err = rStore.CreateRoutingConfig(&model.RoutingConfig{
				ID:         "testid" + strconv.Itoa(i),
				InBounds:   "v1" + strconv.Itoa(i),
				OutBounds:  "v2" + strconv.Itoa(i),
				Revision:   "revision" + strconv.Itoa(i),
				Valid:      true,
				CreateTime: time.Now(),
				ModifyTime: time.Now(),
			})
			assert.NoError(t, err)
		}

		totalCount, rs, err := rStore.GetRoutingConfigs(nil, 0, 20)
		if err != nil {
			t.Fatal(err)
		}
		if totalCount != routeCount {
			t.Fatal(fmt.Sprintf("routing total count not match, expect %d, got %d", routeCount, totalCount))
		}
		if len(rs) != routeCount {
			t.Fatal(fmt.Sprintf("routing count not match, expect %d, got %d", routeCount, len(rs)))
		}
		for _, r := range rs {
			fmt.Printf("routing conf is %+v\n", r.Config)
		}
	})
}

func TestRoutingStore_UpdateRoutingConfig(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "TestRoutingStore_GetRoutingConfigsForCache", func(t *testing.T, handler BoltHandler) {

		rStore := &routingStore{handler: handler}
		sStore := &serviceStore{handler: handler}

		for i := 0; i < routeCount; i++ {
			err := sStore.AddService(&model.Service{
				ID:        "testid" + strconv.Itoa(i),
				Name:      "testid" + strconv.Itoa(i),
				Namespace: "testid" + strconv.Itoa(i),
			})
			assert.NoError(t, err)

			conf := &model.RoutingConfig{
				ID:        "testid" + strconv.Itoa(i),
				InBounds:  "vv1" + strconv.Itoa(i),
				OutBounds: "vv2" + strconv.Itoa(i),
				Revision:  "revi" + strconv.Itoa(i),
			}

			err = rStore.CreateRoutingConfig(conf)
			assert.NoError(t, err)

			err = rStore.UpdateRoutingConfig(conf)
			assert.NoError(t, err)
		}

		// check update result
		totalCount, rs, err := rStore.GetRoutingConfigs(nil, 0, 20)
		if err != nil {
			t.Fatal(err)
		}
		if totalCount != routeCount {
			t.Fatal(fmt.Sprintf("routing total count not match, expect %d, got %d", routeCount, totalCount))
		}
		if len(rs) != routeCount {
			t.Fatal(fmt.Sprintf("routing count not match, expect %d, got %d", routeCount, len(rs)))
		}
		for _, r := range rs {
			fmt.Printf("routing conf is %+v\n", r.Config)
		}
	})
}

func TestRoutingStore_GetRoutingConfigsForCache(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "TestRoutingStore_GetRoutingConfigsForCache",
		func(t *testing.T, handler BoltHandler) {
			rStore := &routingStore{handler: handler}
			sStore := &serviceStore{handler: handler}

			for i := 0; i < routeCount; i++ {
				err := sStore.AddService(&model.Service{
					ID:        "testid" + strconv.Itoa(i),
					Name:      "testid" + strconv.Itoa(i),
					Namespace: "testid" + strconv.Itoa(i),
				})
				assert.NoError(t, err)

				err = rStore.CreateRoutingConfig(&model.RoutingConfig{
					ID:         "testid" + strconv.Itoa(i),
					InBounds:   "v1" + strconv.Itoa(i),
					OutBounds:  "v2" + strconv.Itoa(i),
					Revision:   "revision" + strconv.Itoa(i),
					Valid:      true,
					CreateTime: time.Now(),
					ModifyTime: time.Now(),
				})
				assert.NoError(t, err)
			}

			// get create modify time
			totalCount, rs, err := rStore.GetRoutingConfigs(nil, 0, 20)
			if err != nil {
				t.Fatal(err)
			}
			if totalCount != routeCount {
				t.Fatal(fmt.Sprintf("routing total count not match, expect %d, got %d", routeCount, totalCount))
			}
			if len(rs) != routeCount {
				t.Fatal(fmt.Sprintf("routing count not match, expect %d, got %d", routeCount, len(rs)))
			}

			rss, err := rStore.GetRoutingConfigsForCache(rs[2].Config.ModifyTime, false)
			if err != nil {
				t.Fatal(err)
			}
			if len(rss) != routeCount-2 {
				t.Fatal(fmt.Sprintf("routing config count mismatch, except %d, got %d", routeCount-2, len(rss)))
			}
		})
}

func TestRoutingStore_GetRoutingConfigWithService(t *testing.T) {

	// find service
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if err != nil {
		t.Fatal(err)
	}

	defer handler.Close()

	sStore := &serviceStore{handler: handler}
	err = sStore.AddService(&model.Service{
		ID:        "testid3",
		Name:      "test-svc-name",
		Namespace: "test-svc-namespace",
		Owner:     "test-owner",
		Token:     "test-token",
	})
	if err != nil {
		t.Fatal(err)
	}

	rStore := &routingStore{handler: handler}
	rc, err := rStore.GetRoutingConfigWithService("test-svc-name", "test-svc-namespace")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("get routing config with service %+v", rc)
}

func TestRoutingStore_GetRoutingConfigWithID(t *testing.T) {
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if err != nil {
		t.Fatal(err)
	}

	defer handler.Close()

	rStore := &routingStore{handler: handler}

	rc, err := rStore.GetRoutingConfigWithID("testid0")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("get routing conf %+v\n", rc)
}

func TestRoutingStore_DeleteRoutingConfig(t *testing.T) {
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if err != nil {
		t.Fatal(err)
	}

	defer handler.Close()

	rStore := &routingStore{handler: handler}
	for i := 0; i < routeCount; i++ {
		err := rStore.DeleteRoutingConfig("testid" + strconv.Itoa(i))
		assert.Nil(t, err, "err must nil")
	}
}
