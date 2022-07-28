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

import "github.com/golang/protobuf/ptypes/wrappers"

func NewConfigClientResponse(code uint32, configFile *ClientConfigFileInfo) *ConfigClientResponse {
	return &ConfigClientResponse{
		Code:       &wrappers.UInt32Value{Value: code},
		Info:       &wrappers.StringValue{Value: code2info[code]},
		ConfigFile: configFile,
	}
}

func NewConfigClientResponseWithMessage(code uint32, message string) *ConfigClientResponse {
	return &ConfigClientResponse{
		Code: &wrappers.UInt32Value{Value: code},
		Info: &wrappers.StringValue{Value: message},
	}
}

func NewConfigFileGroupResponse(code uint32, configFileGroup *ConfigFileGroup) *ConfigResponse {
	return &ConfigResponse{
		Code:            &wrappers.UInt32Value{Value: code},
		Info:            &wrappers.StringValue{Value: code2info[code]},
		ConfigFileGroup: configFileGroup,
	}
}

func NewConfigFileGroupResponseWithMessage(code uint32, message string) *ConfigResponse {
	return &ConfigResponse{
		Code: &wrappers.UInt32Value{Value: code},
		Info: &wrappers.StringValue{Value: code2info[code] + ":" + message},
	}
}

func NewConfigFileGroupBatchQueryResponse(code uint32, total uint32,
	configFileGroups []*ConfigFileGroup) *ConfigBatchQueryResponse {
	return &ConfigBatchQueryResponse{
		Code:             &wrappers.UInt32Value{Value: code},
		Info:             &wrappers.StringValue{Value: code2info[code]},
		Total:            &wrappers.UInt32Value{Value: total},
		ConfigFileGroups: configFileGroups,
	}
}

func NewConfigFileReleaseHistoryBatchQueryResponse(code uint32, total uint32,
	configFileReleaseHistories []*ConfigFileReleaseHistory) *ConfigBatchQueryResponse {
	return &ConfigBatchQueryResponse{
		Code:                       &wrappers.UInt32Value{Value: code},
		Info:                       &wrappers.StringValue{Value: code2info[code]},
		Total:                      &wrappers.UInt32Value{Value: total},
		ConfigFileReleaseHistories: configFileReleaseHistories,
	}
}

func NewConfigFileResponse(code uint32, configFile *ConfigFile) *ConfigResponse {
	return &ConfigResponse{
		Code:       &wrappers.UInt32Value{Value: code},
		Info:       &wrappers.StringValue{Value: code2info[code]},
		ConfigFile: configFile,
	}
}

func NewConfigFileResponseWithMessage(code uint32, message string) *ConfigResponse {
	return &ConfigResponse{
		Code: &wrappers.UInt32Value{Value: code},
		Info: &wrappers.StringValue{Value: code2info[code] + ":" + message},
	}
}

func NewConfigFileBatchQueryResponse(code uint32, total uint32, configFiles []*ConfigFile) *ConfigBatchQueryResponse {
	return &ConfigBatchQueryResponse{
		Code:        &wrappers.UInt32Value{Value: code},
		Info:        &wrappers.StringValue{Value: code2info[code]},
		Total:       &wrappers.UInt32Value{Value: total},
		ConfigFiles: configFiles,
	}
}

func NewConfigFileTemplateResponse(code uint32, template *ConfigFileTemplate) *ConfigResponse {
	return &ConfigResponse{
		Code:               &wrappers.UInt32Value{Value: code},
		Info:               &wrappers.StringValue{Value: code2info[code]},
		ConfigFileTemplate: template,
	}
}

func NewConfigFileTemplateResponseWithMessage(code uint32, message string) *ConfigResponse {
	return &ConfigResponse{
		Code: &wrappers.UInt32Value{Value: code},
		Info: &wrappers.StringValue{Value: code2info[code] + ":" + message},
	}
}

func NewConfigFileTemplateBatchQueryResponse(code uint32, total uint32, configFileTemplates []*ConfigFileTemplate) *ConfigBatchQueryResponse {
	return &ConfigBatchQueryResponse{
		Code:                &wrappers.UInt32Value{Value: code},
		Info:                &wrappers.StringValue{Value: code2info[code]},
		Total:               &wrappers.UInt32Value{Value: total},
		ConfigFileTemplates: configFileTemplates,
	}
}

func NewConfigFileReleaseResponse(code uint32, configFileRelease *ConfigFileRelease) *ConfigResponse {
	return &ConfigResponse{
		Code:              &wrappers.UInt32Value{Value: code},
		Info:              &wrappers.StringValue{Value: code2info[code]},
		ConfigFileRelease: configFileRelease,
	}
}

func NewConfigFileReleaseResponseWithMessage(code uint32, message string) *ConfigResponse {
	return &ConfigResponse{
		Code: &wrappers.UInt32Value{Value: code},
		Info: &wrappers.StringValue{Value: code2info[code] + ":" + message},
	}
}

func NewConfigFileReleaseHistoryResponse(code uint32, configFileReleaseHistory *ConfigFileReleaseHistory) *ConfigResponse {
	return &ConfigResponse{
		Code:                     &wrappers.UInt32Value{Value: code},
		Info:                     &wrappers.StringValue{Value: code2info[code]},
		ConfigFileReleaseHistory: configFileReleaseHistory,
	}
}
