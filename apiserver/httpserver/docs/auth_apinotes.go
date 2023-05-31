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

const (
	enrichAuthStatusApiNotes = `

请求示例：

~~~
GET /core/v1/auth/status
~~~



响应示例：

~~~
{
	"code": 200000,
	"info": "execute success",
	"optionSwitch":
	{
		"auth": "true"
	}
}
~~~
`

	enrichCreateStrategyApiNotes = `
请求示例：

~~~
POST /core/v1/auth/strategy
Header X-Polaris-Token: {访问凭据}
~~~

~~~json
{
  "name": "xxx", // 策略名称
  "comment": "xxx",
  "principals": {
    "users": [
      {
          "id": "xxx"
      }
    ],
    "groups": [
      {
          "id": "xxx"
      }
    ]
  },
  "resources": {
    "namespaces": [
      {
          "id": "Polaris"
      }
    ],
    "services": [
      {
          "id": "Polaris"
      }
    ],
    "config_groups": [
      {
          "id": "Polaris"
      }
    ]
  }
}
~~~

响应示例：

~~~json
{
    "code": 200000,
    "info": "execute success"
}
~~~
`

	enrichUpdateStrategiesApiNotes = `
请求示例：

~~~
PUT /core/v1/auth/strategies
Header X-Polaris-Token: {访问凭据}
~~~

~~~json
[
    {
        "id": "xxx", // 策略ID
        "comment": "xxx",
        "add_principals": {
            "users": [{
                "id": "xxx"
            },{
                "id": "xxx"
            }],
            "groups": [{
                "id": "xxx"
            }]
        },
        "remove_principals": {
            "users": [{"id": "xxx"}],
            "groups": [{
                "id": "xxx"
            }]
        },
        "add_resources": {
          "namespaces": [
            {
                "id":"xxx"
            }],
          "services": [{
                "id":"xxx"
           }],
          "config_groups": [
              {
                "id":"xxx"
           }]
        },
        "remove_resources": {
          "namespaces": [
            {
                "id": "xxx"
            }
          ],
          "services": [],
          "config_groups": []
        }
    }
]
~~~

响应示例：

~~~json
{
    "code": 200000,
    "info": "execute success",
    "size": 1,
    "responses": [
        {
            "code": 200000,
            "info": "execute success"
        }
    ]
}
~~~
`

	enrichGetStrategiesApiNotes = `
请求示例：

~~~
GET /core/v1/auth/strategies?{key}={value}
Header X-Polaris-Token: {访问凭据}
~~~

支持的URL参数

| 参数名         | 类型   | 描述                                                                  | 是否必填 |
|----------------|--------|---------------------------------------------------------------------|---------|
| id             | string | 策略ID                                                                | 否       |
| name           | string | 策略名称, 当前仅提供全模糊搜索                                        | 否       |
| default        | string | “0” 查询自定义策略；“1” 查询默认策略；不填则为查询（默认+自定义）鉴权策略 | 否       |
| res_id         | string | 资源ID                                                                | 否       |
| res_type       | string | 资源类型, namespace、service、config_group                              | 否       |
| principal_id   | string | 成员ID                                                                | 否       |
| principal_type | string | 成员类型, user、group                                                  | 否       |
| show_detail    | bool   | 是否显示策略详细                                                      | 否       |
| offset         | int    | 查询偏移量, 默认为0                                                   | 否       |
| limit          | int    | 本次查询条数, 最大为100                                               | 否       |


响应示例：

~~~json
{
    "code": 200000,
    "info": "execute success",
    "amount": 1,
    "size": 1,
    "authStrategies": [
        {
            "id": "xxx",
            "name": "xxx",
            "principals": {
                "users": [
                    {
                        "id": "xxx",
                        "name": "xxx"
                    }
                ],
                "groups": [
                    {
                        "id": "xxx",
                        "name": "xxx"
                    }
                ]
            },
            "resources": {
                "strategy_id": null,
                "namespaces": [],
                "services": [
                    {
                        "id": "xxx",
                        "namespace": "default",
                        "name": "Demo-1"
                    },
                    {
                        "id": "xxx",
                        "namespace": "default",
                        "name": "Demo-2"
                    }
                ],
                "config_groups": []
            },
            "action": "READ_WRITE",
            "comment": "Default Strategy",
            "ctime": "2022-02-09 19:48:53",
            "mtime": "2022-02-09 19:48:53",
            "default_strategy": true
        },
    ]
}
~~~
`

	enrichGetPrincipalResourcesApiNotes = `
请求示例：

~~~
GET /core/v1/auth/principal/resources?principal_id=xxx&principal_type=user
Header X-Polaris-Token: {访问凭据}
~~~

| 参数名         | 类型   | 描述                     | 是否必填 |
|----------------|--------|------------------------|---------|
| principal_id   | string | 策略ID                   | 是       |
| principal_type | string | Principal类别，user/group | 是       |


响应示例：

~~~json
{
    "code": 200000,
    "info": "execute success",
    "resources": {
        "namespaces": [
            {
                "id": "xxx",
                "namespace": "xxx",
                "name": "xxx"
            }
        ],
        "services": [
            {
                "id": "xxx",
                "namespace": "Polaris",
                "name": "xxx"
            }
        ],
        "config_groups": [{
                "id": "xxx",
                "namespace": "xxx",
                "name": "xxx"
            }
        ]
    }
}
~~~
`

	enrichGetStrategyApiNotes = `
根据策略ID查询该策略的具体详细信息

请求示例：

~~~
GET /core/v1/auth/strategy/detail?id=xxx
Header X-Polaris-Token: {访问凭据}
~~~

| 参数名 | 类型   | 描述   | 是否必填 |
|--------|--------|------|---------|
| id     | string | 策略ID | 是       |



响应示例：

~~~json
{
    "code": 200000,
    "info": "execute success",
    "authStrategy": {
        "id": "xxx",
        "name": "xxx",
        "principals": {
            "users": [
                {
                    "id": "xxx",
                    "name": "xxx"
                }
            ],
            "groups": []
        },
        "resources": {
            "namespaces": [],
            "services": [
                {
                    "id": "xxx",
                    "namespace": "default",
                    "name": "Demo-1"
                },
                {
                    "id": "xxx",
                    "namespace": "default",
                    "name": "Demo-2"
                }
            ],
            "config_groups": []
        },
        "action": "READ_WRITE",
        "comment": "Default Strategy",
        "ctime": "2022-02-09 19:43:26",
        "mtime": "2022-02-15 23:20:48",
        "default_strategy": true
    }
}
~~~
`

	enrichDeleteStrategiesApiNotes = `
请求示例：

~~~
POST /core/v1/auth/strategies/delete
Header X-Polaris-Token: {访问凭据}
~~~

~~~json
[
    {
        "id": "xxx" // 策略ID
    }
]
~~~

响应示例：

~~~json
{
    "code": 200000,
    "info": "execute success",
    "size": 1,
    "responses": [
        {
            "code": 200000,
            "info": "execute success"
        }
    ]
}
~~~
`
	enrichLoginApiNotes = `
用于控制台进行用户登录操作

请求示例：

~~~
POST /core/v1/user/login
~~~

| 参数名   | 类型   | 描述     | 是否必填 |
|----------|--------|--------|---------|
| name     | string | 用户名   | 是       |
| password | string | 用户密码 | 是       |


应答示例：

~~~json
{
	"code": 200000,
	"info": "execute success",
	"loginResponse": {
		"token": "xxx",
		"name": "xxx",
		"user_id": "xxx",
		"role": "xxx"
	}
}
~~~

| 参数名        |         | 类型   | 描述                                                    |
|---------------|---------|--------|-------------------------------------------------------|
| code          |         | uint32 | 六位返回码                                              |
| info          |         | string | 返回信息                                                |
| loginResponse |         |        | 命名空间                                                |
|               | token   | string | 用户Token, 用于接口请求访问                             |
|               | name    | string | 用户名                                                  |
|               | role    | string | 当前用户角色, (admin:超级账户, main:主账户, sub:子账户) |
|               | user_id | string | 当前用户ID                                              |

`

	enrichGetUsersApiNotes = `
根据相关条件对用户列表进行查询

请求示例：

~~~
GET /core/v1/users?id=xxx&name=xxx&source=xxx&group_id=xxx&offset=xxx&limit=xxx
Header X-Polaris-Token: {访问凭据}
~~~

| 参数名   | 类型   | 描述                                   | 是否必填 |
|----------|--------|--------------------------------------|---------|
| id       | string | 用户ID                                 | 否       |
| name     | string | 用户名称, 当前仅提供全模糊搜索         | 否       |
| source   | string | 用户来源                               | 否       |
| group_id | string | 用户组ID, 用于查询某个用户组下用户列表 | 否       |
| offset   | int    | 查询偏移量, 默认为0                    | 否       |
| limit    | int    | 本次查询条数, 最大为100                | 否       |


响应示例：

~~~json
{
	"code": 200000,
	"info": "execute success",
	"amount": 1,
	"size": 1,
	"users": [
		{
			"id": "xxx",
			"name": "xxx",
			"source": "",
			"auth_token": "",
			"token_enable": true,
			"comment": "",
			"ctime": "2022-02-09 19:48:53",
			"mtime": "2022-02-09 19:48:53",
		}
	]
}
~~~
`

	enrichCreateUsersApiNotes = `
批量创建用户至北极星

请求示例：

~~~
POST /core/v1/users
Header X-Polaris-Token: {访问凭据}
~~~

~~~json
[
	{
	"name": "polarismesh",
	"password": "polarismesh",
	"comment": "polarismesh",
	"source": "Polaris"
	}
]
~~~

| 参数名   | 类型   | 描述     | 是否必填 |
|----------|--------|--------|---------|
| name     | string | 用户名   | 是       |
| password | string | 用户密码 | 是       |
| comment  | string | 用户备注 | 否       |
| source   | string | 用户来源 | 否       |


应答示例：

~~~json
{
	"code": 200000,
	"info": "execute success",
	"size": 1,
	"responses": [
		{
			"code": 200000,
			"info": "execute success"
		}
	]
}
~~~

数据结构：

> user

| 参数名       | 类型   | 描述                 |
|--------------|--------|--------------------|
| id           | string | 用户ID               |
| name         | string | 用户名称             |
| password     | string | 用户密码             |
| source       | string | 用户来源             |
| auth_token   | string | 用户访问凭据         |
| token_enable | bool   | 用户访问凭据是否可用 |
| comment      | string | 用户备注信息         |
| ctime        | string | 用户创建时间         |
| mtime        | string | 用户修改时间         |
`

	enrichDeleteUsersApiNotes = `
批量删除北极星中的用户

请求示例

~~~
POST /core/v1/users/delete
Header X-Polaris-Token: {访问凭据}
~~~

~~~json
[
	{
		"id": "xxx"
	}
]
~~~

| 参数名 | 类型   | 描述   | 是否必填 |
|--------|--------|------|---------|
| id     | string | 用户ID | 是       |


响应示例：

~~~json
{
	"code": 200000,
	"info": "execute success",
	"responses": [
		{
			"code": 200000,
			"info": "execute success"
		}
	]
}
~~~
`

	enrichUpdateUserApiNotes = `
更新用户的备注信息数据

请求示例：

~~~
PUT /core/v1/user
Header X-Polaris-Token: {访问凭据}
~~~

~~~json
{
	"id": "xxx",
	"comment": "polarismesh"
}
~~~

| 参数名  | 类型   | 描述     | 是否必填 |
|---------|--------|--------|---------|
| id      | string | 用户ID   | 是       |
| comment | string | 用户备注 | 否       |
`

	enrichUpdateUserPasswordApiNotes = `
用户重新更新密码, 如果密码更新成功, 则会一并更新对应用户的访问凭据

请求示例：

~~~
PUT /core/v1/user/password
Header X-Polaris-Token: {访问凭据}
~~~

~~~json
{
	"id": "xxx",
	"old_password": "xxx",
	"new_password": "xxx"
}
~~~

| 参数名       | 类型   | 描述   | 是否必填 |
|--------------|--------|------|---------|
| id           | string | 用户ID | 是       |
| old_password | string | 旧密码 | 否       |
| new_password | string | 新密码 | 否       |


响应示例：
`

	enrichGetUserTokenApiNotes = `
查询用户的Token凭据信息，通过用户ID或者用户名进行查询

请求示例：

~~~
GET /core/v1/user/token?id=xxx
Header X-Polaris-Token: {访问凭据}
~~~

| 参数名 | 类型   | 描述   | 是否必填 |
|--------|--------|------|---------|
| id     | string | 用户ID | 是       |

响应示例：

~~~json
{
	"code": 200000,
	"info": "execute success",
	"user": {
		"id": "xxx",
		"auth_token": "xxx",
		"token_enable": true
	}
}
~~~
`

	enrichUpdateUserTokenApiNotes = `
对用户Token的使用状态进行修改, 如果用户的Token被设置为禁用状态, 则该Token后续无法用于访问北极星接口以及资源, 需使用主账户或者超级账户进行解封

请求示例：

~~~
PUT /core/v1/user/token/status
Header X-Polaris-Token: {访问凭据}
~~~

~~~json
{
	"id": "xxx",
	"token_enable": false
}
~~~

| 参数名       | 类型   | 描述                                       | 是否必填 |
|--------------|--------|------------------------------------------|---------|
| id           | string | 用户ID                                     | 是       |
| token_enable | bool   | 当前Token可用状态, true为启用, false为禁用 | 是       |


响应示例：

~~~json
{
	"code": 200000,
	"info": "execute success"
}
~~~
`

	enrichResetUserTokenApiNotes = `
重置用户的Token, 当Token重置之后，原先的Token失效并且无法进行访问北极星接口以及资源

请求示例：

~~~
PUT /core/v1/user/token/refresh
Header X-Polaris-Token: {访问凭据}
~~~

~~~json
{
	"id": "xxx"
}
~~~

| 参数名 | 类型   | 描述   | 是否必填 |
|--------|--------|------|---------|
| id     | string | 用户ID | 是       |


响应示例：

~~~json
{
	"code": 200000,
	"info": "execute success"
}
~~~
`

	enrichCreateGroupApiNotes = `
请求示例：

~~~
POST /core/v1/usergroup
Header X-Polaris-Token: {访问凭据}
~~~

~~~json
{
	"name": "GROUP_1",
	"comment": "",
	"relation": {
		"users": [
			{
				"id": "xxx"
			}, {
				"id": "xxx"
			}
		]
	}
}
~~~

| 参数名   | 类型     | 描述                   |
|----------|----------|----------------------|
| name     | string   | 用户组名称             |
| comment  | string   | 用户组备注信息         |
| relation | user列表 | 当前用户组关联的用户ID |

响应示例：

~~~json
{
	"code": 200000,
	"info": "execute success"
}
~~~
`

	enrichUpdateGroupsApiNotes = `
请求示例：

~~~
PUT /core/v1/usergroup
Header X-Polaris-Token: {访问凭据}
~~~

~~~json
[
	{
		"id": "xxx",
		"comment": "xxx",
		"add_relations": {
			"users": [{
				"id": "xxx"
			}, {
				"id": "xxx"
			}]
		},
		"remove_relations": {
			"users": [{
				"id": "xxx"
			}, {
				"id": "xxx"
			}]
		}
	}
]
~~~

| 参数名          | 类型     | 描述                       |
|-----------------|----------|--------------------------|
| id              | string   | 用户组ID                   |
| comment         | string   | 用户组备注信息             |
| add_relation    | user列表 | 当前用户组追加关联的用户ID |
| remove_relation | user列表 | 当前用户组移除关联的用户ID |

响应示例：

~~~json
{
	"code": 200000,
	"info": "execute success",
	"size": 1,
	"responses": [
		{
			"code": 200000,
			"info": "execute success"
		}
	]
}
~~~
`

	enrichGetGroupsApiNotes = `
请求示例：

~~~
GET /core/v1/usergroups?{key}={value}
Header X-Polaris-Token: {访问凭据}
~~~

支持的URL参数

| 参数名  | 类型   | 描述                                     | 是否必填 |
|---------|--------|----------------------------------------|---------|
| id      | string | 用户组ID                                 | 否       |
| name    | string | 用户组名称, 当前仅提供全模糊搜索         | 否       |
| user_id | string | 用户ID, 用于查询某个用户关联的用户组列表 | 否       |
| offset  | int    | 查询偏移量, 默认为0                      | 否       |
| limit   | int    | 本次查询条数, 最大为100                  | 否       |


响应示例：

~~~json
{
	"code": 200000,
	"info": "execute success",
	"amount": 1,
	"size": 1,
	"userGroups": [
		{
			"id": "xxx",
			"name": "xxx",
			"auth_token": null,
			"token_enable": true,
			"comment": "",
			"ctime": "2022-02-09 21:46:33",
			"mtime": "2022-02-09 21:46:33",
			"user_count": 1
		}
	]
}
~~~

数据结构：

> userGroups

| 参数名       | 类型   | 描述                   |
|--------------|--------|----------------------|
| id           | string | 用户组ID               |
| name         | string | 用户组名称             |
| auth_token   | string | 用户组访问凭据         |
| token_enable | bool   | 用户组访问凭据是否可用 |
| comment      | string | 用户组备注信息         |
| ctime        | string | 用户组创建时间         |
| mtime        | string | 用户组修改时间         |
| user_count   | int    | 当前用户组下用户的数量 |
`

	enrichGetGroupApiNotes = `
请求示例：

~~~
GET /core/v1/usergroup/detail?id=xxx
Header X-Polaris-Token: {访问凭据}
~~~


响应示例：

~~~json
{
	"code": 200000,
	"info": "execute success",
	"userGroup": {
		"id": "xxx",
		"name": "xxx",
		"token_enable": true,
		"comment": "",
		"ctime": "2022-02-09 21:46:33",
		"mtime": "2022-02-09 21:46:33",
		"relation": {
		"users": [
				{
					"id": "xxx",
					"name": "xxx",
					"source": "",
					"token_enable": true,
					"comment": "",
					"ctime": "2022-02-09 19:48:53",
					"mtime": "2022-02-09 19:48:53",
				}
			]
		},
		"user_count": 1
	}
}
~~~
`

	enrichGetGroupTokenApiNotes = `
请求示例：

~~~
GET /core/v1/usergroup/token?id=xxx
Header X-Polaris-Token: {访问凭据}
~~~

响应示例：

~~~json
{
	"code": 200000,
	"info": "execute success",
	"userGroup": {
		"id": "xxx",
		"auth_token": "xxx",
		"token_enable": true
	}
}
~~~
`

	enrichDeleteGroupsApiNotes = `
请求示例：

~~~
POST /core/v1/usergroups/delete
Header X-Polaris-Token: {访问凭据}
~~~

~~~json
[
	{
		"id": "xxx" // 用户组ID
	}
]
~~~

响应示例：

~~~json
{
	"code": 200000,
	"info": "execute success",
	"responses": [
		{
			"code": 200000,
			"info": "execute success"
		}
	]
}
~~~
`

	enrichUpdateGroupTokenApiNotes = `
请求示例：

~~~
PUT /core/v1/usergroup/token/status
Header X-Polaris-Token: {访问凭据}
~~~

~~~json
{
	"id": "xxx", // 用户ID
	"token_enable": false // 当前Token可用状态, true为启用, false为禁用
}
~~~

响应示例：

~~~json
{
	"code": 200000,
	"info": "execute success"
}
~~~
`

	enrichResetGroupTokenApiNotes = `
请求示例：

~~~
PUT /core/v1/usergroup/token/refresh
Header X-Polaris-Token: {访问凭据}
~~~

~~~json
{
	"id": "xxx" // 用户ID
}
~~~

响应示例：

~~~json
{
	"code": 200000,
	"info": "execute success"
}
~~~
`
)
