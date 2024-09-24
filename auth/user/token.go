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

package defaultuser

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"

	"github.com/polarismesh/polaris/auth"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
)

// decodeToken 解析 token 信息，如果 t == ""，直接返回一个空对象
func (svr *Server) decodeToken(t string) (auth.OperatorInfo, error) {
	if t == "" {
		return auth.OperatorInfo{}, authcommon.ErrorTokenInvalid
	}

	ret, err := DecryptMessage([]byte(svr.authOpt.Salt), t)
	if err != nil {
		return auth.OperatorInfo{}, err
	}
	tokenDetails := strings.Split(ret, TokenSplit)
	if len(tokenDetails) != 2 {
		return auth.OperatorInfo{}, authcommon.ErrorTokenInvalid
	}

	detail := strings.Split(tokenDetails[1], "/")
	if len(detail) != 2 {
		return auth.OperatorInfo{}, authcommon.ErrorTokenInvalid
	}

	tokenInfo := auth.OperatorInfo{
		Origin:      t,
		IsUserToken: detail[0] == authcommon.TokenForUser,
		OperatorID:  detail[1],
		Role:        authcommon.UnknownUserRole,
	}
	return tokenInfo, nil
}

type TokenPrincipal interface {
	GetToken() string
	Disable() bool
	OwnerID() string
	SelfID() string
}

// checkToken 对 token 进行检查，如果 token 是一个空，直接返回默认值，但是不返回错误
// return {owner-id} {is-owner} {error}
func (svr *Server) checkToken(tokenInfo *auth.OperatorInfo) (string, bool, error) {
	if auth.IsEmptyOperator(*tokenInfo) {
		return "", false, nil
	}
	principal, err := svr.getTokenPrincipal(tokenInfo)
	if err != nil {
		return "", false, err
	}

	if tokenInfo.Origin != principal.GetToken() {
		return "", false, authcommon.ErrorTokenNotExist
	}
	tokenInfo.Disable = principal.Disable()
	if principal.OwnerID() == "" {
		return principal.SelfID(), true, nil
	}

	return principal.OwnerID(), false, nil
}

func (svr *Server) getTokenPrincipal(tokenInfo *auth.OperatorInfo) (TokenPrincipal, error) {
	if tokenInfo.IsUserToken {
		user := svr.cacheMgr.User().GetUserByID(tokenInfo.OperatorID)
		if user != nil {
			return user, nil
		}
		if err := svr.cacheMgr.User().Update(); err != nil {
			return nil, err
		}
		user = svr.cacheMgr.User().GetUserByID(tokenInfo.OperatorID)
		if user != nil {
			return user, nil
		}
		return nil, authcommon.ErrorNoUser
	}
	group := svr.cacheMgr.User().GetGroup(tokenInfo.OperatorID)
	if group != nil {
		return group, nil
	}
	if err := svr.cacheMgr.User().Update(); err != nil {
		return nil, err
	}
	group = svr.cacheMgr.User().GetGroup(tokenInfo.OperatorID)
	if group != nil {
		return group, nil
	}
	return nil, authcommon.ErrorNoUserGroup
}

const (
	// TokenPattern token 的格式 随机字符串::[uid/xxx | groupid/xxx]
	TokenPattern string = "%s::%s"
	// TokenSplit token 的分隔符
	TokenSplit string = "::"
)

// createUserToken Create a user token
func createUserToken(uid string, salt string) (string, error) {
	return CreateToken(uid, "", salt)
}

// createGroupToken Create user group token
func createGroupToken(gid string, salt string) (string, error) {
	return CreateToken("", gid, salt)
}

// createToken Determine what type of Token created according to the incoming parameters
func CreateToken(uid, gid string, salt string) (string, error) {
	if uid == "" && gid == "" {
		return "", errors.New("uid and groupid not be empty at the same time")
	}

	var val string
	if uid == "" {
		val = fmt.Sprintf("%s/%s", authcommon.TokenForUserGroup, gid)
	} else {
		val = fmt.Sprintf("%s/%s", authcommon.TokenForUser, uid)
	}

	token := fmt.Sprintf(TokenPattern, uuid.NewString()[8:16], val)
	return encryptMessage([]byte(salt), token)
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
func DecryptMessage(key []byte, message string) (string, error) {
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
