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

// logger type
const (
	// HealthCheckLoggerName health-check logger name, can use FindScope function to get the logger
	HealthCheckLoggerName = "health-check"
	// StoreLoggerName storage logger name, can use FindScope function to get the logger
	StoreLoggerName = "store"
	// NamingLoggerName naming logger name, can use FindScope function to get the logger
	NamingLoggerName = "naming"
	// PluginLoggerName plugin logger name, can use FindScope function to get the logger
	PluginLoggerName = "plugin"
	// ServerLoggerName api server logger name, can use FindScope function to get the logger
	ServerLoggerName = "server"
)

var (
	healthCheckLogger = RegisterScope(HealthCheckLoggerName, "health check logging messages.", 0)
	storeLogger       = RegisterScope(StoreLoggerName, "storage logging messages.", 0)
	namingLogger      = RegisterScope(NamingLoggerName, "naming logging messages.", 0)
	pluginLogger      = RegisterScope(PluginLoggerName, "plugin logging messages.", 0)
	serverLogger      = RegisterScope(ServerLoggerName, "api server logging messages.", 0)
)

func allLoggerType() []string {
	return []string{HealthCheckLoggerName, StoreLoggerName, NamingLoggerName, PluginLoggerName, ServerLoggerName, DefaultLoggerName}
}

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
