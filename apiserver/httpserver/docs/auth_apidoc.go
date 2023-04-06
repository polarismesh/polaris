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

package docs

import (
	"github.com/emicklei/go-restful/v3"
	restfulspec "github.com/polarismesh/go-restful-openapi/v2"
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
)

var (
	authApiTags      = []string{"AuthRule"}
	usersApiTags     = []string{"Users"}
	userGroupApiTags = []string{"Users"}
)

func EnrichAuthStatusApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("查询鉴权开关信息").
		Metadata(restfulspec.KeyOpenAPITags, authApiTags).
		Notes(enrichAuthStatusApiNotes)
}

func EnrichCreateStrategyApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("创建鉴权策略").
		Metadata(restfulspec.KeyOpenAPITags, authApiTags).
		Reads(apisecurity.AuthStrategy{}, "create auth strategy").
		Notes(enrichCreateStrategyApiNotes)
}

func EnrichUpdateStrategiesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("更新鉴权策略").
		Metadata(restfulspec.KeyOpenAPITags, authApiTags).
		Reads(apisecurity.AuthStrategy{}, "update auth strategy").
		Notes(enrichUpdateStrategiesApiNotes)
}

func EnrichGetStrategiesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("查询鉴权策略列表").
		Metadata(restfulspec.KeyOpenAPITags, authApiTags).
		Param(restful.QueryParameter("id", "策略ID").DataType(DataType_String).
			Required(false)).
		Param(restful.QueryParameter("name", "策略名称, 当前仅提供全模糊搜索").DataType(DataType_String).
			Required(false)).
		Param(restful.QueryParameter("default", "“0” 查询自定义策略；“1” 查询默认策略；"+
			"不填则为查询（默认+自定义）鉴权策略").DataType(DataType_String).Required(false)).
		Param(restful.QueryParameter("res_id", "资源ID").DataType(DataType_String).
			Required(false)).
		Param(restful.QueryParameter("res_type", "资源类型, namespace、service、config_group").
			DataType(DataType_String).Required(false)).
		Param(restful.QueryParameter("principal_id", "成员ID").DataType(DataType_String).
			Required(false)).
		Param(restful.QueryParameter("principal_type", "成员类型, user、group").
			DataType(DataType_String).Required(false)).
		Param(restful.QueryParameter("show_detail", "是否显示策略详细").DataType("boolean").
			Required(false)).
		Param(restful.QueryParameter("offset", "查询偏移量, 默认为0").DataType(DataType_Integer).
			Required(false).DefaultValue("0")).
		Param(restful.QueryParameter("limit", "本次查询条数, 最大为100").DataType(DataType_Integer).
			Required(false)).
		Notes(enrichGetStrategiesApiNotes)
}

func EnrichGetPrincipalResourcesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("获取鉴权策略详细").
		Metadata(restfulspec.KeyOpenAPITags, authApiTags).
		Param(restful.QueryParameter("principal_id", "策略ID").
			DataType(DataType_String).
			Required(true)).
		Param(restful.QueryParameter("principal_type", "Principal类别，user/group").
			DataType(DataType_String).
			Required(true)).
		Notes(enrichGetPrincipalResourcesApiNotes)
}

func EnrichGetStrategyApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("获取鉴权策略详细").
		Metadata(restfulspec.KeyOpenAPITags, authApiTags).
		Param(restful.QueryParameter("id", "策略ID").DataType(DataType_String).Required(true)).
		Notes(enrichGetStrategyApiNotes)
}

func EnrichDeleteStrategiesApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("删除鉴权策略").
		Metadata(restfulspec.KeyOpenAPITags, authApiTags).
		Reads(apisecurity.AuthStrategy{}, "delete auth strategy").
		Notes(enrichDeleteStrategiesApiNotes)
}

func EnrichLoginApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("用户登录").
		Metadata(restfulspec.KeyOpenAPITags, usersApiTags).
		Reads(apisecurity.LoginRequest{}, "用户登录接口").
		Notes(enrichLoginApiNotes)
}

func EnrichGetUsersApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("获取用户").
		Metadata(restfulspec.KeyOpenAPITags, usersApiTags).
		Param(restful.QueryParameter("id", "用户ID").
			DataType(DataType_String).
			Required(false)).
		Param(restful.QueryParameter("name", "用户名称, 当前仅提供全模糊搜索").
			DataType(DataType_String).
			Required(false)).
		Param(restful.QueryParameter("source", "用户来源").DataType(DataType_String).Required(false)).
		Param(restful.QueryParameter("group_id", "用户组ID, 用于查询某个用户组下用户列表").DataType(DataType_String).
			Required(false)).
		Param(restful.QueryParameter("offset", "查询偏移量, 默认为0").DataType(DataType_Integer).Required(false).
			DefaultValue("0")).
		Param(restful.QueryParameter("limit", "本次查询条数, 最大为100").DataType(DataType_Integer).Required(false)).
		Notes(enrichGetUsersApiNotes)
}

func EnrichCreateUsersApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("创建用户").
		Metadata(restfulspec.KeyOpenAPITags, usersApiTags).
		Reads(apisecurity.User{}, "create user").
		Notes(enrichCreateUsersApiNotes)
}

func EnrichDeleteUsersApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("删除用户").
		Metadata(restfulspec.KeyOpenAPITags, usersApiTags).
		Reads(apisecurity.User{}, "delete user").
		Notes(enrichDeleteUsersApiNotes)
}

func EnrichUpdateUserApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("更新用户").
		Metadata(restfulspec.KeyOpenAPITags, usersApiTags).
		Reads(apisecurity.User{}, "update user").
		Notes(enrichUpdateUserApiNotes)
}

func EnrichUpdateUserPasswordApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("更新用户密码").
		Metadata(restfulspec.KeyOpenAPITags, usersApiTags).
		Reads(apisecurity.ModifyUserPassword{}, "update user password").
		Notes(enrichUpdateUserPasswordApiNotes)
}

func EnrichGetUserTokenApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("获取用户Token").
		Metadata(restfulspec.KeyOpenAPITags, usersApiTags).
		Param(restful.QueryParameter("id", "用户ID").DataType(DataType_String).Required(true)).
		Notes(enrichGetUserTokenApiNotes)
}

func EnrichUpdateUserTokenApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("更新用户Token").
		Metadata(restfulspec.KeyOpenAPITags, usersApiTags).
		Reads(apisecurity.User{}, "update user token").
		Notes(enrichUpdateUserTokenApiNotes)
}

func EnrichResetUserTokenApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("重置用户Token").
		Metadata(restfulspec.KeyOpenAPITags, usersApiTags).
		Reads(apisecurity.User{}, "reset user token").
		Notes(enrichResetUserTokenApiNotes)
}

func EnrichCreateGroupApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("创建用户组").
		Metadata(restfulspec.KeyOpenAPITags, userGroupApiTags).
		Reads(apisecurity.UserGroup{}, "create group").
		Notes(enrichCreateGroupApiNotes)
}

func EnrichUpdateGroupsApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("更新用户组").
		Metadata(restfulspec.KeyOpenAPITags, userGroupApiTags).
		Reads(apisecurity.UserGroup{}, "update group").
		Notes(enrichUpdateGroupsApiNotes)
}

func EnrichGetGroupsApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("查询用户组列表").
		Metadata(restfulspec.KeyOpenAPITags, userGroupApiTags).
		Param(restful.QueryParameter("id", "用户组ID").DataType(DataType_String).Required(false)).
		Param(restful.QueryParameter("name", "用户组名称, 当前仅提供全模糊搜索").
			DataType(DataType_String).Required(false)).
		Param(restful.QueryParameter("user_id", "用户ID, 用于查询某个用户关联的用户组列表").DataType(DataType_String).
			Required(false)).
		Param(restful.QueryParameter("offset", "查询偏移量, 默认为0").DataType(DataType_Integer).Required(false).
			DefaultValue("0")).
		Param(restful.QueryParameter("limit", "本次查询条数, 最大为100").DataType(DataType_Integer).Required(false)).
		Notes(enrichGetGroupsApiNotes)
}

func EnrichGetGroupApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("获取用户组详情").
		Metadata(restfulspec.KeyOpenAPITags, userGroupApiTags).
		Param(restful.QueryParameter("id", "用户组ID").DataType(DataType_Integer).Required(true)).
		Notes(enrichGetGroupApiNotes)
}

func EnrichGetGroupTokenApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("获取用户组 token").
		Metadata(restfulspec.KeyOpenAPITags, userGroupApiTags).
		Param(restful.QueryParameter("id", "用户组ID").DataType(DataType_Integer).Required(true)).
		Notes(enrichGetGroupTokenApiNotes)
}

func EnrichDeleteGroupsApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("删除用户组").
		Metadata(restfulspec.KeyOpenAPITags, userGroupApiTags).
		Reads(apisecurity.UserGroup{}, "delete group").
		Notes(enrichDeleteGroupsApiNotes)
}

func EnrichUpdateGroupTokenApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("更新用户组 token").
		Metadata(restfulspec.KeyOpenAPITags, userGroupApiTags).
		Reads(apisecurity.UserGroup{}, "update user group token").
		Notes(enrichUpdateGroupTokenApiNotes)
}

func EnrichResetGroupTokenApiDocs(r *restful.RouteBuilder) *restful.RouteBuilder {
	return r.
		Doc("重置用户组 token").
		Metadata(restfulspec.KeyOpenAPITags, userGroupApiTags).
		Reads(apisecurity.UserGroup{}, "reset user group token").
		Notes(enrichResetGroupTokenApiNotes)
}
