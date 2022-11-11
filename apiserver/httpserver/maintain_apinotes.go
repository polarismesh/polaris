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

const (
	enrichGetServerConnectionsApiNotes = `
请求示例：

~~~
GET /maintain/v1/apiserver/conn?protocol=xxx&host=xxx
Header X-Polaris-Token: {访问凭据}
~~~

| 参数名   | 类型   | 描述                | 是否必填 |
|----------|--------|---------------------|----------|
| protocol | string | 查看指定协议 server | 是       |
| host     | string | 查看指定host        | 否       |

应答示例：

~~~json
~~~
`
	enrichGetServerConnStatsApiNotes = `
请求示例：
~~~
GET /maintain/v1/apiserver/conn/stats?protocol=xxx&host=xxx
Header X-Polaris-Token: {访问凭据}
~~~

| 参数名   	| 类型   	| 描述                	| 是否必填 	|
|----------	|--------	|---------------------	|----------	|
| protocol 	| string 	| 查看指定协议 server 	| 是       	|
| host     	| string 	| 查看指定host        	| 否       	|
| amount   	| integer 	| 总量                	| 否       	|
`
	enrichCloseConnectionsApiNotes = `
请求示例：

~~~
POST /maintain/v1/apiserver/conn?protocol=xxx&host=xxx
Header X-Polaris-Token: {访问凭据}
Header Content-Type: application/json

[
    {
        "protocol": "someProtocol",
        "host": "someHost",
        "amount": "someAmount",
        "port": "port",
    } 
]
~~~
`
	enrichFreeOSMemoryApiNotes = `
请求示例：

~~~
POST /maintain/v1/memory/free
Header X-Polaris-Token: {访问凭据}
Header Content-Type: application/json
~~~
`
	enrichCleanInstanceApiNotes = `
请求示例：

~~~
POST /maintain/v1/instance/clean
Header X-Polaris-Token: {访问凭据}
Header Content-Type: application/json

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
~~~

`
	enrichBatchCleanInstancesApiNotes = `
请求示例：

~~~
POST /maintain/v1/instance/batchclean
Header X-Polaris-Token: {访问凭据}
Header Content-Type: application/json

{
    "batch_size": 100
}
~~~

`
	enrichGetLastHeartbeatApiNotes = `
请求示例：

~~~
GET /maintain/v1//instance/heartbeat?id=xxx
Header X-Polaris-Token: {访问凭据}
~~~

请求参数：

| 参数名              | 类型               | 描述                                                              | 是否必填 |
| ------------------- | ------------------ | ----------------------------------------------------------------- | -------- |
| id                  | string             | 实例id 如果存在id，后面参数可以不填名                                   | 否       |
| service             | string             | 服务名                                                            | 否       |
| namespace           | string             | 命名空间                                                          | 否       |
| host                | string             | 实例的IP                                                          | 否       |
| port                | string             | 实例的端口                                                        | 否       |
| vpc_id              | string             | VPC ID                                                            | 否       |
`
	enrichGetLogOutputLevelApiNotes = `
请求示例：

~~~
GET /maintain/v1/log/outputlevel
Header X-Polaris-Token: {访问凭据}
~~~

返回示例：
~~~
{
 "apiserver": "info",
 "auth": "info",
 "cache": "info",
 "config": "info",
 "default": "info",
 "healthcheck": "info",
 "naming": "info",
 "store": "info",
 "xdsv3": "info"
}
~~~
`
	enrichSetLogOutputLevelApiNotes = `
请求示例：

~~~
POST /maintain/v1/log/outputlevel
Header X-Polaris-Token: {访问凭据}

{
    "scope": "apiserver",
    "level": "info"
}
`
)
