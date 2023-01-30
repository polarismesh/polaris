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

package grpc

import (
	"context"

	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
)

func (c *Client) GetConfigFile(ctx context.Context, in *apiconfig.ClientConfigFileInfo) (*apiconfig.ConfigClientResponse, error) {
	return nil, nil
}

// 订阅配置变更
func (c *Client) WatchConfigFiles(ctx context.Context, in *apiconfig.ClientWatchConfigFileRequest) (*apiconfig.ConfigClientResponse, error) {
	return nil, nil
}
