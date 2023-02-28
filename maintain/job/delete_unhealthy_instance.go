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

type deleteUnHealthyInstanceJobConfig struct {
	instanceDeleteTimeout time.Duration `mapstructure:"instanceDeleteTimeout"`
}

type deleteUnHealthyInstanceJob struct {
	cfg     *deleteUnHealthyInstanceJobConfig
	storage store.Store
}

func (job *deleteUnHealthyInstanceJob) init(raw map[string]interface{}) error {
	cfg := &deleteUnHealthyInstanceJobConfig{}
	decodeConfig := &mapstructure.DecoderConfig{
		DecodeHook: mapstructure.StringToTimeDurationHookFunc(),
		Result:     cfg,
	}
	decoder, err := mapstructure.NewDecoder(decodeConfig)
	if err != nil {
		log.Errorf("[Maintain][Job][DeleteUnHealthyInstance] new config decoder err: %v", err)
		return err
	}
	err = decoder.Decode(raw)
	if err != nil {
		log.Errorf("[Maintain][Job][DeleteUnHealthyInstance] parse config err: %v", err)
		return err
	}
	job.cfg = cfg

	err = job.storage.StartLeaderElection(store.ELECTION_KEY_MAINTAIN_JOB_DELETE_UNHEALTHY_INSTANCE)
	if err != nil {
		log.Errorf("[Maintain][Job][DeleteUnHealthyInstance] start leader election err: %v", err)
		return err
	}

	return nil
}

func (job *deleteUnHealthyInstanceJob) execute() {
	if !job.storage.IsLeader(store.ELECTION_KEY_MAINTAIN_JOB_DELETE_UNHEALTHY_INSTANCE) {
		log.Info("[Maintain][Job][DeleteUnHealthyInstance] I am follower")
		return
	}

	log.Info("[Maintain][Job][DeleteUnHealthyInstance] I am leader, execute job")

}
