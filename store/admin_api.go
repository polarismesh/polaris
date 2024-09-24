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

package store

import (
	"time"

	"github.com/polarismesh/polaris/common/model/admin"
)

const (
	ElectionKeySelfServiceChecker = "polaris.checker"
	ElectionKeyMaintainJob        = "MaintainJob"
)

type AdminStore interface {
	// StartLeaderElection start leader election
	StartLeaderElection(key string) error
	// IsLeader whether it is leader node
	IsLeader(key string) bool
	// ListLeaderElections list all leaderelection
	ListLeaderElections() ([]*admin.LeaderElection, error)
	// ReleaseLeaderElection force release leader status
	ReleaseLeaderElection(key string) error
	// BatchCleanDeletedInstances batch clean soft deleted instances
	BatchCleanDeletedInstances(timeout time.Duration, batchSize uint32) (uint32, error)
	// GetUnHealthyInstances get unhealthy instances which mtime time out
	GetUnHealthyInstances(timeout time.Duration, limit uint32) ([]string, error)
	// BatchCleanDeletedClients batch clean soft deleted clients
	BatchCleanDeletedClients(timeout time.Duration, batchSize uint32) (uint32, error)
	// BatchCleanDeletedServices batch clean soft deleted clients
	BatchCleanDeletedServices(timeout time.Duration, batchSize uint32) (uint32, error)
	// BatchCleanDeletedRules batch clean soft deleted clients
	BatchCleanDeletedRules(rule string, timeout time.Duration, batchSize uint32) (uint32, error)
	// BatchCleanDeletedConfigFiles batch clean soft deleted clients
	BatchCleanDeletedConfigFiles(timeout time.Duration, batchSize uint32) (uint32, error)
}

// LeaderChangeEvent
type LeaderChangeEvent struct {
	Key        string
	Leader     bool
	LeaderHost string
}
