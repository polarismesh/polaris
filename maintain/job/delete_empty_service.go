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
	"context"
	"fmt"
	"strconv"

	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/service"
)

type deleteEmptyAutoCreatedServiceJob struct {
	namingServer service.DiscoverServer
}

func (job *deleteEmptyAutoCreatedServiceJob) init(raw map[string]interface{}) error {
	return nil
}

func (job *deleteEmptyAutoCreatedServiceJob) execute() {
	err := job.deleteEmptyAutoCreatedServices()
	if err != nil {
		log.Errorf("[Maintain][Job][DeleteEmptyAutoCreatedService] delete empty autocreated services, err: %v", err)
	}
}

func (job *deleteEmptyAutoCreatedServiceJob) getEmptyAutoCreatedServices() ([]*apiservice.Service, error) {
	var emtpyServices []*apiservice.Service
	var offset uint32 = 0
	var query = map[string]string{
		"keys":   service.MetadataInternalAutoCreated,
		"values": "true",
		"limit":  "100",
		"offset": strconv.Itoa(int(offset))}

	for {
		resp := job.namingServer.GetServices(context.Background(), query)
		log.Infof("recv %v", resp)
		if api.CalcCode(resp) != 200 {
			return nil, fmt.Errorf("GetServices err, code: %d, info: %s", resp.Code.GetValue(), resp.Info.GetValue())
		}

		for _, entry := range resp.Services {
			if entry.TotalInstanceCount.GetValue() > 0 {
				continue
			}
			emtpyServices = append(emtpyServices, entry)
		}

		nextOffset := offset + resp.Size.GetValue()
		if nextOffset >= resp.Amount.GetValue() {
			break
		}
		offset = nextOffset
		query["offset"] = strconv.Itoa(int(offset))
	}
	return emtpyServices, nil
}

func (job *deleteEmptyAutoCreatedServiceJob) deleteEmptyAutoCreatedServices() error {
	emptyServices, err := job.getEmptyAutoCreatedServices()
	if err != nil {
		return err
	}

	deleteBatchSize := 100
	for i := 0; i < len(emptyServices); i += deleteBatchSize {
		j := i + deleteBatchSize
		if j > len(emptyServices) {
			j = len(emptyServices)
		}

		resp := job.namingServer.DeleteServices(context.Background(), emptyServices[i:j])
		if api.CalcCode(resp) != 200 {
			log.Errorf("[Maintain][Job][DeleteEmptyAutoCreatedService] delete services err, code: %d, info: %s",
				resp.Code.GetValue(), resp.Info.GetValue())
		}
	}

	log.Infof("[Maintain][Job][DeleteEmptyAutoCreatedService] delete empty auto-created services count %d",
		len(emptyServices))
	return nil
}
