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

package leader

import (
	"encoding/json"
	"time"

	"github.com/polarismesh/polaris/common/batchjob"
)

type Config struct {
	SoltNum int32
	Batch   batchjob.CtrlConfig
}

func Unmarshal(options map[string]interface{}) (*Config, error) {
	contentBytes, err := json.Marshal(options)
	if err != nil {
		return nil, err
	}

	config := &Config{
		SoltNum: DefaultSoltNum,
		Batch: batchjob.CtrlConfig{
			QueueSize:     10240,
			WaitTime:      128 * time.Millisecond,
			MaxBatchCount: 128,
			Concurrency:   64,
		},
	}
	if err := json.Unmarshal(contentBytes, config); err != nil {
		return nil, err
	}
	return config, nil
}
