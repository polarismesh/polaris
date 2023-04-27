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
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_aesCryptor_GenerateKey(t *testing.T) {
	tests := []struct {
		name   string
		keyLen int
		err    error
	}{
		{
			name:   "genrate aes key",
			keyLen: 16,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &aesCryptor{}
			got, err := c.GenerateKey()
			assert.Nil(t, err)
			assert.Equal(t, tt.keyLen, len(got))
			t.Logf("%x", got)
			t.Log(hex.EncodeToString(got))
		})
	}
}

func Test_aesCryptor_EncryptToBase64(t *testing.T) {
	type args struct {
		plaintext []byte
		key       []byte
	}
	key, _ := hex.DecodeString("777b162a185673cb1b72b467a78221cd")
	tests := []struct {
		name    string
		args    args
		want    string
		wangErr error
	}{
		{
			name: "encrypt to base64",
			args: args{
				plaintext: []byte("polaris"),
				key:       key,
			},
			want: "YnLZ0SYuujFBHjYHAZVN5A==",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &aesCryptor{}
			ciphertext, err := c.EncryptToBase64(tt.args.plaintext, tt.args.key)
			assert.Equal(t, tt.want, ciphertext)
			assert.Equal(t, tt.wangErr, err)
		})
	}
}

func Test_aesCryptor_DecryptFromBase64(t *testing.T) {
	type args struct {
		base64Ciphertext string
		key              []byte
	}
	key, _ := hex.DecodeString("777b162a185673cb1b72b467a78221cd")
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr error
	}{
		{
			name: "decrypt from base64",
			args: args{
				base64Ciphertext: "YnLZ0SYuujFBHjYHAZVN5A==",
				key:              key,
			},
			want: []byte("polaris"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &aesCryptor{}
			got, err := c.DecryptFromBase64(tt.args.base64Ciphertext, tt.args.key)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}
