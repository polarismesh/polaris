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

package xdsserverv3

import (
	"regexp"
	"strings"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
)

// PolarisNodeHash 存放 hash 方法
type PolarisNodeHash struct{}

// node id 的格式是:
// 1. namespace/uuid~hostIp
var nodeIDFormat = regexp.MustCompile(`^((\S+)~(\S+)|(\S+))\/([^~]+)~([^~]+)$`)

func parseNodeID(nodeID string) (runType, polarisNamespace, uuid, hostIP string) {
	groups := nodeIDFormat.FindStringSubmatch(nodeID)
	if len(groups) == 0 {
		// invalid node format
		return
	}
	prefixInfo := groups[1]
	if strings.Contains(prefixInfo, "~") {
		runType = groups[2]
		polarisNamespace = groups[3]
	} else {
		// 默认为 sidecar 模式
		runType = "sidecar"
		polarisNamespace = groups[1]
	}
	uuid = groups[5]
	hostIP = groups[6]
	return
}

// ID id 的格式是 ${sidecar|gateway}~namespace/uuid~hostIp
// case 1: envoy 为 sidecar 模式时，则 NodeID 的格式为以下两种
//
//	eg 1. namespace/uuid~hostIp
//	eg 2. sidecar~namespace/uuid-hostIp
//
// case 2: envoy 为 gateway 模式时，则 NodeID 的格式为： gateway~namespace/uuid~hostIp
func (PolarisNodeHash) ID(node *core.Node) string {
	if node == nil {
		return ""
	}
	runType, ns, _, _ := parseNodeID(node.Id)
	ret := runType + "~" + ns
	if node.Metadata != nil && node.Metadata.Fields != nil {
		tlsMode := node.Metadata.Fields[TLSModeTag].GetStringValue()
		if tlsMode == TLSModePermissive || tlsMode == TLSModeStrict {
			return ret + "/" + tlsMode
		}
	}

	return ret
}
