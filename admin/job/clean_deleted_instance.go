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

	"github.com/polarismesh/polaris/store"
)

type cleanDeletedInstancesJob struct {
	storage store.Store
}

func (job *cleanDeletedInstancesJob) init(raw map[string]interface{}) error {
	return nil
}

func (job *cleanDeletedInstancesJob) execute() {
	batchSize := uint32(100)
	mtime := time.Now().Add(-10 * time.Minute)
	for {
		count, err := job.storage.BatchCleanDeletedInstances(mtime, batchSize)
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

func (job *cleanDeletedInstancesJob) clear() {
}
