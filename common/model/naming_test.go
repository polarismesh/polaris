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

package model

import (
	"testing"

	"github.com/golang/protobuf/ptypes/wrappers"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	"github.com/stretchr/testify/assert"
)

// TestRateLimit_Labels2Arguments 测试标签转换
func TestRateLimit_Labels2Arguments(t *testing.T) {
	rateLimit := &RateLimit{
		Proto:  &apitraffic.Rule{},
		Labels: "{\"business.key\": {\"type\": 1, \"value\": {\"value\": \"1234567\"}}}",
	}
	_, err := rateLimit.Labels2Arguments()
	assert.Nil(t, err)
	assert.Equal(t, 1, len(rateLimit.Proto.Arguments))
	assert.Equal(t, apitraffic.MatchArgument_CUSTOM, rateLimit.Proto.Arguments[0].Type)
	assert.Equal(t, "business.key", rateLimit.Proto.Arguments[0].Key)
	assert.Equal(t, apimodel.MatchString_REGEX, rateLimit.Proto.Arguments[0].Value.Type)
	assert.Equal(t, "1234567", rateLimit.Proto.Arguments[0].Value.Value.GetValue())
}

// TestRateLimit_Arguments2Labels 测试参数转换标签
func TestRateLimit_Arguments2Labels(t *testing.T) {
	rateLimit := &RateLimit{
		Proto: &apitraffic.Rule{
			Arguments: []*apitraffic.MatchArgument{
				{
					Type:  apitraffic.MatchArgument_CUSTOM,
					Key:   "business.key",
					Value: &apimodel.MatchString{Type: apimodel.MatchString_EXACT, Value: &wrappers.StringValue{Value: "1234567"}},
				},
				{
					Type:  apitraffic.MatchArgument_METHOD,
					Value: &apimodel.MatchString{Type: apimodel.MatchString_EXACT, Value: &wrappers.StringValue{Value: "/path"}},
				},
				{
					Type:  apitraffic.MatchArgument_HEADER,
					Key:   "host",
					Value: &apimodel.MatchString{Type: apimodel.MatchString_EXACT, Value: &wrappers.StringValue{Value: "localhost"}},
				},
				{
					Type:  apitraffic.MatchArgument_QUERY,
					Key:   "name",
					Value: &apimodel.MatchString{Type: apimodel.MatchString_EXACT, Value: &wrappers.StringValue{Value: "ok"}},
				},
				{
					Type:  apitraffic.MatchArgument_CALLER_SERVICE,
					Key:   "default",
					Value: &apimodel.MatchString{Type: apimodel.MatchString_EXACT, Value: &wrappers.StringValue{Value: "svc"}},
				},
				{
					Type:  apitraffic.MatchArgument_CALLER_IP,
					Value: &apimodel.MatchString{Type: apimodel.MatchString_EXACT, Value: &wrappers.StringValue{Value: "127.0.0.1"}},
				},
			},
		},
	}
	labels := Arguments2Labels(rateLimit.Proto.GetArguments())
	if len(labels) > 0 {
		rateLimit.Proto.Labels = labels
	}
	var hasValue bool
	var value *apimodel.MatchString
	value, hasValue = rateLimit.Proto.Labels["business.key"]
	assert.True(t, hasValue)
	assert.Equal(t, "1234567", value.Value.GetValue())

	value, hasValue = rateLimit.Proto.Labels["$method"]
	assert.True(t, hasValue)
	assert.Equal(t, "/path", value.Value.GetValue())

	value, hasValue = rateLimit.Proto.Labels["$header.host"]
	assert.True(t, hasValue)
	assert.Equal(t, "localhost", value.Value.GetValue())

	value, hasValue = rateLimit.Proto.Labels["$query.name"]
	assert.True(t, hasValue)
	assert.Equal(t, "ok", value.Value.GetValue())

	value, hasValue = rateLimit.Proto.Labels["$caller_service.default"]
	assert.True(t, hasValue)
	assert.Equal(t, "svc", value.Value.GetValue())

	value, hasValue = rateLimit.Proto.Labels["$caller_ip"]
	assert.True(t, hasValue)
	assert.Equal(t, "127.0.0.1", value.Value.GetValue())
}
