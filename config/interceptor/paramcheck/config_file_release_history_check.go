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

package paramcheck

import (
	"context"
	"strconv"

	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/utils"
)

// GetConfigFileReleaseHistory 获取配置文件发布历史记录
func (s *Server) GetConfigFileReleaseHistories(ctx context.Context,
	filter map[string]string) *apiconfig.ConfigBatchQueryResponse {

	offset, limit, err := utils.ParseOffsetAndLimit(filter)
	if err != nil {
		return api.NewConfigBatchQueryResponseWithInfo(apimodel.Code_BadRequest, err.Error())
	}

	searchFilters := map[string]string{
		"offset": strconv.FormatInt(int64(offset), 10),
		"limit":  strconv.FormatInt(int64(limit), 10),
	}

	for k, v := range filter {
		if nk, ok := availableSearch["config_file_release_history"][k]; ok {
			searchFilters[nk] = v
		}
	}

	return s.nextServer.GetConfigFileReleaseHistories(ctx, searchFilters)
}
