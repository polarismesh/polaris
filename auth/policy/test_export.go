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

package policy

import (
	"github.com/polarismesh/polaris/auth"
	"github.com/polarismesh/polaris/store"
)

// MockAuthChecker mock auth.AuthChecker for unit test
func (svr *Server) MockAuthChecker(checker auth.AuthChecker) {
	svr.checker = checker
}

// MockStore mock store.Store for unit test
func (svr *Server) MockStore(storage store.Store) {
	svr.storage = storage
}

// MockUserServer mock auth.UserServer for unit test
func (svr *Server) MockUserServer(userSvr auth.UserServer) {
	svr.userSvr = userSvr
}
