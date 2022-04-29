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

package service

import (
	"context"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
)

var (
	clientFilterAttributes = map[string]struct{}{
		"type":    {},
		"host":    {},
		"limit":   {},
		"offset":  {},
		"version": {},
	}
)

// CreateInstances create one instance
func (s *Server) GetReportClients(ctx context.Context, query map[string]string) *api.BatchQueryResponse {
	searchFilters := make(map[string]string)
	var (
		offset, limit uint32
		err           error
	)

	for key, value := range query {
		if _, ok := clientFilterAttributes[key]; !ok {
			log.Errorf("[Server][Client] attribute(%s) it not allowed", key)
			return api.NewBatchQueryResponseWithMsg(api.InvalidParameter, key+" is not allowed")
		}
		searchFilters[key] = value
	}

	var (
		total   uint32
		clients []*model.Client
	)

	offset, limit, err = utils.ParseOffsetAndLimit(searchFilters)
	if err != nil {
		return api.NewBatchQueryResponse(api.InvalidParameter)
	}

	total, services, err := s.caches.Client().GetClientsByFilter(searchFilters, offset, limit)
	if err != nil {
		log.Errorf("[Server][Client][Query] req(%+v) store err: %s", query, err.Error())
		return api.NewBatchQueryResponse(api.StoreLayerException)
	}

	resp := api.NewBatchQueryResponse(api.ExecuteSuccess)
	resp.Amount = utils.NewUInt32Value(total)
	resp.Size = utils.NewUInt32Value(uint32(len(services)))
	resp.Clients = enhancedClients2Api(clients, client2Api)
	return resp
}

type Client2Api func(client *model.Client) *api.Client

// client 数组转为[]*api.Client
func enhancedClients2Api(clients []*model.Client, handler Client2Api) []*api.Client {
	out := make([]*api.Client, 0, len(clients))
	for _, entry := range clients {
		outUser := handler(entry)
		out = append(out, outUser)
	}
	return out
}

// model.Client 转为 api.Client
func client2Api(client *model.Client) *api.Client {
	if client == nil {
		return nil
	}
	out := client.Proto()
	return out
}
