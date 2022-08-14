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

package defaultauth

import "errors"

// AuthOption 鉴权的配置信息
var AuthOption = DefaultAuthConfig()

// AuthConfig 鉴权配置
type AuthConfig struct {
	// ConsoleOpen 控制台是否开启鉴权
	ConsoleOpen bool `json:"consoleOpen" xml:"consoleOpen"`
	// ClientOpen 是否开启客户端接口鉴权
	ClientOpen bool `json:"clientOpen" xml:"clientOpen"`
	// Salt 相关密码、token加密的salt
	Salt string `json:"salt" xml:"salt"`
	// Strict 是否启用鉴权的严格模式，即对于没有任何鉴权策略的资源，也必须带上正确的token才能操作, 默认关闭
	Strict bool `json:"strict"`
}

// Verify 检查配置是否合法
func (cfg *AuthConfig) Verify() error {
	k := len(cfg.Salt)
	switch k {
	case 16, 24, 32:
		break
	default:
		return errors.New("[Auth][Config] salt len must 16 | 24 | 32")
	}

	return nil
}

// DefaultAuthConfig 返回一个默认的鉴权配置
func DefaultAuthConfig() *AuthConfig {
	return &AuthConfig{
		// 针对控制台接口，默认开启鉴权操作
		ConsoleOpen: true,
		// 针对客户端接口，默认不开启鉴权操作
		ClientOpen: false,
		Salt:       "polarismesh@2021",
		// 这里默认开启强 Token 检查模式
		Strict: true,
	}
}
