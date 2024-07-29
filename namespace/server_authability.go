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

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/auth"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
)

// serverAuthAbility 带有鉴权能力的 discoverServer
//
// 该层会对请求参数做一些调整，根据具体的请求发起人，设置为数据对应的 owner，不可为为别人进行创建资源
type serverAuthAbility struct {
	targetServer *Server
	userMgn      auth.UserServer
	policySvr    auth.StrategyServer
}

func newServerAuthAbility(targetServer *Server,
	userMgn auth.UserServer, policySvr auth.StrategyServer) NamespaceOperateServer {
	proxy := &serverAuthAbility{
		targetServer: targetServer,
		userMgn:      userMgn,
		policySvr:    policySvr,
	}

	targetServer.SetResourceHooks(proxy)
	return proxy
}

// collectNamespaceAuthContext 对于命名空间的处理，收集所有的与鉴权的相关信息
func (svr *serverAuthAbility) collectNamespaceAuthContext(ctx context.Context, req []*apimodel.Namespace,
	resourceOp authcommon.ResourceOperation, methodName authcommon.ServerFunctionName) *authcommon.AcquireContext {
	return authcommon.NewAcquireContext(
		authcommon.WithRequestContext(ctx),
		authcommon.WithOperation(resourceOp),
		authcommon.WithModule(authcommon.CoreModule),
		authcommon.WithMethod(methodName),
		authcommon.WithAccessResources(svr.queryNamespaceResource(req)),
	)
}

// queryNamespaceResource 根据所给的 namespace 信息，收集对应的 ResourceEntry 列表
func (svr *serverAuthAbility) queryNamespaceResource(
	req []*apimodel.Namespace) map[apisecurity.ResourceType][]authcommon.ResourceEntry {
	names := utils.NewSet[string]()
	for index := range req {
		names.Add(req[index].Name.GetValue())
	}
	param := names.ToSlice()
	nsArr := svr.targetServer.caches.Namespace().GetNamespacesByName(param)

	temp := make([]authcommon.ResourceEntry, 0, len(nsArr))

	for index := range nsArr {
		ns := nsArr[index]
		temp = append(temp, authcommon.ResourceEntry{
			Type:  apisecurity.ResourceType_Namespaces,
			ID:    ns.Name,
			Owner: ns.Owner,
		})
	}

	ret := map[apisecurity.ResourceType][]authcommon.ResourceEntry{
		apisecurity.ResourceType_Namespaces: temp,
	}
	authLog.Debug("[Auth][Server] collect namespace access res", zap.Any("res", ret))
	return ret
}

func convertToErrCode(err error) apimodel.Code {
	if errors.Is(err, authcommon.ErrorTokenNotExist) {
		return apimodel.Code_TokenNotExisted
	}
	if errors.Is(err, authcommon.ErrorTokenDisabled) {
		return apimodel.Code_TokenDisabled
	}
	return apimodel.Code_NotAllowedAccess
}
