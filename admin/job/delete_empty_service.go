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

	"github.com/polarismesh/polaris/cache"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/service"
	"github.com/polarismesh/polaris/store"
)

type DeleteEmptyServiceJobConfig struct {
	ServiceDeleteTimeout time.Duration `mapstructure:"serviceDeleteTimeout"`
}

type deleteEmptyServiceJob struct {
	cfg           *DeleteEmptyServiceJobConfig
	namingServer  service.DiscoverServer
	cacheMgn      *cache.CacheManager
	storage       store.Store
	emptyServices map[string]time.Time
}

func (job *deleteEmptyServiceJob) init(raw map[string]interface{}) error {
	cfg := &DeleteEmptyServiceJobConfig{
		ServiceDeleteTimeout: 30 * time.Minute,
	}
	decodeConfig := &mapstructure.DecoderConfig{
		DecodeHook: mapstructure.StringToTimeDurationHookFunc(),
		Result:     cfg,
	}
	decoder, err := mapstructure.NewDecoder(decodeConfig)
	if err != nil {
		log.Errorf("[Maintain][Job][DeleteEmptyServiceJob] new config decoder err: %v", err)
		return err
	}
	err = decoder.Decode(raw)
	if err != nil {
		log.Errorf("[Maintain][Job][DeleteEmptyServiceJob] parse config err: %v", err)
		return err
	}
	job.cfg = cfg
	job.emptyServices = map[string]time.Time{}
	return nil
}

func (job *deleteEmptyServiceJob) execute() {
	err := job.deleteEmptyServices()
	if err != nil {
		log.Errorf("[Maintain][Job][DeleteEmptyServiceJob] delete empty autocreated services, err: %v", err)
	}
}

func (job *deleteEmptyServiceJob) interval() time.Duration {
	return job.cfg.ServiceDeleteTimeout
}

func (job *deleteEmptyServiceJob) clear() {
	job.emptyServices = map[string]time.Time{}
}

func (job *deleteEmptyServiceJob) getEmptyServices() []*model.Service {
	services := job.getAllEmptyServices()
	return job.filterToDeletedServices(services, time.Now(), job.cfg.ServiceDeleteTimeout)
}

func (job *deleteEmptyServiceJob) getAllEmptyServices() []*model.Service {
	var res []*model.Service
	_ = job.cacheMgn.Service().IteratorServices(func(key string, svc *model.Service) (bool, error) {
		if svc.IsAlias() {
			return true, nil
		}
		count := job.cacheMgn.Instance().GetInstancesCountByServiceID(svc.ID)
		if count.TotalInstanceCount == 0 {
			res = append(res, svc)
		}
		return true, nil
	})
	return res
}

func (job *deleteEmptyServiceJob) filterToDeletedServices(services []*model.Service,
	now time.Time, timeout time.Duration) []*model.Service {
	var toDeleteServices []*model.Service
	m := map[string]time.Time{}
	for _, svc := range services {
		value, ok := job.emptyServices[svc.ID]
		if !ok {
			m[svc.ID] = now
			continue
		}
		if now.After(value.Add(timeout)) {
			toDeleteServices = append(toDeleteServices, svc)
		} else {
			m[svc.ID] = value
		}
	}
	job.emptyServices = m

	return toDeleteServices
}

func (job *deleteEmptyServiceJob) deleteEmptyServices() error {
	emptyServices := job.getEmptyServices()

	deleteBatchSize := 100
	for i := 0; i < len(emptyServices); i += deleteBatchSize {
		j := i + deleteBatchSize
		if j > len(emptyServices) {
			j = len(emptyServices)
		}

		ctx, err := buildContext(job.storage)
		if err != nil {
			log.Errorf("[Maintain][Job][DeleteUnHealthyInstance] build conetxt, err: %v", err)
			return err
		}
		resp := job.namingServer.DeleteServices(ctx, convertDeleteServiceRequest(emptyServices[i:j]))
		if api.CalcCode(resp) != 200 {
			log.Errorf("[Maintain][Job][DeleteEmptyAutoCreatedService] delete services err, code: %d, info: %s",
				resp.Code.GetValue(), resp.Info.GetValue())
		}
	}

	log.Infof("[Maintain][Job][DeleteEmptyAutoCreatedService] delete empty auto-created services count %d",
		len(emptyServices))
	return nil
}

func convertDeleteServiceRequest(infos []*model.Service) []*apiservice.Service {
	var entries = make([]*apiservice.Service, len(infos))
	for i, info := range infos {
		entries[i] = &apiservice.Service{
			Namespace: utils.NewStringValue(info.Namespace),
			Name:      utils.NewStringValue(info.Name),
		}
	}
	return entries
}
