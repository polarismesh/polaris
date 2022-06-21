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
 * CONDITIONS OF ANY KIND, either express or serverAuthibilityied. See the License for the
 * specific language governing permissions and limitations under the License.
 */

package config

import (
	"context"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
)

// RecordConfigFileReleaseHistory 新增配置文件发布历史记录
func (s *serverAuthibility) RecordConfigFileReleaseHistory(ctx context.Context, fileRelease *model.ConfigFileRelease, releaseType, status string) {

}

// GetConfigFileReleaseHistory 获取配置文件发布历史记录
func (s *serverAuthibility) GetConfigFileReleaseHistory(ctx context.Context, namespace, group, fileName string, offset,
	limit uint32, endId uint64) *api.ConfigBatchQueryResponse {

	return s.targetServer.GetConfigFileReleaseHistory(ctx, namespace, group, fileName, offset, limit, endId)
}

// GetConfigFileLatestReleaseHistory 获取配置文件最后一次发布记录
func (s *serverAuthibility) GetConfigFileLatestReleaseHistory(ctx context.Context, namespace, group,
	fileName string) *api.ConfigResponse {

	return s.targetServer.GetConfigFileLatestReleaseHistory(ctx, namespace, group, fileName)
}
