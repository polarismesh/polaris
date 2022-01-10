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

package httpserver

import (
	"github.com/emicklei/go-restful"
	"github.com/google/uuid"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/utils"
	"go.uber.org/zap"
	"strconv"
	"time"
)

const (
	DefaultLongPollingTimeout = 30000 * time.Millisecond
)

func (h *HTTPServer) getConfigFile(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	requestId := handler.HeaderParameter("Request-Id")
	namespace := handler.QueryParameter("namespace")
	group := handler.QueryParameter("group")
	fileName := handler.QueryParameter("fileName")
	clientVersionStr := handler.QueryParameter("version")

	if namespace == "" || group == "" || fileName == "" {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(api.BadRequest, "namespace & group & fileName can not be empty"))
		return
	}

	//从缓存中获取配置内容
	entry, err := h.configServer.Cache().GetOrLoadIfAbsent(namespace, group, fileName)

	if err != nil {
		log.GetConfigLogger().Error("[Config][Client] get or load config file from cache error.",
			zap.String("requestId", requestId),
			zap.Error(err))

		handler.WriteHeaderAndProto(api.NewResponseWithMsg(api.ExecuteException, "load config file error"))
		return
	}

	if entry.Empty {
		handler.WriteHeaderAndProto(api.NewResponse(api.NotFoundResourceConfigFile))
		return
	}

	clientVersion, _ := strconv.ParseUint(clientVersionStr, 10, 64)
	//客户端版本号大于服务端版本号，服务端需要重新加载缓存
	if clientVersion > entry.Version {
		entry, err = h.configServer.Cache().ReLoad(namespace, group, fileName)
		if err != nil {
			log.GetConfigLogger().Error("[Config][Client] reload config file error.",
				zap.String("requestId", requestId),
				zap.Error(err))

			handler.WriteHeaderAndProto(api.NewResponseWithMsg(api.ExecuteException, "reload config file error"))
			return
		}
	}

	//响应请求
	handler.WriteHeaderAndProto(genConfigFileResponse(namespace, group, fileName, entry.Content, entry.Version))

	log.GetConfigLogger().Info("[Config][Client] client get config file success.",
		zap.String("requestId", requestId),
		zap.String("client", req.Request.RemoteAddr),
		zap.String("file", fileName))
}

func (h *HTTPServer) WatchConfigFile(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	requestId := req.HeaderParameter("Request-Id")
	clientAddr := req.Request.RemoteAddr

	log.GetConfigLogger().Debug("[Config][Client] received client listener request.",
		zap.String("requestId", requestId),
		zap.String("client", clientAddr))

	//1. 解析出客户端监听的配置文件列表
	watchConfigFileRequest := &api.ClientWatchConfigFileRequest{}
	_, err := handler.Parse(watchConfigFileRequest)
	if err != nil {
		log.GetConfigLogger().Warn("[Config][Client] parse client watch request error",
			zap.String("requestId", requestId),
			zap.String("client", req.Request.RemoteAddr))

		handler.WriteHeaderAndProto(api.NewResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	if len(watchConfigFileRequest.WatchFiles) == 0 {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(api.InvalidWatchConfigFileFormat, err.Error()))
		return
	}

	//2. 立即比对客户端版本号，如果有落后，立即响应
	watchFiles := watchConfigFileRequest.WatchFiles
	for _, watchConfigFile := range watchFiles {
		namespace := watchConfigFile.Namespace.GetValue()
		group := watchConfigFile.Group.GetValue()
		fileName := watchConfigFile.FileName.GetValue()
		clientVersion := watchConfigFile.Version.GetValue()

		//从缓存中获取最新的配置文件信息
		entry, err := h.configServer.Cache().GetOrLoadIfAbsent(namespace, group, fileName)

		if err != nil {
			log.GetConfigLogger().Error("[Config][Client] get or load config file from cache error.",
				zap.String("requestId", requestId),
				zap.String("fileName", fileName),
				zap.Error(err))

			handler.WriteHeaderAndProto(api.NewResponseWithMsg(api.ExecuteException, "get config file error"))
			return
		}

		if !entry.Empty && clientVersion < entry.Version {
			//客户端版本落后，立刻响应，响应体里不包含 content，需要客户端重新拉取一次最新的
			handler.WriteHeaderAndProto(genConfigFileResponse(namespace, group, fileName, "", entry.Version))
			if err != nil {
				log.GetConfigLogger().Error("[Config][Client] write listener response error.", zap.Error(err))
			}
			return
		}
	}

	//3. 监听配置变更，hold 请求 30s，30s 内如果有配置发布，则响应请求
	id, _ := uuid.NewUUID()
	clientId := clientAddr + "@" + id.String()[0:8]
	finishChan := make(chan interface{})

	h.addWatcher(clientId, watchFiles, handler, finishChan)

	timer := time.NewTimer(DefaultLongPollingTimeout)
	for {
		select {
		case <-timer.C:
			h.removeWatcher(clientId, watchFiles)
			return
		case <-finishChan:
			h.removeWatcher(clientId, watchFiles)
			return
		}
	}
}

func genConfigFileResponse(namespace, group, fileName, content string, version uint64) *api.ConfigClientResponse {
	configFile := &api.ClientConfigFileInfo{
		Namespace: utils.NewStringValue(namespace),
		Group:     utils.NewStringValue(group),
		FileName:  utils.NewStringValue(fileName),
		Content:   utils.NewStringValue(content),
		Version:   utils.NewUInt64Value(version),
	}
	return api.NewConfigClientResponse(api.ExecuteSuccess, configFile)
}
