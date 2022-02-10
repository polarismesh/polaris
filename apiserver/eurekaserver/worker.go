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
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/service"
	"github.com/polarismesh/polaris-server/service/healthcheck"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

//全量服务缓存
type ApplicationsRespCache struct {
	AppsResp  *ApplicationsResponse
	Revision  string
	JsonBytes []byte
	XmlBytes  []byte
}

func sha1s(bytes []byte) string {
	r := sha1.Sum(bytes)
	return hex.EncodeToString(r[:])
}

//应用缓存协程
type ApplicationsWorker struct {
	mutex *sync.Mutex

	started uint32

	waitCtx context.Context

	workerCancel context.CancelFunc

	interval time.Duration

	deltaExpireInterval time.Duration

	unhealthyExpireInterval time.Duration
	//全量服务的缓存，数据结构为ApplicationsRespCache
	appsCache *atomic.Value
	//增量数据缓存，数据结构为ApplicationsRespCache
	deltaCache *atomic.Value

	namingServer *service.Server

	healthCheckServer *healthcheck.Server

	namespace string
	//上一次清理增量缓存的时间
	deltaExpireTimesMilli int64
	//版本自增
	VersionIncrement int64
}

//构造函数
func NewApplicationsWorker(interval time.Duration,
	deltaExpireInterval time.Duration, unhealthyExpireInterval time.Duration,
	namingServer *service.Server, healthCheckServer *healthcheck.Server, namespace string) *ApplicationsWorker {
	return &ApplicationsWorker{
		mutex:                   &sync.Mutex{},
		interval:                interval,
		deltaExpireInterval:     deltaExpireInterval,
		unhealthyExpireInterval: unhealthyExpireInterval,
		appsCache:               &atomic.Value{},
		deltaCache:              &atomic.Value{},
		namingServer:            namingServer,
		healthCheckServer:       healthCheckServer,
		namespace:               namespace,
	}
}

//是否已经启动
func (a *ApplicationsWorker) IsStarted() bool {
	return atomic.LoadUint32(&a.started) > 0
}

//从缓存获取全量服务数据
func (a *ApplicationsWorker) GetCachedApps() *ApplicationsRespCache {
	appsValue := a.appsCache.Load()
	if nil != appsValue {
		return appsValue.(*ApplicationsRespCache)
	}
	return nil
}

// GetCachedAppsWithLoad 从缓存中获取全量服务信息，如果不存在就读取
func (a *ApplicationsWorker) GetCachedAppsWithLoad() *ApplicationsRespCache {
	appsRespCache := a.GetCachedApps()
	if nil == appsRespCache {
		ctx := a.StartWorker()
		if nil != ctx {
			<-ctx.Done()
		}
		appsRespCache = a.GetCachedApps()
	}
	return appsRespCache
}

//从缓存获取增量服务数据
func (a *ApplicationsWorker) GetDeltaApps() *ApplicationsRespCache {
	appsValue := a.deltaCache.Load()
	if nil != appsValue {
		return appsValue.(*ApplicationsRespCache)
	}
	return nil
}

func (a *ApplicationsWorker) getCacheServices() map[string]*model.Service {
	var newServices = make(map[string]*model.Service)
	_ = a.namingServer.Cache().Service().IteratorServices(func(key string, value *model.Service) (bool, error) {
		if value.Namespace == a.namespace {
			newServices[value.Name] = value
		}
		return true, nil
	})
	return newServices
}

func newApplications() *Applications {
	return &Applications{
		ApplicationMap: make(map[string]*Application),
		Application:    make([]*Application, 0),
	}
}

func (a *ApplicationsWorker) buildAppsCache(oldAppsCache *ApplicationsRespCache) *ApplicationsRespCache {
	//获取所有的服务数据
	var newServices = a.getCacheServices()
	var instCount int
	svcToRevision := make(map[string]string, len(newServices))
	svcToToInstances := make(map[string][]*model.Instance)
	var changed bool
	for _, newService := range newServices {
		var instances []*model.Instance
		_ = a.namingServer.Cache().Instance().IteratorInstancesWithService(newService.ID,
			func(key string, value *model.Instance) (bool, error) {
				instCount++
				instances = append(instances, value)
				return true, nil
			})
		revision, err := a.namingServer.GetServiceInstanceRevision(newService.ID, instances)
		if nil != err {
			log.Errorf("[EurekaServer]fail to get revision for service %s, err is %v", newService.Name, err)
		}
		// eureka does not return services without instances
		if len(instances) == 0 {
			continue
		}
		svcToRevision[newService.Name] = revision
		svcToToInstances[newService.Name] = instances
	}
	//比较并构建Applications缓存
	hashBuilder := make(map[string]int)
	newApps := newApplications()
	var oldApps *Applications
	if nil != oldAppsCache {
		oldApps = oldAppsCache.AppsResp.Applications
	}
	for svc, instances := range svcToToInstances {
		var newRevision = svcToRevision[svc]
		var targetApp *Application
		if nil != oldApps {
			oldApp, ok := oldApps.ApplicationMap[svc]
			if ok && len(oldApp.Revision) > 0 && oldApp.Revision == newRevision {
				//没有变化
				targetApp = oldApp
			}
		}
		if nil == targetApp {
			//重新构建
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
	if nil != oldApps && len(oldApps.Application) != len(newApps.Application) {
		changed = true
	}
	a.buildVersionAndHashCode(changed, hashBuilder, newApps)
	return constructResponseCache(newApps, instCount, false)
}

func (a *ApplicationsWorker) buildVersionAndHashCode(changed bool, hashBuilder map[string]int, newApps *Applications) {
	var nextVersion int64
	if changed {
		nextVersion = atomic.AddInt64(&a.VersionIncrement, 1)
	} else {
		nextVersion = atomic.LoadInt64(&a.VersionIncrement)
	}
	//构建hashValue
	newApps.AppsHashCode = buildHashStr(hashBuilder)
	newApps.VersionsDelta = strconv.Itoa(int(nextVersion))
}

func constructResponseCache(newApps *Applications, instCount int, delta bool) *ApplicationsRespCache {
	appsHashCode := newApps.AppsHashCode
	newAppsCache := &ApplicationsRespCache{
		AppsResp: &ApplicationsResponse{Applications: newApps},
	}
	//预先做一次序列化，以免高并发时候序列化会使得内存峰值过高
	jsonBytes, err := json.MarshalIndent(newAppsCache.AppsResp, "", " ")
	if nil != err {
		log.Errorf("[EUREKA_SERVER]fail to marshal apps %s to json, err is %v", appsHashCode, err)
	} else {
		newAppsCache.JsonBytes = jsonBytes
	}
	xmlBytes, err := xml.MarshalIndent(newAppsCache.AppsResp.Applications, " ", " ")
	if nil != err {
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

func parseStatus(instance *model.Instance) string {
	if instance.Proto.GetIsolate().GetValue() {
		return StatusOutOfService
	}
	healthy := instance.Proto.GetHealthy().GetValue()
	if healthy {
		return StatusUp
	}
	return StatusDown
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
	if nil != metadata {
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
		CountryId: DefaultCountryId,
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
			Meta: make(map[string]string),
		},
		RealInstances: make(map[string]*model.Instance),
	}
	instanceInfo.AppName = app.Name
	//属于eureka注册的实例
	instanceInfo.InstanceId = eurekaInstanceId
	metadata := instance.Metadata()
	if nil == metadata {
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
		if nil == err {
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
	if location := instance.Location(); nil != location {
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

//假如实例是不健康，而修改周期超过
func checkInstanceExpired(instance *model.Instance, unhealthyExpireInterval time.Duration) bool {
	if instance.Healthy() {
		return true
	}
	healthCheck := instance.HealthCheck()
	if nil == healthCheck || nil == healthCheck.Heartbeat {
		return true
	}
	modifySince := time.Since(instance.ModifyTime)
	return modifySince < unhealthyExpireInterval
}

func (a *ApplicationsWorker) constructApplication(app *Application, instances []*model.Instance) {
	if len(instances) == 0 {
		return
	}
	app.StatusCounts = make(map[string]int)
	//转换时候要区分2种情况，一种是从eureka注册上来的，一种不是
	for _, instance := range instances {
		if !checkInstanceExpired(instance, a.unhealthyExpireInterval) {
			//不返回不健康太久的数据
			continue
		}
		eurekaInstanceId := instance.Proto.GetId().GetValue()

		instanceInfo := buildInstance(app, eurekaInstanceId, instance)
		instanceInfo.RealInstances[instance.Revision()] = instance
		status := instanceInfo.Status
		app.StatusCounts[status] = app.StatusCounts[status] + 1
		app.Instance = append(app.Instance, instanceInfo)
		app.InstanceMap[instanceInfo.InstanceId] = instanceInfo
	}
}

func (a *ApplicationsWorker) timingReloadAppsCache(workerCtx context.Context) {
	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()
	for {
		select {
		case <-workerCtx.Done():
			return
		case <-ticker.C:
			oldApps := a.GetCachedApps()
			newApps := a.buildAppsCache(oldApps)
			newDeltaApps := a.buildDeltaApps(oldApps, newApps)
			a.appsCache.Store(newApps)
			a.deltaCache.Store(newDeltaApps)
		}
	}
}

func diffApplication(oldApplication *Application, newApplication *Application) *Application {
	oldRevision := oldApplication.Revision
	newRevision := newApplication.Revision
	if len(oldRevision) > 0 && len(newRevision) > 0 && oldRevision == newRevision {
		//完全相同，没有变更
		return nil
	}
	diffApplication := &Application{
		Name: newApplication.Name,
	}
	//获取新增和修改
	newInstances := newApplication.Instance
	if len(newInstances) > 0 {
		for _, instance := range newInstances {
			oldInstance := oldApplication.GetInstance(instance.InstanceId)
			if nil == oldInstance {
				//新增实例
				diffApplication.Instance = append(diffApplication.Instance, instance)
				continue
			}
			//比较实际的实例是否发生了变更
			if oldInstance.Equals(instance) {
				continue
			}
			//新创建一个instance
			diffApplication.Instance = append(diffApplication.Instance, instance.Clone(ActionModified))
		}
	}
	//获取删除
	oldInstances := oldApplication.Instance
	if len(oldInstances) > 0 {
		for _, instance := range oldInstances {
			newInstance := newApplication.GetInstance(instance.InstanceId)
			if nil == newInstance {
				//被删除了
				//新创建一个instance
				diffApplication.Instance = append(diffApplication.Instance, instance.Clone(ActionDeleted))
			}
		}
	}
	if len(diffApplication.Instance) > 0 {
		return diffApplication
	}
	return nil
}

func (a *ApplicationsWorker) buildDeltaApps(
	oldAppsCache *ApplicationsRespCache, newAppsCache *ApplicationsRespCache) *ApplicationsRespCache {
	var oldDeltaAppsCache *ApplicationsRespCache
	curTimeMs := time.Now().UnixNano() / 1e6
	diffTimeMs := curTimeMs - a.deltaExpireTimesMilli
	if diffTimeMs > 0 && diffTimeMs < a.deltaExpireInterval.Milliseconds() {
		oldDeltaAppsCache = a.GetDeltaApps()
	} else {
		a.deltaExpireTimesMilli = curTimeMs
	}
	var instCount int
	newApps := newAppsCache.AppsResp.Applications
	//1. 创建新的delta对象
	newDeltaApps := &Applications{
		VersionsDelta: newApps.VersionsDelta,
		AppsHashCode:  newApps.AppsHashCode,
		Application:   make([]*Application, 0),
	}
	//2. 拷贝老的delta内容
	var oldDeltaApps *Applications
	if nil != oldDeltaAppsCache {
		oldDeltaApps = oldDeltaAppsCache.AppsResp.Applications
	}
	if nil != oldDeltaApps && len(oldDeltaApps.Application) > 0 {
		for _, app := range oldDeltaApps.Application {
			newDeltaApps.Application = append(newDeltaApps.Application, app)
			instCount += len(app.Instance)
		}
	}
	//3. 比较revision是否发生变更
	if oldAppsCache.Revision != newAppsCache.Revision {
		//3. 比较修改和新增
		oldApps := oldAppsCache.AppsResp.Applications
		applications := newApps.Application
		if len(applications) > 0 {
			for _, application := range applications {
				var oldApplication = oldApps.GetApplication(application.Name)
				if nil == oldApplication {
					//新增，全部加入
					newDeltaApps.Application = append(newDeltaApps.Application, application)
					instCount += len(application.Instance)
					continue
				}
				//修改，需要比较实例的变更
				diffApp := diffApplication(oldApplication, application)
				if nil != diffApp {
					newDeltaApps.Application = append(newDeltaApps.Application, diffApp)
					instCount += len(diffApp.Instance)
				}
			}
		}
		//4. 比较删除
		oldApplications := oldApps.Application
		if len(oldApplications) > 0 {
			for _, application := range oldApplications {
				var newApplication = newApps.GetApplication(application.Name)
				if nil == newApplication {
					//删除
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

//启动缓存构建器
func (a *ApplicationsWorker) StartWorker() context.Context {
	if nil != a.GetCachedApps() {
		return nil
	}
	a.mutex.Lock()
	defer a.mutex.Unlock()
	if !atomic.CompareAndSwapUint32(&a.started, 0, 1) {
		return a.waitCtx
	}
	var waitCancel context.CancelFunc
	//进行首次缓存构建
	a.waitCtx, waitCancel = context.WithCancel(context.Background())
	defer waitCancel()
	apps := a.buildAppsCache(nil)
	a.appsCache.Store(apps)
	a.deltaCache.Store(apps)
	a.deltaExpireTimesMilli = time.Now().UnixNano() / 1e6
	//开启定时任务构建
	var workerCtx context.Context
	workerCtx, a.workerCancel = context.WithCancel(context.Background())
	go a.timingReloadAppsCache(workerCtx)
	return nil
}

//结束任务
func (a *ApplicationsWorker) Stop() {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	if atomic.CompareAndSwapUint32(&a.started, 1, 0) {
		a.workerCancel()
	}
}
