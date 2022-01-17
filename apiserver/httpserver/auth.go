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
	"github.com/polarismesh/polaris-server/common/utils"
)

// GetAuthServer 运维接口
func (h *HTTPServer) GetAuthServer() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/core/v1").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

	//
	ws.Route(ws.POST("/user/login").To(h.Login))
	ws.Route(ws.GET("/users").To(h.GetUsers))
	ws.Route(ws.POST("/users").To(h.CreateUsers))
	ws.Route(ws.POST("/users/delete").To(h.DeleteUsers))
	ws.Route(ws.PUT("/user").To(h.UpdateUser))
	ws.Route(ws.GET("/user/token").To(h.GetUserToken))
	ws.Route(ws.PUT("/user/token/status").To(h.UpdateUserToken))
	ws.Route(ws.PUT("/user/token/refresh").To(h.ResetUserToken))

	//
	ws.Route(ws.POST("/usergroup").To(h.CreateGroup))
	ws.Route(ws.PUT("/usergroup").To(h.UpdateGroup))
	ws.Route(ws.POST("/usergroups/delete").To(h.DeleteGroups))
	ws.Route(ws.GET("/usergroups").To(h.GetGroups))
	ws.Route(ws.GET("/usergroup/users").To(h.GetGroupUsers))
	ws.Route(ws.GET("/usergroup/token").To(h.GetGroupToken))
	ws.Route(ws.PUT("/usergroup/token/status").To(h.UpdateGroupToken))
	ws.Route(ws.PUT("/usergroup/token/refresh").To(h.ResetGroupToken))

	ws.Route(ws.POST("/auth/strategy").To(h.CreateStrategy))
	ws.Route(ws.PUT("/auth/strategy").To(h.UpdateStrategy))
	ws.Route(ws.GET("/auth/strategy/detail").To(h.GetStrategy))
	ws.Route(ws.POST("/auth/strategies/delete").To(h.DeleteStrategies))
	ws.Route(ws.GET("/auth/strategies").To(h.GetStrategies))

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

	handler.WriteHeaderAndProto(h.authServer.Login(loginReq))
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

	handler.WriteHeaderAndProto(h.authServer.CreateUsers(ctx, users))
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

	handler.WriteHeaderAndProto(h.authServer.UpdateUser(ctx, user))
}

// DeleteUsers
//  @receiver h
//  @param req
//  @param rsp
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

// GetUsers
//  @receiver h
//  @param req
//  @param rsp
func (h *HTTPServer) GetUsers(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := parseQueryParams(req)
	ctx := handler.ParseHeaderContext()

	handler.WriteHeaderAndProto(h.authServer.GetUsers(ctx, queryParams))
}

// GetUserToken 获取这个用户所关联的所有用户组列表信息，支持翻页
//  @receiver h
//  @param req
//  @param rsp
func (h *HTTPServer) GetUserToken(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}
	queryParams := parseQueryParams(req)

	user := &api.User{
		Id: utils.NewStringValue(queryParams["id"]),
	}

	handler.WriteHeaderAndProto(h.authServer.GetUserToken(handler.ParseHeaderContext(), user))
}

// ChangeUserTokenStatus
//  @receiver h
//  @param req
//  @param rsp
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

// ResetUserToken
//  @receiver h
//  @param req
//  @param rsp
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

// CreateUserGroup
//  @receiver h
//  @param req
//  @param rsp
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

// UpdateGroup
//  @receiver h
//  @param req
//  @param rsp
func (h *HTTPServer) UpdateGroup(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	group := &api.ModifyUserGroup{}

	ctx, err := handler.Parse(group)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.authServer.UpdateGroup(ctx, group))
}

// DeleteUserGroup
//  @receiver h
//  @param req
//  @param rsp
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

// GetGroups
//  @receiver h
//  @param req
//  @param rsp
func (h *HTTPServer) GetGroups(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := parseQueryParams(req)
	ctx := handler.ParseHeaderContext()

	handler.WriteHeaderAndProto(h.authServer.GetGroups(ctx, queryParams))
}

// GetGroupUsers
func (h *HTTPServer) GetGroupUsers(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := parseQueryParams(req)
	ctx := handler.ParseHeaderContext()

	handler.WriteHeaderAndProto(h.authServer.GetGroupUsers(ctx, queryParams))
}

// GetUserGroupToken
//  @receiver h
//  @param req
//  @param rsp
func (h *HTTPServer) GetGroupToken(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := parseQueryParams(req)
	ctx := handler.ParseHeaderContext()

	group := &api.UserGroup{
		Id: utils.NewStringValue(queryParams["id"]),
	}

	handler.WriteHeaderAndProto(h.authServer.GetGroupToken(ctx, group))
}

// UpdateGroupToken
//  @receiver h
//  @param req
//  @param rsp
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

// ResetGroupToken
//  @receiver h
//  @param req
//  @param rsp
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

// CreateStrategy
//  @receiver h
//  @param req
//  @param rsp
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

// UpdateStrategy
//  @receiver h
//  @param req
//  @param rsp
func (h *HTTPServer) UpdateStrategy(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	strategy := &api.ModifyAuthStrategy{}

	ctx, err := handler.Parse(strategy)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.authServer.UpdateStrategy(ctx, strategy))
}

// DeleteStrategies
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

// GetStrategies
//  @receiver h
//  @param req
//  @param rsp
func (h *HTTPServer) GetStrategies(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := parseQueryParams(req)
	ctx := handler.ParseHeaderContext()

	handler.WriteHeaderAndProto(h.authServer.GetStrategies(ctx, queryParams))
}

func (h *HTTPServer) GetStrategy(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := parseQueryParams(req)
	ctx := handler.ParseHeaderContext()

	strategy := &api.AuthStrategy{
		Id: utils.NewStringValue(queryParams["id"]),
	}

	handler.WriteHeaderAndProto(h.authServer.GetStrategy(ctx, strategy))
}
