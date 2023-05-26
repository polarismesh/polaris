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
	"time"

	"github.com/mitchellh/mapstructure"

	"github.com/polarismesh/polaris/store"
)

type CleanDeletedInstancesJobConfig struct {
	InstanceCleanTimeout time.Duration `mapstructure:"instanceCleanTimeout"`
}

type cleanDeletedInstancesJob struct {
	cfg     *CleanDeletedInstancesJobConfig
	storage store.Store
}

func (job *cleanDeletedInstancesJob) init(raw map[string]interface{}) error {
	cfg := &CleanDeletedInstancesJobConfig{
		InstanceCleanTimeout: 10 * time.Minute,
	}
	decodeConfig := &mapstructure.DecoderConfig{
		DecodeHook: mapstructure.StringToTimeDurationHookFunc(),
		Result:     cfg,
	}
	decoder, err := mapstructure.NewDecoder(decodeConfig)
	if err != nil {
		log.Errorf("[Maintain][Job][CleanDeletedInstances] new config decoder err: %v", err)
		return err
	}
	if err = decoder.Decode(raw); err != nil {
		log.Errorf("[Maintain][Job][CleanDeletedInstances] parse config err: %v", err)
		return err
	}
	if cfg.InstanceCleanTimeout < 2*time.Minute {
		cfg.InstanceCleanTimeout = 2 * time.Minute
	}
	job.cfg = cfg
	return nil
}

func (job *cleanDeletedInstancesJob) execute() {
	batchSize := uint32(100)
	for {
		count, err := job.storage.BatchCleanDeletedInstances(job.cfg.InstanceCleanTimeout, batchSize)
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

func (job *cleanDeletedInstancesJob) interval() time.Duration {
	return job.cfg.InstanceCleanTimeout
}

func (job *cleanDeletedInstancesJob) clear() {
}
