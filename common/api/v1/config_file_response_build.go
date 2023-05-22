/*
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
	"github.com/golang/protobuf/ptypes/wrappers"
	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
)

func NewConfigClientResponse(
	code apimodel.Code, configFile *apiconfig.ClientConfigFileInfo) *apiconfig.ConfigClientResponse {
	return &apiconfig.ConfigClientResponse{
		Code:       &wrappers.UInt32Value{Value: uint32(code)},
		Info:       &wrappers.StringValue{Value: code2info[uint32(code)]},
		ConfigFile: configFile,
	}
}

func NewConfigClientSimpleResponse(code apimodel.Code) *apiconfig.ConfigSimpleResponse {
	return &apiconfig.ConfigSimpleResponse{
		Code: &wrappers.UInt32Value{Value: uint32(code)},
		Info: &wrappers.StringValue{Value: code2info[uint32(code)]},
	}
}

func NewConfigClientSimpleResponseWithMessage(code apimodel.Code, message string) *apiconfig.ConfigSimpleResponse {
	return &apiconfig.ConfigSimpleResponse{
		Code: &wrappers.UInt32Value{Value: uint32(code)},
		Info: &wrappers.StringValue{Value: message},
	}
}

func NewConfigClientResponseWithMessage(code apimodel.Code, message string) *apiconfig.ConfigClientResponse {
	return &apiconfig.ConfigClientResponse{
		Code: &wrappers.UInt32Value{Value: uint32(code)},
		Info: &wrappers.StringValue{Value: message},
	}
}

func NewConfigFileGroupResponse(
	code apimodel.Code, configFileGroup *apiconfig.ConfigFileGroup) *apiconfig.ConfigResponse {
	return &apiconfig.ConfigResponse{
		Code:            &wrappers.UInt32Value{Value: uint32(code)},
		Info:            &wrappers.StringValue{Value: code2info[uint32(code)]},
		ConfigFileGroup: configFileGroup,
	}
}

func NewConfigFileGroupResponseWithMessage(code apimodel.Code, message string) *apiconfig.ConfigResponse {
	return &apiconfig.ConfigResponse{
		Code: &wrappers.UInt32Value{Value: uint32(code)},
		Info: &wrappers.StringValue{Value: code2info[uint32(code)] + ":" + message},
	}
}

func NewConfigFileGroupBatchQueryResponse(code apimodel.Code, total uint32,
	configFileGroups []*apiconfig.ConfigFileGroup) *apiconfig.ConfigBatchQueryResponse {
	return &apiconfig.ConfigBatchQueryResponse{
		Code:             &wrappers.UInt32Value{Value: uint32(code)},
		Info:             &wrappers.StringValue{Value: code2info[uint32(code)]},
		Total:            &wrappers.UInt32Value{Value: total},
		ConfigFileGroups: configFileGroups,
	}
}

func NewConfigFileReleaseHistoryBatchQueryResponse(code apimodel.Code, total uint32,
	configFileReleaseHistories []*apiconfig.ConfigFileReleaseHistory) *apiconfig.ConfigBatchQueryResponse {
	return &apiconfig.ConfigBatchQueryResponse{
		Code:                       &wrappers.UInt32Value{Value: uint32(code)},
		Info:                       &wrappers.StringValue{Value: code2info[uint32(code)]},
		Total:                      &wrappers.UInt32Value{Value: total},
		ConfigFileReleaseHistories: configFileReleaseHistories,
	}
}

func NewConfigFileResponse(code apimodel.Code, configFile *apiconfig.ConfigFile) *apiconfig.ConfigResponse {
	return &apiconfig.ConfigResponse{
		Code:       &wrappers.UInt32Value{Value: uint32(code)},
		Info:       &wrappers.StringValue{Value: code2info[uint32(code)]},
		ConfigFile: configFile,
	}
}

func NewConfigFileResponseWithMessage(code apimodel.Code, message string) *apiconfig.ConfigResponse {
	return &apiconfig.ConfigResponse{
		Code: &wrappers.UInt32Value{Value: uint32(code)},
		Info: &wrappers.StringValue{Value: code2info[uint32(code)] + ":" + message},
	}
}

func NewConfigFileBatchQueryResponse(
	code apimodel.Code, total uint32, configFiles []*apiconfig.ConfigFile) *apiconfig.ConfigBatchQueryResponse {
	return &apiconfig.ConfigBatchQueryResponse{
		Code:        &wrappers.UInt32Value{Value: uint32(code)},
		Info:        &wrappers.StringValue{Value: code2info[uint32(code)]},
		Total:       &wrappers.UInt32Value{Value: total},
		ConfigFiles: configFiles,
	}
}

func NewConfigFileTemplateResponse(
	code apimodel.Code, template *apiconfig.ConfigFileTemplate) *apiconfig.ConfigResponse {
	return &apiconfig.ConfigResponse{
		Code:               &wrappers.UInt32Value{Value: uint32(code)},
		Info:               &wrappers.StringValue{Value: code2info[uint32(code)]},
		ConfigFileTemplate: template,
	}
}

func NewConfigFileTemplateResponseWithMessage(code apimodel.Code, message string) *apiconfig.ConfigResponse {
	return &apiconfig.ConfigResponse{
		Code: &wrappers.UInt32Value{Value: uint32(code)},
		Info: &wrappers.StringValue{Value: code2info[uint32(code)] + ":" + message},
	}
}

func NewConfigFileTemplateBatchQueryResponse(code apimodel.Code, total uint32,
	configFileTemplates []*apiconfig.ConfigFileTemplate) *apiconfig.ConfigBatchQueryResponse {
	return &apiconfig.ConfigBatchQueryResponse{
		Code:                &wrappers.UInt32Value{Value: uint32(code)},
		Info:                &wrappers.StringValue{Value: code2info[uint32(code)]},
		Total:               &wrappers.UInt32Value{Value: total},
		ConfigFileTemplates: configFileTemplates,
	}
}

func NewConfigFileReleaseResponse(
	code apimodel.Code, configFileRelease *apiconfig.ConfigFileRelease) *apiconfig.ConfigResponse {
	return &apiconfig.ConfigResponse{
		Code:              &wrappers.UInt32Value{Value: uint32(code)},
		Info:              &wrappers.StringValue{Value: code2info[uint32(code)]},
		ConfigFileRelease: configFileRelease,
	}
}

func NewConfigFileReleaseResponseWithMessage(code apimodel.Code, message string) *apiconfig.ConfigResponse {
	return &apiconfig.ConfigResponse{
		Code: &wrappers.UInt32Value{Value: uint32(code)},
		Info: &wrappers.StringValue{Value: code2info[uint32(code)] + ":" + message},
	}
}

func NewConfigFileReleaseHistoryResponse(
	code apimodel.Code, configFileReleaseHistory *apiconfig.ConfigFileReleaseHistory) *apiconfig.ConfigResponse {
	return &apiconfig.ConfigResponse{
		Code:                     &wrappers.UInt32Value{Value: uint32(code)},
		Info:                     &wrappers.StringValue{Value: code2info[uint32(code)]},
		ConfigFileReleaseHistory: configFileReleaseHistory,
	}
}

func NewConfigFileImportResponse(code apimodel.Code,
	createConfigFiles, skipConfigFiles, overwriteConfigFiles []*apiconfig.ConfigFile) *apiconfig.ConfigImportResponse {
	return &apiconfig.ConfigImportResponse{
		Code:                 &wrappers.UInt32Value{Value: uint32(code)},
		Info:                 &wrappers.StringValue{Value: code2info[uint32(code)]},
		CreateConfigFiles:    createConfigFiles,
		SkipConfigFiles:      skipConfigFiles,
		OverwriteConfigFiles: overwriteConfigFiles,
	}
}

func NewConfigFileImportResponseWithMessage(code apimodel.Code, message string) *apiconfig.ConfigImportResponse {
	return &apiconfig.ConfigImportResponse{
		Code: &wrappers.UInt32Value{Value: uint32(code)},
		Info: &wrappers.StringValue{Value: code2info[uint32(code)] + ":" + message},
	}
}

func NewConfigFileExportResponse(code apimodel.Code, data []byte) *apiconfig.ConfigExportResponse {
	return &apiconfig.ConfigExportResponse{
		Code: &wrappers.UInt32Value{Value: uint32(code)},
		Info: &wrappers.StringValue{Value: code2info[uint32(code)]},
		Data: &wrappers.BytesValue{Value: data},
	}
}
