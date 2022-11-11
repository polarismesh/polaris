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

package plugin

import (
	"os"
	"sync"

	commonLog "github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/model"
)

var (
	// 插件初始化原子变量
	historyOnce sync.Once
)

// History 历史记录插件
type History interface {
	Plugin
	Record(entry *model.RecordEntry)
}

// GetHistory 获取历史记录插件
func GetHistory() History {
	c := &config.History
	plugin, exist := pluginSet[c.Name]
	if !exist {
		return nil
	}

	historyOnce.Do(func() {
		if err := plugin.Initialize(c); err != nil {
			commonLog.GetScopeOrDefaultByName(c.Name).Errorf("plugin init err: %s", err.Error())
			os.Exit(-1)
		}
	})

	return plugin.(History)
}
