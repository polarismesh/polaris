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

package utils

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"

	"github.com/golang/protobuf/ptypes/wrappers"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/utils"
)

const (
	fileContentMaxLength = 20000 // 文件内容限制为 2w 个字符
)

// CheckResourceName 检查资源名称
func CheckResourceName(name *wrappers.StringValue) error {
	if name == nil {
		return errors.New("nil")
	}

	if name.GetValue() == "" {
		return errors.New("empty")
	}

	regStr := "^[0-9A-Za-z-.:_]+$"
	ok, err := regexp.MatchString(regStr, name.GetValue())
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("name contains invalid character")
	}

	return nil
}

// CalMd5 计算md5值
func CalMd5(content string) string {
	h := md5.New()
	_, _ = io.WriteString(h, content)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// CheckContentLength 校验文件内容长度
func CheckContentLength(content string) error {
	if len(content) > fileContentMaxLength {
		return errors.New("content length too long. max length =" + strconv.Itoa(fileContentMaxLength))
	}
	return nil
}

// GenConfigFileResponse 为客户端生成响应对象
func GenConfigFileResponse(namespace, group, fileName, content, md5 string, version uint64) *api.ConfigClientResponse {
	configFile := &api.ClientConfigFileInfo{
		Namespace: utils.NewStringValue(namespace),
		Group:     utils.NewStringValue(group),
		FileName:  utils.NewStringValue(fileName),
		Content:   utils.NewStringValue(content),
		Version:   utils.NewUInt64Value(version),
		Md5:       utils.NewStringValue(md5),
	}
	return api.NewConfigClientResponse(api.ExecuteSuccess, configFile)
}

type kv struct {
	Key   string
	Value string
}

// ToTagJsonStr 把 Tags 转化成 Json 字符串
func ToTagJsonStr(tags []*api.ConfigFileTag) string {
	if len(tags) == 0 {
		return "[]"
	}
	var kvs []kv
	for _, tag := range tags {
		kvs = append(kvs, kv{
			Key:   tag.Key.GetValue(),
			Value: tag.Value.GetValue(),
		})
	}
	bytes, _ := json.Marshal(kvs)
	return string(bytes)
}

// FromTagJson 从 Tags Json 字符串里反序列化出 Tags
func FromTagJson(tagStr string) []*api.ConfigFileTag {
	if tagStr == "" {
		return nil
	}
	kvs := &[]kv{}
	_ = json.Unmarshal([]byte(tagStr), kvs)
	var tags []*api.ConfigFileTag
	for _, kv := range *kvs {
		tags = append(tags, &api.ConfigFileTag{
			Key:   utils.NewStringValue(kv.Key),
			Value: utils.NewStringValue(kv.Value),
		})
	}
	return tags
}
