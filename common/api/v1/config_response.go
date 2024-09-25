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
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// ConfigCollect BatchWriteResponse添加Response
func ConfigCollect(batchWriteResponse *apiconfig.ConfigBatchWriteResponse, response *apiconfig.ConfigResponse) {
	// 非200的code，都归为异常
	if CalcCode(response) != 200 {
		if response.GetCode().GetValue() >= batchWriteResponse.GetCode().GetValue() {
			batchWriteResponse.Code.Value = response.GetCode().GetValue()
			batchWriteResponse.Info.Value = code2info[batchWriteResponse.GetCode().GetValue()]
		}
	}
	batchWriteResponse.Responses = append(batchWriteResponse.Responses, response)
}

func NewConfigClientListResponse(code apimodel.Code) *apiconfig.ConfigClientListResponse {
	return &apiconfig.ConfigClientListResponse{
		Code: &wrappers.UInt32Value{Value: uint32(code)},
		Info: &wrappers.StringValue{Value: code2info[uint32(code)]},
	}
}

func NewConfigClientListResponseWithInfo(code apimodel.Code, msg string) *apiconfig.ConfigClientListResponse {
	return &apiconfig.ConfigClientListResponse{
		Code: &wrappers.UInt32Value{Value: uint32(code)},
		Info: &wrappers.StringValue{Value: msg},
	}
}

func NewConfigClientResponse0(code apimodel.Code) *apiconfig.ConfigClientResponse {
	return &apiconfig.ConfigClientResponse{
		Code: &wrappers.UInt32Value{Value: uint32(code)},
		Info: &wrappers.StringValue{Value: code2info[uint32(code)]},
	}
}

func NewConfigClientResponse(code apimodel.Code, configFile *apiconfig.ClientConfigFileInfo) *apiconfig.ConfigClientResponse {
	return &apiconfig.ConfigClientResponse{
		Code:       &wrappers.UInt32Value{Value: uint32(code)},
		Info:       &wrappers.StringValue{Value: code2info[uint32(code)]},
		ConfigFile: configFile,
	}
}

func NewConfigClientResponseFromConfigResponse(response *apiconfig.ConfigResponse) *apiconfig.ConfigClientResponse {
	return &apiconfig.ConfigClientResponse{
		Code:       response.Code,
		Info:       response.Info,
		ConfigFile: nil,
	}
}

func NewConfigClientResponseWithInfo(code apimodel.Code, message string) *apiconfig.ConfigClientResponse {
	return &apiconfig.ConfigClientResponse{
		Code: &wrappers.UInt32Value{Value: uint32(code)},
		Info: &wrappers.StringValue{Value: message},
	}
}

func NewConfigResponse(code apimodel.Code) *apiconfig.ConfigResponse {
	return &apiconfig.ConfigResponse{
		Code: &wrappers.UInt32Value{Value: uint32(code)},
		Info: &wrappers.StringValue{Value: code2info[uint32(code)]},
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

func NewConfigBatchQueryResponse(code apimodel.Code) *apiconfig.ConfigBatchQueryResponse {
	return &apiconfig.ConfigBatchQueryResponse{
		Code: &wrappers.UInt32Value{Value: uint32(code)},
		Info: &wrappers.StringValue{Value: code2info[uint32(code)]},
	}
}

func NewConfigBatchQueryResponseWithInfo(code apimodel.Code, info string) *apiconfig.ConfigBatchQueryResponse {
	return &apiconfig.ConfigBatchQueryResponse{
		Code: &wrappers.UInt32Value{Value: uint32(code)},
		Info: &wrappers.StringValue{Value: info},
	}
}

func NewConfigBatchWriteResponse(code apimodel.Code) *apiconfig.ConfigBatchWriteResponse {
	return &apiconfig.ConfigBatchWriteResponse{
		Code: &wrappers.UInt32Value{Value: uint32(code)},
		Info: &wrappers.StringValue{Value: code2info[uint32(code)]},
	}
}

func NewConfigBatchWriteResponseWithInfo(code apimodel.Code, info string) *apiconfig.ConfigBatchWriteResponse {
	return &apiconfig.ConfigBatchWriteResponse{
		Code: &wrappers.UInt32Value{Value: uint32(code)},
		Info: &wrappers.StringValue{Value: info},
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

func NewConfigResponseWithInfo(code apimodel.Code, message string) *apiconfig.ConfigResponse {
	return &apiconfig.ConfigResponse{
		Code: &wrappers.UInt32Value{Value: uint32(code)},
		Info: &wrappers.StringValue{Value: message},
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

func NewConfigFileBatchQueryResponseWithMessage(
	code apimodel.Code, message string) *apiconfig.ConfigBatchQueryResponse {
	return &apiconfig.ConfigBatchQueryResponse{
		Code: &wrappers.UInt32Value{Value: uint32(code)},
		Info: &wrappers.StringValue{Value: code2info[uint32(code)]},
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

func NewConfigFileExportResponseWithMessage(code apimodel.Code, message string) *apiconfig.ConfigExportResponse {
	return &apiconfig.ConfigExportResponse{
		Code: &wrappers.UInt32Value{Value: uint32(code)},
		Info: &wrappers.StringValue{Value: code2info[uint32(code)] + ":" + message},
	}
}

func NewConfigEncryptAlgorithmResponse(code apimodel.Code,
	algorithms []*wrapperspb.StringValue) *apiconfig.ConfigEncryptAlgorithmResponse {
	resp := &apiconfig.ConfigEncryptAlgorithmResponse{
		Code:       &wrappers.UInt32Value{Value: uint32(code)},
		Info:       &wrappers.StringValue{Value: code2info[uint32(code)]},
		Algorithms: algorithms,
	}
	return resp
}
