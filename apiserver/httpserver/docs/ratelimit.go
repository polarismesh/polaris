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
	enrichCreateRateLimitsApiNotes = `
为服务创建多个限流规则，以对服务进行流量限制，按优先级顺序进行匹配，匹配到一个则执行该规则。

请求示例：
~~~
POST /naming/v1/ratelimits

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
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
回复示例：
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

数据结构：

> Ratelimit结构参数

| 参数名          | 类型          | 描述                                                         | 是否必填 |
| --------------- | ------------- | ------------------------------------------------------------ | -------- |
| name            | string        | 规则名                                                       | 是       |
| service         | string        | 规则所属的服务名，创建规则时，如果服务不存在，则会自动创建服务。 | 是       |
| namespace       | string        | 规则所属的命名空间                                           | 是       |
| method          | MatchString   | 规则所针对的服务接口                                         | 否       |
| arguments       | MatchArgument | 请求参数匹配条件，需全匹配才通过                             | 否       |
| resource        | string        | 限流资源，默认为QPS(针对QPS进行限流)                         | 否       |
| type            | string        | 限流类型，支持LOCAL（单机限流）, GLOBAL（分布式限流）        | 是       |
| amounts         | Amount[]      | 限流配额，包含限流周期和配额总数，可配置多个                 | 是       |
| regex_combine   | bool          | 合并计算配额，对于匹配到同一条正则表达式规则的多个不同的请求进行合并计算，默认为false | 否       |
| disable         | bool          | 是否启用该限流规则，默认为false（启用）                      | 否       |
| action          | string        | 限流效果，支持REJECT（直接拒绝）,UNIRATE（匀速排队），默认REJECT | 否       |
| failover        | string        | 失败降级措施，仅分布式限流有效，当远程token服务出现故障时，本地如何降级。 | 否       |
| max_queue_delay | int           | 最大排队时长，单位秒，仅对匀速排队生效。默认1秒              | 否       |

> Amount结构参数

| 参数名        | 类型   | 描述                                                 | 是否必填 |
| ------------- | ------ | ---------------------------------------------------- | -------- |
| maxAmount     | uint32 | 周期内最大配额数                                     | 是       |
| validDuration | string | 周期描述，支持duration类型的字符串，比如1s, 1m, 1h等 | 是       |

> MatchString结构参数

| 参数名 | 类型   | 描述                                                         | 是否必填 |
| ------ | ------ | ------------------------------------------------------------ | -------- |
| type   | string | 匹配类型，枚举，支持：EXACT（全匹配，默认），REGEX（正则表达式匹配），NOT_EQUALS（不等于），IN（包含），NOT_IN（不包含） | 是       |
| value  | string | 匹配的目标值，如果选择的是包含和不包含，则通过逗号分割多个值 | 是       |

> MatchArgument结构参数

| 参数名 | 类型        | 描述                                                         | 是否必填 |
| ------ | ----------- | ------------------------------------------------------------ | -------- |
| type   | string      | 参数类型，枚举，支持：CUSTOM，METHOD，HEADER，QUERY，CALLER_SERVICE，CALLER_IP | 是       |
| key    | string      | 参数键，对于HEADER、QUERY、CUSTOM，对应的是key值；对于CALLER_SERVICE，对应的是服务的命名空间值 | 是       |
| value  | MatchString | 参数值，对于HEADER、QUERY、CUSTOM，对应的是key所关联的value；对于CALLER_SERVICE，对应的是服务名，其他类型则是具体的值 | 是  |
`
enrichDeleteRateLimitsApiNotes = `
请求示例：

~~~
POST /naming/v1/ratelimits/delete

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

[
	{
		"id": "6942526fbac545848cd8fb32a3a55bb6" //规则ID，必填
	}
]
~~~

应答示例：

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

请求示例：

~~~
PUT /naming/v1/ratelimits

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
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

应答示例：

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
请求示例：

~~~
GET /naming/v1/ratelimits?参数名=参数值

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}
~~~

| 参数名    | 类型   | 描述                                                         | 是否必填 |
| --------- | ------ | ------------------------------------------------------------ | -------- |
| id        | string | 规则ID                                                       | 否       |
| name      | string | 规则名                                                       | 否       |
| service   | string | 服务名                                                       | 否       |
| namespace | string | 命名空间                                                     | 否       |
| method    | string | 限流接口名，默认为模糊匹配                                   | 否       |
| disable   | bool   | 规则是否启用，true为不启用，false为启用                      | 否       |
| brief     | bool   | 是否只显示概要信息，brief=true时，则不返回规则详情，只返回规则列表概要信息，默认为false | 否       |
| offset    | int    | 分页的起始位置，默认为0                                      | 否       |
| limit     | int    | 每页行数，默认100                                            | 否       |

应答示例：

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
请求示例：

~~~
PUT /naming/v1/ratelimits/enable
[
	{
		"id": "6942526fbac545848cd8fb32a3a55bb6", //规则ID，必填
		"disable": true // 是否禁用，true为不启用，false为启用
	}
]
~~~

应答示例：

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
)
