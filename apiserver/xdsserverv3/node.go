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
	"strconv"
	"strings"
	"sync"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	structpb "github.com/golang/protobuf/ptypes/struct"
)

func newXDSNodeManager() *XDSNodeManager {
	return &XDSNodeManager{
		nodes:         map[string]*XDSClient{},
		streamTonodes: map[int64]*XDSClient{},
		sidecarNodes:  map[string]*XDSClient{},
		gatewayNodes:  map[string]*XDSClient{},
	}
}

type XDSNodeManager struct {
	lock          sync.RWMutex
	nodes         map[string]*XDSClient
	streamTonodes map[int64]*XDSClient
	// sidecarNodes The XDS client is the node list of the SIDECAR run mode
	sidecarNodes map[string]*XDSClient
	// gatewayNodes The XDS client is the node list of the Gateway run mode
	gatewayNodes map[string]*XDSClient
}

func (x *XDSNodeManager) AddNodeIfAbsent(streamId int64, node *core.Node) {
	if node == nil {
		return
	}
	x.lock.Lock()
	defer x.lock.Unlock()

	p := parseNodeProxy(node)

	if _, ok := x.streamTonodes[streamId]; !ok {
		x.streamTonodes[streamId] = p
	}
	if _, ok := x.nodes[node.Id]; !ok {
		x.nodes[node.Id] = p
	}

	switch p.RunType {
	case RunTypeGateway:
		if _, ok := x.gatewayNodes[node.Id]; !ok {
			x.gatewayNodes[node.Id] = p
		}
	case RunTypeSidecar:
		if _, ok := x.sidecarNodes[node.Id]; !ok {
			x.sidecarNodes[node.Id] = p
		}
	}
}

func (x *XDSNodeManager) DelNode(streamId int64) {
	x.lock.Lock()
	defer x.lock.Unlock()

	if p, ok := x.streamTonodes[streamId]; ok {
		delete(x.nodes, p.Node.Id)
	}
	delete(x.streamTonodes, streamId)
}

func (x *XDSNodeManager) GetNodeByStreamID(streamId int64) *XDSClient {
	x.lock.RLock()
	defer x.lock.RUnlock()

	return x.streamTonodes[streamId]
}

func (x *XDSNodeManager) GetNode(id string) *XDSClient {
	x.lock.RLock()
	defer x.lock.RUnlock()

	return x.nodes[id]
}

func (x *XDSNodeManager) ListGatewayNodes() []*XDSClient {
	x.lock.RLock()
	defer x.lock.RUnlock()

	ret := make([]*XDSClient, 0, len(x.gatewayNodes))
	for i := range x.gatewayNodes {
		ret = append(ret, x.gatewayNodes[i])
	}
	return ret
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
	if runType == string(RunTypeSidecar) {
		ret := ns
		if node.Metadata != nil && node.Metadata.Fields != nil {
			tlsMode := node.Metadata.Fields[TLSModeTag].GetStringValue()
			if tlsMode == TLSModePermissive || tlsMode == TLSModeStrict {
				return ret + "/" + tlsMode
			}
		}
		return ret
	}
	return node.Id
}

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
	if strings.Contains(prefixInfo, sep) {
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

type RunType string

var (
	// RunTypeGateway xds node run type is gateway
	RunTypeGateway RunType = "gateway"
	// RunTypeSidecar xds node run type is sidecar
	RunTypeSidecar RunType = "sidecar"
)

const (
	sep = "~"
	// GatewayNamespaceName xds metadata key
	GatewayNamespaceName = "gateway_namespace"
	// GatewayNamespaceName xds metadata key
	GatewayServiceName = "gateway_service"
)

// XDSClient 客户端代码结构体
type XDSClient struct {
	RunType   RunType
	User      string
	Namespace string
	IPAddr    string
	Metadata  map[string]string
	Version   string
	Node      *core.Node

	lock sync.Mutex
	once map[string]*sync.Once
}

func (n *XDSClient) RunOnce(key string, f func()) {
	n.lock.Lock()
	defer n.lock.Unlock()

	if _, ok := n.once[key]; !ok {
		n.once[key] = &sync.Once{}
	}

	n.once[key].Do(f)
}

func (n *XDSClient) IsGateway() bool {
	service := n.Metadata[GatewayServiceName]
	namespace := n.Metadata[GatewayNamespaceName]
	return n.RunType == RunTypeGateway && service != "" && namespace != ""
}

func parseNodeProxy(node *core.Node) *XDSClient {
	runType, polarisNamespace, _, hostIP := parseNodeID(node.Id)
	proxy := &XDSClient{
		IPAddr:    hostIP,
		RunType:   RunType(runType),
		Namespace: polarisNamespace,
		Node:      node,
		once:      make(map[string]*sync.Once),
	}
	proxy.Metadata = parseMetadata(node.GetMetadata())
	return proxy
}

func parseMetadata(metaValues *structpb.Struct) map[string]string {
	fields := metaValues.GetFields()
	if len(fields) == 0 {
		return nil
	}
	var values = make(map[string]string, len(fields))
	for fieldKey, fieldValue := range fields {
		switch fieldValue.GetKind().(type) {
		case *structpb.Value_StringValue:
			values[fieldKey] = fieldValue.GetStringValue()
		case *structpb.Value_NumberValue:
			values[fieldKey] = strconv.FormatInt(int64(fieldValue.GetNumberValue()), 10)
		case *structpb.Value_BoolValue:
			values[fieldKey] = strconv.FormatBool(fieldValue.GetBoolValue())
		}
	}
	return values
}
