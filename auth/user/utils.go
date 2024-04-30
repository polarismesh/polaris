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

package defaultuser

import (
	"errors"
	"regexp"
	"unicode/utf8"

	"github.com/golang/protobuf/ptypes/wrappers"

	"github.com/polarismesh/polaris/common/utils"
)

var (
	// MustOwner 必须超级账户 or 主账户
	MustOwner = true
	// NotOwner 任意账户
	NotOwner = false
	// WriteOp 写操作
	WriteOp = true
	// ReadOp 读操作
	ReadOp = false
)

var (
	regNameStr = regexp.MustCompile("^[\u4E00-\u9FA5A-Za-z0-9_\\-.]+$")
	regEmail   = regexp.MustCompile(`^\w+([-+.]\w+)*@\w+([-.]\w+)*\.\w+([-.]\w+)*$`)
)

// CheckName 名称检查
func CheckName(name *wrappers.StringValue) error {
	if name == nil {
		return errors.New(utils.NilErrString)
	}

	if name.GetValue() == "" {
		return errors.New(utils.EmptyErrString)
	}

	if name.GetValue() == "polariadmin" {
		return errors.New("illegal username")
	}

	if utf8.RuneCountInString(name.GetValue()) > utils.MaxNameLength {
		return errors.New("name too long")
	}

	if ok := regNameStr.MatchString(name.GetValue()); !ok {
		return errors.New("name contains invalid character")
	}

	return nil
}

// CheckPassword 密码检查
func CheckPassword(password *wrappers.StringValue) error {
	if password == nil {
		return errors.New(utils.NilErrString)
	}

	if password.GetValue() == "" {
		return errors.New(utils.EmptyErrString)
	}

	if pLen := len(password.GetValue()); pLen < 6 || pLen > 17 {
		return errors.New("password len need 6 ~ 17")
	}

	return nil
}

// CheckOwner 检查用户的 owner 信息
func CheckOwner(owner *wrappers.StringValue) error {
	if owner == nil {
		return errors.New(utils.NilErrString)
	}

	if owner.GetValue() == "" {
		return errors.New(utils.EmptyErrString)
	}

	if utf8.RuneCountInString(owner.GetValue()) > utils.MaxOwnersLength {
		return errors.New("owners too long")
	}

	return nil
}
