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
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/emicklei/go-restful/v3"
	"github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/apiserver/nacosserver/model"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/metrics"
	commonmodel "github.com/polarismesh/polaris/common/model"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/config"
	"github.com/polarismesh/polaris/plugin"
)

func (n *ConfigServer) handlePublishConfig(ctx context.Context, req *model.ConfigFile) (bool, error) {
	var resp *config_manage.ConfigResponse
	if req.CasMd5 != "" {
		resp = n.configSvr.CasUpsertAndReleaseConfigFileFromClient(ctx, req.ToSpecConfigFile())
	} else {
		resp = n.configSvr.UpsertAndReleaseConfigFileFromClient(ctx, req.ToSpecConfigFile())
	}

	if resp.GetCode().GetValue() == uint32(apimodel.Code_ExecuteSuccess) {
		return true, nil
	}
	nacoslog.Error("[NACOS-V1][Config] publish config file fail",
		zap.Uint32("code", resp.GetCode().GetValue()), zap.String("msg", resp.GetInfo().GetValue()))
	return false, &model.NacosError{
		ErrCode: int32(model.ExceptionCode_ServerError),
		ErrMsg:  resp.GetInfo().GetValue(),
	}
}

func (n *ConfigServer) handleDeleteConfig(ctx context.Context, req *model.ConfigFile) (bool, error) {
	resp := n.configSvr.DeleteConfigFileFromClient(ctx, req.ToDeleteSpec())
	if resp.GetCode().GetValue() == uint32(apimodel.Code_ExecuteSuccess) {
		return true, nil
	}
	nacoslog.Error("[NACOS-V1][Config] delete config file fail",
		zap.Uint32("code", resp.GetCode().GetValue()), zap.String("msg", resp.GetInfo().GetValue()))
	return false, &model.NacosError{
		ErrCode: int32(model.ExceptionCode_ServerError),
		ErrMsg:  resp.GetInfo().GetValue(),
	}
}

func (n *ConfigServer) handleGetConfig(ctx context.Context, req *model.ConfigFile, rsp *restful.Response) (string, error) {
	var queryResp *config_manage.ConfigClientResponse
	startTime := commontime.CurrentMillisecond()
	defer func() {
		plugin.GetStatis().ReportDiscoverCall(metrics.ClientDiscoverMetric{
			Action:    model.ActionGetConfigFile,
			ClientIP:  utils.ParseClientAddress(ctx),
			Namespace: req.Namespace,
			Resource:  metrics.ResourceOfConfigFile(req.Group, req.DataId),
			Timestamp: startTime,
			CostTime:  commontime.CurrentMillisecond() - startTime,
			Revision:  queryResp.GetConfigFile().GetMd5().GetValue(),
			Success:   queryResp.GetCode().GetValue() > uint32(apimodel.Code_DataNoChange),
		})
	}()

	queryResp = n.configSvr.GetConfigFileWithCache(ctx, req.ToQuerySpec())
	if queryResp.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
		nacoslog.Error("[NACOS-V1][Config] query config file fail",
			zap.Uint32("code", queryResp.GetCode().GetValue()), zap.String("msg", queryResp.GetInfo().GetValue()))
		switch queryResp.GetCode().GetValue() {
		case uint32(apimodel.Code_NotFoundResource):
			return "", &model.NacosError{
				ErrCode: int32(http.StatusNotFound),
				ErrMsg:  "config data not exist",
			}
		default:
			return "", &model.NacosError{
				ErrCode: int32(model.ExceptionCode_ServerError),
				ErrMsg:  queryResp.GetInfo().GetValue(),
			}
		}
	}

	viewRelease := queryResp.GetConfigFile()
	disableCache(rsp)
	rsp.AddHeader(model.HeaderLastModified, viewRelease.GetReleaseTime().GetValue())
	rsp.AddHeader(model.HeaderContentMD5, viewRelease.GetMd5().GetValue())

	return viewRelease.GetContent().GetValue(), nil
}

func (n *ConfigServer) handleWatch(ctx context.Context, listenCtx *model.ConfigWatchContext,
	rsp *restful.Response) {

	specWatchReq := listenCtx.ToSpecWatch()
	timeout, ok := listenCtx.IsSupportLongPolling()
	if !ok {
		changeKeys := n.diffChangeFiles(ctx, specWatchReq)
		oldResult := md5OldResult(changeKeys)
		newResult := md5ResultString(changeKeys)

		rsp.WriteHeader(http.StatusOK)
		disableCache(rsp)
		rsp.AddHeader(model.HeaderProbeModifyResponse, oldResult)
		rsp.AddHeader(model.HeaderProbeModifyResponseNew, newResult)
		listenCtx.Request.SetAttribute(model.HeaderContent, newResult)
		return
	}
	if changeKeys := n.diffChangeFiles(ctx, specWatchReq); len(changeKeys) > 0 {
		newResult := md5ResultString(changeKeys)
		nacoslog.Info("[NACOS-V1][Config] client quick compare file result.", zap.String("result", newResult))
		rsp.WriteHeader(http.StatusOK)
		disableCache(rsp)
		_, _ = rsp.Write([]byte(newResult))
		return
	}
	if listenCtx.IsNoHangUp() {
		// 该场景只会在 nacos-client 第一次发起订阅任务的时候
		// /com/alibaba/nacos/nacos-client/1.4.6/nacos-client-1.4.6-sources.jar!/com/alibaba/nacos/client/config/impl/ClientWorker.java:368
		nacoslog.Info("[NACOS-V1][Config] client set listen no hangup, quick return")
		return
	}
	clientId := utils.ParseClientAddress(ctx) + "@" + utils.NewUUID()[0:8]
	configSvr := n.originConfigSvr.(*config.Server)
	watchCtx := configSvr.WatchCenter().AddWatcher(clientId, specWatchReq.GetWatchFiles(),
		n.BuildTimeoutWatchCtx(ctx, timeout))
	nacoslog.Info("[NACOS-V1][Config] client start waitting server send notify message")
	notifyRet := (watchCtx.(*LongPollWatchContext)).GetNotifieResult()
	notifyCode := notifyRet.GetCode().GetValue()
	if notifyCode != uint32(apimodel.Code_ExecuteSuccess) && notifyCode != uint32(apimodel.Code_DataNoChange) {
		nacoslog.Error("[NACOS-V1][Config] notify client config change",
			zap.String("remote", listenCtx.Request.Request.RemoteAddr), zap.Uint32("code", notifyCode),
			zap.String("msg", notifyRet.GetInfo().GetValue()))
		rsp.WriteHeader(api.CalcCode(notifyRet))
		return
	}

	var changeKeys []*model.ConfigListenItem
	if notifyCode == uint32(apimodel.Code_DataNoChange) {
		// 按照 Nacos 原本的设计，只有 WatchClient 超时后才会再次全部 diff 比较
		changeKeys = n.diffChangeFiles(ctx, specWatchReq)
	} else {
		// 如果收到一个事件变化，就立即通知这个文件的变化信息
		changeKeys = []*model.ConfigListenItem{
			{
				Tenant: notifyRet.GetConfigFile().GetNamespace().GetValue(),
				Group:  notifyRet.GetConfigFile().GetGroup().GetValue(),
				DataId: notifyRet.GetConfigFile().GetFileName().GetValue(),
			},
		}
	}
	if len(changeKeys) == 0 {
		nacoslog.Debug("[NACOS-V1][Config] client receive empty watch result.", zap.Any("ret", notifyRet))
		rsp.WriteHeader(http.StatusOK)
		return
	}
	newResult := md5ResultString(changeKeys)
	nacoslog.Info("[NACOS-V1][Config] client receive watch result.", zap.String("result", newResult))
	rsp.WriteHeader(http.StatusOK)
	disableCache(rsp)
	_, _ = rsp.Write([]byte(newResult))
	return
}

func (n *ConfigServer) diffChangeFiles(ctx context.Context,
	listenCtx *config_manage.ClientWatchConfigFileRequest) []*model.ConfigListenItem {
	clientLabels := map[string]string{
		commonmodel.ClientLabel_IP: utils.ParseClientIP(ctx),
	}
	changeKeys := make([]*model.ConfigListenItem, 0, 4)
	// quick get file and compare
	for _, item := range listenCtx.WatchFiles {
		namespace := item.GetNamespace().GetValue()
		group := item.GetGroup().GetValue()
		dataId := item.GetFileName().GetValue()
		mdval := item.GetMd5().GetValue()

		if beta := n.cacheSvr.ConfigFile().GetActiveGrayRelease(namespace, group, dataId); beta != nil {
			if n.cacheSvr.Gray().HitGrayRule(beta.FileKey(), clientLabels) {
				changeKeys = append(changeKeys, &model.ConfigListenItem{
					Tenant: model.ToNacosConfigNamespace(beta.Namespace),
					Group:  beta.Group,
					DataId: dataId,
				})
				continue
			}
		}

		active := n.cacheSvr.ConfigFile().GetActiveRelease(namespace, group, dataId)
		if (active == nil && mdval != "") || (active != nil && active.Md5 != mdval) {
			changeKeys = append(changeKeys, &model.ConfigListenItem{
				Tenant: model.ToNacosConfigNamespace(namespace),
				Group:  group,
				DataId: dataId,
			})
		}
	}
	return changeKeys
}

func (n *ConfigServer) BuildTimeoutWatchCtx(ctx context.Context, watchTimeOut time.Duration) config.WatchContextFactory {
	labels := map[string]string{}
	labels[commonmodel.ClientLabel_IP] = utils.ParseClientIP(ctx)

	return func(clientId string, matcher config.BetaReleaseMatcher) config.WatchContext {
		watchCtx := &LongPollWatchContext{
			clientId:         clientId,
			labels:           labels,
			finishTime:       time.Now().Add(watchTimeOut),
			finishChan:       make(chan *config_manage.ConfigClientResponse),
			watchConfigFiles: map[string]*config_manage.ClientConfigFileInfo{},
			betaMatcher: func(clientLabels map[string]string, event *commonmodel.SimpleConfigFileRelease) bool {
				return n.cacheSvr.Gray().HitGrayRule(commonmodel.GetGrayConfigRealseKey(event), clientLabels)
			},
		}
		return watchCtx
	}
}

func md5OldResult(items []*model.ConfigListenItem) string {
	sb := strings.Builder{}
	for i := range items {
		item := items[i]
		sb.WriteString(item.DataId)
		sb.WriteString(":")
		sb.WriteString(item.Group)
		sb.WriteString(";")
	}
	return sb.String()
}

func md5ResultString(items []*model.ConfigListenItem) string {
	if len(items) == 0 {
		return url.QueryEscape("")
	}
	sb := strings.Builder{}
	for i := range items {
		item := items[i]
		sb.WriteString(item.DataId)
		sb.WriteRune(model.WordSeparatorRune)
		sb.WriteString(item.Group)
		tenant := model.ToNacosConfigNamespace(item.Tenant)
		if len(tenant) != 0 {
			sb.WriteRune(model.WordSeparatorRune)
			sb.WriteString(model.ToNacosConfigNamespace(tenant))
		}
		sb.WriteRune(model.LineSeparatorRune)
	}
	return url.QueryEscape(sb.String())
}

func disableCache(rsp *restful.Response) {
	rsp.AddHeader(model.HeaderExpires, "0")
	rsp.AddHeader(model.HeaderPragma, "no-cache")
	rsp.AddHeader(model.HeaderCacheControl, "no-cache,no-store")
}
