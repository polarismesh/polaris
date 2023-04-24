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
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/golang/mock/gomock"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"

	"github.com/polarismesh/polaris/common/metrics"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	smock "github.com/polarismesh/polaris/store/mock"
)

func init() {
	metrics.InitMetrics()
}

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
		bc, err := NewBatchCtrlWithConfig(nil, nil, config)
		So(err, ShouldBeNil)
		So(bc, ShouldNotBeNil)
		So(bc.register, ShouldNotBeNil)
		So(bc.deregister, ShouldNotBeNil)
	})
	Convey("可以关闭register和deregister的batch操作", t, func() {
		bc, err := NewBatchCtrlWithConfig(nil, nil, nil)
		So(err, ShouldBeNil)
		So(bc, ShouldBeNil)

		config := &Config{
			Register:   &CtrlConfig{Open: false},
			Deregister: &CtrlConfig{Open: false},
		}
		bc, err = NewBatchCtrlWithConfig(nil, nil, config)
		So(err, ShouldBeNil)
		So(bc, ShouldNotBeNil)
		So(bc.register, ShouldBeNil)
		So(bc.deregister, ShouldBeNil)
	})
}

func newCreateInstanceController(t *testing.T) (*gomock.Controller, *Controller, *smock.MockStore, context.CancelFunc) {
	ctl := gomock.NewController(t)
	storage := smock.NewMockStore(ctl)
	config := &Config{
		Register: &CtrlConfig{
			Open:          true,
			QueueSize:     1024,
			WaitTime:      "16ms",
			MaxBatchCount: 8,
			Concurrency:   4,
		},
	}
	bc, err := NewBatchCtrlWithConfig(storage, nil, config)
	if bc == nil || err != nil {
		t.Fatalf("error: %+v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	bc.Start(ctx)
	return ctl, bc, storage, cancel
}

func sendAsyncCreateInstance(bc *Controller, cnt int32) error {
	var wg sync.WaitGroup
	ch := make(chan error, cnt)
	for i := int32(0); i < cnt; i++ {
		wg.Add(1)
		go func(index int32) {
			defer wg.Done()
			future := bc.AsyncCreateInstance(utils.NewUUID(), &apiservice.Instance{
				Id:           utils.NewStringValue(fmt.Sprintf("%d", index)),
				ServiceToken: utils.NewStringValue(fmt.Sprintf("%d", index)),
			}, true)
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
	t.Run("正常创建实例", func(t *testing.T) {
		ctrl, bc, storage, cancel := newCreateInstanceController(t)
		t.Cleanup(func() {
			ctrl.Finish()
			cancel()
		})
		mockSvc := &model.Service{ID: "1"}
		actualCommit := int32(0)
		totalIns := int32(100)
		mockTrx := smock.NewMockTransaction(ctrl)
		mockTrx.EXPECT().Commit().Do(func() {
			atomic.AddInt32(&actualCommit, 1)
		}).AnyTimes()
		mockTrx.EXPECT().RLockService(gomock.Any(), gomock.Any()).DoAndReturn(func(_, _ string) (*model.Service, error) {
			return mockSvc, nil
		}).AnyTimes()

		storage.EXPECT().BatchGetInstanceIsolate(gomock.Any()).Return(nil, nil).AnyTimes()
		storage.EXPECT().GetSourceServiceToken(gomock.Any(), gomock.Any()).
			Return(mockSvc, nil).AnyTimes()
		storage.EXPECT().GetServiceByID(gomock.Any()).Return(mockSvc, nil).AnyTimes()
		storage.EXPECT().CreateTransaction().Return(mockTrx, nil).AnyTimes()
		storage.EXPECT().BatchAddInstances(gomock.Any()).Return(nil).AnyTimes()
		assert.NoError(t, sendAsyncCreateInstance(bc, totalIns))
		assert.True(t, totalIns/int32(8) <= actualCommit && actualCommit <= totalIns/int32(8)+int32(1))
	})

	t.Run("创建实例-lockService随机出现错误", func(t *testing.T) {
		ctrl, bc, storage, cancel := newCreateInstanceController(t)
		t.Cleanup(func() {
			ctrl.Finish()
			cancel()
		})
		mockSvc := &model.Service{ID: "1"}
		actualCommit := int32(0)
		totalIns := int32(100)
		hasErr := int32(0)
		mockTrx := smock.NewMockTransaction(ctrl)
		mockTrx.EXPECT().Commit().Do(func() {
			atomic.AddInt32(&actualCommit, 1)
		}).AnyTimes()
		mockTrx.EXPECT().RLockService(gomock.Any(), gomock.Any()).DoAndReturn(func(_, _ string) (*model.Service, error) {
			if rand.Float64() < 0.5 {
				return mockSvc, nil
			}
			atomic.StoreInt32(&hasErr, 1)
			return nil, errors.New("mock RLockService fail")
		}).AnyTimes()

		storage.EXPECT().BatchGetInstanceIsolate(gomock.Any()).Return(nil, nil).AnyTimes()
		storage.EXPECT().GetSourceServiceToken(gomock.Any(), gomock.Any()).
			Return(mockSvc, nil).AnyTimes()
		storage.EXPECT().GetServiceByID(gomock.Any()).Return(mockSvc, nil).AnyTimes()
		storage.EXPECT().CreateTransaction().Return(mockTrx, nil).AnyTimes()
		storage.EXPECT().BatchAddInstances(gomock.Any()).Return(nil).AnyTimes()
		err := sendAsyncCreateInstance(bc, totalIns)
		if atomic.LoadInt32(&hasErr) == 1 {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
		assert.True(t, totalIns/int32(8) <= actualCommit && actualCommit <= totalIns/int32(8)+int32(1))
	})
}

// TestSendReply 测试reply
func TestSendReply(t *testing.T) {
	Convey("可以正常获取类型", t, func() {
		sendReply(make([]*InstanceFuture, 0, 10), 1, nil)
	})
	Convey("可以正常获取类型2", t, func() {
		sendReply(make(map[string]*InstanceFuture, 10), 1, nil)
	})
	Convey("其他类型不通过", t, func() {
		sendReply("test string", 1, nil)
	})
}
