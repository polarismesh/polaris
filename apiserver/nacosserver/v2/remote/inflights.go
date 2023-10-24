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
	"sync"

	nacosmodel "github.com/polarismesh/polaris/apiserver/nacosserver/model"
	nacospb "github.com/polarismesh/polaris/apiserver/nacosserver/v2/pb"
)

type (
	// InFlights
	InFlights struct {
		lock      sync.RWMutex
		inFlights map[string]*ClientInFlights
	}

	// ClientInFlights
	ClientInFlights struct {
		lock      sync.RWMutex
		inFlights map[string]*InFlight
	}

	// InFlight
	InFlight struct {
		ConnID    string
		RequestID string
		Callback  func(nacospb.BaseResponse, error)
	}
)

func (i *InFlights) NotifyInFlight(connID string, resp nacospb.BaseResponse) {
	i.lock.RLock()
	clientInflight, ok := i.inFlights[connID]
	i.lock.RUnlock()

	if !ok {
		nacoslog.Warnf("[NACOS-V2][InFlight] not found client(%s) inflights", connID)
		return
	}

	clientInflight.lock.Lock()
	defer clientInflight.lock.Unlock()

	inflight, ok := clientInflight.inFlights[resp.GetRequestId()]
	if !ok {
		nacoslog.Warnf("[NACOS-V2][InFlight] not found client(%s) req(%s) inflights", connID, resp.GetRequestId())
		return
	}

	if resp.GetResultCode() != int(nacosmodel.Response_Success.Code) {
		inflight.Callback(resp, &nacosmodel.NacosError{
			ErrCode: int32(resp.GetErrorCode()),
			ErrMsg:  resp.GetMessage(),
		})
		return
	}
	inflight.Callback(resp, nil)
}

// AddInFlight 添加一个待回调通知的 InFligjt
func (i *InFlights) AddInFlight(inflight *InFlight) error {
	i.lock.Lock()
	connID := inflight.ConnID
	if _, ok := i.inFlights[connID]; !ok {
		i.inFlights[connID] = &ClientInFlights{
			inFlights: map[string]*InFlight{},
		}
	}
	clientInFlights := i.inFlights[connID]
	i.lock.Unlock()

	clientInFlights.lock.Lock()
	defer clientInFlights.lock.Unlock()

	if _, ok := clientInFlights.inFlights[inflight.RequestID]; ok {
		return &nacosmodel.NacosError{
			ErrCode: int32(nacosmodel.ExceptionCode_ClientInvalidParam),
			ErrMsg:  "InFlight request id conflict",
		}
	}

	clientInFlights.inFlights[inflight.RequestID] = inflight
	return nil
}
