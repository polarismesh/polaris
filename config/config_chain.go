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
	"errors"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/utils"
	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type ConfigFileChain interface {
	// Init
	Init(svr *Server)
	// Name
	Name() string
	// BeforeCreateFile
	BeforeCreateFile(context.Context, *apiconfig.ConfigFile) *apiconfig.ConfigResponse
	// AfterGetFile
	AfterGetFile(context.Context, *apiconfig.ConfigFile) (*apiconfig.ConfigFile, error)
	// BeforeUpdateFile
	BeforeUpdateFile(context.Context, *apiconfig.ConfigFile) *apiconfig.ConfigResponse
	// AfterGetFileRelease
	AfterGetFileRelease(context.Context, *apiconfig.ConfigFileRelease) (*apiconfig.ConfigFileRelease, error)
	// AfterGetFileHistory
	AfterGetFileHistory(context.Context, *apiconfig.ConfigFileReleaseHistory) (*apiconfig.ConfigFileReleaseHistory, error)
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
	file *apiconfig.ConfigFile) *apiconfig.ConfigResponse {
	// 配置加密
	if file.GetEncrypted().GetValue() && file.GetEncryptAlgo().GetValue() != "" {
		if err := chain.encryptConfigFile(ctx, file, file.GetEncryptAlgo().GetValue(), ""); err != nil {
			log.Error("[Config][Service] encrypt config file error.", utils.ZapRequestIDByCtx(ctx),
				utils.ZapNamespace(file.GetNamespace().GetValue()), utils.ZapGroup(file.GetGroup().GetValue()),
				utils.ZapFileName(file.GetName().GetValue()), zap.Error(err))
			return api.NewConfigFileResponse(apimodel.Code_EncryptConfigFileException, file)
		}
	}
	return nil
}

// AfterCreateFile
func (chain *CryptoConfigFileChain) AfterGetFile(ctx context.Context,
	file *apiconfig.ConfigFile) (*apiconfig.ConfigFile, error) {

	var (
		tags        = file.GetTags()
		dataKey     string
		encryptAlgo string
	)
	if len(tags) == 0 {
		saveTags, err := chain.svr.queryTagsByConfigFileWithAPIModels(ctx, file.GetNamespace().GetValue(),
			file.GetGroup().GetValue(), file.GetName().GetValue())
		if err != nil {
			return nil, err
		}
		tags = saveTags
	}
	newTags := make([]*apiconfig.ConfigFileTag, 0, len(tags))

	file.Encrypted = utils.NewBoolValue(false)
	for i := range tags {
		tag := tags[i]
		if tag.GetKey().GetValue() == utils.ConfigFileTagKeyEncryptAlgo {
			file.EncryptAlgo = utils.NewStringValue(tag.GetValue().GetValue())
			encryptAlgo = tag.GetValue().GetValue()
			file.Encrypted = utils.NewBoolValue(true)
		}
		if tag.GetKey().GetValue() == utils.ConfigFileTagKeyDataKey {
			dataKey = tag.GetValue().GetValue()
			continue
		}
		newTags = append(newTags, tag)
	}
	file.Tags = newTags
	plainContent, err := chain.decryptConfigFileContent(dataKey, encryptAlgo, file.GetContent().GetValue())

	// TODO: 这个逻辑需要优化，在1.17.3处理
	// 前一次发布的配置并未加密，现在准备发布的配置是开启了加密的，因此这里可能配置就是一个未加密的状态
	// 这里就直接原样返回
	if err == nil && plainContent != "" {
		file.Content = wrapperspb.String(plainContent)
	} else {
		log.Error("[Config][Chain][Crypto] decrypt config file content",
			utils.ZapNamespace(file.GetNamespace().GetValue()), utils.ZapGroup(file.GetGroup().GetValue()),
			utils.ZapFileName(file.GetName().GetValue()), zap.Error(err))
	}
	return file, nil
}

// BeforeUpdateFile
func (chain *CryptoConfigFileChain) BeforeUpdateFile(ctx context.Context,
	file *apiconfig.ConfigFile) *apiconfig.ConfigResponse {

	namespace := file.Namespace.GetValue()
	group := file.Group.GetValue()
	name := file.Name.GetValue()

	// 配置加密
	saveAlgo, dataKey, err := chain.getEncryptAlgorithmAndDataKey(ctx, namespace, group, name)
	if err != nil {
		return api.NewConfigFileResponse(apimodel.Code_StoreLayerException, file)
	}
	// 算法以传进来的参数为准
	if algorithm := file.GetEncryptAlgo().GetValue(); algorithm != "" {
		// 如果加密算法进行了调整，dataKey 需要重新生成
		if saveAlgo != "" && saveAlgo != algorithm {
			dataKey = ""
		}
		if err := chain.encryptConfigFile(ctx, file, algorithm, dataKey); err != nil {
			log.Error("[Config][Service] update encrypt config file error.", utils.ZapRequestIDByCtx(ctx),
				utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(name), zap.Error(err))
			return api.NewConfigFileResponse(apimodel.Code_EncryptConfigFileException, file)
		}
	} else {
		chain.cleanEncryptConfigFileInfo(ctx, file)
	}
	return nil
}

// AfterGetFileRelease
func (chain *CryptoConfigFileChain) AfterGetFileRelease(ctx context.Context,
	release *apiconfig.ConfigFileRelease) (*apiconfig.ConfigFileRelease, error) {

	s := chain.svr
	// decryptConfigFileRelease 解密配置文件发布纪录
	if s.cryptoManager == nil || release == nil {
		return release, nil
	}
	algorithm, dataKey, err := chain.getEncryptAlgorithmAndDataKey(ctx, release.GetNamespace().GetValue(),
		release.GetGroup().GetValue(), release.GetFileName().GetValue())
	if err != nil {
		return nil, err
	}
	plainContent, err := chain.decryptConfigFileContent(dataKey, algorithm, release.GetContent().GetValue())
	if err == nil && plainContent != "" {
		release.Content = utils.NewStringValue(plainContent)
	} else {
		log.Error("[Config][Chain][Crypto] decrypt release config file content",
			utils.ZapNamespace(release.GetNamespace().GetValue()), utils.ZapGroup(release.GetGroup().GetValue()),
			utils.ZapFileName(release.GetName().GetValue()), zap.Error(err))
	}
	return release, nil
}

// AfterGetFileHistory
func (chain *CryptoConfigFileChain) AfterGetFileHistory(ctx context.Context,
	history *apiconfig.ConfigFileReleaseHistory) (*apiconfig.ConfigFileReleaseHistory, error) {
	if history == nil {
		return history, nil
	}
	var (
		dataKey   = ""
		algorithm = ""
	)
	for _, tag := range history.GetTags() {
		if tag.Key.GetValue() == utils.ConfigFileTagKeyDataKey {
			dataKey = tag.Value.GetValue()
		}
		if tag.Key.GetValue() == utils.ConfigFileTagKeyEncryptAlgo {
			algorithm = tag.Value.GetValue()
		}
	}
	plainContent, err := chain.decryptConfigFileContent(dataKey, algorithm, history.GetContent().GetValue())
	if err == nil && plainContent != "" {
		history.Content = utils.NewStringValue(plainContent)
	} else {
		log.Error("[Config][Chain][Crypto] decrypt history config file content",
			utils.ZapNamespace(history.GetNamespace().GetValue()), utils.ZapGroup(history.GetGroup().GetValue()),
			utils.ZapFileName(history.GetName().GetValue()), zap.Error(err))
	}
	return history, err
}

// decryptConfigFileContent 解密配置文件
func (chain *CryptoConfigFileChain) decryptConfigFileContent(dataKey, algorithm, content string) (string, error) {
	cryptoMgr := chain.svr.cryptoManager
	if cryptoMgr == nil {
		return "", nil
	}
	// 没有加密算法不加密
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
func (chain *CryptoConfigFileChain) cleanEncryptConfigFileInfo(ctx context.Context, configFile *apiconfig.ConfigFile) {
	newTags := make([]*apiconfig.ConfigFileTag, 0, 4)
	for i := range configFile.Tags {
		tag := configFile.Tags[i]
		keyName := tag.GetKey().GetValue()
		if keyName == utils.ConfigFileTagKeyDataKey || keyName == utils.ConfigFileTagKeyEncryptAlgo ||
			keyName == utils.ConfigFileTagKeyUseEncrypted {
			continue
		}
		newTags = append(newTags, tag)
	}
	configFile.Tags = newTags
}

// encryptConfigFile 加密配置文件
func (chain *CryptoConfigFileChain) encryptConfigFile(ctx context.Context,
	configFile *apiconfig.ConfigFile, algorithm string, dataKey string) error {
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
	content := configFile.Content.GetValue()
	cipherContent, err := crypto.Encrypt(content, dateKeyBytes)
	if err != nil {
		return err
	}
	configFile.Content = utils.NewStringValue(cipherContent)
	tags := []*apiconfig.ConfigFileTag{
		{
			Key:   utils.NewStringValue(utils.ConfigFileTagKeyDataKey),
			Value: utils.NewStringValue(base64.StdEncoding.EncodeToString(dateKeyBytes)),
		},
		{
			Key:   utils.NewStringValue(utils.ConfigFileTagKeyEncryptAlgo),
			Value: utils.NewStringValue(algorithm),
		},
		{
			Key:   utils.NewStringValue(utils.ConfigFileTagKeyUseEncrypted),
			Value: utils.NewStringValue("true"),
		},
	}
	configFile.Tags = append(configFile.Tags, tags...)
	return nil
}

// getConfigFileDataKey 获取加密配置文件数据密钥
func (chain *CryptoConfigFileChain) getEncryptAlgorithmAndDataKey(ctx context.Context,
	namespace, group, fileName string) (string, string, error) {
	s := chain.svr
	tags, err := s.queryTagsByConfigFileWithAPIModels(ctx, namespace, group, fileName)
	if err != nil {
		return "", "", err
	}
	var (
		algorithm string
		dataKey   string
	)
	for _, tag := range tags {
		if tag.Key.GetValue() == utils.ConfigFileTagKeyDataKey {
			dataKey = tag.Value.GetValue()
		}
		if tag.Key.GetValue() == utils.ConfigFileTagKeyEncryptAlgo {
			algorithm = tag.Value.GetValue()
		}
	}
	return algorithm, dataKey, nil
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
	file *apiconfig.ConfigFile) *apiconfig.ConfigResponse {
	return nil
}

// AfterCreateFile
func (chain *ReleaseConfigFileChain) AfterGetFile(ctx context.Context,
	file *apiconfig.ConfigFile) (*apiconfig.ConfigFile, error) {

	namespace := file.Namespace.GetValue()
	group := file.Group.GetValue()
	name := file.Name.GetValue()

	// 填充发布信息
	latestReleaseRsp := chain.svr.GetConfigFileLatestReleaseHistory(ctx, namespace, group, name)
	if latestReleaseRsp.Code.GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
		log.Error("[Config][Service] get config file latest release error.", utils.ZapRequestIDByCtx(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(name),
			zap.String("msg", latestReleaseRsp.GetInfo().GetValue()))
		return nil, errors.New(latestReleaseRsp.GetInfo().GetValue())
	}

	latestRelease := latestReleaseRsp.ConfigFileReleaseHistory
	if latestRelease != nil && latestRelease.Type.GetValue() == utils.ReleaseTypeNormal {
		file.ReleaseBy = latestRelease.CreateBy
		file.ReleaseTime = latestRelease.CreateTime
		// 如果最后一次发布的内容和当前文件内容一致，则展示最后一次发布状态。否则说明文件有修改，待发布
		if latestRelease.Content.GetValue() == file.Content.GetValue() {
			file.Status = latestRelease.Status
		} else {
			file.Status = utils.NewStringValue(utils.ReleaseStatusToRelease)
		}
	} else {
		// 如果从来没有发布过，也是待发布状态
		file.Status = utils.NewStringValue(utils.ReleaseStatusToRelease)
		file.ReleaseBy = utils.NewStringValue("")
		file.ReleaseTime = utils.NewStringValue("")
	}

	return file, nil
}

// BeforeUpdateFile
func (chain *ReleaseConfigFileChain) BeforeUpdateFile(ctx context.Context,
	file *apiconfig.ConfigFile) *apiconfig.ConfigResponse {
	return nil
}

// AfterGetFileRelease
func (chain *ReleaseConfigFileChain) AfterGetFileRelease(ctx context.Context,
	release *apiconfig.ConfigFileRelease) (*apiconfig.ConfigFileRelease, error) {
	return release, nil
}

// AfterGetFileHistory
func (chain *ReleaseConfigFileChain) AfterGetFileHistory(ctx context.Context,
	history *apiconfig.ConfigFileReleaseHistory) (*apiconfig.ConfigFileReleaseHistory, error) {
	return history, nil
}
