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

package auth

import (
	"context"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

// ParseUserRole 从ctx中解析用户角色
func ParseUserRole(ctx context.Context) model.UserRoleType {
	if ctx == nil {
		return model.SubAccountUserRole
	}

	role, _ := ctx.Value(utils.ContextUserRoleIDKey).(model.UserRoleType)
	return role
}
