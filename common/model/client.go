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
	"time"

	"github.com/golang/protobuf/ptypes/wrappers"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	commontime "github.com/polarismesh/polaris/common/time"
)

// Client 客户端上报信息表
type Client struct {
	proto      *apiservice.Client
	valid      bool
	modifyTime time.Time
}

func NewClient(req *apiservice.Client) *Client {
	return &Client{
		proto: req,
	}
}

func (c *Client) Proto() *apiservice.Client {
	return c.proto
}

func (c *Client) SetValid(v bool) {
	c.valid = v
}

func (c *Client) Valid() bool {
	return c.valid
}

func (c *Client) ModifyTime() time.Time {
	return c.modifyTime
}

// ClientStore 对应store层（database）的对象
type ClientStore struct {
	ID         string
	Host       string
	Type       string
	Version    string
	Region     string
	Zone       string
	Campus     string
	Stat       ClientStatStore
	Flag       int
	CreateTime int64
	ModifyTime int64
}

type ClientStatStore struct {
	Target   string
	Port     uint32
	Protocol string
	Path     string
}

// Store2Instance store的数据转换为组合了api的数据结构
func Store2Client(is *ClientStore) *Client {
	ins := &Client{
		proto: &apiservice.Client{
			Id:      &wrappers.StringValue{Value: is.ID},
			Host:    &wrappers.StringValue{Value: is.Host},
			Version: &wrappers.StringValue{Value: is.Version},
			Type:    apiservice.Client_ClientType(apiservice.Client_ClientType_value[is.Type]),
			Location: &apimodel.Location{
				Campus: &wrappers.StringValue{Value: is.Campus},
				Zone:   &wrappers.StringValue{Value: is.Zone},
				Region: &wrappers.StringValue{Value: is.Region},
			},
			Ctime: &wrappers.StringValue{Value: commontime.Int64Time2String(is.CreateTime)},
			Mtime: &wrappers.StringValue{Value: commontime.Int64Time2String(is.ModifyTime)},
		},
		valid:      flag2valid(is.Flag),
		modifyTime: time.Unix(is.ModifyTime, 0),
	}
	statInfo := Store2ClientStat(&is.Stat)
	if nil != statInfo {
		ins.proto.Stat = append(ins.proto.Stat, statInfo)
	}
	return ins
}

func Store2ClientStat(clientStatStore *ClientStatStore) *apiservice.StatInfo {
	if len(clientStatStore.Target) == 0 && clientStatStore.Port == 0 && len(clientStatStore.Path) == 0 &&
		len(clientStatStore.Protocol) == 0 {
		return nil
	}
	statInfo := &apiservice.StatInfo{}
	statInfo.Path = &wrappers.StringValue{Value: clientStatStore.Path}
	statInfo.Protocol = &wrappers.StringValue{Value: clientStatStore.Protocol}
	statInfo.Port = &wrappers.UInt32Value{Value: clientStatStore.Port}
	statInfo.Target = &wrappers.StringValue{Value: clientStatStore.Target}
	return statInfo
}

const (
	StatReportPrometheus string = "prometheus"
)

type PrometheusDiscoveryResponse struct {
	Code     uint32
	Response []PrometheusTarget
}

// PrometheusTarget 用于对接 prometheus service discovery 的数据结构
type PrometheusTarget struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}

const (
	// ClientLabel_IP 客户端 IP
	ClientLabel_IP = "CLIENT_IP"
	// ClientLabel_ID 客户端 ID
	ClientLabel_ID = "CLIENT_ID"
	// ClientLabel_Version 客户端版本
	ClientLabel_Version = "CLIENT_VERSION"
	// ClientLabel_Language 客户端语言
	ClientLabel_Language = "CLIENT_LANGUAGE"
)
