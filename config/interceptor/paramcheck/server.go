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

package paramcheck

import (
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/golang/protobuf/ptypes/wrappers"
	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"

	cachetypes "github.com/polarismesh/polaris/cache/api"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/config"
	"github.com/polarismesh/polaris/store"
)

var _ config.ConfigCenterServer = (*Server)(nil)

// Config 配置中心模块启动参数
type Config struct {
	ContentMaxLength int64 `yaml:"contentMaxLength"`
}

// Server 配置中心核心服务
type Server struct {
	cacheMgr   cachetypes.CacheManager
	nextServer config.ConfigCenterServer
	storage    store.Store
	cfg        Config
}

func New(nextServer config.ConfigCenterServer, cacheMgr cachetypes.CacheManager,
	storage store.Store, cfg config.Config) config.ConfigCenterServer {
	proxy := &Server{
		nextServer: nextServer,
		cacheMgr:   cacheMgr,
		storage:    storage,
		cfg: Config{
			ContentMaxLength: cfg.ContentMaxLength,
		},
	}
	return proxy
}

func (s *Server) checkNamespaceExisted(namespaceName string) bool {
	if val := s.cacheMgr.Namespace().GetNamespace(namespaceName); val != nil {
		return true
	}
	namespace, _ := s.storage.GetNamespace(namespaceName)
	return namespace != nil
}

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

// CheckContentLength 校验文件内容长度
func CheckContentLength(content string, max int) error {
	if utf8.RuneCountInString(content) > max {
		return fmt.Errorf("content length too long. max length =%d", max)
	}

	return nil
}

func checkReadFileParameter(req *apiconfig.ConfigFile) *apiconfig.ConfigResponse {
	if req.GetNamespace().GetValue() == "" {
		return api.NewConfigResponse(apimodel.Code_InvalidNamespaceName)
	}
	if req.GetGroup().GetValue() == "" {
		return api.NewConfigResponse(apimodel.Code_InvalidConfigFileGroupName)
	}
	if req.GetName().GetValue() == "" {
		return api.NewConfigResponse(apimodel.Code_InvalidConfigFileName)
	}
	return nil
}

func (s *Server) checkConfigFileParams(configFile *apiconfig.ConfigFile) *apiconfig.ConfigResponse {
	if configFile == nil {
		return api.NewConfigFileResponse(apimodel.Code_InvalidParameter, configFile)
	}
	if err := CheckFileName(configFile.Name); err != nil {
		return api.NewConfigFileResponse(apimodel.Code_InvalidConfigFileName, configFile)
	}
	if err := utils.CheckResourceName(configFile.Namespace); err != nil {
		return api.NewConfigFileResponse(apimodel.Code_InvalidNamespaceName, configFile)
	}
	if err := CheckContentLength(configFile.Content.GetValue(), int(s.cfg.ContentMaxLength)); err != nil {
		return api.NewConfigResponseWithInfo(apimodel.Code_InvalidConfigFileContentLength, err.Error())
	}
	if len(configFile.Tags) > 0 {
		for _, tag := range configFile.Tags {
			if tag.Key.GetValue() == "" || tag.Value.GetValue() == "" {
				return api.NewConfigFileResponse(apimodel.Code_InvalidConfigFileTags, configFile)
			}
		}
	}
	return nil
}

var (
	availableSearch = map[string]map[string]string{
		"config_file": {
			"namespace":   "namespace",
			"group":       "group",
			"name":        "name",
			"offset":      "offset",
			"limit":       "limit",
			"order_type":  "order_type",
			"order_field": "order_field",
		},
		"config_file_release": {
			"namespace":    "namespace",
			"group":        "group",
			"file_name":    "file_name",
			"fileName":     "file_name",
			"name":         "release_name",
			"release_name": "release_name",
			"offset":       "offset",
			"limit":        "limit",
			"order_type":   "order_type",
			"order_field":  "order_field",
			"only_active":  "only_active",
		},
		"config_file_group": {
			"namespace":   "namespace",
			"group":       "name",
			"name":        "name",
			"business":    "business",
			"department":  "department",
			"offset":      "offset",
			"limit":       "limit",
			"order_type":  "order_type",
			"order_field": "order_field",
		},
		"config_file_release_history": {
			"namespace":   "namespace",
			"group":       "group",
			"name":        "file_name",
			"offset":      "offset",
			"limit":       "limit",
			"endId":       "endId",
			"end_id":      "endId",
			"order_type":  "order_type",
			"order_field": "order_field",
		},
	}
)
