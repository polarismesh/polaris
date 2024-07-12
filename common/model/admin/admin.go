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

package admin

import (
	"time"

	connlimit "github.com/polarismesh/polaris/common/conn/limit"
)

// LeaderElection leader election info
type LeaderElection struct {
	ElectKey   string
	Host       string
	Ctime      int64
	CreateTime time.Time
	Mtime      int64
	ModifyTime time.Time
	Valid      bool
}

type ConnReq struct {
	Protocol string
	Host     string
	Port     int
	Amount   int
}

type ConnCountResp struct {
	Protocol string
	Total    int32
	Host     map[string]int32
}

type ConnStatsResp struct {
	Protocol        string
	ActiveConnTotal int32
	StatsTotal      int
	StatsSize       int
	Stats           []*connlimit.HostConnStat
}

type ScopeLevel struct {
	Name  string
	Level string
}
