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

package configcrypto

import (
	"fmt"

	"github.com/polarismesh/polaris/plugin"
)

const (
	PluginName       = "crypto"
	DefaultAlgorithm = AES
)

func init() {
	plugin.RegisterPlugin(PluginName, &serverCrypto{})
}

type serverCrypto struct {
	algorithm string
	cryptor   Cryptor
}

// Name 插件名词
func (s *serverCrypto) Name() string {
	return PluginName
}

// Initialize 初始化插件
func (s *serverCrypto) Initialize(conf *plugin.ConfigEntry) error {
	algorithm, ok := conf.Option["algorithm"]
	if !ok {
		s.algorithm = DefaultAlgorithm
	} else {
		s.algorithm = algorithm.(string)
	}
	if cryptor, ok := CryptorSet[s.algorithm]; ok {
		s.cryptor = cryptor
	} else {
		return fmt.Errorf("cryptor of %s algorithm not exist", s.algorithm)
	}
	return nil
}

// Destroy 销毁插件
func (s *serverCrypto) Destroy() error {
	return nil
}

// GenerateKey 生成密钥
func (s *serverCrypto) GenerateKey() ([]byte, error) {
	return s.cryptor.GenerateKey()
}

// Encrypt 加密
func (s *serverCrypto) Encrypt(plaintext string, key []byte) (ciphertext string, err error) {
	ciphertext, err = s.cryptor.EncryptToBase64([]byte(plaintext), key)
	if err != nil {
		return
	}
	return
}

// Decrypt 解密
func (s *serverCrypto) Decrypt(ciphertext string, key []byte) (string, error) {
	plaintext, err := s.cryptor.DecryptFromBase64(ciphertext, key)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
