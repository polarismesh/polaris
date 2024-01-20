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
	"github.com/polarismesh/polaris/common/metrics"
	"github.com/polarismesh/polaris/common/model"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/config"
	"github.com/polarismesh/polaris/plugin"
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

	var resp *config_manage.ConfigResponse
	startTime := commontime.CurrentMillisecond()
	defer func() {
		plugin.GetStatis().ReportDiscoverCall(metrics.ClientDiscoverMetric{
			Action:    nacosmodel.ActionGrpcPublishConfigFile,
			ClientIP:  meta.ConnectionID,
			Namespace: configReq.Tenant,
			Resource:  metrics.ResourceOfConfigFile(configReq.Group, configReq.DataId),
			Timestamp: startTime,
			CostTime:  commontime.CurrentMillisecond() - startTime,
			Success:   resp.GetCode().GetValue() == uint32(apimodel.Code_ExecuteSuccess),
		})
	}()

	if configReq.CasMd5 != "" {
		resp = h.configSvr.CasUpsertAndReleaseConfigFileFromClient(ctx, configReq.ToSpec())
	} else {
		resp = h.configSvr.UpsertAndReleaseConfigFileFromClient(ctx, configReq.ToSpec())
	}
	if resp.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
		nacoslog.Error("[NACOS-V2][Config] publish config file fail", zap.String("tenant", configReq.Tenant),
			utils.ZapGroup(configReq.Group), utils.ZapFileName(configReq.DataId),
			zap.Uint32("code", resp.GetCode().GetValue()), zap.String("msg", resp.GetInfo().GetValue()))
		return &nacospb.ConfigPublishResponse{
			Response: &nacospb.Response{
				Success:    false,
				ResultCode: int(nacosmodel.Response_Fail.Code),
				ErrorCode:  int(resp.GetCode().GetValue()),
				Message:    resp.GetInfo().GetValue(),
			},
		}, nil
	}

	return &nacospb.ConfigPublishResponse{
		Response: &nacospb.Response{
			Success:    true,
			ResultCode: int(nacosmodel.Response_Success.Code),
			Message:    nacosmodel.Response_Success.Desc,
		},
	}, nil
}

func (h *ConfigServer) handleGetConfigRequest(ctx context.Context, req nacospb.BaseRequest,
	meta nacospb.RequestMeta) (nacospb.BaseResponse, error) {
	configReq, ok := req.(*nacospb.ConfigQueryRequest)
	if !ok {
		return nil, remote.ErrorInvalidRequestBodyType
	}
	var rsp *nacospb.ConfigQueryResponse

	startTime := commontime.CurrentMillisecond()
	defer func() {
		plugin.GetStatis().ReportDiscoverCall(metrics.ClientDiscoverMetric{
			Action:    nacosmodel.ActionGrpcGetConfigFile,
			ClientIP:  meta.ConnectionID,
			Namespace: configReq.Tenant,
			Resource:  metrics.ResourceOfConfigFile(configReq.Group, configReq.DataId),
			Timestamp: startTime,
			CostTime:  commontime.CurrentMillisecond() - startTime,
			Revision:  rsp.Md5,
			Success:   rsp.Success,
		})
	}()

	queryReq := configReq.ToQuerySpec()
	queryResp := h.configSvr.GetConfigFileWithCache(ctx, queryReq)
	if queryResp.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
		nacoslog.Error("[NACOS-V2][Config] query config file fail", zap.String("tenant", configReq.Tenant),
			utils.ZapNamespace(queryReq.GetNamespace().GetValue()), utils.ZapGroup(configReq.Group),
			utils.ZapFileName(configReq.DataId), zap.Uint32("code", queryResp.GetCode().GetValue()),
			zap.String("msg", queryResp.GetInfo().GetValue()))
		switch queryResp.GetCode().GetValue() {
		case uint32(apimodel.Code_NotFoundResource):
			rsp = &nacospb.ConfigQueryResponse{
				Response: &nacospb.Response{
					ResultCode: int(nacosmodel.Response_Fail.Code),
					ErrorCode:  ErrorConfigNotFound,
					Message:    "config data not exist",
				},
			}
			return rsp, nil
		default:
			rsp = &nacospb.ConfigQueryResponse{
				Response: &nacospb.Response{
					ResultCode: int(nacosmodel.Response_Fail.Code),
					ErrorCode:  int(queryResp.GetCode().GetValue()),
					Message:    queryResp.GetInfo().GetValue(),
				},
			}
			return rsp, nil
		}
	}

	viewRelease := queryResp.GetConfigFile()

	rsp = &nacospb.ConfigQueryResponse{
		Response: &nacospb.Response{
			ResultCode: int(nacosmodel.Response_Success.Code),
			Success:    true,
		},
		Content:      viewRelease.GetContent().GetValue(),
		Md5:          viewRelease.GetMd5().GetValue(),
		LastModified: stringToTimestamp(viewRelease.GetReleaseTime().GetValue()),
	}
	return rsp, nil
}

func (h *ConfigServer) handleDeleteConfigRequest(ctx context.Context, req nacospb.BaseRequest,
	meta nacospb.RequestMeta) (nacospb.BaseResponse, error) {
	configReq, ok := req.(*nacospb.ConfigRemoveRequest)
	if !ok {
		return nil, remote.ErrorInvalidRequestBodyType
	}
	delResp := h.configSvr.DeleteConfigFileFromClient(ctx, configReq.ToSpec())
	if delResp.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
		nacoslog.Error("[NACOS-V2][Config] delete config file fail", zap.String("tenant", configReq.Tenant),
			utils.ZapGroup(configReq.Group), utils.ZapFileName(configReq.DataId),
			zap.Uint32("code", delResp.GetCode().GetValue()), zap.String("msg", delResp.GetInfo().GetValue()))
		return &nacospb.ConfigRemoveResponse{
			Response: &nacospb.Response{
				Success:    false,
				ResultCode: int(nacosmodel.Response_Fail.Code),
				ErrorCode:  int(delResp.GetCode().GetValue()),
				Message:    delResp.GetInfo().GetValue(),
			},
		}, nil
	}

	return &nacospb.ConfigRemoveResponse{
		Response: &nacospb.Response{
			Success:    true,
			ResultCode: int(nacosmodel.Response_Success.Code),
			Message:    nacosmodel.Response_Success.Desc,
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
		watchCtx := configSvr.WatchCenter().AddWatcher(clientId, specReq.GetWatchFiles(), h.BuildGrpcWatchCtx(ctx))
		for i := range specReq.GetWatchFiles() {
			item := specReq.GetWatchFiles()[i]
			namespace := item.GetNamespace().GetValue()
			group := item.GetGroup().GetValue()
			dataId := item.GetFileName().GetValue()
			mdval := item.GetMd5().GetValue()

			var active *model.ConfigFileRelease
			var match bool
			if betaActive := h.cacheSvr.ConfigFile().GetActiveGrayRelease(namespace, group, dataId); betaActive != nil {
				match = h.cacheSvr.Gray().HitGrayRule(model.GetGrayConfigRealseKey(betaActive.SimpleConfigFileRelease), watchCtx.ClientLabels())
				active = betaActive
			}
			if !match {
				active = h.cacheSvr.ConfigFile().GetActiveRelease(namespace, group, dataId)
			}

			// 如果 client 过来的 MD5 是一个空字符串
			if (active == nil && mdval != "") || (active != nil && active.Md5 != mdval) {
				listenResp.ChangedConfigs = append(listenResp.ChangedConfigs, nacospb.ConfigContext{
					Tenant: nacosmodel.ToNacosConfigNamespace(namespace),
					Group:  group,
					DataId: dataId,
				})
			}
		}
	} else {
		watchCtx, ok := configSvr.WatchCenter().GetWatchContext(clientId)
		if ok {
			for i := range specReq.GetWatchFiles() {
				item := specReq.GetWatchFiles()[i]
				watchCtx.RemoveInterest(item)
			}
		}
	}
	return listenResp, nil
}

// BuildGrpcWatchCtx .
func (h *ConfigServer) BuildGrpcWatchCtx(ctx context.Context) config.WatchContextFactory {
	labels := map[string]string{}
	labels[model.ClientLabel_IP] = utils.ParseClientIP(ctx)

	return func(clientId string, matcher config.BetaReleaseMatcher) config.WatchContext {
		watchCtx := &StreamWatchContext{
			clientId:         clientId,
			connMgr:          h.connMgr,
			labels:           labels,
			watchConfigFiles: utils.NewSyncMap[string, *config_manage.ClientConfigFileInfo](),
			betaMatcher: func(clientLabels map[string]string, event *model.SimpleConfigFileRelease) bool {
				return h.cacheSvr.Gray().HitGrayRule(model.GetGrayConfigRealseKey(event), clientLabels)
			},
		}
		return watchCtx
	}
}

// stringToTimestamp Convert string to timestamp
func stringToTimestamp(val string) int64 {
	lastModified, _ := time.Parse("2006-01-02 15:04:05", val)
	return lastModified.UnixMilli()
}
