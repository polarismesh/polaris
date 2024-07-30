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

package utils

const (
	// PolarisCode polaris code
	PolarisCode = "X-Polaris-Code"
	// PolarisMessage polaris message
	PolarisMessage = "X-Polaris-Message"
	// PolarisRequestID request_id
	PolarisRequestID = "Request-Id"
)

var (
	// LocalHost local host
	LocalHost = "127.0.0.1"
	// LocalPort default listen port
	LocalPort = 8091
	// ConfDir default config dir
	ConfDir = "conf/"
)

const (
	// HeaderAuthTokenKey auth token key
	HeaderAuthTokenKey string = "X-Polaris-Token"
	// HeaderIsOwnerKey is owner key
	HeaderIsOwnerKey string = "X-Is-Owner"
	// HeaderUserIDKey user id key
	HeaderUserIDKey string = "X-User-ID"
	// HeaderOwnerIDKey owner id key
	HeaderOwnerIDKey string = "X-Owner-ID"
	// HeaderUserRoleKey user role key
	HeaderUserRoleKey string = "X-Polaris-User-Role"

	// ContextAuthTokenKey auth token key
	ContextAuthTokenKey = StringContext(HeaderAuthTokenKey)
	// ContextIsOwnerKey is owner key
	ContextIsOwnerKey = StringContext(HeaderIsOwnerKey)
	// ContextUserIDKey user id key
	ContextUserIDKey = StringContext(HeaderUserIDKey)
	// ContextOwnerIDKey owner id key
	ContextOwnerIDKey = StringContext(HeaderOwnerIDKey)
	// ContextUserRoleIDKey user role key
	ContextUserRoleIDKey = StringContext(HeaderUserRoleKey)
	// ContextAuthContextKey auth context key
	ContextAuthContextKey = StringContext("X-Polaris-AuthContext")
	// ContextUserNameKey users name key
	ContextUserNameKey = StringContext("X-User-Name")
	// ContextClientAddress client address key
	ContextClientAddress = StringContext("client-address")
	// ContextOpenAsyncRegis open async register key
	ContextOpenAsyncRegis = StringContext("client-asyncRegis")
	// ContextGrpcHeader grpc header key
	ContextGrpcHeader = StringContext("grpc-header")
	// ContextIsFromClient is from client
	ContextIsFromClient = StringContext("from-client")
	// ContextIsFromSystem is from polaris system
	ContextIsFromSystem = StringContext("from-system")
	// ContextOperator operator info
	ContextOperator = StringContext("operator")
	// ContextRequestHeaders request headers
	ContextRequestHeaders = StringContext("request-headers")
)
