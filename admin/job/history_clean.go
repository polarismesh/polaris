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

// 默认保存配置发布天数
const defaultHistoryRetentionDays = 7 * 24 * time.Hour

type CleanConfigFileHistoryJobConfig struct {
	RetentionDays time.Duration `mapstructure:"retentionDays"`
	BatchSize     uint64        `mapstructure:"batchSize"`
}

type cleanConfigFileHistoryJob struct {
	cfg     *CleanConfigFileHistoryJobConfig
	storage store.Store
}

func (job *cleanConfigFileHistoryJob) init(raw map[string]interface{}) error {
	cfg := &CleanConfigFileHistoryJobConfig{
		RetentionDays: defaultHistoryRetentionDays,
		BatchSize:     1000,
	}
	decodeConfig := &mapstructure.DecoderConfig{
		DecodeHook: mapstructure.StringToTimeDurationHookFunc(),
		Result:     cfg,
	}
	decoder, err := mapstructure.NewDecoder(decodeConfig)
	if err != nil {
		log.Errorf("[Maintain][Job][cleanConfigFileHistoryJob] new config decoder err: %v", err)
		return err
	}
	if err = decoder.Decode(raw); err != nil {
		log.Errorf("[Maintain][Job][cleanConfigFileHistoryJob] parse config err: %v", err)
		return err
	}
	if cfg.RetentionDays < time.Minute {
		cfg.RetentionDays = time.Minute
	}
	job.cfg = cfg
	return nil
}

func (job *cleanConfigFileHistoryJob) execute() {
	endTime := time.Now().Add(-1 * job.cfg.RetentionDays)
	if err := job.storage.CleanConfigFileReleaseHistory(endTime, job.cfg.BatchSize); err != nil {
		log.Errorf("[Maintain][Job][cleanConfigFileHistoryJob] execute err: %v", err)
	}
}

func (job *cleanConfigFileHistoryJob) interval() time.Duration {
	return time.Minute
}

func (job *cleanConfigFileHistoryJob) clear() {
}
