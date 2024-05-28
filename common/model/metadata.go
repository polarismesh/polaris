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

package model

const (
	// MetaKeyPolarisService service identifier by self registration
	MetaKeyPolarisService = "polaris_service"

	// MetaKeyBuildRevision build revision for server
	MetaKeyBuildRevision = "build-revision"
)

const (
	// MetaKeyConfigFileUseEncrypted 配置加密开关标识，value 为 boolean
	MetaKeyConfigFileUseEncrypted = "internal-encrypted"
	// MetaKeyConfigFileDataKey 加密密钥 tag key
	MetaKeyConfigFileDataKey = "internal-datakey"
	// MetaKeyConfigFileEncryptAlgo 加密算法 tag key
	MetaKeyConfigFileEncryptAlgo = "internal-encryptalgo"
	// MetaKeyConfigFileSyncToKubernetes 配置同步到 kubernetes
	MetaKeyConfigFileSyncToKubernetes = "internal-sync-to-kubernetes"
	// ---- 以下参数仅适配 polaris-controller 生态 ----
	// MetaKeyConfigFileSyncSourceKey 配置同步来源
	MetaKeyConfigFileSyncSourceKey = "internal-sync-source"
	// MetaKeyConfigFileSyncSourceClusterKey 配置同步来源所在集群
	MetaKeyConfigFileSyncSourceClusterKey = "internal-sync-sourcecluster"
	// MetaKey3RdPlatform 第三方平台标签
	MetaKey3RdPlatform = "internal-3rd-platform"
)
