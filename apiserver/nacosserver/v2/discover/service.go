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

package discover

import (
	"context"
	"strings"

	"github.com/polarismesh/polaris/apiserver/nacosserver/core"
	"github.com/polarismesh/polaris/apiserver/nacosserver/model"
	nacospb "github.com/polarismesh/polaris/apiserver/nacosserver/v2/pb"
	"github.com/polarismesh/polaris/apiserver/nacosserver/v2/remote"
)

func (h *DiscoverServer) handleServiceQueryRequest(ctx context.Context, req nacospb.BaseRequest,
	meta nacospb.RequestMeta) (nacospb.BaseResponse, error) {
	svcQueryReq, ok := req.(*nacospb.ServiceQueryRequest)
	if !ok {
		return nil, remote.ErrorInvalidRequestBodyType
	}
	resp := &nacospb.QueryServiceResponse{
		Response: &nacospb.Response{
			ResultCode: int(model.Response_Success.Code),
			Success:    true,
			Message:    "success",
		},
	}
	namespace := model.ToPolarisNamespace(svcQueryReq.Namespace)
	filterCtx := &core.FilterContext{
		Service:     core.ToNacosService(h.discoverSvr.Cache(), namespace, svcQueryReq.ServiceName, svcQueryReq.GroupName),
		Clusters:    strings.Split(svcQueryReq.Cluster, ","),
		EnableOnly:  true,
		HealthyOnly: svcQueryReq.HealthyOnly,
	}
	// 默认只下发 enable 的实例
	result := h.store.ListInstances(filterCtx, core.SelectInstancesWithHealthyProtection)
	resp.ServiceInfo = model.Service{
		Name:                     svcQueryReq.ServiceName,
		GroupName:                svcQueryReq.GroupName,
		Clusters:                 result.Clusters,
		CacheMillis:              result.CacheMillis,
		Hosts:                    result.Hosts,
		Checksum:                 result.Checksum,
		LastRefTime:              result.LastRefTime,
		ReachProtectionThreshold: result.ReachProtectionThreshold,
	}
	return resp, nil
}
