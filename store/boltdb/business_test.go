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
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/polarismesh/polaris-server/common/model"
)

var bsletterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandBusinessStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func createTestBusiness(id string, createId bool) *model.Business {

	if strings.Compare(id, "") == 0 && createId {
		id = uuid.NewString()
	}

	str := RandBusinessStringRunes(5)

	return &model.Business{
		ID:         id,
		Name:       str,
		Token:      str,
		Owner:      "polaris",
		Valid:      true,
		CreateTime: time.Now(),
		ModifyTime: time.Now(),
	}
}

func CreateBusinessDBHandlerAndRun(t *testing.T, tf func(t *testing.T, handler BoltHandler)) {
	tempDir, _ := ioutil.TempDir("", "test_business")
	_ = os.Remove(filepath.Join(tempDir, "test_business.bolt"))
	handler, err := NewBoltHandler(&BoltConfig{FileName: filepath.Join(tempDir, "test_business.bolt")})
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		_ = handler.Close()
		_ = os.Remove(filepath.Join(tempDir, "test_business.bolt"))
	}()
	tf(t, handler)
}

func Test_businessStore_AddBusiness(t *testing.T) {
	CreateBusinessDBHandlerAndRun(t, func(t *testing.T, handler BoltHandler) {

		bID := uuid.NewString()

		type fields struct {
			handler BoltHandler
		}
		type args struct {
			b *model.Business
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
					handler: handler,
				},
				args: args{
					b: createTestBusiness(bID, false),
				},
				wantErr: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				bs := &businessStore{
					handler: tt.fields.handler,
				}
				if err := bs.AddBusiness(tt.args.b); (err != nil) != tt.wantErr {
					t.Errorf("businessStore.AddBusiness() error = %v, wantErr %v", err, tt.wantErr)
				}
			})
		}
	})

}

func Test_businessStore_DeleteBusiness(t *testing.T) {
	CreateBusinessDBHandlerAndRun(t, func(t *testing.T, handler BoltHandler) {

		bId := uuid.NewString()

		type fields struct {
			handler BoltHandler
		}
		type args struct {
			bid string
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
					handler: handler,
				},
				args: args{
					bid: bId,
				},
				wantErr: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				bs := &businessStore{
					handler: tt.fields.handler,
				}

				if err := bs.AddBusiness(createTestBusiness(bId, false)); err != nil {
					t.Fatal(err)
				}

				if err := bs.DeleteBusiness(tt.args.bid); (err != nil) != tt.wantErr {
					t.Errorf("businessStore.DeleteBusiness() error = %v, wantErr %v", err, tt.wantErr)
				}

				b, err := bs.GetBusinessByID(bId)
				if err != nil {
					t.Fatal(err)
				}

				if b != nil {
					t.Errorf("businessStore.DeleteBusiness() not effect, still exist")
				}
			})
		}
	})
}

func Test_businessStore_UpdateBusiness(t *testing.T) {
	CreateBusinessDBHandlerAndRun(t, func(t *testing.T, handler BoltHandler) {

		bId := uuid.NewString()

		bs := &businessStore{
			handler: handler,
		}

		old := createTestBusiness(bId, false)

		if err := bs.AddBusiness(old); err != nil {
			t.Fatal(err)
		}

		type fields struct {
			handler BoltHandler
		}
		type args struct {
			b *model.Business
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
					handler: handler,
				},
				args: args{
					b: createTestBusiness(bId, false),
				},
				wantErr: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				bs := &businessStore{
					handler: tt.fields.handler,
				}
				if err := bs.UpdateBusiness(tt.args.b); (err != nil) != tt.wantErr {
					t.Errorf("businessStore.UpdateBusiness() error = %v, wantErr %v", err, tt.wantErr)
				}

				b, err := bs.GetBusinessByID(bId)
				if err != nil {
					t.Fatal(err)
				}

				tN := time.Now()
				b.CreateTime = tN
				b.ModifyTime = tN

				tt.args.b.CreateTime = tN
				tt.args.b.ModifyTime = tN

				if !reflect.DeepEqual(b, tt.args.b) {
					t.Errorf("businessStore.UpdateBusiness() not effect")
				}
			})
		}
	})
}

func Test_businessStore_UpdateBusinessToken(t *testing.T) {
	CreateBusinessDBHandlerAndRun(t, func(t *testing.T, handler BoltHandler) {
		bId := uuid.NewString()

		bs := &businessStore{
			handler: handler,
		}

		old := createTestBusiness(bId, false)

		if err := bs.AddBusiness(old); err != nil {
			t.Fatal(err)
		}

		type fields struct {
			handler BoltHandler
		}
		type args struct {
			bid   string
			token string
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
					handler: handler,
				},
				args: args{
					bid:   bId,
					token: RandBusinessStringRunes(10),
				},
				wantErr: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				bs := &businessStore{
					handler: tt.fields.handler,
				}
				if err := bs.UpdateBusinessToken(tt.args.bid, tt.args.token); (err != nil) != tt.wantErr {
					t.Errorf("businessStore.UpdateBusinessToken() error = %v, wantErr %v", err, tt.wantErr)
				}

				b, err := bs.GetBusinessByID(bId)
				if err != nil {
					t.Fatal(err)
				}

				if strings.Compare(b.Token, tt.args.token) != 0 {
					t.Errorf("businessStore.UpdateBusinessToken() not effect, expect : %s, actual : %s", tt.args.token, b.Token)
				}
			})
		}
	})
}

func Test_businessStore_ListBusiness(t *testing.T) {
	CreateBusinessDBHandlerAndRun(t, func(t *testing.T, handler BoltHandler) {
		bs := &businessStore{
			handler: handler,
		}

		owners := []string{"polaris_1", "test_2", "my_3"}
		polarisOne := make([]*model.Business, 0)
		polarisTwo := make([]*model.Business, 0)
		polaristhree := make([]*model.Business, 0)

		for i := 0; i < 20; i++ {
			business := createTestBusiness("", true)
			business.Owner = owners[rand.Intn(len(owners))]

			if err := bs.AddBusiness(business); err != nil {
				t.Fatal(err)
			}

			if strings.Compare(owners[0], business.Owner) == 0 {
				polarisOne = append(polarisOne, business)
			}
			if strings.Compare(owners[1], business.Owner) == 0 {
				polarisTwo = append(polarisTwo, business)
			}
			if strings.Compare(owners[2], business.Owner) == 0 {
				polaristhree = append(polaristhree, business)
			}
		}

		type fields struct {
			handler BoltHandler
		}
		type args struct {
			owner string
		}
		tests := []struct {
			name    string
			fields  fields
			args    args
			want    []*model.Business
			wantErr bool
		}{
			{
				name: "",
				fields: fields{
					handler: handler,
				},
				args: args{
					owner: owners[0],
				},
				want:    polarisOne,
				wantErr: false,
			},
			{
				name: "",
				fields: fields{
					handler: handler,
				},
				args: args{
					owner: owners[1],
				},
				want:    polarisTwo,
				wantErr: false,
			},
			{
				name: "",
				fields: fields{
					handler: handler,
				},
				args: args{
					owner: owners[2],
				},
				want:    polaristhree,
				wantErr: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				bs := &businessStore{
					handler: tt.fields.handler,
				}
				got, err := bs.ListBusiness(tt.args.owner)
				if (err != nil) != tt.wantErr {
					t.Errorf("businessStore.ListBusiness() error = %v, wantErr %v", err, tt.wantErr)
					return
				}

				if got == nil {
					return
				}

				tN := time.Now()

				gotM := make(map[string]*model.Business)
				for i := range got {
					gotM[got[i].ID] = got[i]
					gotM[got[i].ID].CreateTime = tN
					gotM[got[i].ID].ModifyTime = tN
				}

				wantM := make(map[string]*model.Business)
				for i := range tt.want {
					wantM[tt.want[i].ID] = tt.want[i]
					wantM[tt.want[i].ID].CreateTime = tN
					wantM[tt.want[i].ID].ModifyTime = tN
				}

				if !reflect.DeepEqual(gotM, wantM) {
					t.Errorf("businessStore.ListBusiness() = %#v, want %#v", gotM, wantM)
				}
			})
		}
	})
}

func Test_businessStore_GetBusinessByID(t *testing.T) {
	CreateBusinessDBHandlerAndRun(t, func(t *testing.T, handler BoltHandler) {

		bus := createTestBusiness("", true)

		type fields struct {
			handler BoltHandler
		}
		type args struct {
			id string
		}
		tests := []struct {
			name    string
			fields  fields
			args    args
			want    *model.Business
			wantErr bool
		}{
			{
				name: "",
				fields: fields{
					handler: handler,
				},
				args: args{
					id: bus.ID,
				},
				want:    bus,
				wantErr: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				bs := &businessStore{
					handler: tt.fields.handler,
				}
				got, err := bs.GetBusinessByID(tt.args.id)
				if (err != nil) != tt.wantErr {
					t.Errorf("businessStore.GetBusinessByID() error = %v, wantErr %v", err, tt.wantErr)
					return
				}

				if got == nil {
					return
				}

				tN := time.Now()
				got.CreateTime = tN
				got.ModifyTime = tN

				tt.want.CreateTime = tN
				tt.want.ModifyTime = tN

				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("businessStore.GetBusinessByID() = %v, want %v", got, tt.want)
				}
			})
		}
	})
}
