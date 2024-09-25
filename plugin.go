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

package main

import (
	_ "github.com/polarismesh/polaris/apiserver/eurekaserver"
	_ "github.com/polarismesh/polaris/apiserver/grpcserver/config"
	_ "github.com/polarismesh/polaris/apiserver/grpcserver/discover"
	_ "github.com/polarismesh/polaris/apiserver/httpserver"
	_ "github.com/polarismesh/polaris/apiserver/l5pbserver"
	_ "github.com/polarismesh/polaris/apiserver/nacosserver"
	_ "github.com/polarismesh/polaris/apiserver/xdsserverv3"
	_ "github.com/polarismesh/polaris/auth/policy"
	_ "github.com/polarismesh/polaris/auth/user"
	_ "github.com/polarismesh/polaris/cache"
	_ "github.com/polarismesh/polaris/cache/auth"
	_ "github.com/polarismesh/polaris/cache/client"
	_ "github.com/polarismesh/polaris/cache/config"
	_ "github.com/polarismesh/polaris/cache/namespace"
	_ "github.com/polarismesh/polaris/cache/service"
	_ "github.com/polarismesh/polaris/config/interceptor"
	_ "github.com/polarismesh/polaris/plugin/cmdb/memory"
	_ "github.com/polarismesh/polaris/plugin/crypto/aes"
	_ "github.com/polarismesh/polaris/plugin/discoverevent/local"
	_ "github.com/polarismesh/polaris/plugin/healthchecker/leader"
	_ "github.com/polarismesh/polaris/plugin/healthchecker/memory"
	_ "github.com/polarismesh/polaris/plugin/healthchecker/redis"
	_ "github.com/polarismesh/polaris/plugin/history/logger"
	_ "github.com/polarismesh/polaris/plugin/password"
	_ "github.com/polarismesh/polaris/plugin/ratelimit/token"
	_ "github.com/polarismesh/polaris/plugin/statis/logger"
	_ "github.com/polarismesh/polaris/plugin/statis/prometheus"
	_ "github.com/polarismesh/polaris/plugin/whitelist"
	_ "github.com/polarismesh/polaris/service/interceptor"
	_ "github.com/polarismesh/polaris/store/boltdb"
	_ "github.com/polarismesh/polaris/store/mysql"
)
