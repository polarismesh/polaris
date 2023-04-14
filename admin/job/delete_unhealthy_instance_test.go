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
	"testing"
	"time"
)

func Test_DeleteUnHealthyInstanceJobConfigInit(t *testing.T) {
	expectValue := 10 * time.Minute
	raw := map[string]interface{}{
		"instanceDeleteTimeout": "10m",
	}

	job := deleteUnHealthyInstanceJob{}
	err := job.init(raw)
	if err != nil {
		t.Errorf("init deleteUnHealthyInstanceJob config, err: %v", err)
	}

	if job.cfg.InstanceDeleteTimeout != expectValue {
		t.Errorf("init deleteUnHealthyInstanceJob config. expect: %s, actual: %s",
			expectValue, job.cfg.InstanceDeleteTimeout)
	}
}

func Test_DeleteUnHealthyInstanceJobConfigInitErr(t *testing.T) {
	raw := map[string]interface{}{
		"instanceDeleteTimeout": "xx",
	}

	job := deleteUnHealthyInstanceJob{}
	err := job.init(raw)
	if err == nil {
		t.Errorf("init deleteUnHealthyInstanceJob config should err")
	}
}
