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
	"strings"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"

	"github.com/polarismesh/polaris/apiserver/xdsserverv3/resource"
	"github.com/polarismesh/polaris/common/utils"
)

func (x *XDSServer) listXDSNodes(resp http.ResponseWriter, req *http.Request) {
	cType := req.URL.Query().Get("type")
	data := map[string]interface{}{
		"code": apimodel.Code_ExecuteSuccess,
		"info": "execute success",
		"data": x.nodeMgr.ListEnvoyNodesView(resource.RunType(cType)),
	}

	ret := utils.MustJson(data)
	resp.WriteHeader(http.StatusOK)
	_, _ = resp.Write([]byte(ret))
}

func (x *XDSServer) listXDSResource(resp http.ResponseWriter, req *http.Request) {
	cType := req.URL.Query().Get("type")
	nodeId := req.URL.Query().Get("nodeId")
	service := req.URL.Query().Get("service")
	namespace := req.URL.Query().Get("namespace")
	if namespace == "" {
		namespace = "default"
	}

	res := x.cache.GetResources(resource.FromSimpleXDS(cType), namespace, nodeId)
	if len(service) != 0 {
		copyData := make(map[string]types.Resource, len(res))
		hasSvc := len(service) != 0
		for k, v := range res {
			if hasSvc && !strings.Contains(k, service) {
				continue
			}
			copyData[k] = v
		}
		res = copyData
	}

	data := map[string]interface{}{
		"code": apimodel.Code_ExecuteSuccess,
		"info": "execute success",
		"data": res,
	}

	ret := utils.MustJson(data)
	resp.WriteHeader(http.StatusOK)
	_, _ = resp.Write([]byte(ret))
}
