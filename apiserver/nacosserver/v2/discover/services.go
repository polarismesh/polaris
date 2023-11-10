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

	"github.com/polarismesh/polaris/apiserver/nacosserver/model"
	nacosmodel "github.com/polarismesh/polaris/apiserver/nacosserver/model"
	nacospb "github.com/polarismesh/polaris/apiserver/nacosserver/v2/pb"
	"github.com/polarismesh/polaris/apiserver/nacosserver/v2/remote"
)

func (h *DiscoverServer) handleServiceListRequest(ctx context.Context, req nacospb.BaseRequest,
	meta nacospb.RequestMeta) (nacospb.BaseResponse, error) {
	svcListReq, ok := req.(*nacospb.ServiceListRequest)
	if !ok {
		return nil, remote.ErrorInvalidRequestBodyType
	}
	resp := &nacospb.ServiceListResponse{
		Response: &nacospb.Response{
			ResultCode: int(model.Response_Success.Code),
			Success:    true,
			Message:    "success",
		},
		Count:        0,
		ServiceNames: []string{},
	}

	namespace := nacosmodel.ToPolarisNamespace(svcListReq.Namespace)
	viewList, count := model.HandleServiceListRequest(h.discoverSvr, namespace, svcListReq.GroupName,
		svcListReq.PageNo, svcListReq.PageSize)
	resp.ServiceNames = viewList
	resp.Count = count
	return resp, nil
}
