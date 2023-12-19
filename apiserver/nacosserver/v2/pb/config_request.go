/*
 * Copyright 1999-2020 Alibaba Group Holding Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package nacos_grpc_service

import (
	"github.com/polarismesh/specification/source/go/api/v1/config_manage"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/polarismesh/polaris/apiserver/nacosserver/model"
	"github.com/polarismesh/polaris/common/utils"
)

type ConfigListenContext struct {
	Group  string `json:"group"`
	Md5    string `json:"md5"`
	DataId string `json:"dataId"`
	Tenant string `json:"tenant"`
}

type ConfigContext struct {
	Group  string `json:"group"`
	DataId string `json:"dataId"`
	Tenant string `json:"tenant"`
}

type ConfigRequest struct {
	*Request
	Group  string `json:"group"`
	DataId string `json:"dataId"`
	Tenant string `json:"tenant"`
	Module string `json:"module"`
}

func (c *ConfigRequest) RequestMeta() interface{} {
	return c
}

func NewConfigRequest() *ConfigRequest {
	request := Request{
		RequestId: utils.NewUUID(),
		Headers:   make(map[string]string, 8),
	}
	return &ConfigRequest{
		Request: &request,
		Module:  "config",
	}
}

func (r *ConfigRequest) GetDataId() string {
	return r.DataId
}

func (r *ConfigRequest) GetGroup() string {
	return r.Group
}

func (r *ConfigRequest) GetTenant() string {
	return r.Tenant
}

// ConfigBatchListenRequest request of listening a batch of configs.
type ConfigBatchListenRequest struct {
	*ConfigRequest
	Listen               bool                  `json:"listen"`
	ConfigListenContexts []ConfigListenContext `json:"configListenContexts"`
}

func (c *ConfigBatchListenRequest) ToSpec() *config_manage.ClientWatchConfigFileRequest {
	specReq := &config_manage.ClientWatchConfigFileRequest{
		WatchFiles: make([]*config_manage.ClientConfigFileInfo, 0, len(c.ConfigListenContexts)),
	}

	for i := range c.ConfigListenContexts {
		listenCtx := c.ConfigListenContexts[i]
		specReq.WatchFiles = append(specReq.WatchFiles, &config_manage.ClientConfigFileInfo{
			Namespace: wrapperspb.String(model.ToPolarisNamespace(listenCtx.Tenant)),
			Group:     wrapperspb.String(listenCtx.Group),
			FileName:  wrapperspb.String(listenCtx.DataId),
			Md5:       wrapperspb.String(listenCtx.Md5),
		})
	}

	return specReq
}

func (c *ConfigBatchListenRequest) RequestMeta() interface{} {
	return c
}

func NewConfigBatchListenRequest() *ConfigBatchListenRequest {
	return &ConfigBatchListenRequest{
		Listen:               true,
		ConfigListenContexts: make([]ConfigListenContext, 0, 4),
		ConfigRequest:        NewConfigRequest(),
	}
}

func (r *ConfigBatchListenRequest) GetRequestType() string {
	return "ConfigBatchListenRequest"
}

type ConfigChangeNotifyRequest struct {
	*ConfigRequest
}

func NewConfigChangeNotifyRequest() *ConfigChangeNotifyRequest {
	return &ConfigChangeNotifyRequest{
		ConfigRequest: NewConfigRequest(),
	}
}

func (r *ConfigChangeNotifyRequest) GetRequestType() string {
	return "ConfigChangeNotifyRequest"
}

type ConfigQueryRequest struct {
	*ConfigRequest
	Tag string `json:"tag"`
}

func (c *ConfigQueryRequest) ToQuerySpec() *config_manage.ClientConfigFileInfo {
	return &config_manage.ClientConfigFileInfo{
		Namespace: wrapperspb.String(model.ToPolarisNamespace(c.Tenant)),
		Group:     wrapperspb.String(c.Group),
		FileName:  wrapperspb.String(c.DataId),
	}
}

func (c *ConfigQueryRequest) RequestMeta() interface{} {
	return c
}

func NewConfigQueryRequest() *ConfigQueryRequest {
	return &ConfigQueryRequest{ConfigRequest: NewConfigRequest()}
}

func (r *ConfigQueryRequest) GetRequestType() string {
	return "ConfigQueryRequest"
}

type ConfigPublishRequest struct {
	*ConfigRequest
	Content     string            `json:"content"`
	CasMd5      string            `json:"casMd5"`
	AdditionMap map[string]string `json:"additionMap"`
}

func (c *ConfigPublishRequest) ToSpec() *config_manage.ConfigFilePublishInfo {
	ret := &config_manage.ConfigFilePublishInfo{
		Namespace: wrapperspb.String(model.ToPolarisNamespace(c.Tenant)),
		Group:     wrapperspb.String(c.Group),
		FileName:  wrapperspb.String(c.DataId),
		Content:   wrapperspb.String(c.Content),
		Tags:      make([]*config_manage.ConfigFileTag, 0, len(c.AdditionMap)),
		Format:    utils.NewStringValue(utils.FileFormatText),
	}
	if val, ok := c.AdditionMap["type"]; ok {
		ret.Format = utils.NewStringValue(val)
	}

	for k, v := range c.AdditionMap {
		ret.Tags = append(ret.Tags, &config_manage.ConfigFileTag{
			Key:   wrapperspb.String(k),
			Value: wrapperspb.String(v),
		})
	}

	return ret
}

func (c *ConfigPublishRequest) RequestMeta() interface{} {
	return c
}

func NewConfigPublishRequest() *ConfigPublishRequest {
	return &ConfigPublishRequest{
		ConfigRequest: NewConfigRequest(),
		AdditionMap:   make(map[string]string),
	}
}

func (r *ConfigPublishRequest) GetRequestType() string {
	return "ConfigPublishRequest"
}

type ConfigRemoveRequest struct {
	*ConfigRequest
}

func (c *ConfigRemoveRequest) ToSpec() *config_manage.ConfigFile {
	return &config_manage.ConfigFile{
		Namespace: wrapperspb.String(model.ToPolarisNamespace(c.Tenant)),
		Group:     wrapperspb.String(c.Group),
		Name:      wrapperspb.String(c.DataId),
	}
}

func (c *ConfigRemoveRequest) RequestMeta() interface{} {
	return c
}

func NewConfigRemoveRequest() *ConfigRemoveRequest {
	return &ConfigRemoveRequest{ConfigRequest: NewConfigRequest()}
}

func (r *ConfigRemoveRequest) GetRequestType() string {
	return "ConfigRemoveRequest"
}
