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
	enrichGetCircuitBreakerByServiceApiNotes = `
请求示例：

~~~
GET /naming/v1/service/circuitbreaker?service=xxx&namespace=xxx

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}
~~~

应答示例：

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
	enrichCreateCircuitBreakersApiNotes = `

- 为服务创建一个熔断规则，以对服务下的故障节点进行剔除。
- 熔断规则可以分为被调规则和主调规则：
	- 被调规则针对所有的指定主调生效，假如不指定则对所有的主调生效。
	- 主调规则为当前主调方的规则，假如不指定则针对所有被调生效。
	- 被调规则与主调规则同时存在时，被调优先，被调规则生效。


请求示例：

~~~
POST /naming/v1/circuitbreakers

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
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

应答示例：

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

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
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

应答示例：

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

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

[
	{
		"id": "6942526fbac545848cd8fb32a3a55bb6" //熔断规则id
	}
]
~~~
`
	enrichUpdateCircuitBreakersApiNotes = `
请求示例：

~~~
POST /naming/v1/circuitbreakers

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
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

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

~~~
`
	enrichUnBindCircuitBreakersApiNotes = `
~~~
POST /naming/v1/circuitbreakers/unbind

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

~~~
`
	enrichGetCircuitBreakersApiNotes = `
请求示例：
~~~
GET /naming/v1/circuitbreakers?id={参数值}&version={参数值}

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

~~~
`
	enrichGetCircuitBreakerVersionsApiNotes = `
请求示例：
~~~
GET /naming/v1/circuitbreaker/versions?id={参数值}

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

~~~
`
	enrichGetMasterCircuitBreakersApiNotes = `
请求示例：
~~~
GET /naming/v1/circuitbreakers/master?id={参数值}

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

~~~
`
	enrichGetReleaseCircuitBreakersApiNotes = `
请求示例：
~~~
GET /naming/v1/circuitbreakers/release?id={参数值}

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

~~~
`
	enrichCreateCircuitBreakerRulesApiNotes = `
创建一条或多条熔断规则。

请求示例：
~~~
POST /naming/v1/circuitbreaker/rules

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

[
   {
      "name": "xxxx"
      "enable": true,
      "description": "abc",
      "level": SERVICE,
      "rule_matcher": {
      	"source": {    
      	  "service": "srcSvc",
      	  "namespace": "default"
      	},
      	"destination": {
      	  "service": "dstSvc",
      	  "namespace": "default"
      	  "method": {
      	    "type": "EXACT",
      	    "value": "test"
      	  }
      	}
      }, 
      "error_conditions": [          
        {          
          "input_type": "RET_CODE",  
          "condition": {             
             "type": "EXACT",       
      	     "value": "200"
          }
        }
      ],     
      "trigger_condition": [         
       {        	
      		"trigger_type": "ERROR_RATE", 
      		"error_count": 10,          
      		"error_percent": 50,          
      		"interval": 30,             
      		"minimum_request": 5 
      	}
      ],
      "recoverCondition": {
      	"sleep_window": 60
      	"consecutiveSuccess": 3
      },      
      "faultDetectConfig": {
      	"enable": true
      }
      "fallbackConfig”: {
        "enable": false,
        "response": {
        	"code": 500,
        	"headers": [
        	  {
        		"key": "uin",
        		"value": "12345"
        	  }
        	],
        	"body": "request blocked"
        }
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
      "@type": "type.googleapis.com/v1.CircuitBreakerRule",
      "id": "be97bc73ae444b0095a379f56ec98efd",
      
   }
  }
 ]
}
~~~

数据结构：

> CircuitBreakerRule结构参数

| 参数名          | 类型          | 描述                                                         | 是否必填 |
| --------------- | ------------- | ------------------------------------------------------------ | -------- |
| name            | string        | 规则名                                                       | 是       |
| namespace       | string        | 规则所属的命名空间                                           | 是       |
| enable          | string        | 是否启用该限流规则，默认为false（不启用）                    | 是       |
| description     | string        | 规则描述                                                     | 否       |
| level           | enum          | 限流级别：SERVICE, METHOD, GROUP, INSTANCE                   | 是       |
| rule_matcher    | RuleMatcher   | 规则的服务匹配条件                                           | 是       |
| error_conditions       | ErrorCondition        | 错误匹配条件                                  | 否       |
| trigger_condition      | TriggerCondition      | 限流触发条件                                  | 是       |
| max_ejection_percent   | uint32                | 最大熔断比例，防雪崩                          | 否       |
| recoverCondition       | RecoverCondition      | 熔断恢复条件                                  | 是       |
| faultDetectConfig      | FaultDetectConfig     | 故障探测配置                                  | 否       |
| fallbackConfig         | FallbackConfig        | 降级配置                                      | 否       |

> RuleMatcher结构参数

| 参数名        | 类型   | 描述                                                 | 是否必填 |
| ------------- | ------ | ---------------------------------------------------- | -------- |
| source        | SourceService      | 来源服务信息                             | 是       |
| destination   | DestinationService | 目标服务信息                             | 是       |

> SourceService结构参数

| 参数名        | 类型   | 描述                                                 | 是否必填 |
| ------------- | ------ | ---------------------------------------------------- | -------- |
| service       | string | 服务名                                               | 是       |
| namespace     | string | 命名空间                                             | 是       |

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

> ErrorCondition结构参数

| 参数名 | 类型        | 描述                                                         | 是否必填 |
| ------ | ----------- | ------------------------------------------------------------ | -------- |
| input_type   | enum      | 参数类型，枚举，支持：RET_CODE（返回码），DELAY（时延）  | 是       |
| condition  | MatchString | 参数值匹配条件，匹配上了，该请求被认定为失败请求         | 是       |

> TriggerCondition结构参数

| 参数名 | 类型        | 描述                                                         | 是否必填 |
| ------ | ----------- | ------------------------------------------------------------ | -------- |
| trigger_type   | enum      | 触发条件，枚举，支持：ERROR_RATE（一段时间错误率），CONSECUTIVE_ERROR（连续错误数）  | 是       |
| error_count  | uint32 | 连续错误数阈值                                                                            | 否       |
| error_percent  | uint32 | 错误率百分比阈值                                                                        | 否       |
| interval  | uint32 | 错误率统计时长，单位秒                                                                       | 否       |
| minimum_request  | uint32 | 错误率统计的最小请求数起始阈值                                                        | 否       |

> RecoverCondition结构参数

| 参数名 | 类型        | 描述                                                         | 是否必填 |
| ------ | ----------- | ------------------------------------------------------------ | -------- |
| sleep_window   | uint32      | 熔断时长，单位秒                                     | 是       |
| consecutiveSuccess  | uint32 | 恢复所需的连续成功数                                 | 是       |

> FaultDetectConfig结构参数

| 参数名 | 类型        | 描述                                                         | 是否必填 |
| ------ | ----------- | ------------------------------------------------------------ | -------- |
| enable   | bool      | 是否启用故障探测，默认否                                     | 否       |

> FallbackConfig结构参数

| 参数名 | 类型        | 描述                                                         | 是否必填 |
| ------ | ----------- | ------------------------------------------------------------ | -------- |
| enable   | bool      | 是否启用降级，默认否                                         | 否       |
| response   | FallbackResponse      | 降级应答                                       | 否       |


> FallbackResponse结构参数

| 参数名 | 类型        | 描述                                                         | 是否必填 |
| ------ | ----------- | ------------------------------------------------------------ | -------- |
| code   | int32       | 降级返回码                                                   | 是       |
| headers   | MessageHeader      | 降级应答的消息头                                   | 否       |
| body   | string      | 降级应答的消息体                                             | 否       |

> MessageHeader结构参数

| 参数名 | 类型        | 描述                                                         | 是否必填 |
| ------ | ----------- | ------------------------------------------------------------ | -------- |
| key    | string       | 降级应答的消息头的Key                                       | 是       |
| value   | string      | 降级应答的消息头的Value                                     | 是       |

`
	enrichDeleteCircuitBreakerRulesApiNotes = `
删除一条或多条熔断规则。	
	
请求示例：

~~~
POST /naming/v1/circuitbreaker/rules/delete

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
      "@type": "type.googleapis.com/v1.CircuitBreakerRule",
      "id": "6942526fbac545848cd8fb32a3a55bb6",
      
   }
  }
 ]
}
~~~

`
	enrichUpdateCircuitBreakerRulesApiNotes = `
更新一条或多条熔断规则。

请求示例：

~~~
PUT /naming/v1/circuitbreaker/rules

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

[
   {
      "id": "6942526fbac545848cd8fb32a3a55bb6",
      "name": "xxxx"
      "enable": true,
      "description": "abc",
      "level": SERVICE,
      "rule_matcher": {
      	"source": {    
      	  "service": "srcSvc",
      	  "namespace": "default"
      	},
      	"destination": {
      	  "service": "dstSvc",
      	  "namespace": "default"
      	  "method": {
      	    "type": "EXACT",
      	    "value": "test"
      	  }
      	}
      }, 
      "error_conditions": [          
        {          
          "input_type": "RET_CODE",  
          "condition": {             
             "type": "EXACT",       
      	     "value": "200"
          }
        }
      ],     
      "trigger_condition": [         
       {        	
      		"trigger_type": "ERROR_RATE", 
      		"error_count": 10,          
      		"error_percent": 50,          
      		"interval": 30,             
      		"minimum_request": 5 
      	}
      ],
      "recoverCondition": {
      	"sleep_window": 60
      	"consecutiveSuccess": 3
      },      
      "faultDetectConfig": {         // 主动探测配置
      	"enable": true               // 是否启用主动探测，如果启用，则按照服务所关联的探测规则进行探测
      }
      "fallbackConfig”: {           // 降级配置
        "enable": false,            // 是否启用降级
        "response": {               // 降级应答配置
        	"code": 500,            // 降级返回码
        	"headers": [            // 降级返回消息头
        	  {
        		"key": "uin",
        		"value": "12345"
        	  }
        	],
        	"body": "request blocked"   // 降级返回的消息体
        }
      }
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
      "@type": "type.googleapis.com/v1.CircuitBreakerRule",
      "id": "6942526fbac545848cd8fb32a3a55bb6",
      
   }
  }
 ]
}
~~~

`
	enrichGetCircuitBreakerRulesApiNotes = `
根据输入条件查询一条或多条熔断规则。	
	
请求示例：

~~~
GET /naming/v1/circuitbreaker/rules?参数名=参数值

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
| enable           | "true" or "false" | 规则是否启用                                     |
| level            | int数组，逗号分割 | 熔断级别，可输入多个：1服务，2接口，3分组，4实例 |
| service          | string            | 规则所关联服务，必须和serviceNamespace一起用     |
| serviceNamespace | string            | 规则所关联服务的命名空间，必须和service一起用    |
| srcService       | string            | 规则的源服务名，模糊匹配                         |
| srcNamespace     | string            | 规则的源命名空间，模糊匹配                       |
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
      "@type": "type.googleapis.com/v1.CircuitBreakerRule",
      "id": "6942526fbac545848cd8fb32a3a55bb6",
	  "name": "xxxx"
      "enable": true,
      "description": "abc",
      "level": SERVICE,
      "rule_matcher": {
      	"source": {    
      	  "service": "srcSvc",
      	  "namespace": "default"
      	},
      	"destination": {
      	  "service": "dstSvc",
      	  "namespace": "default"
      	  "method": {
      	    "type": "EXACT",
      	    "value": "test"
      	  }
      	}
      }, 
      "error_conditions": [          
        {          
          "input_type": "RET_CODE",  
          "condition": {             
             "type": "EXACT",       
      	     "value": "200"
          }
        }
      ],     
      "trigger_condition": [         
       {        	
      		"trigger_type": "ERROR_RATE", 
      		"error_count": 10,          
      		"error_percent": 50,          
      		"interval": 30,             
      		"minimum_request": 5 
      	}
      ],
      "recoverCondition": {
      	"sleep_window": 60
      	"consecutiveSuccess": 3
      },      
      "faultDetectConfig": {
      	"enable": true
      }
      "fallbackConfig”: {
        "enable": false,
        "response": {
        	"code": 500,
        	"headers": [
        	  {
        		"key": "uin",
        		"value": "12345"
        	  }
        	],
        	"body": "request blocked"
        }
      }
      
   }
}
~~~
`
	enrichEnableCircuitBreakerRulesApiNotes = `
请求示例：

~~~
PUT /naming/v1/circuitbreaker/rules/enable
[
	{
		"id": "6942526fbac545848cd8fb32a3a55bb6", //规则ID，必填
		"enable": true // 是否启用，true为启用，false为禁用
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
      "@type": "type.googleapis.com/v1.CircuitBreakerRule",
      "id": "6942526fbac545848cd8fb32a3a55bb6",
      
   }
  }
 ]
}
~~~
`

	enrichCreateRouterRuleApiNotes = `
创建路由规则

~~~
POST /naming/v2/routings

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

~~~
`
	enrichDeleteRouterRuleApiNotes = `
删除路由规则

~~~
DELETE /naming/v2/routings

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

~~~
`
	enrichUpdateRouterRuleApiNotes = `
更新路由规则

~~~
PUT /naming/v2/routings

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

~~~
`
	enrichGetRouterRuleApiNotes = `
获取路由规则

~~~
GET /naming/v2/routings

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

~~~
`
	enrichEnableRouterRuleApiNotes = `
更新路由规则

~~~
PUT /naming/v2/routings

# 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

~~~
`
)
