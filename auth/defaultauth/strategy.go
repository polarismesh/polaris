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

package defaultauth

import (
	"context"
	"fmt"
	"time"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	commontime "github.com/polarismesh/polaris-server/common/time"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/plugin"
	"github.com/polarismesh/polaris-server/store"
)

type (
	StrategyDetail2Api func(user *model.StrategyDetail) *api.AuthStrategy
)

var (
	StrategyFilterAttributes = map[string]int{
		"id":    1,
		"name":  1,
		"owner": 1,
	}
)

type authStrategyServer struct {
	storage store.Store
	history plugin.History
}

// newAthStrategyServer
func newAthStrategyServer(s store.Store) (*authStrategyServer, error) {
	svr := &authStrategyServer{
		storage: s,
	}

	return svr, svr.initialize()
}

func (svr *authStrategyServer) initialize() error {
	// 获取History插件，注意：插件的配置在bootstrap已经设置好
	svr.history = plugin.GetHistory()
	if svr.history == nil {
		log.Warnf("Not Found History Log Plugin")
	}

	return nil
}

// CreateStrategy
func (svr *authStrategyServer) CreateStrategy(ctx context.Context, req *api.AuthStrategy) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)

	if checkErrResp := checkCreateStrategy(req); checkErrResp != nil {
		return checkErrResp
	}

	strategy, err := svr.storage.GetStrategyDetailByName(req.GetName().GetValue())
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewAuthStrategyResponse(api.StoreLayerException, req)
	}

	if strategy != nil {
		return api.NewAuthStrategyResponse(api.ExistedResource, req)
	}

	data := createAuthStrategyModel(req)
	if err := svr.storage.AddStrategy(data); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	msg := fmt.Sprintf("create auth strategy: name=%v", req.GetName().GetValue())
	log.Info(msg, utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
	svr.RecordHistory(authStrategyRecordEntry(ctx, req, data, model.OCreate))

	out := &api.AuthStrategy{
		Name: req.GetName(),
	}

	return api.NewAuthStrategyResponse(api.ExecuteSuccess, out)
}

// UpdateStrategy
func (svr *authStrategyServer) UpdateStrategy(ctx context.Context, req *api.AuthStrategy) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)

	if checkErrResp := checkCreateStrategy(req); checkErrResp != nil {
		return checkErrResp
	}

	strategy, err := svr.storage.GetStrategyDetail(req.GetId().GetValue())
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewAuthStrategyResponse(api.StoreLayerException, req)
	}

	if strategy == nil {
		return api.NewAuthStrategyResponse(api.NotFoundResource, req)
	}

	data := createAuthStrategyModel(req)
	if err := svr.storage.UpdateStrategyMain(data); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	msg := fmt.Sprintf("update auth strategy: name=%v", req.GetName().GetValue())
	log.Info(msg, utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
	svr.RecordHistory(authStrategyRecordEntry(ctx, req, data, model.OUpdate))

	out := &api.AuthStrategy{
		Name: req.GetName(),
	}

	return api.NewAuthStrategyResponse(api.ExecuteSuccess, out)
}

// DeleteStrategy
func (svr *authStrategyServer) DeleteStrategy(ctx context.Context, req *api.AuthStrategy) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)

	if checkErrResp := checkCreateStrategy(req); checkErrResp != nil {
		return checkErrResp
	}

	strategy, err := svr.storage.GetStrategyDetail(req.GetId().GetValue())
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewAuthStrategyResponse(api.StoreLayerException, req)
	}

	if strategy == nil {
		return api.NewAuthStrategyResponse(api.ExecuteSuccess, req)
	}

	if err := svr.storage.DeleteStrategy(req.GetId().GetValue()); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	msg := fmt.Sprintf("delete auth strategy: name=%v", req.GetName().GetValue())
	log.Info(msg, utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
	svr.RecordHistory(authStrategyRecordEntry(ctx, req, strategy, model.ODelete))

	out := &api.AuthStrategy{
		Name: req.GetName(),
	}

	return api.NewAuthStrategyResponse(api.ExecuteSuccess, out)
}

// ListStrategy
func (svr *authStrategyServer) ListStrategy(ctx context.Context, query map[string]string) *api.BatchQueryResponse {
	searchFilters := make(map[string]string)
	for key, value := range query {
		if _, ok := StrategyFilterAttributes[key]; !ok {
			log.Errorf("[Auth][AuthStrategy][Query] attribute(%s) it not allowed", key)
			return api.NewBatchQueryResponseWithMsg(api.InvalidParameter, key+" is not allowed")
		}
		searchFilters[key] = value
	}

	offset, limit, err := utils.ParseOffsetAndLimit(searchFilters)
	if err != nil {
		return api.NewBatchQueryResponse(api.InvalidParameter)
	}

	total, strategies, err := svr.storage.ListStrategyDetails(searchFilters, offset, limit)
	if err != nil {
		log.Errorf("[Auth][AuthStrategy][Query] req(%+v) store err: %s", query, err.Error())
		return api.NewBatchQueryResponse(api.StoreLayerException)
	}

	resp := api.NewBatchQueryResponse(api.ExecuteSuccess)
	resp.Amount = utils.NewUInt32Value(total)
	resp.Size = utils.NewUInt32Value(uint32(len(strategies)))
	resp.AuthStrategy = enhancedAuthStrategy2Api(strategies, authStrategy2Api)
	return resp
}

// AddStrategyResources
func (svr *authStrategyServer) AddStrategyResources(ctx context.Context, req *api.StrategyResource) *api.BatchWriteResponse {
	return nil
}

// DeleteStrategyResources
func (svr *authStrategyServer) DeleteStrategyResources(ctx context.Context, req *api.StrategyResource) *api.BatchWriteResponse {
	return nil
}

// RecordHistory server对外提供history插件的简单封装
func (svr *authStrategyServer) RecordHistory(entry *model.RecordEntry) {
	// 如果插件没有初始化，那么不记录history
	if svr.history == nil {
		return
	}
	// 如果数据为空，则不需要打印了
	if entry == nil {
		return
	}

	// 调用插件记录history
	svr.history.Record(entry)
}

// enhancedAuthStrategy2Api
func enhancedAuthStrategy2Api(datas []*model.StrategyDetail, apply StrategyDetail2Api) []*api.AuthStrategy {
	out := make([]*api.AuthStrategy, 0, len(datas))
	for _, entry := range datas {
		item := apply(entry)
		out = append(out, item)
	}

	return out
}

// authStrategy2Api
func authStrategy2Api(data *model.StrategyDetail) *api.AuthStrategy {
	if data == nil {
		return nil
	}

	// note: 不包括token，token比较特殊
	out := &api.AuthStrategy{
		Id:      utils.NewStringValue(data.ID),
		Name:    utils.NewStringValue(data.Name),
		Owner:   utils.NewStringValue(data.Owner),
		Comment: utils.NewStringValue(data.Comment),
		Ctime:   utils.NewStringValue(commontime.Time2String(data.CreateTime)),
		Mtime:   utils.NewStringValue(commontime.Time2String(data.ModifyTime)),
	}

	return out
}

// createAuthStrategyModel
func createAuthStrategyModel(strategy *api.AuthStrategy) *model.StrategyDetail {
	return nil
}

// checkCreateStrategy
func checkCreateStrategy(req *api.AuthStrategy) *api.Response {
	return nil
}

// authStrategyRecordEntry
func authStrategyRecordEntry(ctx context.Context, req *api.AuthStrategy, md *model.StrategyDetail,
	operationType model.OperationType) *model.RecordEntry {
	entry := &model.RecordEntry{
		ResourceType:  model.RAuthStrategy,
		UserGroup:     md.Name,
		OperationType: operationType,
		Operator:      utils.ParseOperator(ctx),
		CreateTime:    time.Now(),
	}

	return entry
}
