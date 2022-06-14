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

package eurekaserver

import "time"

const (
	optionListenIP               = "listenIP"
	optionListenPort             = "listenPort"
	optionNamespace              = "namespace"
	optionRefreshInterval        = "refreshInterval"
	optionDeltaExpireInterval    = "deltaExpireInterval"
	optionConnLimit              = "connLimit"
	optionEnableSelfPreservation = "enableSelfPreservation"
)

const (
	DefaultNamespace            = "default"
	DefaultRefreshInterval      = 10
	DefaultDetailExpireInterval = 60
	// DefaultEnableSelfPreservation whether to enable preservation mechanism
	DefaultEnableSelfPreservation = true
	// DefaultSelfPreservationPercent instances unhealthy percent over 85% (around 15 min instances),
	// it will return all checked instances
	DefaultSelfPreservationPercent = 85
	// DefaultSelfPreservationDuration instance unhealthy check point to preservation,
	// instances over 15 min won't get preservation
	DefaultSelfPreservationDuration = 15 * time.Minute
)
