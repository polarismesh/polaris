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

	"github.com/golang/protobuf/ptypes/wrappers"
)

func TestCheckPassword(password *wrappers.StringValue) error {
	return checkPassword(password)
}

func TestCheckName(password *wrappers.StringValue) error {
	return checkName(password)
}

func TestCreateToken(uid, gid string) (string, error) {
	return createToken(uid, gid)
}

func TestDecryptMessage(key []byte, message string) (string, error) {
	return decryptMessage(key, message)
}

func TestParseStrategySearchArgs(ctx context.Context, searchFilters map[string]string) map[string]string {
	return parseStrategySearchArgs(ctx, searchFilters)
}
