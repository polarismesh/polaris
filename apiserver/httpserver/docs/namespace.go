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
	enrichGetNamespacesApiNotes = `
 | 参数名 | 类型   | 描述                                             | 是否必填 |
 | ------ | ------ | ------------------------------------------------ | -------- |
 | name   | string | 命名空间唯一名称                                 | 是       |
 | offset | uint   | 查询偏移量                                       | 否       |
 | limit  | uint   | 查询条数，**最多查询100条**                      | 否       |
 
 
 请求示例：
 
 ~~~
 GET /{core|naming}/v1/namespaces?name=xxx&offset=xxx&limit=xxx
 
 # 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
 Header X-Polaris-Token: {访问凭据}
 ~~~
 
 应答示例：
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
 | user_ids         | []string | 可以操作该资源的用户，**仅当开启北极星鉴权时生效**         | 否       |
 | group_ids        | []string | 可以操作该资源的用户组，，**仅当开启北极星鉴权时生效**     | 否       |
 | remove_user_ids  | []string | 被移除的可操作该资源的用户，**仅当开启北极星鉴权时生效**   | 否       |
 | remove_group_ids | []string | 被移除的可操作该资源的用户组，**仅当开启北极星鉴权时生效** | 否       |
 
 
 请求示例：
 
 ~~~
 POST /{core|naming}/v1/namespaces
 
 # 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
 Header X-Polaris-Token: {访问凭据}
 
 [
     {
         "name": "...",
         "comment": "...",
         "user_ids": [...],
         "group_ids": [...],
         "remove_user_ids": [...],
         "remove_group_ids": [...],
     }
 ]
 ~~~
 
 应答示例：
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
 | token            | string   | 命名空间的token，用于权限鉴定                              | 是       |
 | user_ids         | []string | 可以操作该资源的用户，**仅当开启北极星鉴权时生效**         | 否       |
 | group_ids        | []string | 可以操作该资源的用户组，，**仅当开启北极星鉴权时生效**     | 否       |
 | remove_user_ids  | []string | 被移除的可操作该资源的用户，**仅当开启北极星鉴权时生效**   | 否       |
 | remove_group_ids | []string | 被移除的可操作该资源的用户组，**仅当开启北极星鉴权时生效** | 否       |
 
 请求示例：
 
 ~~~
 PUT /{core|naming}/v1/namespaces
 
 # 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
 Header X-Polaris-Token: {访问凭据}
 
 [
     {
         "name": "...",
         "comment": "...",
         "token": "...",
         "user_ids": [...],
         "group_ids": [...],
         "remove_user_ids": [...],
         "remove_group_ids": [...],
     }
 ]
 ~~~
 
 应答示例：
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
 | token  | string | 命名空间的token，用于权限鉴定  | 是       |
 
 请求示例：
 
 ~~~
 POST /{core|naming}/v1/namespaces/delete
 
 # 开启北极星服务端针对控制台接口鉴权开关后，需要添加下面的 header
 Header X-Polaris-Token: {访问凭据}
 
 [
     {
         "name": "...",
         "token": "..."
     }
 ]
 ~~~
 
 应答示例：
 ~~~json
 {
     "code": 200000,
     "info": "execute success",
     "size": 0
 }
 ~~~
 `
)
