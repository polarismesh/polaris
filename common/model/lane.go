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

	"github.com/golang/protobuf/proto"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"

	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
)

type TrafficEntryType string

const (
	TrafficEntry_TSEGateway         = "polarismesh.cn/gateway/tse-gateway"
	TrafficEntry_SpringCloudGateway = "polarismesh.cn/gateway/spring-cloud-gateway"
	TrafficEntry_MicroService       = "polarismesh.cn/service"
)

type LaneGroupProto struct {
	*LaneGroup
	Proto *apitraffic.LaneGroup
}

// LaneGroup 泳道分组
type LaneGroup struct {
	ID          string
	Name        string
	Rule        string
	Revision    string
	Description string
	Valid       bool
	CreateTime  time.Time
	ModifyTime  time.Time
	// LaneRules id -> *LaneRule
	LaneRules map[string]*LaneRule
}

func (l *LaneGroup) FromSpec(item *apitraffic.LaneGroup) error {
	l.ID = item.GetId()
	l.Name = item.GetName()
	l.Description = item.GetDescription()
	l.LaneRules = map[string]*LaneRule{}
	for i := range item.Rules {
		laneRule := &LaneRule{}
		if err := laneRule.FromSpec(item.Rules[i]); err != nil {
			return err
		}
		// 这里泳道所属泳道组的名称需要这里手动设置
		laneRule.LaneGroup = l.Name
		laneRule.ID = utils.DefaultString(laneRule.ID, utils.NewUUID())
		l.LaneRules[laneRule.ID] = laneRule
	}
	copyItem := proto.Clone(item).(*apitraffic.LaneGroup)
	copyItem.Rules = make([]*apitraffic.LaneRule, 0)
	rule, err := json.Marshal(copyItem)
	if err != nil {
		return err
	}
	l.Rule = string(rule)
	return nil
}

func (l *LaneGroup) ToProto() (*LaneGroupProto, error) {
	ret := &apitraffic.LaneGroup{}
	if err := json.Unmarshal([]byte(l.Rule), ret); err != nil {
		return nil, err
	}

	ret.Id = l.ID
	ret.Revision = l.Revision
	ret.Description = l.Description
	ret.Rules = make([]*apitraffic.LaneRule, 0, 32)
	ret.Ctime = commontime.Time2String(l.CreateTime)
	ret.Mtime = commontime.Time2String(l.ModifyTime)
	protoG := &LaneGroupProto{
		LaneGroup: l,
		Proto:     ret,
	}

	for k := range l.LaneRules {
		protoRule, err := l.LaneRules[k].ToProto()
		if err != nil {
			return nil, err
		}
		protoG.Proto.Rules = append(protoG.Proto.Rules, protoRule.Proto)
	}

	return protoG, nil
}

func (l *LaneGroup) ToSpec() (*apitraffic.LaneGroup, error) {
	ret := &apitraffic.LaneGroup{}
	if err := json.Unmarshal([]byte(l.Rule), ret); err != nil {
		return nil, err
	}
	return ret, nil
}

// LaneRule 泳道规则实体
type LaneRule struct {
	ID          string
	LaneGroup   string
	Name        string
	Rule        string
	Priority    uint32
	Revision    string
	Description string
	Enable      bool
	Valid       bool
	CreateTime  time.Time
	EnableTime  time.Time
	ModifyTime  time.Time

	// 仅仅用于修改 flag 标记
	changeEnable bool
	add          bool
}

func (l *LaneRule) SetChangeEnable(v bool) {
	l.changeEnable = v
}

func (l *LaneRule) IsChangeEnable() bool {
	return l.changeEnable
}

func (l *LaneRule) SetAddFlag(v bool) {
	l.add = v
}

func (l *LaneRule) IsAdd() bool {
	return l.add
}

func (l *LaneRule) FromSpec(item *apitraffic.LaneRule) error {
	l.ID = item.GetId()
	l.Name = item.GetName()
	l.LaneGroup = item.GetGroupName()
	l.Priority = item.GetPriority()
	l.Revision = utils.DefaultString(item.GetRevision(), utils.NewUUID())
	l.Description = item.GetDescription()
	l.Enable = item.GetEnable()

	rule, err := json.Marshal(item)
	if err != nil {
		return err
	}
	l.Rule = string(rule)
	return nil
}

func (l *LaneRule) ToProto() (*LaneRuleProto, error) {
	rule := &apitraffic.LaneRule{}
	if err := json.Unmarshal([]byte(l.Rule), rule); err != nil {
		return nil, err
	}
	rule.Id = l.ID
	rule.Name = l.Name
	rule.Enable = l.Enable
	rule.GroupName = l.LaneGroup
	rule.Description = l.Description
	rule.Priority = l.Priority
	rule.Revision = l.Revision
	rule.Ctime = commontime.Time2String(l.CreateTime)
	rule.Mtime = commontime.Time2String(l.ModifyTime)

	if l.EnableTime.Year() > 2000 {
		rule.Etime = commontime.Time2String(l.EnableTime)
	} else {
		rule.Etime = ""
	}

	return &LaneRuleProto{
		LaneRule: l,
		Proto:    rule,
	}, nil
}

type LaneRuleProto struct {
	*LaneRule
	Proto *apitraffic.LaneRule
}
