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

package v1

import (
	"errors"
	"sync"

	"github.com/emicklei/go-restful/v3"

	"github.com/polarismesh/polaris/apiserver/nacosserver/model"
)

var (
	dlock                    sync.RWMutex
	debugRegistry            = make(map[string]func(*restful.WebService) *restful.RouteBuilder)
	ErrorDuplicateDebugRoute = errors.New("duplicate debug route")
)

func RegistryDebugRoute(name string, rb func(*restful.WebService) *restful.RouteBuilder) error {
	dlock.Lock()
	defer dlock.Unlock()

	if _, ok := debugRegistry[name]; ok {
		return ErrorDuplicateDebugRoute
	}

	debugRegistry[name] = rb
	return nil
}

func (n *NacosV1Server) GetDebugServer() (*restful.WebService, error) {
	ws := new(restful.WebService)
	ws.Path("/nacos/v1/debug").Consumes(restful.MIME_JSON, model.MIME).Produces(restful.MIME_JSON)
	n.addDebugAccess(ws)
	return ws, nil
}

func (n *NacosV1Server) addDebugAccess(ws *restful.WebService) {
	dlock.RLock()
	defer dlock.RUnlock()
	for i := range debugRegistry {
		ws.Route(debugRegistry[i](ws))
	}
}
