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

package leader

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"

	commonhash "github.com/polarismesh/polaris/common/hash"
	"github.com/polarismesh/polaris/common/utils"
)

func Test_LeaderCheckerDebugerHandler(t *testing.T) {
	t.Run("handleDescribeLeaderInfo", func(t *testing.T) {
		t.Run("01_to_early", func(t *testing.T) {
			leader := &LeaderHealthChecker{}
			httpFunc := handleDescribeLeaderInfo(leader)
			recorder := httptest.NewRecorder()
			httpFunc(recorder, httptest.NewRequest(http.MethodPost, "http://127.0.0.1:1234", nil))
			assert.Equal(t, http.StatusTooEarly, recorder.Code)
		})

		t.Run("02_self_leader", func(t *testing.T) {
			leader := &LeaderHealthChecker{}
			httpFunc := handleDescribeLeaderInfo(leader)
			atomic.StoreInt32(&leader.leader, 1)
			atomic.StoreInt32(&leader.initialize, 1)
			leader.remote = &RemotePeer{
				host: "172.0.0.1",
			}

			recorder := httptest.NewRecorder()
			httpFunc(recorder, httptest.NewRequest(http.MethodPost, "http://127.0.0.1:1234", nil))
			assert.Equal(t, http.StatusOK, recorder.Code)

			ret := map[string]interface{}{}
			data, _ := io.ReadAll(recorder.Body)
			_ = json.Unmarshal(data, &ret)

			assert.Equal(t, utils.LocalHost, ret["leader"])
			assert.Equal(t, utils.LocalHost, ret["self"])
		})

		t.Run("03_self_follower", func(t *testing.T) {
			leader := &LeaderHealthChecker{}
			httpFunc := handleDescribeLeaderInfo(leader)
			atomic.StoreInt32(&leader.leader, 0)
			atomic.StoreInt32(&leader.initialize, 1)
			leader.remote = &RemotePeer{
				host: "172.0.0.1",
			}

			recorder := httptest.NewRecorder()
			httpFunc(recorder, httptest.NewRequest(http.MethodPost, "http://127.0.0.1:1234", nil))
			assert.Equal(t, http.StatusOK, recorder.Code)

			ret := map[string]interface{}{}
			data, _ := io.ReadAll(recorder.Body)
			_ = json.Unmarshal(data, &ret)

			assert.Equal(t, "172.0.0.1", ret["leader"])
			assert.Equal(t, utils.LocalHost, ret["self"])
		})
	})

	t.Run("handleDescribeBeatCache", func(t *testing.T) {
		t.Run("00_to_early", func(t *testing.T) {
			leader := &LeaderHealthChecker{}
			httpFunc := handleDescribeBeatCache(leader)
			recorder := httptest.NewRecorder()
			httpFunc(recorder, httptest.NewRequest(http.MethodPost, "http://127.0.0.1:1234", nil))
			assert.Equal(t, http.StatusTooEarly, recorder.Code)
		})

		t.Run("01_self_leader", func(t *testing.T) {
			leader := &LeaderHealthChecker{
				self: &LocalPeer{
					Cache: newLocalBeatRecordCache(1, commonhash.Fnv32),
				},
			}
			leader.self.Storage().Put(WriteBeatRecord{
				Record: RecordValue{
					Server:     utils.LocalHost,
					CurTimeSec: 123,
					Count:      0,
				},
				Key: "123",
			})

			httpFunc := handleDescribeBeatCache(leader)
			atomic.StoreInt32(&leader.leader, 1)
			atomic.StoreInt32(&leader.initialize, 1)

			recorder := httptest.NewRecorder()
			httpFunc(recorder, httptest.NewRequest(http.MethodPost, "http://127.0.0.1:1234", nil))
			assert.Equal(t, http.StatusOK, recorder.Code)

			ret := map[string]interface{}{}
			data, _ := io.ReadAll(recorder.Body)
			_ = json.Unmarshal(data, &ret)

			expectData, _ := json.Marshal(leader.self.Storage().Snapshot())
			expectRet := map[string]interface{}{}
			_ = json.Unmarshal(expectData, &expectRet)

			assert.Equal(t, expectRet, ret["data"])
			assert.Equal(t, utils.LocalHost, ret["self"])
		})

		t.Run("02_self_follower", func(t *testing.T) {
			leader := &LeaderHealthChecker{}
			httpFunc := handleDescribeBeatCache(leader)
			atomic.StoreInt32(&leader.leader, 0)
			atomic.StoreInt32(&leader.initialize, 1)

			recorder := httptest.NewRecorder()
			httpFunc(recorder, httptest.NewRequest(http.MethodPost, "http://127.0.0.1:1234", nil))
			assert.Equal(t, http.StatusOK, recorder.Code)

			ret := map[string]interface{}{}
			data, _ := io.ReadAll(recorder.Body)
			_ = json.Unmarshal(data, &ret)

			assert.Equal(t, "Not Leader", ret["data"])
			assert.Equal(t, utils.LocalHost, ret["self"])
		})
	})
}
