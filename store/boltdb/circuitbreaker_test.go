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
	"encoding/json"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
)

func RandCircuitbreakerStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func createTestCircuitbreaker(id string, createId bool) *model.CircuitBreaker {

	if strings.Compare(id, "") == 0 && createId {
		id = utils.NewUUID()
	}

	str := RandCircuitbreakerStringRunes(5)

	return &model.CircuitBreaker{
		ID:         id,
		Version:    "circuitbreaker_Version_" + str,
		Name:       "circuitbreaker_test_Name_" + str,
		Namespace:  "circuitbreaker_test_Namespace_" + str,
		Business:   "circuitbreaker_test_Business_" + str,
		Department: "circuitbreaker_test_Department_" + str,
		Comment:    "circuitbreaker_test_Comment_" + str,
		Inbounds:   "circuitbreaker_test_Inbounds_" + str,
		Outbounds:  "circuitbreaker_test_Outbounds_" + str,
		Token:      "circuitbreaker_test_Token_" + str,
		Owner:      "circuitbreaker_test_Owner_" + str,
		Revision:   "circuitbreaker_test_Revision_" + str,
		Valid:      false,
		CreateTime: time.Now(),
		ModifyTime: time.Now(),
	}
}

func createTestCircuitbreakerRelation(id string, createId bool) *model.CircuitBreakerRelation {

	if strings.Compare(id, "") == 0 && createId {
		id = uuid.NewString()
	}

	str := RandCircuitbreakerStringRunes(5)

	return &model.CircuitBreakerRelation{
		ServiceID:   "ServiceID_" + str,
		RuleID:      id,
		RuleVersion: "RuleVersion_" + str,
		Valid:       false,
		CreateTime:  time.Now(),
		ModifyTime:  time.Now(),
	}
}

func createTestService(id, name, namespace string, create bool) *model.Service {

	if strings.Compare(id, "") == 0 && create {
		id = uuid.NewString()
	}

	return &model.Service{
		ID:        RandStringRunes(15),
		Name:      name,
		Namespace: namespace,
		Business:  RandStringRunes(5),
		Ports:     "",
		Meta: map[string]string{
			"meta_test": "meta_test",
		},
		Comment:     "Comment_" + RandStringRunes(6),
		Department:  "Department_" + RandStringRunes(6),
		CmdbMod1:    "CmdbMod1_" + RandStringRunes(6),
		CmdbMod2:    "CmdbMod2_" + RandStringRunes(6),
		CmdbMod3:    "CmdbMod3_" + RandStringRunes(6),
		Token:       RandStringRunes(15),
		Owner:       RandStringRunes(15),
		Revision:    "Revision_" + RandStringRunes(6),
		Reference:   "Reference_" + RandStringRunes(6),
		ReferFilter: "ReferFilter_" + RandStringRunes(6),
		PlatformID:  "PlatformID_" + RandStringRunes(6),
		Valid:       false,
		CreateTime:  time.Time{},
		ModifyTime:  time.Time{},
	}
}

func Test_circuitBreakerStore_CreateCircuitBreaker(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_circuitbreaker", func(t *testing.T, handler BoltHandler) {
		type fields struct {
			handler      BoltHandler
			ruleLock     *sync.RWMutex
			relationLock *sync.RWMutex
		}
		type args struct {
			cb *model.CircuitBreaker
		}
		tests := []struct {
			name     string
			fields   fields
			args     args
			wantErr  bool
			checkErr func(err error) bool
		}{
			{
				name: "",
				fields: fields{
					handler:      handler,
					ruleLock:     &sync.RWMutex{},
					relationLock: &sync.RWMutex{},
				},
				args: args{
					cb: createTestCircuitbreaker("", true),
				},
				wantErr: false,
				checkErr: func(err error) bool {
					return true
				},
			},
			{
				name: "",
				fields: fields{
					handler:      handler,
					ruleLock:     &sync.RWMutex{},
					relationLock: &sync.RWMutex{},
				},
				args: args{
					cb: createTestCircuitbreaker("", false),
				},
				wantErr: false,
				checkErr: func(err error) bool {
					return true
				},
			},
			{
				name: "",
				fields: fields{
					handler:      handler,
					ruleLock:     &sync.RWMutex{},
					relationLock: &sync.RWMutex{},
				},
				args: args{
					cb: createTestCircuitbreaker("", true),
				},
				wantErr: false,
				checkErr: func(err error) bool {
					return true
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				c := &circuitBreakerStore{
					handler: tt.fields.handler,
				}
				err := c.CreateCircuitBreaker(tt.args.cb)
				if (err != nil) != tt.wantErr {
					t.Errorf("circuitBreakerStore.CreateCircuitBreaker() error = %v, wantErr %v", err, tt.wantErr)
				}
				if err != nil {
					if !tt.checkErr(err) {
						t.Errorf("circuitBreakerStore.CreateCircuitBreaker() check expect error failed")
					}
					return
				}

				cb, err := c.GetCircuitBreaker(tt.args.cb.ID, tt.args.cb.Version)
				if err != nil {
					t.Fatal(err)
				}

				tN := time.Now()
				cb.CreateTime = tN
				cb.ModifyTime = tN

				tt.args.cb.CreateTime = tN
				tt.args.cb.ModifyTime = tN

				if cb == nil {
					t.Errorf("circuitBreakerStore.CreateCircuitBreaker() not effect")
				}

				if !reflect.DeepEqual(cb, tt.args.cb) {
					t.Errorf("circuitBreakerStore.CreateCircuitBreaker() expcet : %#v, acutal : %#v", tt.args.cb, cb)
				}
			})
		}
	})
}

func Test_circuitBreakerStore_TagCircuitBreaker(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_circuitbreaker", func(t *testing.T, handler BoltHandler) {
		type fields struct {
			handler      BoltHandler
			ruleLock     *sync.RWMutex
			relationLock *sync.RWMutex
		}
		type args struct {
			cb *model.CircuitBreaker
		}
		tests := []struct {
			name    string
			fields  fields
			args    args
			wantErr bool
		}{
			{
				name: "",
				fields: fields{
					handler:      handler,
					ruleLock:     &sync.RWMutex{},
					relationLock: &sync.RWMutex{},
				},
				args: args{
					cb: createTestCircuitbreaker("", true),
				},
				wantErr: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				c := &circuitBreakerStore{
					handler: tt.fields.handler,
				}

				old := createTestCircuitbreaker(tt.args.cb.ID, false)
				old.Version = VersionForMaster

				if err := c.CreateCircuitBreaker(old); err != nil {
					t.Error(err)
				}

				if err := c.TagCircuitBreaker(tt.args.cb); (err != nil) != tt.wantErr {
					t.Errorf("circuitBreakerStore.TagCircuitBreaker() error = %v, wantErr %v", err, tt.wantErr)
				}

				cb, err := c.GetCircuitBreaker(tt.args.cb.ID, tt.args.cb.Version)
				if err != nil {
					t.Fatal(err)
				}

				tN := time.Now()
				cb.CreateTime = tN
				cb.ModifyTime = tN

				tt.args.cb.CreateTime = tN
				tt.args.cb.ModifyTime = tN

				if cb == nil {
					t.Errorf("circuitBreakerStore.CreateCircuitBreaker() not effect")
				}

				if !reflect.DeepEqual(cb, tt.args.cb) {
					t.Errorf("circuitBreakerStore.CreateCircuitBreaker() expcet : %#v, acutal : %#v", tt.args.cb, cb)
				}
			})
		}
	})
}

func Test_circuitBreakerStore_ReleaseCircuitBreaker(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_circuitbreaker", func(t *testing.T, handler BoltHandler) {
		type fields struct {
			handler      BoltHandler
			ruleLock     *sync.RWMutex
			relationLock *sync.RWMutex
		}
		type args struct {
			cbr *model.CircuitBreakerRelation
		}
		tests := []struct {
			name    string
			fields  fields
			args    args
			wantErr bool
		}{
			{
				name: "",
				fields: fields{
					handler:      handler,
					ruleLock:     &sync.RWMutex{},
					relationLock: &sync.RWMutex{},
				},
				args: args{
					cbr: createTestCircuitbreakerRelation("", true),
				},
				wantErr: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				c := &circuitBreakerStore{
					handler: tt.fields.handler,
				}

				cb := createTestCircuitbreaker(tt.args.cbr.RuleID, false)
				cb.Version = tt.args.cbr.RuleVersion

				// 保存一个熔断规则
				if err := c.CreateCircuitBreaker(cb); err != nil {
					t.Error(err)
				}

				if err := c.ReleaseCircuitBreaker(tt.args.cbr); (err != nil) != tt.wantErr {
					t.Errorf("circuitBreakerStore.ReleaseCircuitBreaker() error = %v, wantErr %v", err, tt.wantErr)
				}

				result, err := handler.LoadValues(tblCircuitBreakerRelation, []string{tt.args.cbr.ServiceID}, &model.CircuitBreakerRelation{})

				if err != nil {
					t.Fatal(err)
				}

				val := result[tt.args.cbr.ServiceID]
				if val == nil {
					t.Fatalf("circuitBreakerStore.ReleaseCircuitBreaker() not get expect relation")
				}

				tN := time.Now()

				record := val.(*model.CircuitBreakerRelation)
				record.CreateTime = tN
				record.ModifyTime = tN

				tt.args.cbr.CreateTime = tN
				tt.args.cbr.ModifyTime = tN

				if !reflect.DeepEqual(val, tt.args.cbr) {
					t.Fatalf("circuitBreakerStore.ReleaseCircuitBreaker() except : %#v, acutal : %#v", tt.args.cbr, val)
				}
			})
		}
	})
}

func Test_circuitBreakerStore_UnbindCircuitBreaker(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_circuitbreaker", func(t *testing.T, handler BoltHandler) {

		tCbR := createTestCircuitbreakerRelation("", false)

		type fields struct {
			handler      BoltHandler
			ruleLock     *sync.RWMutex
			relationLock *sync.RWMutex
		}
		type args struct {
			serviceID   string
			ruleID      string
			ruleVersion string
		}
		tests := []struct {
			name    string
			fields  fields
			args    args
			wantErr bool
		}{
			{
				name: "",
				fields: fields{
					handler:      handler,
					ruleLock:     &sync.RWMutex{},
					relationLock: &sync.RWMutex{},
				},
				args: args{
					serviceID:   tCbR.ServiceID,
					ruleID:      tCbR.RuleID,
					ruleVersion: tCbR.RuleVersion,
				},
				wantErr: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				c := &circuitBreakerStore{
					handler: tt.fields.handler,
				}

				cb := createTestCircuitbreaker(tCbR.RuleID, false)
				cb.Version = tCbR.RuleVersion
				if err := c.CreateCircuitBreaker(cb); err != nil {
					t.Fatal(err)
				}

				if err := c.ReleaseCircuitBreaker(tCbR); err != nil {
					t.Fatal(err)
				}

				if err := c.UnbindCircuitBreaker(tt.args.serviceID, tt.args.ruleID, tt.args.ruleVersion); (err != nil) != tt.wantErr {
					t.Errorf("circuitBreakerStore.UnbindCircuitBreaker() error = %v, wantErr %v", err, tt.wantErr)
				}

				result, err := handler.LoadValues(tblCircuitBreakerRelation, []string{tt.args.serviceID}, &model.CircuitBreakerRelation{})

				if err != nil {
					t.Fatal(err)
				}

				val := result[tt.args.serviceID]
				if val == nil {
					t.Fatalf("result[%s] is nil", tt.args.serviceID)
				}

				relation := val.(*model.CircuitBreakerRelation)
				if relation.Valid {
					t.Fatalf("circuitBreakerStore.ReleaseCircuitBreaker() expect delete, but still exist")
				}
			})
		}
	})
}

func Test_circuitBreakerStore_DeleteTagCircuitBreaker(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_circuitbreaker", func(t *testing.T, handler BoltHandler) {

		cb := createTestCircuitbreaker("", true)

		type fields struct {
			handler      BoltHandler
		}
		type args struct {
			id      string
			version string
		}
		tests := []struct {
			name    string
			fields  fields
			args    args
			wantErr bool
		}{
			{
				name: "",
				fields: fields{
					handler:      handler,
				},
				args: args{
					id:      cb.ID,
					version: cb.Version,
				},
				wantErr: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				c := &circuitBreakerStore{
					handler: tt.fields.handler,
				}

				if err := c.CreateCircuitBreaker(cb); err != nil {
					t.Fatal(err)
				}

				if err := c.DeleteTagCircuitBreaker(tt.args.id, tt.args.version); (err != nil) != tt.wantErr {
					t.Errorf("circuitBreakerStore.DeleteTagCircuitBreaker() error = %v, wantErr %v", err, tt.wantErr)
				}

				result, err := c.GetCircuitBreaker(tt.args.id, tt.args.version)
				if err != nil {
					if strings.Contains(err.Error(), "not found tag config") {
						return
					}
					t.Fatal(err)
				}

				if result != nil && !result.Valid {
					t.Fatal("circuitBreakerStore.DeleteTagCircuitBreaker() expect delete, but still exist")
				}
			})
		}
	})
}

func Test_circuitBreakerStore_UpdateCircuitBreaker(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_circuitbreaker", func(t *testing.T, handler BoltHandler) {
		type fields struct {
			handler      BoltHandler
			ruleLock     *sync.RWMutex
			relationLock *sync.RWMutex
		}
		type args struct {
			cb *model.CircuitBreaker
		}
		tests := []struct {
			name    string
			fields  fields
			args    args
			wantErr bool
		}{
			{
				name: "",
				fields: fields{
					handler:      handler,
					ruleLock:     &sync.RWMutex{},
					relationLock: &sync.RWMutex{},
				},
				args: args{
					cb: createTestCircuitbreaker("", true),
				},
				wantErr: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				c := &circuitBreakerStore{
					handler: tt.fields.handler,
				}

				old := createTestCircuitbreaker(tt.args.cb.ID, false)
				if err := c.CreateCircuitBreaker(old); err != nil {
					t.Fatal(err)
				}

				if err := c.UpdateCircuitBreaker(tt.args.cb); (err != nil) != tt.wantErr {
					t.Errorf("circuitBreakerStore.UpdateCircuitBreaker() error = %v, wantErr %v", err, tt.wantErr)
				}

				newCb, err := c.GetCircuitBreaker(tt.args.cb.ID, tt.args.cb.Version)
				if err != nil {
					t.Fatal(err)
				}

				t.Logf("use time.Time to deep equal : %t", reflect.DeepEqual(newCb, tt.args.cb))

				newCbJson, _ := json.Marshal(newCb)
				wantCbJson, _ := json.Marshal(tt.args.cb)

				if !reflect.DeepEqual(newCbJson, wantCbJson) {
					t.Errorf("circuitBreakerStore.UpdateCircuitBreaker() expect : %s, actual : %s", wantCbJson, newCbJson)
				}
			})
		}
	})
}

func Test_circuitBreakerStore_GetCircuitBreaker(t *testing.T) {
	type fields struct {
		handler      BoltHandler
		ruleLock     *sync.RWMutex
		relationLock *sync.RWMutex
	}
	type args struct {
		id      string
		version string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *model.CircuitBreaker
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &circuitBreakerStore{
				handler: tt.fields.handler,
			}
			got, err := c.GetCircuitBreaker(tt.args.id, tt.args.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("circuitBreakerStore.GetCircuitBreaker() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("circuitBreakerStore.GetCircuitBreaker() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_circuitBreakerStore_GetCircuitBreakerVersions(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_circuitbreaker", func(t *testing.T, handler BoltHandler) {

		c := &circuitBreakerStore{
			handler: handler,
		}

		ids := []string{"id_1", "id_2", "id_3"}

		versionOne := make([]string, 0)
		versionTwo := make([]string, 0)
		versionThree := make([]string, 0)

		for i := 0; i < 20; i++ {
			cb := createTestCircuitbreaker(ids[rand.Intn(len(ids))], false)
			if err := c.CreateCircuitBreaker(cb); err != nil {
				t.Fatal(err)
			}

			if strings.Compare(ids[0], cb.ID) == 0 {
				versionOne = append(versionOne, cb.Version)
				continue
			}

			if strings.Compare(ids[1], cb.ID) == 0 {
				versionTwo = append(versionTwo, cb.Version)
				continue
			}

			if strings.Compare(ids[2], cb.ID) == 0 {
				versionThree = append(versionThree, cb.Version)
				continue
			}
		}

		type fields struct {
			handler      BoltHandler
			ruleLock     *sync.RWMutex
			relationLock *sync.RWMutex
		}
		type args struct {
			id string
		}
		tests := []struct {
			name    string
			fields  fields
			args    args
			want    []string
			wantErr bool
		}{
			{
				name: "",
				fields: fields{
					handler:      handler,
					ruleLock:     &sync.RWMutex{},
					relationLock: &sync.RWMutex{},
				},
				args: args{
					id: ids[0],
				},
				want:    versionOne,
				wantErr: false,
			},
			{
				name: "",
				fields: fields{
					handler:      handler,
					ruleLock:     &sync.RWMutex{},
					relationLock: &sync.RWMutex{},
				},
				args: args{
					id: ids[1],
				},
				want:    versionTwo,
				wantErr: false,
			},
			{
				name: "",
				fields: fields{
					handler:      handler,
					ruleLock:     &sync.RWMutex{},
					relationLock: &sync.RWMutex{},
				},
				args: args{
					id: ids[2],
				},
				want:    versionThree,
				wantErr: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				c := &circuitBreakerStore{
					handler: tt.fields.handler,
				}
				got, err := c.GetCircuitBreakerVersions(tt.args.id)
				if (err != nil) != tt.wantErr {
					t.Errorf("circuitBreakerStore.GetCircuitBreakerVersions() error = %v, wantErr %v", err, tt.wantErr)
					return
				}

				sort.Strings(got)
				sort.Strings(tt.want)

				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("circuitBreakerStore.GetCircuitBreakerVersions() = %v, want %v", got, tt.want)
				}
			})
		}
	})
}

func Test_circuitBreakerStore_ListMasterCircuitBreakers(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_circuitbreaker", func(t *testing.T, handler BoltHandler) {

		c := &circuitBreakerStore{
			handler: handler,
		}

		ids := []string{"id_1", "id_2", "id_3"}

		idOne := make([]*model.CircuitBreakerInfo, 0)
		idTwo := make([]*model.CircuitBreakerInfo, 0)
		idThree := make([]*model.CircuitBreakerInfo, 0)

		for i := 0; i < 20; i++ {
			cb := createTestCircuitbreaker("", true)
			cb.Namespace = ids[rand.Intn(len(ids))]
			cb.Version = VersionForMaster
			if err := c.CreateCircuitBreaker(cb); err != nil {
				t.Fatal(err)
			}
			if strings.Compare(ids[0], cb.Namespace) == 0 {
				idOne = append(idOne, convertCircuitBreakerToInfo(cb))
			}
			if strings.Compare(ids[1], cb.Namespace) == 0 {
				idTwo = append(idTwo, convertCircuitBreakerToInfo(cb))
			}
			if strings.Compare(ids[2], cb.Namespace) == 0 {
				idThree = append(idThree, convertCircuitBreakerToInfo(cb))
			}
		}

		type fields struct {
			handler      BoltHandler
			ruleLock     *sync.RWMutex
			relationLock *sync.RWMutex
		}
		type args struct {
			filters map[string]string
			offset  uint32
			limit   uint32
		}
		tests := []struct {
			name    string
			fields  fields
			args    args
			want    *model.CircuitBreakerDetail
			wantErr bool
		}{
			{
				name: "",
				fields: fields{
					handler:      handler,
					ruleLock:     &sync.RWMutex{},
					relationLock: &sync.RWMutex{},
				},
				args: args{
					filters: map[string]string{
						"Namespace": ids[0],
					},
					offset: 0,
					limit:  20,
				},
				want: &model.CircuitBreakerDetail{
					Total:               uint32(len(idOne)),
					CircuitBreakerInfos: idOne,
				},
				wantErr: false,
			},
			{
				name: "",
				fields: fields{
					handler:      handler,
					ruleLock:     &sync.RWMutex{},
					relationLock: &sync.RWMutex{},
				},
				args: args{
					filters: map[string]string{
						"Namespace": ids[1],
					},
					offset: 0,
					limit:  20,
				},
				want: &model.CircuitBreakerDetail{
					Total:               uint32(len(idTwo)),
					CircuitBreakerInfos: idTwo,
				},
				wantErr: false,
			},
			{
				name: "",
				fields: fields{
					handler:      handler,
					ruleLock:     &sync.RWMutex{},
					relationLock: &sync.RWMutex{},
				},
				args: args{
					filters: map[string]string{
						"Namespace": ids[2],
					},
					offset: 0,
					limit:  20,
				},
				want: &model.CircuitBreakerDetail{
					Total:               uint32(len(idThree)),
					CircuitBreakerInfos: idThree,
				},
				wantErr: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				c := &circuitBreakerStore{
					handler: tt.fields.handler,
				}
				got, err := c.ListMasterCircuitBreakers(tt.args.filters, tt.args.offset, tt.args.limit)
				if (err != nil) != tt.wantErr {
					t.Errorf("circuitBreakerStore.ListMasterCircuitBreakers() error = %v, wantErr %v", err, tt.wantErr)
					return
				}

				tN := time.Now()

				sort.Slice(got.CircuitBreakerInfos, func(i, j int) bool {
					got.CircuitBreakerInfos[i].CircuitBreaker.CreateTime = tN
					got.CircuitBreakerInfos[i].CircuitBreaker.ModifyTime = tN

					got.CircuitBreakerInfos[j].CircuitBreaker.CreateTime = tN
					got.CircuitBreakerInfos[j].CircuitBreaker.ModifyTime = tN

					return strings.Compare(got.CircuitBreakerInfos[i].CircuitBreaker.ID, got.CircuitBreakerInfos[j].CircuitBreaker.ID) < 0
				})

				sort.Slice(tt.want.CircuitBreakerInfos, func(i, j int) bool {

					tt.want.CircuitBreakerInfos[i].CircuitBreaker.CreateTime = tN
					tt.want.CircuitBreakerInfos[i].CircuitBreaker.ModifyTime = tN

					tt.want.CircuitBreakerInfos[j].CircuitBreaker.CreateTime = tN
					tt.want.CircuitBreakerInfos[j].CircuitBreaker.ModifyTime = tN

					return strings.Compare(tt.want.CircuitBreakerInfos[i].CircuitBreaker.ID, tt.want.CircuitBreakerInfos[j].CircuitBreaker.ID) < 0
				})

				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("circuitBreakerStore.ListMasterCircuitBreakers() = %v, want %v", got, tt.want)
				}
			})
		}
	})
}

func Test_circuitBreakerStore_ListReleaseCircuitBreakers(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_circuitbreaker", func(t *testing.T, handler BoltHandler) {

		c := &circuitBreakerStore{
			handler: handler,
		}

		ids := []string{"id_1", "id_2", "id_3"}

		idOne := make([]*model.CircuitBreakerInfo, 0)
		idTwo := make([]*model.CircuitBreakerInfo, 0)
		idThree := make([]*model.CircuitBreakerInfo, 0)

		for i := 0; i < 20; i++ {
			cb := createTestCircuitbreaker("", true)
			cb.Namespace = ids[rand.Intn(len(ids))]
			if err := c.CreateCircuitBreaker(cb); err != nil {
				t.Fatal(err)
			}
			if strings.Compare(ids[0], cb.Namespace) == 0 {
				idOne = append(idOne, convertCircuitBreakerToInfo(cb))
			}
			if strings.Compare(ids[1], cb.Namespace) == 0 {
				idTwo = append(idTwo, convertCircuitBreakerToInfo(cb))
			}
			if strings.Compare(ids[2], cb.Namespace) == 0 {
				idThree = append(idThree, convertCircuitBreakerToInfo(cb))
			}
		}

		type fields struct {
			handler      BoltHandler
			ruleLock     *sync.RWMutex
			relationLock *sync.RWMutex
		}
		type args struct {
			filters map[string]string
			offset  uint32
			limit   uint32
		}
		tests := []struct {
			name    string
			fields  fields
			args    args
			want    *model.CircuitBreakerDetail
			wantErr bool
		}{
			{
				name: "",
				fields: fields{
					handler:      handler,
					ruleLock:     &sync.RWMutex{},
					relationLock: &sync.RWMutex{},
				},
				args: args{
					filters: map[string]string{
						"": "",
					},
					offset: 0,
					limit:  0,
				},
				want: &model.CircuitBreakerDetail{
					Total:               0,
					CircuitBreakerInfos: []*model.CircuitBreakerInfo{},
				},
				wantErr: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				c := &circuitBreakerStore{
					handler: tt.fields.handler,
				}
				got, err := c.ListReleaseCircuitBreakers(tt.args.filters, tt.args.offset, tt.args.limit)
				if (err != nil) != tt.wantErr {
					t.Errorf("circuitBreakerStore.ListReleaseCircuitBreakers() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("circuitBreakerStore.ListReleaseCircuitBreakers() = %v, want %v", got, tt.want)
				}
			})
		}
	})
}

func Test_circuitBreakerStore_GetCircuitBreakerForCache(t *testing.T) {

	CreateTableDBHandlerAndRun(t, "test_circuitbreaker", func(t *testing.T, handler BoltHandler) {
		c := &circuitBreakerStore{
			handler: handler,
		}

		tN := time.Now()

		cbs := make([]*model.ServiceWithCircuitBreaker, 0)

		for i := 0; i < 10; i++ {
			cb := createTestCircuitbreaker("", true)
			cb.CreateTime = tN.Add(time.Minute * time.Duration(i+1))
			cb.ModifyTime = tN.Add(time.Minute * time.Duration(i+1))
			if err := c.CreateCircuitBreaker(cb); err != nil {
				t.Fatal(err)
			}

			serviceId := RandStringRunes(5)

			if err := c.releaseCircuitBreaker(&model.CircuitBreakerRelation{
				ServiceID:   serviceId,
				RuleID:      cb.ID,
				RuleVersion: cb.Version,
				CreateTime:  tN.Add(time.Minute * time.Duration(i+1)),
				ModifyTime:  tN.Add(time.Minute * time.Duration(i+1)),
			}); err != nil {
				t.Fatal(err)
			}

			cbs = append(cbs, &model.ServiceWithCircuitBreaker{
				ServiceID:      serviceId,
				CircuitBreaker: cb,
				CreateTime:     tN.Add(time.Minute * time.Duration(i+1)),
				ModifyTime:     tN.Add(time.Minute * time.Duration(i+1)),
			})
		}

		compareCb := func(cbs []*model.ServiceWithCircuitBreaker, mtime time.Time) []*model.ServiceWithCircuitBreaker {
			result := make([]*model.ServiceWithCircuitBreaker, 0)
			for i := range cbs {
				if cbs[i].CircuitBreaker.ModifyTime.After(mtime) {
					result = append(result, cbs[i])
				}
			}
			return result
		}

		type fields struct {
			handler      BoltHandler
			ruleLock     *sync.RWMutex
			relationLock *sync.RWMutex
		}
		type args struct {
			mtime       time.Time
			firstUpdate bool
		}
		tests := []struct {
			name    string
			fields  fields
			args    args
			want    []*model.ServiceWithCircuitBreaker
			wantErr bool
		}{
			{
				name: "",
				fields: fields{
					handler:      handler,
					ruleLock:     &sync.RWMutex{},
					relationLock: &sync.RWMutex{},
				},
				args: args{
					mtime:       tN.Add(time.Minute * time.Duration(1)),
					firstUpdate: false,
				},
				want:    compareCb(cbs, tN.Add(time.Minute*time.Duration(1))),
				wantErr: false,
			},
			{
				name: "",
				fields: fields{
					handler:      handler,
					ruleLock:     &sync.RWMutex{},
					relationLock: &sync.RWMutex{},
				},
				args: args{
					mtime:       tN.Add(time.Minute * time.Duration(6)),
					firstUpdate: false,
				},
				want:    compareCb(cbs, tN.Add(time.Minute*time.Duration(6))),
				wantErr: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				c := &circuitBreakerStore{
					handler: tt.fields.handler,
				}
				got, err := c.GetCircuitBreakerForCache(tt.args.mtime, tt.args.firstUpdate)
				if (err != nil) != tt.wantErr {
					t.Errorf("circuitBreakerStore.GetCircuitBreakerForCache() error = %v, wantErr %v", err, tt.wantErr)
					return
				}

				tN := time.Now()

				for i := range got {
					got[i].CreateTime = tN
					got[i].ModifyTime = tN
					got[i].CircuitBreaker.CreateTime = tN
					got[i].CircuitBreaker.ModifyTime = tN
				}

				for i := range tt.want {
					tt.want[i].CreateTime = tN
					tt.want[i].ModifyTime = tN
					tt.want[i].CircuitBreaker.CreateTime = tN
					tt.want[i].CircuitBreaker.ModifyTime = tN
				}

				sort.Slice(got, func(i, j int) bool {
					return strings.Compare(got[i].ServiceID, got[j].ServiceID) < 0
				})

				sort.Slice(tt.want, func(i, j int) bool {
					return strings.Compare(tt.want[i].ServiceID, tt.want[j].ServiceID) < 0
				})

				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("circuitBreakerStore.GetCircuitBreakerForCache() = %v, want %v", got, tt.want)
				}
			})
		}
	})
}

func Test_circuitBreakerStore_GetCircuitBreakersByService(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_circuitbreaker", func(t *testing.T, handler BoltHandler) {
		type fields struct {
			handler BoltHandler
		}
		type args struct {
			name      string
			namespace string
		}
		tests := []struct {
			name    string
			fields  fields
			args    args
			want    *model.CircuitBreaker
			wantErr bool
			preEnv  func(args struct {
				name   string
				fields fields
				args   args
				want   *model.CircuitBreaker
			})
		}{
			{
				name: "Test all data is normal",
				fields: fields{
					handler: handler,
				},
				args: args{
					name:      RandStringRunes(6),
					namespace: RandStringRunes(10),
				},
				want:    createTestCircuitbreaker("", true),
				wantErr: false,
				preEnv: func(args struct {
					name   string
					fields fields
					args   args
					want   *model.CircuitBreaker
				}) {
					c := &circuitBreakerStore{
						handler: args.fields.handler,
					}

					s := &serviceStore{
						handler: handler,
					}

					service := createTestService("", args.args.name, args.args.namespace, true)

					// first, create one service
					if err := s.AddService(service); err != nil {
						t.Fatal(err)
					}

					// second, create circuitbreaker
					if err := c.CreateCircuitBreaker(args.want); err != nil {
						t.Fatal(err)
					}

					// third, create circuitbreaker relation
					crb := createTestCircuitbreakerRelation(args.want.ID, false)
					crb.ServiceID = service.ID
					crb.RuleVersion = args.want.Version

					if err := c.releaseCircuitBreaker(crb); err != nil {
						t.Fatal(err)
					}
				},
			},
			{
				name: "Test when service not found",
				fields: fields{
					handler: handler,
				},
				args: args{
					name:      RandStringRunes(6),
					namespace: RandStringRunes(10),
				},
				want:    nil,
				wantErr: true,
				preEnv: func(args struct {
					name   string
					fields fields
					args   args
					want   *model.CircuitBreaker
				}) {
					//	do nothing
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				c := &circuitBreakerStore{
					handler: tt.fields.handler,
				}

				tt.preEnv(struct {
					name   string
					fields fields
					args   args
					want   *model.CircuitBreaker
				}{name: tt.name, fields: tt.fields, args: tt.args, want: tt.want})

				got, err := c.GetCircuitBreakersByService(tt.args.name, tt.args.namespace)
				assert.NoError(t, err)

				if got == nil {
					if tt.want == nil {
						return
					} else {
						t.Errorf("circuitBreakerStore.GetCircuitBreakersByService() expect : %#v, actual is nil", tt.want)
						return
					}
				}

				tN := time.Now()
				got.CreateTime = tN
				got.ModifyTime = tN

				tt.want.CreateTime = tN
				tt.want.ModifyTime = tN

				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("circuitBreakerStore.GetCircuitBreakersByService() = %v, want %v", got, tt.want)
				}
			})
		}
	})
}
