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

package batchjob

import "time"

// CtrlConfig CtrlConfig .
type CtrlConfig struct {
	// Label 批任务执行器标签
	Label string
	// QueueSize 注册请求队列的长度
	QueueSize int
	// WaitTime 最长多久一次批量操作
	WaitTime time.Duration
	// MaxBatchCount 每次操作最大的批量数
	MaxBatchCount int
	// Concurrency 任务工作协程数量
	Concurrency int
	// Handler 任务处理函数
	Handler func(tasks []Future)
}
