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
	"time"

	"github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"go.uber.org/zap"

	nacosmodel "github.com/polarismesh/polaris/apiserver/nacosserver/model"
	nacospb "github.com/polarismesh/polaris/apiserver/nacosserver/v2/pb"
	"github.com/polarismesh/polaris/apiserver/nacosserver/v2/remote"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/config"
)

const (
	ErrorConfigNotFound      = 300
	ErrorConfigQueryConflict = 400
)

func (h *ConfigServer) handlePublishConfigRequest(ctx context.Context, req nacospb.BaseRequest,
	meta nacospb.RequestMeta) (nacospb.BaseResponse, error) {
	configReq, ok := req.(*nacospb.ConfigPublishRequest)
	if !ok {
		return nil, remote.ErrorInvalidRequestBodyType
	}

	resp := h.configSvr.UpsertAndReleaseConfigFileFromClient(ctx, configReq.ToSpec())
	if resp.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
		nacoslog.Error("[NACOS-V2][Config] publish config file fail",
			zap.Uint32("code", resp.GetCode().GetValue()), zap.String("msg", resp.GetInfo().GetValue()))
		return &nacospb.ConfigPublishResponse{
			&nacospb.Response{
				Success:   false,
				ErrorCode: int(nacosmodel.Response_Fail.Code),
				Message:   resp.GetInfo().GetValue(),
			},
		}, nil
	}

	return &nacospb.ConfigPublishResponse{
		&nacospb.Response{
			Success:   true,
			ErrorCode: int(nacosmodel.Response_Success.Code),
			Message:   nacosmodel.Response_Success.Desc,
		},
	}, nil
}

func (h *ConfigServer) handleGetConfigRequest(ctx context.Context, req nacospb.BaseRequest,
	meta nacospb.RequestMeta) (nacospb.BaseResponse, error) {
	configReq, ok := req.(*nacospb.ConfigQueryRequest)
	if !ok {
		return nil, remote.ErrorInvalidRequestBodyType
	}

	queryResp := h.configSvr.GetConfigFileForClient(ctx, configReq.ToQuerySpec())
	if queryResp.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
		nacoslog.Error("[NACOS-V2][Config] query config file fail",
			zap.Uint32("code", queryResp.GetCode().GetValue()), zap.String("msg", queryResp.GetInfo().GetValue()))
		switch queryResp.GetCode().GetValue() {
		case uint32(apimodel.Code_NotFoundResource):
			return nil, &nacosmodel.NacosError{
				ErrCode: ErrorConfigNotFound,
				ErrMsg:  "config data not exist",
			}
		default:
			return nil, &nacosmodel.NacosError{
				ErrCode: int32(nacosmodel.ExceptionCode_ServerError),
				ErrMsg:  queryResp.GetInfo().GetValue(),
			}
		}
	}

	viewRelease := queryResp.GetConfigFile()

	ret := &nacospb.ConfigQueryResponse{
		Response: &nacospb.Response{
			ResultCode: int(nacosmodel.Response_Success.Code),
			Success:    true,
		},
		Content:      viewRelease.GetContent().GetValue(),
		Md5:          viewRelease.GetMd5().GetValue(),
		LastModified: stringToTimestamp(viewRelease.GetReleaseTime().GetValue()),
	}
	return ret, nil
}

func (h *ConfigServer) handleDeleteConfigRequest(ctx context.Context, req nacospb.BaseRequest,
	meta nacospb.RequestMeta) (nacospb.BaseResponse, error) {
	configReq, ok := req.(*nacospb.ConfigRemoveRequest)
	if !ok {
		return nil, remote.ErrorInvalidRequestBodyType
	}
	delResp := h.configSvr.DeleteConfigFileFromClient(ctx, configReq.ToSpec())
	if delResp.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
		nacoslog.Error("[NACOS-V2][Config] delete config file fail",
			zap.Uint32("code", delResp.GetCode().GetValue()), zap.String("msg", delResp.GetInfo().GetValue()))
		return &nacospb.ConfigRemoveResponse{
			Response: &nacospb.Response{
				Success:   false,
				ErrorCode: int(nacosmodel.Response_Fail.Code),
				Message:   delResp.GetInfo().GetValue(),
			},
		}, nil
	}

	return &nacospb.ConfigRemoveResponse{
		Response: &nacospb.Response{
			Success:   true,
			ErrorCode: int(nacosmodel.Response_Success.Code),
			Message:   nacosmodel.Response_Success.Desc,
		},
	}, nil
}

func (h *ConfigServer) handleWatchConfigRequest(ctx context.Context, req nacospb.BaseRequest,
	meta nacospb.RequestMeta) (nacospb.BaseResponse, error) {
	watchReq, ok := req.(*nacospb.ConfigBatchListenRequest)
	if !ok {
		return nil, remote.ErrorInvalidRequestBodyType
	}
	configSvr := h.originConfigSvr.(*config.Server)

	listenResp := nacospb.NewConfigChangeBatchListenResponse()
	clientId := meta.ConnectionID
	specReq := watchReq.ToSpec()
	if watchReq.Listen {
		for i := range specReq.GetWatchFiles() {
			item := specReq.GetWatchFiles()[i]

			namespace := item.GetNamespace().GetValue()
			group := item.GetGroup().GetValue()
			dataId := item.GetFileName().GetValue()

			active := h.cacheSvr.ConfigFile().GetActiveRelease(namespace, group, dataId)
			if active == nil || active.Md5 != item.GetMd5().GetValue() {
				listenResp.ChangedConfigs = append(listenResp.ChangedConfigs, nacospb.ConfigContext{
					Tenant: nacosmodel.ToNacosConfigNamespace(namespace),
					Group:  group,
					DataId: dataId,
				})
			}
		}
		configSvr.WatchCenter().AddWatcher(clientId, specReq.GetWatchFiles(), h.BuildGrpcWatchCtx())
	} else {
		configSvr.WatchCenter().RemoveWatcher(clientId, specReq.GetWatchFiles())
	}
	return listenResp, nil
}

func (h *ConfigServer) BuildGrpcWatchCtx() config.WatchContextFactory {
	return func(clientId string, watchFiles []*config_manage.ClientConfigFileInfo) config.WatchContext {
		watchCtx := &StreamWatchContext{
			clientId:          clientId,
			connectionManager: h.connectionManager,
			watchConfigFiles:  utils.NewSyncMap[string, *config_manage.ClientConfigFileInfo](),
		}
		for i := range watchFiles {
			watchCtx.AppendInterest(watchFiles[i])
		}
		return watchCtx
	}
}

// Time2String Convert time.Time to string time
func stringToTimestamp(val string) int64 {
	lastModified, _ := time.Parse("2006-01-02 15:04:05", val)
	return lastModified.UnixMilli()
}
