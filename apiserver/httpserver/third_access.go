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

package httpserver

import (
	"net/http"
	"net/http/pprof"

	"github.com/emicklei/go-restful/v3"
	"github.com/go-openapi/spec"
	restfulspec "github.com/polarismesh/go-restful-openapi/v2"

	"github.com/polarismesh/polaris/common/metrics"
)

// enablePprofAccess 开启pprof接口
func (h *HTTPServer) enablePprofAccess(wsContainer *restful.Container) {
	log.Infof("open http access for pprof")
	wsContainer.Handle("/debug/pprof/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.enablePprof.Load() {
			pprof.Index(w, r)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	wsContainer.Handle("/debug/pprof/cmdline", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.enablePprof.Load() {
			pprof.Cmdline(w, r)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	wsContainer.Handle("/debug/pprof/profile", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.enablePprof.Load() {
			pprof.Profile(w, r)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	wsContainer.Handle("/debug/pprof/symbol", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.enablePprof.Load() {
			pprof.Symbol(w, r)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	wsContainer.Handle("/debug/pprof/trace", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.enablePprof.Load() {
			pprof.Trace(w, r)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
}

// enablePrometheusAccess 开启 Prometheus 接口
func (h *HTTPServer) enablePrometheusAccess(wsContainer *restful.Container) {
	log.Infof("open http access for prometheus")

	wsContainer.Handle("/metrics", metrics.GetHttpHandler())
}

func (h *HTTPServer) enableSwaggerAPI(wsContainer *restful.Container) {
	log.Infof("[HTTPServer] open http access for swagger API")
	config := restfulspec.Config{
		WebServices:                   wsContainer.RegisteredWebServices(), // you control what services are visible
		APIPath:                       "/apidocs.json",
		PostBuildSwaggerObjectHandler: enrichSwaggerObject,
	}

	if h.enableSwagger {
		wsContainer.Add(restfulspec.NewOpenAPIService(config))
	}
}

func enrichSwaggerObject(swo *spec.Swagger) {
	swo.Tags = []spec.Tag{
		{TagProps: spec.TagProps{
			Name:        "Client",
			Description: "客户端接口"}},
		{TagProps: spec.TagProps{
			Name:        "ConfigConsole",
			Description: "配置管理"}},
		{TagProps: spec.TagProps{
			Name:        "CircuitBreakers",
			Description: "熔断规则管理"}},
		{TagProps: spec.TagProps{
			Name:        "Instances",
			Description: "实例管理"}},
		{TagProps: spec.TagProps{
			Name:        "Maintain",
			Description: "运维接口"}},
		{TagProps: spec.TagProps{
			Name:        "Namespaces",
			Description: "命名空间管理"}},
		{TagProps: spec.TagProps{
			Name:        "RoutingRules",
			Description: "路由规则管理"}},
		{TagProps: spec.TagProps{
			Name:        "RateLimits",
			Description: "限流规则管理"}},
		{TagProps: spec.TagProps{
			Name:        "Services",
			Description: "服务管理"}},
		{TagProps: spec.TagProps{
			Name:        "AuthRule",
			Description: "鉴权规则管理"}},
		{TagProps: spec.TagProps{
			Name:        "Users",
			Description: "用户/用户组管理"}},
	}

	swo.SecurityDefinitions = map[string]*spec.SecurityScheme{
		"api_key": spec.APIKeyAuth("X-Polaris-Token", "header"),
	}

	var securitySetting []map[string][]string
	apiKey := make(map[string][]string, 0)
	apiKey["api_key"] = []string{}
	securitySetting = append(securitySetting, apiKey)
	swo.Security = securitySetting
}
