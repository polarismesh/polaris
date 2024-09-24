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

package config

import (
	"context"
	"fmt"
	"io"
	"strconv"

	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/wrapperspb"

	api "github.com/polarismesh/polaris/common/api/v1"
	commonlog "github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/metrics"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
)

var (
	accesslog = commonlog.GetScopeOrDefaultByName(commonlog.APIServerLoggerName)
)

// GetConfigFile 拉取配置
func (g *ConfigGRPCServer) GetConfigFile(ctx context.Context,
	req *apiconfig.ClientConfigFileInfo) (*apiconfig.ConfigClientResponse, error) {
	ctx = utils.ConvertGRPCContext(ctx)

	startTime := commontime.CurrentMillisecond()
	var ret *apiconfig.ConfigClientResponse
	defer func() {
		plugin.GetStatis().ReportDiscoverCall(metrics.ClientDiscoverMetric{
			Action:    metrics.ActionGetConfigFile,
			ClientIP:  utils.ParseClientAddress(ctx),
			Namespace: req.GetNamespace().GetValue(),
			Resource:  metrics.ResourceOfConfigFile(req.GetGroup().GetValue(), req.GetFileName().GetValue()),
			Timestamp: startTime,
			CostTime:  commontime.CurrentMillisecond() - startTime,
			Revision:  strconv.FormatUint(ret.GetConfigFile().GetVersion().GetValue(), 10),
			Success:   ret.GetCode().GetValue() > uint32(apimodel.Code_DataNoChange),
		})
	}()
	ret = g.configServer.GetConfigFileWithCache(ctx, req)
	return ret, nil
}

// CreateConfigFile 创建或更新配置
func (g *ConfigGRPCServer) CreateConfigFile(ctx context.Context,
	configFile *apiconfig.ConfigFile) (*apiconfig.ConfigClientResponse, error) {
	ctx = utils.ConvertGRPCContext(ctx)
	response := g.configServer.CreateConfigFileFromClient(ctx, configFile)
	return response, nil
}

// UpdateConfigFile 创建或更新配置
func (g *ConfigGRPCServer) UpdateConfigFile(ctx context.Context,
	configFile *apiconfig.ConfigFile) (*apiconfig.ConfigClientResponse, error) {
	ctx = utils.ConvertGRPCContext(ctx)
	response := g.configServer.UpdateConfigFileFromClient(ctx, configFile)
	return response, nil
}

// PublishConfigFile 发布配置
func (g *ConfigGRPCServer) PublishConfigFile(ctx context.Context,
	configFile *apiconfig.ConfigFileRelease) (*apiconfig.ConfigClientResponse, error) {
	ctx = utils.ConvertGRPCContext(ctx)
	response := g.configServer.PublishConfigFileFromClient(ctx, configFile)
	return response, nil
}

func (g *ConfigGRPCServer) UpsertAndPublishConfigFile(ctx context.Context,
	req *apiconfig.ConfigFilePublishInfo) (*apiconfig.ConfigClientResponse, error) {
	ctx = utils.ConvertGRPCContext(ctx)
	response := g.configServer.CasUpsertAndReleaseConfigFileFromClient(ctx, req)
	return &apiconfig.ConfigClientResponse{
		Code: response.Code,
		Info: response.Info,
		ConfigFile: &apiconfig.ClientConfigFileInfo{
			Namespace: req.Namespace,
			Group:     req.Group,
			FileName:  req.FileName,
		},
	}, nil
}

// WatchConfigFiles 订阅配置变更
func (g *ConfigGRPCServer) WatchConfigFiles(ctx context.Context,
	request *apiconfig.ClientWatchConfigFileRequest) (*apiconfig.ConfigClientResponse, error) {
	ctx = utils.ConvertGRPCContext(ctx)

	// 阻塞等待响应
	callback, err := g.configServer.LongPullWatchFile(ctx, request)
	if err != nil {
		return nil, err
	}
	return callback(), nil
}

func (g *ConfigGRPCServer) GetConfigFileMetadataList(ctx context.Context,
	req *apiconfig.ConfigFileGroupRequest) (*apiconfig.ConfigClientListResponse, error) {

	startTime := commontime.CurrentMillisecond()
	var ret *apiconfig.ConfigClientListResponse
	defer func() {
		plugin.GetStatis().ReportDiscoverCall(metrics.ClientDiscoverMetric{
			Action:    metrics.ActionListConfigFiles,
			ClientIP:  utils.ParseClientAddress(ctx),
			Namespace: req.GetConfigFileGroup().GetNamespace().GetValue(),
			Resource:  metrics.ResourceOfConfigFileList(req.GetConfigFileGroup().GetName().GetValue()),
			Timestamp: startTime,
			CostTime:  commontime.CurrentMillisecond() - startTime,
			Revision:  ret.GetRevision().GetValue(),
			Success:   ret.GetCode().GetValue() > uint32(apimodel.Code_DataNoChange),
		})
	}()

	ctx = utils.ConvertGRPCContext(ctx)
	ret = g.configServer.GetConfigFileNamesWithCache(ctx, req)
	return ret, nil
}

func (g *ConfigGRPCServer) Discover(svr apiconfig.PolarisConfigGRPC_DiscoverServer) error {
	ctx := utils.ConvertGRPCContext(svr.Context())
	clientIP, _ := ctx.Value(utils.StringContext("client-ip")).(string)
	clientAddress, _ := ctx.Value(utils.StringContext("client-address")).(string)
	requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)
	userAgent, _ := ctx.Value(utils.StringContext("user-agent")).(string)
	method, _ := grpc.MethodFromServerStream(svr)

	for {
		in, err := svr.Recv()
		if err != nil {
			if io.EOF == err {
				return nil
			}
			return err
		}

		msg := fmt.Sprintf("receive grpc discover request: %s", in.String())
		accesslog.Info(msg,
			zap.String("type", apiconfig.ConfigDiscoverRequest_ConfigDiscoverRequestType_name[int32(in.Type)]),
			zap.String("client-address", clientAddress),
			zap.String("user-agent", userAgent),
			utils.ZapRequestID(requestID),
		)

		// 是否允许访问
		if ok := g.allowAccess(method); !ok {
			resp := api.NewConfigDiscoverResponse(apimodel.Code_ClientAPINotOpen)
			if sendErr := svr.Send(resp); sendErr != nil {
				return sendErr
			}
			continue
		}

		// stream模式，需要对每个包进行检测
		if code := g.enterRateLimit(clientIP, method); code != uint32(apimodel.Code_ExecuteSuccess) {
			resp := api.NewConfigDiscoverResponse(apimodel.Code(code))
			if err = svr.Send(resp); err != nil {
				return err
			}
			continue
		}

		var out *apiconfig.ConfigDiscoverResponse
		var action string
		startTime := commontime.CurrentMillisecond()
		defer func() {
			plugin.GetStatis().ReportDiscoverCall(metrics.ClientDiscoverMetric{
				Action:    action,
				ClientIP:  utils.ParseClientAddress(ctx),
				Namespace: in.GetConfigFile().GetNamespace().GetValue(),
				Resource:  metrics.ResourceOfConfigFile(in.GetConfigFile().GetGroup().GetValue(), in.GetConfigFile().GetFileName().GetValue()),
				Timestamp: startTime,
				CostTime:  commontime.CurrentMillisecond() - startTime,
				Revision:  out.GetRevision(),
				Success:   out.GetCode() > uint32(apimodel.Code_DataNoChange),
			})
		}()

		switch in.Type {
		case apiconfig.ConfigDiscoverRequest_CONFIG_FILE:
			action = metrics.ActionGetConfigFile
			ret := g.configServer.GetConfigFileWithCache(ctx, &apiconfig.ClientConfigFileInfo{})
			out = api.NewConfigDiscoverResponse(apimodel.Code(ret.GetCode().GetValue()))
			out.ConfigFile = ret.GetConfigFile()
			out.Type = apiconfig.ConfigDiscoverResponse_CONFIG_FILE
			out.Revision = strconv.Itoa(int(out.GetConfigFile().GetVersion().GetValue()))
		case apiconfig.ConfigDiscoverRequest_CONFIG_FILE_Names:
			action = metrics.ActionListConfigFiles
			ret := g.configServer.GetConfigFileNamesWithCache(ctx, &apiconfig.ConfigFileGroupRequest{
				Revision: wrapperspb.String(in.GetRevision()),
				ConfigFileGroup: &apiconfig.ConfigFileGroup{
					Namespace: in.GetConfigFile().GetNamespace(),
					Name:      in.GetConfigFile().GetGroup(),
				},
			})
			out = api.NewConfigDiscoverResponse(apimodel.Code(ret.GetCode().GetValue()))
			out.ConfigFileNames = ret.GetConfigFileInfos()
			out.Type = apiconfig.ConfigDiscoverResponse_CONFIG_FILE_Names
			out.Revision = ret.GetRevision().GetValue()
		case apiconfig.ConfigDiscoverRequest_CONFIG_FILE_GROUPS:
			action = metrics.ActionListConfigGroups
			req := in.GetConfigFile()
			req.Md5 = wrapperspb.String(in.GetRevision())
			out = g.configServer.GetConfigGroupsWithCache(ctx, req)
			out.Type = apiconfig.ConfigDiscoverResponse_CONFIG_FILE_GROUPS
		default:
			out = api.NewConfigDiscoverResponse(apimodel.Code_InvalidDiscoverResource)
		}

		if err := svr.Send(out); err != nil {
			return err
		}
	}
}
