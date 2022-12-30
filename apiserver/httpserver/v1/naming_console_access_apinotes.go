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

package v1

const (
	enrichGetNamespacesApiNotes = `

| 参数名 | 类型   | 描述                                             | 是否必填 |
| ------ | ------ | ------------------------------------------------ | -------- |
| name   | string | 命名空间唯一名称                                 | 是       |
| offset | uint   | 查询偏移量                                       | 否       |
| limit  | uint   | 查询条数, **最多查询100条**                      | 否       |


请求示例: 

~~~
GET /naming/v1/namespaces?name=&offset=&limit=

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}
~~~

应答示例: 
~~~json
{
    "code": 200000,
    "info": "execute success",
    "amount": 0,
    "size": 3,
    "namespaces": [
        {
            "name": "...",
            "comment": "",
            "ctime": "2021-11-22 23:50:52",
            "mtime": "2021-11-22 23:50:52"
        },
        {
            "name": "...",
            "comment": "",
            "ctime": "2021-11-22 23:50:52",
            "mtime": "2021-11-22 23:50:52"
        }
    ]
}
~~~
`
	enrichCreateNamespacesApiNotes = `
| 参数名           | 类型     | 描述                                                       | 是否必填 |
| ---------------- | -------- | ---------------------------------------------------------- | -------- |
| name             | string   | 命名空间唯一名称                                           | 是       |
| comment          | string   | 描述                                                       | 否       |
| user_ids         | []string | 可以操作该资源的用户, **仅当开启北极星鉴权时生效**         | 否       |
| group_ids        | []string | 可以操作该资源的用户组, , **仅当开启北极星鉴权时生效**     | 否       |
| remove_user_ids  | []string | 被移除的可操作该资源的用户, **仅当开启北极星鉴权时生效**   | 否       |
| remove_group_ids | []string | 被移除的可操作该资源的用户组, **仅当开启北极星鉴权时生效** | 否       |


请求示例: 

~~~
POST /naming/v1/namespaces

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

[
    {
        "name": "...",
        "comment": "..."
    }
]
~~~

应答示例: 
~~~json
{
    "code":200000,
    "info":"...",
    "size":1,
    "responses":[
        {
            "code":200000,
            "info":"execute success",
            "namespace":{
                "name":"...",
                "token":"..."
            }
        }
    ]
}
~~~
`
	enrichUpdateNamespacesApiNotes = `
| 参数名           | 类型     | 描述                                                       | 是否必填 |
| ---------------- | -------- | ---------------------------------------------------------- | -------- |
| name             | string   | 命名空间唯一名称                                           | 是       |
| comment          | string   | 描述                                                       | 否       |
| token            | string   | 命名空间的token, 用于权限鉴定                              | 是       |
| user_ids         | []string | 可以操作该资源的用户, **仅当开启北极星鉴权时生效**         | 否       |
| group_ids        | []string | 可以操作该资源的用户组, , **仅当开启北极星鉴权时生效**     | 否       |
| remove_user_ids  | []string | 被移除的可操作该资源的用户, **仅当开启北极星鉴权时生效**   | 否       |
| remove_group_ids | []string | 被移除的可操作该资源的用户组, **仅当开启北极星鉴权时生效** | 否       |

请求示例: 

~~~
PUT /naming/v1/namespaces

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

[
    {
        "name": "...",
        "comment": "..."
    }
]
~~~

应答示例: 
~~~json
{
    "code": 200000,
    "info": "execute success",
    "size": 0
}
~~~
`
	enrichDeleteNamespacesApiNotes = `
| 参数名 | 类型   | 描述                          | 是否必填 |
| ------ | ------ | ----------------------------- | -------- |
| name   | string | 命名空间唯一名称              | 是       |
| token  | string | 命名空间的token, 用于权限鉴定 | 是       |

请求示例: 

~~~
POST /naming/v1/namespaces/delete

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

[
    {
        "name": "...",
    }
]
~~~

应答示例: 
~~~json
{
    "code": 200000,
    "info": "execute success",
    "size": 0
}
~~~
`
	enrichGetServicesApiNotes = `
请求示例: 

~~~
GET /naming/v1/services?参数名=参数值

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}
~~~

| 参数名     | 类型   | 描述                                                               | 是否必填 |
| ---------- | ------ | ------------------------------------------------------------------ | -------- |
| name       | string | 服务名                                                             | 否       |
| namespace  | string | 命名空间                                                           | 否       |
| business   | string | 业务, 默认模糊查询                                                 | 否       |
| department | string | 部门                                                               | 否       |
| host       | string | 实例IP, **多个IP以英文逗号分隔**                                   | 否       |
| port       | string | **实例端口**, **多个端口以英文逗号分隔**                           | 否       |
| keys       | string | 服务元数据名, keys和values需要同时填写, 目前只支持查询一组元数据。 | 否       |
| values     | string | 服务元数据值, keys和values需要同时填写, 目前只支持查询一组元数据。 | 否       |
| offset     | int    | 默认为0                                                            | 否       |
| limit      | int    | 默认为100, 最大100                                                 | 否       |

应答示例: 

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
| ------ | ------ | --------------------------------------------------------------------------------------- |
| code   | uint32 | 六位返回码                                                                              |
| info   | string | 返回信息                                                                                |
| amount | uint32 | 符合此查询条件的服务总数, 例如查询命名空间为default的服务, 总数为1000, 本次返回100条, 则amount为1000 |
| size   | uint32 | 本次查询返回的服务个数, 例如查询命名空间为default的服务, 总数为1000, 本次返回100条, 则size为100      |
`
	enrichCreateServicesApiNotes = `
请求示例: 

~~~
POST /naming/v1/services

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
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

应答示例: 

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

数据结构: 

| 参数名           | 类型               | 描述                                                       | 是否必填 |
| ---------------- | ------------------ | ---------------------------------------------------------- | -------- |
| name             | string             | 服务名                                                     | 是       |
| namespace        | string             | 命名空间                                                   | 是       |
| metadata         | map<string,string> | 服务标签/元数据                                            | 否       |
| ports            | string             | 端口列表, 多个port以逗号分隔                               | 否       |
| business         | string             | 服务所属业务, 建议填写。                                   | 否       |
| department       | string             | 服务所属部门, 建议填写。                                   | 否       |
| comment          | string             | 描述                                                       | 否       |
| user_ids         | []string           | 可以操作该资源的用户, **仅当开启北极星鉴权时生效**         | 否       |
| group_ids        | []string           | 可以操作该资源的用户组, , **仅当开启北极星鉴权时生效**     | 否       |
| remove_user_ids  | []string           | 被移除的可操作该资源的用户, **仅当开启北极星鉴权时生效**   | 否       |
| remove_group_ids | []string           | 被移除的可操作该资源的用户组, **仅当开启北极星鉴权时生效** | 否       |
`
	enrichDeleteServicesApiNotes = `
删除一个不存在的服务, 认为删除成功

请求示例: 

~~~
POST /naming/v1/services/delete

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

[
    {
        "name":"...",
        "namespace":"..."
    }
]
~~~

应答示例: 

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

数据结构: 

| 参数名    | 类型   | 描述     | 是否必填 |
| --------- | ------ | -------- | -------- |
| name      | string | 服务名   | 是       |
| namespace | string | 命名空间 | 是       |
`
	enrichUpdateServicesApiNotes = `
请求示例: 

~~~
PUT /naming/v1/services

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
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

应答示例: 

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

数据结构: 

| 参数名           | 类型               | 描述                              | 是否必填 | 可否修改 |
| ---------------- | ------------------ | --------------------------------- | -------- | -------- |
| name             | string             | 服务名                            | 是       | 否       |
| namespace        | string             | 命名空间                          | 是       | 否       |
| metadata         | map<string,string> | 服务标签/元数据                   | 否       | 是       |
| ports            | string             | 端口列表, 多个port以逗号分隔      | 否       | 是       |
| business         | string             | 服务所属业务, 建议填写。          | 否       | 是       |
| department       | string             | 服务所属部门, 建议填写。          | 否       | 是       |
| comment          | string             | 描述                          | 否       | 是       |
| user_ids         | []string           | 可以操作该资源的用户, **仅当开启北极星鉴权时生效**         | 否       |
| group_ids        | []string           | 可以操作该资源的用户组, , **仅当开启北极星鉴权时生效**     | 否       |
| remove_user_ids  | []string           | 被移除的可操作该资源的用户, **仅当开启北极星鉴权时生效**   | 否       |
| remove_group_ids | []string           | 被移除的可操作该资源的用户组, **仅当开启北极星鉴权时生效** | 否       |
`
	enrichGetServicesCountApiNotes = `
请求示例: 
~~~
GET /naming/v1/services/count
# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}
~~~

应答示例: 
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
  "platforms": [],
  "users": [],
  "userGroups": [],
  "authStrategies": [],
  "clients": []
}
~~~
`
	enrichCreateServiceAliasApiNotes = `
用户可以为服务创建别名, 可以通过别名来访问服务的资源数据。

请求示例: 

~~~
POST /naming/v1/service/alias

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

{
    "service":"...",
    "namespace":"...",
    "alias":"...",
    "alias_namespace":"...",
    "comment":"..."
}
~~~

应答示例: 

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

数据结构: 

| 参数名          | 类型   | 描述                   | 必填 |
| --------------- | ------ | ---------------------- | ---- |
| alias           | string | 服务别名               | 是   |
| alias_namespace | string | 服务别名所属命名空间   | 是   |
| service         | string | 指向的服务名           | 是   |
| namespace       | string | 指向的服务所属命名空间 | 是   |
| comment         | string | 服务别名描述           | 否   |
`
	enrichUpdateServiceAliasApiNotes = `
请求示例: 

~~~
PUT /naming/v1/service/alias

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

{
    "service":"...",
    "namespace":"...",
    "alias":"...",
    "alias_namespace":"...",
    "comment":"..."
}
~~~

应答示例: 

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

数据结构: 

| 参数名          | 类型   | 描述                   | 必填 |
| --------------- | ------ | ---------------------- | ---- |
| alias           | string | 服务别名               | 是   |
| alias_namespace | string | 服务别名所属命名空间   | 是   |
| service         | string | 指向的服务名           | 是   |
| namespace       | string | 指向的服务所属命名空间 | 是   |
| comment         | string | 服务别名描述           | 否   |
`
	enrichGetServiceAliasesApiNotes = `
请求示例: 

~~~
GET /naming/v1/service/aliases

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

~~~

应答示例: 

| 参数名 | 类型   | 描述                                                                                             |
| ------ | ------ | ------------------------------------------------------------------------------------------------ |
| size   | uint32 | 本次查询返回的服务别名个数, 例如查询命名空间为Production的服务别名, 总数为1000, 本次返回100条, 则size为100      |
| info   | string | 返回信息                                                                                           |
| code   | uint32 | 六位返回码                                                                                         |
| amount | uint32 | 符合此查询条件的服务别名总数, 例如查询命名空间为Production的服务别名, 总数为1000, 本次返回100条, 则amount为1000 |

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

数据结构: 

| 参数名          | 类型   | 描述                         | 必填 |
| --------------- | ------ | ---------------------------- | ---- |
| alias           | string | 服务别名                     | 否   |
| alias_namespace | string | 服务别名所属命名空间         | 否   |
| service         | string | 指向的服务名                 | 否   |
| namespace       | string | 指向的服务所属命名空间       | 否   |
| offset          | int    | 分页偏移, 默认0              | 否   |
| limit           | int    | 分页大小, 默认为100, 最大100 | 否   |
`
	enrichDeleteServiceAliasesApiNotes = `
请求示例: 

~~~
POST /naming/v1/service/aliases/delete

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

[
    {
        "alias":"...",
        "alias_namespace":"..."
    }
]
~~~

应答示例: 

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

数据结构: 

| 参数名          | 类型   | 描述                  | 必填 |
| --------------- | ------ | --------------------- | ---- |
| alias           | string | 服务别名              | 是   |
| alias_namespace | string | 服务别名所属命名空间  | 是   |
`
	enrichGetCircuitBreakerByServiceApiNotes = `
请求示例: 

~~~
GET /naming/v1/service/circuitbreaker?service=xxx&namespace=xxx

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}
~~~

应答示例: 

~~~json
{
    "code":200000,
    "info":"execute success",
    "size":1,
    "configWithServices":[
        {
            "services":[

            ],
            "circuitBreaker": {
                "id": "xxx",
                "version": "xxx",
                "name": "xxx",
                "namespace": "xxx",
                "service": "xxx",
                "service_namespace": "xxx",
                "inbounds": [
                    {
                        "sources":[
                            {
                                "service":"*",
                                "namespace":"*"
                            }
                        ],
                        "destinations":[
                            {
                                "policy":{
                                    "errorRate":{
                                        "enable":true,
                                        "errorRateToOpen":10,
                                        "requestVolumeThreshold":10
                                    }
                                },
                                "recover":{
                                    "sleepWindow":"1s",
                                    "outlierDetectWhen":"NEVER"
                                },
                                "resource":"INSTANCE",
                                "method":{
                                    "value":"qweqwe"
                                }
                            }
                        ]
                    }
                ],
                "outbounds":  [
                    {
                        "sources":[
                            {
                                "service":"*",
                                "namespace":"*"
                            }
                        ],
                        "destinations":[
                            {
                                "policy":{
                                    "errorRate":{
                                        "enable":true,
                                        "errorRateToOpen":10,
                                        "requestVolumeThreshold":10
                                    }
                                },
                                "recover":{
                                    "sleepWindow":"1s",
                                    "outlierDetectWhen":"NEVER"
                                },
                                "resource":"INSTANCE",
                                "method":{
                                    "value":"qweqwe"
                                }
                            }
                        ]
                    }
                ]
            }
        }
    ]
}
~~~
`
	enrichGetServiceOwnerApiNotes = ``
	enrichCreateInstancesApiNotes = `
请求示例: 

~~~
POST /naming/v1/instances

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

[
    {
    "service": "tdsql-ops-server",
    "namespace": "default",
    "host": "127.0.0.1",
    "port": 8080,
    "location": {
        "region": "ap-guangzhou",
        "zone": "ap-guangzhou-3",
        "campus": ""
    },
    "enable_health_check": true,
    "health_check": {
        "type": 1,
        "heartbeat": {
            "ttl": 10
        }
    },
    "metadata": {
        "env": "pre"
    }
}
]
~~~

应答示例: 

~~~json
{
    "code": 200000,
    "info": "execute success",
    "size": 1,
    "responses": [
        {
            "code": 200000,
            "info": "execute success",
            "instance": {
                "id": "...",
                "service": "...",
                "namespace": "...",
                "host": "...",
                "port": 8080
            }
        }
    ]
}
~~~

数据结构: 

> HealthCheck 参数

| 参数名    | 类型         | 描述                        | 是否必填 |
| --------- | ------------ | --------------------------- | -------- |
| type      | int          | 0(Unknow)/1(Heartbeat)      | 是       |
| heartbeat | {"ttl": int} | 心跳间隔(范围为区间(0, 60]) | 是       |

> Location 参数

| 参数名 | 类型   | 描述 | 是否必填 |
| ------ | ------ | ---- | -------- |
| region | string | 地区 | 否       |
| zone   | string | 地域 | 否       |
| campus | string | 园区 | 否       |

> 主请求参数

| 参数名              | 类型               | 描述                                       | 是否必填 |
| ------------------- | ------------------ | ------------------------------------------ | -------- |
| service             | string             | 服务名                                     | 是       |
| namespace           | string             | 命名空间                                   | 是       |
| host                | string             | 实例的IP                                   | 是       |
| port                | string             | 实例的端口                                 | 是       |
| vpc_id              | string             | VPC ID                                     | 否       |
| protocol            | string             | 对应端口的协议                             | 否       |
| version             | string             | 版本                                       | 否       |
| priority            | string             | 优先级                                     | 否       |
| weight              | string             | 权重(默认值100)                            | 是       |
| enable_health_check | bool               | 是否开启健康检查                                    | 是       |
| health_check        | HealthCheck        | 健康检查类别具体描述信息(如果enable_health_check==true, 必须填写) | 否       |
| healthy             | bool               | 实例健康标志(默认为健康的)                              | 是       |
| isolate             | bool               | 实例隔离标志(默认为不隔离的)                            | 是       |
| location            | Location           | 实例位置信息                                            | 是       |
| metadata            | map<string,string> | 实例标签信息, 最多只能存储64对 *key-value*               | 否       |
| service_token       | string             | service的token信息                                      | 是       |
`
	enrichDeleteInstancesApiNotes = `
请求示例: 

~~~
POST /naming/v1/instances/delete

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

[
    {
        "id": "...",
    }
]
~~~

应答示例: 

~~~json
{
    "code": 200000,
    "info": "execute success",
    "amount": 0,
    "size": 0
}
~~~

数据结构: 

| 参数名    | 类型   | 描述     | 是否必填 |
| --------- | ------ | -------- | -------- |
| id        | string | 实例ID   | 是       |
| service   | string | 服务名称 | 是       |
| namespace | string | 命名空间 | 是       |
`
	enrichDeleteInstancesByHostApiNotes = `
请求示例: 

~~~
POST /naming/v1/instances/delete

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

[
    {
        "host": "...",
    }
]
~~~

应答示例: 

~~~json
{
    "code": 200000,
    "info": "execute success",
    "amount": 0,
    "size": 0
}
~~~

数据结构: 

| 参数名    | 类型   | 描述     | 是否必填 |
| --------- | ------ | -------- | -------- |
| id        | string | 实例ID   | 是       |
| service   | string | 服务名称 | 是       |
| namespace | string | 命名空间 | 是       |
`
	enrichUpdateInstancesApiNotes = `
请求示例: 

~~~
PUT /naming/v1/instances

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

[
    {
    "service": "tdsql-ops-server",
    "namespace": "default",
    "host": "127.0.0.1",
    "port": 8080,
    "location": {
        "region": "ap-guangzhou",
        "zone": "ap-guangzhou-3",
        "campus": ""
    },
    "enable_health_check": true,
    "health_check": {
        "type": 1,
        "heartbeat": {
            "ttl": 10
        }
    },
    "metadata": {
        "env": "pre"
    }
}
]
~~~

应答示例: 

~~~json
{
    "code": 200000,
    "info": "execute success",
    "amount": 0,
    "size": 0
}
~~~

数据结构: 

> HealthCheck 参数

| 参数名    | 类型         | 描述                        | 是否必填 |
| --------- | ------------ | --------------------------- | -------- |
| type      | int          | 0(Unknow)/1(Heartbeat)      | 是       |
| heartbeat | {"ttl": int} | 心跳间隔(范围为区间(0, 60]) | 是       |

> Location 参数

| 参数名 | 类型   | 描述 | 是否必填 |
| ------ | ------ | ---- | -------- |
| region | string | 地区 | 否       |
| zone   | string | 地域 | 否       |
| campus | string | 园区 | 否       |

> 主请求参数

| 参数名              | 类型               | 描述                                        | 是否必填 |
| ------------------- | ------------------ | ------------------------------------------- | -------- |
| service             | string             | 服务名                                      | 是       |
| namespace           | string             | 命名空间                                    | 是       |
| host                | string             | 实例的IP                                    | 是       |
| port                | string             | 实例的端口                                  | 是       |
| vpc_id              | string             | VPC ID                                      | 否       |
| protocol            | string             | 对应端口的协议                              | 否       |
| version             | string             | 版本                                        | 否       |
| priority            | string             | 优先级                                      | 否       |
| weight              | string             | 权重(默认值100)                             | 是       |
| enable_health_check | bool               | 是否开启健康检查                            | 是       |
| health_check        | HealthCheck        | 健康检查类别具体描述信息(如果enable_health_check==true, 必须填写) | 否       |
| healthy             | bool               | 实例健康标志(默认为健康的)                               | 是       |
| isolate             | bool               | 实例隔离标志(默认为不隔离的)                             | 是       |
| location            | Location           | 实例位置信息                                             | 是       |
| metadata            | map<string,string> | 实例标签信息, 最多只能存储64对 *key-value*                | 否       |
| service_token       | string             | service的token信息                                       | 是       |
`
	enrichUpdateInstancesIsolateApiNotes = `
请求示例: 

~~~
PUT /instances/isolate/host

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

[
    {
    "service": "tdsql-ops-server",
    "namespace": "default",
    "host": "127.0.0.1",
    "port": 8080,
    "location": {
        "region": "ap-guangzhou",
        "zone": "ap-guangzhou-3",
        "campus": ""
    },
    "enable_health_check": true,
    "health_check": {
        "type": 1,
        "heartbeat": {
            "ttl": 10
        }
    },
    "metadata": {
        "env": "pre"
    }
}
]
~~~

应答示例: 

~~~json
{
    "code": 200000,
    "info": "execute success",
    "amount": 0,
    "size": 0
}
~~~

数据结构: 

> HealthCheck 参数

| 参数名    | 类型         | 描述                        | 是否必填 |
| --------- | ------------ | --------------------------- | -------- |
| type      | int          | 0(Unknow)/1(Heartbeat)      | 是       |
| heartbeat | {"ttl": int} | 心跳间隔(范围为区间(0, 60]) | 是       |

> Location 参数

| 参数名 | 类型   | 描述 | 是否必填 |
| ------ | ------ | ---- | -------- |
| region | string | 地区 | 否       |
| zone   | string | 地域 | 否       |
| campus | string | 园区 | 否       |

> 主请求参数

| 参数名              | 类型               | 描述                                                | 是否必填 |
| ------------------- | ------------------ | --------------------------------------------------- | -------- |
| service             | string             | 服务名                                              | 是       |
| namespace           | string             | 命名空间                                            | 是       |
| host                | string             | 实例的IP                                            | 是       |
| port                | string             | 实例的端口                                          | 是       |
| vpc_id              | string             | VPC ID                                              | 否       |
| protocol            | string             | 对应端口的协议                                      | 否       |
| version             | string             | 版本                                                | 否       |
| priority            | string             | 优先级                                              | 否       |
| weight              | string             | 权重(默认值100)                                     | 是       |
| enable_health_check | bool               | 是否开启健康检查                                    | 是       |
| health_check        | HealthCheck        | 健康检查类别具体描述信息(如果enable_health_check==true, 必须填写) | 否       |
| healthy             | bool               | 实例健康标志(默认为健康的)                           | 是       |
| isolate             | bool               | 实例隔离标志(默认为不隔离的)                         | 是       |
| location            | Location           | 实例位置信息                                         | 是       |
| metadata            | map<string,string> | 实例标签信息, 最多只能存储64对 *key-value*            | 否       |
| service_token       | string             | service的token信息                                   | 是       |
`
	enrichGetInstancesApiNotes = `
请求示例

~~~
GET /naming/v1/instances?service=&namespace=&{参数key}={参数值}

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

~~~

| 参数名      | 类型   | 描述             | 是否必填                                                             |
| ----------- | ------ | ---------------- | -------------------------------------------------------------------- |
| service     | string | 服务名称         | 是                                                                   |
| namespace   | string | 命名空间         | 是                                                                   |
| host        | string | 实例IP           | 是(要么(service, namespace)存在, 要么host存在, 不然视为参数不完整) |
| port        | uint   | 实例端口         | 否                                                                   |
| keys        | string | 标签key          | 只允许填写一个key                                                    |
| values      | string | 标签value        | 只允许填写一个value                                                  |
| healthy     | string | 实例健康状态     | 否                                                                   |
| isolate     | string | 实例隔离状态     | 否                                                                   |
| protocol    | string | 实例端口协议状态 | 否                                                                   |
| version     | string | 实例版本         | 否                                                                   |
| cmdb_region | string | 实例region信息   | 否                                                                   |
| cmdb_zone   | string | 实例zone信息     | 否                                                                   |
| cmdb_idc    | string | 实例idc信息      | 否                                                                   |
| offset      | uint   | 查询偏移量       | 否                                                                   |
| limit       | uint   | 查询条数         | 否                                                                   |

应答示例: 
~~~json
{
    "code": 200000,
    "info": "execute success",
    "amount": 1,
    "size": 1,
    "instances": [
        {
            "id": "...",
            "host": "...",
            "port": 8080,
            "weight": 100,
            "enableHealthCheck": true,
            "healthCheck": {
                "type": "HEARTBEAT",
                "heartbeat": {
                    "ttl": 10
                }
            },
            "healthy": true,
            "isolate": false,
            "location": {
                "region": "ap-guangzhou",
                "zone": "ap-guangzhou-3",
                "campus": ""
            },
            "metadata": {
                "env": "pre"
            },
            "ctime": "2021-11-23 01:59:31",
            "mtime": "2021-11-23 01:59:31",
            "revision": "..."
        }
    ]
}
~~~
`
	enrichGetInstancesCountApiNotes = `
请求示例: 
~~~
GET /naming/v1/instances/count

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

~~~

返回示例: 
~~~json
{
	"code": 200000,
	"info": "execute success",
	"amount": 97,
	"size": 0,
	"namespaces": [],
	"services": [],
	"instances": [],
	"routings": [],
	"aliases": [],
	"rateLimits": [],
	"configWithServices": [],
	"platforms": [],
	"users": [],
	"userGroups": [],
	"authStrategies": [],
	"clients": []
}
~~~
`
	enrichGetInstanceLabelsApiNotes = `
请求示例: 
~~~
GET /naming/v1/instances/labels?service=&namespace=&{参数key}={参数值}

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

~~~

返回示例: 
~~~json
{
  "code": 200000,
  "info": "execute success",
  "client": null,
  "namespace": null,
  "service": null,
  "instance": null,
  "routing": null,
  "alias": null,
  "rateLimit": null,
  "circuitBreaker": null,
  "configRelease": null,
  "platform": null,
  "user": null,
  "userGroup": null,
  "authStrategy": null,
  "relation": null,
  "loginResponse": null,
  "modifyAuthStrategy": null,
  "modifyUserGroup": null,
  "resources": null,
  "optionSwitch": null,
  "instanceLabels": {
    "labels": {
      "campus": {
        "values": [
          ""
        ]
      },
      "region": {
        "values": [
          ""
        ]
      },
      "zone": {
        "values": [
          ""
        ]
      }
    }
  }
}
~~~
`
	enrichCreateRoutingsApiNotes = `
为服务创建一个路由规则, 以对服务进行流量调度, 一个服务只能有一个路由规则。

请求示例: 

~~~
POST /naming/v1/routings

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

[
    {
        "service":"...",
        "namespace":"...",
        "inbounds":[
           {
              "sources": [
                 {
                   "service": "...",
                   "namespace": "...",
                   "metadata": {
                       "...": {
                         "type": "EXACT",
                         "value": "..."
                       }
                    }
                 }
              ],
              "destinations": [
                  {
                    "metadata": {
                       "...": {
                         "type": "EXACT",
                         "value": "..."
                       }
                    }
                    "weight": ...
                  }
              ]
           }
        ],
        "outbounds":[
            {
              "sources": [
                 {
                   "metadata": {
                       "...": {
                         "type": "EXACT",
                         "value": "..."
                       }
                    }
                 }
              ],
              "destinations": [
                  {
                   "service": "...",
                   "namespace": "...",                  
                    "metadata": {
                       "...": {
                         "type": "EXACT",
                         "value": "..."
                       }
                    }
                    "weight": ...
                  }
              ]
           }
        ],
        "service_token":"...",
    }
]
~~~

回复示例: 

~~~
{
    "code":200000,
    "info":"...",
    "size":1,
    "responses":[
        {
            "code":200000,
            "info":"...",
            "routing":{
                "service":"...",
                "namespace":"..."
            }
        }
    ]
}
~~~

数据结构: 

> routing结构参数

| 参数名    | 类型   | 描述               | 是否必填 |
| --------- | ------ | ------------------ | -------- |
| service   | string | 规则所属的服务名   | 是       |
| namespace | string | 规则所属的命名空间 | 是       |
| inbounds  | route[]    | 入流量规则 | 否 |
| outbounds | route[] | 出流量规则 | 否 |
| service_token | string | 鉴权token, 当开启鉴权后需要传入 | 否 |

> route结构参数

| 参数名       | 类型    | 描述 | 是否必填 |
| ------------ | ------- | ---- | -------- |
| sources      | source[] | 请求匹配条件 | 否 |
| destinations | destination[] | 目标实例分组匹配条件 | 是 |

> source结构参数

| 参数名    | 类型                     | 描述                           | 是否必填 |
| --------- | ------------------------ | ------------------------------ | -------- |
| service   | string                   | 主调方服务名, 填*代表全匹配    | 否       |
| namespace | string                   | 被调方服务名, 填*代表全匹配    | 否       |
| metadata  | map<string, matchString> | 匹配参数, 需全匹配所有KV才通过 | 否       |

> destination结构参数

| 参数名    | 类型                     | 描述                                                         | 是否必填 |
| --------- | ------------------------ | ------------------------------------------------------------ | -------- |
| service   | string                   | 被调方服务名, 填*代表全匹配                                  | 否       |
| namespace | string                   | 被调方命名空间, 填*代表全匹配                                | 否       |
| metadata  | map<string, matchString> | 示例标签匹配参数, 需全匹配才通过                             | 否       |
| priority  | int32                    | 优先级, 数值越小, 优先级越高, 请求会优先选取优先级最高的实例分组进行路由, 只有该分组没有可用实例才会选择次高优先级的分组 | 否       |
| weight    | int32                    | 分组权重, 优先级相同的多个分组, 按权重比例进行请求分配       | 否       |

> matchString结构参数

| 参数名     | 类型   | 描述                                                         | 是否必填 |
| ---------- | ------ | ------------------------------------------------------------ | -------- |
| type       | string | 匹配类型, 枚举值, 支持: EXACT(全匹配, 默认), REGEX(正则表达式匹配) | 否       |
| value      | string | 匹配的目标值                                                 | 是       |
| value_type | string | 值类型, 枚举值, 支持: TEXT(文本, 默认), PARAMETER(参数, 路由规则值使用动态参数时用到) | 否       |
`
	enrichDeleteRoutingsApiNotes = `
删除服务下的路由规则

请求示例: 

~~~
POST /naming/v1/routings/delete

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

[
    {
        "service_token":"...",
        "service":"...",
        "namespace":"..."
    }
]
~~~

回复示例: 

~~~
{
    "code":200000,
    "info":"...",
    "size":1,
    "responses":[
        {
            "code":200000,
            "info":"...",
            "routing":{
                "service":"...",
                "namespace":"..."
            }
        }
    ]
}
~~~
`
	enrichUpdateRoutingsApiNotes = `
更新服务下的路由规则的相关信息

请求示例: 

~~~
PUT /naming/v1/routings

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

[
    {
        "service":"...",
        "namespace":"...",
        "inbounds":[...],
        "outbounds":[...],
        "service_token":"...",
    }
]
~~~

回复示例: 

~~~
{
    "code":200000,
    "info":"...",
    "size":1,
    "responses":[
        {
            "code":200000,
            "info":"...",
            "routing":{
                "service":"...",
                "namespace":"..."
            }
        }
    ]
}
~~~

数据结构: 

> routing结构参数

| 参数名    | 类型   | 描述               | 是否必填 |  |
| --------- | ------ | ------------------ | -------- | -------- |
| service   | string | 规则所属的服务名   | 是       | 否      |
| namespace | string | 规则所属的命名空间 | 是       | 否      |
| inbounds  | route[]    | 入流量规则 | 否 | 是 |
| outbounds | route[] | 出流量规则 | 否 | 是 |
| service_token | string | 鉴权token, 当开启鉴权后需要传入 | 否 | 否 |
`
	enrichGetRoutingsApiNotes = `
请求示例: 

~~~
GET /naming/v1/routings?参数名=参数值

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}
~~~

| 参数名    | 类型   | 描述                    | 是否必填 |
| --------- | ------ | ----------------------- | -------- |
| service   | string | 服务名                  | 否       |
| namespace | string | 命名空间                | 否       |
| offset    | int    | 分页的起始位置, 默认为0 | 否       |
| limit     | int    | 每页行数, 默认100       | 否       |

应答示例: 

~~~
{
  	"code": ...,
  	"info": "...",
  	"amount": ...,
  	"size": ...,
  	"routings": [
    	{
          "service": "...",
          "namespace": "...",
          "inbounds": [...],
          "outbounds": [...],
          "ctime": "...",  // 创建时间
          "mtime": "..."   // 修改时间
    	}
  ]
}
~~~
`
	enrichCreateRateLimitsApiNotes = `
为服务创建多个限流规则, 以对服务进行流量限制, 按优先级顺序进行匹配, 匹配到一个则执行该规则。

请求示例: 
~~~
POST /naming/v1/ratelimits

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

[
	{
		"name": "rule1",
		"service": "testsvc1",
		"namespace": "default",
		"method": {
			"type": "EXACT",
			"value": "/getsomething"
		},
		"arguments": [
			{
				"type": "HEADER",
				"key": "host",
				"value": {
					"type": "EXACT",
					"value": "www.baidu.com"
				}
			},
			{
				"type": "CALLER_SERVICE",
				"key": "default",
				"value": {
					"type": "IN",
					"value": "testsvc1,testsvc2"
				}
			}
		],
		"resource": "QPS",
		"type": "LOCAL",
		"amounts": [
			{
				"maxAmount": 1000,
				"validDuration": "1s"
			},
			{
				"maxAmount": 2000,
				"validDuration": "1m"
			}	
		],
		"regex_combine": false,
		"disable": false,
		"failover": "FAILOVER_LOCAL"
	}
]
~~~
回复示例: 
~~~
{
 "code": 200000,
 "info": "execute success",
 "size": 1,
 "responses": [
  {
   "code": 200000,
   "info": "execute success",
   "rateLimit": {
    "id": "e04f201e7b7e4599b42a9b6631a7ba08", //规则ID
    "service": "testsvc1",
    "namespace": "default",
    "name": "rule2"
   }
  }
 ]
}
~~~

数据结构: 

> Ratelimit结构参数

| 参数名          | 类型          | 描述                                                         | 是否必填 |
| --------------- | ------------- | ------------------------------------------------------------ | -------- |
| name            | string        | 规则名                                                       | 是       |
| service         | string        | 规则所属的服务名, 创建规则时, 如果服务不存在, 则会自动创建服务。 | 是       |
| namespace       | string        | 规则所属的命名空间                                           | 是       |
| method          | MatchString   | 规则所针对的服务接口                                         | 否       |
| arguments       | MatchArgument | 请求参数匹配条件, 需全匹配才通过                             | 否       |
| resource        | string        | 限流资源, 默认为QPS(针对QPS进行限流)                         | 否       |
| type            | string        | 限流类型, 支持LOCAL(单机限流), GLOBAL(分布式限流)        | 是       |
| amounts         | Amount[]      | 限流配额, 包含限流周期和配额总数, 可配置多个                 | 是       |
| regex_combine   | bool          | 合并计算配额, 对于匹配到同一条正则表达式规则的多个不同的请求进行合并计算, 默认为false | 否|
| disable         | bool          | 是否启用该限流规则, 默认为false(启用)                      | 否       |
| action          | string        | 限流效果, 支持REJECT(直接拒绝),UNIRATE(匀速排队), 默认REJECT | 否       |
| failover | string | 失败降级措施, 仅分布式限流有效, 支持FAILOVER_LOCAL(降级到单机限流), FAILOVER_PASS(直接通过) | 否 |
| max_queue_delay | int           | 最大排队时长, 单位秒, 仅对匀速排队生效。默认1秒              | 否       |

> Amount结构参数

| 参数名        | 类型   | 描述                                                 | 是否必填 |
| ------------- | ------ | ---------------------------------------------------- | -------- |
| maxAmount     | uint32 | 周期内最大配额数                                     | 是       |
| validDuration | string | 周期描述, 支持duration类型的字符串, 比如1s, 1m, 1h等 | 是       |

> MatchString结构参数

| 参数名 | 类型   | 描述                                                         | 是否必填 |
| ------ | ------ | ------------------------------------------------------------ | -------- |
| type   | string | 匹配类型, 支持: EXACT(全匹配), REGEX(正则表达式匹配), NOT_EQUALS(不等于), IN(包含), NOT_IN(不包含) |是|
| value  | string | 匹配的目标值, 如果选择的是包含和不包含, 则通过逗号分割多个值 | 是       |

> MatchArgument结构参数

| 参数名 | 类型        | 描述                                                         | 是否必填 |
| ------ | ----------- | ------------------------------------------------------------ | -------- |
| type| string|CUSTOM(自定义),METHOD(方法),HEADER(请求头),QUERY(请求参数),CALLER_SERVICE(主调方服务),CALLER_IP(主调方IP)|是 |
| key | string| 参数键, 对于HEADER、QUERY、CUSTOM, 对应的是key值; 对于CALLER_SERVICE, 对应的是服务的命名空间值 | 是       |
| value  | MatchString | 参数值, 支持多种匹配模式(见MatchString的定义) | 是       |
`
	enrichDeleteRateLimitsApiNotes = `
请求示例: 

~~~
POST /naming/v1/ratelimits/delete

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

[
	{
		"id": "6942526fbac545848cd8fb32a3a55bb6" //规则ID, 必填
	}
]
~~~

应答示例: 

~~~
{
 "code": 200000,
 "info": "execute success",
 "size": 1,
 "responses": [
  {
   "code": 200000,
   "info": "execute success",
   "rateLimit": {
    "id": "6942526fbac545848cd8fb32a3a55bb6"
   }
  }
 ]
}
~~~
`
	enrichUpdateRateLimitsApiNotes = `
更新服务下的限流规则的相关信息

请求示例: 

~~~
PUT /naming/v1/ratelimits

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

[
	{
	    "id":   "e04f201e7b7e4599b42a9b6631a7ba08",
		"name": "rule1",
		"service": "testsvc1",
		"namespace": "default",
		"method": {
			"type": "EXACT",
			"value": "/getsomething"
		},
		"arguments": [
			{
				"type": "HEADER",
				"key": "host",
				"value": {
					"type": "EXACT",
					"value": "www.baidu.com"
				}
			},
			{
				"type": "CALLER_SERVICE",
				"key": "default",
				"value": {
					"type": "IN",
					"value": "testsvc1,testsvc2"
				}
			}
		],
		"resource": "QPS",
		"type": "LOCAL",
		"amounts": [
			{
				"maxAmount": 1000,
				"validDuration": "1s"
			},
			{
				"maxAmount": 2000,
				"validDuration": "1m"
			}	
		],
		"regex_combine": false,
		"disable": true,
		"failover": "FAILOVER_LOCAL"
	}
]
~~~

应答示例: 

~~~
{
 "code": 200000,
 "info": "execute success",
 "size": 1,
 "responses": [
  {
   "code": 200000,
   "info": "execute success",
   "rateLimit": {
    "id": "e04f201e7b7e4599b42a9b6631a7ba08", //规则ID
    "service": "testsvc1",
    "namespace": "default",
    "name": "rule2"
   }
  }
 ]
}
~~~
`
	enrichGetRateLimitsApiNotes = `
请求示例: 

~~~
GET /naming/v1/ratelimits?参数名=参数值

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}
~~~

| 参数名    | 类型   | 描述                                                         | 是否必填 |
| --------- | ------ | ------------------------------------------------------------ | -------- |
| id        | string | 规则ID                                                       | 否       |
| name      | string | 规则名                                                       | 否       |
| service   | string | 服务名                                                       | 否       |
| namespace | string | 命名空间                                                     | 否       |
| method    | string | 限流接口名, 默认为模糊匹配                                   | 否       |
| disable   | bool   | 规则是否启用, true为不启用, false为启用                      | 否       |
| brief     | bool   | 是否只显示概要信息, brief=true时, 则不返回规则详情, 只返回规则列表概要信息, 默认为false | 否       |
| offset    | int    | 分页的起始位置, 默认为0                                      | 否       |
| limit     | int    | 每页行数, 默认100                                            | 否       |

应答示例: 

~~~
{
 "code": 200000,
 "info": "execute success",
 "amount": 2,
 "size": 2,
 "rateLimits": [
  {
   "id": "e04f201e7b7e4599b42a9b6631a7ba08",
   "service": "testsvc1",
   "namespace": "default",
   "priority": 0,
   "disable": false,
   "ctime": "2022-07-26 21:03:50",
   "mtime": "2022-07-26 21:03:50",
   "revision": "",
   "method": {
    "value": "/getsomething2"
   },
   "name": "rule2",
   "etime": "2022-07-26 21:03:50"
  },
  {
   "id": "6942526fbac545848cd8fb32a3a55bb6",
   "service": "testsvc1",
   "namespace": "default",
   "priority": 0,
   "disable": false,
   "ctime": "2022-07-26 10:09:49",
   "mtime": "2022-07-26 11:46:07",
   "revision": "",
   "method": {
    "value": "/getsomething"
   },
   "name": "rule1",
   "etime": "2022-07-26 11:46:07"
  }
 ]
}
~~~
`
	enrichEnableRateLimitsApiNotes = `
请求示例: 

~~~
PUT /naming/v1/ratelimits/enable
[
	{
		"id": "6942526fbac545848cd8fb32a3a55bb6", //规则ID, 必填
		"disable": true // 是否禁用, true为不启用, false为启用
	}
]
~~~

应答示例: 

~~~
{
 "code": 200000,
 "info": "execute success",
 "size": 1,
 "responses": [
  {
   "code": 200000,
   "info": "execute success",
   "rateLimit": {
    "id": "e04f201e7b7e4599b42a9b6631a7ba08",
    "disable": false
   }
  }
 ]
}
~~~
`
	enrichCreateCircuitBreakersApiNotes = `

- 为服务创建一个熔断规则, 以对服务下的故障节点进行剔除。
- 熔断规则可以分为被调规则和主调规则: 
	- 被调规则针对所有的指定主调生效, 假如不指定则对所有的主调生效。
	- 主调规则为当前主调方的规则, 假如不指定则针对所有被调生效。
	- 被调规则与主调规则同时存在时, 被调优先, 被调规则生效。


请求示例: 

~~~
POST /naming/v1/circuitbreakers

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

[
    {
        "service":"qweqweqwqe",
        "namespace":"default",
        "inbounds":[
            {
                "sources":[
                    {
                        "service":"*",
                        "namespace":"*"
                    }
                ],
                "destinations":[
                    {
                        "policy":{
                            "errorRate":{
                                "enable":true,
                                "errorRateToOpen":10,
                                "requestVolumeThreshold":10
                            }
                        },
                        "recover":{
                            "sleepWindow":"1s",
                            "outlierDetectWhen":"NEVER"
                        },
                        "resource":"INSTANCE",
                        "method":{
                            "value":"qweqwe"
                        }
                    }
                ]
            }
        ],
        "outbounds":[

        ],
        "id":"xxx",
        "version":"1647356947061",
        "name":"xxx"
    }
]
~~~

应答示例: 

~~~json
{
    "code":200000,
    "info":"execute success",
    "size":1,
    "responses":[
        {
            "code":200000,
            "info":"execute success"
        }
    ]
}
~~~
`
	enrichCreateCircuitBreakerVersionsApiNotes = `
~~~
POST /naming/v1/circuitbreakers/version

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

[
    {
        "id": "xxx",
        "version": "xxx",
        "inbounds":[
            {
                "sources":[
                    {
                        "service":"*",
                        "namespace":"*"
                    }
                ],
                "destinations":[
                    {
                        "policy":{
                            "errorRate":{
                                "enable":true,
                                "errorRateToOpen":10,
                                "requestVolumeThreshold":10
                            }
                        },
                        "recover":{
                            "sleepWindow":"1s",
                            "outlierDetectWhen":"NEVER"
                        },
                        "resource":"INSTANCE",
                        "method":{
                            "value":"qweqwe"
                        }
                    }
                ]
            }
        ],
        "outbounds":[

        ]
    }
]
~~~

应答示例: 

~~~json
{
    "code":200000,
    "info":"execute success"
}
~~~
`
	enrichDeleteCircuitBreakersApiNotes = `
~~~
PUT /naming/v1/circuitbreakers/delete

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

[
	{
		"id": "6942526fbac545848cd8fb32a3a55bb6" //熔断规则id
	}
]
~~~
`
	enrichUpdateCircuitBreakersApiNotes = `
请求示例: 

~~~
POST /naming/v1/circuitbreakers

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

[
    {
        "service":"qweqweqwqe",
        "namespace":"default",
        "inbounds":[
            {
                "sources":[
                    {
                        "service":"*",
                        "namespace":"*"
                    }
                ],
                "destinations":[
                    {
                        "policy":{
                            "errorRate":{
                                "enable":true,
                                "errorRateToOpen":10,
                                "requestVolumeThreshold":10
                            }
                        },
                        "recover":{
                            "sleepWindow":"1s",
                            "outlierDetectWhen":"NEVER"
                        },
                        "resource":"INSTANCE",
                        "method":{
                            "value":"qweqwe"
                        }
                    }
                ]
            }
        ],
        "outbounds":[

        ],
        "id":"xxx",
        "version":"1647356947061",
        "name":"xxx"
    }
]
~~~
`
	enrichReleaseCircuitBreakersApiNotes = `
~~~
POST /naming/v1/circuitbreakers/release

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

~~~
`
	enrichUnBindCircuitBreakersApiNotes = `
~~~
POST /naming/v1/circuitbreakers/unbind

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

~~~
`
	enrichGetCircuitBreakersApiNotes = `
请求示例: 
~~~
GET /naming/v1/circuitbreakers?id={参数值}&version={参数值}

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

~~~
`
	enrichGetCircuitBreakerVersionsApiNotes = `
请求示例: 
~~~
GET /naming/v1/circuitbreaker/versions?id={参数值}

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

~~~
`
	enrichGetMasterCircuitBreakersApiNotes = `
请求示例: 
~~~
GET /naming/v1/circuitbreakers/master?id={参数值}

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

~~~
`
	enrichGetReleaseCircuitBreakersApiNotes = `
请求示例: 
~~~
GET /naming/v1/circuitbreakers/release?id={参数值}

# 开启北极星服务端针对控制台接口鉴权开关后, 需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

~~~
`
)
