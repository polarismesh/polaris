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
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	"github.com/polarismesh/polaris/common/model"
)

// InstanceFuture 创建实例的异步结构体
type ClientFuture struct {
	request *apiservice.Client // api请求对象
	client  *model.Client      // 从数据库中读取到的model信息
	code    apimodel.Code      // 记录对外API的错误码
	result  chan error         // 执行成功/失败的应答chan
}

// Reply future的应答
func (future *ClientFuture) Reply(code apimodel.Code, result error) {
	future.code = code

	select {
	case future.result <- result:
	default:
		log.Warnf("[Batch] client(%s) future is not captured", future.request.GetId().GetValue())
	}
}

// Wait 外部调用者，需要调用Wait等待执行结果
func (future *ClientFuture) Wait() error {
	return <-future.result
}

// SetClient 设置 client 信息
func (future *ClientFuture) SetClient(client *model.Client) {
	future.client = client
}

// Client 获取 client 信息
func (future *ClientFuture) Client() *model.Client {
	return future.client
}

// Code 获取code
func (future *ClientFuture) Code() apimodel.Code {
	return future.code
}

// SendReply 批量答复futures
func SendClientReply(futures interface{}, code apimodel.Code, result error) {
	switch futureType := futures.(type) {
	case []*ClientFuture:
		for _, entry := range futureType {
			entry.Reply(code, result)
		}
	case map[string]*ClientFuture:
		for _, entry := range futureType {
			entry.Reply(code, result)
		}
	default:
		log.Errorf("[Controller] not found reply client futures type: %T", futures)
	}
}
