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
	enrichGetServicesApiNotes = `
请求示例：

~~~
GET /naming/v1/services?参数名=参数值

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}
~~~

应答示例：

~~~json
{
    "code":200000,
    "info":"...",
    "amount":1,
    "size":1,
    "services":[
        {
            "name":"...",
            "namespace":"...",
            "metadata":{

            },
            "ports":"...",
            "business":"...",
            "department":"...",
            "comment":"...",
            "ctime":"...",
            "mtime":"...",
            "total_instance_count": 1,
            "healthy_instance_count":1
        }
    ]
}
~~~

| 参数名 | 类型   | 描述                                                                                    |
| ------ | ------ | ------------------------------------------------------------------------------------- |
| code   | uint32 | 六位返回码                                                                              |
| info   | string | 返回信息                                                                               |
| amount | uint32 | 符合此查询条件的服务总数，例如查询命名空间为default的服务，总数为1000，本次返回100条，则amount为1000 |
| size   | uint32 | 本次查询返回的服务个数，例如查询命名空间为default的服务，总数为1000，本次返回100条，则size为100      |
`
	enrichCreateServicesApiNotes = `
请求示例：

~~~
POST /naming/v1/services

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

[
    {
        "name":"...",
        "namespace":"...",
        "metadata":{

        },
        "ports":"...",
        "business":"...",
        "department":"...",
        "comment":"..."
    }
]
~~~

应答示例：

~~~json
{
    "code":200000,
    "info":"...",
    "size":1,
    "responses":[
        {
            "code":200000,
            "info":"...",
            "service":{
                "name":"...",
                "namespace":"..."
            }
        }
    ]
}
~~~

数据结构：

| 参数名           | 类型               | 描述                                                       | 是否必填 |
| ---------------- | ------------------ | ---------------------------------------------------------- | -------- |
| name             | string             | 服务名                                                     | 是       |
| namespace        | string             | 命名空间                                                   | 是       |
| metadata         | map<string,string> | 服务标签/元数据                                            | 否       |
| ports            | string             | 端口列表，多个port以逗号分隔                               | 否       |
| business         | string             | 服务所属业务，建议填写。                                   | 否       |
| department       | string             | 服务所属部门，建议填写。                                   | 否       |
| comment          | string             | 描述                                                       | 否       |
| user_ids         | []string           | 可以操作该资源的用户，**仅当开启北极星鉴权时生效**         | 否       |
| group_ids        | []string           | 可以操作该资源的用户组，，**仅当开启北极星鉴权时生效**     | 否       |
| remove_user_ids  | []string           | 被移除的可操作该资源的用户，**仅当开启北极星鉴权时生效**   | 否       |
| remove_group_ids | []string           | 被移除的可操作该资源的用户组，**仅当开启北极星鉴权时生效** | 否       |
`
	enrichDeleteServicesApiNotes = `
删除一个不存在的服务，认为删除成功

请求示例：

~~~
POST /naming/v1/services/delete

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

[
    {
        "name":"...",
        "namespace":"..."
    }
]
~~~

应答示例：

~~~json
{
    "code":200000,
    "info":"...",
    "size":1,
    "responses":[
        {
            "code":200000,
            "info":"...",
            "service":{
                "name":"...",
                "namespace":"..."
            }
        }
    ]
}
~~~

数据结构：

| 参数名    | 类型   | 描述     | 是否必填 |
| --------- | ------ | -------- | -------- |
| name      | string | 服务名   | 是       |
| namespace | string | 命名空间 | 是       |
`
	enrichUpdateServicesApiNotes = `
请求示例：

~~~
PUT /naming/v1/services

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

[
    {
        "name":"...",
        "namespace":"...",
        "token":"...",
        "metadata":{

        },
        "ports":"...",
        "business":"...",
        "department":"...",
        "comment":"..."
    }
]
~~~

应答示例：

~~~json
{
    "code":200000,
    "info":"...",
    "size":1,
    "responses":[
        {
            "code":200000,
            "info":"...",
            "service":{
                "name":"...",
                "namespace":"..."
            }
        }
    ]
}
~~~

数据结构：

| 参数名           | 类型               | 描述                                      | 是否必填 | 可否修改 |
| ---------------- | ------------------ | -------------------------------------- | -------- | ------ |
| name             | string             | 服务名                                  | 是       | 否      |
| namespace        | string             | 命名空间                                | 是       | 否       |
| metadata         | map<string,string> | 服务标签/元数据                          | 否       | 是       |
| ports            | string             | 端口列表，多个port以逗号分隔                | 否       | 是       |
| business         | string             | 服务所属业务，建议填写。                    | 否       | 是       |
| department       | string             | 服务所属部门，建议填写。                    | 否       | 是       |
| comment          | string             | 描述                                     | 否       | 是       |
| user_ids         | []string           | 可以操作该资源的用户，**仅当开启北极星鉴权时生效**         | 否       |
| group_ids        | []string           | 可以操作该资源的用户组，，**仅当开启北极星鉴权时生效**     | 否       |
| remove_user_ids  | []string           | 被移除的可操作该资源的用户，**仅当开启北极星鉴权时生效**   | 否       |
| remove_group_ids | []string           | 被移除的可操作该资源的用户组，**仅当开启北极星鉴权时生效** | 否       |
`
	enrichGetServicesCountApiNotes = `
请求示例：
~~~
GET /naming/v1/services/count
# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}
~~~

应答示例：
~~~json
{
  "code": 200000,
  "info": "execute success",
  "amount": 141,
  "size": 0,
  "namespaces": [],
  "services": [],
  "instances": [],
  "routings": [],
  "aliases": [],
  "rateLimits": [],
  "configWithServices": [],
  "users": [],
  "userGroups": [],
  "authStrategies": [],
  "clients": []
}
~~~
`
	enrichCreateServiceAliasApiNotes = `
用户可以为服务创建别名，可以通过别名来访问服务的资源数据。

请求示例：

~~~
POST /naming/v1/service/alias

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

{
    "service":"...",
    "namespace":"...",
    "alias":"...",
    "alias_namespace":"...",
    "comment":"..."
}
~~~

应答示例：

~~~json
{
    "code":200000,
    "info":"...",
    "alias":{
        "service":"...",
        "namespace":"...",
        "alias":"...",
        "alias_namespace":"...",
        "comment":"..."
    }
}
~~~

数据结构：

| 参数名          | 类型   | 描述                   | 必填 |
| --------------- | ------ | ---------------------- | ---- |
| alias           | string | 服务别名               | 是   |
| alias_namespace | string | 服务别名所属命名空间   | 是   |
| service         | string | 指向的服务名           | 是   |
| namespace       | string | 指向的服务所属命名空间 | 是   |
| comment         | string | 服务别名描述           | 否   |
`
	enrichUpdateServiceAliasApiNotes = `
请求示例：

~~~
PUT /naming/v1/service/alias

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}
~~~

应答示例：

~~~json
{
    "code":200000,
    "info":"...",
    "alias":{
        "service":"...",
        "namespace":"...",
        "alias":"...",
        "alias_namespace":"...",
        "comment":"..."
    }
}
~~~

数据结构：

| 参数名          | 类型   | 描述                   | 必填 |
| --------------- | ------ | ---------------------- | ---- |
| alias           | string | 服务别名               | 是   |
| alias_namespace | string | 服务别名所属命名空间   | 是   |
| service         | string | 指向的服务名           | 是   |
| namespace       | string | 指向的服务所属命名空间 | 是   |
| comment         | string | 服务别名描述           | 否   |
`
	enrichGetServiceAliasesApiNotes = `
请求示例：

~~~
GET /naming/v1/service/aliases

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

~~~

应答示例：

| 参数名 | 类型   | 描述                                                           |
| ------ | ------ | ------------------------------------------------------------ |
| size   | uint32 | 本次查询返回的服务别名个数，例如查询命名空间为Production的服务别名，总数为1000，本次返回100条，则size为100      |
| info   | string | 返回信息                                                       |
| code   | uint32 | 六位返回码                                                      |
| amount | uint32 | 符合此查询条件的服务别名总数，例如查询命名空间为Production的服务别名，总数为1000，本次返回100条，则amount为1000 |

~~~json
{
    "code":200000,
    "info":"...",
    "amount":1,
    "size":1,
    "aliases":[
        {
            "alias":"...",
            "alias_namespace":"...",
            "namespace":"...",
            "service":"...",
            "comment":"...",
            "ctime":"...",
            "mtime":"..."
        }
    ]
}
~~~

数据结构：

| 参数名          | 类型   | 描述                         | 必填 |
| --------------- | ------ | ---------------------------- | ---- |
| alias           | string | 服务别名                     | 否   |
| alias_namespace | string | 服务别名所属命名空间         | 否   |
| service         | string | 指向的服务名                 | 否   |
| namespace       | string | 指向的服务所属命名空间       | 否   |
| offset          | int    | 分页偏移，默认0              | 否   |
| limit           | int    | 分页大小，默认为100，最大100 | 否   |
`
	enrichDeleteServiceAliasesApiNotes = `
请求示例：

~~~
POST /naming/v1/service/aliases/delete

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

[
    {
        "alias":"...",
        "alias_namespace":"..."
    }
]
~~~

应答示例：

~~~json
{
    "code":200000,
    "info":"...",
    "size":1,
    "responses":[
        {
            "code":200000,
            "info":"...",
            "alias":{
                "alias":"...",
                "alias_namespace":"..."
            }
        }
    ]
}
~~~

数据结构：

| 参数名          | 类型   | 描述                  | 必填 |
| --------------- | ------ | --------------------- | ---- |
| alias           | string | 服务别名              | 是   |
| alias_namespace | string | 服务别名所属命名空间  | 是   |
`

	enrichGetServiceOwnerApiNotes = ``
)
