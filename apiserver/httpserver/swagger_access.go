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
	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"github.com/go-openapi/spec"
)

func (h *HTTPServer) enableSwaggerAPI(wsContainer *restful.Container) {
	log.Infof("open http access for swagger API")
	config := restfulspec.Config{
		WebServices:                   wsContainer.RegisteredWebServices(), // you control what services are visible
		APIPath:                       "/apidocs.json",
		PostBuildSwaggerObjectHandler: enrichSwaggerObject}
	wsContainer.Add(restfulspec.NewOpenAPIService(config))
}

func enrichSwaggerObject(swo *spec.Swagger) {
	swo.Info = &spec.Info{
		InfoProps: spec.InfoProps{
			Title:       "Polaris Server",
			Description: "一个支持多语言、多框架的云原生服务发现和治理中心\n\n提供高性能SDK和无侵入Sidecar两种接入方式\n\n",
			Contact: &spec.ContactInfo{
				ContactInfoProps: spec.ContactInfoProps{
					Name: "Polaris Mesh",
					//Email: "polaris@qq.com",
					URL: "https://polarismesh.cn/",
				},
			},
			License: &spec.License{
				LicenseProps: spec.LicenseProps{
					Name: "BSD 3-Clause",
					URL:  "https://github.com/polarismesh/polaris/blob/main/LICENSE",
				},
			},
			Version: "v1.12.0-alpha",
		},
	}
	swo.Tags = []spec.Tag{
		{TagProps: spec.TagProps{
			Name:        "Namespaces",
			Description: "命名空间管理"}},
		{TagProps: spec.TagProps{
			Name:        "Services",
			Description: "服务管理"}},
		{TagProps: spec.TagProps{
			Name:        "Alias",
			Description: "服务别名管理"}},
		{TagProps: spec.TagProps{
			Name:        "Instances",
			Description: "实例管理"}},
		{TagProps: spec.TagProps{
			Name:        "Routing",
			Description: "路由规则管理"}},
		{TagProps: spec.TagProps{
			Name:        "RateLimits",
			Description: "限流规则管理"}},
		{TagProps: spec.TagProps{
			Name:        "RegisterInstance",
			Description: "服务发现"}},
		{TagProps: spec.TagProps{
			Name:        "ConfigClient",
			Description: "客户端API接口"}},
		{TagProps: spec.TagProps{
			Name:        "ConfigConsole",
			Description: "服务端接口"}},
		{TagProps: spec.TagProps{
			Name:        "auth",
			Description: "鉴权管理"}},
	}

	swo.SecurityDefinitions = map[string]*spec.SecurityScheme{
		"api_key": spec.APIKeyAuth("X-Polaris-Token", "header"),
	}

	enrichSwaggerObjectSecurity(swo)
}

func enrichSwaggerObjectSecurity(swo *spec.Swagger) {
	for p := range swo.Paths.Paths {
		path, err := swo.Paths.JSONLookup(p)
		if err != nil {
			log.Errorf("skipping Security openapi spec for %s, %s", path, err.Error())
			continue
		}
		pItem := path.(*spec.PathItem)

		var pOption *spec.Operation
		if pItem.Get != nil {
			pOption = pItem.Get
			pOption.SecuredWith("api_key")
		}
		if pItem.Head != nil {
			pOption = pItem.Head
			pOption.SecuredWith("api_key")
		}
		if pItem.Delete != nil {
			pOption = pItem.Delete
			pOption.SecuredWith("api_key")
		}
		if pItem.Put != nil {
			pOption = pItem.Put
			pOption.SecuredWith("api_key")
		}
		if pItem.Options != nil {
			pOption = pItem.Options
			pOption.SecuredWith("api_key")
		}
		if pItem.Patch != nil {
			pOption = pItem.Patch
			pOption.SecuredWith("api_key")
		}
		if pItem.Post != nil {
			pOption = pItem.Post
			pOption.SecuredWith("api_key")
		}
	}

}
