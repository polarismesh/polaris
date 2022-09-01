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

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/utils"
)

// GetAuthServer 运维接口
func (h *HTTPServer) GetAuthServer(ws *restful.WebService) error {

	ws.Route(ws.GET("/auth/status").To(h.AuthStatus))

	//
	ws.Route(ws.POST("/user/login").To(h.Login))
	ws.Route(ws.GET("/users").To(h.GetUsers))
	ws.Route(ws.POST("/users").To(h.CreateUsers))
	ws.Route(ws.POST("/users/delete").To(h.DeleteUsers))
	ws.Route(ws.PUT("/user").To(h.UpdateUser))
	ws.Route(ws.PUT("/user/password").To(h.UpdateUserPassword))
	ws.Route(ws.GET("/user/token").To(h.GetUserToken))
	ws.Route(ws.PUT("/user/token/status").To(h.UpdateUserToken))
	ws.Route(ws.PUT("/user/token/refresh").To(h.ResetUserToken))

	//
	ws.Route(ws.POST("/usergroup").To(h.CreateGroup))
	ws.Route(ws.PUT("/usergroups").To(h.UpdateGroups))
	ws.Route(ws.GET("/usergroups").To(h.GetGroups))
	ws.Route(ws.POST("/usergroups/delete").To(h.DeleteGroups))
	ws.Route(ws.GET("/usergroup/detail").To(h.GetGroup))
	ws.Route(ws.GET("/usergroup/token").To(h.GetGroupToken))
	ws.Route(ws.PUT("/usergroup/token/status").To(h.UpdateGroupToken))
	ws.Route(ws.PUT("/usergroup/token/refresh").To(h.ResetGroupToken))

	ws.Route(ws.POST("/auth/strategy").To(h.CreateStrategy))
	ws.Route(ws.GET("/auth/strategy/detail").To(h.GetStrategy))
	ws.Route(ws.PUT("/auth/strategies").To(h.UpdateStrategies))
	ws.Route(ws.POST("/auth/strategies/delete").To(h.DeleteStrategies))
	ws.Route(ws.GET("/auth/strategies").To(h.GetStrategies))
	ws.Route(ws.GET("/auth/principal/resources").To(h.GetPrincipalResources))

	return nil
}

// AuthStatus auth status
func (h *HTTPServer) AuthStatus(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	checker := h.authServer.GetAuthChecker()

	isOpen := (checker.IsOpenClientAuth() || checker.IsOpenConsoleAuth())
	resp := api.NewResponse(api.ExecuteSuccess)
	resp.OptionSwitch = &api.OptionSwitch{
		Options: map[string]string{
			"auth": strconv.FormatBool(isOpen),
		},
	}

	handler.WriteHeaderAndProto(resp)
}

// Login 登陆函数
func (h *HTTPServer) Login(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	loginReq := &api.LoginRequest{}

	_, err := handler.Parse(loginReq)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.authServer.Login(loginReq))
}

// CreateUsers 批量创建用户
func (h *HTTPServer) CreateUsers(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var users UserArr

	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.User{}
		users = append(users, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.authServer.CreateUsers(ctx, users))
}

// UpdateUser 更新用户
func (h *HTTPServer) UpdateUser(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	user := &api.User{}

	ctx, err := handler.Parse(user)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.authServer.UpdateUser(ctx, user))
}

// UpdateUserPassword 更新用户
func (h *HTTPServer) UpdateUserPassword(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	user := &api.ModifyUserPassword{}

	ctx, err := handler.Parse(user)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.authServer.UpdateUserPassword(ctx, user))
}

// DeleteUsers 批量删除用户
func (h *HTTPServer) DeleteUsers(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var users UserArr

	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.User{}
		users = append(users, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.authServer.DeleteUsers(ctx, users))
}

// GetUsers 查询用户
func (h *HTTPServer) GetUsers(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := utils.ParseQueryParams(req)
	ctx := handler.ParseHeaderContext()

	handler.WriteHeaderAndProto(h.authServer.GetUsers(ctx, queryParams))
}

// GetUserToken 获取这个用户所关联的所有用户组列表信息，支持翻页
func (h *HTTPServer) GetUserToken(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}
	queryParams := utils.ParseQueryParams(req)

	user := &api.User{
		Id: utils.NewStringValue(queryParams["id"]),
	}

	handler.WriteHeaderAndProto(h.authServer.GetUserToken(handler.ParseHeaderContext(), user))
}

// UpdateUserToken 更改用户的token
func (h *HTTPServer) UpdateUserToken(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	user := &api.User{}

	ctx, err := handler.Parse(user)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.authServer.UpdateUserToken(ctx, user))
}

// ResetUserToken 重置用户 token
func (h *HTTPServer) ResetUserToken(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	user := &api.User{}

	ctx, err := handler.Parse(user)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.authServer.ResetUserToken(ctx, user))
}

// CreateGroup 创建用户组
func (h *HTTPServer) CreateGroup(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	group := &api.UserGroup{}

	ctx, err := handler.Parse(group)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.authServer.CreateGroup(ctx, group))
}

// UpdateGroups 更新用户组
func (h *HTTPServer) UpdateGroups(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var groups ModifyGroupArr

	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.ModifyUserGroup{}
		groups = append(groups, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.authServer.UpdateGroups(ctx, groups))
}

// DeleteGroups 删除用户组
func (h *HTTPServer) DeleteGroups(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var groups GroupArr

	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.UserGroup{}
		groups = append(groups, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.authServer.DeleteGroups(ctx, groups))
}

// GetGroups 获取用户组列表
func (h *HTTPServer) GetGroups(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := utils.ParseQueryParams(req)
	ctx := handler.ParseHeaderContext()

	handler.WriteHeaderAndProto(h.authServer.GetGroups(ctx, queryParams))
}

// GetGroup 获取用户组详细
func (h *HTTPServer) GetGroup(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := utils.ParseQueryParams(req)
	ctx := handler.ParseHeaderContext()

	group := &api.UserGroup{
		Id: utils.NewStringValue(queryParams["id"]),
	}

	handler.WriteHeaderAndProto(h.authServer.GetGroup(ctx, group))
}

// GetGroupToken 获取用户组 token
func (h *HTTPServer) GetGroupToken(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := utils.ParseQueryParams(req)
	ctx := handler.ParseHeaderContext()

	group := &api.UserGroup{
		Id: utils.NewStringValue(queryParams["id"]),
	}

	handler.WriteHeaderAndProto(h.authServer.GetGroupToken(ctx, group))
}

// UpdateGroupToken 更新用户组 token
func (h *HTTPServer) UpdateGroupToken(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	group := &api.UserGroup{}

	ctx, err := handler.Parse(group)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.authServer.UpdateGroupToken(ctx, group))
}

// ResetGroupToken 重置用户组 token
func (h *HTTPServer) ResetGroupToken(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	group := &api.UserGroup{}

	ctx, err := handler.Parse(group)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.authServer.ResetGroupToken(ctx, group))
}

// CreateStrategy 创建鉴权策略
func (h *HTTPServer) CreateStrategy(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	strategy := &api.AuthStrategy{}

	ctx, err := handler.Parse(strategy)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.authServer.CreateStrategy(ctx, strategy))
}

// UpdateStrategies 更新鉴权策略
func (h *HTTPServer) UpdateStrategies(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var strategies ModifyStrategyArr

	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.ModifyAuthStrategy{}
		strategies = append(strategies, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.authServer.UpdateStrategies(ctx, strategies))
}

// DeleteStrategies 批量删除鉴权策略
func (h *HTTPServer) DeleteStrategies(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	var strategies StrategyArr

	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.AuthStrategy{}
		strategies = append(strategies, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.authServer.DeleteStrategies(ctx, strategies))
}

// GetStrategies 批量获取鉴权策略
func (h *HTTPServer) GetStrategies(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := utils.ParseQueryParams(req)
	ctx := handler.ParseHeaderContext()

	handler.WriteHeaderAndProto(h.authServer.GetStrategies(ctx, queryParams))
}

// GetStrategy 获取鉴权策略详细
func (h *HTTPServer) GetStrategy(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := utils.ParseQueryParams(req)
	ctx := handler.ParseHeaderContext()

	strategy := &api.AuthStrategy{
		Id: utils.NewStringValue(queryParams["id"]),
	}

	handler.WriteHeaderAndProto(h.authServer.GetStrategy(ctx, strategy))
}

// GetPrincipalResources 获取鉴权策略详细
func (h *HTTPServer) GetPrincipalResources(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := utils.ParseQueryParams(req)
	ctx := handler.ParseHeaderContext()

	handler.WriteHeaderAndProto(h.authServer.GetPrincipalResources(ctx, queryParams))
}
