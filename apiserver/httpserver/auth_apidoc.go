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
	"github.com/emicklei/go-restful/v3"
	restfulspec "github.com/polarismesh/go-restful-openapi/v2"

	api "github.com/polarismesh/polaris/common/api/v1"
)

var (
	authApiTags      = []string{"Auth"}
	usersApiTags     = []string{"Users"}
	userGroupApiTags = []string{"UserGroup"}
)

func enrichAuthStatusApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("查询鉴权开关信息").
		Metadata(restfulspec.KeyOpenAPITags, authApiTags).
		Notes(enrichAuthStatusApiNotes)
}

func enrichCreateStrategyApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("创建鉴权策略").
		Metadata(restfulspec.KeyOpenAPITags, authApiTags).
		Reads(api.AuthStrategy{}, "create auth strategy").
		Notes(enrichCreateStrategyApiNotes)
}

func enrichUpdateStrategiesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("更新鉴权策略").
		Metadata(restfulspec.KeyOpenAPITags, authApiTags).
		Reads(api.AuthStrategy{}, "update auth strategy").
		Notes(enrichUpdateStrategiesApiNotes)
}

func enrichGetStrategiesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("查询鉴权策略列表").
		Metadata(restfulspec.KeyOpenAPITags, authApiTags).
		Param(restful.QueryParameter("id", "策略ID").DataType("string").Required(false)).
		Param(restful.QueryParameter("name", "策略名称, 当前仅提供全模糊搜索").DataType("string").Required(false)).
		Param(restful.QueryParameter("default", "“0” 查询自定义策略；“1” 查询默认策略；不填则为查询（默认+自定义）鉴权策略").DataType("string").Required(false)).
		Param(restful.QueryParameter("res_id", "资源ID").DataType("string").Required(false)).
		Param(restful.QueryParameter("res_type", "资源类型, namespace、service、config_group").DataType("string").Required(false)).
		Param(restful.QueryParameter("principal_id", "成员ID").DataType("string").Required(false)).
		Param(restful.QueryParameter("principal_type", "成员类型, user、group").DataType("string").Required(false)).
		Param(restful.QueryParameter("show_detail", "是否显示策略详细").DataType("boolean").Required(false)).
		Param(restful.QueryParameter("offset", "查询偏移量, 默认为0").DataType("integer").Required(false).DefaultValue("0")).
		Param(restful.QueryParameter("limit", "本次查询条数, 最大为100").DataType("integer").Required(false)).
		Notes(enrichGetStrategiesApiNotes)
}

func enrichGetPrincipalResourcesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("获取鉴权策略详细").
		Metadata(restfulspec.KeyOpenAPITags, authApiTags).
		Param(restful.QueryParameter("principal_id", "策略ID").DataType("string").Required(true)).
		Param(restful.QueryParameter("principal_type", "Principal类别，user/group").DataType("string").Required(true)).
		Notes(enrichGetPrincipalResourcesApiNotes)
}

func enrichGetStrategyApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("获取鉴权策略详细").
		Metadata(restfulspec.KeyOpenAPITags, authApiTags).
		Param(restful.QueryParameter("id", "策略ID").DataType("string").Required(true)).
		Notes(enrichGetStrategyApiNotes)
}

func enrichDeleteStrategiesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("删除鉴权策略").
		Metadata(restfulspec.KeyOpenAPITags, authApiTags).
		Reads(api.AuthStrategy{}, "delete auth strategy").
		Notes(enrichDeleteStrategiesApiNotes)
}

func enrichLoginApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("用户登录").
		Metadata(restfulspec.KeyOpenAPITags, usersApiTags).
		Reads(api.LoginRequest{}, "用户登录接口").
		Notes(enrichLoginApiNotes)
}

func enrichGetUsersApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("获取用户").
		Metadata(restfulspec.KeyOpenAPITags, usersApiTags).
		Param(restful.QueryParameter("id", "用户ID").DataType("string").Required(false)).
		Param(restful.QueryParameter("name", "用户名称, 当前仅提供全模糊搜索").DataType("string").Required(false)).
		Param(restful.QueryParameter("source", "用户来源").DataType("string").Required(false)).
		Param(restful.QueryParameter("group_id", "用户组ID, 用于查询某个用户组下用户列表").DataType("string").Required(false)).
		Param(restful.QueryParameter("offset", "查询偏移量, 默认为0").DataType("integer").Required(false).DefaultValue("0")).
		Param(restful.QueryParameter("limit", "本次查询条数, 最大为100").DataType("integer").Required(false)).
		Notes(enrichGetUsersApiNotes)
}

func enrichCreateUsersApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("创建用户").
		Metadata(restfulspec.KeyOpenAPITags, usersApiTags).
		Reads(api.User{}, "create user").
		Notes(enrichCreateUsersApiNotes)
}

func enrichDeleteUsersApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("删除用户").
		Metadata(restfulspec.KeyOpenAPITags, usersApiTags).
		Reads(api.User{}, "delete user").
		Notes(enrichDeleteUsersApiNotes)
}

func enrichUpdateUserApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("更新用户").
		Metadata(restfulspec.KeyOpenAPITags, usersApiTags).
		Reads(api.User{}, "update user").
		Notes(enrichUpdateUserApiNotes)
}

func enrichUpdateUserPasswordApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("更新用户密码").
		Metadata(restfulspec.KeyOpenAPITags, usersApiTags).
		Reads(api.ModifyUserPassword{}, "update user password").
		Notes(enrichUpdateUserPasswordApiNotes)
}

func enrichGetUserTokenApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("获取用户Token").
		Metadata(restfulspec.KeyOpenAPITags, usersApiTags).
		Param(restful.QueryParameter("id", "用户ID").DataType("string").Required(true)).
		Notes(enrichGetUserTokenApiNotes)
}

func enrichUpdateUserTokenApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("更新用户Token").
		Metadata(restfulspec.KeyOpenAPITags, usersApiTags).
		Reads(api.User{}, "update user token").
		Notes(enrichUpdateUserTokenApiNotes)
}

func enrichResetUserTokenApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("重置用户Token").
		Metadata(restfulspec.KeyOpenAPITags, usersApiTags).
		Reads(api.User{}, "reset user token").
		Notes(enrichResetUserTokenApiNotes)
}

func enrichCreateGroupApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("创建用户组").
		Metadata(restfulspec.KeyOpenAPITags, userGroupApiTags).
		Reads(api.UserGroup{}, "create group").
		Notes(enrichCreateGroupApiNotes)
}

func enrichUpdateGroupsApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("更新用户组").
		Metadata(restfulspec.KeyOpenAPITags, userGroupApiTags).
		Reads(api.UserGroup{}, "update group").
		Notes(enrichUpdateGroupsApiNotes)
}

func enrichGetGroupsApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("查询用户组列表").
		Metadata(restfulspec.KeyOpenAPITags, userGroupApiTags).
		Param(restful.QueryParameter("id", "用户组ID").DataType("string").Required(false)).
		Param(restful.QueryParameter("name", "用户组名称, 当前仅提供全模糊搜索").DataType("string").Required(false)).
		Param(restful.QueryParameter("user_id", "用户ID, 用于查询某个用户关联的用户组列表").DataType("string").Required(false)).
		Param(restful.QueryParameter("offset", "查询偏移量, 默认为0").DataType("integer").Required(false).DefaultValue("0")).
		Param(restful.QueryParameter("limit", "本次查询条数, 最大为100").DataType("integer").Required(false)).
		Notes(enrichGetGroupsApiNotes)
}

func enrichGetGroupApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("获取用户组详情").
		Metadata(restfulspec.KeyOpenAPITags, userGroupApiTags).
		Param(restful.QueryParameter("id", "用户组ID").DataType("integer").Required(true)).
		Notes(enrichGetGroupApiNotes)
}

func enrichGetGroupTokenApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("获取用户组 token").
		Metadata(restfulspec.KeyOpenAPITags, userGroupApiTags).
		Param(restful.QueryParameter("id", "用户组ID").DataType("integer").Required(true)).
		Notes(enrichGetGroupTokenApiNotes)
}

func enrichDeleteGroupsApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("删除用户组").
		Metadata(restfulspec.KeyOpenAPITags, userGroupApiTags).
		Reads(api.UserGroup{}, "delete group").
		Notes(enrichDeleteGroupsApiNotes)
}

func enrichUpdateGroupTokenApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("更新用户组 token").
		Metadata(restfulspec.KeyOpenAPITags, userGroupApiTags).
		Reads(api.UserGroup{}, "update user group token").
		Notes(enrichUpdateGroupTokenApiNotes)
}

func enrichResetGroupTokenApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("重置用户组 token").
		Metadata(restfulspec.KeyOpenAPITags, userGroupApiTags).
		Reads(api.UserGroup{}, "reset user group token").
		Notes(enrichResetGroupTokenApiNotes)
}
