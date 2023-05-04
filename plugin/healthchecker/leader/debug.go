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
	"net/http"

	"github.com/polarismesh/polaris/common/utils"
)

func handleDescribeLeaderInfo(checker *LeaderHealthChecker) func(http.ResponseWriter, *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		if !checker.isInitialize() {
			resp.WriteHeader(http.StatusTooEarly)
			_, _ = resp.Write([]byte("LeaderChecker not initialize"))
			return
		}

		ret := map[string]interface{}{}
		if checker.isLeader() {
			ret["leader"] = utils.LocalHost
		} else {
			if checker.remote != nil {
				ret["leader"] = checker.remote.Host()
			}
		}
		ret["self"] = utils.LocalHost
		ret["lastLeaderRefreshTimeSec"] = checker.LeaderChangeTimeSec()

		data, _ := json.Marshal(ret)
		resp.WriteHeader(http.StatusOK)
		_, _ = resp.Write(data)
	}
}

func handleDescribeBeatCache(checker *LeaderHealthChecker) func(http.ResponseWriter, *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		if !checker.isInitialize() {
			resp.WriteHeader(http.StatusTooEarly)
			_, _ = resp.Write([]byte("LeaderChecker not initialize"))
			return
		}

		ret := map[string]interface{}{}
		ret["self"] = utils.LocalHost
		if checker.isLeader() {
			ret["data"] = checker.self.(*LocalPeer).Cache.Snapshot()
		} else {
			ret["data"] = "Not Leader"
		}

		data, _ := json.Marshal(ret)
		resp.WriteHeader(http.StatusOK)
		_, _ = resp.Write(data)
	}
}
