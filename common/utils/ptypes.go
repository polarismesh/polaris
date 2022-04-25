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

package utils

import (
	"github.com/golang/protobuf/ptypes/wrappers"
)

// NewStringValue returns a new StringValue with the given value.
func NewStringValue(value string) *wrappers.StringValue {
	return &wrappers.StringValue{Value: value}
}

// NewUInt32Value returns a new UInt32Value with the given value.
func NewUInt32Value(value uint32) *wrappers.UInt32Value {
	return &wrappers.UInt32Value{Value: value}
}

// NewUInt64Value returns a new UInt64Value with the given value.
func NewUInt64Value(value uint64) *wrappers.UInt64Value {
	return &wrappers.UInt64Value{Value: value}
}

// NewBoolValue returns a new BoolValue with the given value.
func NewBoolValue(value bool) *wrappers.BoolValue {
	return &wrappers.BoolValue{Value: value}
}
