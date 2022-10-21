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

package version

var (
	// Version version
	Version string
	// BuildDate build date
	BuildDate string
)

const defaultVersion = "v0.1.0"

// Get 获取版本号
func Get() string {
	if Version == "" {
		return defaultVersion
	}

	return Version
}

// GetRevision 获取完整版本号信息，包括时间戳的
func GetRevision() string {
	if Version == "" || BuildDate == "" {
		return defaultVersion
	}

	return Version + "." + BuildDate
}
