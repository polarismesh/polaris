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

package service_auth

import (
	"context"

	"github.com/polarismesh/polaris/common/api/l5"
)

// SyncByAgentCmd 根据sid获取路由信息
// 老函数：
// Stat::instance()->inc_sync_req_cnt();
// 保存client的IP，该函数只是存储到本地的缓存中
// Stat::instance()->add_agent(sbac.agent_ip());
func (svr *Server) SyncByAgentCmd(ctx context.Context, sbac *l5.Cl5SyncByAgentCmd) (
	*l5.Cl5SyncByAgentAckCmd, error) {
	return svr.nextSvr.SyncByAgentCmd(ctx, sbac)
}

// RegisterByNameCmd 根据名字获取sid信息
func (svr *Server) RegisterByNameCmd(rbnc *l5.Cl5RegisterByNameCmd) (*l5.Cl5RegisterByNameAckCmd, error) {
	return svr.nextSvr.RegisterByNameCmd(rbnc)
}
