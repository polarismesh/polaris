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
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/auth"
	"github.com/polarismesh/polaris/cache"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

var (
	// ErrorNotAllowedAccess 鉴权失败
	ErrorNotAllowedAccess error = errors.New(api.Code2Info(api.NotAllowedAccess))

	// ErrorInvalidParameter 不合法的参数
	ErrorInvalidParameter error = errors.New(api.Code2Info(api.InvalidParameter))
)

// defaultAuthChecker 北极星自带的默认鉴权中心
type defaultAuthChecker struct {
	cacheMgn *cache.CacheManager
}

// Initialize 执行初始化动作
func (d *defaultAuthChecker) Initialize(options *auth.Config, s store.Store, cacheMgn *cache.CacheManager) error {
	contentBytes, err := json.Marshal(options.Option)
	if err != nil {
		return err
	}

	cfg := DefaultAuthConfig()
	if err := json.Unmarshal(contentBytes, cfg); err != nil {
		return err
	}

	if err := cfg.Verify(); err != nil {
		return err
	}

	AuthOption = cfg
	d.cacheMgn = cacheMgn

	return nil
}

// Cache 获取缓存统一管理
func (d *defaultAuthChecker) Cache() *cache.CacheManager {
	return d.cacheMgn
}

// IsOpenConsoleAuth 针对控制台是否开启了操作鉴权
func (d *defaultAuthChecker) IsOpenConsoleAuth() bool {
	return AuthOption.ConsoleOpen
}

// IsOpenClientAuth 针对客户端是否开启了操作鉴权
func (d *defaultAuthChecker) IsOpenClientAuth() bool {
	return AuthOption.ClientOpen
}

// IsOpenAuth 返回对于控制台/客户端任意其中的一个是否开启了操作鉴权
func (d *defaultAuthChecker) IsOpenAuth() bool {
	return d.IsOpenConsoleAuth() || d.IsOpenClientAuth()
}

// CheckClientPermission 执行检查客户端动作判断是否有权限，并且对 RequestContext 注入操作者数据
func (d *defaultAuthChecker) CheckClientPermission(preCtx *model.AcquireContext) (bool, error) {
	if !d.IsOpenClientAuth() {
		return true, nil
	}

	return d.CheckPermission(preCtx)
}

// CheckConsolePermission 执行检查控制台动作判断是否有权限，并且对 RequestContext 注入操作者数据
func (d *defaultAuthChecker) CheckConsolePermission(preCtx *model.AcquireContext) (bool, error) {
	if !d.IsOpenConsoleAuth() {
		return true, nil
	}

	if preCtx.GetModule() == model.MaintainModule {
		return d.checkMaintainPermission(preCtx)
	}

	return d.CheckPermission(preCtx)
}

// CheckMaintainPermission 执行检查运维动作判断是否有权限
func (d *defaultAuthChecker) checkMaintainPermission(preCtx *model.AcquireContext) (bool, error) {
	if err := d.VerifyCredential(preCtx); err != nil {
		return false, err
	}

	if preCtx.GetOperation() == model.Read {
		return true, nil
	}

	tokenInfo := preCtx.GetAttachment(model.TokenDetailInfoKey).(OperatorInfo)

	if tokenInfo.Disable {
		return false, model.ErrorTokenDisabled
	}

	if !tokenInfo.IsUserToken {
		return false, errors.New("only user role can access maintain API")
	}

	if tokenInfo.Role != model.OwnerUserRole {
		return false, errors.New("only owner account can access maintain API")
	}

	return true, nil
}

// CheckPermission 执行检查动作判断是否有权限
//
//	step 1. 判断是否开启了鉴权
//	step 2. 对token进行检查判断
//		case 1. 如果 token 被禁用
//				a. 读操作，直接放通
//				b. 写操作，快速失败
//	step 3. 拉取token对应的操作者相关信息，注入到请求上下文中
//	step 4. 进行权限检查
func (d *defaultAuthChecker) CheckPermission(authCtx *model.AcquireContext) (bool, error) {
	reqId := utils.ParseRequestID(authCtx.GetRequestContext())
	if err := d.VerifyCredential(authCtx); err != nil {
		return false, err
	}

	if authCtx.GetOperation() == model.Read {
		return true, nil
	}

	operatorInfo := authCtx.GetAttachment(model.TokenDetailInfoKey).(OperatorInfo)
	// 这里需要检查当 token 被禁止的情况，如果 token 被禁止，无论是否可以操作目标资源，都无法进行写操作
	if operatorInfo.Disable {
		return false, model.ErrorTokenDisabled
	}

	ok, err := d.doCheckPermission(authCtx)
	if ok {
		return ok, nil
	}

	// 强制同步一次db中strategy数据到cache
	err = d.cacheMgn.AuthStrategy().ForceSyncStrategy2Cache()
	if err != nil {
		log.Error("[Auth][Checker] check permission args", utils.ZapRequestID(reqId),
			zap.String("method", authCtx.GetMethod()), zap.Any("resources", authCtx.GetAccessResources()))
		log.Error("[Auth][Checker] force sync strategy to cache failed", utils.ZapRequestID(reqId), zap.Error(err))
		return false, err
	}

	return d.doCheckPermission(authCtx)
}

func canDowngradeAnonymous(authCtx *model.AcquireContext, err error) bool {
	if authCtx.GetModule() == model.AuthModule {
		return false
	}

	if AuthOption.Strict {
		return false
	}

	if errors.Is(err, model.ErrorTokenInvalid) {
		return true
	}
	if errors.Is(err, model.ErrorTokenNotExist) {
		return true
	}

	return false
}

// VerifyCredential 对 token 进行检查验证，并将 verify 过程中解析出的数据注入到 model.AcquireContext 中
// step 1. 首先对 token 进行解析，获取相关的数据信息，注入到整个的 AcquireContext 中
// step 2. 最后对 token 进行一些验证步骤的执行
// step 3. 兜底措施：如果开启了鉴权的非严格模式，则根据错误的类型，判断是否转为匿名用户进行访问
//   - 如果不是访问权限控制相关模块（用户、用户组、权限策略），不得转为匿名用户
func (d *defaultAuthChecker) VerifyCredential(authCtx *model.AcquireContext) error {
	reqId := utils.ParseRequestID(authCtx.GetRequestContext())

	checkErr := func() error {
		authToken := utils.ParseAuthToken(authCtx.GetRequestContext())
		operator, err := d.decodeToken(authToken)
		if err != nil {
			log.Error("[Auth][Checker] decode token", zap.Error(err))
			return model.ErrorTokenInvalid
		}

		ownerId, isOwner, err := d.checkToken(&operator)
		if err != nil {
			log.Errorf("[Auth][Checker] check token err : %s", errors.WithStack(err).Error())
			return err
		}

		operator.OwnerID = ownerId
		ctx := authCtx.GetRequestContext()
		ctx = context.WithValue(ctx, utils.ContextIsOwnerKey, isOwner)
		ctx = context.WithValue(ctx, utils.ContextUserIDKey, operator.OperatorID)
		ctx = context.WithValue(ctx, utils.ContextOwnerIDKey, ownerId)
		// 注入用户名信息
		user := d.Cache().User().GetUserByID(operator.OperatorID)
		if user != nil {
			ctx = context.WithValue(ctx, utils.ContextUserNameKey, user.Name)
		}

		authCtx.SetRequestContext(ctx)
		d.parseOperatorInfo(operator, authCtx)
		if operator.Disable {
			log.Warn("[Auth][Checker] token already disabled", utils.ZapRequestID(reqId),
				zap.Any("token", operator.String()))
		}
		return nil
	}()

	if checkErr != nil {
		if !canDowngradeAnonymous(authCtx, checkErr) {
			return checkErr
		}
		log.Warn("[Auth][Checker] parse operator info, downgrade to anonymous", utils.ZapRequestID(reqId),
			zap.Error(checkErr))
		// 操作者信息解析失败，降级为匿名用户
		authCtx.SetAttachment(model.TokenDetailInfoKey, newAnonymous())
	}

	return nil
}

func (d *defaultAuthChecker) parseOperatorInfo(operator OperatorInfo, authCtx *model.AcquireContext) {
	ctx := authCtx.GetRequestContext()
	if operator.IsUserToken {
		user := d.Cache().User().GetUserByID(operator.OperatorID)
		operator.Role = user.Type
		ctx = context.WithValue(ctx, utils.ContextUserRoleIDKey, user.Type)
	}

	authCtx.SetAttachment(model.OperatorRoleKey, operator.Role)
	authCtx.SetAttachment(model.OperatorPrincipalType, func() model.PrincipalType {
		if operator.IsUserToken {
			return model.PrincipalUser
		}
		return model.PrincipalGroup
	}())
	authCtx.SetAttachment(model.OperatorIDKey, operator.OperatorID)
	authCtx.SetAttachment(model.OperatorOwnerKey, operator)
	authCtx.SetAttachment(model.TokenDetailInfoKey, operator)

	authCtx.SetRequestContext(ctx)
}

// decodeToken 解析 token 信息，如果 t == ""，直接返回一个空对象
func (d *defaultAuthChecker) decodeToken(t string) (OperatorInfo, error) {
	if t == "" {
		return OperatorInfo{}, model.ErrorTokenInvalid
	}

	ret, err := decryptMessage([]byte(AuthOption.Salt), t)
	if err != nil {
		return OperatorInfo{}, err
	}
	tokenDetails := strings.Split(ret, TokenSplit)
	if len(tokenDetails) != 2 {
		return OperatorInfo{}, model.ErrorTokenInvalid
	}

	detail := strings.Split(tokenDetails[1], "/")
	if len(detail) != 2 {
		return OperatorInfo{}, model.ErrorTokenInvalid
	}

	tokenInfo := OperatorInfo{
		Origin:      t,
		IsUserToken: detail[0] == model.TokenForUser,
		OperatorID:  detail[1],
		Role:        model.UnknownUserRole,
	}
	return tokenInfo, nil
}

// checkToken 对 token 进行检查，如果 token 是一个空，直接返回默认值，但是不返回错误
// return {owner-id} {is-owner} {error}
func (d *defaultAuthChecker) checkToken(tokenInfo *OperatorInfo) (string, bool, error) {
	if IsEmptyOperator(*tokenInfo) {
		return "", false, nil
	}

	id := tokenInfo.OperatorID
	if tokenInfo.IsUserToken {
		user := d.Cache().User().GetUserByID(id)
		if user == nil {
			return "", false, model.ErrorNoUser
		}

		if tokenInfo.Origin != user.Token {
			return "", false, model.ErrorTokenNotExist
		}

		tokenInfo.Disable = !user.TokenEnable
		if user.Owner == "" {
			return user.ID, true, nil
		}

		return user.Owner, false, nil
	}
	group := d.Cache().User().GetGroup(id)
	if group == nil {
		return "", false, model.ErrorNoUserGroup
	}

	if tokenInfo.Origin != group.Token {
		return "", false, model.ErrorTokenNotExist
	}

	tokenInfo.Disable = !group.TokenEnable
	return group.Owner, false, nil
}

// findStrategiesByUserID 根据 user-id 查找相关联的鉴权策略（用户自己的 + 用户所在用户组的）
func (d *defaultAuthChecker) findStrategiesByUserID(userId string) []*model.StrategyDetail {
	// Step 1, first pull all the strategy information involved in this user.
	rules := d.cacheMgn.AuthStrategy().GetStrategyDetailsByUID(userId)

	// Step 2, pull the Group information to which this user belongs
	groupIds := d.cacheMgn.User().GetUserLinkGroupIds(userId)
	for i := range groupIds {
		ret := d.findStrategiesByGroupID(groupIds[i])
		rules = append(rules, ret...)
	}

	// Take the strategy that pulls down
	temp := make(map[string]*model.StrategyDetail, len(rules))
	for i := range rules {
		rule := rules[i]
		temp[rule.ID] = rule
	}

	ret := make([]*model.StrategyDetail, 0, len(temp))
	for _, val := range temp {
		ret = append(ret, val)
	}

	return ret
}

// findStrategies Inquire about TOKEN information, the actual all-associated authentication strategy
func (d *defaultAuthChecker) findStrategies(tokenInfo OperatorInfo) ([]*model.StrategyDetail, error) {
	if IsEmptyOperator(tokenInfo) {
		return make([]*model.StrategyDetail, 0), nil
	}

	var strategies []*model.StrategyDetail
	if tokenInfo.IsUserToken {
		user := d.cacheMgn.User().GetUserByID(tokenInfo.OperatorID)
		if user == nil {
			return nil, model.ErrorNoUser
		}

		strategies = d.findStrategiesByUserID(tokenInfo.OperatorID)
	} else {
		group := d.cacheMgn.User().GetGroup(tokenInfo.OperatorID)
		if group == nil {
			return nil, model.ErrorNoUserGroup
		}

		strategies = d.findStrategiesByGroupID(tokenInfo.OperatorID)
	}

	return strategies, nil
}

// findStrategiesByGroupID 根据 group-id 查找相关联的鉴权策略
func (d *defaultAuthChecker) findStrategiesByGroupID(id string) []*model.StrategyDetail {
	return d.cacheMgn.AuthStrategy().GetStrategyDetailsByGroupID(id)
}

// removeNoStrategyResources 移除没有关联任何鉴权策略的资源
func (d *defaultAuthChecker) removeNoStrategyResources(authCtx *model.AcquireContext) bool {
	reqId := utils.ParseRequestID(authCtx.GetRequestContext())
	resources := authCtx.GetAccessResources()
	cacheMgn := d.Cache()
	newAccessRes := make(map[apisecurity.ResourceType][]model.ResourceEntry, 0)
	checkIsFree := func(resType apisecurity.ResourceType, entry model.ResourceEntry) bool {
		// if entry.Owner == "" ||
		// 	strings.Compare(strings.ToLower(entry.Owner), strings.ToLower("polaris")) == 0 {
		// 	return true
		// }
		return !cacheMgn.AuthStrategy().IsResourceLinkStrategy(resType, entry.ID)
	}

	// 检查命名空间
	nsRes := resources[apisecurity.ResourceType_Namespaces]
	newNsRes := make([]model.ResourceEntry, 0)
	for index := range nsRes {
		if checkIsFree(apisecurity.ResourceType_Namespaces, nsRes[index]) {
			continue
		}

		newNsRes = append(newNsRes, nsRes[index])
	}

	newAccessRes[apisecurity.ResourceType_Namespaces] = newNsRes
	if authCtx.GetModule() == model.DiscoverModule {
		// 检查服务
		svcRes := resources[apisecurity.ResourceType_Services]
		newSvcRes := make([]model.ResourceEntry, 0)
		for index := range svcRes {
			if checkIsFree(apisecurity.ResourceType_Services, svcRes[index]) {
				continue
			}

			newSvcRes = append(newSvcRes, svcRes[index])
		}
		newAccessRes[apisecurity.ResourceType_Services] = newSvcRes
	}

	if authCtx.GetModule() == model.ConfigModule {
		// 检查配置
		cfgRes := resources[apisecurity.ResourceType_ConfigGroups]
		newCfgRes := make([]model.ResourceEntry, 0)
		for index := range cfgRes {
			if checkIsFree(apisecurity.ResourceType_ConfigGroups, cfgRes[index]) {
				continue
			}
			newCfgRes = append(newCfgRes, cfgRes[index])
		}
		newAccessRes[apisecurity.ResourceType_ConfigGroups] = newCfgRes
	}

	log.Info("[Auth][Checker] remove no link strategy final result", utils.ZapRequestID(reqId),
		zap.Any("resource", newAccessRes))

	authCtx.SetAccessResources(newAccessRes)
	noResourceNeedCheck := authCtx.IsAccessResourceEmpty()
	if noResourceNeedCheck {
		log.Debug("[Auth][Checker]", utils.ZapRequestID(reqId),
			zap.String("msg", "need check permission resource is empty"))
	}

	return noResourceNeedCheck
}

func (d *defaultAuthChecker) isResourceEditable(
	userid string,
	resourceType apisecurity.ResourceType,
	resEntries []model.ResourceEntry) bool {
	for _, entry := range resEntries {
		principal := model.Principal{
			PrincipalID:   userid,
			PrincipalRole: model.PrincipalUser,
		}
		if !d.cacheMgn.AuthStrategy().IsResourceEditable(principal, resourceType, entry.ID) {
			return false
		}
	}
	return true
}

// doCheckPermission 执行权限检查
func (d *defaultAuthChecker) doCheckPermission(authCtx *model.AcquireContext) (bool, error) {

	userId := utils.ParseUserID(authCtx.GetRequestContext())

	var checkNamespace, checkSvc, checkCfgGroup bool

	reqRes := authCtx.GetAccessResources()
	nsResEntries := reqRes[apisecurity.ResourceType_Namespaces]
	svcResEntries := reqRes[apisecurity.ResourceType_Services]
	cfgResEntries := reqRes[apisecurity.ResourceType_ConfigGroups]

	checkNamespace = d.isResourceEditable(userId, apisecurity.ResourceType_Namespaces, nsResEntries)
	checkSvc = d.isResourceEditable(userId, apisecurity.ResourceType_Services, svcResEntries)
	checkCfgGroup = d.isResourceEditable(userId, apisecurity.ResourceType_ConfigGroups, cfgResEntries)

	checkAllResEntries := checkNamespace && checkSvc && checkCfgGroup

	return checkAllResEntries, nil
}

// checkAction 检查操作是否和策略匹配
func (d *defaultAuthChecker) checkAction(expect string, actual model.ResourceOperation) bool {
	return true
}

// checkAnyElementExist 检查待操作的资源是否符合鉴权资源列表的配置
//
//	@param userId 当前的用户信息
//	@param waitSearch 访问的资源
//	@param searchMaps 鉴权策略中某一类型的资源列表信息
//	@return bool 是否可以操作本次被访问的所有资源
func checkAnyElementExist(userId string, waitSearch []model.ResourceEntry, searchMaps *searchMap) bool {
	if len(waitSearch) == 0 || searchMaps.passAll {
		return true
	}

	for _, entry := range waitSearch {
		if entry.Owner == userId {
			continue
		}

		if _, ok := searchMaps.items[entry.ID]; !ok {
			return false
		}
	}

	return true
}

// emptyVal 空对象，占位而已
var emptyVal = struct{}{}

// buildSearchMap 构建搜索 map
func buildSearchMap(ss []model.StrategyResource) []*searchMap {

	nsSearchMaps := &searchMap{
		items:   make(map[string]struct{}),
		passAll: false,
	}
	svcSearchMaps := &searchMap{
		items:   make(map[string]struct{}),
		passAll: false,
	}
	cfgSearchMaps := &searchMap{
		items:   make(map[string]struct{}),
		passAll: false,
	}

	for _, val := range ss {
		if val.ResType == int32(apisecurity.ResourceType_Namespaces) {
			nsSearchMaps.items[val.ResID] = emptyVal
			nsSearchMaps.passAll = (val.ResID == "*") || nsSearchMaps.passAll
			continue
		}
		if val.ResType == int32(apisecurity.ResourceType_Services) {
			svcSearchMaps.items[val.ResID] = emptyVal
			svcSearchMaps.passAll = (val.ResID == "*") || nsSearchMaps.passAll
			continue
		}
		if val.ResType == int32(apisecurity.ResourceType_ConfigGroups) {
			cfgSearchMaps.items[val.ResID] = emptyVal
			cfgSearchMaps.passAll = (val.ResID == "*") || nsSearchMaps.passAll
			continue
		}
	}

	return []*searchMap{nsSearchMaps, svcSearchMaps, cfgSearchMaps}
}

// searchMap 权限搜索map
type searchMap struct {
	// 某个资源策略的去重map
	items map[string]struct{}
	// 该资源策略是否允许全部操作
	passAll bool
}
