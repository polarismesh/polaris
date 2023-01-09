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

package memory

import (
	"net"
	"strings"

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/polarismesh/polaris/common/model"
)

// IPType ip type
type IPType string

const (
	Host    IPType = "host"
	Mask    IPType = "mask"
	Backoff IPType = "backoff"
)

// Request request cmdb data
type Request struct {
	RequestID string `json:"request_id"`
	PageNo    int64  `json:"page_no"`
	PageSize  int64  `json:"page_size"`
}

// Response response cmdb data
type Response struct {
	Total    int      `json:"total"`
	Size     int      `json:"size"`
	Code     int      `json:"code"`
	Info     string   `json:"info"`
	Priority string   `json:"priority"`
	Data     []IPInfo `json:"data"`
}

// IPInfo ip info
type IPInfo struct {
	IP     string       `json:"ip"`
	Type   IPType       `json:"type"`
	Region LocationInfo `json:"region"`
	Zone   LocationInfo `json:"zone"`
	Campus LocationInfo `json:"campus"`
}

// IP ip info
type IP struct {
	IP    string
	Type  IPType
	ipNet *net.IPNet
	loc   *model.Location
}

func NewIP(info IPInfo) (IP, error) {
	var ip IP

	if info.Type == Mask {
		_, cidr, err := net.ParseCIDR(info.IP)
		if err != nil {
			return IP{}, err
		}

		ip.ipNet = cidr
	}

	regionId, _ := info.Region.Metadata["id"].(int64)
	zoneId, _ := info.Zone.Metadata["id"].(int64)
	campusId, _ := info.Campus.Metadata["id"].(int64)

	ip.IP = info.IP
	ip.Type = info.Type
	ip.loc = &model.Location{
		Proto: &apimodel.Location{
			Region: &wrapperspb.StringValue{
				Value: info.Region.Name,
			},
			Zone: &wrapperspb.StringValue{
				Value: info.Zone.Name,
			},
			Campus: &wrapperspb.StringValue{
				Value: info.Campus.Name,
			},
		},
		RegionID: uint32(regionId),
		ZoneID:   uint32(zoneId),
		CampusID: uint32(campusId),
		Valid:    false,
	}

	return ip, nil
}

// Match target ip is match
func (p IP) Match(ip string) bool {
	switch p.Type {
	case Host:
		return strings.Compare(p.IP, ip) == 0
	case Mask:
		return p.ipNet.Contains(net.ParseIP(ip))
	case Backoff:
		return true
	default:
		return false
	}
}

// LocationInfo
type LocationInfo struct {
	Name     string                 `json:"name"`
	Metadata map[string]interface{} `json:"metadata"`
}
