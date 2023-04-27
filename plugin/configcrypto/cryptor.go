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

import "fmt"

// CryptorSet cryptor set
var CryptorSet = make(map[string]Cryptor)

// RegisterCryptor register cryptor
func RegisterCryptor(name string, cryptor Cryptor) {
	if _, exist := CryptorSet[name]; exist {
		panic(fmt.Sprintf("existed cryptor: name=%v", name))
	}
	CryptorSet[name] = cryptor
}

// Cryptor cryptor interface
type Cryptor interface {
	GenerateKey() ([]byte, error)
	Encrypt(plaintext []byte, key []byte) ([]byte, error)
	Decrypt(ciphertext []byte, key []byte) ([]byte, error)
	EncryptToBase64(plaintext, key []byte) (string, error)
	DecryptFromBase64(base64Ciphertext string, key []byte) ([]byte, error)
}
