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

package service_test

import (
	"testing"

	"github.com/polarismesh/specification/source/go/api/v1/fault_tolerance"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"github.com/stretchr/testify/assert"
)

func TestServer_CircuitBreakersConfig(t *testing.T) {
	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	resp := discoverSuit.DiscoverServer().CreateCircuitBreakers(discoverSuit.DefaultCtx,
		[]*fault_tolerance.CircuitBreaker{})
	assert.Equal(t, apimodel.Code_BadRequest, apimodel.Code(resp.GetCode().GetValue()))

	resp = discoverSuit.DiscoverServer().CreateCircuitBreakerVersions(discoverSuit.DefaultCtx,
		[]*fault_tolerance.CircuitBreaker{})
	assert.Equal(t, apimodel.Code_BadRequest, apimodel.Code(resp.GetCode().GetValue()))

	resp = discoverSuit.DiscoverServer().DeleteCircuitBreakers(discoverSuit.DefaultCtx,
		[]*fault_tolerance.CircuitBreaker{})
	assert.Equal(t, apimodel.Code_BadRequest, apimodel.Code(resp.GetCode().GetValue()))

	resp = discoverSuit.DiscoverServer().UpdateCircuitBreakers(discoverSuit.DefaultCtx,
		[]*fault_tolerance.CircuitBreaker{})
	assert.Equal(t, apimodel.Code_BadRequest, apimodel.Code(resp.GetCode().GetValue()))

	resp = discoverSuit.DiscoverServer().ReleaseCircuitBreakers(discoverSuit.DefaultCtx,
		[]*service_manage.ConfigRelease{})
	assert.Equal(t, apimodel.Code_BadRequest, apimodel.Code(resp.GetCode().GetValue()))

	resp = discoverSuit.DiscoverServer().UnBindCircuitBreakers(discoverSuit.DefaultCtx,
		[]*service_manage.ConfigRelease{})
	assert.Equal(t, apimodel.Code_BadRequest, apimodel.Code(resp.GetCode().GetValue()))

	qresp := discoverSuit.DiscoverServer().GetCircuitBreaker(discoverSuit.DefaultCtx,
		map[string]string{})
	assert.Equal(t, apimodel.Code_BadRequest, apimodel.Code(qresp.GetCode().GetValue()))

	qresp = discoverSuit.DiscoverServer().GetCircuitBreakerVersions(discoverSuit.DefaultCtx,
		map[string]string{})
	assert.Equal(t, apimodel.Code_BadRequest, apimodel.Code(resp.GetCode().GetValue()))

	qresp = discoverSuit.DiscoverServer().GetMasterCircuitBreakers(discoverSuit.DefaultCtx,
		map[string]string{})
	assert.Equal(t, apimodel.Code_BadRequest, apimodel.Code(resp.GetCode().GetValue()))

	qresp = discoverSuit.DiscoverServer().GetReleaseCircuitBreakers(discoverSuit.DefaultCtx,
		map[string]string{})
	assert.Equal(t, apimodel.Code_BadRequest, apimodel.Code(resp.GetCode().GetValue()))

	qresp = discoverSuit.DiscoverServer().GetCircuitBreakerByService(discoverSuit.DefaultCtx,
		map[string]string{})
	assert.Equal(t, apimodel.Code_BadRequest, apimodel.Code(resp.GetCode().GetValue()))

	tresp := discoverSuit.DiscoverServer().GetCircuitBreakerToken(discoverSuit.DefaultCtx,
		&fault_tolerance.CircuitBreaker{})
	assert.Equal(t, apimodel.Code_BadRequest, apimodel.Code(tresp.GetCode().GetValue()))
}
