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

package eurekaserver

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/service"
)

// ApplicationsRespCache 全量服务缓存
type ApplicationsRespCache struct {
	AppsResp      *ApplicationsResponse
	Revision      string
	JsonBytes     []byte
	XmlBytes      []byte
	createTimeSec int64
}

// ApplicationsBuilder builder to do application construct
type ApplicationsBuilder struct {
	namingServer           service.DiscoverServer
	namespace              string
	enableSelfPreservation bool
	// autoincrement version
	VersionIncrement int64
}

func getCacheServices(namingServer service.DiscoverServer, namespace string) map[string]*model.Service {
	var newServices = make(map[string]*model.Service)
	_ = namingServer.Cache().Service().IteratorServices(func(key string, value *model.Service) (bool, error) {
		if value.Namespace == namespace {
			newServices[value.Name] = value
		}
		return true, nil
	})
	return newServices
}

func getCacheInstances(namingServer service.DiscoverServer, svcId string) ([]*model.Instance, string, error) {
	var instances []*model.Instance
	_ = namingServer.Cache().Instance().IteratorInstancesWithService(svcId,
		func(key string, value *model.Instance) (bool, error) {
			instances = append(instances, value)
			return true, nil
		})
	revision, err := namingServer.GetServiceInstanceRevision(svcId, instances)
	return instances, revision, err
}

// BuildApplications build applications cache with compare to the latest cache
func (a *ApplicationsBuilder) BuildApplications(oldAppsCache *ApplicationsRespCache) *ApplicationsRespCache {
	// 获取所有的服务数据
	var newServices = getCacheServices(a.namingServer, a.namespace)
	var instCount int
	svcToRevision := make(map[string]string, len(newServices))
	svcToToInstances := make(map[string][]*model.Instance)
	var changed bool
	for _, newService := range newServices {
		instances, revision, err := getCacheInstances(a.namingServer, newService.ID)
		if err != nil {
			log.Errorf("[EurekaServer]fail to get revision for service %s, err is %v", newService.Name, err)
			continue
		}
		// eureka does not return services without instances
		if len(instances) == 0 {
			continue
		}
		instCount += len(instances)
		svcToRevision[newService.Name] = revision
		svcToToInstances[newService.Name] = instances
	}
	// 比较并构建Applications缓存
	hashBuilder := make(map[string]int)
	newApps := newApplications()
	var oldApps *Applications
	if oldAppsCache != nil {
		oldApps = oldAppsCache.AppsResp.Applications
	}
	for svc, instances := range svcToToInstances {
		var newRevision = svcToRevision[svc]
		var targetApp *Application
		if oldApps != nil {
			oldApp, ok := oldApps.ApplicationMap[svc]
			if ok && len(oldApp.Revision) > 0 && oldApp.Revision == newRevision {
				// 没有变化
				targetApp = oldApp
			}
		}
		if targetApp == nil {
			// 重新构建
			targetApp = &Application{
				Name:        svc,
				InstanceMap: make(map[string]*InstanceInfo),
				Revision:    newRevision,
			}
			a.constructApplication(targetApp, instances)
			changed = true
		}
		statusCount := targetApp.StatusCounts
		if len(statusCount) > 0 {
			for status, count := range statusCount {
				hashBuilder[status] = hashBuilder[status] + count
			}
		}
		newApps.Application = append(newApps.Application, targetApp)
		newApps.ApplicationMap[targetApp.Name] = targetApp
	}
	if oldApps != nil && len(oldApps.Application) != len(newApps.Application) {
		changed = true
	}
	a.buildVersionAndHashCode(changed, hashBuilder, newApps)
	return constructResponseCache(newApps, instCount, false)
}

func filterLatestHealthyInstances(instances []*model.Instance) ([]*model.Instance, int) {
	var out = make([]*model.Instance, 0, len(instances))
	var healthyCount = 0
	for _, instance := range instances {
		if instance.Healthy() {
			out = append(out, instance)
			healthyCount++
			continue
		}
		healthCheck := instance.HealthCheck()
		if healthCheck == nil {
			continue
		}
		modifySince := time.Since(instance.ModifyTime)
		if modifySince < DefaultSelfPreservationDuration {
			out = append(out, instance)
		}
	}
	return out, healthyCount
}

func (a *ApplicationsBuilder) constructApplication(app *Application, instances []*model.Instance) {
	if len(instances) == 0 {
		return
	}
	app.StatusCounts = make(map[string]int)

	var fallbackUnhealthy bool
	var healthyCount int
	instances, healthyCount = filterLatestHealthyInstances(instances)
	if a.enableSelfPreservation && len(instances) > 0 {
		if (healthyCount/len(instances))*100 < DefaultSelfPreservationPercent {
			fallbackUnhealthy = true
		}
	}

	// 转换时候要区分2种情况，一种是从eureka注册上来的，一种不是
	for _, instance := range instances {
		if !instance.Healthy() && !fallbackUnhealthy {
			continue
		}
		var (
			instanceInfo     *InstanceInfo
			eurekaInstanceId = instance.Proto.GetId().GetValue()
		)
		instanceInfo = buildInstance(app, eurekaInstanceId, instance)
		instanceInfo.RealInstances[instance.Revision()] = instance
		status := instanceInfo.Status
		app.StatusCounts[status] = app.StatusCounts[status] + 1
		app.Instance = append(app.Instance, instanceInfo)
		app.InstanceMap[instanceInfo.InstanceId] = instanceInfo
	}
	if nil == app.Instance {
		app.Instance = []*InstanceInfo{}
	}
}

func (a *ApplicationsBuilder) buildVersionAndHashCode(changed bool, hashBuilder map[string]int, newApps *Applications) {
	var nextVersion int64
	if changed {
		nextVersion = atomic.AddInt64(&a.VersionIncrement, 1)
	} else {
		nextVersion = atomic.LoadInt64(&a.VersionIncrement)
	}
	buildHashCode(strconv.Itoa(int(nextVersion)), hashBuilder, newApps)
}

func buildHashCode(version string, hashBuilder map[string]int, newApps *Applications) {
	// 构建hashValue
	newApps.AppsHashCode = buildHashStr(hashBuilder)
	newApps.VersionsDelta = version
}

func (a *ApplicationsBuilder) buildDeltaApps(oldAppsCache *ApplicationsRespCache, newAppsCache *ApplicationsRespCache,
	latestDeltaAppsCache *ApplicationsRespCache) *ApplicationsRespCache {
	var instCount int
	newApps := newAppsCache.AppsResp.Applications
	// 1. 创建新的delta对象
	newDeltaApps := &Applications{
		VersionsDelta: newApps.VersionsDelta,
		AppsHashCode:  newApps.AppsHashCode,
		Application:   make([]*Application, 0),
	}
	// 2. 拷贝老的delta内容
	var oldDeltaApps *Applications
	if latestDeltaAppsCache != nil {
		oldDeltaApps = latestDeltaAppsCache.AppsResp.Applications
	}
	if oldDeltaApps != nil && len(oldDeltaApps.Application) > 0 {
		for _, app := range oldDeltaApps.Application {
			newDeltaApps.Application = append(newDeltaApps.Application, app)
			instCount += len(app.Instance)
		}
	}
	// 3. 比较revision是否发生变更
	if oldAppsCache.Revision != newAppsCache.Revision {
		// 3. 比较修改和新增
		oldApps := oldAppsCache.AppsResp.Applications
		applications := newApps.Application
		if len(applications) > 0 {
			for _, application := range applications {
				var oldApplication = oldApps.GetApplication(application.Name)
				if oldApplication == nil {
					// 新增，全部加入
					newDeltaApps.Application = append(newDeltaApps.Application, application)
					instCount += len(application.Instance)
					continue
				}
				// 修改，需要比较实例的变更
				diffApp := diffApplication(oldApplication, application)
				if diffApp != nil {
					newDeltaApps.Application = append(newDeltaApps.Application, diffApp)
					instCount += len(diffApp.Instance)
				}
			}
		}
		// 4. 比较删除
		oldApplications := oldApps.Application
		if len(oldApplications) > 0 {
			for _, application := range oldApplications {
				var newApplication = newApps.GetApplication(application.Name)
				if newApplication == nil {
					// 删除
					deletedApplication := &Application{
						Name: application.Name,
					}
					for _, instance := range application.Instance {
						deletedApplication.Instance = append(deletedApplication.Instance, instance.Clone(ActionDeleted))
					}
					newDeltaApps.Application = append(newDeltaApps.Application, deletedApplication)
					instCount += len(deletedApplication.Instance)
				}
			}
		}
	}
	return constructResponseCache(newDeltaApps, instCount, true)
}

func parseStatus(instance *model.Instance) string {
	if instance.Proto.GetIsolate().GetValue() {
		return StatusOutOfService
	}
	return StatusUp
}

func parsePortWrapper(info *InstanceInfo, instance *model.Instance) {
	securePort, securePortOk := instance.Metadata()[MetadataSecurePort]
	securePortEnabled, securePortEnabledOk := instance.Metadata()[MetadataSecurePortEnabled]
	insecurePort, insecurePortOk := instance.Metadata()[MetadataInsecurePort]
	insecurePortEnabled, insecurePortEnabledOk := instance.Metadata()[MetadataInsecurePortEnabled]

	if securePortOk && securePortEnabledOk && insecurePortOk && insecurePortEnabledOk {
		// if metadata contains all port/securePort,port.enabled/securePort.enabled
		sePort, err := strconv.Atoi(securePort)
		if err != nil {
			sePort = 0
			log.Errorf("[EUREKA_SERVER]parse secure port error: %+v", err)
		}
		sePortEnabled, err := strconv.ParseBool(securePortEnabled)
		if err != nil {
			sePortEnabled = false
			log.Errorf("[EUREKA_SERVER]parse secure port enabled error: %+v", err)
		}

		info.SecurePort.Port = sePort
		info.SecurePort.Enabled = sePortEnabled

		insePort, err := strconv.Atoi(insecurePort)
		if err != nil {
			insePort = 0
			log.Errorf("[EUREKA_SERVER]parse insecure port error: %+v", err)
		}
		insePortEnabled, err := strconv.ParseBool(insecurePortEnabled)
		if err != nil {
			insePortEnabled = false
			log.Errorf("[EUREKA_SERVER]parse insecure port enabled error: %+v", err)
		}

		info.Port.Port = insePort
		info.Port.Enabled = insePortEnabled

	} else {
		protocol := instance.Proto.GetProtocol().GetValue()
		port := instance.Proto.GetPort().GetValue()
		if protocol == SecureProtocol {
			info.SecurePort.Port = int(port)
			info.SecurePort.Enabled = "true"
			if len(instance.Metadata()) > 0 {
				insecurePortStr, ok := instance.Metadata()[MetadataInsecurePort]
				if ok {
					insecurePort, _ := strconv.Atoi(insecurePortStr)
					if insecurePort > 0 {
						info.Port.Port = insecurePort
						info.Port.Enabled = "true"
					}
				}
			}
		} else {
			info.Port.Port = int(port)
			info.Port.Enabled = "true"
		}
	}
}

func parseLeaseInfo(leaseInfo *LeaseInfo, instance *model.Instance) {
	metadata := instance.Proto.GetMetadata()
	var durationInSec int
	var renewIntervalSec int
	if metadata != nil {
		durationInSecStr, ok := metadata[MetadataDuration]
		if ok {
			durationInSec, _ = strconv.Atoi(durationInSecStr)
		}
		renewIntervalStr, ok := metadata[MetadataRenewalInterval]
		if ok {
			renewIntervalSec, _ = strconv.Atoi(renewIntervalStr)
		}
	}
	if durationInSec > 0 {
		leaseInfo.DurationInSecs = durationInSec
	}
	if renewIntervalSec > 0 {
		leaseInfo.RenewalIntervalInSecs = renewIntervalSec
	}
}

func buildInstance(app *Application, eurekaInstanceId string, instance *model.Instance) *InstanceInfo {
	instanceInfo := &InstanceInfo{
		CountryId: DefaultCountryIdInt,
		Port: &PortWrapper{
			Enabled: "false",
			Port:    DefaultInsecurePort,
		},
		SecurePort: &PortWrapper{
			Enabled: "false",
			Port:    DefaultSSLPort,
		},
		LeaseInfo: &LeaseInfo{
			RenewalIntervalInSecs: DefaultRenewInterval,
			DurationInSecs:        DefaultDuration,
		},
		Metadata: &Metadata{
			Meta: make(map[string]interface{}),
		},
		RealInstances: make(map[string]*model.Instance),
	}
	instanceInfo.AppName = app.Name
	// 属于eureka注册的实例
	instanceInfo.InstanceId = eurekaInstanceId
	metadata := instance.Metadata()
	if metadata == nil {
		metadata = map[string]string{}
	}
	if hostName, ok := metadata[MetadataHostName]; ok {
		instanceInfo.HostName = hostName
	}
	instanceInfo.IpAddr = instance.Proto.GetHost().GetValue()
	instanceInfo.Status = parseStatus(instance)
	instanceInfo.OverriddenStatus = StatusUnknown
	parsePortWrapper(instanceInfo, instance)
	if countryIdStr, ok := metadata[MetadataCountryId]; ok {
		cId, err := strconv.Atoi(countryIdStr)
		if err == nil {
			instanceInfo.CountryId = cId
		}
	}
	dciClazz, ok1 := metadata[MetadataDataCenterInfoClazz]
	dciName, ok2 := metadata[MetadataDataCenterInfoName]
	if ok1 && ok2 {
		instanceInfo.DataCenterInfo = &DataCenterInfo{
			Clazz: dciClazz,
			Name:  dciName,
		}
	} else {
		instanceInfo.DataCenterInfo = DefaultDataCenterInfo
	}
	parseLeaseInfo(instanceInfo.LeaseInfo, instance)
	for metaKey, metaValue := range metadata {
		if strings.HasPrefix(metaKey, "internal-") {
			continue
		}
		instanceInfo.Metadata.Meta[metaKey] = metaValue
	}
	if url, ok := metadata[MetadataHomePageUrl]; ok {
		instanceInfo.HomePageUrl = url
	}
	if url, ok := metadata[MetadataStatusPageUrl]; ok {
		instanceInfo.StatusPageUrl = url
	}
	if url, ok := metadata[MetadataHealthCheckUrl]; ok {
		instanceInfo.HealthCheckUrl = url
	}
	if address, ok := metadata[MetadataVipAddress]; ok {
		instanceInfo.VipAddress = address
	}
	if address, ok := metadata[MetadataSecureVipAddress]; ok {
		instanceInfo.SecureVipAddress = address
	}
	if instanceInfo.VipAddress == "" {
		instanceInfo.VipAddress = app.Name
	}
	if instanceInfo.HostName == "" {
		instanceInfo.HostName = instance.Proto.GetHost().GetValue()
	}
	buildLocationInfo(instanceInfo, instance)
	instanceInfo.LastUpdatedTimestamp = strconv.Itoa(int(instance.ModifyTime.UnixNano() / 1e6))
	instanceInfo.ActionType = ActionAdded
	return instanceInfo
}

func buildLocationInfo(instanceInfo *InstanceInfo, instance *model.Instance) {
	var region string
	var zone string
	var campus string
	if location := instance.Location(); location != nil {
		region = location.GetRegion().GetValue()
		zone = location.GetZone().GetValue()
		campus = location.GetCampus().GetValue()
	}
	if _, ok := instanceInfo.Metadata.Meta[KeyRegion]; !ok && len(region) > 0 {
		instanceInfo.Metadata.Meta[KeyRegion] = region
	}
	if _, ok := instanceInfo.Metadata.Meta[keyZone]; !ok && len(zone) > 0 {
		instanceInfo.Metadata.Meta[keyZone] = zone
	}
	if _, ok := instanceInfo.Metadata.Meta[keyCampus]; !ok && len(campus) > 0 {
		instanceInfo.Metadata.Meta[keyCampus] = campus
	}
}

func newApplications() *Applications {
	return &Applications{
		ApplicationMap: make(map[string]*Application),
		Application:    make([]*Application, 0),
	}
}

func constructResponseCache(newApps *Applications, instCount int, delta bool) *ApplicationsRespCache {
	appsHashCode := newApps.AppsHashCode
	newAppsCache := &ApplicationsRespCache{
		AppsResp:      &ApplicationsResponse{Applications: newApps},
		createTimeSec: time.Now().Unix(),
	}
	// 预先做一次序列化，以免高并发时候序列化会使得内存峰值过高
	jsonBytes, err := json.MarshalIndent(newAppsCache.AppsResp, "", " ")
	if err != nil {
		log.Errorf("[EUREKA_SERVER]fail to marshal apps %s to json, err is %v", appsHashCode, err)
	} else {
		newAppsCache.JsonBytes = jsonBytes
	}
	xmlBytes, err := xml.MarshalIndent(newAppsCache.AppsResp.Applications, " ", " ")
	if err != nil {
		log.Errorf("[EUREKA_SERVER]fail to marshal apps %s to xml, err is %v", appsHashCode, err)
	} else {
		newAppsCache.XmlBytes = xmlBytes
	}
	if !delta && len(jsonBytes) > 0 {
		newAppsCache.Revision = sha1s(jsonBytes)
	}
	log.Infof("[EUREKA_SERVER]success to build apps cache, delta is %v, "+
		"length xmlBytes is %d, length jsonBytes is %d, instCount is %d", delta, len(xmlBytes), len(jsonBytes), instCount)
	return newAppsCache
}

func buildHashStr(counts map[string]int) string {
	if len(counts) == 0 {
		return ""
	}
	slice := make([]string, 0, len(counts))
	for k := range counts {
		slice = append(slice, k)
	}
	sort.Strings(slice)
	builder := &strings.Builder{}
	for _, status := range slice {
		builder.WriteString(fmt.Sprintf("%s_%d_", status, counts[status]))
	}
	return builder.String()
}
