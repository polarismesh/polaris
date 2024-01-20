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

package v2

import (
	"github.com/emicklei/go-restful/v3"

	"github.com/polarismesh/polaris/apiserver/httpserver/docs"
	"github.com/polarismesh/polaris/namespace"
	"github.com/polarismesh/polaris/service"
	"github.com/polarismesh/polaris/service/healthcheck"
)

// HTTPServerV2
type HTTPServerV2 struct {
	namespaceServer   namespace.NamespaceOperateServer
	namingServer      service.DiscoverServer
	healthCheckServer *healthcheck.Server
}

// NewV2Server 创建V2版本的HTTPServer
func NewV2Server(
	namespaceServer namespace.NamespaceOperateServer,
	namingServer service.DiscoverServer,
	healthCheckServer *healthcheck.Server) *HTTPServerV2 {
	return &HTTPServerV2{
		namespaceServer:   namespaceServer,
		namingServer:      namingServer,
		healthCheckServer: healthCheckServer,
	}
}

const (
	defaultReadAccess    string = "default-read"
	defaultAccess        string = "default"
	circuitBreakerAccess string = "circuitbreaker"
	routingAccess        string = "router"
	rateLimitAccess      string = "ratelimit"
)

// GetConsoleAccessServer 注册管理端接口
func (h *HTTPServerV2) GetConsoleAccessServer(include []string) (*restful.WebService, error) {
	consoleAccess := []string{defaultAccess}
	ws := new(restful.WebService)
	ws.Path("/naming/v2").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

	// 如果为空，则开启全部接口
	if len(include) == 0 {
		include = consoleAccess
	}
	oldInclude := include

	for _, item := range oldInclude {
		if item == defaultReadAccess {
			include = []string{defaultReadAccess}
			break
		}
	}

	for _, item := range oldInclude {
		if item == defaultAccess {
			include = consoleAccess
			break
		}
	}
	for _, item := range include {
		switch item {
		case defaultReadAccess:
			h.addDefaultReadAccess(ws)
		case defaultAccess:
			h.addDefaultAccess(ws)
		case routingAccess:
			h.addRouterRuleAccess(ws)
		}
	}
	return ws, nil
}

// addDefaultReadAccess 增加默认读接口
func (h *HTTPServerV2) addDefaultReadAccess(ws *restful.WebService) {
	ws.Route(docs.EnrichGetRouterRuleApiDocs(ws.GET("/routings").To(h.GetRoutings)))
}

// addDefaultAccess 增加默认接口
func (h *HTTPServerV2) addDefaultAccess(ws *restful.WebService) {
	h.addRouterRuleAccess(ws)
}

// addDefaultAccess 增加默认接口
func (h *HTTPServerV2) addRouterRuleAccess(ws *restful.WebService) {
	ws.Route(docs.EnrichCreateRouterRuleApiDocs(ws.POST("/routings").To(h.CreateRoutings)))
	ws.Route(docs.EnrichDeleteRouterRuleApiDocs(ws.POST("/routings/delete").To(h.DeleteRoutings)))
	ws.Route(docs.EnrichUpdateRouterRuleApiDocs(ws.PUT("/routings").To(h.UpdateRoutings)))
	ws.Route(docs.EnrichGetRouterRuleApiDocs(ws.GET("/routings").To(h.GetRoutings)))
	ws.Route(docs.EnrichEnableRouterRuleApiDocs(ws.PUT("/routings/enable").To(h.EnableRoutings)))
}
