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

package config

import (
	"context"
	"github.com/polarismesh/polaris-server/auth"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	commonlog "github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
	"go.uber.org/zap"
	"strconv"
)

var _ ConfigCenterServer = (*serverAuthability)(nil)

// Server 配置中心核心服务
type serverAuthability struct {
	targetServer *Server
	authSvr      auth.AuthServer
	authChecker  auth.AuthChecker
}

func newServerAuthAbility(targetServer *Server, authSvr auth.AuthServer) ConfigCenterServer {
	proxy := &serverAuthability{
		targetServer: targetServer,
		authSvr:      authSvr,
		authChecker:  authSvr.GetAuthChecker(),
	}
	targetServer.SetResourceHooks(proxy)
	return proxy
}

func (s *serverAuthability) collectBaseTokenInfo(ctx context.Context, req []*api.ConfigFileGroup,
	op model.ResourceOperation, methodName string, rType model.Resource) *model.AcquireContext {
	switch rType {
	case model.RConfigFile:
		return model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithToken(utils.ParseAuthToken(ctx)),
			model.WithModule(model.ConfigModule),
		)
	case model.RConfigGroup:
		return model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithToken(utils.ParseAuthToken(ctx)),
			model.WithModule(model.ConfigModule),
			model.WithOperation(op),
			model.WithMethod(methodName),
			model.WithAccessResources(s.queryConfigGroupResource(req)),
		)
	default:
		return nil
	}
}

func (s *serverAuthability) queryConfigGroupResource(req []*api.ConfigFileGroup) map[api.ResourceType][]model.ResourceEntry {
	names := utils.NewStringSet()
	namespace := req[0].Namespace.GetValue()
	for index := range req {
		names.Add(req[index].Name.GetValue())
	}

	configFileGroups, err := s.targetServer.storage.FindConfigFileGroups(namespace, names.ToSlice())
	if err != nil {
		return nil
	}

	temp := make([]model.ResourceEntry, 0, len(configFileGroups))

	for index := range configFileGroups {
		group := configFileGroups[index]
		temp = append(temp, model.ResourceEntry{
			ID:    strconv.FormatUint(group.Id, 10),
			Owner: group.CreateBy, // todo config_file_group owner
		})
	}

	ret := map[api.ResourceType][]model.ResourceEntry{
		api.ResourceType_Namespaces: temp,
	}
	commonlog.AuthScope().Debug("[Auth][Server] collect config_file_group access res", zap.Any("res", ret))
	return ret
}
