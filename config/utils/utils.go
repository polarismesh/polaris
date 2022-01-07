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

package utils

import (
	"crypto/md5"
	"errors"
	"fmt"
	"github.com/golang/protobuf/ptypes/wrappers"
	"io"
	"regexp"
	"strconv"
)

const (
	FileContentMaxLength = 20000
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
	if len(content) > FileContentMaxLength {
		return errors.New("content length too long. max length =" + strconv.Itoa(FileContentMaxLength))
	}
	return nil
}
