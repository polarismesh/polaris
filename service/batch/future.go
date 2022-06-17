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
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
	"go.uber.org/zap"
)

// InstanceFuture 创建实例的异步结构体
type InstanceFuture struct {
	serviceId string
	// api请求对象
	request *api.Instance
	// 从数据库中读取到的model信息
	instance *model.Instance
	// 记录对外API的错误码
	code uint32
	// 这个 future 是否会被外部调用 Wait 接口
	needWait bool
	// 执行成功/失败的应答chan
	result chan error
	// 健康与否
	healthy bool
}

// Reply future的应答
func (future *InstanceFuture) Reply(code uint32, result error) {
	if !future.needWait {
		if result != nil {
			log.Error("[Instance][Regis] receive future result", zap.String("service-id", future.serviceId),
				zap.Error(result))
		}
		return
	}

	future.code = code

	select {
	case future.result <- result:
	default:
		log.Warnf("[Batch] instance(%s) future is not captured", future.request.GetId().GetValue())
	}
}

// Wait 外部调用者，需要调用Wait等待执行结果
func (future *InstanceFuture) Wait() error {
	if !future.needWait {
		return nil
	}
	return <-future.result
}

// SetInstance 设置ins
func (future *InstanceFuture) SetInstance(instance *model.Instance) {
	future.instance = instance
}

// Instance 获取ins
func (future *InstanceFuture) Instance() *model.Instance {
	return future.instance
}

// Code 获取code
func (future *InstanceFuture) Code() uint32 {
	return future.code
}

// sendReply 批量答复futures
func sendReply(futures interface{}, code uint32, result error) {
	switch futureType := futures.(type) {
	case []*InstanceFuture:
		for _, entry := range futureType {
			entry.Reply(code, result)
		}
	case map[string]*InstanceFuture:
		for _, entry := range futureType {
			entry.Reply(code, result)
		}
	default:
		log.Errorf("[Controller] not found reply futures type: %T", futures)
	}
}
