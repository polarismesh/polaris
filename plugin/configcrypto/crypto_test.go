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
	"testing"

	"github.com/polarismesh/polaris/plugin"
	"github.com/stretchr/testify/assert"
)

func Test_serverCrypto_Name(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "get plugin name",
			want: PluginName,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &serverCrypto{}
			got := s.Name()
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_serverCrypto_Initialize(t *testing.T) {
	tests := []struct {
		name      string
		algorithm string
		cryptor   Cryptor
		conf      *plugin.ConfigEntry
		err       error
	}{
		{
			name:      "initialize plugin",
			algorithm: "AES",
			cryptor:   &aesCryptor{},
			conf: &plugin.ConfigEntry{
				Name: "crypto",
				Option: map[string]interface{}{
					"algorithm": "AES",
				},
			},
		},
		{
			name:      "initialize plugin cryptor not exist",
			algorithm: "DES",
			cryptor:   nil,
			conf: &plugin.ConfigEntry{
				Name: "crypto",
				Option: map[string]interface{}{
					"algorithm": "DES",
				},
			},
			err: fmt.Errorf("cryptor of DES algorithm not exist"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &serverCrypto{}
			err := s.Initialize(tt.conf)
			assert.Equal(t, tt.err, err)
			assert.Equal(t, tt.algorithm, s.algorithm)
			assert.Equal(t, tt.cryptor, s.cryptor)
		})
	}
}

func Test_serverCrypto_GenerateKey(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "generate key",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &serverCrypto{
				algorithm: DefaultAlgorithm,
				cryptor:   &aesCryptor{},
			}
			got, err := s.GenerateKey()
			assert.NotNil(t, got)
			assert.Nil(t, err)
		})
	}
}

func Test_serverCrypto_Encrypt(t *testing.T) {
	type args struct {
		plaintext string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "encrypt and decrypt",
			args: args{
				plaintext: "1234abcd!@#$",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &serverCrypto{
				algorithm: DefaultAlgorithm,
				cryptor:   &aesCryptor{},
			}
			key, err := s.GenerateKey()
			assert.Nil(t, err)
			ciphertext, err := s.Encrypt(tt.args.plaintext, key)
			assert.Nil(t, err)
			plaintext, err := s.Decrypt(ciphertext, key)
			assert.Nil(t, err)
			assert.Equal(t, plaintext, tt.args.plaintext)
		})
	}
}
