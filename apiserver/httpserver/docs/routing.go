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
	EnrichCreateRoutingsApiNotes = `
为服务创建一个路由规则，以对服务进行流量调度，一个服务只能有一个路由规则。

请求示例：

~~~
POST /naming/v1/routings

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
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

回复示例：

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

数据结构：

> routing结构参数

| 参数名    | 类型   | 描述               | 是否必填 |
| --------- | ------ | ------------------ | -------- |
| service   | string | 规则所属的服务名   | 是       |
| namespace | string | 规则所属的命名空间 | 是       |
| inbounds  | route[]    | 入流量规则 | 否 |
| outbounds | route[] | 出流量规则 | 否 |
| service_token | string | 鉴权token，当开启鉴权后需要传入 | 否 |

> route结构参数

| 参数名       | 类型    | 描述 | 是否必填 |
| ------------ | ------- | ---- | -------- |
| sources      | source[] | 请求匹配条件 | 否 |
| destinations | destination[] | 目标实例分组匹配条件 | 是 |

> source结构参数

| 参数名    | 类型                     | 描述                           | 是否必填 |
| --------- | ------------------------ | ------------------------------ | -------- |
| service   | string                   | 主调方服务名，填*代表全匹配    | 否       |
| namespace | string                   | 被调方服务名，填*代表全匹配    | 否       |
| metadata  | map<string, matchString> | 匹配参数，需全匹配所有KV才通过 | 否       |

> destination结构参数

| 参数名    | 类型                     | 描述                                                         | 是否必填 |
| --------- | ------------------------ | ------------------------------------------------------------ | -------- |
| service   | string                   | 被调方服务名，填*代表全匹配                                  | 否       |
| namespace | string                   | 被调方命名空间，填*代表全匹配                                | 否       |
| metadata  | map<string, matchString> | 示例标签匹配参数，需全匹配才通过                             | 否       |
| priority  | int32                    | 优先级，数值越小，优先级越高，请求会优先选取优先级最高的实例分组进行路由，只有该分组没有可用实例才会选择次高优先级的分组 | 否       |
| weight    | int32                    | 分组权重，优先级相同的多个分组，按权重比例进行请求分配       | 否       |

> matchString结构参数

| 参数名     | 类型   | 描述                                                         | 是否必填 |
| ---------- | ------ | ------------------------------------------------------------ | -------- |
| type       | string | 匹配类型，枚举值，支持：EXACT（全匹配，默认），REGEX（正则表达式匹配） | 否       |
| value      | string | 匹配的目标值                                                 | 是       |
| value_type | string | 值类型，枚举值，支持：TEXT（文本，默认），PARAMETER（参数，路由规则值使用动态参数时用到） | 否       |
`
	EnrichDeleteRoutingsApiNotes = `
删除服务下的路由规则

请求示例：

~~~
POST /naming/v1/routings/delete

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

[
    {
        "service_token":"...",
        "service":"...",
        "namespace":"..."
    }
]
~~~

回复示例：

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
	EnrichUpdateRoutingsApiNotes = `
更新服务下的路由规则的相关信息

请求示例：

~~~
PUT /naming/v1/routings

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
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

回复示例：

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

数据结构：

> routing结构参数

| 参数名    | 类型   | 描述               | 是否必填 |  |
| --------- | ------ | ------------------ | -------- | -------- |
| service   | string | 规则所属的服务名   | 是       | 否      |
| namespace | string | 规则所属的命名空间 | 是       | 否      |
| inbounds  | route[]    | 入流量规则 | 否 | 是 |
| outbounds | route[] | 出流量规则 | 否 | 是 |
| service_token | string | 鉴权token，当开启鉴权后需要传入 | 否 | 否 |
`
	EnrichGetRoutingsApiNotes = `
请求示例：

~~~
GET /naming/v1/routings?参数名=参数值

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}
~~~

| 参数名    | 类型   | 描述                    | 是否必填 |
| --------- | ------ | ----------------------- | -------- |
| service   | string | 服务名                  | 否       |
| namespace | string | 命名空间                | 否       |
| offset    | int    | 分页的起始位置，默认为0 | 否       |
| limit     | int    | 每页行数，默认100       | 否       |

应答示例：

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
)
