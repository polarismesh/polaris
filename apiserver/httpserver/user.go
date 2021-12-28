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
	api "github.com/polarismesh/polaris-server/common/api/v1"
)

// GetMaintainAccessServer 运维接口
func (h *HTTPServer) GetUserAccessServer() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/core/v1").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

	//
	ws.Route(ws.POST("/user").To(h.CreateUser))
	ws.Route(ws.PUT("/user").To(h.UpdateUser))
	ws.Route(ws.POST("/user/delete").To(h.DeleteUser))
	ws.Route(ws.GET("/users").To(h.ListUsers))
	ws.Route(ws.GET("/user/token").To(h.GetUserToken))
	ws.Route(ws.PUT("/user/token/disable").To(h.DisableUserToken))
	ws.Route(ws.PUT("/user/token/enable").To(h.DisableUserToken))
	ws.Route(ws.PUT("/user/token/refresh").To(h.RefreshUserToken))

	//
	ws.Route(ws.POST("/usergroup").To(h.CreateUserGroup))
	ws.Route(ws.PUT("/usergroup").To(h.UpdateUserGroup))
	ws.Route(ws.POST("/usergroup/delete").To(h.DeleteUserGroup))
	ws.Route(ws.GET("/usergroups").To(h.ListUserGroups))
	ws.Route(ws.GET("/usergroup/user/add").To(h.BatchAddUserToGroup))
	ws.Route(ws.GET("/usergroups/user/remove").To(h.BatchRemoveUserFromGroup))
	ws.Route(ws.GET("/usergroup/token").To(h.GetUserGroupToken))
	ws.Route(ws.PUT("/usergroup/token/disable").To(h.DisableUserGroupToken))
	ws.Route(ws.PUT("/usergroup/token/enable").To(h.EnableUserGroupToken))
	ws.Route(ws.PUT("/usergroup/token/refresh").To(h.RefreshUserGroupToken))
	return ws
}

func (h *HTTPServer) CreateUser(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	user := &api.User{}

	ctx, err := handler.Parse(user)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.userServer.CreateUser(ctx, user))
}

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

func (h *HTTPServer) ListUsers(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := parseQueryParams(req)
	ctx := handler.ParseHeaderContext()

	handler.WriteHeaderAndProto(h.userServer.ListUsers(ctx, queryParams))
}

func (h *HTTPServer) GetUserToken(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	user := &api.User{}

	ctx, err := handler.Parse(user)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.userServer.GetUserToken(ctx, user))
}

func (h *HTTPServer) DisableUserToken(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	user := &api.User{}

	ctx, err := handler.Parse(user)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.userServer.DisableUserToken(ctx, user))
}

func (h *HTTPServer) EnableUserToken(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	user := &api.User{}

	ctx, err := handler.Parse(user)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.userServer.EnableUserToken(ctx, user))
}

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

func (h *HTTPServer) UpdateUserGroup(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	group := &api.UserGroup{}

	ctx, err := handler.Parse(group)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.userServer.UpdateUserGroup(ctx, group))
}

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

func (h *HTTPServer) ListUserGroups(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := parseQueryParams(req)
	ctx := handler.ParseHeaderContext()

	handler.WriteHeaderAndProto(h.userServer.ListUserGroups(ctx, queryParams))
}

func (h *HTTPServer) GetUserGroupToken(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	group := &api.UserGroup{}

	ctx, err := handler.Parse(group)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.userServer.GetUserGroupToken(ctx, group))
}

func (h *HTTPServer) DisableUserGroupToken(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	group := &api.UserGroup{}

	ctx, err := handler.Parse(group)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.userServer.DisableUserGroupToken(ctx, group))
}

func (h *HTTPServer) EnableUserGroupToken(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	group := &api.UserGroup{}

	ctx, err := handler.Parse(group)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.userServer.EnableUserGroupToken(ctx, group))
}

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

func (h *HTTPServer) BatchAddUserToGroup(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	group := &api.UserGroupRelation{}

	ctx, err := handler.Parse(group)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.userServer.BatchAddUserToGroup(ctx, group))
}

func (h *HTTPServer) BatchRemoveUserFromGroup(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	group := &api.UserGroupRelation{}

	ctx, err := handler.Parse(group)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.userServer.BatchRemoveUserFromGroup(ctx, group))
}
