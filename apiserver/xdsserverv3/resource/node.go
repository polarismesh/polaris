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

package resource

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"go.uber.org/zap"
)

type RunType string

var (
	// RunTypeGateway xds node run type is gateway
	RunTypeGateway RunType = "gateway"
	// RunTypeSidecar xds node run type is sidecar
	RunTypeSidecar RunType = "sidecar"
)

const (
	sep = "~"
	// GatewayNamespaceName xds metadata key when node is run in gateway mode
	GatewayNamespaceName = "gateway.polarismesh.cn/serviceNamespace"
	// GatewayNamespaceName xds metadata key when node is run in gateway mode
	GatewayServiceName = "gateway.polarismesh.cn/serviceName"
	// OldGatewayNamespaceName xds metadata key when node is run in gateway mode
	OldGatewayNamespaceName = "gateway_namespace"
	// OldGatewayServiceName xds metadata key when node is run in gateway mode
	OldGatewayServiceName = "gateway_service"
	// SidecarServiceName xds metadata key when node is run in sidecar mode
	SidecarServiceName = "sidecar.polarismesh.cn/serviceName"
	// SidecarNamespaceName xds metadata key when node is run in sidecar mode
	SidecarNamespaceName = "sidecar.polarismesh.cn/serviceNamespace"
	// SidecarBindPort xds metadata key when node is run in sidecar mode
	SidecarBindPort = "sidecar.polarismesh.cn/bindPorts"
)

func NewXDSNodeManager() *XDSNodeManager {
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
			log.Info("[XDS][Node][V3] add gateway xds node", zap.Int64("stream", streamId),
				zap.String("info", p.String()))
		}
	default:
		if _, ok := x.sidecarNodes[node.Id]; !ok {
			x.sidecarNodes[node.Id] = p
			log.Info("[XDS][Node][V3] add sidecar xds node", zap.Int64("stream", streamId),
				zap.String("info", p.String()))
		}
	}
}

func (x *XDSNodeManager) DelNode(streamId int64) {
	x.lock.Lock()
	defer x.lock.Unlock()

	if p, ok := x.streamTonodes[streamId]; ok {
		delete(x.nodes, p.Node.Id)
		log.Info("[XDS][Node][V3] remove xds node", zap.Int64("stream", streamId),
			zap.String("info", p.String()))
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

func (x *XDSNodeManager) ListSidecarNodes() []*XDSClient {
	x.lock.RLock()
	defer x.lock.RUnlock()

	ret := make([]*XDSClient, 0, len(x.sidecarNodes))
	for i := range x.sidecarNodes {
		ret = append(ret, x.sidecarNodes[i])
	}
	return ret
}

// ID id 的格式是 ${sidecar|gateway}~namespace/uuid~hostIp
// case 1: envoy 为 sidecar 模式时，则 NodeID 的格式为以下两种
//
//		eg 1. namespace/uuid~hostIp
//		eg 2. sidecar~namespace/uuid-hostIp
//	 eg 3. envoy_node_id="${NAMESPACE}/${INSTANCE_IP}~${POD_NAME}"
//
// case 2: envoy 为 gateway 模式时，则 NodeID 的格式为： gateway~namespace/uuid~hostIp
func (PolarisNodeHash) ID(node *core.Node) string {
	if node == nil {
		return ""
	}

	runType, ns, _, _ := ParseNodeID(node.Id)
	if node.Metadata == nil || len(node.Metadata.Fields) == 0 {
		return ns
	}

	// Gateway 类型直接按照 gateway_service 以及 gateway_namespace 纬度
	if runType != string(RunTypeSidecar) {
		gatewayNamespace := node.Metadata.Fields[GatewayNamespaceName].GetStringValue()
		gatewayService := node.Metadata.Fields[GatewayServiceName].GetStringValue()
		// 兼容老的 envoy gateway metadata 参数设置
		if gatewayNamespace == "" {
			gatewayNamespace = node.Metadata.Fields[OldGatewayNamespaceName].GetStringValue()
		}
		if gatewayService == "" {
			gatewayService = node.Metadata.Fields[OldGatewayServiceName].GetStringValue()
		}
		if gatewayNamespace == "" {
			gatewayNamespace = ns
		}
		return strings.Join([]string{runType, gatewayNamespace, gatewayService}, "/")
	}
	// 兼容老版本注入的 envoy, 默认获取 snapshot resource 粒度为 namespace 级别, 只能下发 OUTBOUND 规则
	ret := ns

	// 判断是否存在 sidecar_namespace 以及 sidecar_service
	if node.Metadata != nil && node.Metadata.Fields != nil {
		sidecarNamespace := node.Metadata.Fields[SidecarNamespaceName].GetStringValue()
		sidecarService := node.Metadata.Fields[SidecarServiceName].GetStringValue()
		// 如果存在, 则表示是由新版本 controller 注入的 envoy, 可以下发 INBOUND 规则
		if sidecarNamespace != "" && sidecarService != "" {
			ret = runType + "/" + sidecarNamespace + "/" + sidecarService
		}

		// 在判断是否设置了 TLS 相关参数
		tlsMode := node.Metadata.Fields[TLSModeTag].GetStringValue()
		if tlsMode == string(TLSModePermissive) || tlsMode == string(TLSModeStrict) {
			return ret + "/" + tlsMode
		}
	}
	return ret
}

// PolarisNodeHash 存放 hash 方法
type PolarisNodeHash struct {
	NodeMgr *XDSNodeManager
}

// node id 的格式是:
// 1. namespace/uuid~hostIp
var nodeIDFormat = regexp.MustCompile(`^((\S+)~(\S+)|(\S+))\/([^~]+)~([^~]+)$`)

func ParseNodeID(nodeID string) (runType, polarisNamespace, uuid, hostIP string) {
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

// XDSClient 客户端代码结构体
type XDSClient struct {
	RunType   RunType
	User      string
	Namespace string
	IPAddr    string
	PodIP     string
	Metadata  map[string]string
	Version   string
	Node      *core.Node
	TLSMode   TLSMode
}

func (n *XDSClient) String() string {
	return fmt.Sprintf("nodeid=%s|type=%v|user=%s|addr=%s|version=%s|tls=%s|meta=%s", n.Node.GetId(),
		n.RunType, n.User, n.IPAddr, n.Version, n.TLSMode, toJSON(n.Metadata))
}

func (n *XDSClient) IsGateway() bool {
	service := n.Metadata[GatewayServiceName]
	namespace := n.Metadata[GatewayNamespaceName]
	oldSvc := n.Metadata[OldGatewayServiceName]
	oldSvcNamespace := n.Metadata[OldGatewayNamespaceName]
	hasNew := service != "" && namespace != ""
	hasOld := oldSvc != "" && oldSvcNamespace != ""
	return n.RunType == RunTypeGateway && (hasNew || hasOld)
}

// GetSelfService 获取 envoy 对应的 service 信息
func (n *XDSClient) GetSelfService() string {
	if n.IsGateway() {
		val, ok := n.Metadata[GatewayServiceName]
		if ok {
			return val
		}
		return n.Metadata[OldGatewayServiceName]
	}
	return n.Metadata[SidecarServiceName]
}

// GetSelfNamespace 获取 envoy 对应的 namespace 信息
func (n *XDSClient) GetSelfNamespace() string {
	if n.IsGateway() {
		val, ok := n.Metadata[GatewayNamespaceName]
		if ok {
			return val
		}
		val, ok = n.Metadata[OldGatewayNamespaceName]
		if ok {
			return val
		}
		return n.Namespace
	}
	val, ok := n.Metadata[SidecarNamespaceName]
	if ok {
		return val
	}
	return n.Namespace
}

func parseNodeProxy(node *core.Node) *XDSClient {
	runType, polarisNamespace, _, hostIP := ParseNodeID(node.Id)
	proxy := &XDSClient{
		IPAddr:    hostIP,
		PodIP:     hostIP,
		RunType:   RunType(runType),
		Namespace: polarisNamespace,
		Node:      node,
		TLSMode:   TLSModeNone,
	}

	if node.Metadata != nil {
		fieldVal, ok := node.Metadata.Fields[TLSModeTag]
		if ok {
			tlsMode := fieldVal.GetStringValue()
			if tlsMode == string(TLSModePermissive) {
				proxy.TLSMode = TLSModePermissive
			}
			if tlsMode == string(TLSModeStrict) {
				proxy.TLSMode = TLSModeStrict
			}
		}
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

func toJSON(m map[string]string) string {
	if m == nil {
		return "{}"
	}

	ba, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(ba)
}
