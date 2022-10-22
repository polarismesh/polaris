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

package namespace

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"github.com/polarismesh/polaris/auth"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

// serverAuthAbility 带有鉴权能力的 discoverServer
//
// 该层会对请求参数做一些调整，根据具体的请求发起人，设置为数据对应的 owner，不可为为别人进行创建资源
type serverAuthAbility struct {
	targetServer *Server
	authSvr      auth.AuthServer
	authMgn      auth.AuthChecker
}

func newServerAuthAbility(targetServer *Server, authSvr auth.AuthServer) NamespaceOperateServer {
	proxy := &serverAuthAbility{
		targetServer: targetServer,
		authSvr:      authSvr,
		authMgn:      authSvr.GetAuthChecker(),
	}

	targetServer.SetResourceHooks(proxy)
	return proxy
}

// collectNamespaceAuthContext 对于命名空间的处理，收集所有的与鉴权的相关信息
func (svr *serverAuthAbility) collectNamespaceAuthContext(ctx context.Context, req []*api.Namespace,
	resourceOp model.ResourceOperation, methodName string) *model.AcquireContext {

	return model.NewAcquireContext(
		model.WithRequestContext(ctx),
		model.WithOperation(resourceOp),
		model.WithToken(utils.ParseAuthToken(ctx)),
		model.WithModule(model.CoreModule),
		model.WithMethod(methodName),
		model.WithAccessResources(svr.queryNamespaceResource(req)),
	)
}

// queryNamespaceResource 根据所给的 namespace 信息，收集对应的 ResourceEntry 列表
func (svr *serverAuthAbility) queryNamespaceResource(
	req []*api.Namespace) map[api.ResourceType][]model.ResourceEntry {

	names := utils.NewStringSet()
	for index := range req {
		names.Add(req[index].Name.GetValue())
	}
	param := names.ToSlice()
	nsArr := svr.targetServer.caches.Namespace().GetNamespacesByName(param)

	temp := make([]model.ResourceEntry, 0, len(nsArr))

	for index := range nsArr {
		ns := nsArr[index]
		temp = append(temp, model.ResourceEntry{
			ID:    ns.Name,
			Owner: ns.Owner,
		})
	}

	ret := map[api.ResourceType][]model.ResourceEntry{
		api.ResourceType_Namespaces: temp,
	}
	authLog.Debug("[Auth][Server] collect namespace access res", zap.Any("res", ret))
	return ret
}

func convertToErrCode(err error) uint32 {
	if errors.Is(err, model.ErrorTokenNotExist) {
		return api.TokenNotExisted
	}
	if errors.Is(err, model.ErrorTokenDisabled) {
		return api.TokenDisabled
	}
	return api.NotAllowedAccess
}
