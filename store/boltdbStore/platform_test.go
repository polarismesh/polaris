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

package boltdbStore

import (
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/polarismesh/polaris-server/common/model"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func createTestPlatform(id string, createId bool) *model.Platform {

	if strings.Compare(id, "") == 0 && createId {
		id = uuid.NewString()
	}

	str := RandStringRunes(5)

	return &model.Platform{
		ID:         id,
		Name:       "platform_test_" + str,
		Domain:     "polaris." + str,
		QPS:        1000,
		Token:      "platform_test_" + str,
		Owner:      "platform_test_" + str,
		Department: "platform_test_" + str,
		Comment:    "platform_test_" + str,
		Valid:      false,
		CreateTime: time.Now(),
		ModifyTime: time.Now(),
	}
}

func CreatePlatformDBHandlerAndRun(t *testing.T, tf func(t *testing.T, handler BoltHandler)) {
	tempDir, _ := ioutil.TempDir("", "test_platform")
	_ = os.Remove(filepath.Join(tempDir, "test_platform.bolt"))
	handler, err := NewBoltHandler(&BoltConfig{FileName: filepath.Join(tempDir, "test_platform.bolt")})
	if nil != err {
		t.Fatal(err)
	}

	defer func() {
		_ = handler.Close()
		_ = os.Remove(filepath.Join(tempDir, "test_platform.bolt"))
	}()
	tf(t, handler)
}

func Test_platformStore_CreatePlatform(t *testing.T) {
	CreatePlatformDBHandlerAndRun(t, func(t *testing.T, handler BoltHandler) {

		platformId := uuid.NewString()

		type fields struct {
			handler BoltHandler
		}
		type args struct {
			platform *model.Platform
		}
		tests := []struct {
			name     string
			fields   fields
			args     args
			wantErr  bool
			checkErr func(err error) bool
		}{
			{
				name: "platform_test_1",
				fields: fields{
					handler: handler,
				},
				args: args{
					platform: createTestPlatform(platformId, false),
				},
				wantErr: false,
			},
			{
				name: "platform_test_2",
				fields: fields{
					handler: handler,
				},
				args: args{
					platform: createTestPlatform("", false),
				},
				wantErr: true,
				checkErr: func(err error) bool {
					return strings.Contains(err.Error(), "create platform missing id")
				},
			},
			{
				name: "platform_test_3",
				fields: fields{
					handler: handler,
				},
				args: args{
					platform: createTestPlatform(platformId, false),
				},
				wantErr: true,
				checkErr: func(err error) bool {
					return strings.Contains(err.Error(), "duplicate")
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				p := &platformStore{
					handler: tt.fields.handler,
				}
				err := p.CreatePlatform(tt.args.platform)
				if (err != nil) != tt.wantErr {
					t.Errorf("platformStore.CreatePlatform() error = %v, wantErr %v", err, tt.wantErr)
				}
				if err != nil && tt.checkErr != nil && !tt.checkErr(err) {
					t.Errorf("platformStore.CreatePlatform() error = %v, checkErr false", err)
				}
			})
		}
	})
}

func Test_platformStore_UpdatePlatform(t *testing.T) {
	CreatePlatformDBHandlerAndRun(t, func(t *testing.T, handler BoltHandler) {

		platformId := uuid.NewString()

		platforms := make([]*model.Platform, 3)
		for i := 0; i < 3; i++ {
			platforms[i] = createTestPlatform(platformId, false)
		}

		p := &platformStore{
			handler: handler,
		}

		if err := p.CreatePlatform(createTestPlatform(platformId, false)); err != nil {
			t.Fatal(err)
		}

		type fields struct {
			handler BoltHandler
			lock    *sync.RWMutex
		}
		type args struct {
			platform *model.Platform
		}
		tests := []struct {
			name    string
			fields  fields
			args    args
			wantErr bool
		}{
			{
				name: "platform_test_1",
				fields: fields{
					handler: handler,
				},
				args: args{
					platform: platforms[0],
				},
				wantErr: false,
			},
			{
				name: "platform_test_2",
				fields: fields{
					handler: handler,
				},
				args: args{
					platform: platforms[1],
				},
				wantErr: false,
			},
			{
				name: "platform_test_3",
				fields: fields{
					handler: handler,
				},
				args: args{
					platform: platforms[2],
				},
				wantErr: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				p := &platformStore{
					handler: tt.fields.handler,
				}
				if err := p.UpdatePlatform(tt.args.platform); (err != nil) != tt.wantErr {
					t.Errorf("platformStore.UpdatePlatform() error = %v, wantErr %v", err, tt.wantErr)
				}

				val, err := p.GetPlatformById(tt.args.platform.ID)
				if err != nil {
					t.Fatal(err)
				}

				tN := time.Now()
				val.CreateTime = tN
				val.ModifyTime = tN
				tt.args.platform.CreateTime = tN
				tt.args.platform.ModifyTime = tN

				if !reflect.DeepEqual(val, tt.args.platform) {
					t.Errorf("platformStore.UpdatePlatform() failed, expect res %#v: , acutal res : %#v", tt.args.platform, val)
				}
			})
		}
	})
}

func Test_platformStore_DeletePlatform(t *testing.T) {
	CreatePlatformDBHandlerAndRun(t, func(t *testing.T, handler BoltHandler) {
		p := &platformStore{
			handler: handler,
		}

		platform := createTestPlatform(uuid.NewString(), false)

		if err := p.CreatePlatform(platform); err != nil {
			t.Errorf("platformStore.DeletePlatform() create error = %v", err)
		}

		if err := p.DeletePlatform(platform.ID); err != nil {
			t.Errorf("platformStore.DeletePlatform() error = %v", err)
		}

		val, err := p.GetPlatformById(platform.ID)
		if err != nil {
			t.Fatal(err)
		}

		if val != nil {
			t.Errorf("platformStore.DeletePlatform() delete platform not effect")
		}
	})
}

func Test_platformStore_GetPlatforms(t *testing.T) {
	CreatePlatformDBHandlerAndRun(t, func(t *testing.T, handler BoltHandler) {

		p := &platformStore{
			handler: handler,
		}

		platNames := []string{"polaris_1", "polaris_2", "polaris_3"}
		wantResSlice := make([][]*model.Platform, 2)
		wantResSlice[0] = make([]*model.Platform, 0)
		wantResSlice[1] = make([]*model.Platform, 0)

		// create 20 and save
		platforms := make([]*model.Platform, 20)
		platformsOne := make([]*model.Platform, 0)
		platformsTwo := make([]*model.Platform, 0)
		platformsThree := make([]*model.Platform, 0)

		for i := 0; i < 20; i++ {
			platforms[i] = createTestPlatform(uuid.NewString(), false)
			tN := time.Now().Add(time.Minute * time.Duration(i+1))
			platforms[i].Name = platNames[rand.Intn(len(platNames))]
			platforms[i].CreateTime = tN
			platforms[i].ModifyTime = tN

			if err := p.CreatePlatform(platforms[i]); err != nil {
				t.Errorf("platformStore.DeletePlatform() create error = %v", err)
			}

			if strings.Compare(platforms[i].Name, platNames[0]) == 0 {
				platformsOne = append(platformsOne, platforms[i])
				wantResSlice[0] = append(wantResSlice[0], platforms[i])
			}
			if strings.Compare(platforms[i].Name, platNames[1]) == 0 {
				platformsTwo = append(platformsTwo, platforms[i])
				wantResSlice[1] = append(wantResSlice[1], platforms[i])
			}
			if strings.Compare(platforms[i].Name, platNames[2]) == 0 {
				platformsThree = append(platformsThree, platforms[i])
			}
		}

		type fields struct {
			handler BoltHandler
			lock    *sync.RWMutex
		}
		type args struct {
			query  map[string]string
			offset uint32
			limit  uint32
		}
		tests := []struct {
			name    string
			fields  fields
			args    args
			want    uint32
			wantRes []*model.Platform
			wantErr bool
		}{
			{
				name: "",
				fields: fields{
					handler: handler,
				},
				args: args{
					query: map[string]string{
						"Name": platNames[0],
					},
					offset: 0,
					limit:  20,
				},
				want:    uint32(len(wantResSlice[0])),
				wantRes: wantResSlice[0],
				wantErr: false,
			},
			{
				name: "",
				fields: fields{
					handler: handler,
				},
				args: args{
					query: map[string]string{
						"Name": platNames[1],
					},
					offset: 0,
					limit:  20,
				},
				want:    uint32(len(wantResSlice[1])),
				wantRes: wantResSlice[1],
				wantErr: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				p := &platformStore{
					handler: tt.fields.handler,
				}
				got, got1, err := p.GetPlatforms(tt.args.query, tt.args.offset, tt.args.limit)
				if (err != nil) != tt.wantErr {
					t.Errorf("platformStore.GetPlatforms() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if got != tt.want {
					t.Errorf("platformStore.GetPlatforms() got = %v, want %v", got, tt.want)
				}

				// The effect of shielding time on results
				tN := time.Now()

				sort.Slice(got1, func(i, j int) bool {
					got1[i].CreateTime = tN
					got1[i].ModifyTime = tN
					got1[j].CreateTime = tN
					got1[j].ModifyTime = tN
					return strings.Compare(got1[i].ID, got1[j].ID) < 0
				})
				sort.Slice(tt.wantRes, func(i, j int) bool {
					tt.wantRes[i].CreateTime = tN
					tt.wantRes[i].ModifyTime = tN
					tt.wantRes[j].CreateTime = tN
					tt.wantRes[j].ModifyTime = tN
					return strings.Compare(tt.wantRes[i].ID, tt.wantRes[j].ID) < 0
				})

				if !reflect.DeepEqual(got1, tt.wantRes) {
					t.Errorf("platformStore.GetPlatforms() got1 = %v, want %v", got1, tt.wantRes)
				}
			})
		}
	})
}
