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
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/utils"
	"go.uber.org/zap"
	"net/http"
	"time"
)

const (
	DefaultLongPollingTimeout = 30000 * time.Millisecond
)

func (h *HTTPServer) getConfigFile(req *restful.Request, rsp *restful.Response) {
	requestId := req.HeaderParameter("Request-Id")
	namespace := req.QueryParameter("namespace")
	group := req.QueryParameter("group")
	fileName := req.QueryParameter("fileName")

	if namespace == "" || group == "" || fileName == "" {
		err := rsp.WriteErrorString(http.StatusBadRequest, "namespace & group & fileName can not be empty")
		if err != nil {
			log.GetConfigLogger().Error("[Config][Client] write response error.", zap.Error(err))
		}
		return
	}

	//从缓存中获取配置内容
	entry, err := h.configServer.Cache().GetOrLoadIfAbsent(namespace, group, fileName)

	if err != nil {
		log.GetConfigLogger().Error("[Config][Client] get or load config file from cache error.",
			zap.String("requestId", requestId),
			zap.Error(err))

		err := rsp.WriteErrorString(http.StatusInternalServerError, "load config file error")
		if err != nil {
			log.GetConfigLogger().Error("[Config][Client] write response error.", zap.Error(err))
		}
		return
	}

	if entry.Empty {
		err := rsp.WriteErrorString(http.StatusNotFound, "config file not exist")
		if err != nil {
			log.Error("[Config][Client] write response error.", zap.Error(err))
		}
		return
	}

	//响应请求
	response := &api.ClientConfigFileInfo{
		Namespace: utils.NewStringValue(namespace),
		Group:     utils.NewStringValue(group),
		FileName:  utils.NewStringValue(fileName),
		Content:   utils.NewStringValue(entry.Content),
		Md5:       utils.NewStringValue(entry.Md5),
	}

	err = writeResponse(response, rsp)
	if err != nil {
		log.Error("[Config][Client] write get config response error.", zap.Error(err))
	}

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

	//2. 立即比对客户端md5，如果有不一致的立即响应
	watchFiles := watchConfigFileRequest.WatchFiles
	for _, watchConfigFile := range watchFiles {
		namespace := watchConfigFile.Namespace.GetValue()
		group := watchConfigFile.Group.GetValue()
		fileName := watchConfigFile.FileName.GetValue()
		clientMd5 := watchConfigFile.Md5.GetValue()

		//从缓存中获取最新的配置文件信息
		entry, err := h.configServer.Cache().GetOrLoadIfAbsent(namespace, group, fileName)

		if err != nil {
			log.GetConfigLogger().Error("[Config][Client] get or load config file from cache error.",
				zap.String("requestId", requestId),
				zap.Error(err))

			err := rsp.WriteErrorString(http.StatusInternalServerError, "load config file error")
			if err != nil {
				log.GetConfigLogger().Error("[Config][Client] write response error.", zap.Error(err))
			}
			return
		}

		if !entry.Empty && clientMd5 != entry.Md5 {
			//客户端版本落后，立刻响应
			err := writeResponse(genConfigFileChangeResponse(namespace, group, fileName, entry.Md5), rsp)
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

	h.addWatcher(clientId, watchFiles, rsp, finishChan)

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

func genConfigFileChangeResponse(namespace, group, fileName, md5 string) *api.ClientConfigFileInfo {
	return &api.ClientConfigFileInfo{
		Namespace: utils.NewStringValue(namespace),
		Group:     utils.NewStringValue(group),
		FileName:  utils.NewStringValue(fileName),
		Md5:       utils.NewStringValue(md5),
	}
}

func writeResponse(message proto.Message, rsp *restful.Response) error {
	m := jsonpb.Marshaler{Indent: " ", EmitDefaults: true}
	str, err := m.MarshalToString(message)
	if err != nil {
		return err
	}
	_, err = rsp.Write([]byte(str))
	if err != nil {
		return err
	}
	return err
}
