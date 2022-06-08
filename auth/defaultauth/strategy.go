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
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/wrappers"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	commontime "github.com/polarismesh/polaris-server/common/time"
	"github.com/polarismesh/polaris-server/common/utils"
	"go.uber.org/zap"
)

type (
	// StrategyDetail2Api strategy detail to *api.AuthStrategy func
	StrategyDetail2Api func(user *model.StrategyDetail) *api.AuthStrategy
)

var (
	// StrategyFilterAttributes strategy filter attributes
	StrategyFilterAttributes = map[string]bool{
		"id":             true,
		"name":           true,
		"owner":          true,
		"offset":         true,
		"limit":          true,
		"principal_id":   true,
		"principal_type": true,
		"res_id":         true,
		"res_type":       true,
		"default":        true,
		"show_detail":    true,
	}
)

// CreateStrategy 创建鉴权策略
func (svr *server) CreateStrategy(ctx context.Context, req *api.AuthStrategy) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	ownerId := utils.ParseOwnerID(ctx)
	req.Owner = utils.NewStringValue(ownerId)

	if checkErrResp := svr.checkCreateStrategy(req); checkErrResp != nil {
		return checkErrResp
	}

	req.Resources = svr.normalizeResource(req.Resources)

	data := svr.createAuthStrategyModel(req)
	if err := svr.storage.AddStrategy(data); err != nil {
		log.AuthScope().Error("[Auth][Strategy] create strategy into store", utils.ZapRequestID(requestID),
			zap.Error(err))
		return api.NewResponse(StoreCode2APICode(err))
	}

	log.AuthScope().Info("[Auth][Strategy] create strategy", utils.ZapRequestID(requestID),
		zap.String("name", req.Name.GetValue()))
	svr.RecordHistory(authStrategyRecordEntry(ctx, req, data, model.OCreate))

	return api.NewAuthStrategyResponse(api.ExecuteSuccess, req)
}

// UpdateStrategies 批量修改鉴权
func (svr *server) UpdateStrategies(ctx context.Context, reqs []*api.ModifyAuthStrategy) *api.BatchWriteResponse {
	resp := api.NewBatchWriteResponse(api.ExecuteSuccess)

	for index := range reqs {
		ret := svr.UpdateStrategy(ctx, reqs[index])
		resp.Collect(ret)
	}

	return resp
}

// UpdateStrategy 实现鉴权策略的变更
// Case 1. 修改的是默认鉴权策略的话，只能修改资源，不能添加、删除用户 or 用户组
// Case 2. 鉴权策略只能被自己的 owner 对应的用户修改
func (svr *server) UpdateStrategy(ctx context.Context, req *api.ModifyAuthStrategy) *api.Response {
	requestID := utils.ParseRequestID(ctx)

	strategy, err := svr.storage.GetStrategyDetail(req.GetId().GetValue())
	if err != nil {
		log.AuthScope().Error("[Auth][Strategy] get strategy from store", utils.ZapRequestID(requestID),
			zap.Error(err))
		return api.NewModifyAuthStrategyResponse(api.StoreLayerException, req)
	}
	if strategy == nil {
		return api.NewModifyAuthStrategyResponse(api.NotFoundAuthStrategyRule, req)
	}

	if checkErrResp := svr.checkUpdateStrategy(ctx, req, strategy); checkErrResp != nil {
		return checkErrResp
	}

	req.AddResources = svr.normalizeResource(req.AddResources)
	data, needUpdate := svr.updateAuthStrategyAttribute(ctx, req, strategy)
	if !needUpdate {
		return api.NewModifyAuthStrategyResponse(api.NoNeedUpdate, req)
	}

	if err := svr.storage.UpdateStrategy(data); err != nil {
		log.AuthScope().Error("[Auth][Strategy] update strategy into store",
			utils.ZapRequestID(requestID), zap.Error(err))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	log.AuthScope().Info("[Auth][Strategy] update strategy into store", utils.ZapRequestID(requestID),
		zap.String("name", strategy.Name))
	svr.RecordHistory(authModifyStrategyRecordEntry(ctx, req, data, model.OUpdate))

	return api.NewModifyAuthStrategyResponse(api.ExecuteSuccess, req)
}

// DeleteStrategies 批量删除鉴权策略
func (svr *server) DeleteStrategies(ctx context.Context, reqs []*api.AuthStrategy) *api.BatchWriteResponse {
	resp := api.NewBatchWriteResponse(api.ExecuteSuccess)
	for index := range reqs {
		ret := svr.DeleteStrategy(ctx, reqs[index])
		resp.Collect(ret)
	}

	return resp
}

// DeleteStrategy 删除鉴权策略
// Case 1. 只有该策略的 owner 账户可以删除策略
// Case 2. 默认策略不能被删除，默认策略只能随着账户的删除而被清理
func (svr *server) DeleteStrategy(ctx context.Context, req *api.AuthStrategy) *api.Response {
	requestID := utils.ParseRequestID(ctx)

	strategy, err := svr.storage.GetStrategyDetail(req.GetId().GetValue())
	if err != nil {
		log.AuthScope().Error("[Auth][Strategy] get strategy from store", utils.ZapRequestID(requestID),
			zap.Error(err))
		return api.NewAuthStrategyResponse(api.StoreLayerException, req)
	}

	if strategy == nil {
		return api.NewAuthStrategyResponse(api.ExecuteSuccess, req)
	}

	if strategy.Default {
		log.AuthScope().Error("[Auth][Strategy] delete default strategy is denied", utils.ZapRequestID(requestID))
		return api.NewAuthStrategyResponseWithMsg(api.BadRequest, "default strategy can't delete", req)
	}

	if strategy.Owner != utils.ParseUserID(ctx) {
		return api.NewAuthStrategyResponse(api.NotAllowedAccess, req)
	}

	if err := svr.storage.DeleteStrategy(req.GetId().GetValue()); err != nil {
		log.AuthScope().Error("[Auth][Strategy] delete strategy from store",
			utils.ZapRequestID(requestID), zap.Error(err))
		return api.NewResponse(StoreCode2APICode(err))
	}

	log.AuthScope().Info("[Auth][Strategy] delete strategy from store", utils.ZapRequestID(requestID),
		zap.String("name", req.Name.GetValue()))
	svr.RecordHistory(authStrategyRecordEntry(ctx, req, strategy, model.ODelete))

	return api.NewAuthStrategyResponse(api.ExecuteSuccess, req)
}

// GetStrategies 查询鉴权策略列表
// Case 1. 如果是以资源视角来查询鉴权策略，那么就会忽略自动根据账户类型进行数据查看的限制
// 		eg. 比如当前子账户A想要查看资源R的相关的策略，那么不在会自动注入 principal_id 以及 principal_type 的查询条件
// Case 2. 如果是以用户视角来查询鉴权策略，如果没有带上 principal_id，那么就会根据账户类型自动注入 principal_id 以
// 		及 principal_type 的查询条件，从而限制该账户的数据查看
// 		eg.
// 			a. 如果当前是超级管理账户，则按照传入的 query 进行查询即可
// 			b. 如果当前是主账户，则自动注入 owner 字段，即只能查看策略的 owner 是自己的策略
// 			c. 如果当前是子账户，则自动注入 principal_id 以及 principal_type 字段，即稚嫩查询与自己有关的策略
func (svr *server) GetStrategies(ctx context.Context, query map[string]string) *api.BatchQueryResponse {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)

	log.AuthScope().Debug("[Auth][Strategy] origin get strategies query params", utils.ZapRequestID(requestID),
		utils.ZapPlatformID(platformID), zap.Any("query", query))

	showDetail := query["show_detail"]

	searchFilters := make(map[string]string, len(query))
	for key, value := range query {
		if _, ok := StrategyFilterAttributes[key]; !ok {
			log.AuthScope().Errorf("[Auth][Strategy] get strategies attribute(%s) it not allowed", key)
			return api.NewBatchQueryResponseWithMsg(api.InvalidParameter, key+" is not allowed")
		}
		searchFilters[key] = value
	}

	searchFilters = parseStrategySearchArgs(ctx, searchFilters)

	offset, limit, err := utils.ParseOffsetAndLimit(searchFilters)

	if err != nil {
		return api.NewBatchQueryResponse(api.InvalidParameter)
	}

	total, strategies, err := svr.storage.GetStrategies(searchFilters, offset, limit)
	if err != nil {
		log.AuthScope().Error("[Auth][Strategy] get strategies from store", zap.Any("query", searchFilters),
			zap.Error(err))
		return api.NewBatchQueryResponse(api.StoreLayerException)
	}

	resp := api.NewBatchQueryResponse(api.ExecuteSuccess)
	resp.Amount = utils.NewUInt32Value(total)
	resp.Size = utils.NewUInt32Value(uint32(len(strategies)))

	if strings.Compare(showDetail, "true") == 0 {
		log.AuthScope().Info("[Auth][Strategy] fill strategy detail", utils.ZapRequestID(requestID))
		resp.AuthStrategies = enhancedAuthStrategy2Api(strategies, svr.authStrategyFull2Api)
	} else {
		resp.AuthStrategies = enhancedAuthStrategy2Api(strategies, svr.authStrategy2Api)
	}

	return resp
}

var resTypeFilter = map[string]string{
	"namespace":    "0",
	"service":      "1",
	"config_group": "2",
}

var principalTypeFilter = map[string]string{
	"user":   "1",
	"group":  "2",
	"groups": "2",
}

// parseStrategySearchArgs 处理鉴权策略的搜索参数
func parseStrategySearchArgs(ctx context.Context, searchFilters map[string]string) map[string]string {
	if val, ok := searchFilters["res_type"]; ok {
		if v, exist := resTypeFilter[val]; exist {
			searchFilters["res_type"] = v
		} else {
			searchFilters["res_type"] = "0"
		}
	}

	if val, ok := searchFilters["principal_type"]; ok {
		if v, exist := principalTypeFilter[val]; exist {
			searchFilters["principal_type"] = v
		} else {
			searchFilters["principal_type"] = "1"
		}
	}

	if utils.ParseUserRole(ctx) != model.AdminUserRole {
		// 如果当前账户不是 admin 角色，既不是走资源视角查看，也不是指定principal查看，那么只能查询当前操作用户被关联到的鉴权策略，
		if _, ok := searchFilters["res_id"]; !ok {
			// 设置 owner 参数，只能查看对应 owner 下的策略
			searchFilters["owner"] = utils.ParseOwnerID(ctx)
			if _, ok := searchFilters["principal_id"]; !ok {
				// 如果当前不是 owner 角色，那么只能查询与自己有关的策略
				if !utils.ParseIsOwner(ctx) {
					searchFilters["principal_id"] = utils.ParseUserID(ctx)
					searchFilters["principal_type"] = strconv.Itoa(int(model.PrincipalUser))
				}
			}
		}
	}

	return searchFilters
}

// GetStrategy 根据策略ID获取详细的鉴权策略
// Case 1 如果当前操作者是该策略 principal 中的一员，则可以查看
// Case 2 如果当前操作者是该策略的 owner，则可以查看
// Case 3 如果当前操作者是admin角色，直接查看
func (svr *server) GetStrategy(ctx context.Context, req *api.AuthStrategy) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	userId := utils.ParseUserID(ctx)
	isOwner := utils.ParseIsOwner(ctx)

	if req.GetId().GetValue() == "" {
		return api.NewResponse(api.EmptyQueryParameter)
	}

	ret, err := svr.storage.GetStrategyDetail(req.GetId().GetValue())
	if err != nil {
		log.AuthScope().Error("[Auth][Strategy] get strategt from store",
			utils.ZapRequestID(requestID), zap.Error(err))
		return api.NewResponse(api.StoreLayerException)
	}
	if ret == nil {
		return api.NewAuthStrategyResponse(api.NotFoundAuthStrategyRule, req)
	}

	var canView bool
	if isOwner {
		// 是否是本鉴权策略的 owner 账户, 或者是否是超级管理员, 是的话则快速跳过下面的检查
		canView = (ret.Owner == userId) || utils.ParseUserRole(ctx) == model.AdminUserRole
	}

	// 判断是否在该策略所属的成员列表中，如果自己在某个用户组，而该用户组又在这个策略的成员中，则也是可以查看的
	if !canView {
		for index := range ret.Principals {
			principal := ret.Principals[index]
			if principal.PrincipalRole == model.PrincipalUser && principal.PrincipalID == userId {
				canView = true
				break
			}
			if principal.PrincipalRole == model.PrincipalGroup {
				if svr.cacheMgn.User().IsUserInGroup(userId, principal.PrincipalID) {
					canView = true
					break
				}
			}
		}
	}

	if !canView {
		log.AuthScope().Error("[Auth][Strategy] get strategy detail denied",
			utils.ZapRequestID(requestID),
			zap.String("user", userId),
			zap.String("strategy", req.Id.Value),
			zap.Bool("is-owner", isOwner),
		)
		return api.NewAuthStrategyResponse(api.NotAllowedAccess, req)
	}

	return api.NewAuthStrategyResponse(api.ExecuteSuccess, svr.authStrategyFull2Api(ret))
}

// GetPrincipalResources 获取某个principal可以获取到的所有资源ID数据信息
func (svr *server) GetPrincipalResources(ctx context.Context, query map[string]string) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	if len(query) == 0 {
		return api.NewResponse(api.EmptyRequest)
	}

	principalId := query["principal_id"]
	if principalId == "" {
		return api.NewResponse(api.EmptyQueryParameter)
	}

	var principalType string
	if v, exist := principalTypeFilter[query["principal_type"]]; exist {
		principalType = v
	} else {
		principalType = "1"
	}

	principalRole, _ := strconv.ParseInt(principalType, 10, 64)
	if err := model.CheckPrincipalType(int(principalRole)); err != nil {
		return api.NewResponse(api.InvalidPrincipalType)
	}

	var (
		resources = make([]model.StrategyResource, 0, 20)
		err       error
	)

	// 找这个用户所关联的用户组
	if model.PrincipalType(principalRole) == model.PrincipalUser {
		groupIds := svr.cacheMgn.User().GetUserLinkGroupIds(principalId)
		for i := range groupIds {
			res, err := svr.storage.GetStrategyResources(groupIds[i], model.PrincipalGroup)
			if err != nil {
				log.AuthScope().Error("[Auth][Strategy] get principal link resource", utils.ZapRequestID(requestID),
					zap.String("principal-id", principalId), zap.Any("principal-role", principalRole), zap.Error(err))
				return api.NewResponse(api.StoreLayerException)
			}
			resources = append(resources, res...)
		}
	}

	pResources, err := svr.storage.GetStrategyResources(principalId, model.PrincipalType(principalRole))
	if err != nil {
		log.AuthScope().Error("[Auth][Strategy] get principal link resource", utils.ZapRequestID(requestID),
			zap.String("principal-id", principalId), zap.Any("principal-role", principalRole), zap.Error(err))
		return api.NewResponse(api.StoreLayerException)
	}

	resources = append(resources, pResources...)
	tmp := &api.AuthStrategy{
		Resources: &api.StrategyResources{
			Namespaces:   make([]*api.StrategyResourceEntry, 0),
			Services:     make([]*api.StrategyResourceEntry, 0),
			ConfigGroups: make([]*api.StrategyResourceEntry, 0),
		},
	}

	svr.fillResourceInfo(tmp, &model.StrategyDetail{
		Resources: resourceDeduplication(resources),
	})

	return api.NewStrategyResourcesResponse(api.ExecuteSuccess, tmp.Resources)

}

// enhancedAuthStrategy2Api
func enhancedAuthStrategy2Api(s []*model.StrategyDetail, fn StrategyDetail2Api) []*api.AuthStrategy {
	out := make([]*api.AuthStrategy, 0, len(s))
	for k := range s {
		out = append(out, fn(s[k]))
	}

	return out
}

// authStrategy2Api
func (svr *server) authStrategy2Api(s *model.StrategyDetail) *api.AuthStrategy {
	if s == nil {
		return nil
	}

	// note: 不包括token，token比较特殊
	out := &api.AuthStrategy{
		Id:              utils.NewStringValue(s.ID),
		Name:            utils.NewStringValue(s.Name),
		Owner:           utils.NewStringValue(s.Owner),
		Comment:         utils.NewStringValue(s.Comment),
		Ctime:           utils.NewStringValue(commontime.Time2String(s.CreateTime)),
		Mtime:           utils.NewStringValue(commontime.Time2String(s.ModifyTime)),
		Action:          api.AuthAction(api.AuthAction_value[s.Action]),
		DefaultStrategy: utils.NewBoolValue(s.Default),
	}

	return out
}

// authStrategyFull2Api
func (svr *server) authStrategyFull2Api(data *model.StrategyDetail) *api.AuthStrategy {
	if data == nil {
		return nil
	}

	users := make([]*wrappers.StringValue, 0, len(data.Principals))
	groups := make([]*wrappers.StringValue, 0, len(data.Principals))
	for index := range data.Principals {
		principal := data.Principals[index]
		if principal.PrincipalRole == model.PrincipalUser {
			users = append(users, utils.NewStringValue(principal.PrincipalID))
		} else {
			groups = append(groups, utils.NewStringValue(principal.PrincipalID))
		}
	}

	// note: 不包括token，token比较特殊
	out := &api.AuthStrategy{
		Id:              utils.NewStringValue(data.ID),
		Name:            utils.NewStringValue(data.Name),
		Owner:           utils.NewStringValue(data.Owner),
		Comment:         utils.NewStringValue(data.Comment),
		Ctime:           utils.NewStringValue(commontime.Time2String(data.CreateTime)),
		Mtime:           utils.NewStringValue(commontime.Time2String(data.ModifyTime)),
		Action:          api.AuthAction(api.AuthAction_value[data.Action]),
		DefaultStrategy: utils.NewBoolValue(data.Default),
	}

	svr.fillPrincipalInfo(out, data)
	svr.fillResourceInfo(out, data)

	return out
}

// createAuthStrategyModel 创建鉴权策略的存储模型
func (svr *server) createAuthStrategyModel(strategy *api.AuthStrategy) *model.StrategyDetail {
	ret := &model.StrategyDetail{
		ID:         utils.NewUUID(),
		Name:       strategy.Name.GetValue(),
		Action:     api.AuthAction_READ_WRITE.String(),
		Comment:    strategy.Comment.GetValue(),
		Default:    false,
		Owner:      strategy.Owner.GetValue(),
		Valid:      true,
		Revision:   utils.NewUUID(),
		CreateTime: time.Now(),
		ModifyTime: time.Now(),
	}

	// 收集涉及的资源信息
	resEntry := make([]model.StrategyResource, 0, 20)
	resEntry = append(resEntry, svr.collectResEntry(ret.ID, api.ResourceType_Namespaces,
		strategy.GetResources().GetNamespaces(), false)...)
	resEntry = append(resEntry, svr.collectResEntry(ret.ID, api.ResourceType_Services,
		strategy.GetResources().GetServices(), false)...)
	resEntry = append(resEntry, svr.collectResEntry(ret.ID, api.ResourceType_ConfigGroups,
		strategy.GetResources().GetConfigGroups(), false)...)

	// 收集涉及的 principal 信息
	principals := make([]model.Principal, 0, 20)
	principals = append(principals, collectPrincipalEntry(ret.ID, model.PrincipalUser,
		strategy.GetPrincipals().GetUsers())...)
	principals = append(principals, collectPrincipalEntry(ret.ID, model.PrincipalGroup,
		strategy.GetPrincipals().GetGroups())...)

	ret.Resources = resEntry
	ret.Principals = principals

	return ret
}

// updateAuthStrategyAttribute 更新计算鉴权策略的属性
func (svr *server) updateAuthStrategyAttribute(ctx context.Context, strategy *api.ModifyAuthStrategy,
	saved *model.StrategyDetail) (*model.ModifyStrategyDetail, bool) {
	var needUpdate bool
	ret := &model.ModifyStrategyDetail{
		ID:         strategy.Id.GetValue(),
		Name:       saved.Name,
		Action:     saved.Action,
		Comment:    saved.Comment,
		ModifyTime: time.Now(),
	}

	// 只有 owner 可以修改的属性
	if utils.ParseIsOwner(ctx) {
		if strategy.GetComment() != nil && strategy.GetComment().GetValue() != saved.Comment {
			needUpdate = true
			ret.Comment = strategy.GetComment().GetValue()
		}

		if strategy.GetName().GetValue() != "" && strategy.GetName().GetValue() != saved.Name {
			needUpdate = true
			ret.Name = strategy.GetName().GetValue()
		}
	}

	if svr.computeResourceChange(ret, strategy) {
		needUpdate = true
	}
	if computePrincipalChange(ret, strategy) {
		needUpdate = true
	}

	return ret, needUpdate
}

// computeResourceChange 计算资源的变化情况，判断是否涉及变更
func (svr *server) computeResourceChange(modify *model.ModifyStrategyDetail, strategy *api.ModifyAuthStrategy) bool {
	var needUpdate bool
	addResEntry := make([]model.StrategyResource, 0)
	addResEntry = append(addResEntry, svr.collectResEntry(modify.ID, api.ResourceType_Namespaces,
		strategy.GetAddResources().GetNamespaces(), false)...)
	addResEntry = append(addResEntry, svr.collectResEntry(modify.ID, api.ResourceType_Services,
		strategy.GetAddResources().GetServices(), false)...)
	addResEntry = append(addResEntry, svr.collectResEntry(modify.ID, api.ResourceType_ConfigGroups,
		strategy.GetAddResources().GetConfigGroups(), false)...)

	if len(addResEntry) != 0 {
		needUpdate = true
		modify.AddResources = addResEntry
	}

	removeResEntry := make([]model.StrategyResource, 0)
	removeResEntry = append(removeResEntry, svr.collectResEntry(modify.ID, api.ResourceType_Namespaces,
		strategy.GetRemoveResources().GetNamespaces(), true)...)
	removeResEntry = append(removeResEntry, svr.collectResEntry(modify.ID, api.ResourceType_Services,
		strategy.GetRemoveResources().GetServices(), true)...)
	removeResEntry = append(removeResEntry, svr.collectResEntry(modify.ID, api.ResourceType_ConfigGroups,
		strategy.GetRemoveResources().GetConfigGroups(), true)...)

	if len(removeResEntry) != 0 {
		needUpdate = true
		modify.RemoveResources = removeResEntry
	}

	return needUpdate
}

// computePrincipalChange 计算 principal 的变化情况，判断是否涉及变更
func computePrincipalChange(modify *model.ModifyStrategyDetail, strategy *api.ModifyAuthStrategy) bool {
	var needUpdate bool
	addPrincipals := make([]model.Principal, 0)
	addPrincipals = append(addPrincipals, collectPrincipalEntry(modify.ID, model.PrincipalUser,
		strategy.GetAddPrincipals().GetUsers())...)
	addPrincipals = append(addPrincipals, collectPrincipalEntry(modify.ID, model.PrincipalGroup,
		strategy.GetAddPrincipals().GetGroups())...)

	if len(addPrincipals) != 0 {
		needUpdate = true
		modify.AddPrincipals = addPrincipals
	}

	removePrincipals := make([]model.Principal, 0)
	removePrincipals = append(removePrincipals, collectPrincipalEntry(modify.ID, model.PrincipalUser,
		strategy.GetRemovePrincipals().GetUsers())...)
	removePrincipals = append(removePrincipals, collectPrincipalEntry(modify.ID, model.PrincipalGroup,
		strategy.GetRemovePrincipals().GetGroups())...)

	if len(removePrincipals) != 0 {
		needUpdate = true
		modify.RemovePrincipals = removePrincipals
	}

	return needUpdate
}

// collectResEntry 将资源ID转换为对应的 []model.StrategyResource 数组
func (svr *server) collectResEntry(ruleId string, resType api.ResourceType,
	res []*api.StrategyResourceEntry, delete bool) []model.StrategyResource {
	resEntries := make([]model.StrategyResource, 0, len(res)+1)
	if len(res) == 0 {
		return resEntries
	}

	for index := range res {
		// 如果是添加的动作，那么需要进行归一化处理
		if !delete {
			// 归一化处理
			if res[index].GetId().GetValue() == "*" || res[index].GetName().GetValue() == "*" {
				return []model.StrategyResource{
					{
						StrategyID: ruleId,
						ResType:    int32(resType),
						ResID:      "*",
					},
				}
			}
		}

		entry := model.StrategyResource{
			StrategyID: ruleId,
			ResType:    int32(resType),
			ResID:      res[index].GetId().GetValue(),
		}

		resEntries = append(resEntries, entry)
	}

	return resEntries
}

// collectPrincipalEntry 将 Principal 转换为对应的 []model.Principal 数组
func collectPrincipalEntry(ruleID string, uType model.PrincipalType, res []*api.Principal) []model.Principal {
	principals := make([]model.Principal, len(res)+1)
	if len(res) == 0 {
		return principals
	}

	for index := range res {
		principals = append(principals, model.Principal{
			StrategyID:    ruleID,
			PrincipalID:   res[index].GetId().GetValue(),
			PrincipalRole: uType,
		})
	}

	return principals
}

// checkCreateStrategy 检查创建鉴权策略的请求
func (svr *server) checkCreateStrategy(req *api.AuthStrategy) *api.Response {
	// 检查名称信息
	if err := checkName(req.GetName()); err != nil {
		return api.NewAuthStrategyResponse(api.InvalidUserName, req)
	}

	// 检查 owner 信息
	if err := checkOwner(req.GetOwner()); err != nil {
		return api.NewAuthStrategyResponse(api.InvalidAuthStrategyOwners, req)
	}

	// 检查用户是否存在
	if err := svr.checkUserExist(convertPrincipalsToUsers(req.GetPrincipals())); err != nil {
		return api.NewAuthStrategyResponse(api.NotFoundUser, req)
	}

	// 检查用户组是否存在
	if err := svr.checkGroupExist(convertPrincipalsToGroups(req.GetPrincipals())); err != nil {
		return api.NewAuthStrategyResponse(api.NotFoundUserGroup, req)
	}

	// 检查资源是否存在
	if errResp := svr.checkResourceExist(req.GetResources()); errResp != nil {
		return errResp
	}

	return nil
}

// checkUpdateStrategy 检查更新鉴权策略的请求
// Case 1. 修改的是默认鉴权策略的话，只能修改资源，不能添加用户 or 用户组
// Case 2. 鉴权策略只能被自己的 owner 对应的用户修改
func (svr *server) checkUpdateStrategy(ctx context.Context, req *api.ModifyAuthStrategy,
	saved *model.StrategyDetail) *api.Response {
	userId := utils.ParseUserID(ctx)
	if utils.ParseUserRole(ctx) != model.AdminUserRole {
		if !utils.ParseIsOwner(ctx) || userId != saved.Owner {
			log.AuthScope().Error("[Auth][Strategy] modify strategy denied, current user not owner",
				utils.ZapRequestID(utils.ParseRequestID(ctx)),
				zap.String("user", userId),
				zap.String("owner", saved.Owner),
				zap.String("strategy", saved.ID))
			return api.NewModifyAuthStrategyResponse(api.NotAllowedAccess, req)
		}
	}

	if saved.Default {
		if len(req.AddPrincipals.Users) != 0 ||
			len(req.AddPrincipals.Groups) != 0 ||
			len(req.RemovePrincipals.Groups) != 0 ||
			len(req.RemovePrincipals.Users) != 0 {
			return api.NewModifyAuthStrategyResponse(api.NotAllowModifyDefaultStrategyPrincipal, req)
		}
	}

	// 检查用户是否存在
	if err := svr.checkUserExist(convertPrincipalsToUsers(req.GetAddPrincipals())); err != nil {
		return api.NewModifyAuthStrategyResponse(api.NotFoundUser, req)
	}

	// 检查用户组是否存
	if err := svr.checkGroupExist(convertPrincipalsToGroups(req.GetAddPrincipals())); err != nil {
		return api.NewModifyAuthStrategyResponse(api.NotFoundUserGroup, req)
	}

	// 检查资源是否存在
	if errResp := svr.checkResourceExist(req.GetAddResources()); errResp != nil {
		return errResp
	}

	return nil
}

// authStrategyRecordEntry 转换为鉴权策略的记录结构体
func authStrategyRecordEntry(ctx context.Context, req *api.AuthStrategy, md *model.StrategyDetail,
	operationType model.OperationType) *model.RecordEntry {
	entry := &model.RecordEntry{
		ResourceType:  model.RAuthStrategy,
		StrategyName:  md.Name,
		OperationType: operationType,
		Operator:      utils.ParseOperator(ctx),
		CreateTime:    time.Now(),
	}

	return entry
}

// authModifyStrategyRecordEntry
func authModifyStrategyRecordEntry(ctx context.Context, req *api.ModifyAuthStrategy, md *model.ModifyStrategyDetail,
	operationType model.OperationType) *model.RecordEntry {
	entry := &model.RecordEntry{
		ResourceType:  model.RAuthStrategy,
		StrategyName:  md.ID,
		OperationType: operationType,
		Operator:      utils.ParseOperator(ctx),
		CreateTime:    time.Now(),
	}

	return entry
}

func convertPrincipalsToUsers(principals *api.Principals) []*api.User {
	if principals == nil {
		return make([]*api.User, 0)
	}

	users := make([]*api.User, 0, len(principals.Users))
	for k := range principals.GetUsers() {
		user := principals.GetUsers()[k]
		users = append(users, &api.User{
			Id: user.Id,
		})
	}

	return users
}

func convertPrincipalsToGroups(principals *api.Principals) []*api.UserGroup {
	if principals == nil {
		return make([]*api.UserGroup, 0)
	}

	groups := make([]*api.UserGroup, 0, len(principals.Groups))
	for k := range principals.GetGroups() {
		group := principals.GetGroups()[k]
		groups = append(groups, &api.UserGroup{
			Id: group.Id,
		})
	}

	return groups
}

// checkUserExist 检查用户是否存在
func (svr *server) checkUserExist(users []*api.User) error {
	if len(users) == 0 {
		return nil
	}

	userCache := svr.cacheMgn.User()

	for index := range users {
		if val := userCache.GetUserByID(users[index].GetId().GetValue()); val == nil {
			return model.ErrorNoUser
		}
	}

	return nil
}

// checkUserGroupExist 检查用户组是否存在
func (svr *server) checkGroupExist(groups []*api.UserGroup) error {
	if len(groups) == 0 {
		return nil
	}
	userCache := svr.cacheMgn.User()

	for index := range groups {
		if val := userCache.GetGroup(groups[index].GetId().GetValue()); val == nil {
			return model.ErrorNoUserGroup
		}
	}

	return nil
}

// checkResourceExist 检查资源是否存在
func (svr *server) checkResourceExist(resources *api.StrategyResources) *api.Response {
	namespaces := resources.GetNamespaces()

	nsCache := svr.cacheMgn.Namespace()
	for index := range namespaces {
		val := namespaces[index]
		if val.GetId().GetValue() == "*" {
			break
		}
		ns := nsCache.GetNamespace(val.GetId().GetValue())
		if ns == nil {
			return api.NewResponse(api.NotFoundNamespace)
		}
	}

	services := resources.GetServices()
	svcCache := svr.cacheMgn.Service()
	for index := range services {
		val := services[index]
		if val.GetId().GetValue() == "*" {
			break
		}
		svc := svcCache.GetServiceByID(val.GetId().GetValue())
		if svc == nil {
			return api.NewResponse(api.NotFoundService)
		}
	}

	return nil
}

// normalizeResource 对于资源进行归一化处理
//  如果出现 * 的话，则该资源访问策略就是 *
func (svr *server) normalizeResource(resources *api.StrategyResources) *api.StrategyResources {
	namespaces := resources.GetNamespaces()
	for index := range namespaces {
		val := namespaces[index]
		if val.GetId().GetValue() == "*" {
			resources.Namespaces = []*api.StrategyResourceEntry{{
				Id: utils.NewStringValue("*"),
			}}
			break
		}
	}

	services := resources.GetServices()
	for index := range services {
		val := services[index]
		if val.GetId().GetValue() == "*" {
			resources.Services = []*api.StrategyResourceEntry{{
				Id: utils.NewStringValue("*"),
			}}
			break
		}
	}

	return resources
}

// fillPrincipalInfo 填充 principal 摘要信息
func (svr *server) fillPrincipalInfo(resp *api.AuthStrategy, data *model.StrategyDetail) {
	users := make([]*api.Principal, 0, len(data.Principals))
	groups := make([]*api.Principal, 0, len(data.Principals))
	for index := range data.Principals {
		principal := data.Principals[index]
		if principal.PrincipalRole == model.PrincipalUser {
			user := svr.cacheMgn.User().GetUserByID(principal.PrincipalID)
			if user == nil {
				continue
			}
			users = append(users, &api.Principal{
				Id:   utils.NewStringValue(user.ID),
				Name: utils.NewStringValue(user.Name),
			})
		} else {
			group := svr.cacheMgn.User().GetGroup(principal.PrincipalID)
			if group == nil {
				continue
			}
			groups = append(groups, &api.Principal{
				Id:   utils.NewStringValue(group.ID),
				Name: utils.NewStringValue(group.Name),
			})
		}
	}

	resp.Principals = &api.Principals{
		Users:  users,
		Groups: groups,
	}
}

// fillResourceInfo 填充资源摘要信息
func (svr *server) fillResourceInfo(resp *api.AuthStrategy, data *model.StrategyDetail) {
	namespaces := make([]*api.StrategyResourceEntry, 0, len(data.Resources))
	services := make([]*api.StrategyResourceEntry, 0, len(data.Resources))
	configGroups := make([]*api.StrategyResourceEntry, 0, len(data.Resources))

	var (
		autoAllNs  bool
		autoAllSvc bool
	)

	for index := range data.Resources {
		res := data.Resources[index]
		switch res.ResType {
		case int32(api.ResourceType_Namespaces):
			if res.ResID == "*" {
				autoAllNs = true
				namespaces = []*api.StrategyResourceEntry{
					{
						Id:        utils.NewStringValue("*"),
						Namespace: utils.NewStringValue("*"),
						Name:      utils.NewStringValue("*"),
					},
				}
				continue
			}

			if !autoAllNs {
				ns := svr.cacheMgn.Namespace().GetNamespace(res.ResID)
				if ns == nil {
					log.AuthScope().Error("[Auth][Strategy] not found namespace in fill-info",
						zap.String("id", data.ID), zap.String("namespace", res.ResID))
					continue
				}
				namespaces = append(namespaces, &api.StrategyResourceEntry{
					Id:        utils.NewStringValue(ns.Name),
					Namespace: utils.NewStringValue(ns.Name),
					Name:      utils.NewStringValue(ns.Name),
				})
			}
		case int32(api.ResourceType_Services):
			if res.ResID == "*" {
				autoAllSvc = true
				services = []*api.StrategyResourceEntry{
					{
						Id:        utils.NewStringValue("*"),
						Namespace: utils.NewStringValue("*"),
						Name:      utils.NewStringValue("*"),
					},
				}
				continue
			}

			if !autoAllSvc {
				svc := svr.cacheMgn.Service().GetServiceByID(res.ResID)
				if svc == nil {
					log.AuthScope().Error("[Auth][Strategy] not found service in fill-info",
						zap.String("id", data.ID), zap.String("service", res.ResID))
					continue
				}
				services = append(services, &api.StrategyResourceEntry{
					Id:        utils.NewStringValue(svc.ID),
					Namespace: utils.NewStringValue(svc.Namespace),
					Name:      utils.NewStringValue(svc.Name),
				})
			}
		case int32(api.ResourceType_ConfigGroups):
		}
	}

	resp.Resources = &api.StrategyResources{
		Namespaces:   namespaces,
		Services:     services,
		ConfigGroups: configGroups,
	}
}

type resourceFilter struct {
	ns   map[string]struct{}
	svc  map[string]struct{}
	conf map[string]struct{}
}

// filter different types of Strategy resources
func resourceDeduplication(resources []model.StrategyResource) []model.StrategyResource {
	rLen := len(resources)
	ret := make([]model.StrategyResource, 0, rLen)
	rf := resourceFilter{
		ns:   make(map[string]struct{}, rLen),
		svc:  make(map[string]struct{}, rLen),
		conf: make(map[string]struct{}, rLen),
	}

	est := struct{}{}
	for i := range resources {
		res := resources[i]
		if res.ResType == int32(api.ResourceType_Namespaces) {
			if _, exist := rf.ns[res.ResID]; !exist {
				rf.ns[res.ResID] = est
				ret = append(ret, res)
			}
			continue
		}

		if res.ResType == int32(api.ResourceType_Services) {
			if _, exist := rf.svc[res.ResID]; !exist {
				rf.svc[res.ResID] = est
				ret = append(ret, res)
			}

			continue
		}

		// other type conf
		if _, exist := rf.conf[res.ResID]; !exist {
			rf.conf[res.ResID] = est
			ret = append(ret, res)
		}
	}

	return ret
}
