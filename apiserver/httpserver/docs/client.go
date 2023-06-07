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
	//nolint: lll
	enrichWatchConfigFileNotes = `
请求示例

~~~
POST /config/v1/WatchConfigFile

# 开启北极星客户端接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}
~~~

请求参数：

| 参数名              | 类型                | 描述                                                           | 是否必填 |
| ------------------- | ------------------ | -------------------------------------------------------------- | -------- |
| client_ip           | string             | 客户端IP                                                        | 是       |
| service_name        | string             | 服务名                                                          | 是       |
| watch_files         | WatchFiles         | 监听的配置文件                                                   | 是       |

> WatchFiles 参数

| 参数名        | 类型   | 描述                                    | 是否必填 |
| ------------ | ------ | --------------------------------------- | -------- |
| content      | string | 配置内容                                 | 是       |
| data_key     | string | 数据key                                  | 是       |
| file_name    | string | 配置文件名                               | 是       |
| group        | string | 配置文件分组                             | 是       |
| is_encrypted | string | 是否加密                                 | 是       |
| md5          | string | md5码                                    | 是       |
| namespace    | string | 命名空间                                 | 是       |
| public_key   | string | 公钥                                     | 是       |
| version      | string | 配置文件客户端版本号，刚启动时设置为 '0'   | 是       |
`
	enrichRegisterInstanceApiNotes = `
请求示例

~~~
POST /v1/RegisterInstance

# 开启北极星客户端接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

{
    "service": "xxxx",
    "namespace": "xxx",
    "host": "xxx",
    "port": 8080,
	"protocol": "xx",
	"version": "xx",
    "location": {
        "region": "xxx",
        "zone": "xxx",
        "campus": ""
    },
    "metadata": {
        "key": "value"
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
| protocol            | string             | 对应端口的协议                                                   | 是       |
| version             | string             | 版本                                                            | 是       |
| location            | Location           | 实例位置信息                                                     | 是       |
| metadata            | map<string,string> | 实例标签信息，最多只能存储64对 *key-value*                        | 是       |

> Location 参数

| 参数名 | 类型   | 描述 | 是否必填 |
| ------ | ------ | ---- | -------- |
| region | string | 地区 | 否       |
| zone   | string | 地域 | 否       |
| campus | string | 园区 | 否       |
`
	enrichDeregisterInstanceApiNotes = `
请求示例

~~~
POST /v1/DeRegisterInstance

# 开启北极星客户端接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}
~~~
`
	enrichHeartbeatApiNotes = `

请求示例

~~~
POST /v1/Heartbeat

# 开启北极星客户端接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}
~~~


应答示例：

- 正常心跳上报结果：

~~~json
{
    "code": 200000,
    "info": "execute success",
    "instance": {
        "service": "BootEchoServer",
        "namespace": "default",
        "host": "127.0.0.1",
        "port": 28888
    }
}
~~~

- 若实例不存在或者实例未开启心跳上报结果：

~~~json
{
    "code": 400141,
    "info": "heartbeat on disabled instance",
    "instance": {
        "service": "BootEchoServer",
        "namespace": "default",
        "vpc_id": null,
        "host": "127.0.0.1",
        "port": 28881
    }
}
~~~
`
	enrichReportClientApiNotes = `
请求示例

~~~
POST /v1/ReportClient

# 开启北极星客户端接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

{
	"host": "xxx",
	"type": "xxx",
	"version": "xx",
	"location": {
		"region": "xxx",
		"zone": "xxx",	
		"campus": ""
	}
}
~~~

请求参数：

| 参数名              | 类型                | 描述                                                           | 是否必填 |
| ------------------- | ------------------ | -------------------------------------------------------------- | -------- |
| host                | string             | 实例的IP                                                        | 是       |
| type                | string             | 对应端口的协议                                                   | 是       |
| version             | string             | 版本                                                            | 是       |
| location            | Location           | 实例位置信息                                                     | 是       |

> Location 参数

| 参数名 | 类型   | 描述 | 是否必填 |
| ------ | ------ | ---- | -------- |
| region | string | 地区 | 否       |
| zone   | string | 地域 | 否       |
| campus | string | 园区 | 否       |
`
	enrichDiscoverApiNotes = `
请求示例

~~~
POST /v1/Discover

# 开启北极星客户端接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}
~~~`
)
