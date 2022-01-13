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
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/polarismesh/polaris-server/cache"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	commontime "github.com/polarismesh/polaris-server/common/time"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/plugin"
	"github.com/polarismesh/polaris-server/store"
	"go.uber.org/zap"
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
	storage  store.Store
	history  plugin.History
	cacheMgn *cache.NamingCache
}

// newAthStrategyServer
func newAthStrategyServer(s store.Store, cacheMgn *cache.NamingCache) (*authStrategyServer, error) {
	svr := &authStrategyServer{
		storage:  s,
		cacheMgn: cacheMgn,
	}

	return svr, svr.initialize()
}

func (svr *authStrategyServer) initialize() error {
	// 获取History插件，注意：插件的配置在bootstrap已经设置好
	svr.history = plugin.GetHistory()
	if svr.history == nil {
		log.GetAuthLogger().Warnf("Not Found History Log Plugin")
	}

	return nil
}

// CreateStrategy
func (svr *authStrategyServer) CreateStrategy(ctx context.Context, req *api.AuthStrategy) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)
	ownerId := utils.ParseOwnerID(ctx)

	req.Owner = utils.NewStringValue(ownerId)

	if checkErrResp := svr.checkCreateStrategy(req); checkErrResp != nil {
		return checkErrResp
	}

	strategy, err := svr.storage.GetStrategySimpleByName(ownerId, req.GetName().GetValue())
	if err != nil {
		log.GetAuthLogger().Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewAuthStrategyResponse(api.StoreLayerException, req)
	}

	if strategy != nil {
		return api.NewAuthStrategyResponse(api.ExistedResource, req)
	}

	req.Resource = svr.normalizeResource(req.Resource)

	data := createAuthStrategyModel(req)
	if err := svr.storage.AddStrategy(data); err != nil {
		log.GetAuthLogger().Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	msg := fmt.Sprintf("create auth strategy: name=%v", req.GetName().GetValue())
	log.GetAuthLogger().Info(msg, utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
	svr.RecordHistory(authStrategyRecordEntry(ctx, req, data, model.OCreate))

	out := &api.AuthStrategy{
		Name: req.GetName(),
	}

	return api.NewAuthStrategyResponse(api.ExecuteSuccess, out)
}

// UpdateStrategy 实现鉴权策略的增量变更
func (svr *authStrategyServer) UpdateStrategy(ctx context.Context, req *api.ModifyAuthStrategy) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)
	ownerId := utils.ParseOwnerID(ctx)

	req.Owner = utils.NewStringValue(ownerId)

	if checkErrResp := svr.checkUpdateStrategy(req); checkErrResp != nil {
		return checkErrResp
	}

	strategy, err := svr.storage.GetStrategyDetail(req.GetId().GetValue())
	if err != nil {
		log.GetAuthLogger().Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewModifyAuthStrategyResponse(api.StoreLayerException, req)
	}

	if strategy == nil {
		return api.NewModifyAuthStrategyResponse(api.NotFoundResource, req)
	}

	req.AddResource = svr.normalizeResource(req.AddResource)
	data, needUpdate := updateAuthStrategyAttribute(req, strategy)
	if !needUpdate {
		return api.NewModifyAuthStrategyResponse(api.NoNeedUpdate, req)
	}
	if err := svr.storage.UpdateStrategy(data); err != nil {
		log.GetAuthLogger().Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	log.GetAuthLogger().Info("update auth strategy", zap.String("name", req.GetName().GetValue()), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
	svr.RecordHistory(authModifyStrategyRecordEntry(ctx, req, data, model.OUpdate))

	return api.NewModifyAuthStrategyResponse(api.ExecuteSuccess, req)
}

// DeleteStrategy
func (svr *authStrategyServer) DeleteStrategy(ctx context.Context, req *api.AuthStrategy) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)

	strategy, err := svr.storage.GetStrategyDetail(req.GetId().GetValue())
	if err != nil {
		log.GetAuthLogger().Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewAuthStrategyResponse(api.StoreLayerException, req)
	}

	if strategy == nil {
		return api.NewAuthStrategyResponse(api.ExecuteSuccess, req)
	}

	if strings.HasPrefix(strategy.Name, model.DefaultStrategyPrefix) {
		return api.NewAuthStrategyResponseWithMsg(api.BadRequest, "default auth strategy can't delete", req)
	}

	if err := svr.storage.DeleteStrategy(req.GetId().GetValue()); err != nil {
		log.GetAuthLogger().Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	msg := fmt.Sprintf("delete auth strategy: name=%v", req.GetName().GetValue())
	log.GetAuthLogger().Info(msg, utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
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
			log.GetAuthLogger().Errorf("[Auth][AuthStrategy][ListStrategy] attribute(%s) it not allowed", key)
			return api.NewBatchQueryResponseWithMsg(api.InvalidParameter, key+" is not allowed")
		}
		searchFilters[key] = value
	}

	offset, limit, err := utils.ParseOffsetAndLimit(searchFilters)
	if err != nil {
		return api.NewBatchQueryResponse(api.InvalidParameter)
	}

	total, strategies, err := svr.storage.ListStrategySimple(searchFilters, offset, limit)
	if err != nil {
		log.GetAuthLogger().Errorf("[Auth][AuthStrategy][ListStrategy] req(%+v) store err: %s", query, err.Error())
		return api.NewBatchQueryResponse(api.StoreLayerException)
	}

	resp := api.NewBatchQueryResponse(api.ExecuteSuccess)
	resp.Amount = utils.NewUInt32Value(total)
	resp.Size = utils.NewUInt32Value(uint32(len(strategies)))
	resp.AuthStrategy = enhancedAuthStrategy2Api(strategies, authStrategy2Api)
	return resp
}

// GetStrategy
//  @receiver svr
//  @param ctx
//  @param query
//  @return *api.Response
func (svr *authStrategyServer) GetStrategy(ctx context.Context, query map[string]string) *api.Response {
	userId := utils.ParseUserID(ctx)
	isOwner := utils.ParseIsOwner(ctx)

	ruleId, ok := query["id"]
	if !ok || ruleId == "" {
		return api.NewResponse(api.EmptyQueryParameter)
	}

	ret, err := svr.storage.GetStrategyDetail(ruleId)
	if err != nil {
		return api.NewResponseWithMsg(api.StoreLayerException, err.Error())
	}

	if !isOwner {
		canView := false
		for index := range ret.Principals {
			principal := ret.Principals[index]
			if principal.PrincipalRole == model.PrincipalUser && principal.PrincipalID == userId {
				canView = true
				break
			}
			if principal.PrincipalRole == model.PrincipalUserGroup {
				if svr.cacheMgn.User().IsUserInGroup(userId, principal.PrincipalID) {
					canView = true
					break
				}
			}
		}
		if !canView {
			return api.NewResponse(api.NotAllowedAccess)
		}
	}

	return api.NewAuthStrategyResponse(api.ExecuteSuccess, authStrategy2Api(ret))
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

	namespaces := make([]*wrappers.StringValue, 0, 8)
	services := make([]*wrappers.StringValue, 0, 8)
	configGroups := make([]*wrappers.StringValue, 0, 8)

	for index := range data.Resources {
		res := data.Resources[index]

		switch res.ResType {
		case int32(api.ResourceType_Namespaces):
			namespaces = append(namespaces, utils.NewStringValue(res.ResID))
		case int32(api.ResourceType_Services):
			services = append(services, utils.NewStringValue(res.ResID))
		case int32(api.ResourceType_ConfigGroups):
			configGroups = append(configGroups, utils.NewStringValue(res.ResID))
		}
	}

	users := make([]*wrappers.StringValue, 0)
	groups := make([]*wrappers.StringValue, 0)
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
		Id:      utils.NewStringValue(data.ID),
		Name:    utils.NewStringValue(data.Name),
		Owner:   utils.NewStringValue(data.Owner),
		Comment: utils.NewStringValue(data.Comment),
		Ctime:   utils.NewStringValue(commontime.Time2String(data.CreateTime)),
		Mtime:   utils.NewStringValue(commontime.Time2String(data.ModifyTime)),
		Principal: &api.Principal{
			Users:  users,
			Groups: groups,
		},
		Resource: &api.StrategyResource{
			StrategyId:   utils.NewStringValue(data.ID),
			Namespaces:   namespaces,
			Services:     services,
			ConfigGroups: configGroups,
		},
	}

	return out
}

// createAuthStrategyModel
func createAuthStrategyModel(strategy *api.AuthStrategy) *model.StrategyDetail {

	ret := &model.StrategyDetail{
		ID:         utils.NewUUID(),
		Name:       strategy.Name.GetValue(),
		Action:     api.AuthAction_READ_WRITE.String(),
		Comment:    strategy.Comment.GetValue(),
		Default:    false,
		Owner:      strategy.Owner.GetValue(),
		Valid:      true,
		CreateTime: time.Now(),
		ModifyTime: time.Now(),
	}

	resEntry := make([]model.StrategyResource, 0)
	resEntry = append(resEntry, collectResEntry(ret.ID, api.ResourceType_Namespaces, strategy.GetResource().GetNamespaces())...)
	resEntry = append(resEntry, collectResEntry(ret.ID, api.ResourceType_Services, strategy.GetResource().GetServices())...)
	resEntry = append(resEntry, collectResEntry(ret.ID, api.ResourceType_ConfigGroups, strategy.GetResource().GetConfigGroups())...)

	principals := make([]model.Principal, 0)
	principals = append(principals, collectPrincipalEntry(ret.ID, model.PrincipalUser, strategy.Principal.Users)...)
	principals = append(principals, collectPrincipalEntry(ret.ID, model.PrincipalUserGroup, strategy.Principal.Groups)...)

	ret.Resources = resEntry
	ret.Principals = principals

	return ret
}

// updateAuthStrategyAttribute
//  @param strategy
//  @param saved
//  @return *model.ModifyStrategyDetail
//  @return bool
func updateAuthStrategyAttribute(strategy *api.ModifyAuthStrategy, saved *model.StrategyDetail) (*model.ModifyStrategyDetail, bool) {

	needUpdate := false

	ret := &model.ModifyStrategyDetail{
		ID:         strategy.Id.GetValue(),
		ModifyTime: time.Now(),
	}

	if strategy.GetComment().GetValue() != "" && strategy.GetComment().GetValue() != saved.Comment {
		needUpdate = true
		ret.Comment = strategy.GetComment().GetValue()
	}

	needUpdate = computeResourceChange(ret, strategy)
	needUpdate = computePrincipalChange(ret, strategy)

	return ret, needUpdate
}

// computeResourceChange
//  @param modify
//  @param strategy
//  @return bool
func computeResourceChange(modify *model.ModifyStrategyDetail, strategy *api.ModifyAuthStrategy) bool {

	needUpdate := false

	addResEntry := make([]model.StrategyResource, 0)
	addResEntry = append(addResEntry, collectResEntry(modify.ID, api.ResourceType_Namespaces, strategy.GetAddResource().GetNamespaces())...)
	addResEntry = append(addResEntry, collectResEntry(modify.ID, api.ResourceType_Services, strategy.GetAddResource().GetServices())...)
	addResEntry = append(addResEntry, collectResEntry(modify.ID, api.ResourceType_ConfigGroups, strategy.GetAddResource().GetConfigGroups())...)

	if len(addResEntry) != 0 {
		needUpdate = true
		modify.AddResources = addResEntry
	}

	removeResEntry := make([]model.StrategyResource, 0)
	removeResEntry = append(removeResEntry, collectResEntry(modify.ID, api.ResourceType_Namespaces, strategy.GetRemoveResource().GetNamespaces())...)
	removeResEntry = append(removeResEntry, collectResEntry(modify.ID, api.ResourceType_Services, strategy.GetRemoveResource().GetServices())...)
	removeResEntry = append(removeResEntry, collectResEntry(modify.ID, api.ResourceType_ConfigGroups, strategy.GetRemoveResource().GetConfigGroups())...)

	if len(removeResEntry) != 0 {
		needUpdate = true
		modify.RemoveResources = removeResEntry
	}

	return needUpdate
}

// computePrincipalChange
//  @param modify
//  @param strategy
//  @return bool
func computePrincipalChange(modify *model.ModifyStrategyDetail, strategy *api.ModifyAuthStrategy) bool {

	needUpdate := false

	addPrincipals := make([]model.Principal, 0)
	addPrincipals = append(addPrincipals, collectPrincipalEntry(modify.ID, model.PrincipalUser, strategy.GetAddPrincipal().GetUsers())...)
	addPrincipals = append(addPrincipals, collectPrincipalEntry(modify.ID, model.PrincipalUserGroup, strategy.GetAddPrincipal().GetGroups())...)

	if len(addPrincipals) != 0 {
		needUpdate = true
		modify.AddPrincipals = addPrincipals
	}

	removePrincipals := make([]model.Principal, 0)
	removePrincipals = append(removePrincipals, collectPrincipalEntry(modify.ID, model.PrincipalUser, strategy.GetRemovePrincipal().GetUsers())...)
	removePrincipals = append(removePrincipals, collectPrincipalEntry(modify.ID, model.PrincipalUserGroup, strategy.GetRemovePrincipal().GetGroups())...)

	if len(removePrincipals) != 0 {
		needUpdate = true
		modify.RemovePrincipals = removePrincipals
	}

	return needUpdate
}

// collectResEntry
//  @param ruleId
//  @param res
//  @return []model.StrategyResource
func collectResEntry(ruleId string, resType api.ResourceType, res []*wrappers.StringValue) []model.StrategyResource {
	if len(res) == 0 {
		return make([]model.StrategyResource, 0)
	}

	resEntry := make([]model.StrategyResource, 0)
	for index := range res {
		resEntry = append(resEntry, model.StrategyResource{
			StrategyID: ruleId,
			ResType:    int32(resType),
			ResID:      res[index].GetValue(),
			Valid:      true,
			CreateTime: time.Now(),
			ModifyTime: time.Now(),
		})
	}

	return resEntry
}

// collectPrincipalEntry
//  @param ruleId
//  @param uType
//  @param res
//  @return []model.Principal
func collectPrincipalEntry(ruleId string, uType model.PrincipalType, res []*wrappers.StringValue) []model.Principal {
	if len(res) == 0 {
		return make([]model.Principal, 0)
	}

	principals := make([]model.Principal, 0)
	for index := range res {
		principals = append(principals, model.Principal{
			StrategyID:    ruleId,
			PrincipalID:   res[index].GetValue(),
			PrincipalRole: uType,
		})
	}

	return principals
}

// checkCreateStrategy
//  @receiver svr
//  @param req
//  @return *api.Response
func (svr *authStrategyServer) checkCreateStrategy(req *api.AuthStrategy) *api.Response {
	// 检查名称信息
	if err := checkName(req.GetName()); err != nil {
		return api.NewAuthStrategyResponse(api.InvalidUserName, req)
	}
	// 用户自己创建的 strategy 不可以是特殊前缀
	if strings.HasPrefix(req.GetName().GetValue(), model.DefaultStrategyPrefix) {
		return api.NewAuthStrategyResponse(api.BadRequest, req)
	}

	if err := checkOwner(req.GetOwner()); err != nil {
		return api.NewAuthStrategyResponse(api.InvalidAuthStrategyOwners, req)
	}

	// 检查用户是否存在
	if err := svr.checkUserExist(req.GetPrincipal().GetUsers()); err != nil {
		return api.NewAuthStrategyResponse(api.NotFoundUser, req)
	}

	// 检查用户组是否存在
	if err := svr.checkUserGroupExist(req.GetPrincipal().GetGroups()); err != nil {
		return api.NewAuthStrategyResponse(api.NotFoundUserGroup, req)
	}

	// 检查资源是否存在
	if errResp := svr.checkResourceExist(req.GetResource()); errResp != nil {
		return errResp
	}

	return nil
}

// checkUpdateStrategy
//  @receiver svr
//  @param req
//  @return *api.Response
func (svr *authStrategyServer) checkUpdateStrategy(req *api.ModifyAuthStrategy) *api.Response {

	// 检查用户是否存在
	if err := svr.checkUserExist(req.GetAddPrincipal().GetUsers()); err != nil {
		return api.NewModifyAuthStrategyResponse(api.NotFoundUser, req)
	}

	// 检查用户组是否存在
	if err := svr.checkUserGroupExist(req.GetAddPrincipal().GetGroups()); err != nil {
		return api.NewModifyAuthStrategyResponse(api.NotFoundUserGroup, req)
	}

	// 检查资源是否存在
	if errResp := svr.checkResourceExist(req.GetAddResource()); errResp != nil {
		return errResp
	}

	return nil
}

// authStrategyRecordEntry
//  @param ctx
//  @param req
//  @param md
//  @param operationType
//  @return *model.RecordEntry
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

// authModifyStrategyRecordEntry
//  @param ctx
//  @param req
//  @param md
//  @param operationType
//  @return *model.RecordEntry
func authModifyStrategyRecordEntry(ctx context.Context, req *api.ModifyAuthStrategy, md *model.ModifyStrategyDetail,
	operationType model.OperationType) *model.RecordEntry {
	entry := &model.RecordEntry{
		ResourceType:  model.RAuthStrategy,
		OperationType: operationType,
		Operator:      utils.ParseOperator(ctx),
		CreateTime:    time.Now(),
	}

	return entry
}

// checkUserExist
//  @receiver svr
//  @param users
//  @return error
func (svr *authStrategyServer) checkUserExist(users []*wrappers.StringValue) error {
	if len(users) == 0 {
		return nil
	}

	userCache := svr.cacheMgn.User()

	for index := range users {
		if val := userCache.GetUser(users[index].GetValue()); val == nil {
			return ErrorNoUser
		}
	}

	return nil
}

// checkUserGroupExist
//  @receiver svr
//  @param groups
//  @return error
func (svr *authStrategyServer) checkUserGroupExist(groups []*wrappers.StringValue) error {
	if len(groups) == 0 {
		return nil
	}
	userCache := svr.cacheMgn.User()

	for index := range groups {
		if val := userCache.GetUserGroup(groups[index].GetValue()); val == nil {
			return ErrorNoUserGroup
		}
	}
	return nil
}

// checkResourceExist
//  @receiver svr
//  @param resources
//  @return *api.Response
func (svr *authStrategyServer) checkResourceExist(resources *api.StrategyResource) *api.Response {
	namespaces := resources.GetNamespaces()

	nsCache := svr.cacheMgn.Namespace()
	for index := range namespaces {
		val := namespaces[index]
		if val.GetValue() == "*" {
			break
		}
		ns := nsCache.GetNamespace(val.GetValue())
		if ns == nil {
			return api.NewResponse(api.NotFoundNamespace)
		}
	}

	services := resources.GetServices()
	svcCache := svr.cacheMgn.Service()
	for index := range services {
		val := services[index]
		if val.GetValue() == "*" {
			break
		}
		svc := svcCache.GetServiceByID(val.GetValue())
		if svc == nil {
			return api.NewResponse(api.NotFoundService)
		}
	}

	return nil
}

// normalizeResource
//  @receiver svr
//  @param resources
//  @return *api.StrategyResource
func (svr *authStrategyServer) normalizeResource(resources *api.StrategyResource) *api.StrategyResource {
	namespaces := resources.GetNamespaces()
	for index := range namespaces {
		val := namespaces[index]
		if val.GetValue() == "*" {
			resources.Namespaces = []*wrappers.StringValue{utils.NewStringValue("*")}
			break
		}
	}

	services := resources.GetServices()
	for index := range services {
		val := services[index]
		if val.GetValue() == "*" {
			resources.Services = []*wrappers.StringValue{utils.NewStringValue("*")}
			break
		}
	}

	configGroups := resources.GetConfigGroups()
	for index := range configGroups {
		val := configGroups[index]
		if val.GetValue() == "*" {
			resources.ConfigGroups = []*wrappers.StringValue{utils.NewStringValue("*")}
			break
		}
	}

	return resources
}
