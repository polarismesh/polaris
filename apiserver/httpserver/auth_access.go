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

package httpserver

import (
	"strconv"

	"github.com/emicklei/go-restful/v3"
	"github.com/golang/protobuf/proto"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	"github.com/polarismesh/polaris/apiserver/httpserver/docs"
	httpcommon "github.com/polarismesh/polaris/apiserver/httpserver/utils"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/utils"
)

// GetAuthServer 运维接口
func (h *HTTPServer) GetAuthServer(ws *restful.WebService) error {
	ws.Route(docs.EnrichAuthStatusApiDocs(ws.GET("/auth/status").To(h.AuthStatus)))
	//
	ws.Route(docs.EnrichLoginApiDocs(ws.POST("/user/login").To(h.Login)))
	ws.Route(docs.EnrichGetUsersApiDocs(ws.GET("/users").To(h.GetUsers)))
	ws.Route(docs.EnrichCreateUsersApiDocs(ws.POST("/users").To(h.CreateUsers)))
	ws.Route(docs.EnrichDeleteUsersApiDocs(ws.POST("/users/delete").To(h.DeleteUsers)))
	ws.Route(docs.EnrichUpdateUserApiDocs(ws.PUT("/user").To(h.UpdateUser)))
	ws.Route(docs.EnrichUpdateUserPasswordApiDocs(ws.PUT("/user/password").To(h.UpdateUserPassword)))
	ws.Route(docs.EnrichGetUserTokenApiDocs(ws.GET("/user/token").To(h.GetUserToken)))
	ws.Route(docs.EnrichUpdateUserTokenApiDocs(ws.PUT("/user/token/status").To(h.EnableUserToken)))
	ws.Route(docs.EnrichResetUserTokenApiDocs(ws.PUT("/user/token/refresh").To(h.ResetUserToken)))
	//
	ws.Route(docs.EnrichCreateGroupApiDocs(ws.POST("/usergroup").To(h.CreateGroup)))
	ws.Route(docs.EnrichUpdateGroupsApiDocs(ws.PUT("/usergroups").To(h.UpdateGroups)))
	ws.Route(docs.EnrichGetGroupsApiDocs(ws.GET("/usergroups").To(h.GetGroups)))
	ws.Route(docs.EnrichDeleteGroupsApiDocs(ws.POST("/usergroups/delete").To(h.DeleteGroups)))
	ws.Route(docs.EnrichGetGroupApiDocs(ws.GET("/usergroup/detail").To(h.GetGroup)))
	ws.Route(docs.EnrichGetGroupTokenApiDocs(ws.GET("/usergroup/token").To(h.GetGroupToken)))
	ws.Route(docs.EnrichUpdateGroupTokenApiDocs(ws.PUT("/usergroup/token/status").To(h.EnableGroupToken)))
	ws.Route(docs.EnrichResetGroupTokenApiDocs(ws.PUT("/usergroup/token/refresh").To(h.ResetGroupToken)))

	ws.Route(docs.EnrichCreateStrategyApiDocs(ws.POST("/auth/strategy").To(h.CreateStrategy)))
	ws.Route(docs.EnrichGetStrategyApiDocs(ws.GET("/auth/strategy/detail").To(h.GetStrategy)))
	ws.Route(docs.EnrichUpdateStrategiesApiDocs(ws.PUT("/auth/strategies").To(h.UpdateStrategies)))
	ws.Route(docs.EnrichDeleteStrategiesApiDocs(ws.POST("/auth/strategies/delete").To(h.DeleteStrategies)))
	ws.Route(docs.EnrichGetStrategiesApiDocs(ws.GET("/auth/strategies").To(h.GetStrategies)))
	ws.Route(docs.EnrichGetPrincipalResourcesApiDocs(ws.GET("/auth/principal/resources").To(h.GetPrincipalResources)))

	return nil
}

// AuthStatus auth status
func (h *HTTPServer) AuthStatus(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	checker := h.strategyMgn.GetAuthChecker()

	isOpen := (checker.IsOpenClientAuth() || checker.IsOpenConsoleAuth())
	resp := api.NewAuthResponse(apimodel.Code_ExecuteSuccess)
	resp.OptionSwitch = &apiservice.OptionSwitch{
		Options: map[string]string{
			"auth":        strconv.FormatBool(isOpen),
			"clientOpen":  strconv.FormatBool(checker.IsOpenClientAuth()),
			"consoleOpen": strconv.FormatBool(checker.IsOpenConsoleAuth()),
		},
	}

	handler.WriteHeaderAndProto(resp)
}

// Login 登录函数
func (h *HTTPServer) Login(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	loginReq := &apisecurity.LoginRequest{}

	_, err := handler.Parse(loginReq)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewAuthResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.userMgn.Login(loginReq))
}

// CreateUsers 批量创建用户
func (h *HTTPServer) CreateUsers(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var users UserArr

	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &apisecurity.User{}
		users = append(users, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.userMgn.CreateUsers(ctx, users))
}

// UpdateUser 更新用户
func (h *HTTPServer) UpdateUser(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	user := &apisecurity.User{}

	ctx, err := handler.Parse(user)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.userMgn.UpdateUser(ctx, user))
}

// UpdateUserPassword 更新用户
func (h *HTTPServer) UpdateUserPassword(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	user := &apisecurity.ModifyUserPassword{}

	ctx, err := handler.Parse(user)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.userMgn.UpdateUserPassword(ctx, user))
}

// DeleteUsers 批量删除用户
func (h *HTTPServer) DeleteUsers(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var users UserArr

	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &apisecurity.User{}
		users = append(users, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.userMgn.DeleteUsers(ctx, users))
}

// GetUsers 查询用户
func (h *HTTPServer) GetUsers(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	queryParams := httpcommon.ParseQueryParams(req)
	ctx := handler.ParseHeaderContext()

	handler.WriteHeaderAndProto(h.userMgn.GetUsers(ctx, queryParams))
}

// GetUserToken 获取这个用户所关联的所有用户组列表信息，支持翻页
func (h *HTTPServer) GetUserToken(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}
	queryParams := httpcommon.ParseQueryParams(req)

	user := &apisecurity.User{
		Id: utils.NewStringValue(queryParams["id"]),
	}

	handler.WriteHeaderAndProto(h.userMgn.GetUserToken(handler.ParseHeaderContext(), user))
}

// EnableUserToken 更改用户的token
func (h *HTTPServer) EnableUserToken(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	user := &apisecurity.User{}

	ctx, err := handler.Parse(user)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.userMgn.EnableUserToken(ctx, user))
}

// ResetUserToken 重置用户 token
func (h *HTTPServer) ResetUserToken(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	user := &apisecurity.User{}

	ctx, err := handler.Parse(user)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.userMgn.ResetUserToken(ctx, user))
}

// CreateGroup 创建用户组
func (h *HTTPServer) CreateGroup(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	group := &apisecurity.UserGroup{}

	ctx, err := handler.Parse(group)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.userMgn.CreateGroup(ctx, group))
}

// UpdateGroups 更新用户组
func (h *HTTPServer) UpdateGroups(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var groups ModifyGroupArr

	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &apisecurity.ModifyUserGroup{}
		groups = append(groups, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.userMgn.UpdateGroups(ctx, groups))
}

// DeleteGroups 删除用户组
func (h *HTTPServer) DeleteGroups(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var groups GroupArr

	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &apisecurity.UserGroup{}
		groups = append(groups, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.userMgn.DeleteGroups(ctx, groups))
}

// GetGroups 获取用户组列表
func (h *HTTPServer) GetGroups(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	queryParams := httpcommon.ParseQueryParams(req)
	ctx := handler.ParseHeaderContext()

	handler.WriteHeaderAndProto(h.userMgn.GetGroups(ctx, queryParams))
}

// GetGroup 获取用户组详细
func (h *HTTPServer) GetGroup(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	queryParams := httpcommon.ParseQueryParams(req)
	ctx := handler.ParseHeaderContext()

	group := &apisecurity.UserGroup{
		Id: utils.NewStringValue(queryParams["id"]),
	}

	handler.WriteHeaderAndProto(h.userMgn.GetGroup(ctx, group))
}

// GetGroupToken 获取用户组 token
func (h *HTTPServer) GetGroupToken(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	queryParams := httpcommon.ParseQueryParams(req)
	ctx := handler.ParseHeaderContext()

	group := &apisecurity.UserGroup{
		Id: utils.NewStringValue(queryParams["id"]),
	}

	handler.WriteHeaderAndProto(h.userMgn.GetGroupToken(ctx, group))
}

// EnableGroupToken 更新用户组 token
func (h *HTTPServer) EnableGroupToken(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	group := &apisecurity.UserGroup{}

	ctx, err := handler.Parse(group)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.userMgn.EnableGroupToken(ctx, group))
}

// ResetGroupToken 重置用户组 token
func (h *HTTPServer) ResetGroupToken(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	group := &apisecurity.UserGroup{}

	ctx, err := handler.Parse(group)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.userMgn.ResetGroupToken(ctx, group))
}

// CreateStrategy 创建鉴权策略
func (h *HTTPServer) CreateStrategy(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	strategy := &apisecurity.AuthStrategy{}

	ctx, err := handler.Parse(strategy)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewAuthResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.strategyMgn.CreateStrategy(ctx, strategy))
}

// UpdateStrategies 更新鉴权策略
func (h *HTTPServer) UpdateStrategies(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var strategies ModifyStrategyArr

	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &apisecurity.ModifyAuthStrategy{}
		strategies = append(strategies, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.strategyMgn.UpdateStrategies(ctx, strategies))
}

// DeleteStrategies 批量删除鉴权策略
func (h *HTTPServer) DeleteStrategies(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var strategies StrategyArr

	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &apisecurity.AuthStrategy{}
		strategies = append(strategies, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.strategyMgn.DeleteStrategies(ctx, strategies))
}

// GetStrategies 批量获取鉴权策略
func (h *HTTPServer) GetStrategies(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	queryParams := httpcommon.ParseQueryParams(req)
	ctx := handler.ParseHeaderContext()

	handler.WriteHeaderAndProto(h.strategyMgn.GetStrategies(ctx, queryParams))
}

// GetStrategy 获取鉴权策略详细
func (h *HTTPServer) GetStrategy(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	queryParams := httpcommon.ParseQueryParams(req)
	ctx := handler.ParseHeaderContext()

	strategy := &apisecurity.AuthStrategy{
		Id: utils.NewStringValue(queryParams["id"]),
	}

	handler.WriteHeaderAndProto(h.strategyMgn.GetStrategy(ctx, strategy))
}

// GetPrincipalResources 获取鉴权策略详细
func (h *HTTPServer) GetPrincipalResources(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	queryParams := httpcommon.ParseQueryParams(req)
	ctx := handler.ParseHeaderContext()

	handler.WriteHeaderAndProto(h.strategyMgn.GetPrincipalResources(ctx, queryParams))
}
