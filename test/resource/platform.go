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

package resource

import (
	"fmt"
	"time"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/utils"
)

/**
 * @brief 创建测试平台
 */
func CreatePlatforms() []*api.Platform {
	nowStr := time.Now().Format("2006-01-02T15-04-05")
	var platforms []*api.Platform
	for index := 1; index <= 2; index++ {
		platform := &api.Platform{
			Id:         utils.NewStringValue(fmt.Sprintf("id-%v-%v", index, nowStr)),
			Name:       utils.NewStringValue(fmt.Sprintf("name-%v", nowStr)),
			Domain:     utils.NewStringValue(fmt.Sprintf("domain-%v-%v", index, nowStr)),
			Qps:        utils.NewUInt32Value(uint32(index)),
			Owner:      utils.NewStringValue(fmt.Sprintf("owner-%v-%v", index, nowStr)),
			Department: utils.NewStringValue(fmt.Sprintf("department-%v-%v", index, nowStr)),
			Comment:    utils.NewStringValue(fmt.Sprintf("comment-%v-%v", index, nowStr)),
		}
		platforms = append(platforms, platform)
	}

	return platforms
}

/**
 * @brief 更新测试平台
 */
func UpdatePlatforms(platforms []*api.Platform) {
	for _, platform := range platforms {
		platform.Name = utils.NewStringValue("update-name")
		platform.Domain = utils.NewStringValue("update-domain")
		platform.Qps = utils.NewUInt32Value(platform.GetQps().GetValue() + 1)
		platform.Owner = utils.NewStringValue("update-owner")
		platform.Department = utils.NewStringValue("update-department")
		platform.Comment = utils.NewStringValue("update-comment")
	}
}
