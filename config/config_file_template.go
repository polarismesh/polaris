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
	if checkRsp := s.checkConfigFileTemplateParam(template); checkRsp != nil {
		return checkRsp
	}

	rsp := s.GetConfigFileTemplate(ctx, template.GetName().GetValue())
	if rsp.Code.Value == api.ExecuteSuccess {
		return api.NewConfigFileTemplateResponseWithMessage(
			apimodel.Code_BadRequest, "config file template existed")
	}
	if rsp.Code.Value != api.NotFoundResource {
		return rsp
	}

	saveData := model.ToConfigFileTemplateStore(template)
	userName := utils.ParseUserName(ctx)
	template.CreateBy = utils.NewStringValue(userName)
	template.ModifyBy = utils.NewStringValue(userName)
	retData, err := s.storage.CreateConfigFileTemplate(saveData)
	if err != nil {
		log.Error("[Config][Service] create config file template error.", utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return api.NewConfigFileTemplateResponse(commonstore.StoreCode2APICode(err), template)
	}

	return api.NewConfigFileTemplateResponse(apimodel.Code_ExecuteSuccess, model.ToConfigFileTemplateAPI(retData))
}

// GetConfigFileTemplate get config file template by name
func (s *Server) GetConfigFileTemplate(ctx context.Context, name string) *apiconfig.ConfigResponse {
	if len(name) == 0 {
		return api.NewConfigFileTemplateResponse(apimodel.Code_InvalidConfigFileTemplateName, nil)
	}

	saveData, err := s.storage.GetConfigFileTemplate(name)
	if err != nil {
		log.Error("[Config][Service] get config file template error.",
			utils.ZapRequestIDByCtx(ctx), zap.String("name", name), zap.Error(err))
		return api.NewConfigFileTemplateResponse(commonstore.StoreCode2APICode(err), nil)
	}
	if saveData == nil {
		return api.NewConfigFileTemplateResponse(apimodel.Code_NotFoundResource, nil)
	}

	return api.NewConfigFileTemplateResponse(apimodel.Code_ExecuteSuccess, model.ToConfigFileTemplateAPI(saveData))
}

// GetAllConfigFileTemplates get all config file templates
func (s *Server) GetAllConfigFileTemplates(ctx context.Context) *apiconfig.ConfigBatchQueryResponse {
	templates, err := s.storage.QueryAllConfigFileTemplates()
	if err != nil {
		log.Error("[Config][Service]query all config file templates error.", utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return api.NewConfigFileTemplateBatchQueryResponse(commonstore.StoreCode2APICode(err), 0, nil)
	}

	if len(templates) == 0 {
		return api.NewConfigFileTemplateBatchQueryResponse(apimodel.Code_ExecuteSuccess, 0, nil)
	}

	var apiTemplates []*apiconfig.ConfigFileTemplate
	for _, template := range templates {
		apiTemplates = append(apiTemplates, model.ToConfigFileTemplateAPI(template))
	}
	return api.NewConfigFileTemplateBatchQueryResponse(apimodel.Code_ExecuteSuccess,
		uint32(len(templates)), apiTemplates)
}

func (s *Server) checkConfigFileTemplateParam(template *apiconfig.ConfigFileTemplate) *apiconfig.ConfigResponse {
	if err := CheckFileName(template.GetName()); err != nil {
		return api.NewConfigFileTemplateResponse(apimodel.Code_InvalidConfigFileTemplateName, template)
	}
	if err := CheckContentLength(template.Content.GetValue(), int(s.cfg.ContentMaxLength)); err != nil {
		return api.NewConfigFileTemplateResponse(apimodel.Code_InvalidConfigFileContentLength, template)
	}
	if len(template.Content.GetValue()) == 0 {
		return api.NewConfigFileTemplateResponseWithMessage(apimodel.Code_BadRequest, "content can not be blank.")
	}
	return nil
}
