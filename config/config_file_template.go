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

	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"go.uber.org/zap"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	commonstore "github.com/polarismesh/polaris/common/store"
	"github.com/polarismesh/polaris/common/utils"
)

// CreateConfigFileTemplate create config file template
func (s *Server) CreateConfigFileTemplate(
	ctx context.Context, template *apiconfig.ConfigFileTemplate) *apiconfig.ConfigResponse {
	name := template.GetName().GetValue()

	saveData, err := s.storage.GetConfigFileTemplate(name)
	if err != nil {
		log.Error("[Config][Service] get config file template error.",
			utils.RequestID(ctx), zap.String("name", name), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	if saveData != nil {
		return api.NewConfigResponse(apimodel.Code_ExistedResource)
	}

	saveData = model.ToConfigFileTemplateStore(template)
	userName := utils.ParseUserName(ctx)
	template.CreateBy = utils.NewStringValue(userName)
	template.ModifyBy = utils.NewStringValue(userName)
	if _, err := s.storage.CreateConfigFileTemplate(saveData); err != nil {
		log.Error("[Config][Service] create config file template error.", utils.RequestID(ctx), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}

	return api.NewConfigResponse(apimodel.Code_ExecuteSuccess)
}

// GetConfigFileTemplate get config file template by name
func (s *Server) GetConfigFileTemplate(ctx context.Context, name string) *apiconfig.ConfigResponse {
	if len(name) == 0 {
		return api.NewConfigResponse(apimodel.Code_InvalidConfigFileTemplateName)
	}

	saveData, err := s.storage.GetConfigFileTemplate(name)
	if err != nil {
		log.Error("[Config][Service] get config file template error.",
			utils.RequestID(ctx), zap.String("name", name), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	if saveData == nil {
		return api.NewConfigResponse(apimodel.Code_NotFoundResource)
	}
	out := api.NewConfigResponse(apimodel.Code_ExecuteSuccess)
	out.ConfigFileTemplate = model.ToConfigFileTemplateAPI(saveData)
	return out
}

// GetAllConfigFileTemplates get all config file templates
func (s *Server) GetAllConfigFileTemplates(ctx context.Context) *apiconfig.ConfigBatchQueryResponse {
	templates, err := s.storage.QueryAllConfigFileTemplates()
	if err != nil {
		log.Error("[Config][Service]query all config file templates error.", utils.RequestID(ctx), zap.Error(err))
		return api.NewConfigBatchQueryResponse(commonstore.StoreCode2APICode(err))
	}

	var apiTemplates []*apiconfig.ConfigFileTemplate
	for _, template := range templates {
		apiTemplates = append(apiTemplates, model.ToConfigFileTemplateAPI(template))
	}
	return api.NewConfigFileTemplateBatchQueryResponse(apimodel.Code_ExecuteSuccess,
		uint32(len(templates)), apiTemplates)
}
