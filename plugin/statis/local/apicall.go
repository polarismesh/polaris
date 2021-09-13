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

package local

import (
	"fmt"

	"go.uber.org/zap"
)

/**
 * APICall 接口调用
 */
type APICall struct {
	api      string
	code     int
	duration int64
}

/**
 * APICallStatisItem 接口调用统计条目
 */
type APICallStatisItem struct {
	api     string
	code    int
	count   int64
	accTime int64
	minTime int64
	maxTime int64
}

/**
 * APICallStatis 接口调用统计
 */
type APICallStatis struct {
	statis map[string]*APICallStatisItem

	logger *zap.Logger
}

/**
 * @brief 添加接口调用数据
 */
func (a *APICallStatis) add(ac *APICall) {
	index := fmt.Sprintf("%v-%v", ac.api, ac.code)

	item, exist := a.statis[index]
	if exist {
		item.count++

		item.accTime += ac.duration
		if ac.duration < item.minTime {
			item.minTime = ac.duration
		}
		if ac.duration > item.maxTime {
			item.maxTime = ac.duration
		}
	} else {
		a.statis[index] = &APICallStatisItem{
			api:     ac.api,
			code:    ac.code,
			count:   1,
			accTime: ac.duration,
			minTime: ac.duration,
			maxTime: ac.duration,
		}
	}
}

/**
 * @brief 打印接口调用统计
 */
func (a *APICallStatis) log() {
	if len(a.statis) == 0 {
		a.logger.Info("Statis: No API Call\n")
		return
	}

	msg := "Statis:\n"

	msg += fmt.Sprintf("%-48v|%12v|%12v|%12v|%12v|%12v|\n", "", "Code", "Count", "Min(ms)", "Max(ms)", "Avg(ms)")

	for _, item := range a.statis {
		msg += fmt.Sprintf("%-48v|%12v|%12v|%12.3f|%12.3f|%12.3f|\n",
			item.api, item.code, item.count,
			float32(item.minTime)/1e6,
			float32(item.maxTime)/1e6,
			float32(item.accTime)/float32(item.count)/1e6,
		)
	}

	a.logger.Info(msg)

	a.statis = make(map[string]*APICallStatisItem)
}
