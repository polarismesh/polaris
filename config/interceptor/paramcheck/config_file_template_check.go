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

package paramcheck

import (
	"context"

	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"

	api "github.com/polarismesh/polaris/common/api/v1"
)

// GetAllConfigFileTemplates get all config file templates
func (s *Server) GetAllConfigFileTemplates(ctx context.Context) *apiconfig.ConfigBatchQueryResponse {

	return s.nextServer.GetAllConfigFileTemplates(ctx)
}

// GetConfigFileTemplate get config file template
func (s *Server) GetConfigFileTemplate(ctx context.Context, name string) *apiconfig.ConfigResponse {

	return s.nextServer.GetConfigFileTemplate(ctx, name)
}

// CreateConfigFileTemplate create config file template
func (s *Server) CreateConfigFileTemplate(ctx context.Context,
	template *apiconfig.ConfigFileTemplate) *apiconfig.ConfigResponse {
	if checkRsp := s.checkConfigFileTemplateParam(template); checkRsp != nil {
		return checkRsp
	}
	return s.nextServer.CreateConfigFileTemplate(ctx, template)
}

func (s *Server) checkConfigFileTemplateParam(template *apiconfig.ConfigFileTemplate) *apiconfig.ConfigResponse {
	if err := CheckFileName(template.GetName()); err != nil {
		return api.NewConfigResponse(apimodel.Code_InvalidConfigFileTemplateName)
	}
	if err := CheckContentLength(template.Content.GetValue(), int(s.cfg.ContentMaxLength)); err != nil {
		return api.NewConfigResponse(apimodel.Code_InvalidConfigFileContentLength)
	}
	if len(template.Content.GetValue()) == 0 {
		return api.NewConfigFileTemplateResponseWithMessage(apimodel.Code_BadRequest, "content can not be blank.")
	}
	return nil
}
