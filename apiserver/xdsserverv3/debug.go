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
	"net/http"

	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"

	"github.com/polarismesh/polaris/apiserver/xdsserverv3/cache"
	"github.com/polarismesh/polaris/common/utils"
)

func (x *XDSServer) listXDSNodes(resp http.ResponseWriter, req *http.Request) {
	cType := req.URL.Query().Get("type")
	var nodes interface{}

	switch cType {
	case "sidecar":
		nodes = x.nodeMgr.ListSidecarNodes()
	case "gateway":
		nodes = x.nodeMgr.ListGatewayNodes()
	}

	data := map[string]interface{}{
		"code": apimodel.Code_ExecuteSuccess,
		"info": "execute success",
		"data": nodes,
	}

	ret := utils.MustJson(data)
	resp.WriteHeader(http.StatusOK)
	_, _ = resp.Write([]byte(ret))
}

func (x *XDSServer) listXDSResources(resp http.ResponseWriter, req *http.Request) {
	resources := map[string]interface{}{}
	x.cache.Caches.ReadRange(func(key string, val cachev3.Cache) {
		linearCache := val.(*cache.LinearCache)
		resources[key] = map[string]interface{}{
			"resources": linearCache.GetResources(),
		}
	})

	data := map[string]interface{}{
		"code":  apimodel.Code_ExecuteSuccess,
		"info":  "execute success",
		"data":  resources,
		"count": len(resources),
	}

	ret := utils.MustJson(data)
	resp.WriteHeader(http.StatusOK)
	_, _ = resp.Write([]byte(ret))
}
