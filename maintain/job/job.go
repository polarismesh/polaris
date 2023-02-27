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
)

var (
	jobs        map[string]MaintainJob = map[string]MaintainJob{}
	startedJobs map[string]MaintainJob = map[string]MaintainJob{}
	scheduler                          = newCron()
)

func init() {

}

// StartMaintainJobs
func StartMaintianJobs(configs []JobConfig) error {
	for _, cfg := range configs {
		job, ok := jobs[cfg.Name]
		if !ok {
			return fmt.Errorf("[Maintain][Job] job (%s) not exist", cfg.Name)
		}
		_, ok = startedJobs[cfg.Name]
		if ok {
			return fmt.Errorf("[Maintain][Job] job (%s) duplicated", cfg.Name)
		}
		job.Init(cfg.Option)
		_, err := scheduler.AddFunc(cfg.CronSpec, job.Execute())
		if err != nil {
			return fmt.Errorf("[Maintain][Job] job (%s) fail to start", cfg.Name)
		}
		startedJobs[cfg.Name] = job
	}
	scheduler.Start()
	return nil
}

// StopMaintainJobs
func StopMaintainJobs() {
	_ = scheduler.Stop()
	startedJobs = map[string]MaintainJob{}
}

func newCron() *cron.Cron {
	return cron.New(cron.WithChain(
		cron.Recover(cron.DefaultLogger)),
		cron.WithParser(cron.NewParser(
			cron.Minute|cron.Hour|cron.Dom|cron.Month|cron.Dow|cron.Descriptor)))
}

func registerJob(job MaintainJob) {
	jobs[job.Name()] = job
}

type MaintainJob interface {
	Init(cfg map[string]interface{})
	Execute() func()
	Name() string
}
