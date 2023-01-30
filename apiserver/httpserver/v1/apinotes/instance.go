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

package apinotes

const (
	EnrichCreateInstancesApiNotes = `
请求示例：

~~~
POST /naming/v1/instances

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
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

应答示例：

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

数据结构：

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

| 参数名               | 类型               | 描述                                                       | 是否必填  |
| ------------------- | ------------------ | -------------------------------------------------------- | -------- |
| service             | string             | 服务名                                                    | 是       |
| namespace           | string             | 命名空间                                                   | 是       |
| host                | string             | 实例的IP                                                   | 是       |
| port                | string             | 实例的端口                                                  | 是       |
| vpc_id              | string             | VPC ID                                                    | 否       |
| protocol            | string             | 对应端口的协议                                               | 否       |
| version             | string             | 版本                                                       | 否       |
| priority            | string             | 优先级                                                      | 否       |
| weight              | string             | 权重(默认值100)                                              | 是       |
| enable_health_check | bool               | 是否开启健康检查                                              | 是       |
| health_check        | HealthCheck        | 健康检查类别具体描述信息(如果enable_health_check==true，必须填写) | 否       |
| healthy             | bool               | 实例健康标志(默认为健康的)                                     | 是       |
| isolate             | bool               | 实例隔离标志(默认为不隔离的)                                   | 是       |
| location            | Location           | 实例位置信息                                                | 是       |
| metadata            | map<string,string> | 实例标签信息，最多只能存储64对 *key-value*                     | 否       |
| service_token       | string             | service的token信息                                         | 是       |
`
	EnrichDeleteInstancesApiNotes = `
请求示例：

~~~
POST /naming/v1/instances/delete

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

[
    {
        "id": "...",
    }
]
~~~

应答示例：

~~~json
{
    "code": 200000,
    "info": "execute success",
    "amount": 0,
    "size": 0
}
~~~

数据结构：

| 参数名    | 类型   | 描述     | 是否必填 |
| --------- | ------ | -------- | -------- |
| id        | string | 实例ID   | 是       |
| service   | string | 服务名称 | 是       |
| namespace | string | 命名空间 | 是       |
`
	EnrichDeleteInstancesByHostApiNotes = `
请求示例：

~~~
POST /naming/v1/instances/delete

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

[
    {
        "host": "...",
    }
]
~~~

应答示例：

~~~json
{
    "code": 200000,
    "info": "execute success",
    "amount": 0,
    "size": 0
}
~~~

数据结构：

| 参数名    | 类型   | 描述     | 是否必填 |
| --------- | ------ | -------- | -------- |
| id        | string | 实例ID   | 是       |
| service   | string | 服务名称 | 是       |
| namespace | string | 命名空间 | 是       |
`
	EnrichUpdateInstancesApiNotes = `
请求示例：

~~~
PUT /naming/v1/instances

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
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

应答示例：

~~~json
{
    "code": 200000,
    "info": "execute success",
    "amount": 0,
    "size": 0
}
~~~

数据结构：

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

| 参数名              | 类型                 | 描述                                                   | 是否必填  |
| ------------------- | ------------------ | ----------------------------------------------------- | -------- |
| service             | string             | 服务名                                                    | 是     |
| namespace           | string             | 命名空间                                                   | 是     |
| host                | string             | 实例的IP                                                   | 是     |
| port                | string             | 实例的端口                                                  | 是     |
| vpc_id              | string             | VPC ID                                                    | 否     |
| protocol            | string             | 对应端口的协议                                               | 否     |
| version             | string             | 版本                                                       | 否     |
| priority            | string             | 优先级                                                      | 否     |
| weight              | string             | 权重(默认值100)                                              | 是     |
| enable_health_check | bool               | 是否开启健康检查                                             | 是      |
| health_check        | HealthCheck        | 健康检查类别具体描述信息(如果enable_health_check==true，必须填写) | 否     |
| healthy             | bool               | 实例健康标志(默认为健康的)                                     | 是     |
| isolate             | bool               | 实例隔离标志(默认为不隔离的)                                   | 是     |
| location            | Location           | 实例位置信息                                                 | 是     |
| metadata            | map<string,string> | 实例标签信息，最多只能存储64对 *key-value*                      | 否     |
| service_token       | string             | service的token信息                                          | 是     |
`
	EnrichUpdateInstancesIsolateApiNotes = `
请求示例：

~~~
PUT /instances/isolate/host

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
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

应答示例：

~~~json
{
    "code": 200000,
    "info": "execute success",
    "amount": 0,
    "size": 0
}
~~~

数据结构：

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
| ------------------- | ------------------ | ------------------------------------------------ | --------|
| service             | string             | 服务名                                             | 是       |
| namespace           | string             | 命名空间                                           | 是       |
| host                | string             | 实例的IP                                           | 是       |
| port                | string             | 实例的端口                                          | 是       |
| vpc_id              | string             | VPC ID                                            | 否       |
| protocol            | string             | 对应端口的协议                                       | 否       |
| version             | string             | 版本                                                | 否       |
| priority            | string             | 优先级                                              | 否       |
| weight              | string             | 权重(默认值100)                                       | 是       |
| enable_health_check | bool               | 是否开启健康检查                                       | 是       |
| health_check        | HealthCheck        | 健康检查类别具体描述信息(如果enable_health_check==true，必须填写) | 否       |
| healthy             | bool               | 实例健康标志(默认为健康的)                                        | 是     |
| isolate             | bool               | 实例隔离标志(默认为不隔离的)                                      | 是     |
| location            | Location           | 实例位置信息                                                    | 是  |
| metadata            | map<string,string> | 实例标签信息，最多只能存储64对 *key-value*                         | 否   |
| service_token       | string             | service的token信息                                           | 是   |
`
	EnrichGetInstancesApiNotes = `
请求示例

~~~
GET /naming/v1/instances?service=&namespace=&{参数key}={参数值}

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

~~~

| 参数名      | 类型   | 描述             | 是否必填                                                        |
| ----------- | ------ | ---------------- | ----------------------------------------------------------- |
| service     | string | 服务名称         | 是                                                            |
| namespace   | string | 命名空间         | 是                                                            |
| host        | string | 实例IP           | 是(要么（service，namespace）存在，要么host存在，不然视为参数不完整) |
| port        | uint   | 实例端口         | 否                                                            |
| keys        | string | 标签key          | 只允许填写一个key                                               |
| values      | string | 标签value        | 只允许填写一个value                                              |
| healthy     | string | 实例健康状态     | 否                                                             |
| isolate     | string | 实例隔离状态     | 否                                                            |
| protocol    | string | 实例端口协议状态 | 否                                                             |
| version     | string | 实例版本         | 否                                                            |
| cmdb_region | string | 实例region信息   | 否                                                            |
| cmdb_zone   | string | 实例zone信息     | 否                                                           |
| cmdb_idc    | string | 实例idc信息      | 否                                                          |
| offset      | uint   | 查询偏移量       | 否                                                        |
| limit       | uint   | 查询条数         | 否                                                           |

应答示例：
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
	EnrichGetInstancesCountApiNotes = `
请求示例：
~~~
GET /naming/v1/instances/count

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

~~~

返回示例：
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
	"users": [],
	"userGroups": [],
	"authStrategies": [],
	"clients": []
}
~~~
`
	EnrichGetInstanceLabelsApiNotes = `
请求示例：
~~~
GET /naming/v1/instances/labels?service=&namespace=&{参数key}={参数值}

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

~~~

返回示例：
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
)
