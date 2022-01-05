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

package batch

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/golang/mock/gomock"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
	amock "github.com/polarismesh/polaris-server/service/auth/mock"
	smock "github.com/polarismesh/polaris-server/store/mock"
	. "github.com/smartystreets/goconvey/convey"
)

// TestNewBatchCtrlWithConfig 测试New
func TestNewBatchCtrlWithConfig(t *testing.T) {
	Convey("正常新建", t, func() {
		ctrlConfig := &CtrlConfig{
			Open:          true,
			QueueSize:     1024,
			WaitTime:      "20ms",
			MaxBatchCount: 32,
			Concurrency:   64,
		}
		config := &Config{
			Register:   ctrlConfig,
			Deregister: ctrlConfig,
		}
		bc, err := NewBatchCtrlWithConfig(nil, nil, nil, config)
		So(err, ShouldBeNil)
		So(bc, ShouldNotBeNil)
		So(bc.register, ShouldNotBeNil)
		So(bc.deregister, ShouldNotBeNil)
	})
	Convey("可以关闭register和deregister的batch操作", t, func() {
		bc, err := NewBatchCtrlWithConfig(nil, nil, nil, nil)
		So(err, ShouldBeNil)
		So(bc, ShouldBeNil)

		config := &Config{
			Register:   &CtrlConfig{Open: false},
			Deregister: &CtrlConfig{Open: false},
		}
		bc, err = NewBatchCtrlWithConfig(nil, nil, nil, config)
		So(err, ShouldBeNil)
		So(bc, ShouldNotBeNil)
		So(bc.register, ShouldBeNil)
		So(bc.deregister, ShouldBeNil)
	})
}

func newCreateInstanceController(t *testing.T) (*Controller, *smock.MockStore, *amock.MockAuthority,
	context.CancelFunc) {
	ctl := gomock.NewController(t)
	storage := smock.NewMockStore(ctl)
	authority := amock.NewMockAuthority(ctl)
	defer ctl.Finish()
	config := &Config{
		Register: &CtrlConfig{
			Open:          true,
			QueueSize:     1024,
			WaitTime:      "16ms",
			MaxBatchCount: 8,
			Concurrency:   4,
		},
	}
	bc, err := NewBatchCtrlWithConfig(storage, authority, nil, config)
	if bc == nil || err != nil {
		t.Fatalf("error: %+v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	bc.Start(ctx)
	return bc, storage, authority, cancel
	//defer cancel()
}

func sendAsyncCreateInstance(bc *Controller) error {
	var wg sync.WaitGroup
	ch := make(chan error, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			future := bc.AsyncCreateInstance(&api.Instance{
				Id:           utils.NewStringValue(fmt.Sprintf("%d", index)),
				ServiceToken: utils.NewStringValue(fmt.Sprintf("%d", index)),
			},
				"", "")
			if err := future.Wait(); err != nil {
				fmt.Printf("%+v\n", err)
				ch <- err
			}
		}(i)
	}
	wg.Wait()
	select {
	case err := <-ch:
		if err != nil {
			return err
		}
	default:
		return nil
	}
	return nil
}

// TestAsyncCreateInstance test AsyncCreateInstance
func TestAsyncCreateInstance(t *testing.T) {
	Convey("正常创建实例", t, func() {
		bc, storage, authority, cancel := newCreateInstanceController(t)
		defer cancel()
		storage.EXPECT().CheckInstancesExisted(gomock.Any()).Return(nil, nil).AnyTimes()
		storage.EXPECT().GetSourceServiceToken(gomock.Any(), gomock.Any()).
			Return(&model.Service{ID: "1"}, nil).AnyTimes()
		authority.EXPECT().VerifyInstance(gomock.Any(), gomock.Any()).Return(true).AnyTimes()
		storage.EXPECT().BatchAddInstances(gomock.Any()).Return(nil).AnyTimes()
		So(sendAsyncCreateInstance(bc), ShouldBeNil)
	})
	Convey("鉴权失败", t, func() {
		bc, storage, authority, cancel := newCreateInstanceController(t)
		defer cancel()
		storage.EXPECT().CheckInstancesExisted(gomock.Any()).Return(nil, nil).AnyTimes()
		storage.EXPECT().GetSourceServiceToken(gomock.Any(), gomock.Any()).
			Return(&model.Service{ID: "1"}, nil).AnyTimes()
		authority.EXPECT().VerifyInstance(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
		err := sendAsyncCreateInstance(bc)
		So(err, ShouldNotBeNil)
		t.Logf("%+v", err)
	})
}

// TestSendReply 测试reply
func TestSendReply(t *testing.T) {
	Convey("可以正常获取类型", t, func() {
		SendReply(make([]*InstanceFuture, 0, 10), 1, nil)
	})
	Convey("可以正常获取类型2", t, func() {
		SendReply(make(map[string]*InstanceFuture, 10), 1, nil)
	})
	Convey("其他类型不通过", t, func() {
		SendReply("test string", 1, nil)
	})
}
