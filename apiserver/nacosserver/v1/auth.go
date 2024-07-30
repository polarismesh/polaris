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

	"github.com/emicklei/go-restful/v3"

	nacoshttp "github.com/polarismesh/polaris/apiserver/nacosserver/v1/http"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
)

func (n *NacosV1Server) GetAuthServer() (*restful.WebService, error) {
	ws := new(restful.WebService)
	ws.Route(ws.POST("/nacos/v1/auth/login").To(n.Login))
	ws.Route(ws.POST("/nacos/v1/auth/users/login").To(n.Login))
	return ws, nil
}

func (n *NacosV1Server) Login(req *restful.Request, rsp *restful.Response) {
	handler := nacoshttp.Handler{
		Request:  req,
		Response: rsp,
	}

	ctx := handler.ParseHeaderContext()
	data, err := n.handleLogin(ctx, nacoshttp.ParseQueryParams(req))
	if err != nil {
		nacoshttp.WrirteNacosErrorResponse(err, rsp)
		return
	}
	nacoshttp.WrirteNacosResponse(data, rsp)
}

func (n *NacosV1Server) handleLogin(ctx context.Context, params map[string]string) (map[string]interface{}, error) {
	username := params["username"]
	token := params["password"]
	authCtx := authcommon.NewAcquireContext(
		authcommon.WithFromClient(),
		authcommon.WithRequestContext(ctx),
	)

	if err := n.discoverOpt.UserSvr.CheckCredential(authCtx); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"accessToken": token,
		"tokenTtl":    120,
		"globalAdmin": false,
		"username":    username,
	}, nil
}
