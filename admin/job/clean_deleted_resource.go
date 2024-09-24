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

package job

import (
	"sync"
	"time"

	"github.com/mitchellh/mapstructure"

	"github.com/polarismesh/polaris/store"
)

var cleanFuncMapping = map[string]func(timeout time.Duration, job *cleanDeletedResourceJob){
	"instance": cleanDeletedInstances,
	"service":  cleanDeletedServices,
	"clients":  cleanDeletedClients,
	"circuitbreaker_rule": func(timeout time.Duration, job *cleanDeletedResourceJob) {
		cleanDeletedRules("circuitbreaker_rule", timeout, job)
	},
	"ratelimit_rule": func(timeout time.Duration, job *cleanDeletedResourceJob) {
		cleanDeletedRules("ratelimit_rule", timeout, job)
	},
	"router_rule": func(timeout time.Duration, job *cleanDeletedResourceJob) {
		cleanDeletedRules("router_rule", timeout, job)
	},
	"faultdetect_rule": func(timeout time.Duration, job *cleanDeletedResourceJob) {
		cleanDeletedRules("faultdetect_rule", timeout, job)
	},
	"lane_rule": func(timeout time.Duration, job *cleanDeletedResourceJob) {
		cleanDeletedRules("lane_rule", timeout, job)
	},
	"config_file_release": cleanDeletedConfigFiles,
}

type CleanDeletedResource struct {
	// Resource 记录需要清理的资源类型
	Resource string `mapstructure:"resource"`
	// Timeout 记录资源的额外超时时间，用户可自定义
	Timeout *time.Duration `mapstructure:"timeout"`
	// Enable 记录是否开启清理
	Enable bool `mapstructure:"enable"`
}

type CleandeletedResourceConf struct {
	// ResourceTimeout 记录资源的额外超时时间，用户可自定义
	Resources []CleanDeletedResource `json:"resourceTimeout"`
	// Timeout 记录清理资源的超时时间，默认20分钟
	Timeout time.Duration `mapstructure:"timeout"`
}

type cleanDeletedResourceJob struct {
	cfg     *CleandeletedResourceConf
	storage store.Store
}

func (job *cleanDeletedResourceJob) init(raw map[string]interface{}) error {
	cfg := &CleandeletedResourceConf{
		Timeout: 20 * time.Minute,
	}
	decodeConfig := &mapstructure.DecoderConfig{
		DecodeHook: mapstructure.StringToTimeDurationHookFunc(),
		Result:     cfg,
	}
	decoder, err := mapstructure.NewDecoder(decodeConfig)
	if err != nil {
		log.Errorf("[Maintain][Job][CleanDeletedClients] new config decoder err: %v", err)
		return err
	}
	if err := decoder.Decode(raw); err != nil {
		log.Errorf("[Maintain][Job][CleanDeletedClients] parse config err: %v", err)
		return err
	}
	if cfg.Timeout < 2*time.Minute {
		cfg.Timeout = 2 * time.Minute
	}
	job.cfg = cfg
	return nil
}

func (job *cleanDeletedResourceJob) execute() {
	wait := &sync.WaitGroup{}
	for _, resource := range job.cfg.Resources {
		if !resource.Enable {
			continue
		}
		timeout := job.cfg.Timeout
		if resource.Timeout != nil {
			timeout = *resource.Timeout
		}
		if cleanFunc, ok := cleanFuncMapping[resource.Resource]; ok {
			wait.Add(1)
			go func(timeout time.Duration, job *cleanDeletedResourceJob) {
				defer wait.Done()
				cleanFunc(timeout, job)
			}(timeout, job)
		}
	}
	wait.Wait()
}

func (job *cleanDeletedResourceJob) clear() {
}

func (job *cleanDeletedResourceJob) interval() time.Duration {
	return time.Minute
}

func cleanDeletedConfigFiles(timeout time.Duration, job *cleanDeletedResourceJob) {
	batchSize := uint32(100)
	for {
		count, err := job.storage.BatchCleanDeletedConfigFiles(timeout, batchSize)
		if err != nil {
			log.Errorf("[Maintain][Job][CleanDeletedClients] batch clean deleted client, err: %v", err)
			break
		}
		log.Infof("[Maintain][Job][CleanDeletedClients] clean deleted client count %d", count)
		if count < batchSize {
			break
		}
	}
}

func cleanDeletedServices(timeout time.Duration, job *cleanDeletedResourceJob) {
	batchSize := uint32(100)
	for {
		count, err := job.storage.BatchCleanDeletedServices(timeout, batchSize)
		if err != nil {
			log.Errorf("[Maintain][Job][CleanDeletedClients] batch clean deleted client, err: %v", err)
			break
		}
		log.Infof("[Maintain][Job][CleanDeletedClients] clean deleted client count %d", count)
		if count < batchSize {
			break
		}
	}
}

func cleanDeletedClients(timeout time.Duration, job *cleanDeletedResourceJob) {
	batchSize := uint32(100)
	for {
		count, err := job.storage.BatchCleanDeletedClients(timeout, batchSize)
		if err != nil {
			log.Errorf("[Maintain][Job][CleanDeletedClients] batch clean deleted client, err: %v", err)
			break
		}
		log.Infof("[Maintain][Job][CleanDeletedClients] clean deleted client count %d", count)
		if count < batchSize {
			break
		}
	}
}

func cleanDeletedInstances(timeout time.Duration, job *cleanDeletedResourceJob) {
	batchSize := uint32(100)
	for {
		count, err := job.storage.BatchCleanDeletedInstances(timeout, batchSize)
		if err != nil {
			log.Errorf("[Maintain][Job][CleanDeletedInstances] batch clean deleted instance, err: %v", err)
			break
		}

		log.Infof("[Maintain][Job][CleanDeletedInstances] clean deleted instance count %d", count)
		if count < batchSize {
			break
		}
	}
}

func cleanDeletedRules(rule string, timeout time.Duration, job *cleanDeletedResourceJob) {
	batchSize := uint32(100)
	for {
		count, err := job.storage.BatchCleanDeletedRules(rule, timeout, batchSize)
		if err != nil {
			log.Errorf("[Maintain][Job][CleanDeletedClients] batch clean deleted client, err: %v", err)
			break
		}
		log.Infof("[Maintain][Job][CleanDeletedClients] clean deleted client count %d", count)
		if count < batchSize {
			break
		}
	}
}
