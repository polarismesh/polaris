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

	"go.uber.org/zap"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
	utils2 "github.com/polarismesh/polaris/config/utils"
)

// CreateConfigFileTemplate create config file template
func (s *Server) CreateConfigFileTemplate(ctx context.Context, template *api.ConfigFileTemplate) *api.ConfigResponse {
	if checkRsp := checkConfigFileTemplateParam(template); checkRsp != nil {
		return checkRsp
	}

	userName := utils.ParseUserName(ctx)
	template.CreateBy = utils.NewStringValue(userName)
	template.ModifyBy = utils.NewStringValue(userName)

	rsp := s.GetConfigFileTemplate(ctx, template.Name.GetValue())
	if rsp.Code.Value == api.ExecuteSuccess {
		return api.NewConfigFileTemplateResponseWithMessage(api.BadRequest, "config file template existed")
	}
	if rsp.Code.Value != api.NotFoundResource {
		return rsp
	}

	templateStoreModel := transferConfigFileTemplateAPIModel2StoreModel(template)

	createdTemplate, err := s.storage.CreateConfigFileTemplate(templateStoreModel)

	if err != nil {
		requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)
		log.Error("[Config][Service] create config file template error.",
			utils.ZapRequestID(requestID),
			zap.Error(err))
		return api.NewConfigFileTemplateResponse(api.StoreLayerException, template)
	}

	return api.NewConfigFileTemplateResponse(api.ExecuteSuccess,
		transferConfigFileTemplateStoreModel2APIModel(createdTemplate))
}

// GetConfigFileTemplate get config file template by name
func (s *Server) GetConfigFileTemplate(ctx context.Context, name string) *api.ConfigResponse {
	if len(name) == 0 {
		return api.NewConfigFileTemplateResponse(api.InvalidConfigFileTemplateName, nil)
	}

	template, err := s.storage.GetConfigFileTemplate(name)
	if err != nil {
		requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)
		log.Error("[Config][Service] get config file template error.",
			utils.ZapRequestID(requestID),
			zap.String("name", name),
			zap.Error(err))
		return api.NewConfigFileTemplateResponse(api.StoreLayerException, nil)
	}

	if template == nil {
		return api.NewConfigFileTemplateResponse(api.NotFoundResource, nil)
	}

	return api.NewConfigFileTemplateResponse(api.ExecuteSuccess,
		transferConfigFileTemplateStoreModel2APIModel(template))
}

// GetAllConfigFileTemplates get all config file templates
func (s *Server) GetAllConfigFileTemplates(ctx context.Context) *api.ConfigBatchQueryResponse {
	templates, err := s.storage.QueryAllConfigFileTemplates()

	if err != nil {
		requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)
		log.Error("[Config][Service]query all config file templates error.",
			utils.ZapRequestID(requestID),
			zap.Error(err))

		return api.NewConfigFileTemplateBatchQueryResponse(api.StoreLayerException, 0, nil)
	}

	if len(templates) == 0 {
		return api.NewConfigFileTemplateBatchQueryResponse(api.ExecuteSuccess, 0, nil)
	}

	var apiTemplates []*api.ConfigFileTemplate
	for _, template := range templates {
		apiTemplates = append(apiTemplates, transferConfigFileTemplateStoreModel2APIModel(template))
	}
	return api.NewConfigFileTemplateBatchQueryResponse(api.ExecuteSuccess, uint32(len(templates)), apiTemplates)
}

func transferConfigFileTemplateStoreModel2APIModel(template *model.ConfigFileTemplate) *api.ConfigFileTemplate {
	return &api.ConfigFileTemplate{
		Id:         utils.NewUInt64Value(template.Id),
		Name:       utils.NewStringValue(template.Name),
		Content:    utils.NewStringValue(template.Content),
		Comment:    utils.NewStringValue(template.Comment),
		Format:     utils.NewStringValue(template.Format),
		CreateBy:   utils.NewStringValue(template.CreateBy),
		CreateTime: utils.NewStringValue(time.Time2String(template.CreateTime)),
		ModifyBy:   utils.NewStringValue(template.ModifyBy),
		ModifyTime: utils.NewStringValue(time.Time2String(template.ModifyTime)),
	}
}

func transferConfigFileTemplateAPIModel2StoreModel(template *api.ConfigFileTemplate) *model.ConfigFileTemplate {
	return &model.ConfigFileTemplate{
		Id:       template.Id.GetValue(),
		Name:     template.Name.GetValue(),
		Content:  template.Content.GetValue(),
		Comment:  template.Comment.GetValue(),
		Format:   template.Format.GetValue(),
		CreateBy: template.CreateBy.GetValue(),
		ModifyBy: template.ModifyBy.GetValue(),
	}
}

func checkConfigFileTemplateParam(template *api.ConfigFileTemplate) *api.ConfigResponse {
	if err := utils2.CheckFileName(template.GetName()); err != nil {
		return api.NewConfigFileTemplateResponse(api.InvalidConfigFileTemplateName, template)
	}
	if err := utils2.CheckContentLength(template.Content.GetValue()); err != nil {
		return api.NewConfigFileTemplateResponse(api.InvalidConfigFileContentLength, template)
	}
	if len(template.Content.GetValue()) == 0 {
		return api.NewConfigFileTemplateResponseWithMessage(api.BadRequest, "content can not be blank.")
	}
	if !utils.IsValidFileFormat(template.Format.GetValue()) {
		return api.NewConfigFileTemplateResponse(api.InvalidConfigFileFormat, template)
	}
	return nil
}
