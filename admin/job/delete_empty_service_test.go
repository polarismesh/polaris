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

	"github.com/polarismesh/polaris/common/model"
)

func Test_DeleteEmptyAutoCreatedServiceJobConfigInit(t *testing.T) {
	expectValue := 1 * time.Minute
	raw := map[string]interface{}{
		"serviceDeleteTimeout": "1m",
	}

	job := deleteEmptyServiceJob{}
	err := job.init(raw)
	if err != nil {
		t.Errorf("init deleteEmptyServiceJob config, err: %v", err)
	}

	if job.cfg.ServiceDeleteTimeout != expectValue {
		t.Errorf("init deleteEmptyServiceJob config. expect: %s, actual: %s",
			expectValue, job.cfg.ServiceDeleteTimeout)
	}
}

func Test_DeleteEmptyAutoCreatedServiceJobConfigInitErr(t *testing.T) {
	raw := map[string]interface{}{
		"serviceDeleteTimeout": "xx",
	}

	job := deleteEmptyServiceJob{}
	err := job.init(raw)
	if err == nil {
		t.Errorf("init deleteEmptyServiceJob config should err")
	}
}

func Test_FilterToDeletedServices(t *testing.T) {
	job := deleteEmptyServiceJob{}
	t1, _ := time.Parse("2006-01-02 15:04:05", "2023-03-20 12:01:00")
	t2, _ := time.Parse("2006-01-02 15:04:05", "2023-03-20 12:02:00")
	job.emptyServices = map[string]time.Time{
		"a": t1,
		"b": t2,
	}

	services := []*model.Service{
		{
			ID: "a",
		},
		{
			ID: "b",
		},
		{
			ID: "c",
		},
	}

	now, _ := time.Parse("2006-01-02 15:04:05", "2023-03-20 12:03:00")
	toDeleteServices := job.filterToDeletedServices(services, now, time.Minute)
	if len(toDeleteServices) != 1 {
		t.Errorf("one service should be deleted")
	}
	if toDeleteServices[0].ID != "a" {
		t.Errorf("to deleted service. expect: %s, actual: %s", "a", toDeleteServices[0].ID)
	}

	if len(job.emptyServices) != 2 {
		t.Errorf("two service should be candicated, actual: %v", job.emptyServices)
	}
	svcBTime := job.emptyServices["b"]
	if svcBTime != t2 {
		t.Errorf("empty service record time. expect: %s, actual: %s", t2, svcBTime)
	}
}
