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

type CleanDeletedClientsJobConfig struct {
	ClientCleanTimeout time.Duration `mapstructure:"clientCleanTimeout"`
}

type cleanDeletedClientsJob struct {
	cfg     *CleanDeletedClientsJobConfig
	storage store.Store
}

func (job *cleanDeletedClientsJob) init(raw map[string]interface{}) error {
	cfg := &CleanDeletedClientsJobConfig{
		ClientCleanTimeout: 10 * time.Minute,
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
	err = decoder.Decode(raw)
	if err != nil {
		log.Errorf("[Maintain][Job][CleanDeletedClients] parse config err: %v", err)
		return err
	}
	job.cfg = cfg

	return nil
}

func (job *cleanDeletedClientsJob) execute() {
	batchSize := uint32(100)
	for {
		count, err := job.storage.BatchCleanDeletedClients(job.cfg.ClientCleanTimeout, batchSize)
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

func (job *cleanDeletedClientsJob) clear() {
}

func (job *cleanDeletedClientsJob) interval() time.Duration {
	return job.cfg.ClientCleanTimeout
}
