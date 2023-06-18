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

package aes

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"

	"github.com/polarismesh/polaris/plugin"
)

const (
	// PluginName plugin name
	PluginName = "AES"
)

func init() {
	plugin.RegisterPlugin(PluginName, &AESCrypto{})
}

// AESCrypto AES crypto
type AESCrypto struct {
}

// Name 返回插件名字
func (h *AESCrypto) Name() string {
	return PluginName
}

// Destroy 销毁插件
func (h *AESCrypto) Destroy() error {
	return nil
}

// Initialize 插件初始化
func (h *AESCrypto) Initialize(c *plugin.ConfigEntry) error {
	return nil
}

// GenerateKey generate key
func (c *AESCrypto) GenerateKey() ([]byte, error) {
	key := make([]byte, 16)
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// Encrypt AES encrypt plaintext and base64 encode ciphertext
func (c *AESCrypto) Encrypt(plaintext string, key []byte) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	ciphertext, err := c.doEncrypt([]byte(plaintext), key)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt base64 decode ciphertext and AES decrypt
func (c *AESCrypto) Decrypt(ciphertext string, key []byte) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	ciphertextBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	plaintext, err := c.doDecrypt(ciphertextBytes, key)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// Encrypt AES encryption
func (c *AESCrypto) doEncrypt(plaintext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	paddingData := pkcs7Padding(plaintext, blockSize)
	ciphertext := make([]byte, len(paddingData))
	blockMode := cipher.NewCBCEncrypter(block, key[:blockSize])
	blockMode.CryptBlocks(ciphertext, paddingData)
	return ciphertext, nil
}

// Decrypt AES decryption
func (c *AESCrypto) doDecrypt(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	paddingPlaintext := make([]byte, len(ciphertext))
	blockMode.CryptBlocks(paddingPlaintext, ciphertext)
	plaintext, err := pkcs7UnPadding(paddingPlaintext)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

func pkcs7Padding(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

func pkcs7UnPadding(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("invalid encryption data")
	}
	unPadding := int(data[length-1])
	if unPadding > length {
		return nil, errors.New("invalid encryption data")
	}
	return data[:(length - unPadding)], nil
}
