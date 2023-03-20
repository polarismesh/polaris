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

package i18n

//go:generate go run cmd/gen.go

import (
	"fmt"
	"sync"

	"github.com/BurntSushi/toml"
	ii18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"go.uber.org/zap"
	"golang.org/x/text/language"

	"github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/utils"
)

var (
	bundle       *ii18n.Bundle
	i18nMsgCache map[uint32]*ii18n.Message
	once         sync.Once
)

func init() {
	bundle = ii18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
}

// LoadI18nMessageFile 加载i18n配置文件
func LoadI18nMessageFile(path string) {
	if _, err := bundle.LoadMessageFile(path); err != nil {
		log.Error("[i18n][MessageFile] load fail", zap.Error(err))
	}
}

// Translate 国际化code所对应的msg信息
func Translate(code uint32, langs ...string) (string, error) {
	once.Do(func() {
		LoadI18nMessageFile(utils.ConfDir + "i18n/zh.toml")
		LoadI18nMessageFile(utils.ConfDir + "i18n/en.toml")
	})
	msg, ok := i18nMsgCache[code]
	if !ok {
		msg = &ii18n.Message{ID: fmt.Sprintf("%d", code)}
	}
	return ii18n.NewLocalizer(bundle, langs...).LocalizeMessage(msg)
}
