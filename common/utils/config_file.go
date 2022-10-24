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
	// ReleaseTypeDelete 发布类型，删除配置文件
	ReleaseTypeDelete = "delete"

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
)

// IsValidFileFormat 判断文件格式是否合法
func IsValidFileFormat(format string) bool {
	return format == FileFormatText || format == FileFormatYaml || format == FileFormatXml ||
		format == FileFormatJson || format == FileFormatHtml || format == FileFormatProperties
}

// GenFileId 生成文件 Id
func GenFileId(namespace, group, fileName string) string {
	return namespace + FileIdSeparator + group + FileIdSeparator + fileName
}

// ParseFileId 解析文件 Id
func ParseFileId(fileId string) (namespace, group, fileName string) {
	fileInfo := strings.Split(fileId, FileIdSeparator)
	return fileInfo[0], fileInfo[1], fileInfo[2]
}
