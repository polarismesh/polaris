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
	"sort"
	"strings"
)

const (
	entityTypeVip  = 0
	entityTypeSVip = 1
)

// VipCacheKey key for reference the vip cache
type VipCacheKey struct {
	entityType       int
	targetVipAddress string
}

// BuildApplicationsForVip build applications with target vip
func BuildApplicationsForVip(key *VipCacheKey, appsCache *ApplicationsRespCache) *ApplicationsRespCache {
	toReturn := newApplications()
	applications := appsCache.AppsResp.Applications
	hashBuilder := make(map[string]int)
	var instCount int
	for _, application := range applications.Application {
		var appToAdd *Application
		for _, instance := range application.Instance {
			var vipAddress string
			switch key.entityType {
			case entityTypeVip:
				vipAddress = instance.VipAddress
			case entityTypeSVip:
				vipAddress = instance.SecureVipAddress
			default:
				continue
			}
			if len(vipAddress) == 0 {
				continue
			}
			vipAddresses := strings.Split(vipAddress, ",")
			sort.Strings(vipAddresses)
			searchIdx := sort.SearchStrings(vipAddresses, key.targetVipAddress)
			found := searchIdx < len(vipAddresses) && vipAddresses[searchIdx] == key.targetVipAddress
			if found {
				if appToAdd == nil {
					appToAdd = &Application{
						Name:         application.Name,
						InstanceMap:  make(map[string]*InstanceInfo),
						StatusCounts: make(map[string]int),
					}
					toReturn.Application = append(toReturn.Application, appToAdd)
					toReturn.ApplicationMap[application.Name] = appToAdd
				}
				appToAdd.Instance = append(appToAdd.Instance, instance)
				instCount++
				appToAdd.StatusCounts[instance.Status] = appToAdd.StatusCounts[instance.Status] + 1
			}
		}
		if appToAdd == nil {
			continue
		}
		statusCount := appToAdd.StatusCounts
		if len(statusCount) > 0 {
			for status, count := range statusCount {
				hashBuilder[status] = hashBuilder[status] + count
			}
		}
	}
	buildHashCode(applications.VersionsDelta, hashBuilder, toReturn)
	return constructResponseCache(toReturn, instCount, false)
}
