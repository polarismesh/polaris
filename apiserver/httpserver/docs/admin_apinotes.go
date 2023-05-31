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
POST /maintain/v1/apiserver/conn/close
Header X-Polaris-Token: {访问凭据}
Header Content-Type: application/json
~~~

| 参数名   	| 类型   	| 描述                	 | 是否必填 	|
|----------	|--------	|--------------------- | ----------	|
| protocol 	| string 	| 查看指定协议 server 	| 是       	|
| host     	| string 	| 查看指定host        	| 否       	|
| amount   	| integer 	| 总量                  | 否       	|
| port   	| string 	| 实例的端口             | 否       	|
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

请求参数：

| 参数名              | 类型                | 描述                                                           | 是否必填 |
| ------------------- | ------------------ | -------------------------------------------------------------- | -------- |
| service             | string             | 服务名                                                          | 是       |
| namespace           | string             | 命名空间                                                        | 是       |
| host                | string             | 实例的IP                                                        | 是       |
| port                | string             | 实例的端口                                                      | 是       |
| location            | Location             | 实例位置信息                                                  | 是       |
| enable_health_check | boolean            | 是否开启健康检查                                                 | 是       |
| health_check        | HealthCheck        | 健康检查类别具体描述信息(如果enable_health_check==true，必须填写)  | 是       |
| metadata            | map<string,string> | 实例标签信息，最多只能存储64对 *key-value*                        | 是       |

> Location 参数

| 参数名 | 类型   | 描述 | 是否必填 |
| ------ | ------ | ---- | -------- |
| region | string | 地区 | 否       |
| zone   | string | 地域 | 否       |
| campus | string | 园区 | 否       |

> HealthCheck 参数

| 参数名    | 类型         | 描述                        | 是否必填 |
| --------- | ------------ | --------------------------- | -------- |
| type      | int          | 0(Unknow)/1(Heartbeat)      | 是       |
| heartbeat | {"ttl": int} | 心跳间隔(范围为区间(0, 60]) | 是       |
`
	enrichBatchCleanInstancesApiNotes = `
请求示例：

~~~
POST /maintain/v1/instance/batchclean
Header X-Polaris-Token: {访问凭据}
Header Content-Type: application/json
~~~

请求参数：

| 参数名              | 类型                | 描述                                       | 是否必填  |
| ------------------- | ------------------ | ------------------------------------------ | -------- |
| batch_size          | int                | 清理的数量                                  |    是    |
`
	enrichGetLastHeartbeatApiNotes = `
请求示例：

~~~
GET /maintain/v1/instance/heartbeat?id=xxx&service=xxx&namespace=xxx&host=xxx&port=xxx&host=xxx&vpc_id=xxx
Header X-Polaris-Token: {访问凭据}
~~~

请求参数：

| 参数名              | 类型               | 描述                                       | 是否必填 |
| ------------------- | ------------------ | ------------------------------------------ | -------- |
| id                  | string             | 实例id 如果存在id，后面参数可以不填名         | 否       |
| service             | string             | 服务名                                     | 否       |
| namespace           | string             | 命名空间                                   | 否       |
| host                | string             | 实例的IP                                   | 否       |
| port                | string             | 实例的端口                                 | 否       |
| vpc_id              | string             | VPC ID                                     | 否       |
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
`
	enrichListLeaderElectionsApiNotes = `
请求示例：

~~~
GET /maintain/v1/leaders
Header X-Polaris-Token: {访问凭据}
~~~

返回示例：
~~~
[
 {
  "ElectKey": "polaris.checker",
  "Host": "127.0.0.1",
  "Ctime": 1669994957,
  "CreateTime": "2022-12-02T23:29:17+08:00",
  "Mtime": 1671288397,
  "ModifyTime": "2022-12-17T22:46:37+08:00",
  "Valid": true
 }
]
`
	enrichReleaseLeaderElectionApiNotes = `
请求示例：

~~~
POST /maintain/v1/leaders/release
Header X-Polaris-Token: {访问凭据}
`
)
