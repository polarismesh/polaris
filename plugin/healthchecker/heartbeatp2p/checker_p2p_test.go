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

package heartbeatp2p

import (
	"sync"
	"testing"
	"time"

	commonlog "github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
	"github.com/stretchr/testify/assert"
)

var (
	nodes = []plugin.CheckerPeer{
		{
			Host: "127.0.0.1",
			ID:   "127.0.0.1:7070",
			Port: 7070,
		},
		{
			Host: "127.0.0.1",
			ID:   "127.0.0.1:8080",
			Port: 8080,
		},
		{
			Host: "127.0.0.1",
			ID:   "127.0.0.1:9090",
			Port: 9090,
		},
	}
)

func TestPeerToPeerHealthChecker(t *testing.T) {
	log.SetOutputLevel(commonlog.DebugLevel)
	checkers := make(map[string]*PeerToPeerHealthChecker, len(nodes))

	t.Cleanup(func() {
		for i := range checkers {
			_ = checkers[i].Destroy()
		}
	})

	for i := range nodes {
		checker := &PeerToPeerHealthChecker{}
		checker.Initialize(&plugin.ConfigEntry{
			Option: map[string]interface{}{
				"listenPort": int64(nodes[i].Port),
			},
		})

		checker.refreshPeers(nodes)
		for j := range checker.peers {
			peer := checker.peers[j]
			peer.Local = peer.Port == nodes[i].Port
		}
		checkers[nodes[i].ID] = checker
	}

	wait := sync.WaitGroup{}
	wait.Add(len(checkers))
	for i := range checkers {
		checker := checkers[i]
		go func(checker *PeerToPeerHealthChecker) {
			defer wait.Done()
			checker.servePeers()
			checker.calculateContinuum()
		}(checker)
	}

	wait.Wait()

	checker1 := checkers[nodes[0].ID]
	checker2 := checkers[nodes[1].ID]
	checker3 := checkers[nodes[2].ID]

	mockKey := utils.NewUUID()
	mockHost := "172.0.0.1"
	mockTimeSec := time.Now().Unix()

	request := &plugin.ReportRequest{
		QueryRequest: plugin.QueryRequest{
			InstanceId: mockKey,
			Host:       mockHost,
		},
		LocalHost:  nodes[0].Host,
		CurTimeSec: mockTimeSec,
	}
	err := checker1.Report(request)
	assert.NoError(t, err)
	// wait put op success
	time.Sleep(time.Second)

	repPeer1, ok := checker1.findResponsiblePeer(mockKey)
	assert.True(t, ok)
	repPeer2, ok := checker2.findResponsiblePeer(mockKey)
	assert.True(t, ok)
	repPeer3, ok := checker3.findResponsiblePeer(mockKey)
	assert.True(t, ok)
	assert.True(t, repPeer1.ID == repPeer2.ID && repPeer1.ID == repPeer3.ID)

	resp, err := checker1.Query(&request.QueryRequest)
	assert.NoError(t, err)
	assert.True(t, resp.Exists)
	assert.Equal(t, mockTimeSec, resp.LastHeartbeatSec)

	resp, err = checker2.Query(&request.QueryRequest)
	assert.NoError(t, err)
	assert.True(t, resp.Exists)
	assert.Equal(t, mockTimeSec, resp.LastHeartbeatSec)

	resp, err = checker3.Query(&request.QueryRequest)
	assert.NoError(t, err)
	assert.True(t, resp.Exists)
	assert.Equal(t, mockTimeSec, resp.LastHeartbeatSec)

	err = checker2.Delete(mockKey)
	assert.NoError(t, err)
	time.Sleep(time.Second)

	resp, err = checker1.Query(&request.QueryRequest)
	assert.NoError(t, err)
	assert.False(t, resp.Exists)

	resp, err = checker2.Query(&request.QueryRequest)
	assert.NoError(t, err)
	assert.False(t, resp.Exists)

	resp, err = checker3.Query(&request.QueryRequest)
	assert.NoError(t, err)
	assert.False(t, resp.Exists)
}
