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
	"errors"
	"regexp"
	"unicode/utf8"

	"github.com/golang/protobuf/ptypes/wrappers"
	"go.uber.org/zap"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/store"
)

var (
	// 必须超级账户 or 主账户
	MustOwner = true
	// 任意账户
	NotOwner = false
	// 写操作
	WriteOp = true
	// 读操作
	ReadOp = false
)

// storeCodeAPICodeMap 存储层报错与协议层码的映射
var storeCodeAPICodeMap = map[store.StatusCode]uint32{
	store.EmptyParamsErr:             api.InvalidParameter,
	store.OutOfRangeErr:              api.InvalidParameter,
	store.DataConflictErr:            api.DataConflict,
	store.NotFoundNamespace:          api.NotFoundNamespace,
	store.NotFoundService:            api.NotFoundService,
	store.NotFoundMasterConfig:       api.NotFoundMasterConfig,
	store.NotFoundTagConfigOrService: api.NotFoundTagConfigOrService,
	store.ExistReleasedConfig:        api.ExistReleasedConfig,
	store.DuplicateEntryErr:          api.ExistedResource,
}

// StoreCode2APICode store code to api code
func StoreCode2APICode(err error) uint32 {
	code := store.Code(err)
	apiCode, ok := storeCodeAPICodeMap[code]
	if ok {
		return apiCode
	}

	return api.StoreLayerException
}

// checkName 名称检查
func checkName(name *wrappers.StringValue) error {
	if name == nil {
		return errors.New("nil")
	}

	if name.GetValue() == "" {
		return errors.New("empty")
	}

	if name.GetValue() == "polariadmin" {
		return errors.New("illegal username")
	}

	if utf8.RuneCountInString(name.GetValue()) > utils.MaxNameLength {
		return errors.New("name too long")
	}

	regStr := "^[\u4E00-\u9FA5A-Za-z0-9_\\-]+$"
	ok, err := regexp.MatchString(regStr, name.GetValue())
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("name contains invalid character")
	}

	return nil
}

// checkPassword 密码检查
func checkPassword(password *wrappers.StringValue) error {
	if password == nil {
		return errors.New("nil")
	}

	if password.GetValue() == "" {
		return errors.New("empty")
	}

	if len(password.GetValue()) < 6 || len(password.GetValue()) > 17 {
		return errors.New("password len need 6 ~ 17")
	}

	// spcChar := "!@#$%&*"
	// flag := make([]string, len(password.GetValue()))
	// for k, v := range []rune(password.GetValue()) {
	// 	if unicode.IsDigit(v) {
	// 		flag[k] = "Number"
	// 	} else if unicode.IsLower(v) {
	// 		flag[k] = "LowerCaseLetter"
	// 	} else if unicode.IsUpper(v) {
	// 		flag[k] = "UpperCaseLetter"
	// 	} else if strings.Contains(spcChar, string(v)) {
	// 		flag[k] = "SpecialCharacter"
	// 	} else {
	// 		flag[k] = "OtherCharacter"
	// 	}
	// }

	// cpx := make(map[string]bool)
	// for _, v := range flag {
	// 	cpx[v] = true
	// }
	// if len(cpx) < 2 {
	// 	return errors.New("password security is so low")
	// }

	return nil
}

// checkOwner 检查用户的 owner 信息
func checkOwner(owner *wrappers.StringValue) error {
	if owner == nil {
		return errors.New("nil")
	}

	if owner.GetValue() == "" {
		return errors.New("empty")
	}

	if utf8.RuneCountInString(owner.GetValue()) > utils.MaxOwnersLength {
		return errors.New("owners too long")
	}

	return nil
}

// checkMobile 检查用户的 mobile 信息
func checkMobile(mobile *wrappers.StringValue) error {
	if mobile == nil {
		return nil
	}

	if mobile.GetValue() == "" {
		return nil
	}

	if utf8.RuneCountInString(mobile.GetValue()) != 11 {
		return errors.New("invalid mobile")
	}

	return nil
}

// checkEmail 检查用户的 email 信息
func checkEmail(email *wrappers.StringValue) error {
	if email == nil {
		return nil
	}

	if email.GetValue() == "" {
		return nil
	}

	emailReg := regexp.MustCompile(`^\w+([-+.]\w+)*@\w+([-.]\w+)*\.\w+([-.]\w+)*$`)
	if !emailReg.Match([]byte(email.GetValue())) {
		return errors.New("invalid email")
	}

	return nil
}

// verifyAuth 用于 user、group 以及 strategy 模块的鉴权工作检查
func (svr *serverAuthAbility) verifyAuth(ctx context.Context, isWrite bool,
	needOwner bool) (context.Context, *api.Response) {

	reqId := utils.ParseRequestID(ctx)
	authToken := utils.ParseAuthToken(ctx)

	if authToken == "" {
		log.AuthScope().Error("[Auth][Server] auth token is empty", utils.ZapRequestID(reqId))
		return nil, api.NewResponse(api.EmptyAutToken)
	}

	authCtx := model.NewAcquireContext(
		model.WithRequestContext(ctx),
		model.WithToken(authToken),
		model.WithModule(model.AuthModule),
	)

	err := svr.authMgn.VerifyToken(authCtx)

	// case 1. 如果 error 不是 token 被禁止的 error，直接返回
	// case 2. 如果 error 是 token 被禁止，按下面情况判断
	// 		i. 如果当前只是一个数据的读取操作，则放通
	// 		ii. 如果当前是一个数据的写操作，则只能允许处于正常的 token 进行操作

	if err != nil {
		if !errors.Is(err, model.ErrorTokenDisabled) {
			log.AuthScope().Error("[Auth][Server] verify auth token", utils.ZapRequestID(reqId),
				zap.Error(err))
			return nil, api.NewResponse(api.AuthTokenVerifyException)
		}

		if isWrite {
			log.AuthScope().Error("[Auth][Server] token is disabled and op is write", utils.ZapRequestID(reqId),
				zap.Error(err))
			return nil, api.NewResponse(api.TokenDisabled)
		}
	}

	tokenInfo := authCtx.GetAttachment()[model.TokenDetailInfoKey].(TokenInfo)

	if !tokenInfo.IsUserToken {
		log.AuthScope().Error("[Auth][Server] only user role can access this API", utils.ZapRequestID(reqId))
		return nil, api.NewResponse(api.OperationRoleException)
	}

	if needOwner && tokenInfo.IsSubAccount() {
		log.AuthScope().Error("[Auth][Server] only admin/owner account can access this API", utils.ZapRequestID(reqId))
		return nil, api.NewResponse(api.OperationRoleException)
	}

	return authCtx.GetRequestContext(), nil
}
