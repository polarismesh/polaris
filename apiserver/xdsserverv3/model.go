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
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"github.com/polarismesh/specification/source/go/api/v1/traffic_manage"

	"github.com/polarismesh/polaris/common/model"
)

const (
	K8sDnsResolveSuffixSvc             = ".svc"
	K8sDnsResolveSuffixSvcCluster      = ".svc.cluster"
	K8sDnsResolveSuffixSvcClusterLocal = ".svc.cluster.local"
)

const (
	TLSModeTag        = "polarismesh.cn/tls-mode"
	TLSModeNone       = "none"
	TLSModeStrict     = "strict"
	TLSModePermissive = "permissive"
)

// ServiceInfo 北极星服务结构体
type ServiceInfo struct {
	ID                   string
	Name                 string
	Namespace            string
	AliasFor             *model.Service
	Instances            []*apiservice.Instance
	SvcInsRevision       string
	Routing              *traffic_manage.Routing
	SvcRoutingRevision   string
	Ports                string
	RateLimit            *traffic_manage.RateLimit
	SvcRateLimitRevision string
}

func (s *ServiceInfo) matchService(ns, name string) bool {
	if s.Namespace == ns && s.Name == name {
		return true
	}

	if s.AliasFor != nil {
		if s.AliasFor.Namespace == ns && s.AliasFor.Name == name {
			return true
		}
	}
	return false
}
