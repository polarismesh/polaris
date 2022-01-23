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

package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/golang/protobuf/proto"
	"github.com/polarismesh/polaris-server/common/api/l5"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
)

var (
	// Namespace2SidLayoutID namespace to sid layout id
	Namespace2SidLayoutID = map[string]uint32{
		"Production":  1,
		"Development": 2,
		"Pre-release": 3,
		"Test":        4,
		"Polaris":     5,
		"default":     6,
	}

	// SidLayoutID2Namespace sid layout id to namespace
	SidLayoutID2Namespace = map[uint32]string{
		1: "Production",
		2: "Development",
		3: "Pre-release",
		4: "Test",
		5: "Polaris",
		6: "default",
	}
)

// 记录l5service发现中的一些状态
type l5service struct {
	discoverRevision     string
	discoverClusterCount uint32
}

// SyncByAgentCmd 根据sid获取路由信息
// 老函数：
// Stat::instance()->inc_sync_req_cnt();
// 保存client的IP，该函数只是存储到本地的缓存中
// Stat::instance()->add_agent(sbac.agent_ip());
func (s *Server) SyncByAgentCmd(ctx context.Context, sbac *l5.Cl5SyncByAgentCmd) (
	*l5.Cl5SyncByAgentAckCmd, error) {
	clientIP := sbac.GetAgentIp()
	optList := sbac.GetOptList().GetOpt()

	routes := s.getRoutes(clientIP, optList)
	modIDList, callees, sidConfigs := s.getCallees(routes)
	policys, sections := s.getPolicysAndSections(modIDList)

	sbaac := &l5.Cl5SyncByAgentAckCmd{
		AgentIp:  proto.Int32(sbac.GetAgentIp()),
		SyncFlow: proto.Int32(sbac.GetSyncFlow() + 1),
	}
	ipConfigs := make(map[uint32]*model.Location) // 所有的被调IP+主调IP
	if len(callees) != 0 {
		serverList := &l5.Cl5ServList{
			Serv: make([]*l5.Cl5ServObj, 0, len(callees)),
		}
		for _, entry := range callees {
			server := &l5.Cl5ServObj{
				ModId:  proto.Int32(int32(entry.ModID)),
				CmdId:  proto.Int32(int32(entry.CmdID)),
				Ip:     proto.Int32(int32(entry.IP)),
				Port:   proto.Int32(int32(entry.Port)),
				Weight: proto.Int32(int32(entry.Weight)),
			}
			serverList.Serv = append(serverList.Serv, server)
			ipConfigs[entry.IP] = entry.Location // 填充ipConfigs信息
		}
		sbaac.ServList = serverList
	}

	if len(policys) != 0 {
		routeList := &l5.Cl5RuleList{
			Poly: make([]*l5.Cl5PolyObj, 0, len(policys)),
			Sect: make([]*l5.Cl5SectObj, 0, len(sections)),
		}
		for _, entry := range policys {
			obj := &l5.Cl5PolyObj{
				ModId: proto.Int32(int32(entry.ModID)),
				Div:   proto.Int32(int32(entry.Div)),
				Mod:   proto.Int32(int32(entry.Mod)),
			}
			routeList.Poly = append(routeList.Poly, obj)
		}
		for _, entry := range sections {
			obj := &l5.Cl5SectObj{
				ModId: proto.Int32(int32(entry.ModID)),
				From:  proto.Int32(int32(entry.From)),
				To:    proto.Int32(int32(entry.To)),
				CmdId: proto.Int32(int32(entry.Xid)),
			}
			routeList.Sect = append(routeList.Sect, obj)
		}
		sbaac.RuleList = routeList
	}

	// 保持和cl5源码一致，agent的地域信息，如果找不到，则不加入到ipConfigs中
	if loc := s.getLocation(ParseIPInt2Str(uint32(sbac.GetAgentIp()))); loc != nil {
		ipConfigs[uint32(sbac.GetAgentIp())] = loc
	}
	if len(ipConfigs) != 0 {
		ipConfigList := &l5.Cl5IpcList{
			Ipc: make([]*l5.Cl5IpcObj, 0, len(ipConfigs)),
		}
		for key, entry := range ipConfigs {
			obj := &l5.Cl5IpcObj{
				Ip:     proto.Int32(int32(key)),
				AreaId: proto.Int32(int32(entry.RegionID)),
				CityId: proto.Int32(int32(entry.ZoneID)),
				IdcId:  proto.Int32(int32(entry.CampusID)),
			}
			ipConfigList.Ipc = append(ipConfigList.Ipc, obj)
		}
		sbaac.IpcList = ipConfigList
	}

	sbaac.SidList = CreateCl5SidList(sidConfigs)
	sbaac.L5SvrList = s.getCl5DiscoverList(ctx, uint32(clientIP))
	return sbaac, nil
}

// get routes
func (s *Server) getRoutes(clientIP int32, optList []*l5.Cl5OptObj) []*model.Route {
	cl5Cache := s.caches.CL5()
	routes := cl5Cache.GetRouteByIP(uint32(clientIP))
	if routes == nil {
		routes = make([]*model.Route, 0)
	}
	for _, entry := range optList {
		modID := entry.GetModId()
		cmdID := entry.GetCmdId()
		if ok := cl5Cache.CheckRouteExisted(uint32(clientIP), uint32(modID), uint32(cmdID)); !ok {
			route := &model.Route{
				IP:    uint32(clientIP),
				ModID: uint32(entry.GetModId()),
				CmdID: uint32(entry.GetCmdId()),
				SetID: "NOSET",
			}
			routes = append(routes, route)
			// Stat::instance()->add_route(route.ip,route.modId,route.cmdId); TODO
		}
	}

	return routes
}

// get callee
func (s *Server) getCallees(routes []*model.Route) (map[uint32]bool, []*model.Callee, []*model.SidConfig) {
	modIDList := make(map[uint32]bool)
	var callees []*model.Callee
	var sidConfigs []*model.SidConfig
	for _, entry := range routes {
		servers := s.getCalleeByRoute(entry) // 返回nil代表没有找到任何实例
		if servers == nil {
			log.Warnf("[Cl5] can not found the instances for sid(%d:%d)", entry.ModID, entry.CmdID)
			// Stat::instance()->add_lost_route(sbac.agent_ip(),vt_route[i].modId,vt_route[i].cmdId); TODO
			continue
		}
		if len(servers) != 0 { // 不为nil，但是数组长度为0，意味着实例的权重不符合规则
			callees = append(callees, servers...)
		}

		modIDList[entry.ModID] = true
		if sidConfig := s.getSidConfig(entry.ModID, entry.CmdID); sidConfig != nil {
			sidConfigs = append(sidConfigs, sidConfig)
		}
	}

	return modIDList, callees, sidConfigs
}

// get policy and section
func (s *Server) getPolicysAndSections(modIDList map[uint32]bool) ([]*model.Policy, []*model.Section) {
	cl5Cache := s.caches.CL5()
	var policys []*model.Policy
	var sections []*model.Section
	for modID := range modIDList {
		if policy := cl5Cache.GetPolicy(modID); policy != nil {
			policys = append(policys, policy)
		}
		if secs := cl5Cache.GetSection(modID); len(secs) != 0 {
			sections = append(sections, secs...)
		}
	}

	return policys, sections
}

// RegisterByNameCmd 根据名字获取sid信息
func (s *Server) RegisterByNameCmd(rbnc *l5.Cl5RegisterByNameCmd) (*l5.Cl5RegisterByNameAckCmd, error) {
	// Stat::instance()->inc_register_req_cnt(); TODO

	nameList := rbnc.GetNameList()
	sidConfigs := make([]*model.SidConfig, 0)
	for _, name := range nameList.GetName() {
		if sidConfig := s.getSidConfigByName(name); sidConfig != nil {
			sidConfigs = append(sidConfigs, sidConfig)
		}
	}

	cl5RegisterAckCmd := &l5.Cl5RegisterByNameAckCmd{
		CallerIp: proto.Int32(rbnc.GetCallerIp()),
	}

	cl5RegisterAckCmd.SidList = CreateCl5SidList(sidConfigs)
	return cl5RegisterAckCmd, nil
}

func (s *Server) computeService(modID uint32, cmdID uint32) *model.Service {
	sidStr := utils.MarshalModCmd(modID, cmdID)
	// 根据sid找到所述命名空间
	namespaces := ComputeNamespace(modID, cmdID)
	for _, namespace := range namespaces {
		// 根据sid找到polaris服务，这里是源服务
		service := s.getServiceCache(sidStr, namespace)
		if service != nil {
			return service
		}
	}
	return nil
}

// 根据访问关系获取所有符合的被调信息
func (s *Server) getCalleeByRoute(route *model.Route) []*model.Callee {
	out := make([]*model.Callee, 0)
	if route == nil {
		return nil
	}
	service := s.computeService(route.ModID, route.CmdID)
	if service == nil {
		return nil
	}
	s.RecordDiscoverStatis(service.Name, service.Namespace)

	hasInstance := false
	_ = s.caches.Instance().IteratorInstancesWithService(service.ID,
		func(key string, entry *model.Instance) (b bool, e error) {
			hasInstance = true
			// 如果不存在internal-cl5-setId，则默认都是NOSET，适用于别名场景
			setValue := "NOSET"
			metadata := entry.Metadata()
			if val, ok := metadata["internal-cl5-setId"]; ok {
				setValue = val
			}

			// 与route的setID匹配，那么直接返回instance.weight
			weight := entry.Weight()
			found := false
			if setValue == route.SetID {
				found = true
			} else if !strings.Contains(setValue, route.SetID) {
				found = false
			} else {
				var weights []uint32
				if val, ok := metadata["internal-cl5-weight"]; ok {
					weights = ParseWeight(val)
				}
				setIDs := ParseSetID(setValue)
				for i, setID := range setIDs {
					if setID == route.SetID {
						found = true
						if weights != nil && i < len(weights) {
							weight = weights[i]
						}
						break
					}
				}
			}

			// 该Set无被调或者被调的权重为0，则忽略
			if !found || weight == 0 {
				return true, nil
			}

			// 转换ipStr to int
			ip := ParseIPStr2IntV2(entry.Host())

			callee := &model.Callee{
				ModID:  route.ModID,
				CmdID:  route.CmdID,
				IP:     ip,
				Port:   entry.Port(),
				Weight: weight,
				// TODO 没有设置 setID，cl5源码也是没有设置的
			}
			// s.getLocation(entry.Host), // ip的地域信息，统一来源于cmdb插件的数据
			if loc := s.getLocation(entry.Host()); loc != nil {
				callee.Location = loc
			} else {
				// 如果cmdb中找不到数据，则默认地域ID都为0，即默认结构体
				callee.Location = &model.Location{}
			}
			out = append(out, callee)
			return true, nil
		})

	if !hasInstance {
		return nil
	}

	return out
}

// 根据sid读取sidConfig的配置信息
// 注意，sid--> reference，通过索引服务才能拿到真实的数据
func (s *Server) getSidConfig(modID uint32, cmdID uint32) *model.SidConfig {
	sid := &model.Sid{ModID: modID, CmdID: cmdID}
	sidStr := utils.MarshalSid(sid)

	// 先获取一下namespace
	namespaces := ComputeNamespace(modID, cmdID)

	var sidService *model.Service
	for _, namespace := range namespaces {
		sidService = s.caches.Service().GetServiceByName(sidStr, namespace)
		if sidService != nil {
			break
		}
	}
	if sidService == nil {
		return nil
	}
	sidConfig := s.getRealSidConfigMeta(sidService)
	if sidConfig == nil {
		return nil
	}

	sidConfig.ModID = modID
	sidConfig.CmdID = cmdID

	return sidConfig
}

// 根据名字找到sidConfig
// 注意：通过cache，根据cl5Name，找到对应的sid
func (s *Server) getSidConfigByName(name string) *model.SidConfig {
	nameService := s.caches.Service().GetServiceByCl5Name(name)
	if nameService == nil {
		return nil
	}

	sidConfig := s.getRealSidConfigMeta(nameService)
	if sidConfig == nil {
		return nil
	}

	sidMeta, ok := nameService.Meta["internal-cl5-sid"]
	if !ok {
		log.Errorf("[Server] not found name(%s) sid", name)
		return nil
	}

	sid, err := utils.UnmarshalSid(sidMeta)
	if err != nil {
		log.Errorf("[Server] unmarshal sid(%s) err: %s", sidMeta, err.Error())
		return nil
	}

	sidConfig.ModID = sid.ModID
	sidConfig.CmdID = sid.CmdID
	return sidConfig
}

// 只返回服务名+policy属性
func (s *Server) getRealSidConfigMeta(service *model.Service) *model.SidConfig {
	if service == nil {
		return nil
	}

	realService := service
	// 找一下，是否存在索引服务（别名服务）
	// 如果存在索引服务，读取索引服务的属性
	if service.IsAlias() {
		if referService := s.caches.Service().GetServiceByID(service.Reference); referService != nil {
			realService = referService
		}
	}

	out := &model.SidConfig{
		Name:   "",
		Policy: 0,
	}
	if nameMeta, ok := realService.Meta["internal-cl5-name"]; ok {
		out.Name = nameMeta
	}
	if policyMeta, ok := realService.Meta["internal-enable-nearby"]; ok {
		if policyMeta == "true" {
			out.Policy = 1
		}
	}

	return out
}

// 获取cl5.discover
func (s *Server) getCl5DiscoverList(ctx context.Context, clientIP uint32) *l5.Cl5L5SvrList {
	clusterName, _ := ctx.Value(utils.Cl5ServerCluster{}).(string)
	if clusterName == "" {
		log.Warnf("[Cl5] get server cluster name is empty")
		return nil
	}
	protocol, _ := ctx.Value(utils.Cl5ServerProtocol{}).(string)

	service := s.getCl5DiscoverService(clusterName, clientIP)
	if service == nil {
		log.Errorf("[Cl5] not found server cluster service(%s)", clusterName)
		return nil
	}
	instances := s.caches.Instance().GetInstancesByServiceID(service.ID)
	if len(instances) == 0 {
		log.Errorf("[Cl5] not found any instances for the service(%s, %s)",
			clusterName, "Polaris")
		return nil
	}

	var out l5.Cl5L5SvrList
	out.Ip = make([]int32, 0, len(instances))
	for _, entry := range instances {
		// 获取同协议的数据
		if entry.Protocol() != protocol {
			continue
		}
		// 过滤掉不健康或者隔离状态的server
		if !entry.Healthy() || entry.Isolate() {
			continue
		}
		ip := ParseIPStr2IntV2(entry.Host())
		out.Ip = append(out.Ip, int32(ip))
	}
	// 如果没有任何数据，那直接返回空，使用agent配置的IPlist
	if len(out.GetIp()) == 0 {
		log.Errorf("[Cl5] get cl5 cluster(%s) instances count 0", service.Name)
		return nil
	}

	return &out
}

// 根据集群名获取对应的服务
func (s *Server) getCl5DiscoverService(clusterName string, clientIP uint32) *model.Service {
	service := s.getServiceCache(clusterName, "Polaris")
	if service == nil {
		log.Errorf("[Cl5] not found server cluster service(%s)", clusterName)
		return nil
	}

	// 根据service的metadata判断，有多少个子集群
	clusterCount := uint32(0)
	if service.Revision == s.l5service.discoverRevision {
		clusterCount = atomic.LoadUint32(&s.l5service.discoverClusterCount)
	} else {
		if meta, ok := service.Meta["internal-cluster-count"]; ok {
			count, err := strconv.Atoi(meta)
			if err != nil {
				log.Errorf("[Cl5] get service count , parse err: %s", err.Error())
			} else {
				clusterCount = uint32(count)
				s.l5service.discoverRevision = service.Revision
				atomic.StoreUint32(&s.l5service.discoverClusterCount, clusterCount)
			}
		}
	}

	// 如果集群数为0，那么返回埋点的集群
	if clusterCount == 0 {
		return service
	}

	subIndex := clientIP%uint32(clusterCount) + 1
	subClusterName := fmt.Sprintf("%s.%d", clusterName, subIndex)
	// log.Infof("[Cl5] ip(%d), clusterCount(%d), name(%s)", clientIP, clusterCount, subClusterName) // TODO
	subService := s.getServiceCache(subClusterName, "Polaris")
	if subService == nil {
		log.Errorf("[Cl5] not found server cluster for ip(%d), cluster count(%d), cluster name(%s)",
			clientIP, clusterCount, subClusterName)
		return service
	}

	return subService
}

// CreateCl5SidList 构造sidConfigs
func CreateCl5SidList(sidConfigs []*model.SidConfig) *l5.Cl5SidList {
	if len(sidConfigs) == 0 {
		return nil
	}

	sidList := &l5.Cl5SidList{
		Sid: make([]*l5.Cl5SidObj, 0, len(sidConfigs)),
	}
	for _, entry := range sidConfigs {
		obj := &l5.Cl5SidObj{
			ModId:  proto.Int32(int32(entry.ModID)),
			CmdId:  proto.Int32(int32(entry.CmdID)),
			Name:   proto.String(entry.Name),
			Policy: proto.Int32(int32(entry.Policy)),
		}
		sidList.Sid = append(sidList.Sid, obj)
	}

	return sidList
}

// ParseSetID 解析metadata保存的setID字符串
func ParseSetID(str string) []string {
	if str == "" {
		return nil
	}

	return strings.Split(str, ",")
}

// 解析metadata保存的weight字符串
func ParseWeight(str string) []uint32 {
	if str == "" {
		return nil
	}

	items := strings.Split(str, ",")
	if len(items) == 0 {
		return nil
	}
	out := make([]uint32, 0, len(items))
	for _, item := range items {
		data, err := strconv.ParseUint(item, 10, 32)
		if err != nil {
			log.Errorf("[L5Service] parse uint (%s) err: %s", item, err.Error())
			return nil
		}

		out = append(out, uint32(data))
	}

	return out
}

// ParseIPStr2Int 字符串IP转为uint32
// 转换失败的，需要明确错误
func ParseIPStr2Int(ip string) (uint32, error) {
	ips := strings.Split(ip, ".")
	if len(ips) != 4 {
		log.Errorf("[l5Service] ip str(%s) is invalid", ip)
		return 0, errors.New("ip string is invalid")
	}

	out := uint32(0)
	for i := 0; i < 4; i++ {
		tmp, err := strconv.ParseUint(ips[i], 10, 64)
		if err != nil {
			log.Errorf("[L5Service] ip str(%s) to int is err: %s", ip, err.Error())
			return 0, err
		}

		out = out | (uint32(tmp) << uint(i*8))
	}

	return out, nil
}

// ParseIPStr2IntV2 字符串IP转为Int，V2
func ParseIPStr2IntV2(ip string) uint32 {
	item := 0
	var sum uint32
	var index uint
	for i := 0; i < len(ip); i++ {
		if ip[i] == '.' {
			sum = sum | (uint32(item) << (index * 8))
			item = 0
			index++
		} else {
			item = item*10 + int(ip[i]) - int('0')
		}
	}

	sum = sum | (uint32(item) << (index * 8))
	return sum
}

// ParseIPInt2Str uint32的IP转换为字符串型
func ParseIPInt2Str(ip uint32) string {
	ipStr := make([]uint32, 4)
	for i := 0; i < 4; i++ {
		ipStr[i] = (ip >> uint(i*8)) & 255
	}
	str := fmt.Sprintf("%d.%d.%d.%d", ipStr[0], ipStr[1], ipStr[2], ipStr[3])
	return str
}

// ComputeNamespace 根据SID分析，返回其对应的namespace
func ComputeNamespace(modID uint32, cmdID uint32) []string {
	// 为了兼容老的sid，只对新的别名sid才生效
	// 老的sid都属于生产环境的
	// 3000001是新的moduleID的开始值
	if moduleID := modID >> 6; moduleID < 3000001 {
		return []string{DefaultNamespace, ProductionNamespace}
	}

	layoutID := modID & 63 // 63 -> 111111
	namespace, ok := SidLayoutID2Namespace[layoutID]
	if !ok {
		// 找不到命名空间的，全部返回默认的，也就是Production
		log.Warnf("sid(%d:%d) found the layoutID is(%d), not match the namespace list",
			modID, cmdID, layoutID)
		return []string{DefaultNamespace}
	}

	log.Infof("Sid(%d:%d) layoutID(%d), the namespace is: %s",
		modID, cmdID, layoutID, namespace)
	return []string{namespace}
}
