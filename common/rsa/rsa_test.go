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

package rsa

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateRSAKey(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "generate rsa key",
			err:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateRSAKey()
			t.Logf("PrivateKey: %s", got.PrivateKey)
			t.Logf("PublicKey: %s", got.PublicKey)
			assert.Nil(t, err)
		})
	}
}

func TestEncryptToBase64(t *testing.T) {
	type args struct {
		plaintext []byte
	}
	tests := []struct {
		name string
		args args
		want string
		err  error
	}{
		{
			name: "encrypt to base64",
			args: args{
				plaintext: []byte("1234abcd!@#$"),
			},
		},
		{
			name: "encrypt lang text to base64",
			args: args{
				plaintext: []byte(`aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
				aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rasKey, err := GenerateRSAKey()
			assert.Nil(t, err)
			ciphertext, err := EncryptToBase64(tt.args.plaintext, rasKey.PublicKey)
			assert.Nil(t, err)
			plaintext, err := DecryptFromBase64(ciphertext, rasKey.PrivateKey)
			assert.Nil(t, err)
			assert.Equal(t, plaintext, tt.args.plaintext)
		})
	}
}
