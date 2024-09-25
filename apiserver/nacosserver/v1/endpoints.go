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
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"google.golang.org/protobuf/types/known/wrapperspb"

	api "github.com/polarismesh/polaris/common/api/v1"
)

func (n *NacosV1Server) GetAddressServer() (*restful.WebService, error) {
	ws := new(restful.WebService)
	ws.Route(ws.GET("/nacos/serverlist").To(n.FetchNacosEndpoints))
	return ws, nil
}

// FetchNacosEndpoints 处理 nacos 地址服务器
func (n *NacosV1Server) FetchNacosEndpoints(req *restful.Request, rsp *restful.Response) {
	serverSvcName, _ := n.option["serverService"].(string)
	serverSvcNamespace, _ := n.option["serverNamespace"].(string)

	if serverSvcName == "" || serverSvcNamespace == "" {
		rsp.WriteHeader(http.StatusNotFound)
		return
	}

	insResp := n.discoverOpt.OriginDiscoverSvr.ServiceInstancesCache(context.Background(), &service_manage.DiscoverFilter{
		OnlyHealthyInstance: true,
	}, &service_manage.Service{
		Namespace: wrapperspb.String(serverSvcNamespace),
		Name:      wrapperspb.String(serverSvcName),
	})

	if !api.IsSuccess(insResp) {
		rsp.WriteHeader(api.CalcCode(insResp))
		return
	}

	ips := []string{}
	for i := range insResp.GetInstances() {
		item := insResp.GetInstances()[i]
		ips = append(ips, fmt.Sprintf("%s:%d", item.GetHost().GetValue(), item.GetPort().GetValue()))
	}
	if len(ips) == 0 {
		rsp.WriteHeader(http.StatusNotFound)
		return
	}

	rsp.WriteHeader(http.StatusOK)
	rsp.Write([]byte(strings.Join(ips, "\n")))
}
