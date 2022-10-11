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
	// NamingLoggerName naming logger name, can use FindScope function to get the logger
	NamingLoggerName = "naming"
	// ConfigLoggerName config logger name, can use FindScope function to get the logger
	ConfigLoggerName = "config"
	// CacheLoggerName cache logger name, can use FindScope function to get the logger
	CacheLoggerName = "cache"
	// AuthLoggerName auth logger name, can use FindScope function to get the logger
	AuthLoggerName = "auth"
	// StoreLoggerName store logger name, can use FindScope function to get the logger
	StoreLoggerName = "store"
	// APIServerLoggerName apiserver logger name, can use FindScope function to get the logger
	APIServerLoggerName = "apiserver"
	// XDSLoggerName xdsv3 logger name, can use FindScope function to get the logger
	XDSLoggerName = "xdsv3"
	// AuthPlatformLoggerName platform logger name, can use FindScope function to get the logger
	AuthPlatformLoggerName = "platform"
	// DiscoverEventLoggerName discoverEventLocal logger name, can use FindScope function to get the logger
	DiscoverEventLoggerName = "discoverEventLocal"
	// DiscoverEventLokiLoggerName discoverEventLoki logger name, can use FindScope function to get the logger
	DiscoverEventLokiLoggerName = "discoverEventLoki"
	// DiscoverStatLoggerName discoverStat logger name, can use FindScope function to get the logger
	DiscoverStatLoggerName = "discoverStat"
	// HealthcheckLoggerName healthcheck logger name, can use FindScope function to get the logger
	HealthcheckLoggerName = "healthcheck"
	// RateLimitLoggerName rateLimit logger name, can use FindScope function to get the logger
	RateLimitLoggerName = "rateLimit"
	// StatisLoggerName statis logger name, can use FindScope function to get the logger
	StatisLoggerName = "statis"
	// CmdbLoggerName cmdb logger name, can use FindScope function to get the logger
	CmdbLoggerName = "cmdb"
	// HistoryLoggerName history logger name, can use FindScope function to get the logger
	HistoryLoggerName = "history"
	// PasswordLoggerName password logger name, can use FindScope function to get the logger
	PasswordLoggerName = "password"
)

func allLoggerTypes() []string {
	return []string{NamingLoggerName, ConfigLoggerName, CacheLoggerName,
		AuthLoggerName, StoreLoggerName, APIServerLoggerName, XDSLoggerName,
		DiscoverEventLoggerName, AuthPlatformLoggerName, DiscoverEventLokiLoggerName,
		DiscoverStatLoggerName, HealthcheckLoggerName, RateLimitLoggerName, StatisLoggerName, CmdbLoggerName,
		HistoryLoggerName, PasswordLoggerName, DefaultLoggerName}
}
