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

	"github.com/polarismesh/polaris/common/api/l5"
)

// RegisterByNameCmd implements service.DiscoverServer.
func (svr *Server) RegisterByNameCmd(rbnc *l5.Cl5RegisterByNameCmd) (*l5.Cl5RegisterByNameAckCmd, error) {
	return svr.nextSvr.RegisterByNameCmd(rbnc)
}

// SyncByAgentCmd implements service.DiscoverServer.
func (svr *Server) SyncByAgentCmd(ctx context.Context, sbac *l5.Cl5SyncByAgentCmd) (*l5.Cl5SyncByAgentAckCmd, error) {
	return svr.nextSvr.SyncByAgentCmd(ctx, sbac)
}
