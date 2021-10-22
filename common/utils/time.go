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

package utils

import (
	"syscall"
	"time"
)

//获取低精度的微秒时间
func CurrentMillisecond() int64 {
	var tv syscall.Timeval
	if err := syscall.Gettimeofday(&tv); err != nil {
		return time.Now().UnixNano() / 1e6
	}
	return int64(tv.Sec)*1e3 + int64(tv.Usec)/1e3
}
