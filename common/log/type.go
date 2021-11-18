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

package log

//日志类型
const (
	// HealthCheckLoggerName 健康检查日志对象
	HealthCheckLoggerName = "health-check"
	// StoreLoggerName 存储日志对象
	StoreLoggerName = "store"
	// NamingLoggerName 注册日志对象
	NamingLoggerName = "naming"
	// PluginLoggerName 插件日志对象
	PluginLoggerName = "plugin"
	// ServerLoggerName 接口日志对象
	ServerLoggerName = "server"
)

var (
	healthCheckLogger = RegisterScope(HealthCheckLoggerName, "health check logging messages.", 0)
	storeLogger       = RegisterScope(StoreLoggerName, "storage logging messages.", 0)
	namingLogger      = RegisterScope(NamingLoggerName, "naming logging messages.", 0)
	pluginLogger      = RegisterScope(PluginLoggerName, "plugin logging messages.", 0)
	serverLogger      = RegisterScope(ServerLoggerName, "api server logging messages.", 0)
)

func GetDefaultLogger() *Scope {
	return defaultScope
}

func GetHealthCheckLogger() *Scope {
	return healthCheckLogger
}

func GetStoreLogger() *Scope {
	return storeLogger
}

func GetNamingLogger() *Scope {
	return namingLogger
}

func GetPluginLogger() *Scope {
	return pluginLogger
}

func GetServerLogger() *Scope {
	return serverLogger
}
