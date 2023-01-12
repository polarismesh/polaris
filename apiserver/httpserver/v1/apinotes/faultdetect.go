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
	EnrichGetFaultDetectRulesApiNotes = `
根据输入条件查询一条或多条故障探测规则。	
	
请求示例：

~~~
GET /naming/v1/faultdetects?参数名=参数值

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}
~~~

| 参数名           | 类型              | 含义                                             |
| ---------------- | ----------------- | ------------------------------------------------ |
| brief            | "true" or "false" | 只返回列表信息，不返回详情                       |
| offset           | int               | 分页起始下标                                     |
| limit            | int               | 分页长度                                         |
| id               | string            | 规则ID，精准匹配                                 |
| name             | string            | 规则名，模糊匹配                                 |
| service          | string            | 规则所关联服务，必须和serviceNamespace一起用     |
| serviceNamespace | string            | 规则所关联服务的命名空间，必须和service一起用    |
| dstService       | string            | 规则的目标服务名，模糊匹配                       |
| dstNamespace     | string            | 规则的目标命名空间，模糊匹配                     |
| dstMethod        | string            | 规则的目标方法名，模糊匹配                       |
| description      | string            | 规则描述，模糊匹配                               |

应答示例：

~~~
{
 "code": 200000,
 "info": "execute success",
 "amount": 2,
 "size": 2,
 "data": [
   {
      "@type": "type.googleapis.com/v1.FaultDetectRule",
      "id": "6942526fbac545848cd8fb32a3a55bb6",
	  "name": "xxxx"
      "description": "abc",
      "target_service": {
      	  "service": "dstSvc",
      	  "namespace": "default"
      	  "method": {
      	    "type": "EXACT",
      	    "value": "test"
      	  }
	  }, 
      "interval": 30,
      "timeout": 60,
      "protocol": "HTTP",
      "http_config": {
      	"method": "GET",
      	"url": "/health",
      	"headers": [
         {
        	"key": "uin",
        	"value": "12345"
         }
        ],
        "body": "request blocked"
      },
      "tcp_config": {
      	"send": "0x11",
      	"receive": [
      		"0x50",
      		"0x51"
      	]
      },
      "udp_config": {
      	"send": "0x11",
      	"receive": [
      		"0x50",
      		"0x51"
      	]
      }
   }
}
~~~
`

	EnrichCreateFaultDetectRulesApiNotes = `
创建一条或多条故障探测规则。

请求示例：
~~~
POST /naming/v1/faultdetects

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

[
   {
      "name": "xxxx"
      "description": "abc",
      "target_service": {
      	  "service": "dstSvc",
      	  "namespace": "default"
      	  "method": {
      	    "type": "EXACT",
      	    "value": "test"
      	  }
	  }, 
      "interval": 30,
      "timeout": 60,
      "protocol": "HTTP",
      "http_config": {
      	"method": "GET",
      	"url": "/health",
      	"headers": [
         {
        	"key": "uin",
        	"value": "12345"
         }
        ],
        "body": "request blocked"
      },
      "tcp_config": {
      	"send": "0x11",
      	"receive": [
      		"0x50",
      		"0x51"
      	]
      },
      "udp_config": {
      	"send": "0x11",
      	"receive": [
      		"0x50",
      		"0x51"
      	]
      }
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
   "data": [
   {
      "@type": "type.googleapis.com/v1.FaultDetectRule",
      "id": "be97bc73ae444b0095a379f56ec98efd",
      
   }
  }
 ]
}
~~~

数据结构：

> FaultDetectRule结构参数

| 参数名          | 类型          | 描述                                                         | 是否必填 |
| --------------- | ------------- | ------------------------------------------------------------ | -------- |
| name            | string        | 规则名                                                       | 是       |
| namespace       | string        | 规则所属的命名空间                                           | 是       |
| description     | string        | 规则描述                                                     | 否       |
| target_service       | DestinationService        | 探测规则所关联的服务                          | 否       |
| interval             | uint32      | 探测周期                                                 | 是       |
| timeout              | uint32                | 探测超时时长                                    | 否       |
| port                 | uint32                | 探测端口号                                      | 否       |
| protocol             | Protocol              | 熔断协议（HTTP, TCP, UDP)                       | 是       |
| http_config          | HttpProtocolConfig      | Http探测配置                                  | 否       |
| tcp_config           | TcpProtocolConfig        | Tcp探测配置                                  | 否       |
| udp_config           | UdpProtocolConfig        | Udp探测配置                                  | 否       |

> DestinationService结构参数

| 参数名        | 类型   | 描述                                                 | 是否必填 |
| ------------- | ------ | ---------------------------------------------------- | -------- |
| service       | string | 服务名                                               | 是       |
| namespace     | string | 命名空间                                             | 是       |
| method        | MatchString结构参数 | 接口匹配条件                            | 否       |

> MatchString结构参数

| 参数名 | 类型   | 描述                                                         | 是否必填 |
| ------ | ------ | ------------------------------------------------------------ | -------- |
| type   | string | 匹配类型，枚举，支持：EXACT（全匹配，默认），REGEX（正则表达式匹配），NOT_EQUALS（不等于），IN（包含），NOT_IN（不包含） | 是       |
| value  | string | 匹配的目标值，如果选择的是包含和不包含，则通过逗号分割多个值 | 是       |

> HttpProtocolConfig结构参数

| 参数名 | 类型        | 描述                                                         | 是否必填 |
| ------ | ----------- | ------------------------------------------------------------ | -------- |
| method   | string      | HTTP探测请求的method                                         | 否       |
| url      | string      | HTTP探测请求的URL                                            | 否       |
| headers      | MessageHeader      | HTTP探测请求的消息头                               | 否       |
| body      | string      | HTTP探测请求的消息体                                            | 否       |

> TcpProtocolConfig结构参数

| 参数名 | 类型        | 描述                                                         | 是否必填 |
| ------ | ----------- | ------------------------------------------------------------ | -------- |
| send   | string      | TCP探测的发送的二进制请求                                        | 否       |
| receive  | []string  | TCP探测的发送的二进制应答数，用于对比应答是否正确                     | 否       |

> UdpProtocolConfig结构参数

| 参数名 | 类型        | 描述                                                         | 是否必填 |
| ------ | ----------- | ------------------------------------------------------------ | -------- |
| send   | string      | UDP探测的发送的二进制请求                                        | 否       |
| receive  | []string  | UDP探测的发送的二进制应答数，用于对比应答是否正确                     | 否       |
`

	EnrichDeleteFaultDetectRulesApiNotes = `
删除一条或多条故障探测规则。

请求示例：

~~~
POST /naming/v1/faultdetects/delete

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
   "data": [
   {
      "@type": "type.googleapis.com/v1.FaultDetectRule",
      "id": "6942526fbac545848cd8fb32a3a55bb6",
      
   }
  }
 ]
}
~~~
`

	EnrichUpdateFaultDetectRulesApiNotes = `
更新一条或多条故障探测规则。

请求示例：
~~~
PUT /naming/v1/faultdetects

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

[
   {
      "id": "6942526fbac545848cd8fb32a3a55bb6",
      "name": "xxxx"
      "description": "abc",
      "target_service": {
      	  "service": "dstSvc",
      	  "namespace": "default"
      	  "method": {
      	    "type": "EXACT",
      	    "value": "test"
      	  }
	  }, 
      "interval": 30,
      "timeout": 60,
      "protocol": "HTTP",
      "http_config": {
      	"method": "GET",
      	"url": "/health",
      	"headers": [
         {
        	"key": "uin",
        	"value": "12345"
         }
        ],
        "body": "request blocked"
      },
      "tcp_config": {
      	"send": "0x11",
      	"receive": [
      		"0x50",
      		"0x51"
      	]
      },
      "udp_config": {
      	"send": "0x11",
      	"receive": [
      		"0x50",
      		"0x51"
      	]
      }
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
   "data": [
   {
      "@type": "type.googleapis.com/v1.FaultDetectRule",
      "id": "be97bc73ae444b0095a379f56ec98efd",
      
   }
  }
 ]
}
~~~
`
)
