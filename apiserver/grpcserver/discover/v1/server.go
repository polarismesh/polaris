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

package v1

import (
	"github.com/polarismesh/polaris/service"
	"github.com/polarismesh/polaris/service/healthcheck"
)

type DiscoverServer struct {
	namingServer      service.DiscoverServer
	healthCheckServer *healthcheck.Server
	enterRateLimit    func(ip string, method string) uint32
	allowAccess       func(method string) bool
}

func NewDiscoverServer(options ...Option) *DiscoverServer {
	s := &DiscoverServer{}

	for i := range options {
		options[i](s)
	}

	return s
}

type Option func(s *DiscoverServer)

func WithNamingServer(svr service.DiscoverServer) Option {
	return func(s *DiscoverServer) {
		s.namingServer = svr
	}
}

func WithHealthCheckerServer(svr *healthcheck.Server) Option {
	return func(s *DiscoverServer) {
		s.healthCheckServer = svr
	}
}

func WithEnterRateLimit(f func(ip string, method string) uint32) Option {
	return func(s *DiscoverServer) {
		s.enterRateLimit = f
	}
}

func WithAllowAccess(f func(method string) bool) Option {
	return func(s *DiscoverServer) {
		s.allowAccess = f
	}
}
