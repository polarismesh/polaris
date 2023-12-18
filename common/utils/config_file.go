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

package utils

import "strings"

const (
	// ReleaseTypeNormal 发布类型，全量发布
	ReleaseTypeNormal = "normal"
	// ReleaseTypeGray 灰度发布
	ReleaseTypeGray = "betaing"
	// ReleaseTypeCancelGray 取消灰度发布
	ReleaseTypeCancelGray = "cancel-gray"
	// ReleaseTypeDelete 发布类型，删除配置发布
	ReleaseTypeDelete = "delete"
	// ReleaseTypeRollback 发布类型 回滚
	ReleaseTypeRollback = "rollback"
	// ReleaseTypeClean 发布类型，清空配置发布
	ReleaseTypeClean = "clean"

	// ReleaseStatusSuccess 发布成功状态
	ReleaseStatusSuccess = "success"
	// ReleaseStatusFail 发布失败状态
	ReleaseStatusFail = "failure"
	// ReleaseStatusToRelease 待发布状态
	ReleaseStatusToRelease = "to-be-released"

	// 文件格式
	FileFormatText       = "text"
	FileFormatYaml       = "yaml"
	FileFormatXml        = "xml"
	FileFormatJson       = "json"
	FileFormatHtml       = "html"
	FileFormatProperties = "properties"

	FileIdSeparator = "+"

	// MaxRequestBodySize 导入配置文件请求体最大 4M
	MaxRequestBodySize = 4 * 1024 * 1024
	// ConfigFileFormKey 配置文件表单键
	ConfigFileFormKey = "config"
	// ConfigFileMetaFileName 配置文件元数据文件名
	ConfigFileMetaFileName = "META"
	// ConfigFileImportConflictSkip 导入配置文件发生冲突跳过
	ConfigFileImportConflictSkip = "skip"
	// ConfigFileImportConflictOverwrite 导入配置文件发生冲突覆盖原配置文件
	ConfigFileImportConflictOverwrite = "overwrite"
)

// GenFileId 生成文件 Id
func GenFileId(namespace, group, fileName string) string {
	return namespace + FileIdSeparator + group + FileIdSeparator + fileName
}

// ParseFileId 解析文件 Id
func ParseFileId(fileId string) (namespace, group, fileName string) {
	fileInfo := strings.Split(fileId, FileIdSeparator)
	return fileInfo[0], fileInfo[1], fileInfo[2]
}

// ConfigFileMeta 导入配置文件ZIP包中的元数据结构
type ConfigFileMeta struct {
	Tags    map[string]string `json:"tags"`
	Comment string            `json:"comment"`
}
