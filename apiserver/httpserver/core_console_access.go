/*
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
	"fmt"

	"github.com/emicklei/go-restful/v3"
)

// GetCoreConsoleAccessServer 增加配置中心模块之后，namespace 作为两个模块的公共模块需要独立， restful path 以 /core 开头
func (h *HTTPServer) GetCoreConsoleAccessServer(ws *restful.WebService, include []string) error {
	consoleAccess := []string{defaultAccess}

	if len(include) == 0 {
		include = consoleAccess
	}

	var hasDefault = false
	for _, item := range include {
		if item == defaultAccess {
			hasDefault = true
			break
		}
	}
	for _, item := range include {
		switch item {
		case defaultReadAccess:
			if !hasDefault {
				h.addCoreDefaultReadAccess(ws)
			}
		case defaultAccess:
			h.addCoreDefaultAccess(ws)
		default:
			log.Errorf("[HttpServer][Core] method %s does not exist in httpserver console access", item)
			return fmt.Errorf("method %s does not exist in httpserver console access", item)
		}
	}
	return nil
}

func (h *HTTPServer) addCoreDefaultReadAccess(ws *restful.WebService) {
	ws.Route(ws.GET("/namespaces").To(h.GetNamespaces))
	ws.Route(ws.GET("/namespaces/token").To(h.GetNamespaceToken))
}

func (h *HTTPServer) addCoreDefaultAccess(ws *restful.WebService) {
	ws.Route(ws.POST("/namespaces").To(h.CreateNamespaces))
	ws.Route(ws.POST("/namespaces/delete").To(h.DeleteNamespaces))
	ws.Route(ws.PUT("/namespaces").To(h.UpdateNamespaces))
	ws.Route(ws.GET("/namespaces").To(h.GetNamespaces))
	ws.Route(ws.GET("/namespaces/token").To(h.GetNamespaceToken))
	ws.Route(ws.PUT("/namespaces/token").To(h.UpdateNamespaceToken))
}
