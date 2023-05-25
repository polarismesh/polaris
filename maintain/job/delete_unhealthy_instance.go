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
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/service"
	"github.com/polarismesh/polaris/store"
)

type DeleteUnHealthyInstanceJobConfig struct {
	InstanceDeleteTimeout time.Duration `mapstructure:"instanceDeleteTimeout"`
}

type deleteUnHealthyInstanceJob struct {
	cfg          *DeleteUnHealthyInstanceJobConfig
	namingServer service.DiscoverServer
	storage      store.Store
}

func (job *deleteUnHealthyInstanceJob) init(raw map[string]interface{}) error {
	cfg := &DeleteUnHealthyInstanceJobConfig{
		InstanceDeleteTimeout: 60 * time.Minute,
	}
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
	return nil
}

func (job *deleteUnHealthyInstanceJob) interval() time.Duration {
	return job.cfg.InstanceDeleteTimeout
}

func (job *deleteUnHealthyInstanceJob) execute() {
	batchSize := uint32(100)
	var count int = 0
	for {
		instanceIds, err := job.storage.GetUnHealthyInstances(job.cfg.InstanceDeleteTimeout, batchSize)
		if err != nil {
			log.Errorf("[Maintain][Job][DeleteUnHealthyInstance] get unhealthy instances, err: %v", err)
			break
		}
		if len(instanceIds) == 0 {
			break
		}

		var req []*apiservice.Instance
		for _, id := range instanceIds {
			req = append(req, &apiservice.Instance{Id: utils.NewStringValue(id)})
		}

		ctx, err := buildContext(job.storage)
		if err != nil {
			log.Errorf("[Maintain][Job][DeleteUnHealthyInstance] build conetxt, err: %v", err)
			return
		}
		resp := job.namingServer.DeleteInstances(ctx, req)
		if api.CalcCode(resp) == 200 {
			log.Infof("[Maintain][Job][DeleteUnHealthyInstance] delete instance count %d, list: %v",
				len(instanceIds), instanceIds)
		} else {
			log.Errorf("[Maintain][Job][DeleteUnHealthyInstance] delete instance list: %v, err: %d %s",
				instanceIds, resp.Code.GetValue(), resp.Info.GetValue())
			break
		}
		count += len(instanceIds)
	}

	log.Infof("[Maintain][Job][DeleteUnHealthyInstance] delete unhealthy instance count %d", count)

}

func (job *deleteUnHealthyInstanceJob) clear() {
}
