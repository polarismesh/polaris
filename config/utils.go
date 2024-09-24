/*
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
	"archive/zip"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"regexp"
	"unicode/utf8"

	"github.com/golang/protobuf/ptypes/wrappers"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

var (
	regFileName = regexp.MustCompile(`^[\dA-Za-z-./:_]+$`)
)

// CheckFileName 校验文件名
func CheckFileName(name *wrappers.StringValue) error {
	if name == nil {
		return errors.New(utils.NilErrString)
	}

	if name.GetValue() == "" {
		return errors.New(utils.EmptyErrString)
	}
	return nil
}

// CalMd5 计算md5值
func CalMd5(content string) string {
	h := md5.New()
	h.Write([]byte(content))
	return hex.EncodeToString(h.Sum(nil))
}

// CheckContentLength 校验文件内容长度
func CheckContentLength(content string, max int) error {
	if utf8.RuneCountInString(content) > max {
		return fmt.Errorf("content length too long. max length =%d", max)
	}

	return nil
}

func CompressConfigFiles(files []*model.ConfigFile,
	fileID2Tags map[uint64][]*model.ConfigFileTag, isExportGroup bool) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	defer w.Close()

	var configFileMetas = make(map[string]*utils.ConfigFileMeta)
	for _, file := range files {
		fileName := file.Name
		if isExportGroup {
			fileName = path.Join(file.Group, file.Name)
		}

		configFileMetas[fileName] = &utils.ConfigFileMeta{
			Tags:    make(map[string]string),
			Comment: file.Comment,
		}
		for _, tag := range fileID2Tags[file.Id] {
			configFileMetas[fileName].Tags[tag.Key] = tag.Value
		}
		f, err := w.Create(fileName)
		if err != nil {
			return nil, err
		}
		if _, err := f.Write([]byte(file.Content)); err != nil {
			return nil, err
		}
	}
	// 生成配置元文件
	f, err := w.Create(utils.ConfigFileMetaFileName)
	if err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(configFileMetas, "", "\t")
	if err != nil {
		return nil, err
	}
	if _, err := f.Write(data); err != nil {
		return nil, err
	}
	return &buf, nil
}
