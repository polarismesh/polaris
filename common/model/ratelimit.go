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
	"encoding/json"
	"time"

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
)

// RateLimit 限流规则
type RateLimit struct {
	Proto     *apitraffic.Rule
	ID        string
	ServiceID string
	Name      string
	Method    string
	// Labels for old compatible, will be removed later
	Labels     string
	Priority   uint32
	Rule       string
	Revision   string
	Disable    bool
	Valid      bool
	CreateTime time.Time
	ModifyTime time.Time
	EnableTime time.Time
}

// Labels2Arguments 适配老的标签到新的参数列表
func (r *RateLimit) Labels2Arguments() (map[string]*apimodel.MatchString, error) {
	if len(r.Proto.Arguments) == 0 && len(r.Labels) > 0 {
		var labels = make(map[string]*apimodel.MatchString)
		if err := json.Unmarshal([]byte(r.Labels), &labels); err != nil {
			return nil, err
		}
		for key, value := range labels {
			r.Proto.Arguments = append(r.Proto.Arguments, &apitraffic.MatchArgument{
				Type:  apitraffic.MatchArgument_CUSTOM,
				Key:   key,
				Value: value,
			})
		}
		return labels, nil
	}
	return nil, nil
}

const (
	LabelKeyPath          = "$path"
	LabelKeyMethod        = "$method"
	LabelKeyHeader        = "$header"
	LabelKeyQuery         = "$query"
	LabelKeyCallerService = "$caller_service"
	LabelKeyCallerIP      = "$caller_ip"
)

// Arguments2Labels 将参数列表适配成旧的标签模型
func Arguments2Labels(arguments []*apitraffic.MatchArgument) map[string]*apimodel.MatchString {
	if len(arguments) > 0 {
		var labels = make(map[string]*apimodel.MatchString)
		for _, argument := range arguments {
			key := BuildArgumentKey(argument.Type, argument.Key)
			labels[key] = argument.Value
		}
		return labels
	}
	return nil
}

func BuildArgumentKey(argumentType apitraffic.MatchArgument_Type, key string) string {
	switch argumentType {
	case apitraffic.MatchArgument_HEADER:
		return LabelKeyHeader + "." + key
	case apitraffic.MatchArgument_QUERY:
		return LabelKeyQuery + "." + key
	case apitraffic.MatchArgument_CALLER_SERVICE:
		return LabelKeyCallerService + "." + key
	case apitraffic.MatchArgument_CALLER_IP:
		return LabelKeyCallerIP
	case apitraffic.MatchArgument_CUSTOM:
		return key
	case apitraffic.MatchArgument_METHOD:
		return LabelKeyMethod
	default:
		return key
	}
}

// AdaptArgumentsAndLabels 对存量标签进行兼容，同时将argument适配成标签
func (r *RateLimit) AdaptArgumentsAndLabels() error {
	// 新的限流规则，需要适配老的SDK使用场景
	labels := Arguments2Labels(r.Proto.GetArguments())
	if len(labels) > 0 {
		r.Proto.Labels = labels
	} else {
		var err error
		// 存量限流规则，需要适配成新的规则
		labels, err = r.Labels2Arguments()
		if nil != err {
			return err
		}
		r.Proto.Labels = labels
	}
	return nil
}

// AdaptLabels 对存量标签进行兼容，对存量labels进行清空
func (r *RateLimit) AdaptLabels() error {
	// 存量限流规则，需要适配成新的规则
	_, err := r.Labels2Arguments()
	if nil != err {
		return err
	}
	r.Proto.Labels = nil
	return nil
}

// ExtendRateLimit 包含服务信息的限流规则
type ExtendRateLimit struct {
	ServiceName   string
	NamespaceName string
	RateLimit     *RateLimit
}

// RateLimitRevision 包含最新版本号的限流规则
type RateLimitRevision struct {
	ServiceID    string
	LastRevision string
	ModifyTime   time.Time
}
