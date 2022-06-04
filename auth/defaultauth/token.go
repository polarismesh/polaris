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

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/polarismesh/polaris-server/common/model"
)

// OperatorInfo 根据 token 解析出来的具体额外信息
type OperatorInfo struct {

	// Origin 原始 token 字符串
	Origin string

	// OperatorID 当前 token 绑定的 用户/用户组 ID
	OperatorID string

	// OwnerID 当前用户/用户组对应的 owner
	OwnerID string

	// Role 如果当前是 user token 的话，该值才能有信息
	Role model.UserRoleType

	// IsUserToken 当前 token 是否是 user 的 token
	IsUserToken bool

	// Disable 标识用户 token 是否被禁用
	Disable bool

	// 是否属于匿名操作者
	Anonymous bool
}

func newAnonymous() OperatorInfo {
	return OperatorInfo{
		Origin:     "",
		OwnerID:    "",
		OperatorID: "__anonymous__",
		Anonymous:  true,
	}
}

// IsEmptyOperator token 是否是一个空类型
func IsEmptyOperator(t OperatorInfo) bool {
	return t.Origin == "" || t.Anonymous
}

// IsSubAccount 当前 token 对应的账户类型
func IsSubAccount(t OperatorInfo) bool {
	return t.Role == model.SubAccountUserRole
}

func (t *OperatorInfo) String() string {
	return fmt.Sprintf("operator-id=%s, owner=%s, role=%d, is-user=%v, disable=%v",
		t.OperatorID, t.OwnerID, t.Role, t.IsUserToken, t.Disable)
}

const (
	// TokenPattern token 的格式 随机字符串::[uid/xxx | groupid/xxx]
	TokenPattern string = "%s::%s"
	// TokenSplit token 的分隔符
	TokenSplit string = "::"
)

// createUserToken Create a user token
func createUserToken(uid string) (string, error) {
	return createToken(uid, "")
}

// createGroupToken Create user group token
func createGroupToken(gid string) (string, error) {
	return createToken("", gid)
}

// createToken Determine what type of Token created according to the incoming parameters
func createToken(uid, gid string) (string, error) {
	if uid == "" && gid == "" {
		return "", errors.New("uid and groupid not be empty at the same time")
	}

	var val string
	if uid == "" {
		val = fmt.Sprintf("%s/%s", model.TokenForUserGroup, gid)
	} else {
		val = fmt.Sprintf("%s/%s", model.TokenForUser, uid)
	}

	token := fmt.Sprintf(TokenPattern, uuid.NewString()[8:16], val)
	return encryptMessage([]byte(AuthOption.Salt), token)
}

// encryptMessage 对消息进行加密
func encryptMessage(key []byte, message string) (string, error) {
	byteMsg := []byte(message)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("could not create new cipher: %v", err)
	}

	cipherText := make([]byte, aes.BlockSize+len(byteMsg))
	iv := cipherText[:aes.BlockSize]
	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		return "", fmt.Errorf("could not encrypt: %v", err)
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(cipherText[aes.BlockSize:], byteMsg)

	return base64.StdEncoding.EncodeToString(cipherText), nil
}

// decryptMessage 对消息进行解密
func decryptMessage(key []byte, message string) (string, error) {
	cipherText, err := base64.StdEncoding.DecodeString(message)
	if err != nil {
		return "", fmt.Errorf("could not base64 decode: %v", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("could not create new cipher: %v", err)
	}

	if len(cipherText) < aes.BlockSize {
		return "", fmt.Errorf("invalid ciphertext block size")
	}

	iv := cipherText[:aes.BlockSize]
	cipherText = cipherText[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(cipherText, cipherText)

	return string(cipherText), nil
}
