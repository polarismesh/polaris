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
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
)

// RSAKey RSA key pair
type RSAKey struct {
	PrivateKey string
	PublicKey  string
}

// GenerateKey generate RSA key pair
func GenerateRSAKey() (*RSAKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, err
	}
	rsaKey := &RSAKey{
		PrivateKey: base64.StdEncoding.EncodeToString(x509.MarshalPKCS1PrivateKey(privateKey)),
		PublicKey:  base64.StdEncoding.EncodeToString(x509.MarshalPKCS1PublicKey(&privateKey.PublicKey)),
	}
	return rsaKey, nil
}

// Encrypt RSA encrypt plaintext using public key
func Encrypt(plaintext, publicKey []byte) ([]byte, error) {
	pub, err := x509.ParsePKCS1PublicKey(publicKey)
	if err != nil {
		return nil, err
	}
	totalLen := len(plaintext)
	segLen := pub.Size() - 11
	start := 0
	buffer := bytes.Buffer{}
	for start < totalLen {
		end := start + segLen
		if end > totalLen {
			end = totalLen
		}
		seg, err := rsa.EncryptPKCS1v15(rand.Reader, pub, plaintext[start:end])
		if err != nil {
			return nil, err
		}
		buffer.Write(seg)
		start = end
	}
	return buffer.Bytes(), nil
}

// Decrypt RSA decrypt ciphertext using private key
func Decrypt(ciphertext, privateKey []byte) ([]byte, error) {
	priv, err := x509.ParsePKCS1PrivateKey(privateKey)
	if err != nil {
		return nil, err
	}
	keySize := priv.Size()
	totalLen := len(ciphertext)
	start := 0
	buffer := bytes.Buffer{}
	for start < totalLen {
		end := start + keySize
		if end > totalLen {
			end = totalLen
		}
		seg, err := rsa.DecryptPKCS1v15(rand.Reader, priv, ciphertext[start:end])
		if err != nil {
			return nil, err
		}
		buffer.Write(seg)
		start = end
	}
	return buffer.Bytes(), nil
}

// EncryptToBase64 RSA encrypt plaintext and base64 encode ciphertext
func EncryptToBase64(plaintext []byte, base64PublicKey string) (string, error) {
	pub, err := base64.StdEncoding.DecodeString(base64PublicKey)
	if err != nil {
		return "", err
	}
	ciphertext, err := Encrypt(plaintext, pub)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptFromBase64 base64 decode ciphertext and RSA decrypt
func DecryptFromBase64(base64Ciphertext, base64PrivateKey string) ([]byte, error) {
	priv, err := base64.StdEncoding.DecodeString(base64PrivateKey)
	if err != nil {
		return nil, err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(base64Ciphertext)
	if err != nil {
		return nil, err
	}
	return Decrypt(ciphertext, priv)
}
