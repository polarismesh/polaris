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
	discoverEventOnce sync.Once
	_discoverChannel  DiscoverChannel
)

// DiscoverChannel is used to receive discover events from the agent
type DiscoverChannel interface {
	Plugin
	// PublishEvent Release a service event
	PublishEvent(event model.InstanceEvent)
}

// GetDiscoverEvent Get service discovery event plug -in
func GetDiscoverEvent() DiscoverChannel {
	if _discoverChannel != nil {
		return _discoverChannel
	}

	discoverEventOnce.Do(func() {
		var (
			entries []ConfigEntry
		)

		if len(config.DiscoverEvent.Entries) != 0 {
			entries = append(entries, config.DiscoverEvent.Entries...)
		} else {
			entries = append(entries, ConfigEntry{
				Name:   config.DiscoverEvent.Name,
				Option: config.DiscoverEvent.Option,
			})
		}

		_discoverChannel = newCompositeDiscoverChannel(entries)
		if err := _discoverChannel.Initialize(nil); err != nil {
			log.Errorf("DiscoverChannel plugin init err: %s", err.Error())
			os.Exit(-1)
		}
	})

	return _discoverChannel
}

// newCompositeDiscoverChannel creates Composite DiscoverChannel
func newCompositeDiscoverChannel(options []ConfigEntry) *compositeDiscoverChannel {
	return &compositeDiscoverChannel{
		chain:   make([]DiscoverChannel, 0, len(options)),
		options: options,
	}
}

// compositeDiscoverChannel is used to receive discover events from the agent
type compositeDiscoverChannel struct {
	chain   []DiscoverChannel
	options []ConfigEntry
}

func (c *compositeDiscoverChannel) Name() string {
	return "CompositeDiscoverChannel"
}

func (c *compositeDiscoverChannel) Initialize(config *ConfigEntry) error {
	for i := range c.options {
		entry := c.options[i]
		item, exist := pluginSet[entry.Name]
		if !exist {
			log.Errorf("plugin DiscoverChannel not found target: %s", entry.Name)
			continue
		}

		discoverChannel, ok := item.(DiscoverChannel)
		if !ok {
			log.Errorf("plugin target: %s not DiscoverChannel", entry.Name)
			continue
		}

		if err := discoverChannel.Initialize(&entry); err != nil {
			return err
		}
		c.chain = append(c.chain, discoverChannel)
	}
	return nil
}

func (c *compositeDiscoverChannel) Destroy() error {
	for i := range c.chain {
		if err := c.chain[i].Destroy(); err != nil {
			return err
		}
	}
	return nil
}

// PublishEvent Release a service event
func (c *compositeDiscoverChannel) PublishEvent(event model.InstanceEvent) {
	for i := range c.chain {
		c.chain[i].PublishEvent(event)
	}
}
