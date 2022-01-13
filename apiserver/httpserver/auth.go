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
	"github.com/emicklei/go-restful"
	proto "github.com/golang/protobuf/proto"
	api "github.com/polarismesh/polaris-server/common/api/v1"
)

// GetAuthServer 运维接口
func (h *HTTPServer) GetAuthServer() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/core/v1").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

	//
	ws.Route(ws.POST("/user/login").To(h.Login))
	ws.Route(ws.POST("/users").To(h.CreateUsers))
	ws.Route(ws.PUT("/user").To(h.UpdateUser))
	ws.Route(ws.POST("/user/delete").To(h.DeleteUser))
	ws.Route(ws.GET("/users").To(h.ListUsers))
	ws.Route(ws.GET("/user/token").To(h.GetUserToken))
	ws.Route(ws.PUT("/user/token/status").To(h.ChangeUserTokenStatus))
	ws.Route(ws.PUT("/user/token/refresh").To(h.RefreshUserToken))

	//
	ws.Route(ws.POST("/usergroup").To(h.CreateUserGroup))
	ws.Route(ws.PUT("/usergroup").To(h.UpdateUserGroup))
	ws.Route(ws.POST("/usergroup/delete").To(h.DeleteUserGroup))
	ws.Route(ws.GET("/usergroups").To(h.ListUserGroups))
	ws.Route(ws.GET("/usergroup/users").To(h.ListUserByGroup))
	ws.Route(ws.GET("/usergroup/token").To(h.GetUserGroupToken))
	ws.Route(ws.PUT("/usergroup/token/status").To(h.ChangeUserGroupTokenStatus))
	ws.Route(ws.PUT("/usergroup/token/refresh").To(h.RefreshUserGroupToken))

	ws.Route(ws.POST("/auth/strategy").To(h.CreateAuthStrategy))
	ws.Route(ws.PUT("/auth/strategy").To(h.UpdateAuthStrategy))
	ws.Route(ws.POST("/auth/strategy/delete").To(h.DeleteStrategy))
	ws.Route(ws.GET("/auth/strategies").To(h.ListStrategy))
	ws.Route(ws.GET("/auth/strategy/detail").To(h.GetStrategy))

	return ws
}

// Login
//  @receiver h
//  @param req
//  @param rsp
func (h *HTTPServer) Login(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	loginReq := &api.LoginRequest{}

	_, err := handler.Parse(loginReq)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.authMgn.Login(loginReq))
}

// CreateUsers
//  @receiver h
//  @param req
//  @param rsp
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

	handler.WriteHeaderAndProto(h.userServer.CreateUsers(ctx, users))
}

// UpdateUser
//  @receiver h
//  @param req
//  @param rsp
func (h *HTTPServer) UpdateUser(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	user := &api.User{}

	ctx, err := handler.Parse(user)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.userServer.UpdateUser(ctx, user))
}

// DeleteUser
//  @receiver h
//  @param req
//  @param rsp
func (h *HTTPServer) DeleteUser(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	user := &api.User{}

	ctx, err := handler.Parse(user)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.userServer.DeleteUser(ctx, user))
}

// ListUsers
//  @receiver h
//  @param req
//  @param rsp
func (h *HTTPServer) ListUsers(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := parseQueryParams(req)
	ctx := handler.ParseHeaderContext()

	handler.WriteHeaderAndProto(h.userServer.ListUsers(ctx, queryParams))
}

// GetUserToken
//  @receiver h
//  @param req
//  @param rsp
func (h *HTTPServer) GetUserToken(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}
	queryParams := parseQueryParams(req)

	handler.WriteHeaderAndProto(h.userServer.GetUserToken(handler.ParseHeaderContext(), queryParams))
}

// ChangeUserTokenStatus
//  @receiver h
//  @param req
//  @param rsp
func (h *HTTPServer) ChangeUserTokenStatus(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	user := &api.User{}

	ctx, err := handler.Parse(user)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.userServer.ChangeUserTokenStatus(ctx, user))
}

// RefreshUserToken
//  @receiver h
//  @param req
//  @param rsp
func (h *HTTPServer) RefreshUserToken(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	user := &api.User{}

	ctx, err := handler.Parse(user)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.userServer.RefreshUserToken(ctx, user))
}

// CreateUserGroup
//  @receiver h
//  @param req
//  @param rsp
func (h *HTTPServer) CreateUserGroup(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	group := &api.UserGroup{}

	ctx, err := handler.Parse(group)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.userServer.CreateUserGroup(ctx, group))
}

// UpdateUserGroup
//  @receiver h
//  @param req
//  @param rsp
func (h *HTTPServer) UpdateUserGroup(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	group := &api.ModifyUserGroup{}

	ctx, err := handler.Parse(group)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.userServer.UpdateUserGroup(ctx, group))
}

// DeleteUserGroup
//  @receiver h
//  @param req
//  @param rsp
func (h *HTTPServer) DeleteUserGroup(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	group := &api.UserGroup{}

	ctx, err := handler.Parse(group)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.userServer.DeleteUserGroup(ctx, group))
}

// ListUserGroups
//  @receiver h
//  @param req
//  @param rsp
func (h *HTTPServer) ListUserGroups(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := parseQueryParams(req)
	ctx := handler.ParseHeaderContext()

	handler.WriteHeaderAndProto(h.userServer.ListUserGroups(ctx, queryParams))
}

// ListUserByGroup
//  @receiver h
//  @param req
//  @param rsp
func (h *HTTPServer) ListUserByGroup(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := parseQueryParams(req)
	ctx := handler.ParseHeaderContext()

	handler.WriteHeaderAndProto(h.userServer.ListUserByGroup(ctx, queryParams))
}

// GetUserGroupToken
//  @receiver h
//  @param req
//  @param rsp
func (h *HTTPServer) GetUserGroupToken(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := parseQueryParams(req)
	ctx := handler.ParseHeaderContext()

	handler.WriteHeaderAndProto(h.userServer.GetUserGroupToken(ctx, queryParams))
}

// ChangeUserGroupTokenStatus
//  @receiver h
//  @param req
//  @param rsp
func (h *HTTPServer) ChangeUserGroupTokenStatus(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	group := &api.UserGroup{}

	ctx, err := handler.Parse(group)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.userServer.ChangeUserGroupTokenStatus(ctx, group))
}

// RefreshUserGroupToken
//  @receiver h
//  @param req
//  @param rsp
func (h *HTTPServer) RefreshUserGroupToken(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	group := &api.UserGroup{}

	ctx, err := handler.Parse(group)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.userServer.RefreshUserGroupToken(ctx, group))
}

// CreateAuthStrategy
//  @receiver h
//  @param req
//  @param rsp
func (h *HTTPServer) CreateAuthStrategy(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	strategy := &api.AuthStrategy{}

	ctx, err := handler.Parse(strategy)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.strategyServer.CreateStrategy(ctx, strategy))
}

// UpdateAuthStrategy
//  @receiver h
//  @param req
//  @param rsp
func (h *HTTPServer) UpdateAuthStrategy(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	strategy := &api.ModifyAuthStrategy{}

	ctx, err := handler.Parse(strategy)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.strategyServer.UpdateStrategy(ctx, strategy))
}

// DeleteStrategy
//  @receiver h
//  @param req
//  @param rsp
func (h *HTTPServer) DeleteStrategy(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	strategy := &api.AuthStrategy{}

	ctx, err := handler.Parse(strategy)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.strategyServer.DeleteStrategy(ctx, strategy))
}

// ListStrategy
//  @receiver h
//  @param req
//  @param rsp
func (h *HTTPServer) ListStrategy(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := parseQueryParams(req)
	ctx := handler.ParseHeaderContext()

	handler.WriteHeaderAndProto(h.strategyServer.ListStrategy(ctx, queryParams))
}

func (h *HTTPServer) GetStrategy(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := parseQueryParams(req)
	ctx := handler.ParseHeaderContext()

	handler.WriteHeaderAndProto(h.strategyServer.GetStrategy(ctx, queryParams))
}
