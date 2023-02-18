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
	EnrichRegisterInstanceApiNotes = `
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
`
	EnrichDeregisterInstanceApiNotes = `
请求示例

~~~
POST /v1/DeRegisterInstance

# 开启北极星客户端接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}
~~~
`
	EnrichHeartbeatApiNotes = `

请求示例

~~~
POST /v1/Heartbeat

# 开启北极星客户端接口鉴权开关后，需要添加下面的 header
Header X-Polaris-Token: {访问凭据}

{
    "namespace": "", // 命名空间，必填；string
    "service": "",	// 服务名称，必填；string
    "host":"",		// 实例 host 信息，必填；string
    "port": 80		// 实例 port 信息，必填；int
}
~~~


应答示例：

- 正常心跳上报结果。

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

- 若实例不存在或者实例未开启心跳上报

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
	EnrichReportClientApiNotes = `
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
`
	EnrichDiscoverApiNotes = ``
)
