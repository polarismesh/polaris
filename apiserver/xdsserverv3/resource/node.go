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
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	_struct "github.com/golang/protobuf/ptypes/struct"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/common/model"
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
	// SidecarRegisterService xds metadata key when node what register service from envoy healthcheck
	// value example: [{"name":"","ports":{"TCP":[8080],"DUBBO":[28080]},"health_check_path":"","health_check_port":8080,"health_check_ttl":5}]
	SidecarRegisterService = "sidecar.polarismesh.cn/registerServices"
	// SidecarTLSModeTag .
	SidecarTLSModeTag = "sidecar.polarismesh.cn/tlsMode"
	// SidecarOpenOnDemandFeature .
	SidecarOpenOnDemandFeature = "sidecar.polarismesh.cn/openOnDemand"
	// SidecarOpenOnDemandServer .
	SidecarOpenOnDemandServer = "sidecar.polarismesh.cn/demandServer"
)

type EnvoyNodeView struct {
	ID           string
	RunType      RunType
	User         string
	Namespace    string
	IPAddr       string
	PodIP        string
	Metadata     map[string]string
	Version      string
	TLSMode      TLSMode
	OpenOnDemand bool
}

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

func (x *XDSNodeManager) HasEnvoyNodes() bool {
	x.lock.RLock()
	defer x.lock.RUnlock()

	return len(x.gatewayNodes) != 0 || len(x.sidecarNodes) != 0
}

func (x *XDSNodeManager) ListEnvoyNodes() []*XDSClient {
	x.lock.RLock()
	defer x.lock.RUnlock()

	ret := make([]*XDSClient, 0, len(x.sidecarNodes))
	for i := range x.sidecarNodes {
		ret = append(ret, x.sidecarNodes[i])
	}
	for i := range x.gatewayNodes {
		ret = append(ret, x.gatewayNodes[i])
	}
	return ret
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

func (x *XDSNodeManager) ListEnvoyNodesView(run RunType) []*EnvoyNodeView {
	x.lock.RLock()
	defer x.lock.RUnlock()

	if run == RunTypeSidecar {
		ret := make([]*EnvoyNodeView, 0, len(x.sidecarNodes))
		for i := range x.sidecarNodes {
			ret = append(ret, x.sidecarNodes[i].toView())
		}
		return ret
	}
	ret := make([]*EnvoyNodeView, 0, len(x.gatewayNodes))
	for i := range x.gatewayNodes {
		ret = append(ret, x.gatewayNodes[i].toView())
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

type RegisterService struct {
	Name            string           `json:"name"`
	Ports           map[string][]int `json:"ports"`
	HealthCheckPath string           `json:"health_check_path"`
	HealthCheckPort int              `json:"health_check_port"`
	HealthCheckTtl  int              `json:"health_check_ttl"`
	TracingSampling float64          `json:"tracing_sampling"`
}

// XDSClient 客户端代码结构体
type XDSClient struct {
	ID           string
	RunType      RunType
	User         string
	Namespace    string
	IPAddr       string
	PodIP        string
	Metadata     map[string]string
	Version      string
	Node         *core.Node
	TLSMode      TLSMode
	OpenOnDemand bool
	DemandServer string
}

func (n *XDSClient) toView() *EnvoyNodeView {
	return &EnvoyNodeView{
		ID:           n.ID,
		RunType:      n.RunType,
		User:         n.User,
		Namespace:    n.Namespace,
		IPAddr:       n.IPAddr,
		PodIP:        n.PodIP,
		Metadata:     n.Metadata,
		Version:      n.Version,
		TLSMode:      n.TLSMode,
		OpenOnDemand: n.OpenOnDemand,
	}
}

func (n *XDSClient) GetNodeID() string {
	return n.ID
}

func (n *XDSClient) ResourceKey() string {
	return n.Namespace + "/" + string(n.TLSMode)
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

func (n *XDSClient) GetRegisterServices() []*RegisterService {
	if n.IsGateway() {
		return []*RegisterService{}
	}
	val, ok := n.Metadata[SidecarRegisterService]
	if !ok {
		return []*RegisterService{}
	}
	ret := make([]*RegisterService, 0, 4)
	_ = json.Unmarshal([]byte(val), &ret)
	return ret
}

// GetSelfServiceKey 获取 envoy 对应的 service 信息
func (n *XDSClient) GetSelfServiceKey() model.ServiceKey {
	return model.ServiceKey{
		Namespace: n.GetSelfNamespace(),
		Name:      n.GetSelfService(),
	}
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

// ParseXDSClient .
func ParseXDSClient(node *core.Node) *XDSClient {
	return parseNodeProxy(node)
}

func parseNodeProxy(node *core.Node) *XDSClient {
	runType, polarisNamespace, _, hostIP := ParseNodeID(node.Id)
	proxy := &XDSClient{
		ID:        node.Id,
		IPAddr:    hostIP,
		PodIP:     hostIP,
		RunType:   RunType(runType),
		Namespace: polarisNamespace,
		Node:      node,
		TLSMode:   TLSModeNone,
	}

	if node.Metadata != nil {
		if tlsMode, ok := getEnvoyMetaField(node.Metadata, TLSModeTag, ""); ok {
			if tlsMode == string(TLSModePermissive) {
				proxy.TLSMode = TLSModePermissive
			}
			if tlsMode == string(TLSModeStrict) {
				proxy.TLSMode = TLSModeStrict
			}
		}
		if onDemand, ok := getEnvoyMetaField(node.Metadata, SidecarOpenOnDemandFeature, ""); ok {
			proxy.OpenOnDemand = onDemand == "true"
		}
	}

	proxy.Metadata = parseMetadata(node.GetMetadata())
	return proxy
}

func GetEnvoyMetaField[T any](meta *_struct.Struct, fileName string, fType T) (T, bool) {
	return getEnvoyMetaField[T](meta, fileName, fType)
}

func getEnvoyMetaField[T any](meta *_struct.Struct, fileName string, fType T) (T, bool) {
	fieldVal, ok := meta.Fields[fileName]
	if !ok {
		return fType, false
	}
	v := reflect.ValueOf(fType)
	var ret interface{}
	switch v.Type().Kind() {
	case reflect.Bool:
		ret = fieldVal.GetBoolValue()
		return ret.(T), true
	default:
		ret = fieldVal.GetStringValue()
		return ret.(T), true
	}
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
