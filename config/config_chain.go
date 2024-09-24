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

package config

import (
	"context"
	"encoding/base64"
	"fmt"

	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"go.uber.org/zap"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

// ConfigFileChain
type ConfigFileChain interface {
	// Init
	Init(svr *Server)
	// Name
	Name() string
	// BeforeCreateFile
	BeforeCreateFile(context.Context, *model.ConfigFile) *apiconfig.ConfigResponse
	// AfterGetFile
	AfterGetFile(context.Context, *model.ConfigFile) (*model.ConfigFile, error)
	// BeforeUpdateFile
	BeforeUpdateFile(context.Context, *model.ConfigFile) *apiconfig.ConfigResponse
	// AfterGetFileRelease
	AfterGetFileRelease(context.Context, *model.ConfigFileRelease) (*model.ConfigFileRelease, error)
	// AfterGetFileHistory
	AfterGetFileHistory(context.Context, *model.ConfigFileReleaseHistory) (*model.ConfigFileReleaseHistory, error)
}

type CryptoConfigFileChain struct {
	svr *Server
}

func (chain *CryptoConfigFileChain) Init(svr *Server) {
	chain.svr = svr
}

func (chain *CryptoConfigFileChain) Name() string {
	return "CryptoConfigFileChain"
}

// BeforeCreateFile
func (chain *CryptoConfigFileChain) BeforeCreateFile(ctx context.Context,
	file *model.ConfigFile) *apiconfig.ConfigResponse {
	// 配置加密
	if !file.IsEncrypted() {
		return nil
	}
	if err := chain.encryptConfigFile(ctx, file, file.GetEncryptAlgo(), ""); err != nil {
		log.Error("[Config][Service] encrypt config file error.", utils.RequestID(ctx),
			utils.ZapNamespace(file.Namespace), utils.ZapGroup(file.Group),
			utils.ZapFileName(file.Name), zap.Error(err))
		return api.NewConfigResponseWithInfo(apimodel.Code_EncryptConfigFileException, err.Error())
	}
	return nil
}

// AfterCreateFile
func (chain *CryptoConfigFileChain) AfterGetFile(ctx context.Context,
	file *model.ConfigFile) (*model.ConfigFile, error) {
	encryptAlgo := file.GetEncryptAlgo()
	dataKey := file.GetEncryptDataKey()
	if file.IsEncrypted() {
		file.Encrypt = true
	}

	plainContent, err := chain.decryptConfigFileContent(dataKey, encryptAlgo, file.Content)

	// TODO: 这个逻辑需要优化，在1.17.3处理
	// 前一次发布的配置并未加密，现在准备发布的配置是开启了加密的，因此这里可能配置就是一个未加密的状态
	// 这里就直接原样返回
	if err == nil && plainContent != "" {
		file.Content = plainContent
	}
	if err != nil {
		log.Error("[Config][Chain][Crypto] decrypt config file content",
			utils.ZapNamespace(file.Namespace), utils.ZapGroup(file.Group),
			utils.ZapFileName(file.Name), zap.Error(err))
	}
	delete(file.Metadata, model.MetaKeyConfigFileDataKey)
	return file, nil
}

// BeforeUpdateFile
func (chain *CryptoConfigFileChain) BeforeUpdateFile(ctx context.Context,
	file *model.ConfigFile) *apiconfig.ConfigResponse {

	// 配置加密
	encryAlgo := file.GetEncryptAlgo()
	encryDataKey := file.GetEncryptDataKey()
	// 算法以传进来的参数为准
	if file.IsEncrypted() {
		// 如果加密算法进行了调整，dataKey 需要重新生成
		if err := chain.encryptConfigFile(ctx, file, encryAlgo, encryDataKey); err != nil {
			log.Error("[Config][Service] update encrypt config file error.", utils.RequestID(ctx),
				utils.ZapNamespace(file.Namespace), utils.ZapGroup(file.Group), utils.ZapFileName(file.Name),
				zap.Error(err))
			return api.NewConfigResponseWithInfo(apimodel.Code_EncryptConfigFileException, err.Error())
		}
	} else {
		chain.cleanEncryptConfigFileInfo(ctx, file)
	}
	return nil
}

// AfterGetFileRelease
func (chain *CryptoConfigFileChain) AfterGetFileRelease(ctx context.Context,
	release *model.ConfigFileRelease) (*model.ConfigFileRelease, error) {

	s := chain.svr
	// decryptConfigFileRelease 解密配置文件发布纪录
	if s.cryptoManager == nil || release == nil {
		return release, nil
	}
	encryptAlgo := release.GetEncryptAlgo()
	encryptDataKey := release.GetEncryptDataKey()
	plainContent, err := chain.decryptConfigFileContent(encryptDataKey, encryptAlgo, release.Content)
	if err == nil && plainContent != "" {
		release.Content = plainContent
	}
	if err != nil {
		log.Error("[Config][Chain][Crypto] decrypt release config file content",
			utils.ZapNamespace(release.Namespace), utils.ZapGroup(release.Group),
			utils.ZapFileName(release.Name), zap.Error(err))
	}
	delete(release.Metadata, model.MetaKeyConfigFileDataKey)
	return release, nil
}

// AfterGetFileHistory
func (chain *CryptoConfigFileChain) AfterGetFileHistory(ctx context.Context,
	history *model.ConfigFileReleaseHistory) (*model.ConfigFileReleaseHistory, error) {
	if history == nil {
		return history, nil
	}
	if !history.IsEncrypted() {
		return history, nil
	}
	encryptAlgo := history.GetEncryptAlgo()
	dataKey := history.GetEncryptDataKey()
	plainContent, err := chain.decryptConfigFileContent(dataKey, encryptAlgo, history.Content)
	if err == nil && plainContent != "" {
		history.Content = plainContent
	} else {
		log.Error("[Config][Chain][Crypto] decrypt history config file content",
			utils.ZapNamespace(history.Namespace), utils.ZapGroup(history.Group),
			utils.ZapFileName(history.Name), zap.Error(err))
	}
	delete(history.Metadata, model.MetaKeyConfigFileDataKey)
	return history, err
}

// decryptConfigFileContent 解密配置文件
func (chain *CryptoConfigFileChain) decryptConfigFileContent(dataKey, algorithm, content string) (string, error) {
	cryptoMgr := chain.svr.cryptoManager
	if cryptoMgr == nil {
		return "", nil
	}
	// 没有加密算法不解密
	if algorithm == "" {
		return "", nil
	}
	crypto, err := cryptoMgr.GetCrypto(algorithm)
	if err != nil {
		return "", err
	}
	if crypto == nil {
		return "", nil
	}
	dateKeyBytes, err := base64.StdEncoding.DecodeString(dataKey)
	if err != nil {
		return "", err
	}
	// 解密
	plainContent, err := crypto.Decrypt(content, dateKeyBytes)
	if err != nil {
		return "", err
	}
	return plainContent, nil
}

// cleanEncryptConfigFileInfo 清理配置加密文件的内容信息
func (chain *CryptoConfigFileChain) cleanEncryptConfigFileInfo(ctx context.Context, configFile *model.ConfigFile) {
	delete(configFile.Metadata, model.MetaKeyConfigFileDataKey)
	delete(configFile.Metadata, model.MetaKeyConfigFileEncryptAlgo)
	delete(configFile.Metadata, model.MetaKeyConfigFileUseEncrypted)
}

// encryptConfigFile 加密配置文件
func (chain *CryptoConfigFileChain) encryptConfigFile(ctx context.Context, configFile *model.ConfigFile,
	algorithm string, dataKey string) error {

	s := chain.svr
	if s.cryptoManager == nil || configFile == nil {
		return nil
	}
	crypto, err := s.cryptoManager.GetCrypto(algorithm)
	if err != nil {
		return err
	}

	var dateKeyBytes []byte
	if dataKey == "" {
		dateKeyBytes, err = crypto.GenerateKey()
		if err != nil {
			return err
		}
	} else {
		dateKeyBytes, err = base64.StdEncoding.DecodeString(dataKey)
		if err != nil {
			return err
		}
	}
	content := configFile.Content
	cipherContent, err := crypto.Encrypt(content, dateKeyBytes)
	if err != nil {
		return err
	}
	configFile.Content = cipherContent
	if len(configFile.Metadata) == 0 {
		configFile.Metadata = map[string]string{}
	}
	configFile.Metadata[model.MetaKeyConfigFileDataKey] = base64.StdEncoding.EncodeToString(dateKeyBytes)
	configFile.Metadata[model.MetaKeyConfigFileEncryptAlgo] = algorithm
	configFile.Metadata[model.MetaKeyConfigFileUseEncrypted] = "true"

	return nil
}

type ReleaseConfigFileChain struct {
	svr *Server
}

func (chain *ReleaseConfigFileChain) Init(svr *Server) {
	chain.svr = svr
}

func (chain *ReleaseConfigFileChain) Name() string {
	return "CryptoConfigFileChain"
}

// BeforeCreateFile
func (chain *ReleaseConfigFileChain) BeforeCreateFile(ctx context.Context,
	file *model.ConfigFile) *apiconfig.ConfigResponse {
	return nil
}

// AfterCreateFile
func (chain *ReleaseConfigFileChain) AfterGetFile(ctx context.Context,
	file *model.ConfigFile) (*model.ConfigFile, error) {

	namespace := file.Namespace
	group := file.Group
	name := file.Name
	// 首先检测灰度版本
	if grayFile := chain.svr.fileCache.GetActiveGrayRelease(namespace, group, name); grayFile != nil {
		if grayFile.Content == file.OriginContent {
			file.Status = utils.ReleaseTypeGray
			file.ReleaseBy = grayFile.ModifyBy
			file.ReleaseTime = grayFile.ModifyTime
		} else {
			file.Status = utils.ReleaseStatusToRelease
		}
	} else if fullFile := chain.svr.fileCache.GetActiveRelease(namespace, group, name); fullFile != nil {
		// 如果最后一次发布的内容和当前文件内容一致，则展示最后一次发布状态。否则说明文件有修改，待发布
		if fullFile.Content == file.OriginContent {
			file.Status = utils.ReleaseTypeNormal
			file.ReleaseBy = fullFile.ModifyBy
			file.ReleaseTime = fullFile.ModifyTime
		} else {
			file.Status = utils.ReleaseStatusToRelease
		}
	} else {
		// 如果从来没有发布过，也是待发布状态
		file.Status = utils.ReleaseStatusToRelease
	}
	return file, nil
}

// BeforeUpdateFile
func (chain *ReleaseConfigFileChain) BeforeUpdateFile(ctx context.Context,
	file *model.ConfigFile) *apiconfig.ConfigResponse {
	return nil
}

// AfterGetFileRelease
func (chain *ReleaseConfigFileChain) AfterGetFileRelease(ctx context.Context,
	ret *model.ConfigFileRelease) (*model.ConfigFileRelease, error) {

	if ret.ReleaseType == model.ReleaseTypeGray {
		rule := chain.svr.caches.Gray().GetGrayRule(model.GetGrayConfigRealseKey(ret.SimpleConfigFileRelease))
		if rule == nil {
			return nil, fmt.Errorf("gray rule not found")
		}
		ret.BetaLabels = rule
	}
	return ret, nil
}

// AfterGetFileHistory
func (chain *ReleaseConfigFileChain) AfterGetFileHistory(ctx context.Context,
	history *model.ConfigFileReleaseHistory) (*model.ConfigFileReleaseHistory, error) {
	return history, nil
}
