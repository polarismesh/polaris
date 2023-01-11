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

import (
	"testing"

	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	"github.com/stretchr/testify/assert"

	v2 "github.com/polarismesh/polaris/common/api/v2"
)

func TestConvertSameStructureMessage(t *testing.T) {
	configV1 := &apitraffic.RuleRoutingConfig{}
	configV2 := &v2.RuleRoutingConfig{
		Sources: []*v2.Source{
			{
				Service:   "testSvc",
				Namespace: "test",
				Arguments: []*v2.SourceMatch{},
			},
		},
	}
	err := ConvertSameStructureMessage(configV2, configV1)
	assert.Nil(t, err)
	assert.Equal(t, configV2.Sources[0].Service, configV1.Sources[0].Service)
	assert.Equal(t, configV2.Sources[0].Namespace, configV1.Sources[0].Namespace)
}
