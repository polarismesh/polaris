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
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/polarismesh/polaris/common/model"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func createTestRateLimit(id string, createId bool) *model.RateLimit {

	if strings.Compare(id, "") == 0 && createId {
		id = uuid.NewString()
	}

	return &model.RateLimit{
		ID:         id,
		ServiceID:  RandStringRunes(10),
		Labels:     RandStringRunes(20),
		Rule:       RandStringRunes(20),
		Revision:   RandStringRunes(30),
		Valid:      false,
		CreateTime: time.Now(),
		ModifyTime: time.Now(),
		EnableTime: time.Now(),
	}
}

func Test_rateLimitStore_CreateRateLimit(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_ratelimit", func(t *testing.T, handler BoltHandler) {

		testVal := createTestRateLimit("", true)

		store := &rateLimitStore{
			handler: handler,
		}
		if err := store.CreateRateLimit(testVal); (err != nil) != false {
			t.Fatalf("rateLimitStore.CreateRateLimit() error = %v, wantErr %v", err, false)
		}

		saveVal, err := store.GetRateLimitWithID(testVal.ID)
		if err != nil {
			t.Fatal(err)
		}

		tN := time.Now()
		tVal := testVal
		tVal.ModifyTime = tN
		tVal.CreateTime = tN
		tVal.EnableTime = tN
		saveVal.ModifyTime = tN
		saveVal.CreateTime = tN
		saveVal.EnableTime = tN
		if !reflect.DeepEqual(saveVal, tVal) {
			t.FailNow()
		}
	})
}

func Test_rateLimitStore_CreateRateLimitWithBadParam(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_ratelimit", func(t *testing.T, handler BoltHandler) {

		testVal := createTestRateLimit("", false)

		store := &rateLimitStore{
			handler: handler,
		}
		if err := store.CreateRateLimit(testVal); err == nil {
			t.Fatalf("rateLimitStore.CreateRateLimit() need to return error")
		} else {
			if strings.Compare(ErrBadParam.Error(), err.Error()) != 0 {
				t.Fatalf("error msg must : %s", ErrBadParam.Error())
			}
		}
	})
}

func Test_rateLimitStore_UpdateRateLimit(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_ratelimit", func(t *testing.T, handler BoltHandler) {
		r := &rateLimitStore{
			handler: handler,
		}

		testVal := createTestRateLimit("", true)

		if err := r.CreateRateLimit(testVal); err != nil {
			t.Fatalf("rateLimitStore.CreateRateLimit() error = %v", err)
		}

		newVal := *testVal
		newVal.Revision = RandStringRunes(15)

		if err := r.UpdateRateLimit(&newVal); err != nil {
			t.Errorf("rateLimitStore.UpdateRateLimit() error = %v", err)
		}

		saveVal, err := r.GetRateLimitWithID(newVal.ID)
		if err != nil {
			t.Fatal(err)
		}

		tN := time.Now()

		newVal.ModifyTime = tN
		newVal.CreateTime = tN
		newVal.EnableTime = tN
		saveVal.ModifyTime = tN
		saveVal.CreateTime = tN
		saveVal.EnableTime = tN

		if !reflect.DeepEqual(saveVal, &newVal) {
			t.FailNow()
		}
	})

}

func Test_rateLimitStore_DeleteRateLimit(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_ratelimit", func(t *testing.T, handler BoltHandler) {
		r := &rateLimitStore{
			handler: handler,
		}

		testVal := createTestRateLimit("", true)

		if err := r.CreateRateLimit(testVal); err != nil {
			t.Fatalf("rateLimitStore.CreateRateLimit() error = %v", err)
		}

		if err := r.DeleteRateLimit(testVal); err != nil {
			t.Errorf("rateLimitStore.DeleteRateLimit() error = %v", err)
		}

		saveVal, err := r.GetRateLimitWithID(testVal.ID)
		if err != nil {
			t.Fatal(err)
		}

		assert.Nil(t, saveVal, "delete failed")
	})
}

func Test_rateLimitStore_GetExtendRateLimits(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_ratelimit", func(t *testing.T, handler BoltHandler) {
		r := &rateLimitStore{
			handler: handler,
		}

		svcS := &serviceStore{
			handler: handler,
		}

		vals := make([]*model.RateLimit, 0)

		Cluster_2 := make([]*model.RateLimit, 0)
		Cluster_3 := make([]*model.RateLimit, 0)
		Cluster_5 := make([]*model.RateLimit, 0)

		for i := 0; i < 10; i++ {
			testVal := createTestRateLimit(uuid.NewString(), false)

			if i%2 == 0 {
				testVal.Method = "Service_Cluster_2"
				testVal.Labels = "Cluster_2@@Labels@@12345"
				Cluster_2 = append(Cluster_2, testVal)
			} else if i%3 == 0 {
				testVal.Method = "Service_Cluster_3"
				testVal.Labels = "Cluster_3@@Labels@@67890"
				Cluster_3 = append(Cluster_3, testVal)
			} else if i%5 == 0 {
				testVal.Method = "Service_Cluster_5"
				testVal.Labels = "Cluster_5@@Labels@@abcde"
				Cluster_5 = append(Cluster_5, testVal)
			}

			//  create service
			svcS.AddService(&model.Service{
				ID:        testVal.ServiceID,
				Name:      testVal.ServiceID,
				Namespace: testVal.ServiceID,
				Owner:     "Polaris",
				Token:     testVal.Revision,
			})

			vals = append(vals, testVal)
			if err := r.CreateRateLimit(testVal); err != nil {
				t.Fatalf("rateLimitStore.CreateRateLimit() error = %v", err)
			}
		}

		// Test 1
		got, got1, err := r.GetExtendRateLimits(map[string]string{
			"method": "Cluster_2",
		}, 0, 10)
		if err != nil {
			t.Errorf("rateLimitStore.GetExtendRateLimits() error = %v", err)
			return
		}
		if int(got) != len(Cluster_2) {
			t.Fatalf("expect result cnt : %d, actual cnt : %d", len(Cluster_2), got)
		}

		got1Limits := make([]*model.RateLimit, 0)
		for i := range got1 {
			got1Limits = append(got1Limits, got1[i].RateLimit)
		}

		tN := time.Now()

		sort.Slice(got1, func(i, j int) bool {
			got1Limits[i].CreateTime = tN
			got1Limits[i].ModifyTime = tN
			got1Limits[i].EnableTime = tN
			got1Limits[j].CreateTime = tN
			got1Limits[j].ModifyTime = tN
			got1Limits[j].EnableTime = tN
			return strings.Compare(got1Limits[i].ID, got1Limits[j].ID) < 0
		})
		sort.Slice(Cluster_2, func(i, j int) bool {
			Cluster_2[i].CreateTime = tN
			Cluster_2[i].ModifyTime = tN
			Cluster_2[i].EnableTime = tN
			Cluster_2[j].CreateTime = tN
			Cluster_2[j].ModifyTime = tN
			Cluster_2[j].EnableTime = tN
			return strings.Compare(Cluster_2[i].ID, Cluster_2[j].ID) < 0
		})
		assert.ElementsMatch(t, got1Limits, Cluster_2, "result must be equal")

		// Test 2
		got, got1, err = r.GetExtendRateLimits(map[string]string{
			strings.ToLower("Labels"): "Cluster_3",
		}, 0, 10)
		if err != nil {
			t.Errorf("rateLimitStore.GetExtendRateLimits() error = %v", err)
			return
		}
		if int(got) != len(Cluster_3) {
			t.Fatalf("expect result cnt : %d, actual cnt : %d", len(Cluster_3), got)
		}

		got1Limits = make([]*model.RateLimit, 0)
		for i := range got1 {
			got1Limits = append(got1Limits, got1[i].RateLimit)
		}

		sort.Slice(got1, func(i, j int) bool {
			got1Limits[i].CreateTime = tN
			got1Limits[i].ModifyTime = tN
			got1Limits[i].EnableTime = tN
			got1Limits[j].CreateTime = tN
			got1Limits[j].ModifyTime = tN
			got1Limits[j].EnableTime = tN
			return strings.Compare(got1Limits[i].ID, got1Limits[j].ID) < 0
		})
		sort.Slice(Cluster_3, func(i, j int) bool {
			Cluster_3[i].CreateTime = tN
			Cluster_3[i].ModifyTime = tN
			Cluster_3[i].EnableTime = tN
			Cluster_3[j].CreateTime = tN
			Cluster_3[j].ModifyTime = tN
			Cluster_3[j].EnableTime = tN
			return strings.Compare(Cluster_3[i].ID, Cluster_3[j].ID) < 0
		})
		assert.ElementsMatch(t, got1Limits, Cluster_3, "result must be equal")
	})

}

func Test_rateLimitStore_GetExtendRateLimitsDisable(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_ratelimit_disable", func(t *testing.T, handler BoltHandler) {
		r := &rateLimitStore{
			handler: handler,
		}

		svcS := &serviceStore{
			handler: handler,
		}

		vals := make([]*model.RateLimit, 0)

		for i := 0; i < 10; i++ {
			testVal := createTestRateLimit(uuid.NewString(), false)
			if i%2 == 0 {
				testVal.Disable = true
			} else {
				testVal.Disable = false
			}
			//  create service
			svcS.AddService(&model.Service{
				ID:        testVal.ServiceID,
				Name:      testVal.ServiceID,
				Namespace: testVal.ServiceID,
				Owner:     "Polaris",
				Token:     testVal.Revision,
			})

			vals = append(vals, testVal)
			if err := r.CreateRateLimit(testVal); err != nil {
				t.Fatalf("rateLimitStore.CreateRateLimit() error = %v", err)
			}
		}

		// Test 1
		_, ret, err := r.GetExtendRateLimits(map[string]string{
			"disable": "true",
		}, 0, 100)
		if err != nil {
			t.Errorf("rateLimitStore.GetExtendRateLimits() error = %v", err)
			return
		}
		assert.Equal(t, int32(5), int32(len(ret)))

		// Test 1
		_, ret, err = r.GetExtendRateLimits(map[string]string{
			"disable": "false",
		}, 0, 100)
		if err != nil {
			t.Errorf("rateLimitStore.GetExtendRateLimits() error = %v", err)
			return
		}
		assert.Equal(t, int32(5), int32(len(ret)))
	})
}

func Test_rateLimitStore_GetRateLimitWithID(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_ratelimit", func(t *testing.T, handler BoltHandler) {
		r := &rateLimitStore{
			handler: handler,
		}

		testVal := createTestRateLimit("", true)

		if err := r.CreateRateLimit(testVal); err != nil {
			t.Fatalf("rateLimitStore.CreateRateLimit() error = %v", err)
		}

		saveVal, err := r.GetRateLimitWithID(testVal.ID)
		if err != nil {
			t.Errorf("rateLimitStore.GetRateLimitWithID() error = %v", err)
		}

		tN := time.Now()
		testVal.CreateTime = tN
		testVal.ModifyTime = tN
		testVal.EnableTime = tN
		saveVal.CreateTime = tN
		saveVal.ModifyTime = tN
		saveVal.EnableTime = tN

		assert.Equal(t, testVal, saveVal, "not equal")
	})
}

func Test_rateLimitStore_GetRateLimitsForCache(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_ratelimit", func(t *testing.T, handler BoltHandler) {
		r := &rateLimitStore{
			handler: handler,
		}

		vals := make([]*model.RateLimit, 0, 10)

		tN := time.Now().Add(time.Duration(-30) * time.Minute)

		for i := 0; i < 10; i++ {
			testVal := createTestRateLimit(uuid.NewString(), false)
			testVal.Valid = true
			testVal.ModifyTime = tN.Add(time.Duration(i+20) * time.Minute)
			vals = append(vals, testVal)
			if err := r.createRateLimit(testVal); err != nil {
				t.Fatalf("rateLimitStore.CreateRateLimit() error = %v", err)
			}
		}

		testT_1 := time.Now().Add(time.Duration(-5) * time.Minute)

		limits, err := r.GetRateLimitsForCache(testT_1, false)
		if err != nil {
			t.Fatal(err)
		}

		expectList := make([]*model.RateLimit, 0)

		for i := range vals {
			item := vals[i]
			if !item.ModifyTime.Before(testT_1) {
				expectList = append(expectList, item)
			}
		}

		assert.Equal(t, len(expectList), len(limits), "len(limits) not equal len(expectList)")

		for i := range expectList {
			expectList[i].CreateTime = testT_1
			expectList[i].ModifyTime = testT_1
			expectList[i].EnableTime = testT_1
			limits[i].CreateTime = testT_1
			limits[i].ModifyTime = testT_1
			limits[i].EnableTime = testT_1
		}

		assert.ElementsMatch(t, limits, expectList, "not equal")

	})

}
