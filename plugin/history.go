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

	"github.com/polarismesh/polaris/common/model"
)

var (
	// historyOnce Plugin initialization atomic variable
	historyOnce      sync.Once
	compositeHistory *CompositeHistory
)

// History 历史记录插件
type History interface {
	Plugin
	Record(entry *model.RecordEntry)
}

// GetHistory Get the historical record plugin
func GetHistory() History {
	if compositeHistory != nil {
		return compositeHistory
	}

	historyOnce.Do(func() {
		var (
			entries []ConfigEntry
		)

		if len(config.History.Entries) != 0 {
			entries = append(entries, config.History.Entries...)
		} else {
			entries = append(entries, ConfigEntry{
				Name:   config.History.Name,
				Option: config.History.Option,
			})
		}

		compositeHistory = &CompositeHistory{
			chain:   make([]History, 0, len(entries)),
			options: entries,
		}

		if err := compositeHistory.Initialize(nil); err != nil {
			log.Errorf("History plugin init err: %s", err.Error())
			os.Exit(-1)
		}
	})

	return compositeHistory
}

type CompositeHistory struct {
	chain   []History
	options []ConfigEntry
}

func (c *CompositeHistory) Name() string {
	return "CompositeHistory"
}

func (c *CompositeHistory) Initialize(config *ConfigEntry) error {
	for i := range c.options {
		entry := c.options[i]
		item, exist := pluginSet[entry.Name]
		if !exist {
			log.Errorf("plugin History not found target: %s", entry.Name)
			continue
		}

		history, ok := item.(History)
		if !ok {
			log.Errorf("plugin target: %s not History", entry.Name)
			continue
		}

		if err := history.Initialize(&entry); err != nil {
			return err
		}
		c.chain = append(c.chain, history)
	}
	return nil
}

func (c *CompositeHistory) Destroy() error {
	for i := range c.chain {
		if err := c.chain[i].Destroy(); err != nil {
			return err
		}
	}
	return nil
}

func (c *CompositeHistory) Record(entry *model.RecordEntry) {
	for i := range c.chain {
		c.chain[i].Record(entry)
	}
}
