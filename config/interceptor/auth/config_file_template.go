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

package config_auth

import (
	"context"

	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
)

// GetAllConfigFileTemplates get all config file templates
func (s *ServerAuthability) GetAllConfigFileTemplates(ctx context.Context) *apiconfig.ConfigBatchQueryResponse {
	authCtx := s.collectConfigFileTemplateAuthContext(ctx,
		[]*apiconfig.ConfigFileTemplate{}, auth.Read, auth.DescribeAllConfigFileTemplates)
	if _, err := s.policyMgr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewConfigFileBatchQueryResponseWithMessage(auth.ConvertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return s.nextServer.GetAllConfigFileTemplates(ctx)
}

// GetConfigFileTemplate get config file template
func (s *ServerAuthability) GetConfigFileTemplate(ctx context.Context, name string) *apiconfig.ConfigResponse {
	authCtx := s.collectConfigFileTemplateAuthContext(ctx,
		[]*apiconfig.ConfigFileTemplate{}, auth.Read, auth.DescribeConfigFileTemplate)
	if _, err := s.policyMgr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewConfigResponseWithInfo(auth.ConvertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return s.nextServer.GetConfigFileTemplate(ctx, name)
}

// CreateConfigFileTemplate create config file template
func (s *ServerAuthability) CreateConfigFileTemplate(ctx context.Context,
	template *apiconfig.ConfigFileTemplate) *apiconfig.ConfigResponse {

	authCtx := s.collectConfigFileTemplateAuthContext(ctx,
		[]*apiconfig.ConfigFileTemplate{template}, auth.Create, auth.CreateConfigFileTemplate)
	if _, err := s.policyMgr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewConfigResponseWithInfo(auth.ConvertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return s.nextServer.CreateConfigFileTemplate(ctx, template)
}
