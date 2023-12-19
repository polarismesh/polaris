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

package remote

import (
	"context"
	"sync"
	"time"

	nacosmodel "github.com/polarismesh/polaris/apiserver/nacosserver/model"
	nacospb "github.com/polarismesh/polaris/apiserver/nacosserver/v2/pb"
	"github.com/polarismesh/polaris/common/utils"
)

type (
	// InFlights
	InFlights struct {
		inFlights *utils.SyncMap[string, *ClientInFlights]
	}

	// ClientInFlights
	ClientInFlights struct {
		inFlights *utils.SyncMap[string, *InFlight]
	}

	// InFlight
	InFlight struct {
		once       sync.Once
		ConnID     string
		RequestID  string
		Callback   func(map[string]interface{}, nacospb.BaseResponse, error)
		ExpireTime time.Time
		Attachment map[string]interface{}
	}
)

func (i *InFlight) IsExpire(now time.Time) bool {
	return i.ExpireTime.Before(now)
}

func NewInFlights(ctx context.Context) *InFlights {
	inFlights := &InFlights{inFlights: utils.NewSyncMap[string, *ClientInFlights]()}
	go inFlights.notifyOutDateInFlight(ctx)
	return inFlights
}

func (i *InFlights) NotifyInFlight(connID string, resp nacospb.BaseResponse) {
	clientInflight, ok := i.inFlights.Load(connID)
	if !ok {
		nacoslog.Warnf("[NACOS-V2][InFlight] not found client(%s) inflights", connID)
		return
	}

	inflight, ok := clientInflight.inFlights.Delete(resp.GetRequestId())
	if !ok {
		nacoslog.Warnf("[NACOS-V2][InFlight] not found client(%s) req(%s) inflights", connID, resp.GetRequestId())
		return
	}

	if resp.GetResultCode() != int(nacosmodel.Response_Success.Code) {
		inflight.Callback(inflight.Attachment, resp, &nacosmodel.NacosError{
			ErrCode: int32(resp.GetErrorCode()),
			ErrMsg:  resp.GetMessage(),
		})
		return
	}
	inflight.once.Do(func() {
		inflight.Callback(inflight.Attachment, resp, nil)
	})
}

// AddInFlight 添加一个待回调通知的 InFligjt
func (i *InFlights) AddInFlight(inflight *InFlight) error {
	connID := inflight.ConnID
	clientInFlights, _ := i.inFlights.ComputeIfAbsent(connID, func(k string) *ClientInFlights {
		return &ClientInFlights{
			inFlights: utils.NewSyncMap[string, *InFlight](),
		}
	})

	_, isAdd := clientInFlights.inFlights.ComputeIfAbsent(inflight.RequestID, func(k string) *InFlight {
		return inflight
	})
	if !isAdd {
		return &nacosmodel.NacosError{
			ErrCode: int32(nacosmodel.ExceptionCode_ClientInvalidParam),
			ErrMsg:  "InFlight request id conflict",
		}
	}
	return nil
}

func (i *InFlights) notifyOutDateInFlight(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			i.inFlights.ReadRange(func(connID string, val *ClientInFlights) {
				val.inFlights.ReadRange(func(reqId string, inFlight *InFlight) {
					if !inFlight.IsExpire(now) {
						return
					}
					val.inFlights.Delete(reqId)
					inFlight.once.Do(func() {
						inFlight.Callback(inFlight.Attachment, nil, context.DeadlineExceeded)
					})
				})
			})
		}
	}
}
