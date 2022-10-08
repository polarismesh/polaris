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

package secure

import (
	"github.com/mitchellh/mapstructure"

	"github.com/polarismesh/polaris/common/log"
)

// TLSConfig tls 相关配置
type TLSConfig struct {
	// CertFile 证书
	CertFile string `mapstructure:"certFile"`
	// KeyFile 密钥
	KeyFile string `mapstructure:"keyFile"`
	// TrustedCAFile CA 证书
	TrustedCAFile string `mapstructure:"trustedCAFile"`
	// ServerName 客户端发送的 Server Name Indication 扩展的值
	ServerName string `mapstructure:"serverName"`

	// InsecureSkipVerify tls 的一个配置
	// 客户端是否验证证书和服务器主机名
	InsecureSkipVerify bool `mapstructure:"insecureSkipTlsVerify"`
}

// ParseTLSConfig 解析 tls 配置
func ParseTLSConfig(raw map[interface{}]interface{}) (*TLSConfig, error) {
	if raw == nil {
		return nil, nil
	}

	tlsConfig := &TLSConfig{}
	decodeConfig := &mapstructure.DecoderConfig{
		DecodeHook: mapstructure.StringToTimeDurationHookFunc(),
		Result:     tlsConfig,
	}
	decoder, err := mapstructure.NewDecoder(decodeConfig)
	if err != nil {
		log.Errorf("tls config new decoder err: %s", err.Error())
		return nil, err
	}

	err = decoder.Decode(raw)
	if err != nil {
		log.Errorf("parse tls config(%+v) err: %s", raw, err.Error())
		return nil, err
	}

	return tlsConfig, nil
}
