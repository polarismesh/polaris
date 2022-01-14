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
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/store"
)

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

// ============ common ============
func checkName(name *wrappers.StringValue) error {
	if name == nil {
		return errors.New("nil")
	}

	if name.GetValue() == "" {
		return errors.New("empty")
	}

	regStr := "^[0-9A-Za-z-.:_]+$"
	ok, err := regexp.MatchString(regStr, name.GetValue())
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("name contains invalid character")
	}

	return nil
}

func checkPassword(password *wrappers.StringValue) error {
	
	if password == nil {
		return errors.New("nil")
	}

	if password.GetValue() == "" {
		return errors.New("empty")
	}

	regStr := "^[a-zA-Z]\\w{5,17}$"
	ok, err := regexp.MatchString(regStr, password.GetValue())
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("password contains invalid character")
	}

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

// verifyAuth token
func verifyAuth(ctx context.Context, authMgn *defaultAuthManager, token string, needOwner bool) (context.Context, *api.Response) {
	ctx, tokenInfo, err := authMgn.verifyToken(ctx, token)

	if err != nil {
		return nil, api.NewResponseWithMsg(api.NotAllowedAccess, err.Error())
	}

	if !tokenInfo.IsUserToken {
		return nil, api.NewResponseWithMsg(api.NotAllowedAccess, "only user role can access this API")
	}

	if needOwner && tokenInfo.IsSubAccount() {
		return nil, api.NewResponseWithMsg(api.NotAllowedAccess, "only admin/owner account can access this API")
	}

	return ctx, nil
}
