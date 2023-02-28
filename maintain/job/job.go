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
	"fmt"

	"github.com/robfig/cron/v3"

	commonlog "github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/store"
)

var log = commonlog.GetScopeOrDefaultByName(commonlog.DefaultLoggerName)

// MaintainJobs
type MaintainJobs struct {
	jobs        map[string]maintainJob
	startedJobs map[string]maintainJob
	scheduler   *cron.Cron
}

// NewMaintainJobs
func NewMaintainJobs(storage store.Store) *MaintainJobs {
	return &MaintainJobs{
		jobs: map[string]maintainJob{
			"DeleteUnHealthyInstance": &deleteUnHealthyInstanceJob{storage: storage},
		},
		startedJobs: map[string]maintainJob{},
		scheduler:   newCron(),
	}
}

// StartMaintainJobs
func (mj *MaintainJobs) StartMaintianJobs(configs []JobConfig) error {
	for _, cfg := range configs {
		job, ok := mj.jobs[cfg.Name]
		if !ok {
			return fmt.Errorf("[Maintain][Job] job (%s) not exist", cfg.Name)
		}
		_, ok = mj.startedJobs[cfg.Name]
		if ok {
			return fmt.Errorf("[Maintain][Job] job (%s) duplicated", cfg.Name)
		}
		err := job.init(cfg.Option)
		if err != nil {
			log.Errorf("[Maintain][Job] job (%s) fail to init, err: %v", cfg.Name, err)
			return fmt.Errorf("[Maintain][Job] job (%s) fail to init", cfg.Name)
		}
		_, err = mj.scheduler.AddFunc(cfg.CronSpec, func() { job.execute() })
		if err != nil {
			log.Errorf("[Maintain][Job] job (%s) fail to start, err: %v", cfg.Name, err)
			return fmt.Errorf("[Maintain][Job] job (%s) fail to start", cfg.Name)
		}
		mj.startedJobs[cfg.Name] = job
	}
	mj.scheduler.Start()
	return nil
}

// StopMaintainJobs
func (mj *MaintainJobs) StopMaintainJobs() {
	ctx := mj.scheduler.Stop()
	<-ctx.Done()
	mj.startedJobs = map[string]maintainJob{}
}

func newCron() *cron.Cron {
	return cron.New(cron.WithChain(
		cron.Recover(cron.DefaultLogger)),
		cron.WithParser(cron.NewParser(
			cron.Minute|cron.Hour|cron.Dom|cron.Month|cron.Dow|cron.Descriptor)))
}

type maintainJob interface {
	init(cfg map[string]interface{}) error
	execute()
}
