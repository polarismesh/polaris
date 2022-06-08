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

package eurekaserver

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/polarismesh/polaris-server/common/model"
)

// PortWrapper 端口包装类
type PortWrapper struct {
	Port interface{} `json:"$" xml:",chardata"`

	RealPort int `json:"-" xml:"-"`

	Enabled interface{} `json:"@enabled" xml:"enabled,attr"`

	RealEnable bool `json:"-" xml:"-"`
}

// UnmarshalXML PortWrapper xml 反序列化
func (p *PortWrapper) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var err error
	port := struct {
		Port    string `xml:",chardata"`
		Enabled string `xml:"enabled,attr"`
	}{}
	if err = d.DecodeElement(&port, &start); err != nil {
		return err
	}

	p.RealPort, err = strconv.Atoi(port.Port)
	if err != nil {
		return err
	}
	p.Port = port.Port
	p.RealEnable, err = strconv.ParseBool(port.Enabled)
	if err != nil {
		return err
	}
	p.Enabled = port.Enabled
	return nil
}

func (p *PortWrapper) convertPortValue() error {
	if jsonNumber, ok := p.Port.(json.Number); ok {
		realPort, err := jsonNumber.Int64()
		if err != nil {
			return err
		}
		p.RealPort = int(realPort)
		return nil
	}
	if floatValue, ok := p.Port.(float64); ok {
		p.RealPort = int(floatValue)
		return nil
	}
	if strValue, ok := p.Port.(string); ok {
		var err error
		p.RealPort, err = strconv.Atoi(strValue)
		return err
	}
	return fmt.Errorf("unknow type of port value, type is %v", reflect.TypeOf(p.Port))
}

func (p *PortWrapper) convertEnableValue() error {
	if jsonEnableStr, ok := p.Enabled.(string); ok {
		enableValue, err := strconv.ParseBool(jsonEnableStr)
		if err != nil {
			return err
		}
		p.RealEnable = enableValue
		return nil
	}
	if jsonEnableValue, ok := p.Enabled.(bool); ok {
		p.RealEnable = jsonEnableValue
		return nil
	}
	return fmt.Errorf("unknow type of enable value, type is %v", reflect.TypeOf(p.Enabled))
}

// DataCenterInfo 数据中心信息
type DataCenterInfo struct {
	Clazz string `json:"@class" xml:"class,attr"`

	Name string `json:"name" xml:"name"`
}

// LeaseInfo 租约信息
type LeaseInfo struct {

	// Client settings
	RenewalIntervalInSecs int `json:"renewalIntervalInSecs" xml:"renewalIntervalInSecs"`

	DurationInSecs int `json:"durationInSecs" xml:"durationInSecs"`

	// Server populated
	RegistrationTimestamp int `json:"registrationTimestamp" xml:"registrationTimestamp"`

	LastRenewalTimestamp int `json:"lastRenewalTimestamp" xml:"lastRenewalTimestamp"`

	EvictionTimestamp int `json:"evictionTimestamp" xml:"evictionTimestamp"`

	ServiceUpTimestamp int `json:"serviceUpTimestamp" xml:"serviceUpTimestamp"`
}

// RegistrationRequest 实例注册请求
type RegistrationRequest struct {
	Instance *InstanceInfo `json:"instance"`
}

// ApplicationsResponse 服务拉取应答
type ApplicationsResponse struct {
	Applications *Applications `json:"applications"`
}

// ApplicationResponse 单个服务拉取响应
type ApplicationResponse struct {
	Application *Application `json:"application"`
}

// InstanceResponse 单个服务实例拉取响应
type InstanceResponse struct {
	InstanceInfo *InstanceInfo `json:"instance" xml:"instance"`
}

// Metadata 元数据信息，xml 格式无法直接反序列化成 map[string]string 类型。这里通过 []byte 类型的 Raw 接收，并反序列化到 Meta 中。
// 反序列化后，polaris 在业务中只使用 Meta 字段。
type Metadata struct {
	Raw []byte `xml:",innerxml" json:"-"`

	Attributes StringMap

	Meta StringMap
}

// UnmarshalJSON Metadata json 反序列化方法
func (i *Metadata) UnmarshalJSON(b []byte) error {
	i.Raw = b

	err := json.Unmarshal(b, &i.Meta)
	if err != nil {
		return err
	}

	return nil
}

// UnmarshalXML Metadata xml 反序列化方法
func (i *Metadata) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	i.Meta = make(map[string]interface{})
	values, err := xmlToMapParser(start.Name.Local, start.Attr, d, true)
	if err != nil {
		return err
	}
	var subValues map[string]interface{}
	if len(values) > 0 {
		subValuesRaw := values[start.Name.Local]
		subValues, _ = subValuesRaw.(map[string]interface{})
	}
	if len(subValues) > 0 {
		for k, v := range subValues {
			i.Meta[k] = v
		}
	}
	return nil
}

// MarshalJSON Metadata json 序列化方法
func (i *Metadata) MarshalJSON() ([]byte, error) {
	if i.Meta != nil {
		return json.Marshal(i.Meta)
	}

	if i.Raw == nil {
		i.Raw = []byte("{}")
	}

	return i.Raw, nil
}

func startLocalName(local string) xml.StartElement {
	return xml.StartElement{Name: xml.Name{Space: "", Local: local}}
}

func startLocalAttribute(attrName string, attrValue string) xml.Attr {
	return xml.Attr{Name: xml.Name{Space: "", Local: attrName}, Value: attrValue}
}

// MarshalXML Metadata xml 序列化方法
func (i *Metadata) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	tokens := []xml.Token{start}
	if i.Meta != nil {
		for key, value := range i.Meta {
			strValue := ObjectToString(value)
			if strings.HasPrefix(key, attributeNotion) {
				start.Attr = append(start.Attr, startLocalAttribute(key[1:], strValue))
				continue
			}
			// 兼容，后续去掉
			if strings.HasPrefix(key, attributeNotionCross) {
				start.Attr = append(start.Attr, startLocalAttribute(key[1:], strValue))
				continue
			}
			t := startLocalName(key)
			tokens = append(tokens, t, xml.CharData(strValue), xml.EndElement{Name: t.Name})
		}
	}
	tokens = append(tokens, xml.EndElement{Name: start.Name})
	if len(start.Attr) > 0 {
		tokens[0] = start
	}
	for _, t := range tokens {
		err := e.EncodeToken(t)
		if err != nil {
			return err
		}
	}

	// flush to ensure tokens are written
	return e.Flush()
}

// InstanceInfo 实例信息
type InstanceInfo struct {
	XMLName struct{} `json:"-" xml:"instance"`

	InstanceId string `json:"instanceId" xml:"instanceId"`

	AppName string `json:"app" xml:"app"`

	AppGroupName string `json:"appGroupName" xml:"appGroupName,omitempty"`

	IpAddr string `json:"ipAddr" xml:"ipAddr"`

	Sid string `json:"sid" xml:"sid,omitempty"`

	Port *PortWrapper `json:"port" xml:"port,omitempty"`

	SecurePort *PortWrapper `json:"securePort" xml:"securePort,omitempty"`

	HomePageUrl string `json:"homePageUrl" xml:"homePageUrl,omitempty"`

	StatusPageUrl string `json:"statusPageUrl" xml:"statusPageUrl,omitempty"`

	HealthCheckUrl string `json:"healthCheckUrl" xml:"healthCheckUrl,omitempty"`

	SecureHealthCheckUrl string `json:"secureHealthCheckUrl" xml:"secureHealthCheckUrl,omitempty"`

	VipAddress string `json:"vipAddress" xml:"vipAddress,omitempty"`

	SecureVipAddress string `json:"secureVipAddress" xml:"secureVipAddress,omitempty"`

	CountryId interface{} `json:"countryId" xml:"countryId,omitempty"`

	DataCenterInfo *DataCenterInfo `json:"dataCenterInfo" xml:"dataCenterInfo"`

	HostName string `json:"hostName" xml:"hostName,omitempty"`

	Status string `json:"status" xml:"status"`

	OverriddenStatus string `json:"overriddenStatus" xml:"overriddenStatus,omitempty"`

	LeaseInfo *LeaseInfo `json:"leaseInfo" xml:"leaseInfo,omitempty"`

	IsCoordinatingDiscoveryServer interface{} `json:"isCoordinatingDiscoveryServer" xml:"isCoordinatingDiscoveryServer,omitempty"`

	Metadata *Metadata `json:"metadata" xml:"metadata"`

	LastUpdatedTimestamp interface{} `json:"lastUpdatedTimestamp" xml:"lastUpdatedTimestamp,omitempty"`

	LastDirtyTimestamp interface{} `json:"lastDirtyTimestamp" xml:"lastDirtyTimestamp,omitempty"`

	ActionType string `json:"actionType" xml:"actionType"`

	// 实际的北极星实例模型, key为revision
	RealInstances map[string]*model.Instance `json:"-" xml:"-"`
}

// Clone 对实例进行拷贝
func (i *InstanceInfo) Clone(actionType string) *InstanceInfo {
	return &InstanceInfo{
		InstanceId:                    i.InstanceId,
		AppName:                       i.AppName,
		AppGroupName:                  i.AppGroupName,
		IpAddr:                        i.IpAddr,
		Sid:                           i.Sid,
		Port:                          i.Port,
		SecurePort:                    i.SecurePort,
		HomePageUrl:                   i.HomePageUrl,
		StatusPageUrl:                 i.StatusPageUrl,
		HealthCheckUrl:                i.HealthCheckUrl,
		SecureHealthCheckUrl:          i.SecureHealthCheckUrl,
		VipAddress:                    i.VipAddress,
		SecureVipAddress:              i.SecureVipAddress,
		CountryId:                     i.CountryId,
		DataCenterInfo:                i.DataCenterInfo,
		HostName:                      i.HostName,
		Status:                        i.Status,
		OverriddenStatus:              i.OverriddenStatus,
		LeaseInfo:                     i.LeaseInfo,
		IsCoordinatingDiscoveryServer: i.IsCoordinatingDiscoveryServer,
		Metadata:                      i.Metadata,
		LastUpdatedTimestamp:          i.LastUpdatedTimestamp,
		LastDirtyTimestamp:            i.LastDirtyTimestamp,
		ActionType:                    actionType,
	}
}

// Equals 判断实例是否发生变更
func (i *InstanceInfo) Equals(another *InstanceInfo) bool {
	if len(i.RealInstances) != len(another.RealInstances) {
		return false
	}
	if len(i.RealInstances) == 0 {
		return true
	}
	for revision := range i.RealInstances {
		if _, ok := another.RealInstances[revision]; !ok {
			return false
		}
	}
	return true
}

// Application 服务数据
type Application struct {
	XMLName struct{} `json:"-" xml:"application"`

	Name string `json:"name" xml:"name"`

	Instance []*InstanceInfo `json:"instance" xml:"instance"`

	InstanceMap map[string]*InstanceInfo `json:"-" xml:"-"`

	Revision string `json:"-" xml:"-"`

	StatusCounts map[string]int `json:"-" xml:"-"`
}

// GetInstance 获取eureka实例
func (a *Application) GetInstance(instId string) *InstanceInfo {
	if len(a.InstanceMap) > 0 {
		return a.InstanceMap[instId]
	}
	return nil
}

// Applications 服务列表
type Applications struct {
	XMLName struct{} `json:"-" xml:"applications"`

	VersionsDelta string `json:"versions__delta" xml:"versions__delta"`

	AppsHashCode string `json:"apps__hashcode" xml:"apps__hashcode"`

	Application []*Application `json:"application" xml:"application"`

	ApplicationMap map[string]*Application `json:"-" xml:"-"`
}

// GetApplication 获取eureka应用
func (a *Applications) GetApplication(appId string) *Application {
	if len(a.ApplicationMap) > 0 {
		return a.ApplicationMap[appId]
	}
	return nil
}

// GetInstance get instance by instanceId
func (a *Applications) GetInstance(instId string) *InstanceInfo {
	if len(a.Application) == 0 {
		return nil
	}
	for _, app := range a.Application {
		inst, ok := app.InstanceMap[instId]
		if ok {
			return inst
		}
	}
	return nil
}

// StringMap is a map[string]string.
type StringMap map[string]interface{}
