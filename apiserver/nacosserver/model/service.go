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
	"strings"

	"github.com/polarismesh/polaris/service"
)

type ServiceKey struct {
	Namespace string
	Group     string
	Name      string
}

func (s ServiceKey) String() string {
	return s.Namespace + "/" + s.Group + "/" + s.Name
}

type ServiceMetadata struct {
	ServiceKey
	ServiceID           string
	Clusters            map[string]struct{}
	ProtectionThreshold float64
	ExtendData          map[string]string
}

func HandleServiceListRequest(discoverSvr service.DiscoverServer, namespace string, groupName string,
	pageNo int, pageSize int) ([]string, int) {
	_, services := discoverSvr.Cache().Service().ListServices(namespace)
	offset := (pageNo - 1) * pageSize
	limit := pageSize
	if offset < 0 {
		offset = 0
	}
	if offset > len(services) {
		return []string{}, 0
	}
	groupPrefix := groupName + ReplaceNacosGroupConnectStr
	if groupName == DefaultServiceGroup {
		groupPrefix = ""
	}
	hasGroupPrefix := len(groupPrefix) != 0
	temp := make([]string, 0, len(services))
	for i := range services {
		svc := services[i]
		if !hasGroupPrefix {
			temp = append(temp, svc.Name)
			continue
		}
		if strings.HasPrefix(svc.Name, groupPrefix) {
			temp = append(temp, GetServiceName(svc.Name))
		}
	}
	var viewList []string
	if offset+limit > len(services) {
		viewList = temp[offset:]
	} else {
		viewList = temp[offset : offset+limit]
	}
	return viewList, len(services)
}
