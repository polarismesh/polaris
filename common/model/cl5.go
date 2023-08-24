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

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Cl5ServerCluster cl5集群的ctx的key
type Cl5ServerCluster struct{}

// Cl5ServerProtocol cl5.server的协议ctx
type Cl5ServerProtocol struct{}

// MarshalSid Sid结构体，序列化转为sid字符串
func MarshalSid(sid *Sid) string {
	return fmt.Sprintf("%d:%d", sid.ModID, sid.CmdID)
}

// MarshalModCmd mod cmd转为sid
func MarshalModCmd(modID uint32, cmdID uint32) string {
	return fmt.Sprintf("%d:%d", modID, cmdID)
}

// UnmarshalSid 把sid字符串反序列化为结构体Sid
func UnmarshalSid(sidStr string) (*Sid, error) {
	items := strings.Split(sidStr, ":")
	if len(items) != 2 {
		return nil, errors.New("invalid format sid string")
	}

	modID, err := strconv.ParseUint(items[0], 10, 32)
	if err != nil {
		return nil, err
	}

	var cmdID uint64
	cmdID, err = strconv.ParseUint(items[1], 10, 32)
	if err != nil {
		return nil, err
	}

	out := &Sid{
		ModID: uint32(modID),
		CmdID: uint32(cmdID),
	}
	return out, nil
}
