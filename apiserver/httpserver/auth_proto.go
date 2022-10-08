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

package httpserver

import (
	"github.com/golang/protobuf/proto"

	api "github.com/polarismesh/polaris/common/api/v1"
)

// UserArr 命名空间数组定义
type UserArr []*api.User

// Reset 清空数组
func (m *UserArr) Reset() { *m = UserArr{} }

// String return string
func (m *UserArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage return proto message
func (*UserArr) ProtoMessage() {}

// GroupArr 命名空间数组定义
type GroupArr []*api.UserGroup

// Reset 清空数组
func (m *GroupArr) Reset() { *m = GroupArr{} }

// String return string
func (m *GroupArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage return proto message
func (*GroupArr) ProtoMessage() {}

// ModifyGroupArr 命名空间数组定义
type ModifyGroupArr []*api.ModifyUserGroup

// Reset 清空数组
func (m *ModifyGroupArr) Reset() { *m = ModifyGroupArr{} }

// String return string
func (m *ModifyGroupArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage return proto message
func (*ModifyGroupArr) ProtoMessage() {}

// StrategyArr 命名空间数组定义
type StrategyArr []*api.AuthStrategy

// Reset 清空数组
func (m *StrategyArr) Reset() { *m = StrategyArr{} }

// String return string
func (m *StrategyArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage return proto message
func (*StrategyArr) ProtoMessage() {}

// ModifyStrategyArr 命名空间数组定义
type ModifyStrategyArr []*api.ModifyAuthStrategy

// Reset 清空数组
func (m *ModifyStrategyArr) Reset() { *m = ModifyStrategyArr{} }

// String return string
func (m *ModifyStrategyArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage return proto message
func (*ModifyStrategyArr) ProtoMessage() {}

// AuthResourceArr 命名空间数组定义
type AuthResourceArr []*api.StrategyResources

// Reset 清空数组
func (m *AuthResourceArr) Reset() { *m = AuthResourceArr{} }

// String return string
func (m *AuthResourceArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage return proto message
func (*AuthResourceArr) ProtoMessage() {}
