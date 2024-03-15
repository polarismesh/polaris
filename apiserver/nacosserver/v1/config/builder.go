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
	"github.com/emicklei/go-restful/v3"

	"github.com/polarismesh/polaris/apiserver/nacosserver/model"
	nacoshttp "github.com/polarismesh/polaris/apiserver/nacosserver/v1/http"
)

func BuildConfigFile(req *restful.Request) (*model.ConfigFile, error) {
	baseInfo, err := parseConfigFileBase(req)
	if err != nil {
		return nil, err
	}

	content, err := nacoshttp.Required(req, "content")
	if err != nil {
		return nil, err
	}

	configfile := &model.ConfigFile{
		ConfigFileBase: *baseInfo,
		Content:        content,
		AppName:        nacoshttp.Optional(req, "appName", ""),
		SrcUser:        nacoshttp.Optional(req, "src_user", ""),
		Labels:         nacoshttp.Optional(req, "config_tags", ""),
		Description:    nacoshttp.Optional(req, "desc", ""),
	}

	return configfile, nil
}
