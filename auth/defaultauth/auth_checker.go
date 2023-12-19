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
	cachetypes "github.com/polarismesh/polaris/cache/api"
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
	// ErrorNotPermission .
	ErrorNotPermission = errors.New("no permission")
)

// DefaultAuthChecker 北极星自带的默认鉴权中心
type DefaultAuthChecker struct {
	cacheMgn cachetypes.CacheManager
}

func (d *DefaultAuthChecker) SetCacheMgr(mgr cachetypes.CacheManager) {
	d.cacheMgn = mgr
}

// Initialize 执行初始化动作
func (d *DefaultAuthChecker) Initialize(options *auth.Config, s store.Store, cacheMgr cachetypes.CacheManager) error {
	// 新版本鉴权策略配置均从auth.Option中迁移至auth.user.option及auth.strategy.option中
	var (
		strategyContentBytes []byte
		userContentBytes     []byte
		authContentBytes     []byte
		err                  error
	)

	cfg := DefaultAuthConfig()

	// 一旦设置了auth.user.option或auth.strategy.option，将不会继续读取auth.option
	if len(options.Strategy.Option) > 0 || len(options.User.Option) > 0 {
		// 判断auth.option是否还有值，有则不兼容
		if len(options.Option) > 0 {
			log.Warn("auth.user.option or auth.strategy.option has set, auth.option will ignore")
		}
		strategyContentBytes, err = json.Marshal(options.Strategy.Option)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(strategyContentBytes, cfg); err != nil {
			return err
		}
		userContentBytes, err = json.Marshal(options.User.Option)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(userContentBytes, cfg); err != nil {
			return err
		}
	} else {
		log.Warn("[Auth][Checker] auth.option has deprecated, use auth.user.option and auth.strategy.option instead.")
		authContentBytes, err = json.Marshal(options.Option)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(authContentBytes, cfg); err != nil {
			return err
		}
	}

	if err := cfg.Verify(); err != nil {
		return err
	}
	// 兼容原本老的配置逻辑
	if cfg.Strict {
		cfg.ConsoleOpen = cfg.Strict
	}
	AuthOption = cfg
	d.cacheMgn = cacheMgr
	return nil
}

// Cache 获取缓存统一管理
func (d *DefaultAuthChecker) Cache() cachetypes.CacheManager {
	return d.cacheMgn
}

// IsOpenConsoleAuth 针对控制台是否开启了操作鉴权
func (d *DefaultAuthChecker) IsOpenConsoleAuth() bool {
	return AuthOption.ConsoleOpen
}

// IsOpenClientAuth 针对客户端是否开启了操作鉴权
func (d *DefaultAuthChecker) IsOpenClientAuth() bool {
	return AuthOption.ClientOpen
}

// IsOpenAuth 返回对于控制台/客户端任意其中的一个是否开启了操作鉴权
func (d *DefaultAuthChecker) IsOpenAuth() bool {
	return d.IsOpenConsoleAuth() || d.IsOpenClientAuth()
}

// CheckClientPermission 执行检查客户端动作判断是否有权限，并且对 RequestContext 注入操作者数据
func (d *DefaultAuthChecker) CheckClientPermission(preCtx *model.AcquireContext) (bool, error) {
	preCtx.SetFromClient()
	if !d.IsOpenClientAuth() {
		return true, nil
	}
	return d.CheckPermission(preCtx)
}

// CheckConsolePermission 执行检查控制台动作判断是否有权限，并且对 RequestContext 注入操作者数据
func (d *DefaultAuthChecker) CheckConsolePermission(preCtx *model.AcquireContext) (bool, error) {
	preCtx.SetFromConsole()
	if !d.IsOpenConsoleAuth() {
		return true, nil
	}
	if preCtx.GetModule() == model.MaintainModule {
		return d.checkMaintainPermission(preCtx)
	}
	return d.CheckPermission(preCtx)
}

// CheckMaintainPermission 执行检查运维动作判断是否有权限
func (d *DefaultAuthChecker) checkMaintainPermission(preCtx *model.AcquireContext) (bool, error) {
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
func (d *DefaultAuthChecker) CheckPermission(authCtx *model.AcquireContext) (bool, error) {
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

	log.Debug("[Auth][Checker] check permission args", utils.RequestID(authCtx.GetRequestContext()),
		zap.String("method", authCtx.GetMethod()), zap.Any("resources", authCtx.GetAccessResources()))

	ok, err := d.doCheckPermission(authCtx)
	if ok {
		return ok, nil
	}

	// 强制同步一次db中strategy数据到cache
	if err = d.cacheMgn.AuthStrategy().ForceSync(); err != nil {
		log.Error("[Auth][Checker] force sync strategy to cache failed",
			utils.RequestID(authCtx.GetRequestContext()), zap.Error(err))
		return false, err
	}

	return d.doCheckPermission(authCtx)
}

func canDowngradeAnonymous(authCtx *model.AcquireContext, err error) bool {
	if authCtx.GetModule() == model.AuthModule {
		return false
	}
	if authCtx.IsFromClient() && AuthOption.ClientStrict {
		return false
	}
	if authCtx.IsFromConsole() && AuthOption.ConsoleStrict {
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
//   - 如果是访问权限控制相关模块（用户、用户组、权限策略），不得转为匿名用户
func (d *DefaultAuthChecker) VerifyCredential(authCtx *model.AcquireContext) error {
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

func (d *DefaultAuthChecker) parseOperatorInfo(operator OperatorInfo, authCtx *model.AcquireContext) {
	ctx := authCtx.GetRequestContext()
	if operator.IsUserToken {
		user := d.Cache().User().GetUserByID(operator.OperatorID)
		if user != nil {
			operator.Role = user.Type
			ctx = context.WithValue(ctx, utils.ContextOperator, user.Name)
			ctx = context.WithValue(ctx, utils.ContextUserNameKey, user.Name)
			ctx = context.WithValue(ctx, utils.ContextUserRoleIDKey, user.Type)
		}
	} else {
		userGroup := d.Cache().User().GetGroup(operator.OperatorID)
		if userGroup != nil {
			ctx = context.WithValue(ctx, utils.ContextOperator, userGroup.Name)
			ctx = context.WithValue(ctx, utils.ContextUserNameKey, userGroup.Name)
		}
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

// DecodeToken
func (d *DefaultAuthChecker) DecodeToken(t string) (OperatorInfo, error) {
	return d.decodeToken(t)
}

// decodeToken 解析 token 信息，如果 t == ""，直接返回一个空对象
func (d *DefaultAuthChecker) decodeToken(t string) (OperatorInfo, error) {
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
func (d *DefaultAuthChecker) checkToken(tokenInfo *OperatorInfo) (string, bool, error) {
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

func (d *DefaultAuthChecker) isResourceEditable(
	principal model.Principal,
	resourceType apisecurity.ResourceType,
	resEntries []model.ResourceEntry) bool {
	for _, entry := range resEntries {
		if !d.cacheMgn.AuthStrategy().IsResourceEditable(principal, resourceType, entry.ID) {
			return false
		}
	}
	return true
}

// doCheckPermission 执行权限检查
func (d *DefaultAuthChecker) doCheckPermission(authCtx *model.AcquireContext) (bool, error) {

	var checkNamespace, checkSvc, checkCfgGroup bool

	reqRes := authCtx.GetAccessResources()
	nsResEntries := reqRes[apisecurity.ResourceType_Namespaces]
	svcResEntries := reqRes[apisecurity.ResourceType_Services]
	cfgResEntries := reqRes[apisecurity.ResourceType_ConfigGroups]

	principleID, _ := authCtx.GetAttachment(model.OperatorIDKey).(string)
	principleType, _ := authCtx.GetAttachment(model.OperatorPrincipalType).(model.PrincipalType)
	p := model.Principal{
		PrincipalID:   principleID,
		PrincipalRole: principleType,
	}
	checkNamespace = d.isResourceEditable(p, apisecurity.ResourceType_Namespaces, nsResEntries)
	checkSvc = d.isResourceEditable(p, apisecurity.ResourceType_Services, svcResEntries)
	checkCfgGroup = d.isResourceEditable(p, apisecurity.ResourceType_ConfigGroups, cfgResEntries)

	checkAllResEntries := checkNamespace && checkSvc && checkCfgGroup

	var err error
	if !checkAllResEntries {
		err = ErrorNotPermission
	}
	return checkAllResEntries, err
}

// checkAction 检查操作是否和策略匹配
func (d *DefaultAuthChecker) checkAction(expect string, actual model.ResourceOperation, method string) bool {
	// TODO 后续可针对读写操作进行鉴权, 并且可以针对具体的方法调用进行鉴权控制
	return true
}
