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

package model

// Route 访问关系
type Route struct {
	IP    uint32
	ModID uint32
	CmdID uint32
	SetID string
	Valid bool
	Flow  uint32
}

// Policy 有状态规则路由策略信息
type Policy struct {
	ModID uint32
	Div   uint32
	Mod   uint32
	Valid bool
	Flow  uint32
}

// Section 有状态规则路由分段信息
type Section struct {
	ModID uint32
	From  uint32
	To    uint32
	Xid   uint32
	Valid bool
	Flow  uint32
}

// IPConfig IP的区域信息
type IPConfig struct {
	IP     uint32
	AreaID uint32
	CityID uint32
	IdcID  uint32
	Valid  bool
	Flow   uint32
}

// Sid sid信息
type Sid struct {
	ModID uint32
	CmdID uint32
}

// Callee 被调信息，对应t_server+t_ip_config
type Callee struct {
	ModID    uint32
	CmdID    uint32
	SetID    string
	IP       uint32
	Port     uint32
	Weight   uint32
	Location *Location
	// AreaID uint32
	// CityID uint32
	// IdcID  uint32
}

// SidConfig sid信息，对应t_sid表
type SidConfig struct {
	ModID  uint32
	CmdID  uint32
	Name   string
	Policy uint32
}
